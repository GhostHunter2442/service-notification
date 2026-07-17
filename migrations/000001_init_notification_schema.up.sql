-- ตารางของ notification service (อยู่ใน schema dbo)
-- เป็นตารางใหม่ ไม่ทับตารางเดิม (ถ้ามีชื่อชนจะ error ให้เห็น — ไม่ overwrite)

-- ============================================================
-- batches: งานส่ง 1 ครั้ง (แคมเปญ) — เก็บ counter สรุปผลระดับ batch
-- ============================================================
CREATE TABLE dbo.batches (
    id                UNIQUEIDENTIFIER NOT NULL CONSTRAINT PK_batches PRIMARY KEY,
    name              NVARCHAR(200)    NULL,
    channel           VARCHAR(10)      NOT NULL,          -- sms | fcm
    total             INT              NOT NULL CONSTRAINT DF_batches_total        DEFAULT 0,
    sent_count        INT              NOT NULL CONSTRAINT DF_batches_sent         DEFAULT 0,
    delivered_count   INT              NOT NULL CONSTRAINT DF_batches_delivered    DEFAULT 0,
    undelivered_count INT              NOT NULL CONSTRAINT DF_batches_undelivered  DEFAULT 0,
    failed_count      INT              NOT NULL CONSTRAINT DF_batches_failed       DEFAULT 0,
    pending_count     INT              NOT NULL CONSTRAINT DF_batches_pending      DEFAULT 0,
    status            VARCHAR(20)      NOT NULL,          -- scheduled|queued|processing|completed|failed
    send_date         DATETIME2(3)     NULL,              -- null = ส่งทันที
    approver_email    NVARCHAR(256)    NULL,              -- ส่งสรุปให้ใคร
    created_date      DATETIME2(3)     NOT NULL CONSTRAINT DF_batches_created      DEFAULT SYSUTCDATETIME(),
    completed_date    DATETIME2(3)     NULL
);

-- ============================================================
-- notifications: 1 แถวต่อผู้รับ 1 คน (ตารางใหญ่ หลายหมื่น/batch)
-- ============================================================
CREATE TABLE dbo.notifications (
    id                  UNIQUEIDENTIFIER NOT NULL CONSTRAINT PK_notifications PRIMARY KEY,
    batch_id            UNIQUEIDENTIFIER NOT NULL,
    customer_id         NVARCHAR(100)    NULL,            -- id ลูกค้าในระบบ — ระบุว่าใคร fail
    customer_name       NVARCHAR(200)    NULL,
    recipient           NVARCHAR(256)    NOT NULL,        -- เบอร์ / device token / email
    channel             VARCHAR(10)      NOT NULL,
    payload             NVARCHAR(MAX)    NULL,            -- JSON: title, body, data
    status              VARCHAR(20)      NOT NULL,        -- pending|queued|sent|delivered|undelivered|failed
    provider_message_id NVARCHAR(256)    NULL,            -- ใช้ map ตอน DLR callback กลับมา
    error_code          VARCHAR(50)      NULL,
    error_message       NVARCHAR(1000)   NULL,
    retry_count         INT              NOT NULL CONSTRAINT DF_notif_retry   DEFAULT 0,
    created_date        DATETIME2(3)     NOT NULL CONSTRAINT DF_notif_created DEFAULT SYSUTCDATETIME(),
    updated_date        DATETIME2(3)     NOT NULL CONSTRAINT DF_notif_updated DEFAULT SYSUTCDATETIME(),
    sent_date           DATETIME2(3)     NULL,
    delivered_date      DATETIME2(3)     NULL,
    CONSTRAINT FK_notifications_batch FOREIGN KEY (batch_id)
        REFERENCES dbo.batches(id)
);

-- index: ดึง report ตาม batch, map DLR จาก provider_message_id, filter ตาม status
CREATE INDEX IX_notifications_batch_id            ON dbo.notifications (batch_id);
CREATE INDEX IX_notifications_provider_message_id ON dbo.notifications (provider_message_id);
CREATE INDEX IX_notifications_batch_status        ON dbo.notifications (batch_id, status);
