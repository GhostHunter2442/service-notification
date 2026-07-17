// Package service รวม business logic ของการส่ง notification
// orchestrate ระหว่าง sender (provider) กับ repository (data access)
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

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

// SendSMS สร้าง batch → บันทึก notifications (pending) → ยิงผ่าน sender
// → อัปเดตผลรายแถว → ปิด batch พร้อม counter สรุป
func (s *Notification) SendSMS(ctx context.Context, batchName string, msgs []domain.Message) ([]domain.Result, error) {
	now := time.Now().UTC()
	channel := s.sender.Channel()

	// 1) สร้าง batch (ต้องมีก่อน notifications เพราะ FK)
	batch := domain.Batch{
		ID:           uuid.NewString(),
		Name:         batchName,
		Channel:      channel,
		Total:        len(msgs),
		PendingCount: len(msgs),
		Status:       domain.BatchProcessing,
		CreatedAt:    now,
	}
	if err := s.repo.CreateBatch(ctx, batch); err != nil {
		return nil, fmt.Errorf("create batch: %w", err)
	}

	// 2) สร้าง notification (pending) — gen GUID id ให้แต่ละแถว
	notifs := make([]domain.Notification, len(msgs))
	idx := make(map[string]int, len(msgs))
	for i := range msgs {
		if msgs[i].NotificationID == "" {
			msgs[i].NotificationID = uuid.NewString()
		}
		notifs[i] = domain.Notification{
			ID:        msgs[i].NotificationID,
			BatchID:   batch.ID,
			Recipient: msgs[i].Recipient,
			Channel:   channel,
			Payload:   msgs[i].Payload,
			Status:    domain.StatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		}
		idx[msgs[i].NotificationID] = i
	}
	if err := s.repo.CreateNotifications(ctx, notifs); err != nil {
		return nil, fmt.Errorf("create notifications: %w", err)
	}

	// 3) ยิงผ่าน sender
	results, err := s.sender.Send(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}

	// 4) อัปเดตสถานะตามผล (accepted=sent / error=failed) + นับ counter
	updatedAt := time.Now().UTC()
	var sent, failed int
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
			failed++
		} else {
			n.Status = domain.StatusSent
			n.SentAt = &updatedAt
			sent++
		}
	}
	if err := s.repo.UpdateNotificationResults(ctx, notifs); err != nil {
		return results, fmt.Errorf("update notifications: %w", err)
	}

	// 5) ปิด batch พร้อม counter สรุป
	batch.SentCount = sent
	batch.FailedCount = failed
	batch.PendingCount = 0
	batch.Status = domain.BatchCompleted
	batch.CompletedAt = &updatedAt
	if err := s.repo.UpdateBatch(ctx, batch); err != nil {
		return results, fmt.Errorf("update batch: %w", err)
	}

	return results, nil
}
