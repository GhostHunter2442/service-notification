// Package service รวม business logic ของการส่ง notification
// orchestrate ระหว่าง sender (provider) กับ repository (data access)
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/GhostHunter2442/service-notification/internal/domain"
)

// Notification = service สำหรับส่ง notification
// รับ dependency เป็น interface ทั้งคู่ → test ง่าย, สลับ provider/DB ได้
type Notification struct {
	sender domain.Sender
	repo   domain.NotificationRepository
}

// New สร้าง service
func New(sender domain.Sender, repo domain.NotificationRepository) *Notification {
	return &Notification{sender: sender, repo: repo}
}

// SendSMS บันทึกสถานะ pending → ยิงผ่าน sender → อัปเดตผลตาม result
// คืน result รายข้อความให้ชั้นบนเอาไปแสดง
func (s *Notification) SendSMS(ctx context.Context, msgs []domain.Message) ([]domain.Result, error) {
	now := time.Now()

	// 1) สร้าง record สถานะ pending
	notifs := make([]domain.Notification, len(msgs))
	idx := make(map[string]int, len(msgs))
	for i, m := range msgs {
		notifs[i] = domain.Notification{
			ID:        m.NotificationID,
			Recipient: m.Recipient,
			Channel:   s.sender.Channel(),
			Payload:   m.Payload,
			Status:    domain.StatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		}
		idx[m.NotificationID] = i
	}
	if err := s.repo.SaveAll(ctx, notifs); err != nil {
		return nil, fmt.Errorf("save pending: %w", err)
	}

	// 2) ยิงผ่าน sender
	results, err := s.sender.Send(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}

	// 3) อัปเดตสถานะตามผล (accepted=sent / error=failed) แล้ว upsert อีกครั้ง
	updatedAt := time.Now()
	for _, r := range results {
		i, ok := idx[r.NotificationID]
		if !ok {
			continue
		}
		n := &notifs[i]
		n.ProviderMessageID = r.ProviderMessageID
		n.UpdatedAt = updatedAt
		if r.Err != nil {
			n.Status = domain.StatusFailed
			n.ErrorCode = r.Err.Code
			n.ErrorMessage = r.Err.Message
		} else {
			n.Status = domain.StatusSent
			n.SentAt = &updatedAt
		}
	}
	if err := s.repo.SaveAll(ctx, notifs); err != nil {
		return results, fmt.Errorf("save results: %w", err)
	}

	return results, nil
}
