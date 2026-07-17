# Install & Setup Guide — Service Notification

คู่มือติดตั้งตั้งแต่เครื่องเปล่า จนพร้อมเขียนโค้ดและรัน local
(คำสั่งเป็น **Windows / PowerShell** เป็นหลัก — มีหมายเหตุ mac/linux ที่จำเป็น)

> อ่าน [CLAUDE.md](CLAUDE.md) ประกอบเพื่อเข้าใจสถาปัตยกรรมรวม

---

## สารบัญ

- [Step 0 — ภาพรวมสิ่งที่ต้องติดตั้ง](#step-0)
- [Step 1 — ติดตั้ง Go](#step-1)
- [Step 2 — ติดตั้ง Git](#step-2)
- [Step 3 — ติดตั้ง Docker Desktop](#step-3)
- [Step 4 — ติดตั้ง Dev Tools (CLI)](#step-4)
- [Step 5 — วางโครงสร้างโปรเจค (best practice)](#step-5)
- [Step 6 — ติดตั้ง Go dependencies](#step-6)
- [Step 7 — ไฟล์ config / .env](#step-7)
- [Step 8 — เชื่อมต่อ RabbitMQ server กลาง](#step-8)
- [Step 9 — Makefile / Taskfile](#step-9)
- [Step 10 — ตั้งค่า Swagger (API docs)](#step-10)
- [Step 11 — ตรวจสอบว่าทุกอย่างพร้อม](#step-11)

---

<a name="step-0"></a>
## Step 0 — ภาพรวมสิ่งที่ต้องติดตั้ง

| # | ตัว | ทำไมต้องมี | จำเป็น |
|---|-----|-----------|--------|
| 1 | **Go 1.26.4** | ภาษาหลัก | ✅ |
| 2 | **Git** | version control | ✅ |
| 3 | **Docker Desktop** | ใช้รัน RabbitMQ server กลาง (repo แยก `rabbit-MQ`) | ✅ |
| 4 | **golang-migrate** | จัดการ DB migration | ✅ |
| 5 | **golangci-lint** | ตรวจคุณภาพโค้ด (lint) | ✅ |
| 6 | **swag (Swagger)** | สร้าง API documentation | ✅ |
| 7 | **Air** | hot reload ตอน dev | 🔧 แนะนำ |
| 8 | **Make** (หรือ Task) | รวมคำสั่ง build/run/test | 🔧 แนะนำ |
| 9 | **VS Code + Go extension** | IDE | 🔧 แนะนำ |

> **Infra ของโปรเจคนี้:**
> - **Database** → Azure SQL (ต่อตรงจากเครื่อง dev)
> - **RabbitMQ** → **server กลางแยก repo** (`GhostHunter2442/rabbit-MQ`) — project นี้แค่ต่อเข้าไปผ่าน `RABBITMQ_URL` ไม่ได้รัน broker เอง
> - โปรเจคนี้จึง **ไม่มี docker-compose ของตัวเอง**

### อธิบายละเอียด — แต่ละตัวทำงานส่วนไหนในโปรเจค

**1. Go** — ตัวภาษาและ compiler
ใช้เขียนทุกอย่างในระบบ: API server, worker, scheduler, splitter รวมถึง compile เป็น binary ไปรัน production มาพร้อม `go` command ที่ใช้ build / test / จัดการ dependency (`go mod`)

**2. Git** — เก็บประวัติโค้ด (version control)
ใช้ track การเปลี่ยนแปลง, แตก branch ทำฟีเจอร์, ส่งขึ้น GitHub (`GhostHunter2442/service-notification`) และเปิด PR ไม่เกี่ยวกับ runtime ของแอปโดยตรง แต่เป็นหัวใจของการทำงานเป็นทีม

**3. Docker Desktop** — ใช้รัน RabbitMQ server กลาง
โปรเจคนี้ **ไม่ได้รัน RabbitMQ เอง** — broker อยู่ในอีก repo (`rabbit-MQ`) ที่รันด้วย Docker
เราลง Docker Desktop เพื่อ **ไปรัน repo นั้น** (ครั้งเดียว) แล้ว service-notification ค่อยต่อเข้าไปผ่าน `RABBITMQ_URL`
- ส่วน **Database** ใช้ **Azure SQL** ต่อตรงจากเครื่อง dev (ไม่ต้องมี DB ใน docker)
> ถ้าในทีมมี RabbitMQ กลางรันอยู่แล้ว (บน server/เครื่องเพื่อน) คุณอาจไม่ต้องลง Docker บนเครื่องตัวเอง แค่ต่อ URL ไปหา broker นั้น

**4. golang-migrate** — จัดการโครงสร้างตาราง database (schema migration)
เวลาสร้าง/แก้ตาราง (`batches`, `notifications`) เราไม่ไปแก้ DB ด้วยมือ แต่เขียนเป็นไฟล์ SQL เวอร์ชัน (`001_xxx.up.sql` / `.down.sql`) แล้วให้ tool นี้รันตามลำดับ → ทุก environment (local/staging/prod) ได้ schema ตรงกัน และ **rollback ได้** ถ้าผิด

**5. golangci-lint** — ตรวจคุณภาพโค้ดอัตโนมัติ (linter)
รวม linter หลายตัวไว้ในคำสั่งเดียว คอยจับ: bug ที่มองไม่เห็น, error ที่ลืมเช็ค, ช่องโหว่ security (`gosec`), โค้ดที่ไม่ได้ใช้ ฯลฯ รันก่อน commit / ใน CI เพื่อคุมมาตรฐานโค้ดทั้งทีมให้เท่ากัน

**6. swag (Swagger)** — สร้างเอกสาร API อัตโนมัติ
อ่าน annotation (comment พิเศษ) ที่เราเขียนไว้บน handler แล้ว generate เป็นหน้าเว็บ Swagger UI ที่คนเรียก API (frontend/ทีมอื่น) เข้ามาดูได้ว่ามี endpoint อะไร รับ-ส่งข้อมูลหน้าตาไหน และ **กดทดลองยิง API ได้จากหน้าเว็บเลย** — ไม่ต้องเขียน doc มือ

**7. Air** — hot reload ตอน develop
ปกติแก้โค้ด Go ต้อง stop → `go run` ใหม่ทุกครั้ง เสียเวลา Air คอยจับว่าไฟล์เปลี่ยน แล้ว **rebuild + restart ให้อัตโนมัติ** ใช้เฉพาะตอน dev (ไม่ใช้ตอน production) ช่วยให้ลูปแก้-เห็นผลเร็วขึ้นมาก
> รันแค่ `air` ที่ root — repo มีไฟล์ `.air.toml` ตั้งไว้แล้ว (build `./cmd/api` → `tmp/main.exe`)
> ⚠️ ถ้าไม่มี `.air.toml` air จะ build package root (`.`) ซึ่งไม่มี main → error `tmp\main.exe not recognized`

**8. Make / Task** — รวมคำสั่งยาวๆ ให้สั้น
แทนที่จะจำคำสั่ง migrate ที่ยาวเหยียด เราเก็บไว้ใน `Makefile` แล้วเรียกสั้นๆ เช่น `make api`, `make migrate-up`, `make swagger` เป็น "ปุ่มลัด" ของงานที่ทำบ่อย ทั้งทีมใช้คำสั่งชุดเดียวกัน

**9. VS Code + Go extension** — เครื่องมือเขียนโค้ด
ให้ autocomplete, จับ error ตอนพิมพ์, jump ไปดู definition, debug ได้ ไม่บังคับ (ใช้ IDE อื่นได้) แต่ทำให้เขียน Go สะดวกขึ้นมาก

---

<a name="step-1"></a>
## Step 1 — ติดตั้ง Go

```powershell
winget install --id GoLang.Go -e
```
> **คำสั่งนี้ทำอะไร:** `winget` คือตัวจัดการโปรแกรมของ Windows (เหมือน app store แบบ command line)
> - `install` = ติดตั้งโปรแกรม
> - `--id GoLang.Go` = ระบุ ID โปรแกรมที่จะลง (ตัว Go อย่างเป็นทางการ)
> - `-e` = exact match ให้ตรง ID เป๊ะ ไม่เดา/ลงตัวใกล้เคียงผิด

ปิด-เปิด terminal ใหม่ แล้วตรวจสอบ:

```powershell
go version    # ต้องได้ go version go1.26.x
```
> **ทำไปทำไม:** เช็คว่า Go ติดตั้งสำเร็จและ terminal มองเห็น ถ้าขึ้นเลขเวอร์ชัน = พร้อมใช้

ตั้ง PATH ให้ binary ที่ `go install` ติดตั้งเรียกใช้ได้ (สำคัญมากสำหรับ Step 4):

```powershell
# เพิ่ม %USERPROFILE%\go\bin เข้า PATH ถาวร
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:USERPROFILE\go\bin", "User")
```
> **ทำไปทำไม:** เครื่องมือที่ลงด้วย `go install` (migrate, swag, air ใน Step 4) จะไปอยู่ในโฟลเดอร์ `C:\Users\<คุณ>\go\bin`
> ถ้าโฟลเดอร์นี้ไม่อยู่ใน PATH → พิมพ์ `migrate` / `swag` แล้วเครื่องจะหาไม่เจอ (command not found)
> คำสั่งนี้เลย "บอก Windows ให้รู้จักโฟลเดอร์นั้น" แบบถาวร

> ปิด-เปิด terminal ใหม่หลังตั้ง PATH (เพื่อให้ค่า PATH ใหม่มีผล)

---

<a name="step-2"></a>
## Step 2 — ติดตั้ง Git

```powershell
winget install --id Git.Git -e
git --version
```
> **ทำไปทำไม:** ลง Git แล้วเช็คเวอร์ชันว่าใช้ได้ Git คือตัวเก็บประวัติโค้ด + ส่งขึ้น GitHub

ตั้งค่าเบื้องต้น (ถ้ายังไม่เคย):

```powershell
git config --global user.name "ชื่อคุณ"
git config --global user.email "email@example.com"
```
> **ทำไปทำไม:** บอก Git ว่า "ใครเป็นคน commit" ชื่อ+อีเมลนี้จะติดไปกับทุก commit ที่คุณทำ
> `--global` = ตั้งครั้งเดียวใช้กับทุก repo บนเครื่อง (ไม่ต้องตั้งใหม่ทุกโปรเจค)

---

<a name="step-3"></a>
## Step 3 — ติดตั้ง Docker Desktop

ใช้รัน RabbitMQ และ Database บนเครื่อง local (ไม่ต้องลง native)

```powershell
winget install --id Docker.DockerDesktop -e
```
> **ทำไปทำไม:** ลง Docker Desktop ตัวโปรแกรมที่รัน container (RabbitMQ) บนเครื่อง
> ต้อง **เปิดโปรแกรม Docker Desktop ค้างไว้** เวลาจะใช้ container (มันคือ engine เบื้องหลัง)

- เปิด Docker Desktop รอจน status = **Running**
- ตรวจสอบ:

```powershell
docker --version
docker compose version
```
> **ทำไปทำไม:** เช็คว่า Docker กับ Docker Compose ใช้งานได้
> - `docker` = สั่งงาน container ทีละตัว
> - `docker compose` = สั่งงานหลาย container พร้อมกันตามไฟล์ `docker-compose.yml` (ใช้ตอนรัน RabbitMQ กลางใน repo `rabbit-MQ`)

---

<a name="step-4"></a>
## Step 4 — ติดตั้ง Dev Tools (CLI)

ติดตั้งผ่าน `go install` (ลงใน `%USERPROFILE%\go\bin` ที่ตั้ง PATH ไว้แล้ว):

> **`go install ...@latest` คืออะไร:** ให้ Go ดาวน์โหลด source ของเครื่องมือ → compile → วาง binary ไว้ที่ `go\bin`
> `@latest` = เอาเวอร์ชันล่าสุด หลังลงเสร็จจะเรียกใช้เป็นคำสั่งได้เลย (เพราะตั้ง PATH ไว้แล้ว)

```powershell
# DB migration tool — สร้าง/แก้/rollback ตารางใน database
go install -tags 'sqlserver' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
#   ↑ -tags 'sqlserver' = build ให้รองรับ Azure SQL / SQL Server (migrate รองรับหลาย DB ต้องเลือกตอนลง)

# Linter — ตัวตรวจคุณภาพ/หา bug ในโค้ดก่อน commit
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

# Hot reload ตอน dev — แก้โค้ดแล้ว rebuild + restart ให้อัตโนมัติ ไม่ต้องหยุดรันเอง
go install github.com/air-verse/air@latest

# Swagger generator — อ่าน annotation ในโค้ด แล้วสร้างหน้า API docs
go install github.com/swaggo/swag/cmd/swag@latest
```

**ติดตั้งตัวรวมคำสั่ง (Make หรือ Task) — เลือกอย่างใดอย่างหนึ่ง:**

#### Make/Task คืออะไร และทำไมต้องใช้?

มันคือ **"ตัวจดคำสั่ง"** — เอาคำสั่งยาวๆ ที่ต้องพิมพ์บ่อยมาตั้งชื่อเล่นสั้นๆ ไว้ แล้วเรียกด้วยชื่อสั้นแทน

**ปัญหาถ้าไม่มี:** ในโปรเจคนี้ คำสั่งที่ต้องใช้ประจำมัน **ยาวและจำยากมาก** เช่น เวลาจะรัน migration ต้องพิมพ์:

```powershell
# ❌ ต้องพิมพ์ยาวขนาดนี้ทุกครั้ง แถมพิมพ์ผิดนิดเดียวก็ error
migrate -path migrations -database "sqlserver://user:pass@xxx.database.windows.net:1433?database=pawnshop" up
```

**เมื่อมี Make/Task:** เราจดคำสั่งยาวข้างบนไว้ในไฟล์ (Step 9) แล้วตั้งชื่อว่า `migrate-up` ต่อไปแค่พิมพ์:

```powershell
# ✅ สั้น จำง่าย พิมพ์ไม่ผิด
make migrate-up          # (ถ้าใช้ make)
task migrate-up          # (ถ้าใช้ task)
```

**ประโยชน์ที่ได้:**
| ข้อดี | อธิบาย |
|-------|--------|
| ไม่ต้องจำคำสั่งยาว | จำแค่ `make api`, `make migrate-up`, `make swagger` |
| พิมพ์ผิดยาก | คำสั่งจริงถูกจดไว้แล้ว ไม่ต้องพิมพ์ url/flag เองทุกครั้ง |
| ทั้งทีมใช้เหมือนกัน | เพื่อนร่วมทีม clone โปรเจคมา ใช้ `make api` ได้เลย ไม่ต้องถามว่ารันยังไง |
| เป็นเอกสารในตัว | เปิดไฟล์ Makefile ดู = รู้ว่าโปรเจคนี้ทำอะไรได้บ้าง |

> ⚠️ ตอนนี้แค่ **"ติดตั้งตัวโปรแกรม"** ก่อน — ส่วน "รายการคำสั่งลัด" จริงๆ จะไปสร้างใน **Step 9** (ไฟล์ `Makefile`)
> ถ้ายังไม่อยากใช้ตอนนี้ ข้ามได้ แล้วพิมพ์คำสั่งเต็มๆ เอาก็ได้ (แค่ยาวกว่า) — ไม่ใช่ของบังคับ

```powershell
# ตัวเลือก A: make — มาตรฐานดั้งเดิม (ไฟล์ชื่อ Makefile) คนส่วนใหญ่คุ้นเคย มีในโลก Linux/Mac มานาน
winget install --id GnuWin32.Make -e

# ตัวเลือก B (แนะนำบน Windows): Task — รุ่นใหม่ ใช้ไฟล์ YAML อ่านง่าย ทำงานข้าม Windows/Mac/Linux ได้ดีกว่า
winget install --id Task.Task -e
```
> **เลือกตัวไหนดี:** ถ้าทีมคุ้น make อยู่แล้ว → **A**; ถ้าเริ่มใหม่บน Windows → แนะนำ **B (Task)**
> เพราะ make บน Windows บางทีมีปัญหาเรื่อง syntax/การเว้น tab ส่วน Task ออกแบบมาให้ข้าม OS ตั้งแต่แรก
> **เลือกลงแค่ตัวเดียวพอ** ไม่ต้องลงทั้งคู่

ตรวจสอบว่าทุกตัวลงสำเร็จ (แต่ละคำสั่งควรขึ้นเลขเวอร์ชัน):

```powershell
migrate -version          # migration tool
golangci-lint version     # linter
air -v                    # hot reload
swag --version            # swagger generator
```

> ถ้า command not found → PATH ยังไม่ถูก กลับไปเช็ค Step 1 (ปิด-เปิด terminal ใหม่ด้วย)

---

<a name="step-5"></a>
## Step 5 — วางโครงสร้างโปรเจค (best practice)

โครงสร้างตาม [Standard Go Project Layout](https://github.com/golang-standards/project-layout) + Clean Architecture

สร้างโฟลเดอร์ทั้งหมด (รันที่ root ของโปรเจค):

```powershell
# entrypoints
New-Item -ItemType Directory -Force -Path cmd/api, cmd/worker, cmd/scheduler, cmd/splitter

# internal (โค้ดที่ห้าม import จากข้างนอก module)
New-Item -ItemType Directory -Force -Path `
  internal/config, internal/domain, internal/api/handler, internal/api/middleware, `
  internal/service, internal/batch, internal/scheduler, internal/queue, `
  internal/ratelimit, internal/sender/sms, internal/sender/firebase, internal/sender/email, `
  internal/webhook, internal/report, internal/repository, internal/worker

# reusable + supporting
New-Item -ItemType Directory -Force -Path pkg/logger, migrations, deployments, configs, scripts
```
> **คำสั่งนี้ทำอะไร:** `New-Item` = สร้างไฟล์/โฟลเดอร์ใน PowerShell
> - `-ItemType Directory` = สร้างเป็นโฟลเดอร์
> - `-Force` = ถ้ามีอยู่แล้วไม่ต้อง error (สร้างซ้ำได้ปลอดภัย)
> - เครื่องหมาย `` ` `` ท้ายบรรทัด = บอกว่าคำสั่งยังไม่จบ ต่อบรรทัดถัดไป (ขึ้นบรรทัดใหม่เพื่ออ่านง่าย)
>
> ตอนนี้แค่ **สร้างโฟลเดอร์เปล่า** ให้ครบตามโครงสร้าง ยังไม่มีไฟล์ `.go` ข้างใน (จะเขียนขั้นถัดไป)

โครงสร้างที่ได้ (สรุป):

```
service-notification/
├── cmd/               # entrypoints (main.go แต่ละตัว)
│   ├── api/           # HTTP server (publisher)
│   ├── worker/        # consumer ส่งจริง
│   ├── scheduler/     # poll งานตั้งเวลา
│   └── splitter/      # แตก batch → message
├── internal/          # โค้ดหลัก (import นอก module ไม่ได้ = ปลอดภัย)
│   ├── config/        # โหลด env/config
│   ├── domain/        # entity + interface (core, ไม่ผูก framework)
│   ├── api/           # handler, middleware, router
│   ├── service/       # business logic
│   ├── batch/         # batch lifecycle + counter
│   ├── scheduler/     # logic ตั้งเวลา
│   ├── queue/         # RabbitMQ
│   ├── ratelimit/     # token bucket ต่อ provider
│   ├── sender/        # interface + adapters (sms/firebase/email)
│   ├── webhook/       # รับ DLR callback
│   ├── report/        # สรุป + CSV + storage
│   ├── repository/    # data access (DB)
│   └── worker/        # consume + retry logic
├── pkg/               # reusable (logger ฯลฯ)
├── migrations/        # SQL migration files
├── deployments/       # Dockerfile (containerize app ตอน deploy)
├── configs/           # config ตัวอย่าง
├── scripts/           # script ช่วยงาน
├── go.mod
├── CLAUDE.md
└── install.md
```

**หลัก best practice ที่ยึด:**
- `internal/` = โค้ดที่ import จากนอก module ไม่ได้ → กันคนอื่นเอาไปใช้ผิด
- `domain/` ไม่ import อะไรจาก layer อื่น (dependency ชี้เข้าใน = Clean Architecture)
- interface อยู่ที่ผู้ใช้ (`domain`), implementation อยู่ที่ layer นอก → test ง่าย/สลับ provider ได้
- `cmd/` บางที่สุด แค่ประกอบ dependency แล้วเรียก internal

---

<a name="step-6"></a>
## Step 6 — ติดตั้ง Go Dependencies

> **`go get` ต่างจาก `go install` (Step 4) ยังไง:**
> - `go install` = ลง **เครื่องมือ CLI** ไว้เรียกใช้ในเทอร์มินอล (migrate, swag)
> - `go get` = เพิ่ม **library** ที่โค้ดเราจะ `import` ไปใช้ → บันทึกลง `go.mod`/`go.sum` ให้ทั้งทีมได้เวอร์ชันเดียวกัน
>
> รันคำสั่งเหล่านี้ที่ root ของโปรเจค (ที่มีไฟล์ `go.mod`)

```powershell
# HTTP framework — รับ HTTP request, จัดการ routing/middleware ที่ชั้น API
go get github.com/gin-gonic/gin

# RabbitMQ (official client) — เชื่อมต่อ queue: publish message เข้า / consume ออก
go get github.com/rabbitmq/amqp091-go

# Database — Azure SQL (SQL Server) driver: อ่าน/เขียนตาราง batches, notifications
go get github.com/microsoft/go-mssqldb
#   (ใช้ตัวนี้เพราะ DB อยู่บน Azure SQL — ไม่ต้องลง pgx ของ Postgres)

# Firebase / FCM — ส่ง push notification (รวม multicast 500 token/call)
go get firebase.google.com/go/v4

# Rate limiting — คุมความเร็วยิงต่อ provider (token bucket) กันโดนแบน
go get golang.org/x/time/rate

# Config — โหลดค่าจาก yaml/env มาเป็น struct (host DB, url rabbitmq, rate ฯลฯ)
go get github.com/spf13/viper
go get github.com/joho/godotenv           # โหลดค่าลับจากไฟล์ .env เข้า environment

# Logger (structured) — log แบบ JSON ใส่ batch_id/message_id ตามรอยได้
go get github.com/rs/zerolog

# Utils
go get github.com/google/uuid                    # สร้าง id ให้ batch/message (ใช้ทำ idempotency)
go get github.com/go-playground/validator/v10    # validate ข้อมูลที่รับเข้ามาทาง API

# Swagger — สร้างหน้า API docs + middleware ฝั่ง gin
go get github.com/swaggo/gin-swagger
go get github.com/swaggo/files
go get github.com/swaggo/swag@v1.16.4     # ต้องตรงกับเวอร์ชัน swag CLI (ดูหมายเหตุ Step 10)

# จัดระเบียบ go.mod/go.sum (ลบตัวไม่ใช้ / เติมตัวขาด)
go mod tidy
```

### แต่ละ library ทำงานส่วนไหน (สรุป)

| Library | ชั้นที่ใช้ | หน้าที่ในโปรเจค |
|---------|-----------|-----------------|
| **gin** | `internal/api` | เว็บเซิร์ฟเวอร์ — รับ request `POST /notifications/batch`, routing, middleware (auth/log) |
| **amqp091-go** | `internal/queue` | คุยกับ RabbitMQ — publisher ส่งงานเข้า queue, consumer ดึงงานออกมาส่ง |
| **go-mssqldb** | `internal/repository` | ต่อ Azure SQL — insert/update สถานะ notification, query สรุป report |
| **firebase/go** | `internal/sender/firebase` | ส่ง FCM จริง — รวม `SendEachForMulticast` ยิงทีละ 500 token |
| **x/time/rate** | `internal/ratelimit` | จำกัด req/วินาที ต่อ provider ก่อนยิงจริง กันโดน rate limit |
| **viper** | `internal/config` | อ่าน config.yaml/.env → แปลงเป็นค่าที่โค้ดใช้ (ปรับ rate/worker ได้โดยไม่แก้โค้ด) |
| **zerolog** | `pkg/logger` | log ทั้งระบบแบบ structured ตามรอยด้วย batch_id/message_id |
| **uuid** | `internal/domain`, batch | สร้าง id ให้ batch/message — ใช้ทำ idempotency กันส่งซ้ำ |
| **validator** | `internal/api/handler` | เช็คข้อมูลที่รับเข้ามา (เบอร์ว่าง? channel ถูกไหม?) ก่อนประมวลผล |
| **gin-swagger** | `internal/api/router` | เสิร์ฟหน้า Swagger UI ให้เปิดดู/ทดลอง API ได้ |

> **หมายเหตุ:** โปรเจคนี้ยืนยันใช้ **Azure SQL (SQL Server)** แล้ว → ลง `go-mssqldb` ตัวเดียว (ไม่ต้องลง pgx)

---

<a name="step-7"></a>
## Step 7 — ไฟล์ Config / .env

สร้าง `configs/config.example.yaml`:

```yaml
app:
  env: local
  http_port: 8080

database:
  driver: sqlserver                              # Azure SQL (SQL Server)
  host: <ชื่อเซิร์ฟเวอร์>.database.windows.net    # FQDN ของ Azure SQL (ดูใน Azure Portal)
  port: 1433                                     # SQL Server ใช้ 1433
  user: ${DB_USER}                               # ดึงจาก .env — ไม่ hardcode
  password: ${DB_PASSWORD}                       # ดึงจาก .env — ห้ามใส่ตรงๆ
  name: notification
  encrypt: true                                  # Azure บังคับเข้ารหัสการเชื่อมต่อ

rabbitmq:
  url: ${RABBITMQ_URL}       # ต่อ RabbitMQ กลาง (repo rabbit-MQ) — ค่าจริงอยู่ใน .env
  exchange: notification     # ยิงเข้า vhost "notification" ที่ตั้งไว้ฝั่ง server กลาง

sms:
  provider: mock            # mock | twilio | vonage | thaibulk ...
  rate_per_second: 100      # ปรับตาม provider จริง
  max_concurrent: 50
  batch_size: 1

fcm:
  credentials_file: ./secrets/firebase.json
  rate_per_second: 500
  multicast_size: 500

email:
  provider: smtp            # smtp | sendgrid | azure_communication
  from: no-reply@example.com

worker:
  count: 10
  prefetch: 100

report:
  dlr_timeout_minutes: 20   # รอ DLR นานสุดก่อนปิด batch แล้วสรุป
```

> **สำคัญ (เพราะต่อ Azure SQL ตรงจากเครื่อง dev):**
> - ต้องเปิด **Firewall rule** ให้ IP เครื่องคุณใน Azure Portal → *SQL server → Networking → Firewall rules → Add current client IP* ก่อน ไม่งั้นต่อไม่ติด
> - `encrypt: true` บังคับ — Azure ไม่ยอมให้ต่อแบบไม่เข้ารหัส

### สร้างไฟล์ `.env` (เก็บค่าลับจริง)

**วางที่ไหน:** ที่ **root ของโปรเจค** (ระดับเดียวกับ `go.mod`) — ชื่อไฟล์คือ `.env` เป๊ะๆ

```
service-notification/
├── .env              ◄── ตรงนี้ (root) — ห้าม commit
├── .gitignore
├── go.mod
├── configs/
│   └── config.example.yaml
└── ...
```
> ⚠️ วางที่ root เท่านั้น **อย่าวางใน `configs/`** เพราะ tool ส่วนใหญ่หา `.env` ที่ root เป็น default

**ใส่อะไรบ้าง** — ทุกค่าที่เป็นความลับ (รหัส/API key) ค่าไหนยังไม่มีเว้นว่างไว้ก่อนได้:

```bash
# ===== Database (Azure SQL) — อ่านจาก Azure Portal → Connection strings =====
DB_USER=notif_admin
DB_PASSWORD=<รหัสจริงของ Azure SQL>

# ===== RabbitMQ — ต่อ server กลาง (repo rabbit-MQ) ผ่าน vhost "notification" =====
# รูปแบบ: amqp://<user>:<pass>@<host>:5672/<vhost>
RABBITMQ_URL=amqp://notif_user:<รหัส NOTIF_PASS จาก repo rabbit-MQ>@localhost:5672/notification

# ===== SMS Provider (ยังไม่รู้เจ้า → เว้นไว้ก่อน) =====
SMS_API_KEY=
SMS_API_SECRET=

# ===== Email (ตอนทำ report แจ้งผู้อนุมัติค่อยใส่) =====
EMAIL_API_KEY=

# ===== Firebase (FCM) — ชี้ไปไฟล์ credential ไม่ใส่ key ตรงๆ =====
FCM_CREDENTIALS_FILE=./secrets/firebase.json
```

### สร้างไฟล์ `.env.example` (commit ขึ้น git ได้)

ก๊อป `.env` แต่ **ลบค่าจริงออกให้เหลือ key เปล่า** — ไว้ให้เพื่อนร่วมทีมรู้ว่าต้องตั้งค่าอะไรบ้าง

```bash
DB_USER=
DB_PASSWORD=
RABBITMQ_URL=amqp://notif_user:@localhost:5672/notification
SMS_API_KEY=
SMS_API_SECRET=
EMAIL_API_KEY=
FCM_CREDENTIALS_FILE=./secrets/firebase.json
```
> ทีมอื่น clone โปรเจคมา → `copy .env.example .env` แล้วเติมค่าจริงของตัวเอง
> (ค่า RabbitMQ ต้องตรงกับ user/vhost ที่ตั้งไว้ใน repo `rabbit-MQ`)

### `.env` กับ `config.yaml` ทำงานคู่กันยังไง

- **ค่าไม่ลับ** (host, port, rate) → อยู่ใน `config.yaml`
- **ค่าลับ** (user, password, key) → อยู่ใน `.env` แล้วให้ yaml อ้างด้วย `${DB_USER}`

ตอนรัน โปรแกรมอ่าน `config.yaml` เจอ `${DB_USER}` → ไปดึงค่าจริงจาก `.env` มาแทน
> viper ไม่อ่าน `.env` อัตโนมัติ — ตอนเขียนโค้ด config (ขั้น scaffold) ต้องเพิ่ม `go get github.com/joho/godotenv`
> แล้วเรียก `godotenv.Load()` ตอนเริ่มโปรแกรม เพื่อโหลด `.env` เข้า environment ก่อน viper อ่าน

### สร้าง `.gitignore` (กันไฟล์ลับหลุดขึ้น git):

```powershell
Set-Content -Path .gitignore -Encoding utf8 -Value @"
# secrets & env
.env
!.env.example
secrets/
*.local.yaml

# build
/bin/
/tmp/
*.exe

# go
vendor/
"@
```
> **คำสั่งนี้ทำอะไร:** `Set-Content` = เขียนไฟล์ใหม่ (ในที่นี้คือไฟล์ `.gitignore`)
> ส่วน `@"..."@` คือการเขียนข้อความหลายบรรทัดในครั้งเดียว
> ไฟล์ `.gitignore` = รายชื่อไฟล์/โฟลเดอร์ที่ **สั่ง Git ให้ไม่เก็บ** — กันไฟล์ลับ (รหัสผ่าน) กับไฟล์ build หลุดขึ้น GitHub

> secret จริง (รหัส DB, API key provider, firebase.json) เก็บใน `.env` / `secrets/` เท่านั้น ไม่เข้า git

---

<a name="step-8"></a>
## Step 8 — เชื่อมต่อ RabbitMQ server กลาง

> **โปรเจคนี้ไม่รัน RabbitMQ เอง** — broker อยู่ในอีก repo (`GhostHunter2442/rabbit-MQ`) เพื่อให้หลาย project ใช้ร่วมกันได้
> ที่นี่แค่ **ทำให้ broker พร้อม แล้วต่อเข้าไป** ผ่าน `RABBITMQ_URL`

### 1. เปิด RabbitMQ server กลาง (ทำครั้งเดียว)

clone + รัน repo `rabbit-MQ` (ดูรายละเอียดเต็มใน `install.md` ของ repo นั้น):

```powershell
git clone https://github.com/GhostHunter2442/rabbit-MQ.git
cd rabbit-MQ
copy .env.example .env          # แล้วเติม RABBITMQ_PASS + NOTIF_PASS
docker compose up -d
bash setup-vhosts.sh            # สร้าง vhost "notification" + user "notif_user"
```
> ถ้าในทีมมี server กลางรันอยู่แล้ว ข้ามขั้นนี้ไปเลย — แค่ขอ user/pass/host มาใส่ใน `.env`

### 2. ตั้ง `RABBITMQ_URL` ใน `.env` ของ service-notification

```bash
# ต้องตรงกับ user/vhost ที่ตั้งไว้ฝั่ง repo rabbit-MQ
RABBITMQ_URL=amqp://notif_user:<NOTIF_PASS>@localhost:5672/notification
```
> `notification` ท้าย URL = **vhost** ของ project นี้ (แยกจาก project อื่นบน broker เดียวกัน)

### 3. เช็คว่าต่อได้

- เปิด Management UI: http://localhost:15672 → login ด้วย admin (รหัสใน `.env` ของ repo rabbit-MQ)
- ดูว่ามี vhost `notification` และ user `notif_user` แล้ว
- Database: ใช้ **Azure SQL** ที่ตั้งค่าไว้ใน Step 7 (ต่อ cloud ตรง)

---

<a name="step-9"></a>
## Step 9 — Makefile / Taskfile

สร้าง `Makefile` (ถ้าเลือก make):

```makefile
.PHONY: api worker migrate-up migrate-down swagger lint test tidy

# connection string ของ Azure SQL — อ่านค่าจาก environment (ตั้งใน .env ก่อนรัน)
# DB = pawnshop — ตารางเรา (batches, notifications) อยู่ใน schema dbo
DB_URL = sqlserver://$(DB_USER):$(DB_PASSWORD)@pawnshop-dev.database.windows.net:1433?database=pawnshop

# หมายเหตุ: RabbitMQ อยู่ในอีก repo (rabbit-MQ) — เปิด/ปิด broker ที่ repo นั้น
# ไม่มี target up/down ในนี้ เพราะ project นี้ไม่รัน infra เอง

api:           ## รัน API server
	go run ./cmd/api

worker:        ## รัน worker
	go run ./cmd/worker

migrate-up:    ## รัน migration ขึ้น Azure SQL
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1

swagger:       ## generate Swagger docs จาก annotation
	swag init -g cmd/api/main.go -o ./docs

lint:
	golangci-lint run ./...

test:
	go test ./... -race -cover

tidy:
	go mod tidy
```

> **หมายเหตุ:** `$(DB_USER)` / `$(DB_PASSWORD)` ต้องถูก export เป็น environment variable ก่อนรัน `make migrate-up`
> (เช่น โหลดจาก `.env` ด้วยเครื่องมืออย่าง `dotenv` หรือ `set -a; . ./.env; set +a` บน bash)
> — ไม่ hardcode รหัสลงไฟล์ Makefile ที่ commit ขึ้น git

> บน Windows ถ้าใช้ **Task** แทน ให้สร้าง `Taskfile.yml` แทน syntax เดียวกันเชิงแนวคิด

สร้าง `.golangci.yml` (config linter — best practice) ไว้ที่ root:

```yaml
run:
  timeout: 5m
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - gosec
    - revive
    - ineffassign
    - unused
```

---

<a name="step-10"></a>
## Step 10 — ตั้งค่า Swagger (API docs)

ใช้ **swaggo** สร้าง Swagger UI จาก annotation ในโค้ด (ไม่ต้องเขียน YAML มือ)

**1. ใส่ annotation ระดับ API ที่ `cmd/api/main.go`:**

```go
// @title           Service Notification API
// @version         1.0
// @description     API สำหรับส่ง SMS / FCM / Email notification
// @host            localhost:8080
// @BasePath        /api/v1
func main() { ... }
```

**2. ใส่ annotation ที่ handler แต่ละตัว** (ตัวอย่าง):

```go
// CreateBatch godoc
// @Summary      สร้างงานส่ง notification (batch)
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateBatchRequest true "batch payload"
// @Success      202 {object} dto.BatchResponse
// @Failure      400 {object} dto.ErrorResponse
// @Router       /notifications/batch [post]
func (h *Handler) CreateBatch(c *gin.Context) { ... }
```

**3. generate docs** (สร้างโฟลเดอร์ `docs/` อัตโนมัติ):

```powershell
swag init -g cmd/api/main.go -o ./docs
# หรือ: make swagger
```

**4. mount Swagger UI ใน router** (`internal/api/router.go`):

```go
import (
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
    _ "github.com/GhostHunter2442/service-notification/docs"  // สำคัญ: import docs ที่ generate
)

r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

**5. เปิดดู:** http://localhost:8080/swagger/index.html

> **หมายเหตุ best practice:**
> - ต้องรัน `swag init` ใหม่ทุกครั้งที่แก้ annotation → ใส่ไว้ใน `make swagger` / CI
> - เพิ่ม `docs/` ที่ generate เข้า git ได้ (ทีมอื่นไม่ต้องรัน swag เอง) หรือจะ generate ตอน build ก็ได้
> - ปิด Swagger UI บน production ด้วย env flag (เปิดเฉพาะ non-prod เพื่อความปลอดภัย)
> - annotation ต้องอยู่บน **named function** (เช่น handler method) — swag อ่าน closure/inline ไม่ได้
>
> ⚠️ **ปัญหาที่เจอบ่อย — version ไม่ตรง:** ถ้า build แล้ว error `unknown field LeftDelim/RightDelim in ... swag.Spec`
> แปลว่า **swag CLI** (ที่ generate) เวอร์ชันใหม่กว่า **library `swaggo/swag`** ใน go.mod → แก้ด้วย:
> ```bash
> go get github.com/swaggo/swag@v1.16.4   # ให้ตรงกับ swag --version
> ```

---

<a name="step-11"></a>
## Step 11 — ตรวจสอบว่าทุกอย่างพร้อม

รันเช็ค checklist:

```powershell
go version                                              # ✅ Go พร้อม
migrate -version                                        # ✅ migration tool
golangci-lint version                                   # ✅ linter
swag --version                                          # ✅ swagger generator
go build ./...                                          # ✅ build ผ่าน
```
> RabbitMQ: เช็คที่ repo กลาง (`rabbit-MQ`) ว่า container `healthy` + มี vhost `notification` แล้ว
> — ดู http://localhost:15672

ถ้าผ่านหมด = **พร้อมเริ่มเขียนโค้ดขั้นต่อไป**

---

## รัน Database Migration

สร้างตาราง `dbo.batches` / `dbo.notifications` ใน DB `pawnshop`
> migration เป็น **CREATE อย่างเดียว** เป็นตารางใหม่ ไม่แตะ/ทับตารางเดิมใน `dbo`

> ⚠️ **สำคัญ — อย่าสลับ syntax ข้าม shell:** คุณใช้ **Git Bash** (`MINGW64`) → ใช้ `${VAR}` และ `$DB_URL`
> **ห้าม** copy คำสั่งแบบ PowerShell (`$env:...` / `$($env:...)`) มารันใน Git Bash — bash จะแปลผิดจนได้ค่าว่าง แล้ว migrate จะ error `Windows logins are not supported`

| Shell | ตัวแปร | โหลด .env |
|-------|--------|-----------|
| **Git Bash** (แนะนำ) | `${DB_USER}` , `"$DB_URL"` | `set -a; source .env; set +a` |
| PowerShell | `$env:DB_USER` | (ดูสคริปต์ด้านล่าง) |

---

### วิธีรัน (Git Bash) — รัน 3 บรรทัดตามลำดับ

```bash
# 1) โหลดรหัสจาก .env เข้า environment
set -a; source .env; set +a

# 2) ตั้ง DB_URL (ใช้ ${...} แบบ bash)
export DB_URL="sqlserver://${DB_USER}:${DB_PASSWORD}@pawnshop-dev.database.windows.net:1433?database=pawnshop"

# 3) รัน migration ขึ้น
migrate -path migrations -database "$DB_URL" up
```

**เช็คก่อนว่าตัวแปรไม่ว่าง** (ถ้าไม่แน่ใจว่า .env โหลดติด):
```bash
echo "user=$DB_USER | url=${DB_URL:0:30}..."
# ควรเห็น user=azure_pawnshop_dev และ url=sqlserver://...  ถ้า user ว่าง = ยังไม่ได้ทำข้อ 1
```

### วิธีรัน (PowerShell) — ทางเลือก

```powershell
# 1) โหลด .env
Get-Content .env | ForEach-Object {
  if ($_ -match '^\s*([^#][^=]*)=(.*)$') { [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim()) }
}
# 2) รัน (ใช้ $env: แบบ PowerShell)
migrate -path migrations -database "sqlserver://$($env:DB_USER):$($env:DB_PASSWORD)@pawnshop-dev.database.windows.net:1433?database=pawnshop" up
```

> ทางลัด (ทั้ง 2 shell): `make migrate-up` — Makefile ตั้ง `DB_URL` ให้แล้ว แค่ทำข้อโหลด .env ก่อน

### ตรวจสอบหลังรัน

```bash
go run ./cmd/dbcheck        # จะเห็น "dbo.batches: มีอยู่แล้ว ✅"
```

### คำสั่ง migration อื่นๆ (Git Bash — ต้อง export DB_URL ก่อน)

```bash
migrate -path migrations -database "$DB_URL" version   # ดูเวอร์ชันปัจจุบัน
migrate -path migrations -database "$DB_URL" down 1     # ย้อน 1 step (ลบ 2 ตารางของเรา)
```
(`make migrate-down` = ย้อน 1 step)

> **หมายเหตุ:**
> - `$DB_URL` หายเมื่อปิด terminal — เปิดใหม่ต้อง `source .env` + `export DB_URL=...` อีกรอบ
> - รหัสผ่านถ้ามีอักขระพิเศษ (`@ : / ?`) ต้อง **URL-encode** ก่อนใส่ใน URL
> - ตาราง `dbo.batches` / `dbo.notifications` เป็นตารางใหม่ — ถ้าใน `dbo` มีชื่อชนอยู่แล้ว migrate จะ error ให้เห็น (ไม่ทับ)
> - migrate สร้าง `dbo.schema_migrations` เองไว้จำว่ารันถึงเวอร์ชันไหน

---

## สรุป checklist ที่ต้องติดตั้ง/สร้าง

**ติดตั้งบนเครื่อง (ครั้งเดียว):**
- [ ] Go 1.26.4
- [ ] Git
- [ ] Docker Desktop
- [ ] golang-migrate
- [ ] golangci-lint
- [ ] swag (Swagger)
- [ ] air (แนะนำ)
- [ ] make หรือ task (แนะนำ)

**สร้างในโปรเจค:**
- [ ] โครงสร้างโฟลเดอร์ (Step 5)
- [ ] Go dependencies (Step 6)
- [ ] configs/config.example.yaml + .env + .gitignore (Step 7)
- [ ] เชื่อมต่อ RabbitMQ กลาง + ตั้ง `RABBITMQ_URL` (Step 8)
- [ ] Makefile + .golangci.yml (Step 9)

**Infra ที่ต้องมี (repo แยก):**
- [ ] RabbitMQ server กลาง — repo `rabbit-MQ` (ดู install.md ของ repo นั้น)

---

## ขั้นตอนถัดไป (หลัง setup เสร็จ)

1. **Scaffold โค้ด**: domain models + Sender interface + mock sender → รัน end-to-end ได้
2. **Migration แรก**: สร้างตาราง `batches` + `notifications` (ดู schema ใน CLAUDE.md)
3. **ต่อ RabbitMQ**: publisher + consumer เบื้องต้น
4. Implement ทีละ component ตาม CLAUDE.md
