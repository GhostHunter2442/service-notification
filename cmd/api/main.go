// cmd/api = entrypoint ของ API server (publisher)
package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/GhostHunter2442/service-notification/internal/api"
	"github.com/GhostHunter2442/service-notification/internal/api/handler"
	"github.com/GhostHunter2442/service-notification/internal/config"
	"github.com/GhostHunter2442/service-notification/internal/repository"
	"github.com/GhostHunter2442/service-notification/internal/sender/sms"
	"github.com/GhostHunter2442/service-notification/internal/service"
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

	// เปิด connection ไป Azure SQL
	db, err := repository.Open(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("connect database")
	}
	defer db.Close()

	// wire dependency: sender + repository -> service -> handler
	smsSender := sms.NewHTTPSender(cfg.SMS.Source, cfg.SMS.Endpoint)
	repo := repository.NewSQLServer(db)
	svc := service.New(smsSender, repo)

	h := handler.New(cfg.App.Env, svc)
	r := api.NewRouter(h)

	addr := fmt.Sprintf(":%d", cfg.App.HTTPPort)
	if err := r.Run(addr); err != nil {
		log.Fatal().Err(err).Msg("server stopped")
	}
}
