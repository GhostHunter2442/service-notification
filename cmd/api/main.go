// cmd/api = entrypoint ของ API server (publisher)
package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/GhostHunter2442/service-notification/internal/api"
	"github.com/GhostHunter2442/service-notification/internal/api/handler"
	"github.com/GhostHunter2442/service-notification/internal/config"
	"github.com/GhostHunter2442/service-notification/internal/domain"
	"github.com/GhostHunter2442/service-notification/internal/sender/sms"
	"github.com/GhostHunter2442/service-notification/pkg/logger"
)

// @title           Service Notification API
// @version         1.0
// @description     API สำหรับส่ง SMS / FCM / Email notification (RabbitMQ + Azure SQL)
// @host            localhost:8080
// @BasePath        /
func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/config.example.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.App.Env)
	log.Info().
		Str("env", cfg.App.Env).
		Int("port", cfg.App.HTTPPort).
		Str("sms_provider", cfg.SMS.Provider).
		Msg("starting api server")

	// เลือก SMS sender ตาม config (ตอนนี้มีแค่ mock)
	var smsSender domain.Sender = sms.NewMockSender()

	h := handler.New(cfg.App.Env, smsSender)
	r := api.NewRouter(h)

	addr := fmt.Sprintf(":%d", cfg.App.HTTPPort)
	if err := r.Run(addr); err != nil {
		log.Fatal().Err(err).Msg("server stopped")
	}
}
