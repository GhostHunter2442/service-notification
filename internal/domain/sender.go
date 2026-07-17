package domain

import "context"

// Message = 1 ข้อความที่จะยิงออกไปหา provider
type Message struct {
	NotificationID string
	Recipient      string
	Payload        Payload
}

// Result = ผลการส่งรายข้อความ (Err == nil = สำเร็จ)
type Result struct {
	NotificationID    string
	ProviderMessageID string     // id ที่ provider คืนมา (เก็บไว้ map DLR)
	Err               *SendError // nil = สำเร็จ
}

// ErrorType แยก error ชั่วคราว (retry ได้) กับถาวร (mark failed เลย)
type ErrorType int

const (
	ErrorTemporary ErrorType = iota // network/timeout/rate limit -> retry
	ErrorPermanent                  // เบอร์ผิด/token invalid -> ไม่ต้อง retry
)

// SendError = error กลาง — worker ตัดสินใจ retry จากตรงนี้ ไม่ต้องรู้จัก provider
type SendError struct {
	Type    ErrorType
	Code    string
	Message string
}

func (e *SendError) Error() string {
	return e.Code + ": " + e.Message
}

// Sender = interface กลาง ทุก provider (sms/fcm/email) ต้อง implement
// ทำให้เปลี่ยน/เพิ่ม provider ได้โดยไม่แตะ business logic
type Sender interface {
	// Send ยิงหลายข้อความในครั้งเดียว (รองรับ bulk/multicast)
	Send(ctx context.Context, msgs []Message) ([]Result, error)
	// Channel บอกว่า sender นี้ส่งช่องทางไหน
	Channel() Channel
}
