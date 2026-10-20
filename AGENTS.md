# Instruksi Codex untuk Repository

Instruksi ini berlaku untuk pekerjaan rekayasa di repository ini. Instruksi sistem dan developer dengan prioritas lebih tinggi tetap berlaku. Kontrak produk yang disetujui tidak boleh diubah melalui instruksi kerja ini.

## Sumber konteks

Saat memulai tugas:

1. Periksa `git status` dan pertahankan perubahan lokal yang belum di-commit.
2. Baca `AGENTS.md` dan `docs/HANDOVER.md`, kemudian dokumentasi relevan dari indeks di `README.md`.
3. Periksa kode, tes, migrasi, dan perilaku aktual sebelum menyimpulkan atau mengusulkan perubahan.
4. Jika dokumentasi bertentangan dengan implementasi, verifikasi bukti dari kode dan kontrak. Laporkan perbedaannya; jangan menganggap salah satunya benar tanpa pemeriksaan.

## Aturan implementasi

1. Jangan mengubah aturan bisnis yang sudah disepakati tanpa persetujuan eksplisit.
2. Ikuti stack, arsitektur, pola, dan konvensi repository yang berlaku.
3. Utamakan perubahan kecil, terfokus, mudah dibaca, dan mudah diuji.
4. Terapkan clean code, pemisahan tanggung jawab, dan penanganan error yang jelas. Jangan menambah abstraksi atau dependency tanpa kebutuhan.
5. Jangan melakukan refactoring di luar ruang lingkup tugas tanpa alasan dan persetujuan.
6. Untuk perubahan perilaku, tambahkan atau perbarui pengujian yang relevan.
7. Jangan mengubah migration yang mungkin sudah diterapkan tanpa strategi migrasi dan rollback yang aman. Buat migration baru bila diperlukan dan jangan menjalankannya terhadap database nyata tanpa otorisasi.
8. Jangan menghapus data, mereset database, atau menjalankan tindakan destruktif tanpa persetujuan eksplisit dan penjelasan dampak.
9. Jangan menyimpan password, token, API key, DSN rahasia, atau rahasia lain dalam source, dokumentasi, log, Git, maupun output.
10. Jangan membuat commit, push, rebase, atau menjalankan operasi Git destruktif tanpa instruksi eksplisit.
11. Perbarui dokumentasi yang terdampak oleh perubahan penting. Jangan menggandakan kontrak resmi; tautkan ke sumber otoritatif.

## Protokol setiap sesi

### Mulai

- Periksa status Git sebelum mengedit.
- Baca `AGENTS.md`, `docs/HANDOVER.md`, dan dokumentasi relevan.
- Verifikasi struktur dan perilaku aktual yang disentuh.
- Untuk pekerjaan besar, jelaskan rencana, lingkup, dan dampaknya sebelum implementasi.

### Selesai

- Tinjau diff dan pastikan tidak menimpa perubahan pengguna.
- Jalankan tes dan pemeriksaan statis relevan yang aman untuk lingkungan ini.
- Untuk tes integrasi Go, gunakan `make test-integration` atau `GOFLAGS=-race make test-integration`; jangan bypass guard database.
- Laporkan hasil yang benar-benar dijalankan. Jangan menyatakan tes berhasil jika belum dijalankan.
- Catat risiko dan pekerjaan tersisa.
- Perbarui `docs/HANDOVER.md` berdasarkan bukti aktual sebelum mengakhiri pekerjaan yang mengubah keadaan repository.

## Batas sumber kebenaran

- Keputusan produk yang dikunci: kontrak terkait di `docs/contracts/`.
- Roadmap resmi dan urutan fase: `Roadmap_Resmi_Tabungan_Siswa.pdf`.
- Perilaku runtime: implementasi dan tes yang berlaku pada commit/working tree yang sedang ditinjau.
- Ringkasan arsitektur, aturan, pengujian, keputusan, dan handover: dokumen di `docs/`; semuanya menunjuk ke sumber otoritatif dan tidak menggantikannya.
- Ketika sumber berbeda, klasifikasikan klaim sebagai fakta implementasi, keputusan disetujui, rekomendasi, atau pekerjaan belum selesai.
