# MQ-AUDIT-00 — Audit Maintainability Tabungan Siswa

## Metadata audit

- Baseline commit: `4d31e2f9` (`docs(fase-24): lock verified implementation`)
- Branch saat audit: `main`; laporan dipublikasikan dari branch `review`.
- Sumber audit: working tree lokal pada baseline tersebut, yang belum bersih.
- Kondisi awal yang diamati: 85 file tracked berubah, 25 path untracked, tidak ada staged change.
- Pengujian: tidak dijalankan; audit ini read-only.
- Verifikasi GitHub: baseline tersedia di `origin/main`, tetapi seluruh perubahan source lokal belum dipublikasikan. Karena itu isi working tree bukan representasi lengkap source di remote.
- Tidak ada file aplikasi, database, migration, atau histori Git yang diubah oleh audit. File laporan ini ditambahkan sebagai pengecualian publikasi audit.

> Status fitur di bawah berarti implementasi dapat ditelusuri pada working tree yang diaudit. Status tersebut bukan bukti acceptance lulus atau kesiapan produksi.

## A. Identitas repository

- Root: `~/Projects/tabungan-siswa`.
- Branch awal: `main`; HEAD awal `4d31e2f9d88c84da5aa64084250afbe380284b51`.
- Working tree awal: 85 tracked changes dan 25 untracked paths. Perubahan tersebar di dokumentasi, kode Go, template, CSS/JavaScript, migration, dan tes. Tidak ada perubahan staged.
- Stack: Go 1.27.1, `net/http`, `html/template`, HTMX, Tailwind CSS/PostCSS, JavaScript, PostgreSQL lewat `database/sql` dan pgx. Dependensi Go utama lain meliputi `x/crypto` dan `excelize`.
- Struktur utama: `cmd/`, `internal/{app,domain,http,platform,repository,security,service}/`, `migrations/`, `docs/`, `scripts/`, `e2e/`, `assets/`, `static/`.
- Perintah yang tersedia: `make dev`, `make test-integration`, `make bootstrap-users`; npm menyediakan build CSS, tes JavaScript, dan Playwright E2E. Server tidak menjalankan migration saat startup; tidak ditemukan migration runner umum pada `Makefile`/`cmd/web`.

Bukti: [`go.mod`](../../go.mod), [`Makefile`](../../Makefile), [`package.json`](../../package.json), [`README.md`](../../README.md), dan [arsitektur](../ARCHITECTURE.md).

## B. Dokumen kontrak

Dokumen yang tersedia: README, `AGENTS.md`, PRD, arsitektur, aturan bisnis, strategi tes, keputusan, handover, roadmap resmi PDF, panduan operasi/UI, dan kontrak fase 16–25. Indeks dokumentasi ada di [`README.md`](../../README.md); batas sumber kebenaran ada di [`AGENTS.md`](../../AGENTS.md).

Kontrak yang relevan mencakup otorisasi finansial, settlement/penutupan tahun, administrasi pengguna, dashboard/laporan, koreksi transaksi, dan transaksi terlambat untuk siswa pindah. Contoh: [akses finansial](../contracts/financial-supervision-access.md), [FASE 23](../contracts/fase-23-year-end-settlement-contract.md), [FASE 24](../contracts/fase-24-user-administration-contract.md), [FASE 25](../contracts/fase-25-dashboard-reporting-contract.md), dan [amendment transaksi terlambat](../contracts/late-transaction-request-amendment.md).

PRD mencatat FASE 25 telah memiliki implementasi dan pemeriksaan otomatis, tetapi owner review/implementation lock belum dicatat ([PRD.md:41](../PRD.md#L41)). Itu status dokumentasi; audit ini tidak memverifikasi runtime atau acceptance. Handover memuat catatan historis berurutan. Catatan awalnya tidak boleh dianggap sebagai keadaan terbaru tanpa membaca bagian bertanggal yang lebih baru. Banyak perubahan lokal membuat klaim lama di dokumentasi maupun riwayat tes tidak membuktikan kondisi working tree saat ini.

## C. Peta arsitektur

Arah dependency yang terlihat: `cmd/web` → composition root `internal/app` → router/middleware/handler → service → kontrak repository → PostgreSQL. Domain tidak bergantung pada HTTP maupun persistence.

- `cmd/web`: membaca DSN environment, membuka/ping database, menyusun aplikasi, menjalankan HTTP server (`cmd/web/main.go:20-55`).
- `internal/app`: wiring repository, Unit of Work, security, service, template, middleware, router, dan static assets (`internal/app/application.go:24-96`).
- `internal/domain`: entitas, nilai, validasi, serta transisi domain.
- `internal/service`: use case, otorisasi, validasi aplikasi, Unit of Work, dan audit.
- `internal/repository`: kontrak persistence dan Unit of Work; `internal/repository/postgres`: query dan mapping PostgreSQL.
- `internal/http`: router, middleware, handler, view model, error/redirect, dan static assets. Handler umumnya memvalidasi transport, memanggil service, lalu merender template.
- `internal/security`: Argon2id, session token/cookie, dan CSRF.
- Template HTML di `internal/app/templates/` di-embed; CSS/JS berada di `assets/` dan `static/`.
- Migration SQL bernomor berada di `migrations/`; keberadaan file migration tidak membuktikan penerapannya pada database tertentu.

Bukti: [`docs/ARCHITECTURE.md:13-48`](../ARCHITECTURE.md#L13), [`application.go:24-96`](../../internal/app/application.go#L24), [`main.go:20-55`](../../cmd/web/main.go#L20), [`router.go`](../../internal/http/router.go).

## D. Matriks status fitur

Klasifikasi berdasarkan implementasi yang dapat ditelusuri pada working tree; bukan klaim tes lulus.

| Fitur | Status | Bukti | Catatan |
|---|---|---|---|
| Pengguna dan autentikasi | IMPLEMENTED | `internal/service/authentication_service.go`, `internal/http/authentication_handler.go`, `internal/http/user_page.go` | Tes tersedia, tidak dijalankan. |
| Otorisasi ADMIN/WALI_KELAS/SISWA | IMPLEMENTED | `internal/service/authorization.go`, `scope.go`, `internal/http/middleware/` | Pemeriksaan scope ada di service. |
| Manajemen siswa | IMPLEMENTED | `internal/service/student_service.go`, `internal/http/student_page.go`, migrations 000001/000008 | Termasuk lifecycle siswa dan riwayat kelas. |
| Tahun ajaran | IMPLEMENTED | `internal/service/academic_year_service.go`, `internal/http/academic_year_page.go` | Ada transisi lifecycle/penutupan. |
| Kelas | IMPLEMENTED | `internal/service/class_service.go`, `internal/http/class_page.go` | Kelas year-scoped melalui migration 000007. |
| Penugasan Wali Kelas | IMPLEMENTED | `internal/service/class_assignment_service.go`, `internal/http/teacher_assignment_page.go` | Scope historis dipakai jalur finansial. |
| Rekening tabungan | IMPLEMENTED | `internal/service/savings_account_service.go`, `internal/repository/postgres/savings_account_repository.go` | Skema membatasi satu rekening per siswa/tahun. |
| Setoran dan penarikan | IMPLEMENTED | `internal/service/transaction_service.go:61-229` | Memeriksa role, tahun/rekening, scope, status siswa, saldo, dan audit. |
| Perhitungan saldo | IMPLEMENTED | `internal/domain/balance.go:17-44`, `internal/service/savings_account_service.go` | Dihitung dari ledger transaksi aktif, bukan kolom saldo rekening. |
| Riwayat transaksi | IMPLEMENTED | `internal/service/transaction_service.go:232-335`, `internal/http/transaction_page.go` | Memiliki pagination dan saldo berjalan. |
| Koreksi/pembatalan transaksi | IMPLEMENTED | `internal/service/transaction_service.go:338-598` | Memerlukan alasan, validasi saldo, scope, dan audit. |
| Audit trail | IMPLEMENTED | `internal/domain/audit_log.go`, `internal/service/audit_service.go`, `internal/repository/postgres/audit_repository.go` | Kelengkapan setiap aksi perlu dicocokkan per acceptance; tidak diaudit lewat runtime. |
| Dashboard dan laporan | PARTIAL | `internal/service/reporting_service.go`, `internal/repository/postgres/reporting_repository.go`, `internal/http/reporting_page.go` | Kode ada pada working tree; status FASE 25 masih menunggu review/lock menurut PRD. |
| Settlement dan penutupan tahun | IMPLEMENTED | `internal/service/settlement_service.go`, `internal/http/settlement_page.go`, migration 000011 | Settlement, penyerahan, readiness, dan reopen dapat ditelusuri. |
| Permintaan transaksi terlewat siswa pindah | PARTIAL | `internal/service/leaving_transaction_service.go`, `internal/http/leaving_transaction_page.go`, migration 000013 | Kode ada pada working tree; skema dan acceptance belum diverifikasi dalam audit ini. |
| Impor siswa/kenaikan kelas | IMPLEMENTED | `internal/service/student_service.go`, `docs/student-promotion-workflow.md` | Implementasi serta alur terdokumentasi ditemukan. |

## E. Audit keuangan read-only

### Kontrak

Kontrak menetapkan rupiah bulat, Wali melakukan mutasi dalam scope, transaksi batal dikecualikan dari saldo, koreksi memakai alasan/audit, dan settlement memiliki otorisasi/lifecycle tersendiri. Rujukan: [FASE 21](../contracts/fase-21-transaction-http-contract.md), [FASE 22](../contracts/fase-22-transaction-correction-audit-contract.md), [akses finansial](../contracts/financial-supervision-access.md).

### Perilaku yang terlihat pada kode

- Relasi rekening menghubungkan siswa dan tahun; transaksi serta settlement mengacu ke rekening. Definisi utama di [`000001_init_schema.up.sql:130-198`](../../migrations/000001_init_schema.up.sql#L130).
- `domain.Money` memakai `int64`, membatasi nominal dan memeriksa overflow/saldo negatif; `CalculateBalance` mengakumulasi transaksi aktif ([`money.go:3-40`](../../internal/domain/money.go#L3), [`balance.go:17-44`](../../internal/domain/balance.go#L17)).
- Setor/tarik memakai Unit of Work, mengunci baris tahun/rekening/siswa sesuai urutan jalur yang tampak, memeriksa scope/status, menghitung ulang saldo, dan mencatat audit sebelum commit ([`transaction_service.go:61-229`](../../internal/service/transaction_service.go#L61), [`transaction_service.go:737-788`](../../internal/service/transaction_service.go#L737)).
- Koreksi mengunci rekening, membaca ulang transaksi setelah lock, menghitung ulang saldo dan mencatat audit ([`transaction_service.go:338-472`](../../internal/service/transaction_service.go#L338)).
- Settlement membandingkan saldo aktual dengan saldo preview sebelum menulis ([`settlement_service.go:17-159`](../../internal/service/settlement_service.go#L17)). Tes integrasi konkurensi tersedia, tetapi tidak dijalankan.
- Audit memakai repository dan Unit of Work di jalur yang menerapkannya. Ini menunjukkan operasi tersebut dirancang atomik; belum dibuktikan runtime pada audit ini.

### Temuan integritas potensial — P2

Migration 000013 mendefinisikan `leaving_transaction_requests.amount NUMERIC(15,2)` dengan pemeriksaan positif, tetapi tidak menambahkan constraint jumlah bulat. Migration 000002 memiliki constraint integer untuk transaksi dan settlement ([000013:7-8](../../migrations/000013_late_transaction_requests.up.sql#L7), [000002:35-42](../../migrations/000002_align_domain_contract.up.sql#L35)). Domain/service menerima nominal integer; jadi jalur aplikasi terlihat membatasi nilai, tetapi batas database untuk permintaan baru tidak setara. Nilai pecahan yang masuk lewat jalur SQL lain dapat gagal saat dipetakan kembali ke `domain.Money` atau bertentangan dengan kontrak.

**Kandidat perbaikan:** evaluasi constraint rupiah bulat melalui migration baru setelah keadaan database/data diverifikasi. Ini indikasi dari skema dan kode, bukan hasil percobaan database.

## F. Audit keamanan

Mekanisme yang dapat dibuktikan dari source:

- Argon2id digunakan untuk hashing kata sandi (`internal/security/argon2id.go:16-32`, `:92-119`).
- Token sesi acak disimpan sebagai hash SHA-256 (`internal/security/session_token.go:10-24`).
- Cookie sesi menggunakan prefix `__Host-`, `HttpOnly`, `Secure`, dan `SameSite=Lax` (`internal/security/session_cookie.go:9-63`).
- Middleware memeriksa token sesi, kedaluwarsa, status akun, dan kewajiban ganti kata sandi (`internal/http/middleware/authentication.go:36-95`).
- Middleware CSRF memeriksa token cookie terhadap token form/header dengan perbandingan constant-time pada metode mutasi (`internal/http/middleware/csrf.go:23-53`).
- Handler memakai `html/template`; otorisasi utama juga ditegakkan oleh service.

Batas audit: database tidak diakses; privilege akun DB, status skema, konfigurasi TLS/HSTS/proxy, backup/restore, dan kontrol jaringan **UNKNOWN**. Bukti repository yang diperiksa tidak cukup untuk menetapkan kesiapan keamanan deployment. Temuan keamanan sensitif tidak dicantumkan dalam dokumen publik ini dan harus disampaikan terpisah kepada pemilik.

## G. Temuan maintainability

| ID | Lokasi | Masalah konkret | Dampak | Risiko | Kandidat perbaikan kecil |
|---|---|---|---|---|---|
| M-01 | `internal/service/settlement_service.go` (1.112 baris; settlement tahun akhir, pindah, reopen, readiness, handover, audit) | Banyak alur settlement berada dalam satu file | Perubahan satu alur memerlukan navigasi/review lintas fungsi yang panjang | P2 | Jika ada tugas berikutnya, pecah file berdasarkan alur dalam package yang sama; jangan ubah interface atau lock order. |
| M-02 | `internal/service/transaction_service.go` (810 baris); `internal/service/student_service.go` (1.016 baris) | Sejumlah use case/lifecycle dan helper terkait terkumpul dalam file besar | Navigasi dan review lebih berat, walau batas service masih terlihat | P2 | Pecah hanya saat perubahan berikutnya memberi batas alami; jangan refactor API tanpa kebutuhan. |
| M-03 | `internal/repository/postgres/leaving_transaction_request_repository.go:16-124` | Beberapa fungsi SQL baru dipadatkan ke satu baris dan sejumlah error scan dikembalikan tanpa konteks operasi | Query/urutan parameter lebih sulit direview; sumber kegagalan persistence lebih sulit dilacak | P2 | Format fungsi/query konsisten dan tambahkan konteks error tanpa mengubah perilaku. |
| M-04 | Komentar pada `internal/security/argon2id.go` dan tes tertentu | Sebagian komentar berbahasa Inggris di tengah kode yang umumnya berbahasa Indonesia | Konsistensi dokumentasi berkurang, dampak perilaku nihil | BACKLOG | Seragamkan oportunistis ketika file terkait disentuh; jangan menghapus komentar yang menjelaskan alasan teknis. |

Ukuran file sendiri bukan bukti bug. M-01/M-02 adalah rekomendasi pemeliharaan bertahap, bukan alasan untuk refactoring di luar tugas.

## H. Baseline pengujian

- Ditemukan **114 file `_test.go`** di domain, service, repository PostgreSQL, HTTP, security, platform/database, dan app/template.
- Unit test domain/security tersedia. Service dan repository PostgreSQL memiliki tes integrasi; `TestMain` package tersebut memerlukan `DATABASE_URL`, koneksi DB, dan advisory lock (`internal/service/testmain_test.go:16-68`, `internal/repository/postgres/testmain_test.go:16-68`).
- Guard menerima hanya nama database tes yang di-allowlist dan menolak database development (`internal/platform/database/test_dsn_guard.go:10-42`).
- `scripts/test-integration.sh` memverifikasi nama database sebelum `go test ./...` (`scripts/test-integration.sh:22-44`).
- Playwright E2E dan fixture tersedia di `e2e/settlement.spec.cjs`, `scripts/test-e2e.sh`, dan `cmd/e2e-fixture/`; E2E membuat dan membersihkan fixture pada database tes.
- Tes dengan cakupan bernama untuk saldo/transaksi/scope/settlement/audit ditemukan; keberadaan tes bukan bukti kelulusan.
- **Tidak ada tes, build, lint, aplikasi, migration, atau akses database yang dijalankan pada audit ini.** Status DB dan isolasi target runtime tidak dibuktikan.

Rujukan prosedur: [`docs/TESTING.md`](../TESTING.md).

## I. Prioritas awal

### P0

Belum ada temuan terverifikasi dari audit read-only ini yang cukup untuk menetapkan risiko kritis aktif.

### P1

Belum ada temuan terverifikasi yang memerlukan penghentian perubahan segera.

### P2

1. Tinjau constraint rupiah bulat pada migration 000013 dan rencanakan migration baru hanya setelah keadaan database/data diverifikasi.
2. Tingkatkan keterbacaan repository permintaan transaksi terlewat (M-03).
3. Pertimbangkan pemisahan file service settlement dan service besar lain saat perubahan berikutnya memiliki batas tanggung jawab yang jelas (M-01/M-02).
4. Lakukan review security deployment ketika konfigurasi/infrastruktur yang relevan tersedia.

### BACKLOG

- Seragamkan komentar bahasa Indonesia secara oportunistis.
- Dokumentasikan/verifikasi TLS, hak akses database, backup/restore, dan kontrol operasi.

Tidak ada rekomendasi yang dilaksanakan. MQ-01 tidak dimulai.

## J. Hal yang belum dapat dipastikan

- Migration sampai versi 13 telah diterapkan atau belum pada database tertentu.
- Semua perubahan lokal memenuhi acceptance dan apakah tes pada working tree lulus.
- Keadaan/performa query dan indeks pada data produksi.
- Kelengkapan audit log untuk seluruh tindakan dan privilege untuk mengubah log.
- Keberadaan TLS/HSTS, rate limiting, backup/restore, serta privilege least-privilege pada deployment.
- Apakah data permintaan transaksi terlambat mengandung nominal pecahan sebelum constraint baru dipertimbangkan.
- Penerimaan owner dan implementation lock FASE 25; PRD menyatakan status itu masih menunggu review/lock.
- Apakah source lokal yang belum dipublikasikan sama dengan source yang tersedia pada remote.

## K. Ringkasan eksekutif

Repository memiliki pemisahan modul Go yang cukup jelas antara domain, service, PostgreSQL, HTTP, security, dan composition root. Ledger transaksi adalah sumber saldo; jalur keuangan terlihat menggunakan Unit of Work, pemeriksaan scope, lock, validasi saldo, dan audit. Cakupan tes repository luas dan ada guard database serta browser E2E, tetapi tidak satu pun dijalankan dalam audit ini.

Risiko paling konkret yang ditemukan adalah tidak adanya constraint nominal integer pada tabel permintaan transaksi terlambat, dibanding tabel transaksi dan settlement. Maintainability dapat ditingkatkan pada service settlement yang memuat beberapa alur dan repository baru yang dipadatkan. Status working tree lokal yang belum bersih membuat hasil audit merepresentasikan baseline ditambah perubahan lokal, bukan hanya source pada GitHub.

Langkah berikut yang disarankan: supervisor menilai temuan P2 integritas nominal dan inventaris perubahan lokal; jangan membuat perubahan skema atau refactor sebelum review tersebut.
