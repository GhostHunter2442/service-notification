// cmd/smstest = เครื่องมือทดสอบส่ง SMS จริงผ่าน easymoney adapter
// รัน: go run ./cmd/smstest [เบอร์] [ข้อความ]
// ค่า default: อ่าน source/endpoint จาก config, ข้อความ "ทดสอบ sms"
//
// ⚠️ ส่ง SMS จริง — ยิงเข้า provider ตาม config จริง
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/GhostHunter2442/service-notification/internal/config"
	"github.com/GhostHunter2442/service-notification/internal/domain"
	"github.com/GhostHunter2442/service-notification/internal/sender/sms"
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

	// รับ arg: [เบอร์] [ข้อความ]
	number := "080690600388"
	message := "ทดสอบ sms"
	if len(os.Args) > 1 {
		number = os.Args[1]
	}
	if len(os.Args) > 2 {
		message = os.Args[2]
	}

	sender := sms.NewEasyMoneySender(cfg.SMS.Source, sms.WithEasyMoneyEndpoint(cfg.SMS.Endpoint))

	fmt.Printf("source  : %s\n", cfg.SMS.Source)
	fmt.Printf("number  : %s\n", number)
	fmt.Printf("message : %s\n", message)
	fmt.Println("ส่ง...")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	results, err := sender.Send(ctx, []domain.Message{{
		NotificationID: "smstest-1",
		Recipient:      number,
		Payload:        domain.Payload{Body: message},
	}})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ส่งไม่สำเร็จ: %v\n", err)
		os.Exit(1)
	}

	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("❌ ล้มเหลว (%s): %s\n", r.Err.Type, r.Err.Error())
			os.Exit(1)
		}
		fmt.Printf("✅ ส่งสำเร็จ — message_id=%s (provider รับแล้ว, สถานะส่งถึงจริงรอ DLR)\n", r.ProviderMessageID)
	}
}
