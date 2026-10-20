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

Migrasi `000007_year_scoped_classes` adalah prasyarat alur kelas year-scoped.
Lihat [kontrak alur dan migrasi](docs/contracts/student-class-placement-workflow.md)
untuk format impor, aturan tanggal, dan penanganan kelas lama. Ini bukan daftar
lengkap versi skema terbaru; periksa migrasi dan kontrak operasional yang
relevan sebelum menyiapkan atau mengubah database.

Kenaikan kelas dan kelulusan massal dikelola Admin melalui menu **Kenaikan &
Kelulusan** setelah kelas tahun tujuan disiapkan. Migrasi
`000008_student_graduation` adalah prasyarat untuk alur ini. Lihat
[alur kenaikan dan kelulusan](docs/student-promotion-workflow.md) untuk syarat,
pratinjau, dan penanganan pengecualian.

## Dokumentasi proyek

- [Project requirements dan status produk](docs/PRD.md)
- [Arsitektur aktual](docs/ARCHITECTURE.md)
- [Indeks aturan bisnis](docs/BUSINESS_RULES.md)
- [Pengujian dan guard database](docs/TESTING.md)
- [Indeks keputusan](docs/DECISIONS.md)
- [Handover sesi dan status pekerjaan](docs/HANDOVER.md)
- [Instruksi kerja Codex](AGENTS.md)
- [Roadmap resmi](Roadmap_Resmi_Tabungan_Siswa.pdf)
