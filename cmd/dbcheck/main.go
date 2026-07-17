// cmd/dbcheck = เครื่องมือทดสอบการเชื่อมต่อ Azure SQL (read-only ล้วน)
// รัน: go run ./cmd/dbcheck
package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"time"

	_ "github.com/microsoft/go-mssqldb"

	"github.com/GhostHunter2442/service-notification/internal/config"
)

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

	// สร้าง connection string ของ go-mssqldb
	q := url.Values{}
	q.Add("database", cfg.Database.Name)
	if cfg.Database.Encrypt {
		q.Add("encrypt", "true")
	}
	dsn := (&url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(cfg.Database.User, cfg.Database.Password),
		Host:     fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port),
		RawQuery: q.Encode(),
	}).String()

	fmt.Printf("connecting to %s:%d db=%s user=%s encrypt=%v ...\n",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, cfg.Database.User, cfg.Database.Encrypt)

	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "PING FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ping OK ✅")

	// อ่านข้อมูลเบาๆ (read-only): ชื่อ DB + ตารางของเรามีหรือยัง + จำนวนตารางเดิม
	var dbName string
	var batchesID, notifID sql.NullInt64
	var dboTables int
	err = db.QueryRowContext(ctx, `
		SELECT DB_NAME(),
		       OBJECT_ID('dbo.batches'),
		       OBJECT_ID('dbo.notifications'),
		       (SELECT COUNT(*) FROM sys.tables t
		          JOIN sys.schemas s ON s.schema_id = t.schema_id
		         WHERE s.name = 'dbo')
	`).Scan(&dbName, &batchesID, &notifID, &dboTables)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("connected db       : %s\n", dbName)
	fmt.Printf("existing dbo tables: %d\n", dboTables)
	fmt.Printf("dbo.batches        : %s\n", existsLabel(batchesID))
	fmt.Printf("dbo.notifications  : %s\n", existsLabel(notifID))
}

func existsLabel(id sql.NullInt64) string {
	if id.Valid {
		return "มีอยู่แล้ว ✅"
	}
	return "ยังไม่มี — migration จะสร้างให้"
}
