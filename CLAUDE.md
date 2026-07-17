# Service Notification

ระบบส่ง notification (SMS + Firebase/FCM + Email) ด้วย Go
ใช้ RabbitMQ เป็น queue กันข้อมูลสูญหาย และเก็บข้อมูลลง Database บน Azure

> ไฟล์นี้สรุป design + decision ที่ตกลงกันไว้ เพื่อให้เข้าใจโครงสร้างทันทีเมื่อกลับมาทำต่อ

---

## เป้าหมาย / Requirements

- ส่ง **SMS** และ **Firebase (FCM)** push notification
- ส่งครั้งละ **หลายหมื่นคน** ต้องเสร็จ **ภายใน 5 นาที หรือเร็วที่สุด**
- รองรับส่ง **ทันที (immediate)** และ **ตั้งเวลา (scheduled)**
- ใช้ **RabbitMQ** เป็น queue กันข้อมูลสูญหาย
- เก็บข้อมูล + สถานะลง **Database บน Azure**
- **สรุป report** ผลการส่งแต่ละครั้ง (total / success / undelivered / fail / pending)
- **ส่งอีเมลแจ้งผู้อนุมัติ** หลังส่งครบ พร้อมสรุปผล
- รายงานต้องระบุ **ลูกค้าที่ล้มเหลว** ได้ว่าใคร + เพราะอะไร

---

## สถาปัตยกรรมภาพรวม

```
                                          publish              consume
 Client/API ─> [API Server] ─> [บันทึก DB] ─────> [RabbitMQ] ─────> [Splitter] แตก batch เป็น message
                                                                          │
                                                                   publish เข้า channel queue
                                                                          │
                                                       ┌──────────────────┼──────────────────┐
                                                       ▼                  ▼                  ▼
                                                 [SMS Workers]      [FCM Workers]      (rate-limited, scale ได้)
                                                       │                  │
                                                       ▼                  ▼
                                                 SMS Provider        Firebase FCM
                                                       │
                                          DLR callback │ (มาทีหลัง)
                                                       ▼
                                              [Webhook] ─> update สถานะจริง (delivered/undelivered)
                                                       │
                            ทุก message จบ ─> atomic counter ที่ batch ─> ครบ ─> event "batch.completed"
                                                       │
                                          [Report Builder] ─> [Email Sender] ส่งสรุปให้ผู้อนุมัติ

 [Scheduler] poll DB ทุก ~1 นาที ─> ถึงเวลา send_at ─> publish เข้า queue (สำหรับงานตั้งเวลา)
```

**หลักการ:**
- API รับงาน → บันทึก DB (`pending`) → publish → ตอบกลับทันที (async)
- งานหนักทำใน background worker ที่ **scale แนวนอนได้**
- คอขวดจริงคือ **provider rate limit** ไม่ใช่ระบบเรา → throughput ทำเป็น **config ปรับได้**

---

## Decision ที่ตกลงแล้ว

| หัวข้อ | ข้อสรุป |
|--------|---------|
| ปริมาณ/เวลา | หลายหมื่น/ครั้ง ภายใน 5 นาทีหรือเร็วสุด (~167 msg/s ระบบทำได้สบาย) |
| Database | **Azure SQL (SQL Server)** — driver `go-mssqldb`, dev ต่อ Azure ตรง (ไม่มี DB ใน docker) |
| DB | ใช้ DB เดิม **`pawnshop`** — ตารางใหม่ `dbo.batches`, `dbo.notifications` (อยู่ใน `dbo`), migration CREATE อย่างเดียว ไม่แตะตารางเดิม |
| SMS provider | **ยังไม่ทราบ** → ออกแบบ provider-agnostic (interface + adapter) เริ่มด้วย mock sender |
| Immediate + Scheduled | รองรับทั้งคู่ — scheduled ใช้ DB polling |
| สถานะ | แยก **2 ชั้น**: accepted (ยิงเข้า provider) vs delivered (DLR ยืนยันจริง) |
| Report | สรุป total/success/undelivered/fail/pending ระดับ batch |
| แจ้งผู้อนุมัติ | email channel ยิงหลัง `batch.completed` |
| รายคนที่ fail | เก็บ customer_id/name + error → export CSV ได้ |

---

## จุดออกแบบสำคัญ (อย่าพลาด)

1. **สถานะ 2 ชั้น** — SMS "ยิงออก" ≠ "ส่งถึง" ต้องรอ **DLR webhook** มายืนยัน
   state: `pending → queued → sent → delivered / undelivered / failed`
2. **เก็บ `provider_message_id`** ตอนส่ง เพื่อ map กลับตอน DLR มาทีหลัง
3. **Fan-out**: API ไม่ publish ทีละหมื่นตัวใน request เดียว → รับเป็น batch job แล้วให้ Splitter แตก
4. **Rate limiter ต่อ provider** (`golang.org/x/time/rate`) กันโดนแบน — ค่าปรับใน config
5. **FCM multicast** ส่งได้ 500 token/call ลดจำนวน call มหาศาล
6. **Bulk insert/update DB** ไม่ทำทีละแถว
7. **Idempotency**: message มี id, worker เช็คก่อนส่งกัน redeliver ซ้ำ
8. **Retry**: exponential backoff + แยก error ชั่วคราว (retry) vs ถาวร (mark failed) + DLQ
9. **Batch completion**: atomic counter, มี **timeout** (รอ DLR 15–30 นาที) กัน batch ค้าง
10. **PII / PDPA**: report มีข้อมูลลูกค้า → **ไม่แนบ CSV ดิบในเมล**, ใช้ลิงก์ดาวน์โหลดที่ต้อง login + หมดอายุ (เก็บไฟล์บน Azure Blob + SAS URL)

---

## โครงสร้างไดเรกทอรี (เป้าหมาย)

```
service-notification/
├── cmd/
│   ├── api/          # entrypoint: API server (publisher)
│   ├── scheduler/    # poll DB งานตั้งเวลา → publish
│   ├── splitter/     # แตก batch → message รายตัว
│   └── worker/       # consumer: ส่งจริง (scale ได้)
│
├── internal/
│   ├── config/       # DB, RabbitMQ, provider, rate, worker (ปรับได้ทั้งหมด)
│   ├── domain/       # models + interfaces (ไม่ผูก framework)
│   ├── api/          # handler, middleware, router
│   ├── service/      # business logic
│   ├── batch/        # batch lifecycle + counter
│   ├── scheduler/    # logic ตั้งเวลา
│   ├── queue/        # RabbitMQ connection/publisher/consumer
│   ├── ratelimit/    # token bucket ต่อ provider
│   ├── sender/       # interface Sender + adapters
│   │   ├── sender.go      # interface กลาง + SendError (Temporary/Permanent)
│   │   ├── sms/          # หลาย adapter (twilio/vonage/thaibulk/mock)
│   │   ├── firebase/     # FCM (multicast)
│   │   └── email/        # ส่งสรุปผู้อนุมัติ (Azure Comm Services / SendGrid / SMTP)
│   ├── webhook/      # รับ DLR callback จาก provider
│   ├── report/       # builder.go, csv.go, storage.go (Azure Blob + SAS)
│   ├── repository/   # data access (Azure SQL / sqlserver)
│   └── worker/       # logic consume + เรียก sender + retry
│
├── pkg/              # logger ฯลฯ (reusable)
├── docs/             # Swagger docs (generate ด้วย `swag init`)
├── migrations/       # SQL migration
├── deployments/      # Dockerfile (app) — RabbitMQ อยู่ repo แยก, DB บน Azure
├── configs/          # config.example.yaml
├── go.mod            # module: github.com/GhostHunter2442/service-notification (Go 1.26.4)
├── CLAUDE.md
└── install.md        # คู่มือติดตั้ง step-by-step
```

---

## RabbitMQ Topology

> **Broker อยู่ repo แยก** `GhostHunter2442/rabbit-MQ` (server กลาง หลาย project ใช้ร่วม)
> project นี้ต่อผ่าน `RABBITMQ_URL` เข้า **vhost `notification`** — queue/exchange ข้างล่างอยู่ใน vhost นี้

```
exchange: notification (direct)
├── notification.batch     → Splitter consume
├── notification.sms       → SMS workers
├── notification.fcm       → FCM workers
├── notification.email     → Email workers (สรุปผู้อนุมัติ)
├── notification.retry     → TTL + กลับเข้า main (backoff)
└── notification.dlq       → dead letter (ตรวจสอบ manual)
```
ตั้งค่า: durable queue + persistent message + publisher confirm + manual ack

---

## Database Schema

> DB = **`pawnshop`** — ตารางใหม่ `dbo.batches`, `dbo.notifications` (schema `dbo`)
> migration แรก: [migrations/000001_init_notification_schema.up.sql](migrations/000001_init_notification_schema.up.sql) — SQL Server (UNIQUEIDENTIFIER, DATETIME2, NVARCHAR(MAX) สำหรับ JSON)

```sql
batches:
  id              PK
  name                          -- ชื่อแคมเปญ
  channel                       -- sms | fcm
  total, sent_count, delivered_count, undelivered_count, failed_count, pending_count
  status                        -- scheduled | queued | processing | completed | failed
  send_date       DATETIME2     -- null = ส่งทันที
  approver_email                -- ส่งสรุปให้ใคร
  created_date, completed_date

notifications:
  id              PK
  batch_id        FK
  customer_id                   -- id ลูกค้าในระบบ
  customer_name
  recipient                     -- เบอร์ / device token / email
  channel
  payload         JSON          -- title, body, data
  status                        -- pending|queued|sent|delivered|undelivered|failed
  provider_message_id           -- ใช้ map ตอน DLR callback กลับมา
  error_code, error_message
  retry_count
  created_date, updated_date, sent_date, delivered_date
```

---

## API Endpoints (ร่าง)

```
POST   /notifications/batch      สร้างงานส่ง (immediate ถ้าไม่ใส่ send_at / scheduled ถ้าใส่)
GET    /batches/:id              ดูสถานะ + ความคืบหน้า
DELETE /batches/:id              ยกเลิก (ก่อนถึงเวลาส่ง)
GET    /batches/:id/report       ดึงสรุป (JSON)
GET    /batches/:id/report.csv   ดาวน์โหลดรายคน (ต้อง auth, ลิงก์หมดอายุ)
POST   /webhooks/sms/:provider   รับ delivery report (DLR) จาก provider
GET    /swagger/index.html       Swagger UI (API docs) — ปิดบน production
```

> API doc สร้างด้วย swaggo: ใส่ annotation ที่ handler → `swag init -g cmd/api/main.go -o ./docs` → mount `gin-swagger`
> ต้อง generate ใหม่ทุกครั้งที่แก้ annotation (มีใน `make swagger`)

---

## Tech Stack แนะนำ

| ส่วน | Library |
|------|---------|
| HTTP | gin (หรือ chi/echo) |
| RabbitMQ | github.com/rabbitmq/amqp091-go |
| DB | **go-mssqldb** (Azure SQL / SQL Server) |
| FCM | firebase.google.com/go/v4 |
| Rate limit | golang.org/x/time/rate |
| Config | viper |
| Logger | zerolog / slog |
| API docs | swaggo (swag + gin-swagger) → Swagger UI |

---

## คำถามที่ยังค้าง (ต้องตอบก่อนเขียนโค้ดบางส่วน)

1. ~~**Azure DB** เป็นตัวไหน?~~ → **ตัดสินใจแล้ว: Azure SQL (SQL Server)** — driver `go-mssqldb`, dev ต่อ Azure ตรง
2. **SMS provider** เจ้าไหน? (ยังไม่รู้ → เริ่ม mock sender ก่อน) — พอรู้แล้วเพิ่ม adapter + ปรับ rate
3. **Email** ส่งผ่านอะไร? → Azure Communication Services / SendGrid / SMTP
4. รายงานคน fail: **ลิงก์ดาวน์โหลดที่ป้องกัน** (แนะนำ) หรือ **แนบไฟล์ในเมลตรง**?
5. รอ DLR นานสุดเท่าไหร่ก่อนปิด batch แล้วสรุป? (แนะนำ 15–30 นาที)

---

## สถานะงานปัจจุบัน

- [x] ออกแบบสถาปัตยกรรม + decision หลัก
- [x] init go.mod (branch `chore/init-go-module`, ยังไม่เปิด PR — ค้างขั้น push)
- [ ] วาด diagram สถาปัตยกรรมรวม
- [ ] scaffold โครงสร้าง package + interface หลัก + mock sender
- [ ] schema + migration
- [ ] implement ทีละ component

## ขั้นตอนถัดไปที่วางไว้

1. Scaffold โครงสร้างโค้ด (package + Sender interface + mock sender ให้รัน end-to-end ได้)
2. Schema + migration
3. Implement: queue → worker → sender → report → email ตามลำดับ
