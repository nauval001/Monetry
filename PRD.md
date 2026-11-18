# PRODUCT REQUIREMENTS DOCUMENT (PRD)

- **Nama Proyek:** Monetry - Aplikasi Pencatatan Keuangan Pribadi
- **Mata Kuliah:** Praktikum Pemrograman Backend Lanjut
- **Arsitektur:** Clean Architecture (Go + Fiber + PostgreSQL)
- **Target AI Agent:** Cline (VS Code)

---

## 1. Ringkasan Eksekutif

**Monetry** adalah sebuah REST API backend yang dirancang untuk membantu pengguna mencatat, mengelola, dan menganalisis keuangan pribadi mereka. Pengguna dapat mencatat pemasukan (*income*) dan pengeluaran (*expense*), mengelompokkannya ke dalam kategori, serta menetapkan batas anggaran (*budget*) bulanan. Sistem ini dibangun dengan prinsip **Clean Architecture** yang memisahkan logika bisnis dari framework dan database.

---

## 2. Ruang Lingkup Sistem (Scope)

### In-Scope
- Autentikasi berbasis JWT (*Access Token* & *Refresh Token*).
- Manajemen Kategori (Khusus milik user yang login).
- Pencatatan Transaksi (Pemasukan/Pengeluaran) dengan fitur filter & paginasi.
- Penetapan Budget bulanan per kategori.
- Endpoint Dashboard (Ringkasan total kekayaan bersih/sisa budget).
- Dokumentasi API (Swagger).
- Deployment menggunakan Docker.

### Out-of-Scope (Tidak dikerjakan di UAS)
- Integrasi payment gateway (Midtrans/Xendit).
- Fitur sinkronisasi multi-device secara real-time (Websocket).

---

## 3. Peran Pengguna (Role Based Access Control)

Sistem menerapkan RBAC dengan dua peran utama:
1. **User (Default):** Dapat melakukan CRUD pada data kategori, transaksi, dan budget miliknya sendiri. Tidak dapat melihat data milik user lain.
2. **Admin (Superuser):** Memiliki akses untuk melihat daftar seluruh user di sistem. Dapat menghapus (moderasi) data kategori global jika diperlukan.

---

## 4. Kebutuhan Fungsional (Fitur)

### 4.1 Autentikasi (`/api/v1/auth`)
- **Register:** User mendaftar dengan `username`, `email`, dan `password` (di-hash dengan `bcrypt`).
- **Login:** Menghasilkan `access_token` (masa berlaku 15 menit) dan `refresh_token` (masa berlaku 7 hari).
- **Refresh:** Menukar `refresh_token` lama dengan pasangan token baru (Rotasi Token).
- **Logout:** Mencabut `refresh_token`.
- **Me:** Mendapatkan profil user yang sedang login beserta role-nya.

### 4.2 Manajemen Kategori (`/api/v1/categories`)
- **Create:** Membuat kategori baru (`type`: `income` atau `expense`).
- **List:** Melihat semua kategori milik user. Mendukung paginasi.
- **Update:** Mengubah nama atau tipe kategori.
- **Delete:** Menghapus kategori (menggunakan metode *soft delete* atau restriksi jika masih memiliki transaksi terkait).

### 4.3 Manajemen Transaksi (`/api/v1/transactions`)
- **Create:** Mencatat transaksi (`amount`, `description`, `transaction_date`, `category_id`).
  - *Validasi:* Nominal harus `> 0`. Kategori harus milik user tersebut.
- **List:** Melihat riwayat transaksi.
  - *Fitur:* Filter berdasarkan bulan/tahun (`month`, `year`), filter tipe (`income`/`expense`), dan *cursor-based pagination*.
- **Update & Delete:** Mengubah atau menghapus transaksi milik sendiri (wajib menerapkan pengecekan kepemilikan / *Ownership check* di layer Service).

### 4.4 Manajemen Budget (`/api/v1/budgets`)
- **Set Budget:** Menentukan batas pengeluaran untuk kategori tertentu di bulan dan tahun tertentu.
- **Get Budget Status:** Melihat sisa budget. Sistem menghitung:
  $$\text{Sisa Budget} = \text{Total Budget} - \text{Total Pengeluaran pada Kategori Tersebut}$$

### 4.5 Dashboard (`/api/v1/dashboard`)
- **Summary:** Mengembalikan data JSON berisi **Total Pemasukan**, **Total Pengeluaran**, dan **Selisih (Saldo Bersih)** untuk bulan berjalan.

---

## 5. Rancangan Database (PostgreSQL)

```sql
-- 1. Users
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Categories
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    type VARCHAR(10) NOT NULL CHECK (type IN ('income', 'expense')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, name)
);

-- 3. Transactions
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    amount DECIMAL(15,2) NOT NULL CHECK (amount > 0),
    description TEXT,
    transaction_date DATE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_transactions_user_date ON transactions (user_id, transaction_date DESC);

-- 4. Budgets
CREATE TABLE budgets (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    amount DECIMAL(15,2) NOT NULL CHECK (amount > 0),
    month INTEGER NOT NULL CHECK (month BETWEEN 1 AND 12),
    year INTEGER NOT NULL,
    UNIQUE(user_id, category_id, month, year)
);
```

---

## 6. Rancangan Endpoint (REST API)

Base URL: `/api/v1`

| Method | Endpoint | Akses / Role | Keterangan |
| :--- | :--- | :--- | :--- |
| `POST` | `/auth/register` | Publik | Registrasi akun user baru |
| `POST` | `/auth/login` | Publik | Login dan mendapatkan Access & Refresh Token |
| `POST` | `/auth/refresh` | Publik | Rotasi Access & Refresh Token |
| `POST` | `/auth/logout` | User | Logout / mencabut Refresh Token |
| `GET` | `/auth/me` | User | Mendapatkan profil user yang sedang login |
| `POST` | `/categories` | User | Membuat kategori pemasukan/pengeluaran |
| `GET` | `/categories` | User | Menampilkan daftar kategori milik user (paginasi) |
| `PUT` | `/categories/:id` | User | Mengubah kategori milik user |
| `DELETE` | `/categories/:id` | User | Menghapus kategori milik user |
| `POST` | `/transactions` | User | Mencatat transaksi baru (menggunakan `WithTx`) |
| `GET` | `/transactions` | User | Menampilkan transaksi (filter & cursor pagination) |
| `PUT` | `/transactions/:id` | User | Mengubah transaksi milik user |
| `DELETE` | `/transactions/:id` | User | Menghapus transaksi milik user |
| `POST` | `/budgets` | User | Menetapkan/memperbarui budget bulanan per kategori |
| `GET` | `/budgets` | User | Melihat status & sisa budget bulanan |
| `GET` | `/dashboard/summary` | User | Ringkasan pemasukan, pengeluaran, dan saldo bersih |
| `GET` | `/users` | Admin | Melihat seluruh daftar user di sistem |

---

## 7. Blueprint Clean Architecture (Struktur Folder)

```text
monetry-api/
├── app/
│   ├── model/           # Struct entitas & request/response
│   ├── repository/      # Interface & Implementasi Query SQL
│   └── service/         # Logika Bisnis & Menerima *fiber.Ctx (Controller)
├── config/              # Env & Logger
├── database/            # Koneksi PostgreSQL & Helper WithTx
├── helper/              # JWT, Response Formatter, Validator
├── middleware/          # Auth (JWT) & RequirePermission (RBAC)
├── migrations/          # File .sql
├── route/               # Registrasi Endpoint (Peta Akses)
├── main.go
├── .env.example
├── Dockerfile
└── docker-compose.yml
```

---

## 8. Alur Kerja Git (Micro-commits & Git Master)

### 8.1 Pembagian Peran
- **Git Master:** Setup boilerplate awal, konfigurasi Docker, review Pull Request, dan merging ke branch `main`.
- **Backend 1:** Fitur Autentikasi, Middleware, Tabel User.
- **Backend 2:** Fitur Kategori & Transaksi (CRUD + Paginasi).
- **Backend 3:** Fitur Budget, Dashboard Summary, dan Swagger.

### 8.2 Aturan Commit (Conventional Commits)
- `feat(auth): add login endpoint and jwt generation`
- `feat(transaction): implement cursor pagination on list endpoint`
- `fix(repository): resolve sql error on budget query`
- `docs: add swagger annotations for category endpoints`

### 8.3 Alur Kerja (Workflow)
1. Git Master melakukan inisialisasi repositori, push kerangka folder, dan file `.gitignore`.
2. Anggota melakukan `git clone`, lalu membuat branch fitur: `git checkout -b feat/auth`.
3. Anggota melakukan coding dan *micro-commits* di branch masing-masing.
4. Anggota melakukan `git push origin feat/auth`.
5. Anggota membuat Pull Request (PR) di GitHub.
6. Git Master me-review kode. Jika ada konflik, selesaikan dan lakukan Merge ke `main`.

---

## 9. Kebutuhan Non-Fungsional (Nilai Tambah)

1. **Concurrency & Transaction (Modul 9):** Saat membuat transaksi, wajib menggunakan database transaction (`WithTx`) untuk memastikan data transaksi dan pembaruan/validasi terkait berjalan secara atomik.
2. **Testing (Modul 12):**
   - Minimal **1 Unit Test** untuk fungsi perhitungan sisa budget.
   - Minimal **1 Integration Test** untuk endpoint `Create Transaction`.
3. **API Documentation (Modul 12):** Wajib menggunakan Swagger (`swag init`) dan menyertakan anotasi lengkap di setiap endpoint.
4. **Dockerization (Modul 12):** Menyediakan `Dockerfile` dan `docker-compose.yml`. Sistem harus dapat dijalankan hanya dengan perintah `docker compose up -d`.

---

## 10. Instruksi & Checklist Implementasi untuk Cline

> **Catatan untuk AI Agent (Cline):** Gunakan dokumen ini sebagai sumber kebenaran utama (*Single Source of Truth*). Patuhi struktur folder Clean Architecture pada Bagian 7, pisahkan query SQL di `app/repository/`, dan pastikan seluruh endpoint memvalidasi kepemilikan data (`user_id`) dari JWT context.

### Checklist Pengembangan:
- [ ] **Phase 1 - Core & Infrastructure:** Setup `config/`, `database/` (termasuk wrapper `WithTx`), `migrations/001_init_schema.sql`, `helper/` (JWT & standar JSON response), `Dockerfile`, dan `docker-compose.yml`.
- [ ] **Phase 2 - Auth & RBAC:** Implementasi `user_repository.go`, `auth_service.go` (Register, Login, Refresh rotation, Logout, Me, Admin GetAllUsers), serta `middleware/` (`AuthMiddleware` & `RequirePermission`).
- [ ] **Phase 3 - Categories & Transactions:** Implementasi CRUD Categories (dengan paginasi & proteksi relasi transaksi) dan CRUD Transactions (validasi `amount > 0`, ownership check, `WithTx`, filter bulan/tahun/tipe, dan *cursor-based pagination*).
- [ ] **Phase 4 - Budgets & Dashboard:** Implementasi `budget_repository.go`, `budget_service.go` (set budget & kalkulasi sisa budget), dan `dashboard_service.go` (summary pemasukan, pengeluaran, saldo bersih bulan berjalan).
- [ ] **Phase 5 - Testing & Docs:** Buat Unit Test perhitungan sisa budget, Integration Test `POST /api/v1/transactions`, tambahkan anotasi Swagger di semua handler, lalu jalankan `swag init` dan verifikasi dengan `docker compose up -d`.