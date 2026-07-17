// Package sms รวม adapter ของ SMS provider ต่างๆ ใต้ domain.Sender
package sms

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/GhostHunter2442/service-notification/internal/domain"
)

// MockSender = sender ปลอมไว้ให้ระบบรัน end-to-end ได้ก่อนรู้ provider จริง
// พอรู้ provider แล้วเขียน adapter ตัวใหม่ (twilio.go/vonage.go) ใต้ interface เดิม
type MockSender struct{}

// NewMockSender สร้าง mock sender
func NewMockSender() *MockSender {
	return &MockSender{}
}

// Channel บอกว่าเป็นช่องทาง SMS
func (s *MockSender) Channel() domain.Channel {
	return domain.ChannelSMS
}

// Send แกล้งส่งสำเร็จทุกตัว + คืน provider_message_id ปลอม (log ให้เห็นว่าถูกเรียก)
func (s *MockSender) Send(ctx context.Context, msgs []domain.Message) ([]domain.Result, error) {
	results := make([]domain.Result, 0, len(msgs))
	for _, m := range msgs {
		pid := uuid.NewString()
		log.Info().
			Str("recipient", m.Recipient).
			Str("provider_message_id", pid).
			Str("body", m.Payload.Body).
			Msg("[mock-sms] sent")
		results = append(results, domain.Result{
			NotificationID:    m.NotificationID,
			ProviderMessageID: pid,
			Err:               nil,
		})
	}
	return results, nil
}
