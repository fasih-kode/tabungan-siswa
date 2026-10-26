# Strategi dan Perintah Pengujian

Perintah di bawah berasal dari `Makefile`, `package.json`, script tes, dan kontrak fase. Keberadaan perintah bukan bukti bahwa ia sudah dijalankan atau lulus pada perubahan terbaru.

## Go

- Tes package domain yang tidak memerlukan PostgreSQL: `go test ./internal/domain -count=1`.
- Suite Go dengan tes PostgreSQL: `make test-integration`.
- Suite Go dengan race detector: `GOFLAGS=-race make test-integration`.
- Analisis statis: `go vet ./...`.
- Build seluruh package: `go build ./...`.

`internal/service` dan `internal/repository/postgres` menjalankan `TestMain` yang membutuhkan `DATABASE_URL`, ping ke PostgreSQL, dan advisory lock. Database harus bernama `tabungan_siswa_fase168_test` atau `tabungan_siswa_fase168_clean_test`; database development ditolak. Jalur yang disarankan adalah wrapper `make test-integration`, yang mengubah path database ke target disposable dan memverifikasi nama database aktual sebelum tes. Jangan mengekspor DSN ke log atau melewati guard.

## Frontend

- Tes JavaScript: `npm run test:instant-search` (Node test runner terhadap `scripts/instant-search.test.cjs`).
- Build CSS: `npm run css:build`.

## Browser E2E

Persiapan satu kali:

```bash
npm install
npm run test:e2e:install
```

Jalankan seluruh alur settlement dari browser Chromium:

```bash
npm run test:e2e
```

Perintah tersebut membutuhkan `.env` dengan `TABUNGAN_SISWA_DATABASE_DSN`, PostgreSQL yang dapat diakses, `psql`, Go, dan Node.js. Script mengganti hanya nama database pada DSN menjadi `tabungan_siswa_fase168_clean_test`, lalu memastikan database aktual dan schema v13 bersih. Jangan arahkan tes ke database development atau produksi. Aplikasi lokal dijalankan otomatis pada port `8080`; tes gagal jika port tersebut sudah dipakai agar tidak tanpa sengaja menguji server lain.

Fixture akun Admin/Wali, siswa, kelas, tahun, dan rekening dibuat dengan ID khusus oleh `cmd/e2e-fixture`. Helper menolak target selain database uji yang diizinkan, menolak schema selain v13 bersih, dan menolak benturan ID. Setelah tes, script menghentikan server dan menghapus hanya fixture tersebut. Jangan jalankan Go integration suite bersamaan karena keduanya memakai database uji yang sama.

Tes di `e2e/settlement.spec.cjs` juga memeriksa halaman inbox Transaksi Terlewat untuk Wali dan Admin. Alur lain mencakup login Wali, dashboard dan laporan FASE 25, overview/readiness per wali, filter histori Admin menurut tahun dan kelas, settlement, konfirmasi penyerahan, login Admin dan tutup tahun; tes kedua memeriksa ubah username, nonaktif/aktif, reset password, login dengan password sementara, serta ganti password. Trace dan screenshot disimpan otomatis ketika tes gagal. Untuk memasang Chromium di lokasi cache khusus, set `PLAYWRIGHT_BROWSERS_PATH` saat menjalankan `npm run test:e2e:install` dan gunakan nilai yang sama untuk `npm run test:e2e`.

Jika proses dihentikan paksa sehingga trap cleanup tidak sempat berjalan, jalankan `npm run test:e2e:cleanup`. Perintah ini hanya menghapus ID fixture E2E yang tetap dan tetap menolak database selain database uji yang diizinkan.

## Pemeriksaan diff

- `git diff --check` untuk whitespace/conflict-marker issues pada diff.
- `gofmt` untuk file Go yang diubah.

## Perubahan skema

Tidak ada migration runner umum pada `cmd/web` atau Makefile. Migration berupa SQL bernomor dan tidak otomatis diterapkan oleh server saat startup. Sebelum mengubah atau menjalankan migration, baca kontrak operasional terkait, verifikasi status database tujuan, dan gunakan database disposable untuk verifikasi upgrade/rollback. Jangan menguji rollback pada database development atau produksi.

## Pelaporan hasil

Catat perintah persis, database tes yang diverifikasi (nama saja, tanpa DSN), hasil, dan kegagalan. Bedakan tes yang tidak dijalankan dari tes yang gagal. Jangan menyatakan seluruh aplikasi aman hanya berdasarkan unit test.
