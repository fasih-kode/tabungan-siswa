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

## Administrasi akun

- Hanya Admin yang dapat mengelola akun Wali Kelas dan Siswa.
- Akun Admin hanya diprovisikan melalui bootstrap; UI tidak membuat Admin.
- Role akun tidak dapat diubah melalui administrasi. Akun Admin dan akun pengguna sendiri tidak dapat dinonaktifkan.
- Penonaktifan dan reset password mencabut seluruh sesi. Reset membuat password sementara yang ditampilkan satu kali dan mewajibkan pergantian password saat login berikutnya.

Sumber: [kontrak FASE 24](contracts/fase-24-user-administration-contract.md).

## Settlement akhir tahun

FASE 23 menetapkan pratinjau dari ledger aktif penuh, settlement per akun, konfirmasi penyerahan terpisah untuk nominal positif, pemeriksaan readiness, dan workflow reopen/correction terkontrol. Rekening tahun baru mulai dari nol; saldo tidak dibawa otomatis. Rincian persis, termasuk pengecualian dan konkurensi, ada di [kontrak FASE 23](contracts/fase-23-year-end-settlement-contract.md).

**Status implementasi:** masih berjalan pada working tree saat dokumen ini dibuat; jangan menganggap seluruh aturan di atas sudah terpenuhi oleh kode.

## Aturan yang belum dikonfirmasi

Kebutuhan yang tidak ditemukan di kontrak atau implementasi yang dapat diverifikasi tidak ditambahkan di sini. Minta keputusan pemilik produk sebelum menerapkannya.

## Ringkasan dan laporan FASE 25

Keputusan FASE 25 yang disetujui owner: dashboard/laporan mengikuti satu tahun yang belum `CLOSED` (`OPEN` atau `CLOSING`); setor/tarik menjumlah nominal transaksi `ACTIVE`; laporan tersedia sebagai tabel web. Lihat [kontrak FASE 25](contracts/fase-25-dashboard-reporting-contract.md) untuk scope role, konteks penempatan, dan definisi report. Angka saldo tetap mengikuti ledger dan otorisasi finansial yang telah dikunci.

- Pindah sekolah dan penyelesaian tabungan adalah dua tahap. Admin memulai proses saat siswa belum `LEFT`; Wali yang berwenang menyelesaikan rekening relevan dan mengonfirmasi penyerahan bila saldo positif; backend memvalidasi kewajiban sebelum Admin mengesahkan pindah, mengubah status menjadi `LEFT`, dan menutup riwayat placement. Siswa tanpa rekening tidak dipaksa membuat rekening. Detail terbaru ada pada [amendment penyelesaian keuangan sebelum pengesahan pindah](contracts/student-transfer-financial-clearance-amendment.md), yang menggantikan urutan lama.
- Hanya Wali Kelas yang dapat mencatat setoran/penarikan. Transaksi baru mensyaratkan siswa `ACTIVE`; siswa `LEFT` atau `GRADUATED` tidak dapat menerima transaksi baru. Koreksi transaksi yang sudah ada tetap mengikuti scope, alasan, status rekening/tahun, dan audit FASE 22.
- Rekening siswa `GRADUATED` yang masih terbuka diselesaikan pada proses tutup tahun ajaran reguler. Admin tetap tidak melakukan mutasi keuangan; Wali tetap menangani transaksi, settlement, dan penyerahan sesuai scope.
- Untuk uang yang benar-benar terjadi sebelum pindah tetapi terlewat dicatat, Wali mengajukan permintaan dengan alasan dan referensi bukti; Admin memutuskan dengan alasan; hanya Wali mencatat rincian yang disetujui, satu kali. Dalam alur pindah baru, permintaan harus terkait proses aktif, memakai tanggal kejadian efektif pindah, dan diselesaikan sebelum settlement rekening. Permintaan `PENDING`/`APPROVED` menghambat settlement/finalisasi. Kontrol dan kompatibilitas histori diatur oleh [amendment transaksi terlambat](contracts/late-transaction-request-amendment.md) bersama amendment pindah terbaru.

Sumber keputusan: [amendment permintaan transaksi terlewat](contracts/late-transaction-request-amendment.md).
