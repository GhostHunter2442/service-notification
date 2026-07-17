// Package domain เก็บ entity + business type หลักของระบบ
// ชั้นนี้ต้องไม่ import framework/infra ใดๆ (Clean Architecture — dependency ชี้เข้าใน)
package domain

import "time"

// Channel = ช่องทางส่ง notification
type Channel string

const (
	ChannelSMS   Channel = "sms"
	ChannelFCM   Channel = "fcm"
	ChannelEmail Channel = "email"
)

// Status = สถานะของ notification (state machine 2 ชั้น: accepted -> delivered)
// pending -> queued -> sent -> delivered | undelivered | failed
type Status string

const (
	StatusPending     Status = "pending"     // บันทึกแล้ว รอ publish
	StatusQueued      Status = "queued"       // อยู่ใน queue
	StatusSent        Status = "sent"         // ยิงเข้า provider แล้ว (accepted)
	StatusDelivered   Status = "delivered"    // DLR ยืนยันส่งถึง
	StatusUndelivered Status = "undelivered"  // DLR บอกส่งไม่ถึง
	StatusFailed      Status = "failed"       // ยิงไม่ออกตั้งแต่แรก
)

// Payload = เนื้อหาที่จะส่ง
type Payload struct {
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data,omitempty"` // ข้อมูลเสริม (โดยเฉพาะ FCM)
}

// Notification = 1 ข้อความที่ส่งถึงลูกค้า 1 คน
type Notification struct {
	ID                string
	BatchID           string
	CustomerID        string // id ลูกค้าในระบบ — ใช้ระบุว่าใคร fail
	CustomerName      string
	Recipient         string // เบอร์ / device token / email
	Channel           Channel
	Payload           Payload
	Status            Status
	ProviderMessageID string // ใช้ map ตอน DLR callback กลับมา
	ErrorCode         string
	ErrorMessage      string
	RetryCount        int
	CreatedAt         time.Time
	UpdatedAt         time.Time
	SentAt            *time.Time
	DeliveredAt       *time.Time
}

// BatchStatus = สถานะของงานส่งทั้ง batch
type BatchStatus string

const (
	BatchScheduled  BatchStatus = "scheduled"
	BatchQueued     BatchStatus = "queued"
	BatchProcessing BatchStatus = "processing"
	BatchCompleted  BatchStatus = "completed"
	BatchFailed     BatchStatus = "failed"
)

// Batch = งานส่ง 1 ครั้ง (แคมเปญ) มีหลาย Notification
type Batch struct {
	ID               string
	Name             string
	Channel          Channel
	Total            int
	SentCount        int
	DeliveredCount   int
	UndeliveredCount int
	FailedCount      int
	PendingCount     int
	Status           BatchStatus
	SendAt           *time.Time // nil = ส่งทันที
	ApproverEmail    string     // ส่งสรุปให้ใคร
	CreatedAt        time.Time
	CompletedAt      *time.Time
}
