# Panduan Git & Commit Bertahap — Monetry API

Dokumen ini melengkapi **PRD §8** (Workflow & Conventional Commits) dan sudah disesuaikan
dengan kondisi nyata repo ini (struktur folder, isi file, dan dependency antar layer).

Target: setiap PR kecil, **tidak ada konflik file**, dan history git memperlihatkan
pembagian peran A/B/C secara jujur.

---

## 1. Peran, branch, dan reviewer

| PIC | Peran | Branch | Reviewer PR |
| --- | --- | --- | --- |
| **Orang A** | Git Master + Backend 3 | `feat/core-infra`<br>`feat/budget-dashboard`<br>`feat/integration-docs` | Orang B |
| **Orang B** | Backend 1 | `feat/auth` | Orang A |
| **Orang C** | Backend 2 | `feat/category-transaction`<br>`test/transaction-integration` | Orang A |

Aturan review: **A tidak boleh me-review/merge PR miliknya sendiri** → PR A direview B.
Jika B tidak tersedia, C boleh menggantikan.

---

## 2. Aturan wajib (semua anggota)

1. **Conventional Commits**: `feat(scope): ...`, `fix(scope): ...`, `test(scope): ...`,
   `docs: ...`, `chore: ...`, `refactor(scope): ...`.
2. **Satu commit = satu concern.** Jangan `git add -A`/`git add .`; tambahkan path eksplisit.
3. **Dilarang di-commit**: `.env`, `monetry-api(.exe)`, `.tmp_*`, file log, dump database.
   `.gitignore` sudah menutup semuanya — jangan dipaksa dengan `git add -f`.
4. Sebelum push: `git pull --rebase origin main`.
5. Hanya PIC yang boleh mengubah **file panas** (lihat §5).
6. Sebelum commit, jalankan:
   ```bash
   gofmt -l .          # harus kosong
   go build ./...
   go vet ./...
   go test ./...
   ```
7. Judul PR memakai Conventional Commits; isi PR memakai template §9.
8. Merge PR: **Create a merge commit** (bukan Squash) supaya micro-commit tiap anggota
   tetap terlihat di `main` saat dinilai.

---

## 3. Persiapan semua anggota

```bash
git clone <url-repo> monetry-api
cd monetry-api
git config user.name "Nama Lengkap"
git config user.email "email@kampus.ac.id"
go version          # butuh Go 1.26+
go mod download
docker --version    # opsional, untuk verifikasi compose
```

> Penting: setiap anggota **wajib** set `user.name`/`user.email` masing-masing.
> `git shortlog -sne` di akhir proyek menjadi bukti kontribusi tiap peran.

Setelah satu PR di-merge, **selalu** sinkronkan branch kerja:

```bash
git checkout main
git pull origin main
git checkout <branch-kamu>
git rebase main
```

---

## 4. Urutan commit (dependency-safe)

Urutan ini mengikuti dependency kode yang sebenarnya: `middleware.Auth` + `user_id` context
(Backend 1) dibutuhkan oleh fitur C dan A; `CategoryRepository` dibutuhkan oleh Transaksi (C)
sekaligus Budget (A); `TransactionRepository.SummaryByPeriod` dibutuhkan Dashboard (A).

| # | PIC | Branch | Isi | Butuh (prasyarat) |
| --- | --- | --- | --- | --- |
| 1 | A | `feat/core-infra` | config, database + WithTx, migration, helper, docker | — |
| 2 | B | `feat/auth` | user, refresh token, JWT, middleware auth, RBAC, seed admin | 1 |
| 3 | C | `feat/category-transaction` | kategori + transaksi (cursor pagination, WithTx) | 2 |
| 4 | A | `feat/budget-dashboard` | budget + dashboard + unit test sisa budget | 3 |
| 5 | A | `feat/integration-docs` | `main.go`, `route.go`, Swagger, README | 4 |
| 6 | C | `test/transaction-integration` | integration test `POST /transactions` | 5 |

Catatan build: commit 1–4 sudah lolos `go build ./...` untuk seluruh package, tetapi
**binary baru bisa dijalankan setelah langkah 5** karena `main.go` + `route/route.go`
sengaja digabung di sana (dua file itu "hot file" milik A).

---

## 5. Kepemilikan file (mencegah konflik)

| File / folder | Pemilik | Catatan |
| --- | --- | --- |
| `main.go`, `route/route.go` | **A** | semua route & wiring DI |
| `migrations/001_initial_schema.sql` | **A** | satu file berisi semua tabel |
| `docs/` (generated) | **A** | hanya A yang menjalankan `swag init` |
| `Dockerfile`, `docker-compose.yml`, `.env.example`, `README.md`, `CONTRIBUTING.md` | **A** | |
| `go.mod`, `go.sum` | siapa pun | rebase dulu; hindari edit manual |
| `config/*`, `database/postgres.go`, `database/migrate.go` | **A** | infra |
| `helper/{errors,response,validator,pagination}.go`, `middleware/middleware.go` | **A** | shared kernel, "dibekukan" |
| `app/repository/dbtx.go`, `app/service/common.go` | **A** | shared kernel |
| `app/model/user.go`, `refresh_token.go`, `helper/jwt.go`, `middleware/auth.go`, `database/seed.go`, `user_service.go`, `auth_service.go`, `user_repository.go`, `refresh_token_repository.go` | **B** | auth + tabel user + RBAC |
| `app/model/{category,transaction,date}.go`, `category_/transaction_repository.go`, `category_/transaction_service.go` | **C** | |
| `app/model/{budget,dashboard}.go`, `budget_repository.go`, `budget_service.go`, `dashboard_service.go`, `budget_test.go` | **A** | |
| `test/integration/*` | **C** | setelah langkah 5 |

Anotasi Swagger **ditulis PIC masing-masing** di file handler-nya. Bagian Swagger milik A =
menjaga konvensi + menjalankan `swag init` + commit `docs/`.

---

## 6. Langkah commit per orang

> Semua perintah dijalankan dari root repo, satu perintah per baris (aman untuk cmd/PowerShell).
> Blok **Verifikasi & PR** dijalankan setelah seluruh commit pada fase itu selesai — yang
> diverifikasi adalah isi branch fase tersebut, bukan tiap micro-commit.

### 👤 A — Langkah 1: `feat/core-infra`

```bash
git checkout main
git pull origin main
git checkout -b feat/core-infra
```

**1.1 — `chore: add git and env scaffolding`**
```bash
git add .gitignore .env.example go.mod go.sum
git commit -m "chore: add gitignore, env example, and go module"
```

**1.2 — `feat(core): add env config, logger, fiber bootstrap, response formatter, and validator`**
```bash
git add config/env.go
git add config/logger.go
git add config/app.go
git add helper/errors.go
git add helper/response.go
git add helper/validator.go
git add middleware/middleware.go
git commit -m "feat(core): add env config, slog logger, fiber bootstrap, response formatter, and validator"
```

**1.3 — `feat(database): add postgres pool, with-tx helper, migration runner, and pagination helper`**
```bash
git add database/postgres.go
git add database/migrate.go
git add helper/pagination.go
git add helper/pagination_test.go
git commit -m "feat(database): add postgres pool, with-tx helper, and migration runner"
```

**1.4 — `feat(core): add shared dbtx abstraction and service helpers`**
```bash
git add app/repository/dbtx.go
git add app/service/common.go
git commit -m "feat(core): add shared dbtx abstraction and service helpers"
```

**1.5 — `feat(db): add initial schema migration`**
```bash
git add migrations/001_initial_schema.sql
git commit -m "feat(db): add initial schema for users, categories, transactions, budgets, and refresh tokens"
```

**1.6 — `chore(docker): add multi-stage dockerfile and compose stack`**
```bash
git add Dockerfile
git add docker-compose.yml
git commit -m "chore(docker): add multi-stage dockerfile and compose stack"
```

**Verifikasi & PR**
```bash
gofmt -l .
go build ./...
go vet ./...
go test ./...
git push -u origin feat/core-infra
```

- Judul PR: `feat(core): infrastructure, database transaction helper, and docker setup`
- Base: `main` · Reviewer: **B** · Merge: *Create a merge commit*
- DoD: build+vet hijau, `go test ./helper` PASS, `docker compose config --quiet` sukses,
  `docker compose up -d --build` menghasilkan container `api` **healthy** (opsional tapi disarankan).

---

### 👤 B — Langkah 2: `feat/auth`

```bash
git checkout main
git pull origin main
git checkout -b feat/auth
```

**2.1 — `feat(auth): add user and refresh token models`**
```bash
git add app/model/user.go
git add app/model/refresh_token.go
git commit -m "feat(auth): add user and refresh token models with request and response dto"
```

**2.2 — `feat(auth): add user and refresh token repositories`**
```bash
git add app/repository/user_repository.go
git add app/repository/refresh_token_repository.go
git commit -m "feat(auth): add user and refresh token repositories with find by identifier"
```

**2.3 — `feat(auth): add jwt manager, auth middleware, and auth endpoints`**
```bash
git add helper/jwt.go
git add middleware/auth.go
git add app/service/auth_service.go
git commit -m "feat(auth): add jwt manager, rbac middleware, and register login refresh logout endpoints"
```

**2.4 — `feat(user): add profile endpoint and admin user listing`**
```bash
git add app/service/user_service.go
git commit -m "feat(user): add me endpoint and admin paginated user listing"
```

**2.5 — `feat(seed): seed default admin from environment`**
```bash
git add database/seed.go
git commit -m "feat(seed): create default admin account from environment on startup"
```

**Verifikasi & PR**
```bash
gofmt -l .
go build ./...
go vet ./...
git push -u origin feat/auth
```

- Judul PR: `feat(auth): jwt authentication with refresh token rotation and rbac`
- Base: `main` · Reviewer: **A**
- DoD: build hijau; rotasi refresh token & `RequireRole` terimplementasi; **tidak** mengubah
  `route/route.go` atau `main.go` (wiring dilakukan A di langkah 5).

---

### 👤 C — Langkah 3: `feat/category-transaction`

```bash
git checkout main
git pull origin main
git checkout -b feat/category-transaction
```

**3.1 — `feat(category): add category model, repository, and crud endpoints`**
```bash
git add app/model/category.go
git add app/repository/category_repository.go
git add app/service/category_service.go
git commit -m "feat(category): add category model, repository, and crud endpoints with pagination"
```

**3.2 — `feat(transaction): add transaction model and repository with cursor pagination`**
```bash
git add app/model/date.go
git add app/model/transaction.go
git add app/repository/transaction_repository.go
git commit -m "feat(transaction): add transaction model and repository with cursor pagination"
```

**3.3 — `feat(transaction): add crud endpoints with ownership check and with-tx`**
```bash
git add app/service/transaction_service.go
git commit -m "feat(transaction): add crud endpoints with ownership check and database transaction"
```

**Verifikasi & PR**
```bash
gofmt -l .
go build ./...
go vet ./...
git push -u origin feat/category-transaction
```

- Judul PR: `feat(transaction): category crud and cursor paginated transactions`
- Base: `main` · Reviewer: **A**
- DoD: `amount > 0` divalidasi, filter `month/year/type/category_id` jalan, cursor
  `(transaction_date, id)` ter-encode di `meta.next_cursor`, ownership check di service.

---

### 👤 A — Langkah 4: `feat/budget-dashboard`

```bash
git checkout main
git pull origin main
git checkout -b feat/budget-dashboard
```

**4.1 — `feat(budget): add budget model and repository with upsert and period status`**
```bash
git add app/model/budget.go
git add app/repository/budget_repository.go
git commit -m "feat(budget): add budget model and repository with upsert and period status query"
```

**4.2 — `feat(budget): add set budget endpoint and remaining budget calculation`**
```bash
git add app/service/budget_service.go
git commit -m "feat(budget): add set budget endpoint and remaining budget calculation"
```

**4.3 — `test(budget): add unit test for remaining budget calculation`**
```bash
git add app/model/budget_test.go
git commit -m "test(budget): add unit test for remaining budget calculation"
```

**4.4 — `feat(dashboard): add monthly summary endpoint`**
```bash
git add app/model/dashboard.go
git add app/service/dashboard_service.go
git commit -m "feat(dashboard): add monthly summary for income, expense, and balance"
```

**Verifikasi & PR**
```bash
gofmt -l .
go build ./...
go vet ./...
go test ./app/model -run TestComputeBudgetUsage -v
git push -u origin feat/budget-dashboard
```

- Judul PR: `feat(budget): monthly budget status and dashboard summary`
- Base: `main` · Reviewer: **B** · DoD: unit test sisa budget PASS (minimal 4 skenario).

---

### 👤 A — Langkah 5: `feat/integration-docs` (wiring + Swagger + README)

> Ini titik pertama binary bisa dijalankan (`go run .` / `docker compose up`).

```bash
git checkout main
git pull origin main
git checkout -b feat/integration-docs
```

**5.1 — `feat(main): wire dependencies with graceful shutdown`**
```bash
git add main.go
git commit -m "feat(main): wire repository, service, and middleware with graceful shutdown"
```

**5.2 — `feat(route): register all v1 endpoints with auth and rbac`**
```bash
git add route/route.go
git commit -m "feat(route): register auth, category, transaction, budget, dashboard, and admin endpoints"
```

**5.3 — `docs: regenerate swagger for all endpoints`**
```bash
go run github.com/swaggo/swag/cmd/swag@latest init -g main.go -o docs --parseDependency --parseInternal
git add docs/docs.go
git add docs/swagger.json
git add docs/swagger.yaml
git commit -m "docs: regenerate swagger with annotations for all endpoints"
```

**5.4 — `docs: add readme with setup, endpoints, and design notes`**
```bash
git add README.md
git add CONTRIBUTING.md
git commit -m "docs: add readme with setup, endpoint list, and design notes"
```

**Verifikasi & PR**
```bash
gofmt -l .
go build ./...
go vet ./...
go test ./...
docker compose up -d --build
docker compose ps
git push -u origin feat/integration-docs
```

- Judul PR: `feat(core): wire all endpoints, swagger docs, and project readme`
- Base: `main` · Reviewer: **B**
- DoD: `docker compose up -d --build` → `monetry-api` & `monetry-postgres` **healthy**;
  `/health` 200; `/swagger/index.html` 200; login admin berhasil.
- Setelah merge, **semua anggota** `git pull origin main`.
- Opsional (rilis): `git tag -a v1.0.0 -m "Monetry API v1.0.0"` lalu `git push origin v1.0.0`.

---

### 👤 C — Langkah 6: `test/transaction-integration`

```bash
git checkout main
git pull origin main
git checkout -b test/transaction-integration
```

**6.1 — `test(transaction): add integration test for create transaction endpoint`**
```bash
git add test/integration/setup_test.go
git add test/integration/create_transaction_test.go
git commit -m "test(transaction): add integration test for create transaction endpoint"
git push -u origin test/transaction-integration
```

Cara menjalankan (butuh PostgreSQL aktif):

```bash
docker compose exec postgres createdb -U postgres monetry_test
```

```bash
# PowerShell
$env:TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/monetry_test?sslmode=disable"
go test ./test/integration -run TestCreateTransactionIntegration -v
```

```bash
# cmd
set TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5433/monetry_test?sslmode=disable
go test ./test/integration -run TestCreateTransactionIntegration -v
```

- Judul PR: `test(transaction): integration test for create transaction endpoint`
- Base: `main` · Reviewer: **A** · DoD: test **PASS** (bukan skip) terhadap DB nyata;
  skenario: create 201, amount 0 → 422, kategori milik user lain → 403, dashboard ikut terupdate.

---

## 7. Kalau terjadi konflik

| File | Cara resolve |
| --- | --- |
| `go.sum` | `git checkout origin/main -- go.sum` lalu `go mod tidy` |
| `docs/*` (generated) | `git checkout origin/main -- docs` lalu `swag init` ulang, baru commit |
| `route/route.go`, `main.go` | **hanya A** yang resolve |
| `migrations/001_initial_schema.sql` | A resolve; jangan ubah urutan/tipe kolom tanpa diskusi |
| file fitur (`app/**`, `helper/**`, `middleware/*`) | PIC masing-masing |

Setelah resolve:

```bash
git add <file-yang-diselesaikan>
git rebase --continue
gofmt -l .
go build ./...
go vet ./...
```

Jangan pernah `git add -A` saat konflik — file generated bisa ikut ter-stage setengah jadi.

---

## 8. Review & aturan merge

1. Aktifkan **branch protection** untuk `main`: wajib PR + minimal 1 approval + larangan push langsung.
2. Minimal 1 approval dari PIC lain (A tidak boleh approve PR A; B tidak boleh approve PR B).
3. Syarat merge: build/vet/test hijau di lokal reviewer, tidak ada konflik, tidak ada file asing di diff.
4. Merge memakai **Create a merge commit** (bukan Squash/Rebase merge) → micro-commit tetap tampil.
5. Hapus branch setelah merge, lalu semua anggota `git pull origin main`.

---

## 9. Template PR (salin ke deskripsi PR)

````markdown
## Ringkasan
<apa yang dikerjakan, 1–2 kalimat>

## PIC / Peran
PIC: Orang B — Backend 1

## Perubahan
- <poin 1>
- <poin 2>

## Cara menguji
```bash
go build ./... && go vet ./... && go test ./...
docker compose up -d --build
curl http://localhost:8081/health
```

## Checklist
- [ ] `gofmt -l .` kosong
- [ ] `go build ./...` hijau
- [ ] `go vet ./...` hijau
- [ ] `go test ./...` hijau
- [ ] tidak mengubah file milik PIC lain (`main.go`, `route/route.go`, `docs/`, dll)
- [ ] anotasi Swagger untuk endpoint baru sudah ditambahkan
- [ ] dokumentasi/README diperbarui bila perlu

Closes #<nomor-issue>
````

---

## 10. Kondisi repo saat ini (implementasi belum di-commit)

Saat ini git baru berisi skeleton (file 0 byte); seluruh implementasi ada di working tree
dan **belum di-commit**. Ada dua cara memecahnya tanpa kehilangan alur PR.

### Opsi 1 — Teknik branch `staging` (disarankan, authorship tiap orang tetap benar)

1. **A** membuat snapshot sementara:
   ```bash
   git checkout -b staging
   git add .gitignore .env.example go.mod go.sum
   git add config database helper middleware migrations app route test docs
   git add main.go README.md CONTRIBUTING.md PRD.md Dockerfile docker-compose.yml
   git commit -m "chore: staging snapshot for team split"
   git push -u origin staging
   ```
2. **A** mengambil file infra ke branch fase-1:
   ```bash
   git checkout main
   git checkout -b feat/core-infra
   git checkout origin/staging -- .gitignore .env.example go.mod go.sum config helper/errors.go helper/response.go helper/validator.go helper/pagination.go helper/pagination_test.go middleware/middleware.go database/postgres.go database/migrate.go migrations app/repository/dbtx.go app/service/common.go Dockerfile docker-compose.yml
   ```
   lalu lanjutkan commit **1.1–1.6** seperti §6 (file sudah ada di branch, tinggal `git add` per path).
3. **B** dan **C** mengambil file miliknya dari `staging` ke branch masing-masing:
   ```bash
   git fetch origin
   git checkout -b feat/auth origin/main
   git checkout origin/staging -- app/model/user.go app/model/refresh_token.go app/repository/user_repository.go app/repository/refresh_token_repository.go helper/jwt.go middleware/auth.go database/seed.go app/service/auth_service.go app/service/user_service.go
   ```
   lalu commit **2.1–2.5** dengan identitas B, push, dan buka PR.

   Untuk C:

   ```bash
   git fetch origin
   git checkout -b feat/category-transaction origin/main
   git checkout origin/staging -- app/model/category.go app/model/transaction.go app/model/date.go app/repository/category_repository.go app/repository/transaction_repository.go app/service/category_service.go app/service/transaction_service.go
   ```

   lalu commit **3.1–3.3** dengan identitas C, push, dan buka PR.
4. Setelah semua fase selesai dan PR di-merge:
   ```bash
   git push origin --delete staging
   ```

Teknik `git checkout origin/staging -- <path>` **mengambil isi file tanpa membawa commit
staging**, sehingga setiap orang tetap menjadi author commit-nya sendiri.

### Opsi 2 — Satu mesin, paling cepat

A melakukan seluruh commit berurutan sesuai §6 langsung dari working tree, lalu push ke `main`
via PR. Gunakan `git commit --author="Nama B <email@kampus>"` **hanya jika B benar-benar
mengerjakan bagian tersebut** — jangan mengklaim kontribusi yang tidak dikerjakan.
Cara ini paling cepat, tetapi alur review antar-anggota tidak terlihat di GitHub.

---

## 11. Checklist sebelum demo UAS

- [ ] `git shortlog -sne` menampilkan A, B, dan C dengan kontribusi wajar
- [ ] setiap PR direview orang lain (bukan author-nya)
- [ ] Phase 1–5 pada PRD §10 selesai
- [ ] unit test sisa budget **PASS**
- [ ] integration test `POST /api/v1/transactions` **PASS** (bukan SKIP)
- [ ] `docs/swagger.json` memuat 13 path + `securityDefinitions.BearerAuth`
- [ ] `docker compose up -d --build` → `monetry-api` & `monetry-postgres` healthy
- [ ] `.env` **tidak** pernah ter-commit: `git log --all -- .env` harus kosong
- [ ] tag rilis `v1.0.0` sudah dipush

---

## 12. Cheat sheet harian

```bash
git status
git pull --rebase origin main
git checkout -b <branch>
git add <path>
git commit -m "<type>(<scope>): <subject>"
git push -u origin <branch>
```

Setelah PR di-approve dan di-merge:

```bash
git checkout main
git pull origin main
git branch -d <branch>
```



