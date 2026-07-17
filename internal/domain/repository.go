package domain

import "context"

// NotificationRepository = data access ของ batch + notification
// service พึ่งพา interface นี้ ไม่ผูก DB จริง → สลับ implementation ได้
// (dependency ชี้เข้า domain ตาม Clean Architecture)
type NotificationRepository interface {
	// CreateBatch insert batch 1 แถว (ต้องมีก่อน notifications เพราะ FK)
	CreateBatch(ctx context.Context, b Batch) error
	// UpdateBatch อัปเดต counter + สถานะ batch หลังส่งเสร็จ
	UpdateBatch(ctx context.Context, b Batch) error
	// CreateNotifications insert notifications (สถานะ pending)
	CreateNotifications(ctx context.Context, ns []Notification) error
	// UpdateNotificationResults อัปเดตสถานะ/provider_message_id/error รายแถวหลังส่ง
	UpdateNotificationResults(ctx context.Context, ns []Notification) error
}
