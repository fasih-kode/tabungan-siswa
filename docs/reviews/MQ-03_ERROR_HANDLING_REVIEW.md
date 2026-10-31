# MQ-03 — Review Konsistensi Error Handling

## Ringkasan eksekutif

Review ini hanya membaca kode pada HEAD `14df34a101931bc4465fdcb3e12697275d29a684`. Working tree awal bersih. Tidak ada source code, migration, atau database yang diubah maupun diakses.

Repository permintaan transaksi terlambat memakai `translateSettlementRepositoryError` pada operasi tulis dan menerjemahkan `sql.ErrNoRows` menjadi `repository.ErrNotFound` pada pembacaan satu baris. Metode list dan pemeriksaan `EXISTS` mengembalikan error query/scan secara langsung atau membungkus error query list dengan `%w`. Perbedaan tersebut tidak dengan sendirinya merupakan bug: operasi baca tidak diharapkan menghasilkan konflik unique, sementara error yang dibungkus tetap dapat diperiksa melalui `errors.Is`/`errors.As`.

Belum ditemukan bukti bahwa perubahan implementasi diperlukan. Kekurangan utama adalah cakupan tes langsung untuk repository permintaan transaksi terlambat: yang tersedia adalah satu integration test alur service; tidak ditemukan test khusus repository ini untuk cabang not-found, error scan/rows, atau penerjemahan unique constraint aktual. Rekomendasi terkecil adalah mempertahankan perilaku dan, bila supervisor menghendaki implementasi lanjutan, menambahkan regression test terarah sebelum mengubah error path. Perubahan teks diagnostik dengan `%w` dapat dipertimbangkan terpisah, tetapi bukan prasyarat correctness.

## Matriks operasi

| Operasi | Sumber error dan perilaku repository | `errors.Is` / kontrak | Dampak service dan HTTP | Bukti tes dan penilaian |
|---|---|---|---|---|
| `CreateLeavingTransactionRequest` | `ExecContext` error diteruskan ke `translateSettlementRepositoryError`. `nil` tetap `nil`; `sql.ErrNoRows` menjadi `repository.ErrNotFound`; PostgreSQL `23505` dan `23P01` menjadi `repository.ErrConflict`; error lain dikembalikan apa adanya. Translator tidak membungkus sentinel dengan error PostgreSQL asal. | `errors.Is(err, repository.ErrConflict/ErrNotFound)` bekerja untuk hasil sentinel. Setelah pemetaan konflik, `errors.As` ke `*pgconn.PgError` dan `errors.Is` ke error asal tidak lagi bekerja. Ini merupakan perilaku translator yang sama dengan repository settlement. | Service `Create` meneruskan error. Handler memetakan conflict ke HTTP 409, not-found ke 404, dan error lain ke 500 dengan pesan publik generik. | Integration test membuktikan create sukses dan penolakan bukti duplikat melalui pemeriksaan `HasEvidenceReference`; tes itu **tidak** membuktikan konflik unique yang terjadi langsung di `INSERT`. `KEEP`; tambahkan tes DB untuk cabang indeks unik sebelum mengubah translator. |
| `GetLeavingTransactionRequest` | Query/scanning melewati helper `get`. Error yang sama persis dengan `sql.ErrNoRows` diganti `repository.ErrNotFound`; error lain dikembalikan langsung. | Sentinel not-found tetap dikenali dengan `errors.Is`. `sql.ErrNoRows` sengaja tidak dipertahankan setelah dipetakan. Pemeriksaan menggunakan `==`, bukan `errors.Is`; saat ini `scanLeavingRequest` tidak membungkus error Scan sehingga `database/sql` mengembalikan sentinel langsung. | Service `Execute` meneruskan error; handler `Execute` mengubah `repository.ErrNotFound` menjadi 404. Error lain menjadi 500. | Ada happy-path dalam integration test alur approve/execute; tidak ditemukan assertion langsung untuk ID yang tidak ada pada repository ini. `KEEP` pada keadaan sekarang; perubahan ke `errors.Is` akan memperluas pemetaan bagi error terbungkus dan perlu tes/keputusan karena itu perubahan perilaku kecil. |
| `GetLeavingTransactionRequestForUpdate` | Memakai `get` yang sama dengan `Get`; SQL menambahkan `FOR UPDATE`, tetapi error mapping identik. | Sama dengan `Get`: `repository.ErrNotFound` stabil; error lain lewat langsung. | Service `Decide` dan `Execute` meneruskan error; handler mengirim 404 untuk not-found dan 500 untuk error internal. | Integration flow memanggil jalur ini pada keputusan/eksekusi, tetapi tidak memeriksa cabang not-found. `KEEP`; pertahankan mapping serta semantik lock. |
| `ListLeavingTransactionRequests` | Error `QueryContext` dibungkus sebagai `fmt.Errorf("list leaving transaction requests: %w", err)`. Error `scanLeavingRequest` dan `rows.Err()` dikembalikan langsung. | `%w` menjaga `errors.Is` dan `errors.As` untuk query error. Keduanya juga tetap bekerja pada error Scan/Rows yang langsung dikembalikan. Tidak ada pemetaan `sql.ErrNoRows` di list karena hasil kosong adalah slice kosong, bukan `ErrNoRows`. | Service `List` meneruskan error. Handler mencatat error ke log, lalu `httperror.Write`; error internal normal menjadi HTTP 500 dengan pesan generik. | Tidak ditemukan test langsung untuk operasi list repository atau jalur error-nya. Perbedaan konteks diagnostik query-vs-scan adalah `REFACTOR` opsional; bukan alasan untuk menambahkan translator atau mengubah status HTTP. |
| `UpdateLeavingTransactionRequest` | `ExecContext` error diterjemahkan helper. Error `RowsAffected()` langsung dikembalikan. Jumlah nol dipetakan menjadi `repository.ErrNotFound`; selainnya sukses. | Conflict/not-found sentinel dapat dikenali dengan `errors.Is`. Error `RowsAffected` mempertahankan tipe/rantai aslinya; tidak ada konteks tambahan. | Service `Decide`/`Execute` meneruskan error; handler memetakan not-found ke 404, conflict ke 409, dan error tak dikenal ke 500. | Tidak ditemukan tes repository ini untuk update nol baris atau error `RowsAffected`; alur integration test mencakup update sukses. Repository settlement juga mengembalikan `ErrNotFound` untuk nol baris, tetapi membungkus error `RowsAffected` dengan `%w`. `KEEP` untuk sentinel; pembungkusan konteks adalah `REFACTOR` opsional yang mengubah teks error namun menjaga rantai. |
| `HasPendingLeavingTransactionRequests` | `QueryRowContext(...).Scan(&b)` dikembalikan langsung. Untuk `SELECT EXISTS`, query normal menghasilkan satu boolean; kegagalan DB/Scan tetap error teknis. | Error asal tidak dibungkus, sehingga `errors.Is`/`errors.As` masih dapat memeriksanya. `sql.ErrNoRows` yang hipotetis bukan `repository.ErrNotFound` dan akan berstatus 500. | Service kesiapan tahun meneruskan error; handler yang memanggil jalur layanan terkait menggunakan pemetaan umum, sehingga kegagalan DB menjadi HTTP 500. | Integration test kesiapan membuktikan nilai pending pada alur valid; tidak menguji kegagalan query. `KEEP`; tidak ada pemetaan not-found yang diperlukan untuk scalar `EXISTS`. |
| `HasPendingForSavingsAccount` | Sama: error `QueryRowContext`/`Scan` dikembalikan langsung. | Rantai error asal terpelihara karena tidak dibungkus. | Pemanggil service settlement meneruskan error; HTTP default 500 bila tampil ke handler. | Tidak ditemukan test langsung untuk query ini pada alur permintaan terlambat. `KEEP`; tambahkan tes kegagalan hanya bila ada perubahan pada jalur ini. |
| `HasEvidenceReference` | Sama: error query/Scan dikembalikan langsung; hasil boolean dipakai service untuk mengembalikan `repository.ErrConflict` sebelum insert jika referensi sudah digunakan. Indeks unik parsial di migration tetap menjadi perlindungan database untuk race. | Error query mempertahankan rantai; konflik hasil pemeriksaan adalah sentinel stabil. Konflik dari indeks DB saat race dipetakan oleh translator `Create` ke `repository.ErrConflict`, tanpa mempertahankan detail PG asal. | Service `Create` meneruskan error; konflik menjadi HTTP 409. | Integration test memverifikasi penolakan referensi duplikat secara berurutan, bukan race yang mencapai indeks unik. `KEEP`; regression test race/unique DB adalah penguatan cakupan, bukan bukti bahwa perilaku saat ini salah. |
| `scanLeavingRequest` | Error `row.Scan` dikembalikan langsung. Setelah Scan sukses, nominal teks diproses oleh `parseSettlementAmount`; error parser dikembalikan langsung. | Error Scan mempertahankan rantainya. Parser mengembalikan error biasa untuk nilai kosong/non-integer; untuk `strconv.ParseInt` error dibungkus `%w`, sehingga tipe `*strconv.NumError` masih bisa ditemukan. Tidak ada sentinel domain/repository untuk data tersimpan yang malformed. | Service menerima error dari repository dan meneruskannya. Handler memetakan error non-sentinel ke 500 dan menampilkan pesan umum, bukan detail parser. | Tidak ditemukan tes `scanLeavingRequest`/parser dalam scope ini. `KEEP` untuk error malformed sebagai internal/server error; jangan mengubahnya menjadi HTTP 400 karena sumbernya adalah data baca tersimpan, bukan input baru pada request ini. |

## Perilaku yang wajib dipertahankan

1. Not-found pada Get/ForUpdate dan update nol baris tetap dapat dikenali sebagai `repository.ErrNotFound`; HTTP tetap 404.
2. Unique/exclusion violation yang dipetakan translator tetap menjadi `repository.ErrConflict`; HTTP tetap 409. Jangan membocorkan detail constraint/database ke klien.
3. Error tak dikenal dari DB, Scan, Rows, dan parsing data tersimpan tetap error internal (HTTP 500) dengan pesan generik.
4. `fmt.Errorf(... %w ...)` pada query list harus tetap mempertahankan rantai error. Jangan menggantinya dengan `%v` atau error baru tanpa `%w`.
5. Jangan mengubah SQL, status transaksi, urutan lock, batas transaksi/audit, atau perilaku keuangan sebagai bagian dari review error handling.
6. `sql.ErrNoRows` pada helper Get saat ini dibandingkan langsung. Jangan menggantinya dengan `errors.Is` tanpa menyetujui perluasan perilaku untuk error pembungkus.

## Temuan dan klasifikasi

- **KEEP — translator bersama:** penggunaan translator pada Create/Update sesuai pola repository settlement. Pemetaan `23505`/`23P01` ke sentinel konflik mempertahankan kontrak publik, tetapi sengaja menghilangkan tipe/detail PostgreSQL dari error yang keluar.
- **KEEP — sentinel not-found:** `get` mengubah `sql.ErrNoRows`; Update mengubah affected rows nol. Keduanya sesuai kontrak repository generik dan status HTTP 404.
- **REFACTOR opsional — konteks diagnostik:** List hanya memberi konteks pada `QueryContext`; error scan/rows dan error `RowsAffected` Update tidak diberi konteks. Jika supervisor menghendaki keseragaman log, tambahkan hanya pembungkus `%w` dan regression test `errors.Is`/`errors.As`. Ini mengubah string `Error()` tetapi seharusnya menjaga rantai error dan pemetaan status; tetap minta persetujuan sebelum perubahan.
- **BUG CANDIDATE — celah pengujian konflik konkuren:** tes layanan mengecek referensi duplikat melalui pre-check; belum ada bukti test langsung bahwa constraint unik aktif referensi bukti saat dua request berlomba diterjemahkan menjadi `repository.ErrConflict`. Implementasi translator sudah memiliki pemetaan untuk SQLSTATE 23505. Buat regression test pada tiket implementasi terpisah bila supervisor meminta; tidak ada bukti cukup untuk menyebut ini bug implementasi.
- **BACKLOG — cakupan error repository:** belum ditemukan test langsung untuk request repository atas `Get` not-found, `Update` nol baris, error query/Scan/Rows, maupun nominal malformed. Settlement repository memberi contoh tes analog untuk not-found, update nol baris, dan unique conflict, tetapi bukan bukti khusus bagi repository ini.

## Usulan perubahan minimal dan pengujian jika disetujui

Usulan default: **tidak mengubah implementasi**. Jika supervisor menyetujui peningkatan cakupan, buat perubahan terpisah yang hanya menambah tes integration repository berikut:

1. Create duplikat yang benar-benar mencapai unique index menghasilkan `errors.Is(err, repository.ErrConflict)`.
2. Get dan GetForUpdate untuk ID tak ada menghasilkan `errors.Is(err, repository.ErrNotFound)`.
3. Update ID tak ada menghasilkan `errors.Is(err, repository.ErrNotFound)`.
4. Scan nilai nominal malformed gagal dan tidak berubah menjadi input error bisnis/HTTP 400.
5. Query/Rows error tetap menjadi error internal; uji rantai hanya bila driver/fake dapat menyimulasikan tanpa menambah abstraksi produksi.

Jika konteks diagnostik `%w` disetujui, batasi ke metode repository ini, jangan menambah helper generik, dan pastikan regression test untuk sentinel serta status HTTP tidak berubah. Tes package `internal/repository/postgres` menjalankan `TestMain` yang memerlukan PostgreSQL; gunakan hanya `make test-integration` dengan database disposable terverifikasi.

## Urutan rekomendasi

1. Supervisor meninjau matriks ini dan memutuskan apakah hanya menambah tes atau juga mengubah konteks diagnostik.
2. Jika disetujui, buka MQ implementasi terpisah dengan scope eksplisit; jangan mencampurnya dengan BUG-01, BUG-02, atau SEC-01.
3. Jalankan tes regression pada PostgreSQL disposable, `go build ./...`, `go vet ./...`, dan `git diff --check`; periksa bahwa HTTP 404/409/500 dan `errors.Is` tetap sesuai.

## Sumber kode dan tes

- [Repository permintaan transaksi terlambat](../../internal/repository/postgres/leaving_transaction_request_repository.go#L65)
- [Translator error settlement bersama](../../internal/repository/postgres/settlement_repository.go#L299)
- [Interface repository](../../internal/repository/contracts.go#L255)
- [Service permintaan transaksi terlambat](../../internal/service/leaving_transaction_service.go#L23)
- [Pemetaan HTTP](../../internal/http/httperror/http_error.go#L12)
- [Tes pemetaan HTTP dan error generik](../../internal/http/httperror/http_error_test.go#L16)
- [Integration test alur permintaan](../../internal/service/service_transaction_integration_test.go#L223)
- [Constraint indeks unik bukti aktif](../../migrations/000013_late_transaction_requests.up.sql#L26)
- [Tes repository settlement sebagai pola analog](../../internal/repository/postgres/settlement_repository_test.go#L49)
