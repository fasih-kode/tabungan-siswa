# Indeks Aturan Bisnis

Dokumen ini menunjuk aturan yang sudah tercatat; ia tidak mengesahkan aturan baru. Kontrak produk di `contracts/` adalah sumber aturan yang lebih rinci. Saat ada beda antara kode dan kontrak, catat keduanya dan minta keputusan, jangan diam-diam mengubah business rule.

## Tahun ajaran, kelas, dan siswa

- Tahun ajaran dan penempatan kelas bersifat year-scoped; riwayat penempatan menggunakan tanggal efektif. Rentang tahun ajaran pada kontrak transaksi start-inclusive/end-exclusive.
- Pembuatan tahun ajaran baru mensyaratkan tahun sebelumnya ditutup; service dan database memiliki enforcement terkait.
- Transfer kelas yang masih pending menghambat lifecycle tahun ajaran sesuai kontrak.
- Siswa dapat memiliki status lifecycle; perilaku promosi/kelulusan dan penempatan mengikuti [student-promotion-workflow.md](student-promotion-workflow.md) serta kontrak siswa/kelas.

Sumber utama: kontrak FASE 18 dan audit lifecycle di [`contracts/`](contracts/).

## Saldo, transaksi, dan audit

- Nilai uang direpresentasikan oleh tipe domain `Money`; ledger dan batas nominal mengikuti kontrak finansial.
- Saldo dihitung dari transaksi aktif menurut urutan kanonis di kontrak FASE 21. Perubahan yang membuat saldo kronologis negatif ditolak.
- Pembatalan transaksi adalah perubahan status historis, bukan penghapusan fisik. Koreksi dan aksi terkait dicatat pada audit trail.
- Mutasi finansial terikat ke akun, siswa, tahun ajaran, aktor, dan scope otorisasi yang diturunkan server-side.

Sumber: [FASE 21](contracts/fase-21-transaction-http-contract.md), [FASE 22](contracts/fase-22-transaction-correction-audit-contract.md), dan [audit kontrak finansial](contracts/fase-16-5-financial-contract-audit.md).

## Peran dan akses finansial

- Admin memiliki akses baca pengawasan keuangan lintas kelas/tahun, tetapi tidak melakukan mutasi keuangan langsung.
- Wali Kelas melakukan operasi keuangan dalam scope historis kelas/tahun yang ditugaskan.
- Akses siswa terbatas pada data finansial milik siswa yang terhubung dan bersifat baca.
- Reopen settlement hanya mengikuti alur pengecualian FASE 23 yang disetujui; detail dan batasannya ada di [financial-supervision-access.md](contracts/financial-supervision-access.md) dan kontrak FASE 23.

Jangan gunakan ringkasan ini untuk mengganti pengecekan service atau matriks akses kontrak.

## Settlement akhir tahun

FASE 23 menetapkan pratinjau dari ledger aktif penuh, settlement per akun, konfirmasi penyerahan terpisah untuk nominal positif, pemeriksaan readiness, dan workflow reopen/correction terkontrol. Rekening tahun baru mulai dari nol; saldo tidak dibawa otomatis. Rincian persis, termasuk pengecualian dan konkurensi, ada di [kontrak FASE 23](contracts/fase-23-year-end-settlement-contract.md).

**Status implementasi:** masih berjalan pada working tree saat dokumen ini dibuat; jangan menganggap seluruh aturan di atas sudah terpenuhi oleh kode.

## Aturan yang belum dikonfirmasi

Kebutuhan yang tidak ditemukan di kontrak atau implementasi yang dapat diverifikasi tidak ditambahkan di sini. Minta keputusan pemilik produk sebelum menerapkannya.
