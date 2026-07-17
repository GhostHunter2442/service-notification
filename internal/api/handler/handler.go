// Package handler รวม HTTP handler ของ API (ชั้น api)
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/GhostHunter2442/service-notification/internal/domain"
)

// Handler ถือ dependency ที่ handler ต้องใช้
type Handler struct {
	env       string
	smsSender domain.Sender
}

// New สร้าง Handler
func New(env string, smsSender domain.Sender) *Handler {
	return &Handler{env: env, smsSender: smsSender}
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

// TestSendRequest = payload ทดสอบส่ง
type TestSendRequest struct {
	Recipient string `json:"recipient" binding:"required" example:"0861234567"`
	Body      string `json:"body" binding:"required" example:"ทดสอบส่ง"`
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
// @Summary      ทดสอบส่งผ่าน mock sender
// @Description  demo ชั่วคราว — พิสูจน์ว่า Sender interface ทำงาน (ยังไม่ผ่าน DB/queue)
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
	results, err := h.smsSender.Send(context.Background(), []domain.Message{{
		NotificationID: "demo-1",
		Recipient:      req.Recipient,
		Payload:        domain.Payload{Body: req.Body},
	}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, TestSendResponse{
		Channel: string(h.smsSender.Channel()),
		Results: results,
	})
}
