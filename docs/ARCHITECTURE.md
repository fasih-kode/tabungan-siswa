# Arsitektur Aktual

Dokumen ini merangkum struktur yang terlihat pada source. Periksa ulang source sebelum menggunakan ringkasan ini untuk perubahan desain.

## Stack

- Go (`go.mod` menyatakan Go 1.27.1), `net/http`, dan `html/template`.
- PostgreSQL melalui `database/sql` dan driver pgx.
- HTML server-rendered, HTMX untuk progressive enhancement, dan Tailwind CSS.
- `assets/css/app.css` menjadi input CSS; `static/css/app.css` adalah keluaran build. JavaScript aplikasi berada di `static/js/`.
- Template di-embed oleh `internal/app/application.go`; static assets dilayani oleh handler static HTTP.

## Batas modul

| Modul | Tanggung jawab yang terlihat |
|---|---|
| `cmd/web` | Membaca DSN, membuka database, menyusun aplikasi, menjalankan HTTP server. |
| `cmd/bootstrap-users` | Membuat akun awal melalui use case bootstrap. |
| `internal/domain` | Entitas, tipe nilai, validasi, dan transisi domain. |
| `internal/service` | Use case, otorisasi, validasi aplikasi, batas transaksi, dan audit. |
| `internal/repository` | Kontrak persistence dan Unit of Work. |
| `internal/repository/postgres` | Query PostgreSQL dan penyusunan repository berbasis `database/sql`. |
| `internal/platform/database` | Koneksi PostgreSQL dan guard konfigurasi database tes. |
| `internal/security` | Password, session token/cookie, CSRF, dan wiring komponen security. |
| `internal/http` | Router, middleware, handler, view model, error/redirect, dan static asset handler. |
| `internal/app` | Composition root, template parsing dan embedding. |

Alur umum: `cmd/web` → composition root `internal/app` → router dan handler → service → kontrak repository → implementasi PostgreSQL. Domain tidak mengambil dependency pada HTTP atau persistence.

## Persistence dan transaksi

- Skema dikelola lewat file SQL bernomor di `migrations/` dengan pasangan `up.sql` dan `down.sql`.
- Repository PostgreSQL menerima `*sql.DB` atau `*sql.Tx` melalui interface `DBTX`. `PostgresUnitOfWorkManager` membangun repository yang terikat ke satu transaksi SQL.
- Entry point server membuka dan melakukan ping database, tetapi tidak menjalankan migrasi saat startup. Prosedur migrasi harus mengikuti kontrak fase dan prosedur operasional yang relevan.
- File migrasi di working tree tidak membuktikan migrasi sudah diterapkan pada database tertentu.

## HTTP dan antarmuka

- Handler mengambil aktor terautentikasi dari context, memvalidasi input transport, memanggil service, lalu merender template atau melakukan redirect.
- Middleware authentication dan CSRF dipasang pada route; otorisasi juga ditegakkan di service.
- Template utama berada di `internal/app/templates/`. HTMX bukan lapisan business logic dan halaman dasar dirender di server.

## Konvensi keamanan yang teramati

- Password diproses melalui paket `internal/security`; session dan CSRF mempunyai implementasi serta tes tersendiri.
- Tes integrasi menolak database development dan hanya mengizinkan database bernama eksplisit untuk tes.
- Jangan menampilkan secret atau DSN. Jangan menjalankan tes integrasi pada database development.

## Batas pengetahuan dan pekerjaan tersisa

- Diagram deployment, topologi produksi, monitoring, backup/restore, dan pengelolaan migrasi di lingkungan produksi tidak ditetapkan oleh source composition ini.
- Verifikasi FASE 23 dan status fitur aktif dicatat di [HANDOVER.md](HANDOVER.md) serta kontraknya, bukan disimpulkan dari keberadaan file.
