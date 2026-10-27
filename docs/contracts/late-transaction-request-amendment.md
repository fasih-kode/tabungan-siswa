# Addendum: permintaan transaksi terlambat untuk siswa pindah

> **Prasyarat status di bawah telah superseded untuk proses pindah baru.** MQ-08-CONTRACT-02 mengizinkan penyelesaian permintaan terlambat selama proses pindah aktif dan sebelum siswa `LEFT`, dengan tanggal kejadian dari proses tersebut. Gunakan [amendment penyelesaian keuangan sebelum pengesahan pindah](student-transfer-financial-clearance-amendment.md); uraian di bawah tetap mendokumentasikan kontrak dan implementasi sebelumnya.

Status: disetujui owner 2026-10-21; implementasi tahap pertama.

## Kebijakan yang disetujui

- Wali dapat meminta pencatatan setoran/penarikan yang benar-benar terjadi sebelum tanggal efektif pindah siswa. Permintaan memuat rekening/tahun, jenis, nominal, tanggal kejadian, alasan, dan referensi bukti.
- Admin meninjau rincian dan menyetujui atau menolak dengan alasan. Persetujuan terikat pada rincian permintaan dan hanya dapat dieksekusi satu kali oleh Wali.
- Siswa tetap berstatus `LEFT`; transaksi biasa tetap ditolak. Penyerahan sisa saldo akibat pindah tetap melalui settlement dan rekonsiliasi.
- Tahap ini hanya menerima rekening belum selesai pada tahun `OPEN` atau `CLOSING`. Tahun `CLOSED` dan rekening settled di luar cakupan; jangan menggunakan reopen FASE 23 untuk membuat transaksi yang belum pernah dicatat.
- Permintaan `PENDING` atau `APPROVED` menghambat settlement rekening dan penutupan tahun. Permintaan `REJECTED` atau `COMPLETED` tidak menghambat.
- Tanggal efektif pindah disimpan saat Admin menandai siswa `LEFT`. Karena rentang kelas memakai akhir eksklusif, transaksi yang terlambat harus bertanggal sebelum tanggal pindah.

## Alur antarmuka

1. Wali membuka **Tabungan**, memilih buku siswa `LEFT`, mengisi transaksi yang tertinggal, alasan, dan referensi bukti.
2. Permintaan tampil di **Transaksi Terlewat**. Admin memeriksa dan memutuskan dengan alasan.
3. Setelah disetujui, Wali memakai tombol **Catat transaksi yang disetujui**. Backend mengaitkan transaksi dengan permintaan dan menolak pemakaian kedua.
4. Wali meninjau saldo terkini, menjalankan settlement pindah, lalu mencatat penyerahan uang. Admin memantau status pada **Penutupan Tahun**.

## Fakta implementasi

- Migration 13 menambahkan `students.leaving_date`, tabel permintaan, tautan satu-ke-satu dari transaksi yang dicatat ke permintaan, serta indeks unik referensi bukti aktif per rekening.
- Data siswa `LEFT` yang sudah ada sebelum migration akan memiliki `leaving_date = NULL`; tanggal tidak ditebak dari riwayat kelas. Permintaan untuk data tersebut ditolak sampai tanggal perpindahan terverifikasi tersedia.
- Perubahan status dan transaksi dijalankan dalam UOW dengan penguncian tahun/rekening/siswa/permintaan. Audit mencatat permintaan, keputusan, dan transaksi.

Sumber kebijakan umum keuangan dan settlement: [Aturan Bisnis](../BUSINESS_RULES.md), [kontrak FASE 23](fase-23-year-end-settlement-contract.md), dan [amendment settlement pindah](student-leaving-settlement-ui-amendment.md).
