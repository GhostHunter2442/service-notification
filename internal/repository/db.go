// Package repository เก็บ data access ต่อ Azure SQL (SQL Server)
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/microsoft/go-mssqldb" // driver sqlserver

	"github.com/GhostHunter2442/service-notification/internal/config"
)

// Open เปิด connection ไป Azure SQL แล้ว ping ให้แน่ใจว่าต่อได้
func Open(cfg config.DatabaseConfig) (*sql.DB, error) {
	q := url.Values{}
	q.Add("database", cfg.Name)
	if cfg.Encrypt {
		q.Add("encrypt", "true")
	}
	dsn := (&url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		RawQuery: q.Encode(),
	}).String()

	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}
