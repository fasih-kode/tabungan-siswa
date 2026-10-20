# Kenaikan kelas dan kelulusan massal

Admin menyelesaikan settlement dan menutup tahun ajaran asal terlebih dahulu.
Tahun ajaran baru tidak boleh dibuat selama masih ada tahun berstatus `OPEN`
atau `CLOSING`; penutupan tahun asal mensyaratkan semua rekening tabungannya
`SETTLED`. Admin lalu membuat tahun tujuan berstatus `OPEN`, membuat kelas
tujuannya, dan membuka **Kenaikan & Kelulusan** untuk memeriksa pratinjau.
Tahun asal harus `CLOSED`; tahun tujuan harus dimulai pada atau setelah tanggal
akhir tahun asal. Kelas tujuan harus sudah dibuat pada tahun tujuan.

Setiap siswa aktif dengan riwayat di tahun asal mendapat pilihan kelas tingkat
berikutnya atau kelas tingkat yang sama untuk mengulang. Kelas tingkat tertinggi
dianggap calon lulus hanya jika tahun tujuan tidak memiliki kelas tingkat
lanjutan. Siswa dengan transfer yang masih menunggu, tanpa riwayat asal, status
`LEFT`/`GRADUATED`, atau tanpa kelas tujuan yang sesuai ditampilkan sebagai
pengecualian dan tidak diubah otomatis. Siswa yang ditinggal kelas dapat
ditetapkan ke kelas dengan tingkat yang sama.

Tombol proses mengirim keputusan untuk semua siswa yang memenuhi syarat di
pratinjau. Sistem memeriksa ulang pratinjau di dalam transaksi, mengunci siswa
dan tahun ajaran, menolak keputusan basi atau transfer menunggu, lalu membuat
riwayat pada tanggal efektif kelas tujuan (minimal tanggal mulai tahun tujuan).
Riwayat asal sudah dibatasi oleh akhir tahun ajaran berdasarkan kontrak riwayat
kelas; riwayat tersebut dipertahankan sebagai rekam jejak. Siswa lulus berubah
ke status `GRADUATED` tanpa riwayat kelas baru. Seluruh perubahan dibatalkan
jika satu keputusan gagal.

Status `GRADUATED` tidak dapat diaktifkan kembali melalui alur siswa pindah
sekolah. Settlement tabungan tetap dapat dilakukan untuk siswa `LEFT` atau
`GRADUATED`; tahun ajaran hanya dapat ditutup setelah rekening terbuka selesai
di-settle sesuai lifecycle tahun ajaran.

Sebelum menggunakan fitur, jalankan migrasi `000008_student_graduation` agar
constraint status siswa menerima `GRADUATED`.
Migrasi turun akan menolak rollback jika masih ada siswa `GRADUATED`, agar status
lulus tidak diam-diam diubah menjadi status pindah sekolah.
