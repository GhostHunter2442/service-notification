// Package handler รวม HTTP handler ของ API (ชั้น api)
// หน้าที่: parse/validate request → เรียก service → format response
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/GhostHunter2442/service-notification/internal/domain"
	"github.com/GhostHunter2442/service-notification/internal/service"
)

// Handler ถือ dependency ที่ handler ต้องใช้
type Handler struct {
	env string
	svc *service.Notification
}

// New สร้าง Handler
func New(env string, svc *service.Notification) *Handler {
	return &Handler{env: env, svc: svc}
}

// HealthResponse = ผลลัพธ์ health check
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
	Env    string `json:"env" example:"local"`
}

// Health godoc
// @Summary      Health check
// @Description  เช็คว่า service พร้อมทำงาน
// @Tags         system
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Router       /health [get]
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok", Env: h.env})
}

// TestSendRequest = payload ทดสอบส่ง SMS
// source มาจาก config (sms.source) ไม่รับจาก client แล้ว (best practice)
type TestSendRequest struct {
	Message string   `json:"message" binding:"required" example:"ทดสอบ sms"`       // เนื้อความ
	Numbers []string `json:"numbers" binding:"required,min=1" example:"0806906003"` // เบอร์ปลายทาง
}

// TestSendResponse = ผลการทดสอบส่ง
type TestSendResponse struct {
	Channel string          `json:"channel" example:"sms"`
	Results []domain.Result `json:"results"`
}

// ErrorResponse = รูปแบบ error กลาง
type ErrorResponse struct {
	Error string `json:"error"`
}

// TestSend godoc
// @Summary      ทดสอบส่ง SMS
// @Description  ยิง SMS จริงผ่าน service → sender (บันทึกสถานะผ่าน repository) — คืน message_id ที่ provider ตอบกลับ
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        request  body      TestSendRequest  true  "payload"
// @Success      200      {object}  TestSendResponse
// @Failure      400      {object}  ErrorResponse
// @Router       /test-send [post]
func (h *Handler) TestSend(c *gin.Context) {
	var req TestSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// numbers ทุกเบอร์ใช้ body เดียวกัน (service gen id + adapter group เป็น 1 call)
	msgs := make([]domain.Message, len(req.Numbers))
	for i, n := range req.Numbers {
		msgs[i] = domain.Message{
			Recipient: n,
			Payload:   domain.Payload{Body: req.Message},
		}
	}

	results, err := h.svc.SendSMS(c.Request.Context(), "test-send", msgs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, TestSendResponse{
		Channel: string(domain.ChannelSMS),
		Results: results,
	})
}
