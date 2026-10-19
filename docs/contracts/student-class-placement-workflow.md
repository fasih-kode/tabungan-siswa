# Alur kelas dan penempatan siswa

Implementasi 16 Oktober 2026. Dokumen ini menggantikan bagian pembuatan kelas,
pembuatan siswa, penempatan awal, impor, dan daftar siswa pada kontrak FASE 18/19.

## Aturan yang diterapkan

- Kelas memiliki satu `academic_year_id` dan satu `effective_date`. Nama dan
  tingkat kelas unik dalam tahun ajaran yang sama; nama yang sama dapat dipakai
  lagi pada tahun ajaran lain. Tanggal efektif wajib berada dalam periode
  `[start_date, end_date)` tahun ajaran.
- Tahun ajaran kelas yang sudah dikonfigurasi tidak dapat dipindah. Admin dapat
  mengubah tanggal efektif untuk penempatan berikutnya. Riwayat siswa yang
  sudah tersimpan tidak ditulis ulang ketika konfigurasi kelas diubah.
- Modal Tambah Siswa meminta nama, NIS, NISN, tahun ajaran, dan kelas. Pilihan
  kelas hanya memuat kelas yang sudah dikonfigurasi untuk tahun ajaran tersebut.
- `StudentService.Create` menyimpan siswa ACTIVE dan riwayat kelas dalam satu
  transaksi. Periode riwayat dimulai dari tanggal efektif kelas sampai batas
  akhir tahun ajaran. Kegagalan salah satu penyimpanan membatalkan keduanya.
- Penempatan tambahan dari detail siswa menggunakan konfigurasi tanggal kelas
  yang sama. Overlap riwayat tetap ditolak. Perpindahan di tengah tahun tetap
  mengikuti alur transfer yang sudah ada.
- Akun login dibuat melalui menu Pengguna. Pembuatan siswa dan impor tidak
  membuat akun login maupun akun tabungan secara otomatis.
- Admin dapat mengaktifkan kembali siswa berstatus LEFT dengan memilih tahun
  ajaran, kelas, dan tanggal efektif. Sistem membuat riwayat kelas baru mulai
  tanggal tersebut dan mengubah status ke ACTIVE dalam satu transaksi. Riwayat
  lama tetap tertutup; tanggal yang bertumpang tindih dengan riwayat sebelumnya
  ditolak.

## HTTP dan daftar siswa

- `POST /classes`: `name`, `level`, `academic_year_id`, `effective_date` wajib.
- `POST /classes/{id}`: form menyertakan tahun ajaran yang sama dan tanggal
  efektif kelas. Perubahan nama/tingkat dan tanggal mengikuti validasi service.
- `POST /students`: `name`, `nis`, `nisn`, `academic_year_id`, `class_id` wajib.
  Tidak ada input tanggal atau akun di modal. Sukses diarahkan ke daftar siswa.
- `GET /students/class-options?academic_year_id=...`: JSON untuk pilihan kelas
  pada modal, khusus ADMIN. Respons memuat ID, nama, dan tanggal efektif kelas.
- Daftar ADMIN langsung menampilkan nama, NIS, NISN, tahun ajaran, kelas, dan
  status. Filter tahun ajaran dapat dipakai tanpa memilih kelas. Pencarian,
  filter status, filter belum ditempatkan, dan paginasi tetap tersedia.
- Satu baris mewakili satu siswa. Kelas yang ditampilkan adalah riwayat terakhir
  pada tahun ajaran yang dipilih, atau tahun ajaran/riwayat terakhir jika tanpa
  filter tahun. Filter kelas mengikuti penempatan terakhir tersebut.
- Filter belum ditempatkan memeriksa riwayat pada tahun yang dipilih; tanpa
  filter tahun, hanya siswa yang belum memiliki riwayat sama sekali.
- WALI_KELAS otomatis memakai tahun ajaran yang terbuka dan melihat siswa dari
  seluruh kelas yang ditugaskan pada tahun itu. Filter kelas mempersempit daftar
  ke satu kelas yang diampu; tahun lain dapat dipilih untuk melihat penugasan
  pada tahun tersebut. SISWA tetap diarahkan ke detail miliknya. Mutasi tetap
  memerlukan autentikasi dan CSRF; endpoint pilihan kelas tidak mengizinkan
  wali kelas atau siswa.

## Impor Excel

Lembar pertama memiliki lima header wajib:

| Header | Isi |
| --- | --- |
| `name` | Nama lengkap |
| `nis` | NIS sebagai teks agar nol awal tidak hilang |
| `nisn` | NISN sebagai teks agar nol awal tidak hilang |
| `academic_year` | Nama tahun ajaran persis seperti di aplikasi |
| `class` | Nama kelas yang terdaftar pada tahun ajaran tersebut |

Templat unduhan menggunakan format teks pada kelima kolom. Header tidak
membedakan huruf besar/kecil; nilai nama tahun/kelas dicocokkan setelah spasi
tepi dihapus. Nama kelas atau tahun ajaran yang ambigu ditolak dengan alasan
pada nomor baris terkait. Baris kosong dilewati tanpa mengubah nomor baris.

Batas tetap `.xlsx`, 5 MiB, 5.000 baris siswa. Service memeriksa seluruh baris
sebelum menulis: isian wajib, keunikan NIS/NISN dalam berkas dan database,
ketersediaan tahun ajaran, kecocokan kelas, serta batas tanggal efektif.
Kesalahan dilaporkan dengan nomor baris Excel asli. Semua siswa dan riwayat
kelas disimpan dalam satu transaksi; konflik identitas yang terjadi setelah
validasi juga membatalkan seluruh transaksi.

Service mengunci tahun ajaran yang dirujuk dalam urutan ID yang konsisten agar
konfigurasi periode/kelas tidak berubah selama validasi dan penyimpanan.
Database juga menolak referensi kelas yang sudah dikonfigurasi ke tahun ajaran
lain, serta perubahan periode tahun ajaran yang mengecualikan tanggal kelas.

## Migrasi 000007

Jalankan migrasi sebelum memakai binary aplikasi yang diperbarui.

Kelas lama sebelumnya terlihat di semua tahun ajaran. Migrasi mempertahankan
katalog itu dengan membuat satu salinan kelas per tahun ajaran yang sudah ada.
ID asli dipakai pada tahun pertama; tahun lain mendapat ID tersendiri. Referensi
riwayat siswa, penugasan wali kelas, serta kelas asal/tujuan transfer diarahkan
ke salinan yang sesuai. Periode riwayat yang sudah ada tidak diubah.

Tanggal konfigurasi kelas lama diisi dari awal tahun ajaran. Admin dapat
menyesuaikannya dari detail kelas untuk penempatan berikutnya. Tahun ajaran baru
yang dibuat setelah migrasi memerlukan pembuatan kelasnya sendiri.

Jika belum ada tahun ajaran saat migrasi dijalankan, kelas lama tetap tersimpan
sebagai kelas belum dikonfigurasi. Setelah membuat tahun ajaran, kelas tersebut
muncul di daftar ADMIN dan dapat dikonfigurasi dari detail. Kelas tanpa
konfigurasi tidak ditawarkan pada modal siswa atau impor.

Migrasi turun menggabungkan kembali kelas dengan nama/tingkat yang sama dan
mengarahkan seluruh referensi ke kelas global tersebut. Konfigurasi tahun dan
tanggal kelas dihapus; riwayat siswa, penugasan, serta transfer tetap disimpan.

## Pemeriksaan

Pengujian mencakup pembuatan dan penempatan dalam satu transaksi, rollback
ketika penyimpanan riwayat gagal (manual dan impor), validasi berkas menyeluruh,
nomor baris/duplikat, penempatan lintas tahun, batas tanggal, daftar dengan
penempatan terbaru, parser/templat Excel dengan identitas bernol awal,
otorisasi pilihan kelas, dan render modal/tabel.

Migrasi naik/turun/naik diuji pada skema sementara dengan kelas global yang
dipakai di dua tahun, termasuk riwayat siswa, wali kelas, dan transfer. Pengujian
integrasi hanya memakai database uji yang lolos penjagaan DSN proyek.
