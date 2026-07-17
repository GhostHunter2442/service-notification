// Package config โหลดค่าตั้งค่าจาก yaml + .env มาเป็น struct
package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config = ค่าตั้งค่าทั้งหมดของระบบ
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	SMS      SMSConfig      `mapstructure:"sms"`
	FCM      FCMConfig      `mapstructure:"fcm"`
	Email    EmailConfig    `mapstructure:"email"`
	Worker   WorkerConfig   `mapstructure:"worker"`
	Report   ReportConfig   `mapstructure:"report"`
}

type AppConfig struct {
	Env      string `mapstructure:"env"`
	HTTPPort int    `mapstructure:"http_port"`
}

type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	Encrypt  bool   `mapstructure:"encrypt"`
}

type RabbitMQConfig struct {
	URL      string `mapstructure:"url"`
	Exchange string `mapstructure:"exchange"`
}

type SMSConfig struct {
	Provider      string `mapstructure:"provider"`
	RatePerSecond int    `mapstructure:"rate_per_second"`
	MaxConcurrent int    `mapstructure:"max_concurrent"`
	BatchSize     int    `mapstructure:"batch_size"`
}

type FCMConfig struct {
	CredentialsFile string `mapstructure:"credentials_file"`
	RatePerSecond   int    `mapstructure:"rate_per_second"`
	MulticastSize   int    `mapstructure:"multicast_size"`
}

type EmailConfig struct {
	Provider string `mapstructure:"provider"`
	From     string `mapstructure:"from"`
}

type WorkerConfig struct {
	Count    int `mapstructure:"count"`
	Prefetch int `mapstructure:"prefetch"`
}

type ReportConfig struct {
	DLRTimeoutMinutes int `mapstructure:"dlr_timeout_minutes"`
}

// Load อ่าน .env เข้า environment ก่อน แล้วอ่าน yaml (แทน ${VAR} ด้วยค่าจาก env)
func Load(path string) (*Config, error) {
	// โหลด .env เข้า environment (ไม่ error ถ้าไม่มีไฟล์ — prod อาจตั้ง env ตรงๆ)
	_ = godotenv.Load()

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	// แทน ${DB_USER} ฯลฯ ด้วยค่าจริงจาก environment
	expanded := os.ExpandEnv(string(raw))

	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBufferString(expanded)); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}
