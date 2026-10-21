# Ringkasan Kebutuhan Produk

## Status dan cara menggunakan dokumen

Dokumen ini adalah indeks ringkas kebutuhan dan konteks produk yang telah tercatat. Dokumen ini **bukan pengganti PRD yang disetujui atau kontrak fase**. Jika rincian kebutuhan tidak tertulis di sumber yang dirujuk, anggap belum diputuskan; jangan menyimpulkannya dari UI atau pola umum aplikasi.

Label yang digunakan:

- **Terkonfirmasi**: tercantum dalam roadmap atau keputusan kontrak yang disetujui.
- **Status implementasi**: kondisi kode yang harus diverifikasi pada commit dan working tree saat tugas dilakukan.
- **Belum selesai**: pekerjaan yang belum mempunyai bukti implementasi, verifikasi, atau penerimaan yang memadai.

## Tujuan produk terkonfirmasi

- Mengelola data akademik siswa dan kelas, tabungan siswa, transaksi, audit, serta proses penutupan tahun ajaran secara terkontrol.
- Menjaga integritas saldo dan sejarah finansial, serta menerapkan hak akses berdasarkan peran dan cakupan akademik.
- Mengembangkan fitur sebagai vertical slice dari use case sampai tampilan, mengikuti urutan fase roadmap.

Sumber: [roadmap resmi](../Roadmap_Resmi_Tabungan_Siswa.pdf), kontrak fase di [`contracts/`](contracts/).

## Aktor yang tercatat dalam kontrak

- **ADMIN**: mengelola area administrasi dan membaca pengawasan keuangan sesuai kontrak akses. Kebijakan finansial membatasi Admin dari mutasi keuangan.
- **WALI_KELAS**: mengelola operasi yang diizinkan dalam cakupan kelas dan tahun ajaran yang menjadi tanggung jawabnya.
- **SISWA**: akses baca dibatasi ke data milik siswa yang terhubung, sesuai kontrak fitur.

Rincian otorisasi harus mengacu pada [financial-supervision-access.md](contracts/financial-supervision-access.md) dan kontrak fitur terkait; ringkasan ini bukan matriks otorisasi lengkap.

## Area produk dalam roadmap

FASE 16–17 menetapkan baseline kontrak dan fondasi HTTP. FASE 18–25 membangun fitur akademik, siswa, tabungan, transaksi, koreksi/audit, settlement, administrasi, serta dashboard/pelaporan. FASE 26–33 mencakup enhancement UI, review visual, audit keamanan, integrasi, UAT, hardening, deployment, dan verifikasi rilis.

Roadmap menetapkan syarat produksi sebagai akumulasi integritas finansial, otorisasi terverifikasi, security audit, integration test, UAT, hardening, deployment, smoke test, dan dokumentasi. Kehadiran kode fitur saja tidak membuktikan kesiapan produksi.

## Keadaan yang diketahui saat dokumen dibuat

- FASE 21 memiliki catatan implementation lock di [fase-21-implementation-lock.md](contracts/fase-21-implementation-lock.md).
- Kontrak FASE 22 menyatakan implementasi terverifikasi, dengan owner review dan catatan lock masih tertunda; lihat [fase-22-transaction-correction-audit-contract.md](contracts/fase-22-transaction-correction-audit-contract.md).
- FASE 23 mempunyai kontrak dan implementation lock; lihat [kontrak FASE 23](contracts/fase-23-year-end-settlement-contract.md) dan [catatan implementation lock](contracts/fase-23-implementation-lock.md).
- FASE 24 sudah diimplementasikan, diterima owner, dan dikunci; lihat [kontrak FASE 24](contracts/fase-24-user-administration-contract.md) dan [implementation lock](contracts/fase-24-implementation-lock.md).

## Belum ditetapkan di ringkasan ini

- Kebutuhan produk yang belum tercantum dalam roadmap atau kontrak disetujui.
- Detail penerimaan UAT, deployment, backup/restore, dan dukungan operasional yang belum memiliki keputusan atau bukti tersendiri.
- Status fitur yang belum ditetapkan melalui verifikasi fase dan catatan penerimaan/lock terkait.

Jangan mengisi bagian ini dengan asumsi. Ajukan keputusan produk bila pekerjaan membutuhkannya.
