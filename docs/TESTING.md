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

## Pemeriksaan diff

- `git diff --check` untuk whitespace/conflict-marker issues pada diff.
- `gofmt` untuk file Go yang diubah.

## Perubahan skema

Tidak ada migration runner umum pada `cmd/web` atau Makefile. Migration berupa SQL bernomor dan tidak otomatis diterapkan oleh server saat startup. Sebelum mengubah atau menjalankan migration, baca kontrak operasional terkait, verifikasi status database tujuan, dan gunakan database disposable untuk verifikasi upgrade/rollback. Jangan menguji rollback pada database development atau produksi.

## Pelaporan hasil

Catat perintah persis, database tes yang diverifikasi (nama saja, tanpa DSN), hasil, dan kegagalan. Bedakan tes yang tidak dijalankan dari tes yang gagal. Jangan menyatakan seluruh aplikasi aman hanya berdasarkan unit test.
