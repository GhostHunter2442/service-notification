// Package logger ตั้งค่า global logger (zerolog)
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init ตั้งค่า logger — dev ใช้ console อ่านง่าย, prod ใช้ JSON
func Init(env string) {
	zerolog.TimeFieldFormat = time.RFC3339
	if env == "local" || env == "dev" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen})
	} else {
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}
}
