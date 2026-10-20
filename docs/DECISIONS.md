# Indeks Keputusan

Register ini menunjuk keputusan yang telah tercatat di sumber proyek. Jika keputusan baru diperlukan, dokumentasikan alasan, status persetujuan, dan dampaknya sebelum mengubah implementasi.

## Keputusan teknis terkonfirmasi

| Keputusan | Status | Sumber |
|---|---|---|
| Go `net/http` dan `html/template`; tidak memakai `templ`. | Terkunci dalam roadmap | [Roadmap resmi](../Roadmap_Resmi_Tabungan_Siswa.pdf) |
| HTMX adalah progressive enhancement; HTML server-rendered tetap menyediakan perilaku dasar. | Terkunci dalam roadmap | [Roadmap resmi](../Roadmap_Resmi_Tabungan_Siswa.pdf), [pola UI](ui/form-patterns.md) |
| PostgreSQL adalah database utama; repository mengisolasi query dari use case. | Terkunci/terimplementasi | [Arsitektur aktual](ARCHITECTURE.md), kontrak fase |
| Domain tidak bergantung pada HTTP, database, template, atau HTMX. | Terkunci dalam roadmap dan sesuai batas package saat ini | [Roadmap resmi](../Roadmap_Resmi_Tabungan_Siswa.pdf), [arsitektur](ARCHITECTURE.md) |
| Tailwind/TailAdmin menjadi presentation layer, bukan framework arsitektur. | Terkunci dalam roadmap | [Roadmap resmi](../Roadmap_Resmi_Tabungan_Siswa.pdf) |
| Tes integrasi memakai database disposable yang di-allowlist dan guard DSN. | Terkunci secara operasional dan terimplementasi | [TESTING.md](TESTING.md), `internal/platform/database/test_dsn_guard.go` |

## Keputusan produk terkunci

- Otorisasi finansial dan pengecualian reopen: [kebijakan akses keuangan](contracts/financial-supervision-access.md).
- Transaksi, koreksi, dan audit: [kontrak FASE 21](contracts/fase-21-transaction-http-contract.md) dan [kontrak FASE 22](contracts/fase-22-transaction-correction-audit-contract.md).
- Settlement akhir tahun, readiness, penyerahan, dan reopen: [kontrak FASE 23](contracts/fase-23-year-end-settlement-contract.md).
- Alur operasional siswa/kelas: kontrak FASE 18–20 dan dokumen workflow terkait dalam [`contracts/`](contracts/).

Keputusan-keputusan tersebut tidak ditulis ulang di sini. Kontrak bersangkutan menjadi sumber otoritatif untuk rincian dan tanggal amandemen.

## Rekomendasi yang belum disetujui

- Menambah workflow CI untuk tes aman, vet, build, dan tes frontend.
- Menyediakan indeks migrasi dan prosedur setup database yang konsisten untuk lingkungan baru.
- Meninjau apakah setup `TestMain` dapat membuat tes non-integrasi Go berjalan tanpa database, tanpa melemahkan guard keamanan.

Butir di atas hanya rekomendasi audit, bukan keputusan atau otorisasi implementasi.

## Review checkpoint FASE 23 — 2026-10-20

Status: **review teknis dan verifikasi acceptance lulus; siap untuk implementation lock**.

### Bukti yang dijalankan

- Tes unit/service mencakup penolakan Admin/siswa pada operasi yang dilarang, Wali di luar kelas historis, cakupan readiness Wali, atribusi tak terselesaikan untuk Admin, blocker transfer Wali/Admin, blocker rekonsiliasi, saldo nol, saldo positif, settlement berulang, stale preview, dan rollback saat audit gagal.
- Tes PostgreSQL menguji settlement bersamaan dengan penarikan, dua settlement bersamaan, settlement terhadap year close, handover positif termasuk settlement siswa keluar, alur reopen disetujui/ditolak, koreksi FASE 22, re-settlement, rekonsiliasi selisih, requester tidak dapat menyetujui permintaannya sendiri, larangan Admin mengoreksi keuangan, dan race request dengan pembuatan tahun penerus.
- Tes HTTP/template mencakup preview stale dengan CSRF dan saldo baru, overview Admin dan cakupan Wali, grup tanpa atribusi, respons error aman, detail/history pelaku, detail siswa read-only, serta keadaan overview kosong/siap/terblokir.
- Migration 10 dan 11 diuji turun-naik pada database disposable; status akhir database tercatat versi 11 bersih. Anomali metadata awal dicatat di [handover](HANDOVER.md).
- Perintah verifikasi akhir: `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration`, `GOCACHE=/tmp/tabungan-siswa-gocache GOFLAGS=-race make test-integration`, `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...`, `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...`, `npm run test:instant-search`, dan `git diff --check`; semuanya lulus. Tes frontend: 7 lulus.

### Keputusan review

Bukti teknis dan acceptance checks yang ditinjau telah lulus. Tidak ditemukan alasan teknis untuk menahan implementation lock berdasarkan tes yang tersedia. Owner memberi instruksi untuk menuntaskan penutupan; implementation commit dan lock record akan dicatat setelah seluruh pemeriksaan akhir lulus.

Keputusan produk pada §14 tidak berubah. Tidak ada browser E2E yang dijalankan.
