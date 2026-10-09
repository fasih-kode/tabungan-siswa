# FASE 16.8: Pemulihan metadata migrasi

Status: **RECOVERY BERHASIL DIJALANKAN — METADATA VERSI 6 TERVERIFIKASI**

## Temuan yang menjadi dasar

Pemeriksaan database pengembangan menunjukkan:
- `schema_migrations` mencatat `version=4, dirty=false`.
- Constraint exclusion `academic_years_period_no_overlap` dari migrasi 000005 sudah ada.
- Objek 000006 terlihat sudah ada: tujuh trigger, fungsi terkait, indeks unik pending transfer, dan constraint alasan penolakan.
- Tabel `migration_000006_student_class_history_validity_backup` ada dan kosong.
- `student_class_histories` saat pemeriksaan berisi nol baris.

Ini bukan alasan untuk menjalankan `000006 up.sql` ulang. SQL migrasi awal menggunakan `CREATE TABLE`, `CREATE FUNCTION`, `CREATE TRIGGER`, `CREATE INDEX`, dan `ADD CONSTRAINT` tanpa mekanisme idempotensi. Karena sebagian objek sudah ada, menjalankannya ulang memang gagal.

## Pemulihan

Skrip `scripts/recover_fase_16_8_migration_state.sql` hanya merekonsiliasi metadata `schema_migrations` dari versi 4 ke versi 6 setelah validasi prasyarat:
- database tidak dalam status dirty;
- versi tercatat adalah 4 atau 6;
- constraint 000005 ditemukan;
- tabel backup memiliki lima kolom yang diharapkan dan masih kosong;
- ketujuh trigger mengarah ke fungsi yang diharapkan;
- definisi fungsi memiliki klausa kontrak penting yang diharapkan;
- indeks unik transfer pending dan constraint alasan penolakan tervalidasi.

Skrip **tidak** menghapus tabel backup, tidak mengubah data aplikasi, tidak membuat/menghapus trigger atau fungsi, dan tidak menghapus constraint 000005. Semua validasi dan perubahan metadata berada dalam satu transaksi. Jika validasi gagal, transaksi dibatalkan dan metadata tidak diubah.

## Cara menjalankan

Dari root repo:

```bash
cd ~/Projects/tabungan-siswa || exit 1
set -a
. ./.env
set +a

psql "$TABUNGAN_SISWA_DATABASE_DSN" -X -v ON_ERROR_STOP=1 \
  -f scripts/recover_fase_16_8_migration_state.sql
```

Jalankan hanya setelah meninjau skrip. Jangan menjalankan `migrations/000006_fase_16_8_contract_enforcement.up.sql` lagi pada database yang sudah memiliki objek-objek tersebut.

## Verifikasi setelah pemulihan

```bash
psql "$TABUNGAN_SISWA_DATABASE_DSN" -X -P pager=off -c \
  "TABLE public.schema_migrations;"

go test ./internal/domain -count=1
go test ./internal/repository/postgres -count=1
go test ./internal/service -count=1
go test ./...
go test -race ./...
go vet ./...
go build ./...
git diff --check
```

Skrip recovery telah berhasil dijalankan pada database lokal; hasilnya mengonfirmasi objek 000005/000006 dan merekonsiliasi `schema_migrations` ke versi 6 dengan `dirty=false`. Verifikasi kode lokal telah lulus, dan pengujian rollback `000006 up/down` pada database disposable berhasil untuk skenario data tidak berubah, berubah, dan dihapus. FASE 16.8 tetap belum boleh dikunci sampai final review dan commit selesai. Skrip recovery ini bersifat insidental, bukan bagian dari alur migrasi rutin.
