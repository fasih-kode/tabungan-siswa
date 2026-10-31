# MQ-08-CONTRACT-01 — Rancangan Siswa Kembali dan Siklus Tabungan

> **Pembaharuan kontrak pindah:** MQ-08-CONTRACT-02 disetujui setelah rancangan ini ditulis. Untuk alur pindah baru, settlement dan handover yang diwajibkan harus selesai sebelum siswa `LEFT`; rekomendasi lock table di bawah yang menyebut `Student LEFT` sebagai prasyarat settlement adalah snapshot rancangan lama dan digantikan oleh [amendment terbaru](../contracts/student-transfer-financial-clearance-amendment.md). Rekomendasi multi-siklus lainnya tetap belum final.

**Jenis dokumen:** rancangan teknis awal; bukan persetujuan implementasi.
**Kontrak bisnis:** [Kontrak Siswa Kembali dan Siklus Tabungan](../contracts/student-return-and-savings-cycle-contract.md).
**Audit konkurensi sumber:** [MQ-08-REVIEW-01](MQ-08_CONCURRENCY_SETTLEMENT_REVIEW.md).
**Baseline yang diperiksa:** `fe3b754cc880ebe21977bd6de06648e29b95ae86`.

## 1. Ringkasan rekomendasi

Gunakan identitas `students.id` yang sama sepanjang siswa pindah/kembali. Pertahankan placement kelas lama sebagai interval historis baru, jangan buka atau timpa baris lama. Perlakukan satu row `savings_accounts` sebagai satu siklus ledger: transaksi dan settlement yang sudah ada tetap berelasi ke account ID yang sama; satu siklus berikutnya menggunakan account ID baru dan saldo ledger awal nol.

Pilihan awal yang paling kecil untuk menjaga relasi historis adalah **row-per-cycle pada `savings_accounts` dengan nomor siklus eksplisit**, bukan tabel siklus terpisah. Namun perubahan ini menyentuh invariant schema, service, repository, reporting, HTTP, dan UI; belum boleh diterapkan sebelum desain detail dan migration disetujui.

Pisahkan tanggung jawab: Admin melakukan perubahan lifecycle dan placement; Wali membuat/mengelola siklus finansial dalam assignment; Siswa tetap read-only. Sistem harus menggunakan satu urutan lock lintas lifecycle dan finansial. Jangan sekadar menambahkan lock student ke settlement.

## 2. Status keputusan dan rekomendasi

### Keputusan bisnis yang disetujui pemilik

- Reaktivasi biasa diperbolehkan pada tahun yang sama jika rekening belum settled.
- Jika rekening telah settled, gunakan alur khusus Siswa Kembali; placement baru dibuat dan siklus baru dapat dibuat bila dibutuhkan.
- Kembali pada tahun berbeda menggunakan placement dan siklus tahun tersebut.
- Identitas siswa dan seluruh histori class/financial/settlement/handover/audit tetap dipertahankan.
- Tidak ada pembatalan settlement otomatis.
- Admin mengelola lifecycle/placement, sedangkan mutasi keuangan tetap kewenangan Wali sesuai scope.

### Rekomendasi teknis yang belum disetujui

- Baris `savings_accounts` menjadi representasi siklus; tambah `cycle_number` positif dan identitas account tetap dipakai semua ledger FK.
- Batasi maksimal satu siklus `OPEN` per siswa/tahun; siklus settled lama tetap tersimpan.
- Gunakan lookup eksplisit untuk siklus aktif, daftar semua siklus, dan detail berdasarkan ID; jangan pilih account dengan query nondeterministik.
- Terapkan lock order konseptual `academic_year → account (urut ID bila lebih dari satu) → student`; semua pemeriksaan status dilakukan saat lock yang relevan ditahan.
- Tambahkan test concurrency PostgreSQL deterministik sebelum menyatakan race aman.

## 3. Model data

### Fakta schema saat ini

`student_class_histories` sudah menyimpan placement per siswa/tahun sebagai `DATERANGE`, dengan exclusion constraint yang menolak overlap untuk siswa dan tahun yang sama ([000001 schema, class history](../../migrations/000001_init_schema.up.sql#L64-L81)). Ini cocok untuk menambah interval kelas baru tanpa mengubah interval lama.

`savings_accounts` saat ini unik pada `(student_id, academic_year_id)` dan statusnya `OPEN` atau `SETTLED`. `transactions` dan `savings_settlements` memiliki FK ke `savings_account_id`, bukan langsung ke siswa/tahun ([000001 schema, accounts/ledger](../../migrations/000001_init_schema.up.sql#L130-L188)). Account sudah menjadi batas ledger yang baik untuk memisahkan saldo per siklus.

### Alternatif model

| Alternatif | Keuntungan | Biaya/risiko | Penilaian |
|---|---|---|---|
| Hapus unique `(student, year)` tanpa identitas siklus tambahan | Perubahan kolom kecil. | Tidak ada urutan siklus eksplisit; query tunggal dapat mengembalikan beberapa row; pemilihan “current” menjadi ambigu; laporan dapat menggandakan siswa/saldo. | Tidak direkomendasikan. |
| **Setiap row account adalah siklus; tambahkan `cycle_number`** | Relasi transaksi/settlement/audit tetap memakai ID account yang sama; data historis tidak perlu dipindahkan; siklus mudah diurutkan. | Unique index dan semua lookup/report harus diperbarui; harus ada aturan satu OPEN dan alokasi nomor yang aman saat concurrency. | **Rekomendasi awal paling sederhana**, menunggu review teknis. |
| Buat tabel induk `savings_cycles` dan relasi account ke cycle | Memisahkan konsep periode siklus dari rekening/ledger jika kelak satu siklus memiliki beberapa rekening atau produk. | Entitas, FK, join, lifecycle, migration, dan API baru; saat ini satu account sudah cukup menjadi batas ledger. | Tidak direkomendasikan kecuali kebutuhan domain baru membuktikan account ≠ cycle. |

### Bentuk schema yang disarankan untuk dievaluasi

Jangan mengedit migration lama. Migration baru (nomor ditentukan saat implementasi) perlu dirancang sebagai berikut:

1. Tambah `cycle_number` integer dengan nilai lama `1`, `NOT NULL`, dan `CHECK (cycle_number > 0)`. Semua row historis existing menjadi siklus pertama tanpa mengubah account ID, nilai, status, timestamps, transaksi, settlement, cash movement, atau audit.
2. Ganti unique lama dengan unique `(student_id, academic_year_id, cycle_number)`.
3. Tambahkan partial unique index untuk paling banyak satu account berstatus `OPEN` bagi `(student_id, academic_year_id)`. Pastikan strategi ini kompatibel dengan transaksi reopen yang telah disetujui: reopen account lama harus tetap menjadi satu-satunya siklus OPEN; account siklus baru tidak boleh dibuat bersamaan.
4. Alokasikan nomor siklus di dalam transaksi setelah lock yang menyerialisasi semua create untuk student/tahun. Jangan memakai `MAX(cycle_number)+1` tanpa lock yang sama.
5. Jangan menyalin nominal settlement lama sebagai saldo awal. Ledger baru kosong dan mulai `Rp. 0`; transfer uang antar siklus bukan bagian dari kontrak.

Migration harus diuji pada schema kosong dan database disposable yang berisi data valid. Down migration tidak boleh menghapus siklus atau menggabungkan transaksi. Jika sudah ada lebih dari satu siklus untuk student/tahun, rollback ke constraint lama harus berhenti dengan pesan eksplisit; pemulihan dilakukan melalui rencana forward-fix yang disetujui, bukan penghapusan atau pemindahan data.

### Backward compatibility dan permukaan yang terdampak

- **Repository/API:** `GetByStudentAndAcademicYear` saat ini mengasumsikan hasil tunggal. Ganti dengan API semantik jelas seperti `ListByStudentAndYear`, `GetOpenByStudentAndYear`, dan `GetByAccountID`; daftar cycle harus terurut `cycle_number`. Jangan gunakan `LIMIT 1` untuk menyembunyikan multi-row.
- **Account creation:** saat ini hanya Wali yang dapat membuat account, dan service memeriksa siswa ACTIVE, tahun OPEN, serta scope kelas ([savings_account_service.go](../../internal/service/savings_account_service.go#L14-L58)). Pertahankan aturan itu. Admin menyelesaikan student return; jika tabungan baru diperlukan, UI mengarahkan Wali yang memiliki scope untuk membuat siklus.
- **Eligibility database:** trigger `savings_accounts_creation_eligibility_trigger` mengambil lock tahun `FOR SHARE` lalu siswa `FOR SHARE` dan memvalidasi OPEN/ACTIVE ([000006 enforcement](../../migrations/000006_fase_16_8_contract_enforcement.up.sql#L150-L189)). Ini melindungi status pada INSERT, tetapi urutannya juga berarti `Reactivate` saat ini (student `FOR UPDATE` lalu year `FOR UPDATE`) dapat berpotensi membentuk lock cycle dengan create account baru. Rancangan lock dan test harus meliputi trigger path, bukan hanya service Go.
- **Savings page/passbook:** tampilkan pilihan siklus dan penanda aktif/selesai; histori setiap account dibuka berdasarkan account ID. Jangan gabungkan transaksi dua siklus menjadi satu passbook tanpa label yang jelas.
- **Transaction/settlement:** relasi account ID sudah memisahkan ledger. Pastikan operasi hanya menggunakan cycle ID terpilih, menolak account settled untuk transaksi biasa, dan tidak mengubah settlement historis. Reopen tetap memakai workflow approval yang sudah ada dan tidak menjadi efek samping return.
- **Dashboard/laporan:** reporting SQL saat ini join paling banyak satu account per student/tahun dan menghasilkan baris StudentReport dari account itu ([reporting_repository.go](../../internal/repository/postgres/reporting_repository.go#L24-L56), [StudentReport](../../internal/repository/postgres/reporting_repository.go#L83-L118)). Setelah multi-cycle, report harus memilih satu dari dua kontrak eksplisit: satu baris per siswa dengan agregat saldo/transaksi lintas cycle, atau satu baris per siswa-cycle dengan detail siklus. Rekomendasi: ringkasan per siswa mengagregasi nilai per tahun, sedangkan drill-down menunjukkan tiap cycle dan settlement; validasi ulang metrik Dashboard “rekening” agar jelas apakah menghitung siklus atau siswa.
- **Penentuan cycle aktif:** gunakan status `OPEN` sebagai kriteria, disertai constraint satu OPEN. Bila tak ada OPEN tetapi ada settled, UI menampilkan siklus terbaru selesai dan tindakan pembuatan siklus terpisah. Jika ada lebih dari satu kandidat karena data lama/anomali, tampilkan error administratif, jangan memilih diam-diam.
- **Status keluar:** domain `Student.Reactivate` saat ini tidak menghapus `LeavingDate`, sementara `LeaveOn` mengisi kolom itu lagi. Kolom tunggal ini bukan histori seluruh peristiwa keluar; jangan menjadikannya sumber kebenaran untuk memilih cycle atau menghitung jumlah return. Histori placement tetap tersimpan. Rekomendasi event lifecycle terpisah perlu keputusan teknis dan schema tersendiri.

## 4. Konkurensi dan urutan lock

MQ-08-REVIEW-01 menunjukkan reaktivasi sekarang mengunci student lalu year, sedangkan setor/tarik menggunakan year → account → student dan settlement menggunakan year → account lalu membaca student tanpa lock. Interleaving reaktivasi/settlement dapat membuat ACTIVE+SETTLED; menambah lock student di akhir settlement saja berisiko deadlock. Bukti serta skenario detail ada pada [laporan concurrency](MQ-08_CONCURRENCY_SETTLEMENT_REVIEW.md).

### Protokol lock yang direkomendasikan untuk dirancang

Pilih urutan total yang dapat dipatuhi semua path: **tahun ajaran → rekening yang terkait (urut ID konsisten bila banyak) → siswa**. Gunakan lock mode secukupnya berdasarkan operasi, tetapi jangan mengambil lock kategori lebih awal dengan urutan terbalik. Setelah mengambil lock, baca dan validasi ulang status yang menjadi prasyarat operasi.

| Operasi | Urutan konseptual yang perlu dipenuhi | Validasi di dalam lock |
|---|---|---|
| `Leave` | Tahun dari input → rekening terkait jika keputusan bisnis mengharuskan serialisasi terhadap account → student → history/update. | Student masih ACTIVE; transfer/status account sesuai kontrak; rentang kelas dapat ditutup. |
| Reaktivasi biasa | Tahun tujuan → account siswa/tahun jika ada → student → create placement/update. | Student masih LEFT; account jika ada belum SETTLED; kelas/tanggal/year cocok. |
| `Siswa Kembali` | Tahun → account/cycle lama yang relevan (kunci terurut) → student → placement. | Student masih LEFT; settlement lama tetap immutable; cycle baru tidak terbentuk dari transisi Admin. |
| Setor/tarik | Tahun → account → student (sudah searah secara konseptual). | Tahun/account boleh ditransaksikan; student ACTIVE saat lock student diambil. |
| Settlement | Tahun → account → student. | Student LEFT, account OPEN, tahun memenuhi status; periksa pending request dan expected balance sesudah lock yang relevan. |
| Buat siklus | Tahun → account lama yang perlu diperiksa → student → tentukan nomor/insert account baru. | Student ACTIVE, Wali masih dalam scope, tahun OPEN, siklus sebelumnya settled, tidak ada cycle OPEN, dan aturan handover-pending telah diputuskan. |

Tabel ini **rancangan**, bukan instruksi perubahan langsung. Lock mode, lock terhadap rekening yang belum ada, penguncian beberapa cycle, dan interaksi dengan year close/reopen harus diverifikasi terhadap semua path finansial lain sebelum implementasi. Operasi harus menjaga seluruh audit dan write dalam UOW. Aturan Admin read-only tidak boleh berubah.

### Test concurrency yang diperlukan

Gunakan PostgreSQL disposable, dua koneksi/transaksi nyata, dan barrier terkendali (test-only repository decorator/channel atau transaksi pengendali yang memegang row lock). Sinkronkan dengan pengamatan `pg_blocking_pids`/`pg_stat_activity` dan deadline; jangan mengandalkan `time.Sleep` untuk menentukan urutan.

Minimal jalankan interleaving:

1. Admin reaktifasi lebih dulu, lalu settlement membaca status: settlement harus mengikuti status commit.
2. Settlement sudah mengunci year/account dan memvalidasi LEFT; reaktivasi menunggu; hasil akhir harus mengikuti aturan owner yang sudah ditetapkan.
3. Reaktivasi berhadapan dengan deposit/withdrawal: tidak ada hang/ deadlock tak tertangani; transaksi biasa tidak commit pada status LEFT.
4. Reaktivasi berhadapan dengan pembuatan cycle/account baru melalui trigger eligibility: tidak ada hang/deadlock dan INSERT hanya commit bila student/year eligible.
5. Dua pembuatan cycle bersamaan: tidak ada nomor ganda, dua cycle OPEN, atau account ganda.
6. Rollback salah satu pihak tidak meninggalkan perubahan parsial pada student, class history, account, transaction, settlement, atau audit.

Assert hasil operasi, status student/account, jumlah cycles dan settlement, nomor cycle, class validity, transaksi/nominal, cash movements, audit, SQLSTATE deadlock, dan tidak adanya perubahan ledger yang tidak diminta. Sertakan test sequential untuk kasus A–F, migration upgrade/rollback aman, role/scope, dan laporan multi-cycle. Pertahankan test settlement year close/reopen yang ada.

## 5. Rancangan workflow FE

### Admin — detail Siswa

1. Di detail siswa berstatus LEFT, tampilkan histori kelas dan histori rekening per tahun/cycle (status, saldo ledger, settlement, nilai yang diserahkan, sisa yang belum direkonsiliasi).
2. Jika rekening tahun pilihan belum SETTLED, tombol **Aktifkan Kembali** membuka form tahun ajaran, kelas, dan tanggal efektif yang sama dengan validasi penempatan saat ini. Konfirmasi menjelaskan bahwa rekening OPEN yang lama akan dipakai kembali.
3. Jika rekening tahun yang sama SETTLED, tampilkan tombol **Siswa Kembali** (jangan gunakan tombol Reopen Settlement). Wizard meminta tahun, kelas, tanggal efektif, dan menampilkan rekening/settlement lama sebagai referensi. Form tidak menawarkan nominal, setoran, penarikan, konfirmasi penyerahan, edit, atau pembukaan settlement.
4. Setelah submit berhasil, tampilkan status ACTIVE dan placement baru. Jika cycle baru diperlukan, beri petunjuk bahwa Wali dengan scope harus membuatnya dari menu Tabungan; Admin tidak membuat account.
5. Jika settlement lama belum diserahkan, tampilkan peringatan yang terpisah. Apakah ini memblokir cycle baru menunggu keputusan produk.

### Wali Kelas — Tabungan

1. Daftar berdasarkan tahun/kelas dan scope assignment menampilkan siswa ACTIVE tanpa cycle OPEN, termasuk yang memiliki siklus settled sebelumnya.
2. Untuk siswa kembali dengan hanya settled cycles, tampilkan tindakan **Buat Siklus Tabungan** beserta ringkasan cycle lama, tanggal settlement, jumlah, dan status penyerahan. Tidak ada tombol untuk membatalkan settlement lama.
3. Setelah Wali membuat cycle, buka buku tabungan cycle baru yang ber saldo `Rp. 0`; cycle lama tetap tersedia untuk histori baca.
4. Tabungan aktif menyediakan setor/tarik pada cycle OPEN saja. Histori settled dibuka read-only kecuali koreksi/reopen yang sudah ada dan memperoleh otorisasi workflow terkait.
5. Jika tidak ada Wali berwenang, tahun belum OPEN, penempatan belum tersedia, cycle OPEN sudah ada, nomor cycle bentrok, atau ada hambatan yang ditetapkan kebijakan, jangan tampilkan aksi yang akan gagal. Beri pesan sebab dan tindakan role yang benar.

### Siswa

Siswa melihat cycle per tahun dengan label aktif/selesai, buku transaksi tiap cycle, settlement dan penyerahan. Siswa tidak dapat meminta perubahan ledger melalui halaman baca ini.

## 6. Rencana implementasi berurutan

1. **Persetujuan kontrak teknis:** putuskan unresolved questions di bawah dan lakukan review design untuk lock order.
2. **Migration:** rancang upgrade v14+ di disposable; backfill cycle number=1, ubah uniqueness, tambahkan invariant satu OPEN; buktikan semua FK/history tetap sama. Jangan jalankan terhadap database operasional dalam tahap desain.
3. **Domain/repository:** tambah model/query list/current/by-ID yang eksplisit, penjaga nomor cycle dan uniqueness; pertahankan account ID semua row lama.
4. **Service/concurrency:** ubah urutan lock secara terkoordinasi untuk Leave, Reactivate/Siswa Kembali, Create cycle, transaction, settlement, dan interaksi year close/reopen; validasi ulang status sesudah lock.
5. **Tests:** domain/unit, PostgreSQL migration/integration, concurrency deterministic, authorization/scope, audit, rollback, dan reporting; semua DB test pada disposable.
6. **HTTP/templates:** implementasi Admin return flow, Wali create-cycle flow, riwayat multi-cycle, error-state yang bisa dipahami, CSRF dan authorization server-side.
7. **Reporting:** putuskan agregasi per student versus per cycle, lalu verifikasi jumlah, saldo, settlement, pagination, dan scope tanpa fan-out duplikasi.
8. **Dokumentasi/rollout:** update kontrak hanya setelah keputusan resmi, TESTING/HANDOVER, prosedur upgrade/rollback, snapshot data, dan langkah observasi deployment.

Setiap tahap harus mendapat review supervisor sesuai workflow branch `review`; tidak ada merge atau perubahan database operasional dari rancangan ini.

## 7. Keputusan teknis yang memerlukan supervisi

1. Setujui atau ubah pilihan row-per-cycle `savings_accounts` + `cycle_number`; apakah partial unique satu OPEN per siswa/tahun sesuai reopen settlement yang ada?
2. Jika penyerahan settlement lama masih pending, bolehkah Wali membuat cycle baru? Rekomendasi awal: tampilkan status dan minta keputusan; jangan diam-diam memblokir atau mengizinkan tanpa aturan.
3. Apakah riwayat leave/return memerlukan tabel lifecycle-event terpisah atau cukup history placement ditambah audit?
4. Untuk laporan, pilih agregat per siswa per tahun dan drill-down cycle sebagai default, atau satu baris per cycle.
5. Apakah akun historis `OPEN` di tahun tertutup dapat dipakai oleh siswa yang kembali? Kontrak yang ada membatasi pembuatan account baru pada tahun OPEN, tetapi belum menjawab penggunaan siklus lama.
6. Tetapkan lock modes yang konkret dan cakupan perombakan semua jalur finansial; pastikan cycle-number allocation dan year close/reopen tidak membentuk lock cycle.
7. Tetapkan pesan/validation untuk tanggal kembali yang overlap, backdate, placement pada year CLOSED, siswa tidak dalam status LEFT, dan scope Wali yang berubah.

## 8. Risiko dan batas verifikasi

- Laporan dibuat dari pembacaan source/dokumentasi pada baseline; tidak ada kode, migration, database, maupun test yang diubah/dijalankan.
- Migration belum dibuat; ukuran tabel, lock duration, data produksi, runner deployment, dan prosedur rollback operasional belum diverifikasi.
- Belum ada keputusan khusus mengenai settlement dengan handover pending, siklus lama OPEN pada tahun berbeda, agregasi reporting, atau penyimpanan lifecycle event.
- Race state `ACTIVE + SETTLED` dan deadlock lock-order masih risiko dari MQ-08 review, bukan bug yang direproduksi.
- Fitur multi-cycle, endpoint, UI, dan test belum ada pada baseline; dokumen ini tidak menyatakan implementasi selesai.

## 9. Pembaruan setelah MQ-08-CONTRACT-02

- Pemilik menyetujui pemisahan proses pindah dari `students.status`: student tetap belum `LEFT` saat kewajiban diselesaikan. Settlement pindah yang baru harus terkait proses aktif; status LEFT ditetapkan setelah pemeriksaan finansial dan penutupan placement.
- Tabel state dan UI lama dalam rancangan ini perlu ditafsirkan bersama amendment baru. `PENDING_LEAVE` belum disetujui sebagai enum student. Model tabel proses, rekening lintas tahun, serta tanggal backdate masih menunggu keputusan/desain.
- Amendment baru tidak mengunci pilihan row-per-cycle, saldo awal cycle, agregasi report, atau lifecycle-event. Jangan menganggap rekomendasi tersebut sudah menjadi aturan produk.
- Perubahan kontrak belum diimplementasikan. Semua gap/race di bagian ini tetap perlu review dan tes, dengan lock protocol yang mencakup trigger PostgreSQL.

## 10. Pembaruan setelah MQ-08-CONTRACT-03

- Pemilik menyetujui pembatalan proses pindah sebelum `LEFT`, termasuk setelah settlement bila penyerahan positif yang diwajibkan sudah selesai. Student tetap `ACTIVE`; placement tidak berubah; settlement dan handover tidak dibatalkan; status settled tetap final. Ini bukan reaktivasi/Siswa Kembali.
- Jika tabungan dilanjutkan setelah pembatalan, siklus baru dibuat oleh Wali yang berwenang, bukan Admin. Multi-siklus belum diimplementasikan; schema baseline masih unik `(student_id, academic_year_id)`. Rekomendasi invariant tetap maksimal satu siklus `OPEN` per siswa/tahun.
- Pembatalan ketika handover positif masih tertunda belum diputuskan. Serialisasi terhadap settlement, handover, serta pembuatan/reopen siklus perlu dirancang dan diuji; analisis lama tidak membuktikan race tertutup.
- Keputusan saldo awal `Rp. 0` tetap rekomendasi, bukan keputusan MQ-08-CONTRACT-03. Tidak ada migration, source, atau test yang diubah untuk catatan ini.
