// Package handler รวม HTTP handler ของ API (ชั้น api)
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/GhostHunter2442/service-notification/internal/config"
	"github.com/GhostHunter2442/service-notification/internal/domain"
	"github.com/GhostHunter2442/service-notification/internal/sender/sms"
)

// Handler ถือ dependency ที่ handler ต้องใช้
type Handler struct {
	env    string
	smsCfg config.SMSConfig
}

// New สร้าง Handler
func New(env string, smsCfg config.SMSConfig) *Handler {
	return &Handler{env: env, smsCfg: smsCfg}
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

// TestSendRequest = payload ทดสอบส่ง SMS ผ่าน easymoney (ตรงตาม API จริง)
type TestSendRequest struct {
	Source  string   `json:"source" example:"easymoney"`                          // ว่างได้ = ใช้ค่าจาก config
	Message string   `json:"message" binding:"required" example:"ทดสอบ sms"`      // เนื้อความ
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
// @Summary      ทดสอบส่ง SMS ผ่าน easymoney
// @Description  ยิง SMS จริงผ่าน easymoney adapter (ยังไม่ผ่าน DB/queue) — คืน message_id ที่ provider ตอบกลับ
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

	source := req.Source
	if source == "" {
		source = h.smsCfg.Source
	}
	sender := sms.NewHTTPSender(source, h.smsCfg.Endpoint)

	// numbers ทุกเบอร์ใช้ body เดียวกัน → adapter จะ group เป็น 1 call
	msgs := make([]domain.Message, len(req.Numbers))
	for i, n := range req.Numbers {
		msgs[i] = domain.Message{
			NotificationID: "test-" + n,
			Recipient:      n,
			Payload:        domain.Payload{Body: req.Message},
		}
	}

	results, err := sender.Send(context.Background(), msgs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, TestSendResponse{
		Channel: string(sender.Channel()),
		Results: results,
	})
}
