# Roadmap Maintainability Tabungan Siswa

## Metadata dan status

- Baseline yang ditinjau: commit `2475fe524c817a713657059d84454d9b4e9f0112` pada branch `review`.
- Dasar: [audit maintainability](MQ-AUDIT-00.md), [strategi pengujian](../TESTING.md), dan [handover](../HANDOVER.md).
- Status MQ-TEST-02-FIX02: PASS menurut supervisor. Tes terfokus lulus dan integration test pada PostgreSQL disposable melaporkan 11 package PASS serta 3 package tanpa tes. E2E dan race detector tidak dijalankan pada pekerjaan itu.
- Semua MQ-01 sampai MQ-04 di bawah **belum dimulai**. Dokumen ini adalah rencana kerja; bukan persetujuan implementasi MQ-03/MQ-04.

## Prinsip pelaksanaan

- Satu MQ memiliki scope dan commit tersendiri. Periksa branch, HEAD, dan status Git sebelum mulai; jangan mencampur perubahan di luar scope.
- Pertahankan kontrak bisnis, API, locking, urutan transaksi, otorisasi, dan audit yang berlaku.
- Package `internal/service` dan `internal/repository/postgres` memakai `TestMain` yang membutuhkan PostgreSQL. Tes package tersebut hanya boleh dijalankan melalui prosedur database disposable yang diverifikasi dan wrapper resmi `make test-integration`; lihat [TESTING.md](../TESTING.md).
- Catat bukti yang dijalankan. Kelulusan build atau vet sendiri tidak membuktikan kebenaran perilaku.
- Backlog di bawah dicatat terpisah dan **tidak otomatis menghalangi** MQ maintainability berisiko rendah, kecuali pemeriksaan awal suatu tugas menunjukkan risiko langsung pada scope-nya.

## Urutan pekerjaan maintainability

### MQ-01 — Konsistensi format kode tes

- **Tujuan:** Merapikan format file tes yang terdeteksi `gofmt`, tanpa mengubah assertion atau perilaku.
- **File sasaran:** [`internal/service/authentication_service_unit_test.go`](../../internal/service/authentication_service_unit_test.go).
- **Fakta awal:** `gofmt -l internal/service/authentication_service_unit_test.go` mencantumkan file tersebut. Tes autentikasi berada di package `internal/service` dan package ini memiliki `TestMain` yang membutuhkan database; jangan menjalankan package secara langsung tanpa database disposable.
- **Risiko:** Perubahan selain whitespace dapat tidak sengaja menyentuh assertion atau fixture. Risiko rendah bila diff hanya pemformatan.
- **Prasyarat:** Working tree bersih; review ulang diff file tunggal sebelum pengujian.
- **Pengujian:** `gofmt -d internal/service/authentication_service_unit_test.go` untuk meninjau hasil; pemeriksaan `git diff -w` untuk memastikan tidak ada perubahan non-whitespace; `go build ./...`, `go vet ./...`, serta tes relevan melalui `make test-integration` dengan target disposable terverifikasi.
- **Kriteria PASS:** Diff hanya perubahan format, `gofmt -l` tidak lagi mencantumkan target, tes relevan lulus, dan tidak ada file lain berubah.

### MQ-02 — Keterbacaan repository transaksi terlambat

- **Tujuan:** Memformat dan menata kode serta SQL agar query, parameter, pemetaan hasil, dan penanganan error mudah ditinjau.
- **File sasaran:** [`internal/repository/postgres/leaving_transaction_request_repository.go`](../../internal/repository/postgres/leaving_transaction_request_repository.go). Tes integrasi terkait yang ada berada di [`internal/service/service_transaction_integration_test.go`](../../internal/service/service_transaction_integration_test.go), termasuk `TestLeavingTransactionRequestApprovalExecutesExactlyOnce`.
- **Risiko:** Perubahan whitespace atau pemecahan ekspresi dapat tanpa sengaja mengubah SQL, urutan argumen, locking, atau alur error.
- **Prasyarat:** Baca kontrak [permintaan transaksi terlambat](../contracts/late-transaction-request-amendment.md), repository contract, dan semua pemanggil. Tidak ada perubahan query atau error semantics pada MQ ini.
- **Pengujian:** `gofmt -d` dan `git diff -w` pada target; tinjau kesetaraan teks query dan urutan argumen; `go build ./...`, `go vet ./...`, dan `make test-integration` terhadap PostgreSQL disposable. Tes harus menggunakan wrapper, bukan DSN development.
- **Kriteria PASS:** Diff tidak mengubah query, parameter, lock, transaksi, maupun error semantics; build dan vet lulus; integration test relevan lulus; hanya file dalam scope berubah.

### MQ-03 — Konsistensi error handling

- **Tujuan:** Menilai lalu, hanya bila disetujui supervisor, menyelaraskan konteks error dan pemetaannya pada satu repository.
- **File sasaran awal:** [`internal/repository/postgres/leaving_transaction_request_repository.go`](../../internal/repository/postgres/leaving_transaction_request_repository.go), termasuk tes repository yang baru ditambahkan bila perubahan semantik memang disetujui. Scope dibatasi pada repository ini.
- **Risiko:** Wrapping atau penerjemahan yang keliru dapat mengubah `errors.Is`, tipe error publik, respons HTTP, atau penanganan konflik/tidak ditemukan.
- **Prasyarat:** Periksa kontrak repository, penerjemah error yang sudah ada, pemanggil service, dan tes terkait. Serahkan temuan serta usulan diff kepada supervisor; **jangan mulai implementasi sebelum persetujuan supervisor**.
- **Pengujian:** Tes regression bila ada perubahan semantik; tes package yang relevan melalui lingkungan disposable/wrapper resmi; `go build ./...`, `go vet ./...`, dan `git diff --check`.
- **Kriteria PASS:** Persetujuan supervisor tercatat; makna error dan pemetaan publik yang disepakati terjaga; regression test relevan lulus; tidak ada abstraksi umum baru tanpa kebutuhan; perubahan hanya mencakup repository dan tes yang disetujui.

### MQ-04 — Evaluasi pemisahan service besar

- **Tujuan:** Mengevaluasi batas tanggung jawab service settlement dan menghasilkan rekomendasi yang dapat direview; panjang file saja bukan alasan pemisahan.
- **File sasaran evaluasi:** [`internal/service/settlement_service.go`](../../internal/service/settlement_service.go) dan tes terkait, terutama [`settlement_service_unit_test.go`](../../internal/service/settlement_service_unit_test.go) serta [`settlement_workflow_integration_test.go`](../../internal/service/settlement_workflow_integration_test.go).
- **Fakta awal:** File memuat alur settlement akhir tahun, settlement siswa pindah, reopen, preview, handover, readiness, dan audit. Komentar kode mendokumentasikan urutan lock settlement di sekitar fungsi `SettleYearEnd`.
- **Risiko:** Pemisahan yang salah dapat mengubah API, aturan keuangan, urutan lock, transaksi atomik, otorisasi, atau audit. Risiko tinggi karena service mengelola ledger dan penutupan tabungan.
- **Prasyarat:** Petakan tanggung jawab, dependency, call graph, batas Unit of Work, urutan lock, dan cakupan tes. Minta supervisor mereview rekomendasi serta rencana regression test.
- **Pengujian:** Pada tahap evaluasi, tidak perlu mengubah atau menjalankan kode. Jika pemisahan kelak disetujui sebagai MQ terpisah, wajib jalankan tes unit dan integration workflow pada database disposable; gunakan race detector bila scope menyentuh concurrency dan setelah resource aman tersedia.
- **Kriteria PASS evaluasi:** Dokumen review mengidentifikasi batas tanggung jawab berbasis perilaku, invariants yang harus dipertahankan, peta tes, risiko, dan usulan file/API. Tidak ada pemisahan kode dalam MQ evaluasi ini. Implementasi memerlukan persetujuan dan MQ terpisah.

## Backlog terpisah

Temuan berikut dicatat untuk triase tersendiri. Tidak ada yang diperbaiki dalam roadmap ini.

### Bug

- **BUG-01 — Relasi `LateRequestID` tidak dibaca kembali.** `Create` menulis `late_request_id`, tetapi query `GetByID`, list, dan `scanTransaction` tidak memilih atau mengisi field tersebut. Bukti: [`transaction_repository.go:30-104`](../../internal/repository/postgres/transaction_repository.go#L30) dan [`transaction_repository.go:107-179`](../../internal/repository/postgres/transaction_repository.go#L107), [`scanTransaction`](../../internal/repository/postgres/transaction_repository.go#L350). Dampak dan cakupan pemanggil perlu dinilai saat tiket bug dimulai.
- **BUG-02 — Constraint rupiah bulat belum ada pada permintaan transaksi terlambat.** Migration 000013 membatasi `amount > 0`, tetapi tidak memastikan nilai integer; migration 000002 memiliki constraint integer pada transaksi dan settlement. Bukti: [`000013_late_transaction_requests.up.sql:7-9`](../../migrations/000013_late_transaction_requests.up.sql#L7) dan [`000002_align_domain_contract.up.sql:35-42`](../../migrations/000002_align_domain_contract.up.sql#L35). Jangan mengubah skema tanpa tiket tersendiri, pemeriksaan status database, serta migration baru yang aman.

### Keamanan — perlu evaluasi, belum merupakan temuan terkonfirmasi

- **SEC-01 — Potensi perbedaan waktu respons autentikasi.** Pada [`authentication_service.go:42-61`](../../internal/service/authentication_service.go#L42), nama yang tidak ditemukan dan akun nonaktif kembali sebelum verifikasi hash kata sandi, sedangkan akun aktif menjalankan `PasswordHasher.Compare`. Ini menunjukkan jalur kerja yang berbeda, bukan bukti eksploitabilitas atau pengukuran timing. Evaluasi terkontrol diperlukan sebelum menentukan mitigasi.

### Pengujian dan lingkungan

- **TEST-BACKLOG — E2E:** Belum dijalankan dalam MQ-TEST-02/FIX02. Riwayat pengujian E2E sebelumnya di [HANDOVER](../HANDOVER.md) adalah bukti untuk sesi terdahulu, bukan verifikasi ulang baseline ini.
- **TEST-BACKLOG — Race detector:** Tidak dijalankan dalam MQ-TEST-02/FIX02. Catatan historis di handover perlu dibedakan dari verifikasi pada commit baseline terkini.
- **TEST-BACKLOG — Cleanup lingkungan sementara:** Cluster PostgreSQL FIX01 sudah dihentikan, tetapi direktori sementara masih tertinggal setelah automatic review menolak penghapusannya. Status terkini dan lokasi dicatat di [HANDOVER](../HANDOVER.md); pemilik perlu menangani penghapusan secara manual. Jangan mencoba mengakali penolakan keamanan.

## Batas roadmap

Dokumen ini tidak mengubah kode aplikasi, aturan bisnis, migration, atau database. MQ-01 dan MQ-02 merupakan kandidat maintenance terfokus; MQ-03 dan MQ-04 memerlukan review supervisor sebelum implementasi. Mulai pekerjaan berikutnya hanya setelah supervisor meninjau roadmap ini.
