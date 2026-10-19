# Tabungan Siswa

Aplikasi pengelolaan tabungan siswa.

## Technology

- Go
- net/http
- html/template
- HTMX
- https://github.com/TailAdmin/tailadmin-free-tailwind-dashboard-template
- PostgreSQL
- SQL

## Alur siswa dan kelas

Kelas disimpan per tahun ajaran dengan tanggal efektif penempatan. Tambah siswa
melalui modal atau impor Excel langsung membuat riwayat kelas dalam transaksi
yang sama. Akun login dibuat melalui menu Pengguna.

Terapkan migrasi `000007_year_scoped_classes` sebelum menjalankan versi ini.
Lihat [kontrak alur dan migrasi](docs/contracts/student-class-placement-workflow.md)
untuk format impor, aturan tanggal, dan penanganan kelas lama.

Kenaikan kelas dan kelulusan massal dikelola Admin melalui menu **Kenaikan &
Kelulusan** setelah kelas tahun tujuan disiapkan. Terapkan migrasi
`000008_student_graduation` terlebih dahulu. Lihat
[alur kenaikan dan kelulusan](docs/student-promotion-workflow.md) untuk syarat,
pratinjau, dan penanganan pengecualian.
