# FASE 16.7 — Decision Resolution

Status: LOCKED
Tujuan: Menetapkan keputusan kontrak sebelum perubahan source code atau migration.

| ID | Keputusan yang Diusulkan | Dampak |
|---|---|---|
| DEC-16-01 | Periode tahun ajaran tidak boleh tumpang tindih. | Database |
| DEC-16-02 | Riwayat kelas harus berada dalam periode tahun ajaran terkait. | Domain, database |
| DEC-16-03 | Tahun ajaran hanya boleh CLOSED jika seluruh rekeningnya sudah SETTLED. | Lifecycle |
| DEC-16-04 | Siswa LEFT harus mengakhiri riwayat kelas aktif pada tanggal keluar. | Lifecycle |
| DEC-16-05 | Rekening baru tidak boleh dibuat untuk siswa LEFT. | Service, database |
| DEC-16-06 | Rekening baru tidak boleh dibuat pada tahun ajaran CLOSED. | Service, database |
| DEC-16-07 | Transfer request hanya boleh dibuat dan diproses saat tahun ajaran OPEN. | Service |
| DEC-16-08 | Maksimal satu transfer request PENDING per siswa per tahun ajaran. | Service, database |
| DEC-16-09 | Rekening tidak boleh dibuka kembali setelah tahun ajaran CLOSED. | Lifecycle |
| DEC-16-10 | Wali kelas mendapat akses sesuai penugasan dan riwayat kelas dalam tahun ajaran yang diminta. | Authorization |
| DEC-16-11 | Reopen settlement tetap hanya oleh ADMIN; workflow request/approval ditunda. | Authorization |
| DEC-16-12 | Settlement pengganti wajib menyimpan referensi settlement yang digantikannya. | Domain, database |
| DEC-16-13 | Transfer berstatus REJECTED wajib memiliki alasan penolakan; status lain tidak boleh memilikinya. | Domain, database |
| DEC-16-14 | Transaksi boleh dicatat mundur, tetapi tanggalnya harus berada dalam periode tahun ajaran terkait. | Service, database |

## Pemetaan Implementasi

| Keputusan | Tindak lanjut | Pemilik |
|---|---|---|
| DEC-16-01 | Cegah periode tahun ajaran tumpang tindih di PostgreSQL. | FASE 16.8 |
| DEC-16-02 | Validasi periode riwayat kelas dan tambah tes. | FASE 16.8 |
| DEC-16-03 | Validasi syarat penutupan tahun ajaran. | FASE 16.8 |
| DEC-16-04 | Tutup riwayat kelas saat siswa keluar. | FASE 19 |
| DEC-16-05–06 | Validasi pembuatan rekening berdasarkan status siswa dan tahun ajaran. | FASE 16.8 |
| DEC-16-07–08 | Validasi transfer dan cegah permintaan PENDING ganda. | FASE 16.8 |
| DEC-16-09 | Tolak reopen rekening setelah tahun ajaran CLOSED. | FASE 16.8 |
| DEC-16-10 | Verifikasi scope wali kelas; ubah hanya jika belum sesuai. | FASE 16.8 |
| DEC-16-11 | Tidak ada perubahan; workflow approval tetap ditunda. | DEFERRED |
| DEC-16-12 | Simpan hubungan settlement pengganti dengan settlement sebelumnya. | FASE 22 |
| DEC-16-13 | Validasi alasan penolakan transfer. | FASE 16.8 |
| DEC-16-14 | Validasi tanggal transaksi dalam periode tahun ajaran. | FASE 21 |

Implementasi hanya dilakukan setelah inspeksi kode dan migration yang relevan.
Jangan mengubah bagian yang sudah memenuhi kontrak.

## Aturan Pelaksanaan

1. Status seluruh keputusan di atas masih PROPOSED.
2. Jangan mengubah source code atau migration sebelum keputusan dikonfirmasi.
3. Setelah keputusan dikunci, petakan dampaknya ke implementasi dan tes.
4. Perubahan implementasi dilakukan minimal, mengikuti workflow:
   INSPECT → IMPLEMENT → FORMAT → TEST → VERIFY → LOCK.

## Status Akhir

FASE 16.7 LOCKED. Ke-14 keputusan disetujui sebagai kontrak final.
