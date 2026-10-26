# FASE 25: Dashboard & Reporting Contract

Status: **APPROVED** (owner selections, 2026-10-21)

## Tujuan dan batas

Menyediakan dashboard nyata dan laporan siswa, kelas, tahun ajaran, dan settlement. Semua data dalam satu tampilan harus mempunyai konteks tahun ajaran yang jelas dan tidak boleh menggabungkan tahun berbeda. Fase ini tidak mengubah aturan saldo, transaksi, settlement, maupun scope akses yang sudah dikunci.

## Konteks dan definisi metrik

- Dashboard menggunakan satu-satunya tahun ajaran yang belum `CLOSED` (`OPEN` atau `CLOSING`) sebagai tahun aktif. Status `CLOSING` tetap ditampilkan karena settlement masih berjalan. Jika belum ada tahun aktif, tampilkan keadaan kosong; jangan diam-diam memilih tahun historis yang sudah ditutup.
- Jumlah siswa dan kelas mengikuti penempatan siswa/kelas pada tahun aktif. Dashboard Wali hanya mencakup kelas dalam penugasan tahun tersebut. Dashboard Siswa hanya mencakup siswa yang terhubung ke akun.
- Penempatan/penugasan yang dipakai laporan adalah yang berlaku pada tanggal hari ini jika hari ini berada dalam rentang tahun ajaran; jika tahun aktif bertanggal di masa lalu/depan, gunakan tanggal terdekat di dalam rentangnya. Rentang tanggal tetap mengikuti semantik start-inclusive/end-exclusive.
- Saldo dihitung dari ledger transaksi aktif dengan aturan kanonis domain, pada rekening dalam tahun aktif dan scope aktor.
- Setor dan tarik adalah total nominal rupiah transaksi bertipe `DEPOSIT` dan `WITHDRAWAL` berstatus `ACTIVE` dalam rekening tahun aktif dan scope aktor. Transaksi `CANCELLED` tidak dihitung.
- Ringkasan settlement menghitung rekening tahun aktif menurut status rekening/settlement serta nominal yang telah diselesaikan/penyerahannya sesuai data settlement yang berlaku. Riwayat settlement `SUPERSEDED` tidak dihitung sebagai settlement terkini.
- Dashboard Siswa menampilkan saldo dan ringkasan setor/tarik milik sendiri serta transaksi terbaru dari rekening pada tahun aktif. Jika tidak ada rekening/tahun aktif, tampilkan keadaan kosong.

## Laporan

- Laporan Siswa: tabel siswa dan penempatan pada tahun aktif, status rekening, saldo, total setor, dan total tarik.
- Laporan Kelas: satu baris per kelas tahun aktif dengan jumlah siswa, saldo, total setor, dan total tarik.
- Laporan Tahun Ajaran: ringkasan terpisah untuk tahun aktif (siswa, kelas, saldo, setor, tarik, dan settlement). Laporan ini tidak merupakan agregasi lintas tahun.
- Laporan Settlement: tabel rekening/siswa pada tahun aktif dengan saldo rekening, status `Belum settlement` / `Menunggu penyerahan` / `Selesai`, nilai settlement terkini, dan nilai penyerahan bersih yang tercatat. Settlement nominal nol dianggap selesai; penyerahan menunggu bila nilai penyerahan bersih belum sama dengan nilai settlement.
- Semua laporan berupa tabel HTML di aplikasi. Ekspor file tidak termasuk FASE 25.
- Konteks tahun aktif ditampilkan di setiap halaman laporan. Saat tidak ada tahun aktif, halaman menjelaskan bahwa laporan memerlukan tahun aktif.

## Otorisasi

- `ADMIN`: membaca seluruh data tahun aktif.
- `WALI_KELAS`: hanya membaca siswa/rekening pada kelas yang ditugaskan pada tahun aktif.
- `SISWA`: dashboard dan laporan pribadi saja; tidak dapat membuka laporan direktori siswa, kelas, atau settlement.
- Service menegakkan scope; menyembunyikan tautan saja tidak cukup.
- Matriks dan aturan akses finansial rinci tetap merujuk [financial-supervision-access.md](financial-supervision-access.md).

## Batas arsitektur dan acceptance

- Agregasi berada dalam service/query khusus dengan query PostgreSQL terfokus. Handler hanya memvalidasi request, memanggil service, dan merender view model.
- Jangan membuat satu operasi `GetDashboardEverything`; dashboard role-specific memakai operasi yang sesuai.
- Query metrik mengikuti ledger kanonis dan scope server-side; query tidak mengandalkan filter browser untuk otorisasi.
- Tambahkan tes untuk definisi metrik, transaksi batal, tahun tanpa data, role/scope, dan pembatasan laporan Siswa.
- Tes integrasi PostgreSQL dan browser E2E mencakup dashboard dan laporan.

## Keputusan owner

- Periode: tahun ajaran aktif saja.
- Setor/tarik: total nominal transaksi aktif (transaksi batal dikecualikan).
- Format laporan: tabel web; tanpa ekspor CSV.

## Belum termasuk

- Pemilihan tahun historis atau agregasi lintas tahun.
- Ekspor CSV/Excel/PDF, grafik tren, dan laporan kustom.
- Definisi laporan operasional/keuangan lain di luar empat laporan dalam ruang lingkup.

## Klarifikasi UX yang disetujui owner — 2026-10-21

- Menu Admin dipisahkan menjadi **Penutupan Tahun**, **Histori Tabungan**, dan **Laporan**.
- Penutupan Tahun menonjolkan wali kelas/kelas dengan rekening belum settlement atau penyerahan belum direkonsiliasi; detail rekening tetap dapat dibuka dari daftar hambatan. Hak tindakan tetap mengikuti kontrak FASE 23.
- Histori Tabungan memberi Admin filter tahun ajaran dan kelas, serta opsi menampilkan hanya siswa yang memiliki rekening pada tahun terpilih. Kelas berasal dari penempatan terakhir siswa pada tahun ajaran tersebut; detail rekening tetap tunduk pada aturan baca-saja Admin dalam [kebijakan akses keuangan](financial-supervision-access.md).
- Menu Laporan membuka indeks empat laporan. Laporan tetap terbatas ke tahun aktif sebagaimana keputusan §2 dan §3.
- Perubahan ini memperjelas navigasi dan pencarian histori; tidak mengubah aturan settlement, keuangan, atau scope Wali/Siswa.
