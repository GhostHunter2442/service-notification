package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/GhostHunter2442/service-notification/internal/config"
)

// TestTestSend_EndToEnd ยิง POST /test-send จริงผ่าน gin → easymoney adapter → mock provider
// พิสูจน์ full path โดยไม่แตะ provider จริง (endpoint ชี้ไป httptest)
func TestTestSend_EndToEnd(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// mock provider ตอบแบบ easymoney จริง
	var gotBody string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		_, _ = w.Write([]byte(`{"success":true,"task_id":"255837653","message_id":"381230037"}`))
	}))
	defer provider.Close()

	h := New("test", config.SMSConfig{Source: "easymoney", Endpoint: provider.URL})
	r := gin.New()
	r.POST("/test-send", h.TestSend)

	payload := `{"source":"easymoney","message":"ทดสอบ sms","numbers":["0806906003"]}`
	req := httptest.NewRequest(http.MethodPost, "/test-send", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("คาด 200, ได้ %d body=%s", w.Code, w.Body.String())
	}

	var resp TestSendResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Channel != "sms" {
		t.Errorf("คาด channel sms, ได้ %q", resp.Channel)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("คาด 1 result, ได้ %d", len(resp.Results))
	}
	if resp.Results[0].Err != nil {
		t.Errorf("ไม่ควร error: %v", resp.Results[0].Err)
	}
	if resp.Results[0].ProviderMessageID != "381230037" {
		t.Errorf("คาด message_id 381230037, ได้ %q", resp.Results[0].ProviderMessageID)
	}
	// payload ที่ส่งถึง provider ต้องมี source/message/numbers ครบ
	if !strings.Contains(gotBody, `"source":"easymoney"`) || !strings.Contains(gotBody, `"0806906003"`) {
		t.Errorf("payload ที่ส่งถึง provider ผิด: %s", gotBody)
	}
}

// TestTestSend_Validation เบอร์ว่าง → 400
func TestTestSend_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New("test", config.SMSConfig{Source: "easymoney"})
	r := gin.New()
	r.POST("/test-send", h.TestSend)

	req := httptest.NewRequest(http.MethodPost, "/test-send", strings.NewReader(`{"message":"hi","numbers":[]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("numbers ว่าง คาด 400, ได้ %d", w.Code)
	}
}
