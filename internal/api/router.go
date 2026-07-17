// Package api ประกอบ router + route ทั้งหมด
package api

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/GhostHunter2442/service-notification/docs" // docs ที่ generate ด้วย swag
	"github.com/GhostHunter2442/service-notification/internal/api/handler"
)

// NewRouter สร้าง gin engine + ผูก route
func NewRouter(h *handler.Handler) *gin.Engine {
	r := gin.Default()

	r.GET("/health", h.Health)
	r.POST("/test-send", h.TestSend)

	// Swagger UI → /swagger/index.html (ควรปิดบน production ด้วย env flag ภายหลัง)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
