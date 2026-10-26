# MQ-04-REG-01 — Verifikasi Regresi Gabungan A/B/C

## Status

**PASS untuk cakupan regresi yang dijalankan.** Baseline source yang diuji: `23bcd427a4aa41536e01d9c3e6a5078e689b2b45` pada branch `review`. Working tree bersih sebelum dan sesudah pengujian source.

## Kesetaraan enam fungsi

Source pada HEAD yang diuji dibandingkan per blok fungsi dengan implementasi monolitik sebelum ekstraksi pada `cb7add40796c81a508770c85b595fbfacecff8c2`. Semua blok byte-identik, termasuk signature dan body. Setiap deklarasi hanya muncul sekali pada package; gabungan dua file tetap memiliki 21 deklarasi fungsi.

| Fungsi | Hasil |
|---|---|
| `(*settlementService).Get` | Identik; 1 deklarasi |
| `(*settlementService).List` | Identik; 1 deklarasi |
| `settlementPointers` | Identik; 1 deklarasi |
| `(*settlementService).PreviewStudentLeaving` | Identik; 1 deklarasi |
| `(*settlementService).PreviewYearEnd` | Identik; 1 deklarasi |
| `(*settlementService).YearReadiness` | Identik; 1 deklarasi |

Jumlah deklarasi fungsi pada `settlement_service.go` dan `settlement_service_read.go`: **21 sebelum/sesudah rangkaian ekstraksi**. Tidak ada SQL atau service interface baru akibat ekstraksi ini; sumber dan bukti per tahap tercatat di handover.

## Pemeriksaan non-database

| Perintah | Hasil |
|---|---|
| `gofmt -l internal/service/settlement_service.go internal/service/settlement_service_read.go` | PASS; tidak mencetak file |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test ./internal/domain ./internal/security ./internal/http/... -count=1` | PASS; 6 package |
| `git diff --check` | PASS |

## PostgreSQL dan integration test

`make test-integration` dijalankan dari snapshot `git archive` commit `23bcd427a4aa41536e01d9c3e6a5078e689b2b45`, bukan dari source commit lama. Snapshot menerima `.env` sementara milik pengujian; `.env` repository utama tidak dibaca atau diubah.

Lingkungan dibuat khusus untuk tugas ini:

- PostgreSQL 18, dijalankan sebagai pengguna non-root `fasih`.
- Listener hanya `127.0.0.1`, port sementara `46043` (bukan port sistem 5432).
- Direktori data privat: `/tmp/mq-04-reg-01.g8uscyki/pgdata`.
- Database `tabungan_siswa_fase168_clean_test`, role khusus `mq_reg01`; tidak ada kredensial dicatat dalam laporan.
- Identitas diverifikasi langsung: database, role, `data_directory`, alamat server, dan port aktual.
- Migration 000001–000013 diterapkan; `schema_migrations` menunjukkan versi 13, `dirty=false`.
- Tepat sebelum tes seluruh tabel aplikasi kosong. Tidak ada koneksi ke database development atau production.

Hasil `make test-integration`: **11 package PASS, 3 package tanpa test, 0 FAIL; exit code 0.** Package tanpa test: `cmd/bootstrap-users`, `cmd/e2e-fixture`, `cmd/web`. Package `internal/service` dan `internal/repository/postgres` lulus.

Suite `internal/service` menjalankan tes settlement termasuk `TestYearEndSettlementHandoverReopenCorrectionAndSuccessorGuard`, `TestSettlementServicePreviewsStudentLeavingWhileYearIsOpen`, `TestSettlementServiceStudentLeavingPreviewRejectsGraduatedStudent`, tes akses `Get`/`List`, dan tes cakupan `YearReadiness` untuk Wali/Admin serta penolakan Siswa. Package PostgreSQL juga lulus. Rincian semua tes mengikuti seluruh package Go yang dijalankan oleh `go test ./... -count=1` pada script resmi.

Sesudah tes, migration tetap versi 13 bersih. Tabel bisnis selain `users` dan `sessions` kosong; dua row `users` dan satu row `sessions` tertinggal dari suite. Karena database hanya berisi skema disposable baru dan tabel aplikasi kosong sebelum pengujian, seluruh cluster tugas dihentikan dan dibuang setelah tes.

## Cleanup

Cluster dihentikan dengan `pg_ctl stop`; PID terverifikasi tidak berjalan dan port 46043 tidak lagi menerima koneksi. Direktori task-owned `/tmp/mq-04-reg-01.g8uscyki` dihapus setelah memverifikasi path, pemilik, mode privat, data directory, PID, dan port. Pemeriksaan `pg_lsclusters` sesudahnya menunjukkan cluster sistem `18/main` tetap online pada port 5432 dan data directory `/var/lib/postgresql/18/main`.

Percobaan setup awal yang memakai port/socket default gagal sebelum migration atau tes; konfigurasi diperbaiki agar menunjuk port sementara. Percobaan pembuatan database dalam satu batch SQL juga ditolak PostgreSQL; perintah dijalankan ulang terpisah. Keduanya tidak menyentuh cluster sistem.

## Batas cakupan dan risiko tersisa

Verifikasi ini membuktikan build, pemeriksaan statis, tes package yang ditentukan, dan integrasi pada PostgreSQL disposable dari baseline tersebut. Ini bukan E2E browser, bukan race detector, serta bukan validasi produksi. Risiko race reaktivasi siswa dan settlement siswa pindah tetap belum ditangani. Tahap ekstraksi A/B/C telah melewati regresi gabungan; sebelum refactoring fungsi mutasi tetap diperlukan review supervisor tersendiri.
