package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GhostHunter2442/service-notification/internal/domain"
)

// SQLServer = NotificationRepository ที่ insert/update ลง dbo.batches + dbo.notifications จริง
type SQLServer struct {
	db *sql.DB
}

// NewSQLServer สร้าง repository จาก *sql.DB ที่เปิดไว้แล้ว
func NewSQLServer(db *sql.DB) *SQLServer {
	return &SQLServer{db: db}
}

// CreateBatch insert batch 1 แถว (คอลัมน์อื่นใช้ DEFAULT ของตาราง)
func (r *SQLServer) CreateBatch(ctx context.Context, b domain.Batch) error {
	const q = `
INSERT INTO dbo.batches (id, name, channel, total, pending_count, status, created_date)
VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7)`
	_, err := r.db.ExecContext(ctx, q,
		b.ID, nullStr(b.Name), string(b.Channel), b.Total, b.PendingCount, string(b.Status), b.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert batch: %w", err)
	}
	return nil
}

// UpdateBatch อัปเดต counter + สถานะ + completed_date
func (r *SQLServer) UpdateBatch(ctx context.Context, b domain.Batch) error {
	const q = `
UPDATE dbo.batches
   SET sent_count = @p2, failed_count = @p3, pending_count = @p4,
       status = @p5, completed_date = @p6
 WHERE id = @p1`
	_, err := r.db.ExecContext(ctx, q,
		b.ID, b.SentCount, b.FailedCount, b.PendingCount, string(b.Status), nullTime(b.CompletedAt))
	if err != nil {
		return fmt.Errorf("update batch: %w", err)
	}
	return nil
}

// CreateNotifications insert notifications ทั้งชุดใน transaction เดียว
func (r *SQLServer) CreateNotifications(ctx context.Context, ns []domain.Notification) error {
	return r.inTx(ctx, func(tx *sql.Tx) error {
		const q = `
INSERT INTO dbo.notifications
  (id, batch_id, customer_id, customer_name, recipient, channel, payload, status, created_date, updated_date)
VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10)`
		for _, n := range ns {
			payload, err := json.Marshal(n.Payload)
			if err != nil {
				return fmt.Errorf("marshal payload %s: %w", n.ID, err)
			}
			if _, err := tx.ExecContext(ctx, q,
				n.ID, n.BatchID, nullStr(n.CustomerID), nullStr(n.CustomerName),
				n.Recipient, string(n.Channel), string(payload), string(n.Status),
				n.CreatedAt, n.UpdatedAt,
			); err != nil {
				return fmt.Errorf("insert notification %s: %w", n.ID, err)
			}
		}
		return nil
	})
}

// UpdateNotificationResults อัปเดตสถานะรายแถวหลังส่ง (ใน transaction เดียว)
func (r *SQLServer) UpdateNotificationResults(ctx context.Context, ns []domain.Notification) error {
	return r.inTx(ctx, func(tx *sql.Tx) error {
		const q = `
UPDATE dbo.notifications
   SET status = @p2, provider_message_id = @p3, error_code = @p4,
       error_message = @p5, sent_date = @p6, updated_date = @p7
 WHERE id = @p1`
		for _, n := range ns {
			if _, err := tx.ExecContext(ctx, q,
				n.ID, string(n.Status), nullStr(n.ProviderMessageID),
				nullStr(n.ErrorCode), nullStr(n.ErrorMessage),
				nullTime(n.SentAt), n.UpdatedAt,
			); err != nil {
				return fmt.Errorf("update notification %s: %w", n.ID, err)
			}
		}
		return nil
	})
}

// inTx รัน fn ใน transaction — commit ถ้าสำเร็จ, rollback ถ้า error
func (r *SQLServer) inTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// nullStr แปลง "" เป็น NULL (สำหรับคอลัมน์ nullable)
func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullTime แปลง nil *time.Time เป็น NULL
func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}
