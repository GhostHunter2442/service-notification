// Package repository เก็บ implementation ของ data access (domain.NotificationRepository)
package repository

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/GhostHunter2442/service-notification/internal/domain"
)

// Memory = NotificationRepository เก็บใน memory ล้วน
// ใช้ชั่วคราวให้ service รัน end-to-end ได้ก่อนต่อ Azure SQL
// พอทำ SQL repo แล้วสลับตัวนี้ออกได้เลย (interface เดียวกัน)
type Memory struct {
	mu    sync.Mutex
	store map[string]domain.Notification
}

// NewMemory สร้าง in-memory repository
func NewMemory() *Memory {
	return &Memory{store: make(map[string]domain.Notification)}
}

// SaveAll upsert notification ตาม ID (log ให้เห็นว่าถูกบันทึก/อัปเดต)
func (m *Memory) SaveAll(ctx context.Context, ns []domain.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, n := range ns {
		m.store[n.ID] = n
		log.Info().
			Str("id", n.ID).
			Str("recipient", n.Recipient).
			Str("status", string(n.Status)).
			Str("provider_message_id", n.ProviderMessageID).
			Msg("[repo] save notification")
	}
	return nil
}
