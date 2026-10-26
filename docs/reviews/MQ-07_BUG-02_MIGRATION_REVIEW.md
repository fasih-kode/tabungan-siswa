# MQ-07-REVIEW-01 — Audit BUG-02 dan rancangan migration aman

## Status dan batas audit

- Baseline: branch `review`, commit `ab7ffdf0c394ac851506dea46f9653bdcfe1b1f6`; working tree bersih saat audit dimulai.
- Audit dilakukan dari source, migration, kontrak, dan test repository. Database development/production tidak diakses; tidak ada migration atau test yang dijalankan.
- Fakta tentang data yang sudah tersimpan tidak dapat dipastikan dari audit ini. Tidak ada dasar untuk menyatakan data produksi mengandung nominal pecahan.
- Dokumen ini merancang kandidat migration `000014`; bukan persetujuan untuk membuat atau menerapkannya.

## Ringkasan temuan

BUG-02 terkonfirmasi sebagai ketidaksesuaian perlindungan skema: `leaving_transaction_requests.amount` bertipe `NUMERIC(15,2)` dan hanya memiliki pemeriksaan `amount > 0` pada migration 000013. Migration 000002 memasang `amount = trunc(amount, 0)` untuk `transactions` dan `savings_settlements`, tetapi tidak menyentuh tabel permintaan transaksi terlambat yang baru dibuat kemudian.

Jalur aplikasi yang diperiksa saat ini membatasi nominal ke `domain.Money` (bilangan bulat), dan parser repository menolak bagian pecahan yang bukan nol. Ini mengurangi kemungkinan data pecahan melalui alur normal aplikasi, tetapi tidak menggantikan constraint database: SQL langsung atau klien lain masih dapat menyimpan nilai pecahan positif.

Jika ada permintaan lama dengan bagian pecahan, parser `scanLeavingRequest` akan gagal ketika halaman/list membacanya, ketika service mengambil permintaan untuk persetujuan, atau ketika mengambil permintaan untuk eksekusi. Dengan demikian, constraint yang hilang berpotensi membuat data yang diterima database tidak dapat diproses oleh aplikasi. Ini dampak yang dapat disimpulkan dari kode, bukan insiden atau data pecahan yang telah ditemukan.

## Bukti dari skema, domain, repository, service, dan tes

| Area | Fakta terverifikasi | Referensi |
|---|---|---|
| Transaksi | `transactions.amount NUMERIC(15,2)` memiliki `amount > 0` pada skema awal dan constraint integer `amount = trunc(amount, 0)` ditambahkan oleh migration 000002. | `migrations/000001_init_schema.up.sql:153-172`; `migrations/000002_align_domain_contract.up.sql:34-37` |
| Settlement | `savings_settlements.amount NUMERIC(15,2)` menerima nol tetapi tidak nilai negatif; migration 000002 menambahkan constraint integer yang sama. | `migrations/000001_init_schema.up.sql:174-192`; `migrations/000002_align_domain_contract.up.sql:39-42` |
| Permintaan terlambat | Kolom `amount NUMERIC(15,2) NOT NULL CHECK (amount > 0)` hanya mewajibkan nilai positif, bukan bilangan bulat. | `migrations/000013_late_transaction_requests.up.sql:3-9` |
| Domain | `Money` adalah alias bertipe `int64`; `MaxMoney` adalah `9_999_999_999_999`. Constructor permintaan mensyaratkan nominal positif dan tidak melebihi batas itu. | `internal/domain/money.go:3-5`; `internal/domain/leaving_transaction_request.go:34-47` |
| Service | `Create` menerima `domain.Money`, membangun permintaan melalui constructor domain, lalu menyimpan lewat repository. | `internal/service/leaving_transaction_service.go:23-94` |
| Readback | Query request mengambil `amount::text`; `scanLeavingRequest` meneruskannya ke `parseSettlementAmount`. Parser menolak digit pecahan selain nol dan mengembalikan error. | `internal/repository/postgres/leaving_transaction_request_repository.go:26-40, 270-305`; `internal/repository/postgres/settlement_repository.go:260-294` |
| Persetujuan/eksekusi | `Decide` mengambil request dengan `Get...ForUpdate`; `Execute` membaca request biasa lalu mengambilnya lagi dengan `FOR UPDATE`. Kedua jalur memakai scanner yang sama. | `internal/service/leaving_transaction_service.go:135-164, 176-230` |
| Kebijakan uang | Kontrak keuangan menyebut uang sebagai integer, tanpa pecahan; nilai database untuk transaksi dan settlement juga dibatasi integer. | `docs/contracts/fase-16-5-financial-contract-audit.md:294-339` |
| Tes yang tersedia | Tes domain membuat request dengan amount bulat dan menguji state/reason/evidence; integration test service menguji alur create-approve-execute sekali. Tes tersebut tidak menguji penolakan nominal pecahan oleh DB atau readback nilai pecahan. | `internal/domain/leaving_transaction_request_test.go:11-43`; `internal/service/service_transaction_integration_test.go:223-310`; `internal/repository/postgres/transaction_repository_test.go:54-88` |

## Dampak potensial terhadap integritas

1. Constraint database tidak selaras dengan kontrak uang integer dan constraint tabel ledger yang berdekatan.
2. Nilai pecahan positif dapat lolos pemeriksaan schema 000013 jika ditulis di luar constructor/service.
3. Bila tersimpan, pembacaan request gagal pada parser sebelum keputusan atau eksekusi; mekanisme itu menolak/menggagalkan readback, tetapi tidak memperbaiki atau membulatkan row.
4. Nilai integer yang sah tetap memiliki batas `NUMERIC(15,2)` dan batas domain. Constraint integer baru tidak mengubah rentang maupun aturan `amount > 0`.

Audit tidak mengakses database dan tidak mereproduksi penulisan pecahan. BUG-02 adalah celah constraint yang terbukti dari definisi schema, bukan bukti bahwa ada row yang melanggar di database mana pun.

## Alternatif migration kandidat 000014

Nama constraint yang konsisten dengan migration 000002: `leaving_transaction_requests_amount_integer_check`.

### A. Tambah constraint tervalidasi langsung

`ADD CONSTRAINT ... CHECK (amount = trunc(amount, 0))` akan memeriksa semua row yang sudah ada ketika constraint dibuat. Jika ada pecahan, operasi gagal tanpa mengubah nominal. Pendekatan ini sederhana dan cocok bila tabel kecil serta pemeriksaan data sebelumnya menunjukkan semua row valid. Pada PostgreSQL, penambahan constraint tervalidasi umumnya mengambil `ACCESS EXCLUSIVE` lock dan melakukan scan; lama lock/scan bergantung ukuran dan beban tabel.

### B. Tambah `NOT VALID`, lalu validasi

Tambahkan CHECK dengan `NOT VALID`, kemudian jalankan `VALIDATE CONSTRAINT` sebagai langkah eksplisit. CHECK langsung berlaku pada INSERT/UPDATE baru, sementara row lama belum dinyatakan tervalidasi sampai validasi berhasil. PostgreSQL mendokumentasikan bahwa validasi memindai row lama dan memakai `SHARE UPDATE EXCLUSIVE` lock; penambahan constraint tanpa scan menghindari scan awal, tetapi tetap memerlukan lock untuk perubahan metadata. Jika row historis pecahan ada, validasi gagal dan tidak boleh dilewati atau diubah otomatis.

Pilihan ini memberi kontrol lebih baik atas waktu scan/lock, tetapi memperumit alur versi: antara dua perintah constraint ada dan berlaku untuk penulisan baru namun belum tervalidasi. Jika alat migration tidak menjalankan kedua langkah dan pencatatan versi secara atomik, kegagalan dapat meninggalkan constraint `NOT VALID` dan status migration yang perlu pemulihan terencana. Dokumentasi PostgreSQL: [ALTER TABLE — ADD CONSTRAINT NOT VALID dan VALIDATE CONSTRAINT](https://www.postgresql.org/docs/current/sql-altertable.htm).

### Rekomendasi untuk supervisor

Untuk ukuran tabel saat ini yang belum diukur, pilihan paling aman belum dapat diputuskan hanya dari source. Rekomendasi desain adalah:

1. Pemeriksaan awal read-only menghitung row pecahan (`amount <> trunc(amount, 0)`) dan hanya menampilkan jumlah, bukan identitas siswa atau nominal.
2. Jika hasilnya nol dan ukuran/lock dapat diterima, gunakan constraint tervalidasi langsung agar satu migration menghasilkan schema yang konsisten.
3. Jika scan langsung menimbulkan risiko lock yang berarti, pertimbangkan pola `NOT VALID` lalu `VALIDATE CONSTRAINT`, dengan kedua tahap dan perilaku gagal/dirty-state diuji pada PostgreSQL disposable serta prosedur pemulihan ditetapkan sebelum deployment.
4. Jika ditemukan row pecahan, hentikan rollout. Jangan membulatkan, menghapus, mengubah, mengabaikan, atau menonaktifkan constraint. Minta keputusan owner tentang koreksi setiap data berdasarkan bukti keuangan, lalu jalankan pemeriksaan ulang sebelum validasi.

Ekspresi `amount = trunc(amount, 0)` menyamai constraint yang sudah dipakai pada transaksi dan settlement. Kolom `NOT NULL` tetap mengurus NULL; constraint positif yang ada tetap mengurus nol/negatif.

## Pemeriksaan data historis sebelum penerapan

Pada database tujuan, sebelum migration dan melalui prosedur operasional yang disetujui:

1. Pastikan database, host/port, role, schema, versi migration, dan status dirty secara independen; jangan menyimpulkan aman hanya dari nama database.
2. Jalankan pemeriksaan agregat untuk menghitung row yang memenuhi `amount <> trunc(amount, 0)`. Hindari menyalin data pribadi/keuangan ke log atau laporan publik.
3. Jika ada pelanggaran, identifikasi dan rekonsiliasi melalui prosedur keuangan yang disetujui pemilik; migration sendiri tidak mengubah row.
4. Ulangi hitungan dan pastikan nol sesaat sebelum validasi agar tidak ada penulisan baru di antara pemeriksaan dan penerapan. Constraint database menjadi perlindungan terhadap race setelah aktif.
5. Database kosong: tidak ada row lama untuk divalidasi; migration tetap perlu diuji dari schema v13 kosong.
6. Database berisi row yang semuanya valid: validasi seharusnya lulus; ukur waktu dan dampak lock di disposable dengan volume representatif sebelum jadwal deployment.

Query audit tidak dijalankan dalam MQ-07 karena tugas ini read-only terhadap source dan melarang akses database.

## Matriks regression test yang diperlukan

| Skenario | Cakupan dan hasil yang diharapkan | Status bukti saat audit |
|---|---|---|
| Nominal positif bulat | Simpan request positif bulat melalui domain/service dan pastikan readback serta siklus approval/execution tetap berhasil. | Ada integration alur umum dengan 12.000; belum ada tes khusus constraint request. |
| Pecahan ditolak DB | Pada DB disposable v14, INSERT langsung `amount` pecahan positif dengan fixture valid harus gagal sebagai violation CHECK; jangan melewati service untuk jalur produk, hanya untuk menguji pertahanan DB. | Belum ada. |
| Nol dan negatif tetap ditolak | Uji nol/negatif pada DB; constructor domain juga menolak nilai nonpositif. | Constructor menegakkan validasi; unit test request yang ditemukan belum mengassert nol/negatif. DB migration test belum ada. |
| Row historis valid dapat dibaca | Siapkan row integer pada schema v13 sebelum upgrade, upgrade ke v14, lalu list/get/readback dan alur approval tetap sukses. | Ada alur integration umum; belum menguji upgrade fixture lintas versi. |
| Row historis pecahan menghambat validasi | Pada disposable v13, sisipkan pecahan sebagai row legacy; proses migrasi/validasi harus gagal terkontrol tanpa mengubah nilai. Pastikan versi/dirty-state sesuai kontrak alat migrasi dan dokumentasikan pemulihan. | Belum ada. |
| Upgrade v13 → v14 | Terapkan urutan migration lengkap dari schema v13, pastikan constraint ada/valid dan `schema_migrations` mencatat versi 14 bersih. | Belum ada. |
| Rollback v14 → v13 | Di disposable, rollback hanya menjatuhkan constraint baru; semua row tetap sama dan schema kembali sesuai v13. Pastikan kelemahan constraint memang kembali sebagai konsekuensi rollback. | Belum ada. |
| Dirty state | Verifikasi metadata setelah upgrade sukses, kegagalan yang disengaja pada disposable, rollback sukses, dan kegagalan rollback. Jangan mengedit metadata manual sebagai jalan pintas. | Belum ada. |

Semua tes migration harus berjalan pada PostgreSQL disposable dengan cluster, data directory, port, role, database, dan isi awal diverifikasi. Tidak boleh memakai database development/production.

## Risiko deployment, rollback, dan hal yang belum diketahui

- Migration berjalan manual dari file SQL bernomor; `docs/TESTING.md` menyatakan tidak ada runner migration umum pada server/Makefile. Mekanisme produksi yang mencatat `schema_migrations`, tool/version yang menjalankan file, batas transaksi migration, dan prosedur penanganan dirty belum didokumentasikan lengkap.
- `scripts/recover_fase_16_8_migration_state.sql` menunjukkan metadata `schema_migrations(version, dirty)` pernah dipakai dan menolak dirty state secara otomatis. Itu bukan bukti alat yang menjalankan migration saat ini atau prosedur pemulihan yang boleh dipakai untuk 000014.
- Tidak diketahui apakah database mana pun berisi pecahan, berapa besar tabel, beban tulis saat deployment, atau berapa lama scan constraint berlangsung.
- Migration up yang gagal pada data lama tidak boleh diulang secara buta atau diikuti manipulasi version/dirty flag. Hentikan deployment, pertahankan bukti keadaan schema, lalu gunakan prosedur pemulihan yang disetujui.
- Down migration yang hanya melepas CHECK aman terhadap data karena tidak menulis row, tetapi mengembalikan celah integritas. Rollback bukan pengganti perbaikan data dan sesudah rollback aplikasi tetap mengandalkan validasi service/parser.
- Perlu dipastikan apakah pola file saat ini mengharapkan satu up/down statement dan bagaimana runner memperlakukan kegagalan sebagian. `000002` sendiri menjalankan beberapa statement; perilaku transaksional runner perlu diketahui sebelum menetapkan pilihan `NOT VALID` dua tahap.

## Keputusan supervisor yang dibutuhkan sebelum implementasi

1. Setujui penambahan migration baru bernomor `000014` dengan constraint bernama `leaving_transaction_requests_amount_integer_check` dan ekspresi setara `amount = trunc(amount, 0)`.
2. Pilih constraint tervalidasi langsung atau `NOT VALID` + `VALIDATE CONSTRAINT` setelah hasil audit data, ukuran tabel, dan perilaku runner diketahui.
3. Tetapkan respons jika pemeriksaan menemukan row pecahan: pemilik harus menyetujui koreksi berbasis bukti satu per satu; migration tidak boleh mengubah data sendiri.
4. Tetapkan runner, urutan deployment, transaksi, serta prosedur recovery dirty state untuk migration 000014.
5. Setujui regression tests dan kriteria rollback di atas sebelum implementasi.

Sampai keputusan tersebut diberikan, BUG-02 tetap berupa celah schema yang terkonfirmasi; tidak ada migration, perubahan data, atau perbaikan implementasi dalam pekerjaan ini.
