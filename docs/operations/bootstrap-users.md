# Bootstrap akun awal

Perintah ini hanya untuk inisialisasi lokal pertama, bukan pengelolaan pengguna harian.

```bash
cd ~/Projects/tabungan-siswa
make bootstrap-users
```

Perintah akan:

- menolak database selain `tabungan_siswa_dev`;
- meminta username dan kata sandi admin serta wali kelas secara interaktif;
- tidak menampilkan kata sandi saat diketik dan meminta konfirmasi kata sandi;
- mensyaratkan kata sandi minimal 12 byte;
- menyimpan kata sandi sebagai hash Argon2id menggunakan implementasi aplikasi;
- menolak berjalan jika sudah ada pengguna berperan `ADMIN` atau username yang dipilih telah digunakan;
- membuat kedua akun dalam satu transaksi, sehingga kegagalan tidak meninggalkan hanya satu akun;
- meminta konfirmasi eksplisit nama database sebelum menulis data.

Default username adalah `admin` dan `wali_kelas`; keduanya dapat diganti saat prompt.

Perintah ini hanya membuat akun. Akun `WALI_KELAS` tetap harus diberi penugasan kelas dan tahun ajaran melalui fitur penugasan yang sesuai sebelum dapat mengakses data dalam lingkup kelas.
