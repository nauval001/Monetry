# Monetry API

REST API pencatatan keuangan pribadi — **Go + Fiber v2 + PostgreSQL** dengan **Clean Architecture**.

Dokumen kebutuhan: [`PRD.md`](./PRD.md).

## Prasyarat

- **Go 1.26+** (lihat direktif `go` pada `go.mod`)
- **PostgreSQL 14+**
- **Docker & Docker Compose** (opsional, untuk menjalankan seluruh stack)

> Alur kerja Git, pembagian peran A/B/C, dan panduan commit bertahap ada di
> [`CONTRIBUTING.md`](./CONTRIBUTING.md).

## Fitur

- **Autentikasi JWT**: access token 15 menit + refresh token 7 hari, **rotasi refresh token**, logout (pencabutan token), endpoint `me`.
- **RBAC**: `user` (default) hanya mengakses datanya sendiri, `admin` bisa melihat seluruh user.
- **Kategori**: CRUD dengan paginasi, filter tipe, dan proteksi relasi transaksi.
- **Transaksi**: pencatatan pemasukan/pengeluaran dengan `amount > 0`, ownership check, filter bulan/tahun/tipe/kategori, dan **cursor-based pagination**.
- **Budget bulanan**: upsert per kategori + kalkulasi **sisa budget** dan persentase pemakaian.
- **Dashboard**: total pemasukan, total pengeluaran, saldo bersih bulan berjalan.
- **Dokumentasi Swagger**: `/swagger/index.html`.
- **Docker**: cukup `docker compose up -d`.

## Struktur Folder (Clean Architecture)

```text
monetry-api/
├── app/
│   ├── model/           # Entitas & request/response DTO
│   ├── repository/      # Interface + implementasi query SQL (DBTX/WithTx aware)
│   └── service/         # Logika bisnis & handler Fiber (controller)
├── config/              # Env, logger (slog), factory Fiber app
├── database/            # Koneksi PostgreSQL, helper WithTx, migrator, seeder
├── docs/                # Hasil `swag init`
├── helper/              # JWT, response formatter, validator, pagination/cursor
├── middleware/          # Auth JWT, RequireRole (RBAC), error handler, request logger
├── migrations/          # File .sql (auto-migrate saat startup)
├── route/               # Registrasi endpoint
├── test/integration/    # Integration test endpoint
├── main.go
├── Dockerfile
└── docker-compose.yml
```

## Menjalankan dengan Docker

```bash
cp .env.example .env          # opsional, docker-compose punya default
docker compose up -d --build
docker compose logs -f api
```

- API: `http://localhost:8081`
- Swagger: `http://localhost:8081/swagger/index.html`
- Health check: `http://localhost:8081/health`
- PostgreSQL (dari host): `localhost:5433`

**Port mapping** (host → container) sengaja tidak memakai 8080/5432 agar tidak bentrok dengan service lain di mesin developer:

| Service | Host | Container | Alasan |
| --- | --- | --- | --- |
| api | `8081` | `8080` | 8080 sering dipakai proyek/container lain |
| postgres | `5433` | `5432` | 5432 dipakai PostgreSQL yang terpasang di host |

Jika 8080/5432 bebas di mesinmu, ubah `docker-compose.yml` menjadi `"8080:8080"` dan `"5432:5432"`, lalu `docker compose up -d`. Swagger UI otomatis mengikuti origin yang dipakai (tidak di-pin ke port tertentu).

Migrasi `migrations/*.sql` dijalankan otomatis saat startup, lalu akun admin dibuat dari env `SEED_ADMIN_*` (default `admin@monetry.local` / `Admin#12345`).

## Menjalankan tanpa Docker

```bash
cp .env.example .env    # sesuaikan DB_* dan JWT_SECRET
go run .
```

Butuh PostgreSQL yang aktif dan database `monetry` sudah dibuat.

## Konfigurasi Environment

| Variable | Default | Keterangan |
| --- | --- | --- |
| `APP_PORT` | `8080` | Port HTTP |
| `APP_ENV` | `development` | `production` → log JSON |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | — | Koneksi PostgreSQL |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` / `DB_CONN_MAX_LIFETIME` | `25` / `10` / `30m` | Connection pool |
| `JWT_SECRET` | — (wajib) | Secret HS256, minimal 16 karakter |
| `JWT_ACCESS_EXPIRY` | `15m` | Masa berlaku access token |
| `JWT_REFRESH_EXPIRY` | `168h` | Masa berlaku refresh token |
| `RATE_LIMIT_MAX` | `120` | Batas request/menit per IP |
| `SEED_ADMIN_USERNAME` / `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD` | `admin` / `admin@monetry.local` / kosong | Seeder admin (seeding dilewati jika password kosong) |

## Endpoint

Base URL: `/api/v1`

| Method | Endpoint | Akses |
| --- | --- | --- |
| `POST` | `/auth/register` | Publik |
| `POST` | `/auth/login` | Publik |
| `POST` | `/auth/refresh` | Publik |
| `POST` | `/auth/logout` | User |
| `GET` | `/auth/me` | User |
| `POST` | `/categories` | User |
| `GET` | `/categories` | User |
| `PUT` | `/categories/:id` | User |
| `DELETE` | `/categories/:id` | User |
| `POST` | `/transactions` | User |
| `GET` | `/transactions` | User |
| `PUT` | `/transactions/:id` | User |
| `DELETE` | `/transactions/:id` | User |
| `POST` | `/budgets` | User |
| `GET` | `/budgets` | User |
| `GET` | `/dashboard/summary` | User |
| `GET` | `/users` | Admin |

Format response standar:

```json
{
  "success": true,
  "message": "login berhasil",
  "data": { "...": "..." },
  "meta": { "page": 1, "limit": 10, "total": 25, "total_pages": 3, "has_next": true }
}
```

Contoh cepat:

```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@monetry.local","password":"Admin#12345"}'

curl -X POST http://localhost:8081/api/v1/transactions \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"category_id":1,"amount":75000,"description":"Makan siang","transaction_date":"2026-11-18"}'
```

## Testing

```bash
go test ./...                 # unit test (integration otomatis di-skip)
go test ./... -cover          # dengan coverage
```

Integration test `POST /api/v1/transactions` membutuhkan PostgreSQL:

```bash
# Windows PowerShell
$env:TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/monetry?sslmode=disable"
go test ./test/integration -run TestCreateTransactionIntegration -v

# Linux/macOS
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/monetry?sslmode=disable" \
  go test ./test/integration -run TestCreateTransactionIntegration -v
```

## Regenerate Swagger

```bash
go run github.com/swaggo/swag/cmd/swag@latest init -g main.go -o docs --parseDependency --parseInternal
```

## Catatan Desain

1. **`WithTx`** (`database/postgres.go`) membungkus `BeginTx → fn → Commit/Rollback`. Repository menerima `repository.DBTX` sehingga query yang sama bisa jalan di `*sql.DB` atau `*sql.Tx`. Dipakai pada create/update/delete transaksi dan rotasi refresh token.
2. **Lock kategori** memakai `SELECT ... FOR SHARE` saat mencatat transaksi, sehingga kategori tidak terhapus di tengah validasi.
3. **Cursor pagination** memakai pasangan `(transaction_date, id)` yang di-encode base64 → stabil meski ada data baru.
4. **Ownership check** selalu dilakukan di service layer menggunakan `user_id` dari JWT context (bukan dari body).
5. **Kategori dengan transaksi** tidak bisa dihapus (409) — sesuai `ON DELETE RESTRICT` pada skema.
6. **Refresh token** disimpan sebagai hash SHA-256 + `jti`, sehingga logout/rotasi bisa mencabut token spesifik.
