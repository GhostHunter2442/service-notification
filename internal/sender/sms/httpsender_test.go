package sms

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GhostHunter2442/service-notification/internal/domain"
)

// captureServer คืน test server + ตัวเก็บ payload ที่ถูกยิงเข้ามา
// respBody = body ที่ server จะตอบกลับ (ปล่อยว่างได้สำหรับเคส error status)
func captureServer(t *testing.T, status int, respBody string) (*httptest.Server, *[]sendRequest) {
	t.Helper()
	var got []sendRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req sendRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("payload ไม่ใช่ JSON ที่คาด: %v", err)
		}
		got = append(got, req)
		w.WriteHeader(status)
		if respBody != "" {
			_, _ = w.Write([]byte(respBody))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

const okResponse = `{"success":true,"task_id":"255837653","message_id":"381230037"}`

func TestHTTPSender_Success(t *testing.T) {
	srv, got := captureServer(t, http.StatusOK, okResponse)
	sender := NewHTTPSender("easymoney", srv.URL)

	results, err := sender.Send(context.Background(), []domain.Message{
		{NotificationID: "1", Recipient: "0801111111", Payload: domain.Payload{Body: "hello"}},
		{NotificationID: "2", Recipient: "0802222222", Payload: domain.Payload{Body: "hello"}},
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}

	// body เหมือนกัน → group เป็น 1 request ที่มี 2 เบอร์
	if len(*got) != 1 {
		t.Fatalf("คาด 1 request, ได้ %d", len(*got))
	}
	req := (*got)[0]
	if req.Source != "easymoney" || req.Message != "hello" {
		t.Errorf("payload ผิด: %+v", req)
	}
	if len(req.Numbers) != 2 {
		t.Errorf("คาด 2 เบอร์ใน 1 call, ได้ %d", len(req.Numbers))
	}
	for _, r := range results {
		if r.Err != nil {
			t.Errorf("ไม่ควร error: %v", r.Err)
		}
		// message_id จาก response เก็บลงทุกตัวในกลุ่ม (ใช้ map DLR)
		if r.ProviderMessageID != "381230037" {
			t.Errorf("คาด message_id 381230037, ได้ %q", r.ProviderMessageID)
		}
	}
}

func TestHTTPSender_ProviderRejected(t *testing.T) {
	// HTTP 200 แต่ success:false → ต้องเป็น error ถาวร
	srv, _ := captureServer(t, http.StatusOK, `{"success":false,"error":"invalid number"}`)
	sender := NewHTTPSender("easymoney", srv.URL)

	results, err := sender.Send(context.Background(), []domain.Message{
		{NotificationID: "1", Recipient: "bad", Payload: domain.Payload{Body: "x"}},
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if results[0].Err == nil {
		t.Fatal("success:false ต้องได้ error")
	}
	if results[0].Err.Type != domain.ErrorPermanent {
		t.Errorf("คาด permanent, ได้ %v", results[0].Err.Type)
	}
}

func TestHTTPSender_GroupsByBody(t *testing.T) {
	srv, got := captureServer(t, http.StatusOK, okResponse)
	sender := NewHTTPSender("easymoney", srv.URL)

	_, err := sender.Send(context.Background(), []domain.Message{
		{NotificationID: "1", Recipient: "0801111111", Payload: domain.Payload{Body: "A"}},
		{NotificationID: "2", Recipient: "0802222222", Payload: domain.Payload{Body: "B"}},
		{NotificationID: "3", Recipient: "0803333333", Payload: domain.Payload{Body: "A"}},
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	// 2 body ต่างกัน → 2 request
	if len(*got) != 2 {
		t.Fatalf("คาด 2 request (group ตาม body), ได้ %d", len(*got))
	}
}

func TestHTTPSender_ErrorClassification(t *testing.T) {
	cases := []struct {
		status int
		want   domain.ErrorType
	}{
		{http.StatusBadRequest, domain.ErrorPermanent},
		{http.StatusUnauthorized, domain.ErrorPermanent},
		{http.StatusTooManyRequests, domain.ErrorTemporary},
		{http.StatusInternalServerError, domain.ErrorTemporary},
	}
	for _, tc := range cases {
		srv, _ := captureServer(t, tc.status, "")
		sender := NewHTTPSender("easymoney", srv.URL)
		results, err := sender.Send(context.Background(), []domain.Message{
			{NotificationID: "1", Recipient: "0801111111", Payload: domain.Payload{Body: "x"}},
		})
		if err != nil {
			t.Fatalf("status %d: Send error: %v", tc.status, err)
		}
		if results[0].Err == nil {
			t.Fatalf("status %d: คาด error แต่ได้ nil", tc.status)
		}
		if results[0].Err.Type != tc.want {
			t.Errorf("status %d: คาด %v, ได้ %v", tc.status, tc.want, results[0].Err.Type)
		}
	}
}

func TestHTTPSender_EmptyEndpoint(t *testing.T) {
	// endpoint ว่าง → top-level error (config ผิด)
	sender := NewHTTPSender("easymoney", "")
	_, err := sender.Send(context.Background(), []domain.Message{
		{NotificationID: "1", Recipient: "0801111111", Payload: domain.Payload{Body: "x"}},
	})
	if err == nil {
		t.Fatal("endpoint ว่าง ต้องได้ error")
	}
}
