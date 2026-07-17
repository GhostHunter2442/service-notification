-- ย้อน migration: ลบเฉพาะ 2 ตารางของเรา
-- ลบ notifications ก่อน เพราะมี FK ชี้ไป batches
DROP TABLE IF EXISTS dbo.notifications;
DROP TABLE IF EXISTS dbo.batches;
