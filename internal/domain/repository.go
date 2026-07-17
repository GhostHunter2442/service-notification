package domain

import "context"

// NotificationRepository = data access ของ notification
// service พึ่งพา interface นี้ ไม่ผูกกับ DB จริง → เปลี่ยนเป็น Azure SQL ภายหลัง
// โดยไม่ต้องแก้ service (dependency ชี้เข้า domain ตาม Clean Architecture)
type NotificationRepository interface {
	// SaveAll บันทึก/อัปเดต (upsert) notification หลายรายการในครั้งเดียว
	// เรียกทั้งตอนสร้าง (pending) และตอนอัปเดตผลหลังส่ง
	SaveAll(ctx context.Context, ns []Notification) error
}
