package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/GhostHunter2442/service-notification/internal/domain"
)

// easyMoneyDefaultEndpoint = endpoint ส่ง SMS ของ easymoney
const easyMoneyDefaultEndpoint = "https://easymoneydev.com/services/sms/sendMessage"

// EasyMoneySender = adapter ส่ง SMS ผ่าน HTTP API ของ easymoney
// API รับ 1 ข้อความ ยิงเข้าหลายเบอร์ในครั้งเดียว (payload: source/message/numbers[])
// จึง group message ที่ body เหมือนกันเป็น call เดียว ลดจำนวน request ตอนส่งหลายหมื่น
type EasyMoneySender struct {
	endpoint string
	source   string
	http     *http.Client
}

// EasyMoneyOption ปรับแต่งค่าของ EasyMoneySender
type EasyMoneyOption func(*EasyMoneySender)

// WithEasyMoneyEndpoint กำหนด endpoint เอง (เช่น ชี้ไป staging)
func WithEasyMoneyEndpoint(url string) EasyMoneyOption {
	return func(s *EasyMoneySender) {
		if url != "" {
			s.endpoint = url
		}
	}
}

// WithEasyMoneyHTTPClient ใส่ *http.Client เอง (กำหนด timeout/transport เอง)
func WithEasyMoneyHTTPClient(hc *http.Client) EasyMoneyOption {
	return func(s *EasyMoneySender) { s.http = hc }
}

// NewEasyMoneySender สร้าง sender โดย source = ชื่อผู้ส่งที่แนบไปกับ payload
func NewEasyMoneySender(source string, opts ...EasyMoneyOption) *EasyMoneySender {
	s := &EasyMoneySender{
		endpoint: easyMoneyDefaultEndpoint,
		source:   source,
		http:     &http.Client{Timeout: 15 * time.Second},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Channel บอกว่าเป็นช่องทาง SMS
func (s *EasyMoneySender) Channel() domain.Channel {
	return domain.ChannelSMS
}

// easyMoneyRequest = payload ที่ส่งไปยัง API
type easyMoneyRequest struct {
	Source  string   `json:"source"`
	Message string   `json:"message"`
	Numbers []string `json:"numbers"`
}

// easyMoneyResponse = ผลตอบกลับต่อ 1 call (ระดับ task ไม่ใช่รายเบอร์)
// เช่น {"success":true,"task_id":"255837653","message_id":"381230037"}
type easyMoneyResponse struct {
	Success   bool   `json:"success"`
	TaskID    string `json:"task_id"`
	MessageID string `json:"message_id"`
	// เผื่อ error message จาก provider (ยังไม่ทราบชื่อ field จริงตอน success=false)
	Message string `json:"message"`
	Error   string `json:"error"`
}

// Send ยิงหลายข้อความ — group ตาม body แล้วส่ง 1 call ต่อ 1 body
// per-message error ใส่ใน Result.Err (worker ตัดสิน retry จากตรงนั้น)
// คืน top-level error เฉพาะกรณีที่ระบบพังทั้งชุด (ไม่มี message ให้ส่ง)
func (s *EasyMoneySender) Send(ctx context.Context, msgs []domain.Message) ([]domain.Result, error) {
	if len(msgs) == 0 {
		return nil, fmt.Errorf("easymoney: ไม่มี message ให้ส่ง")
	}

	// group index ของ message ตาม body ที่เหมือนกัน
	groups := make(map[string][]int)
	order := make([]string, 0)
	for i, m := range msgs {
		if _, ok := groups[m.Payload.Body]; !ok {
			order = append(order, m.Payload.Body)
		}
		groups[m.Payload.Body] = append(groups[m.Payload.Body], i)
	}

	results := make([]domain.Result, len(msgs))
	for i, m := range msgs {
		results[i].NotificationID = m.NotificationID
	}

	for _, body := range order {
		idxs := groups[body]
		numbers := make([]string, len(idxs))
		for j, idx := range idxs {
			numbers[j] = msgs[idx].Recipient
		}

		messageID, sendErr := s.sendGroup(ctx, body, numbers)
		// API ตอบระดับ task (1 message_id ต่อ 1 call ไม่ใช่รายเบอร์)
		// เก็บ message_id ไว้ทุกตัวในกลุ่ม → DLR ต้องระบุเบอร์มาด้วยเพื่อ map กลับ
		// สถานะจริงรายเบอร์รอ DLR webhook มายืนยันทีหลัง
		for _, idx := range idxs {
			results[idx].ProviderMessageID = messageID
			results[idx].Err = sendErr
		}
	}

	return results, nil
}

// sendGroup ยิง 1 request (1 body หลายเบอร์)
// คืน (message_id, nil) = สำเร็จ / (\"\", err) = ล้มเหลว
func (s *EasyMoneySender) sendGroup(ctx context.Context, message string, numbers []string) (string, *domain.SendError) {
	payload, err := json.Marshal(easyMoneyRequest{
		Source:  s.source,
		Message: message,
		Numbers: numbers,
	})
	if err != nil {
		return "", &domain.SendError{Type: domain.ErrorPermanent, Code: "marshal", Message: err.Error()}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", &domain.SendError{Type: domain.ErrorPermanent, Code: "build_request", Message: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		// network/timeout → ชั่วคราว retry ได้
		return "", &domain.SendError{Type: domain.ErrorTemporary, Code: "network", Message: err.Error()}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &domain.SendError{
			Type:    classifyStatus(resp.StatusCode),
			Code:    fmt.Sprintf("http_%d", resp.StatusCode),
			Message: string(respBody),
		}
	}

	// 2xx: parse body — {"success":true,"task_id":"...","message_id":"..."}
	var body easyMoneyResponse
	if err := json.Unmarshal(respBody, &body); err != nil {
		// provider รับแล้ว (2xx) แต่ body ไม่ใช่ JSON ที่คาด → ถือว่ารับ แต่ไม่มี message_id
		// (ไม่ flip เป็น error เพราะ provider accept แล้ว) — log ให้เห็น response ดิบ
		return "", nil
	}

	if !body.Success {
		// 2xx แต่ provider บอกไม่สำเร็จ → ถือเป็นถาวร (payload/บัญชีมีปัญหา)
		// ยังไม่ทราบ field error จริงตอน fail → เก็บทั้ง message/error/raw ไว้ debug
		msg := firstNonEmpty(body.Error, body.Message, string(respBody))
		return "", &domain.SendError{Type: domain.ErrorPermanent, Code: "provider_rejected", Message: msg}
	}

	return body.MessageID, nil
}

// firstNonEmpty คืน string แรกที่ไม่ว่าง
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// classifyStatus แยก HTTP status เป็น error ชั่วคราว (retry) vs ถาวร
func classifyStatus(status int) domain.ErrorType {
	// 429 rate limit หรือ 5xx server → ลองใหม่ได้
	if status == http.StatusTooManyRequests || status >= 500 {
		return domain.ErrorTemporary
	}
	// 4xx อื่นๆ (payload ผิด/เบอร์ผิด) → ถาวร ไม่ต้อง retry
	return domain.ErrorPermanent
}
