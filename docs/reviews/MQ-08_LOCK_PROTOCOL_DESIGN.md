# MQ-08-LOCK-PLAN-01 — Protokol Lock Proses Pindah

**Status:** rancangan untuk review; belum diimplementasikan
**Baseline audit:** `d4bfcb74d17b137b24a88444017801c4c9e8323b` (`review`)
**Tanggal:** 2026-10-27

Dokumen ini menetapkan satu arah protokol lock untuk implementasi proses pindah dua tahap. Temuan menggambarkan source dan migration pada baseline di atas; ini bukan hasil pengujian konkurensi runtime. Rujukan kontrak: [amendment proses pindah](../contracts/student-transfer-financial-clearance-amendment.md), [kontrak siswa kembali dan siklus tabungan](../contracts/student-return-and-savings-cycle-contract.md), [review konkurensi settlement](MQ-08_CONCURRENCY_SETTLEMENT_REVIEW.md), dan [rancangan siswa kembali](MQ-08_STUDENT_RETURN_DESIGN.md).

## Ringkasan keputusan teknis

Gunakan urutan global berikut untuk setiap transaksi yang menyentuh siklus hidup siswa/rekening:

1. Semua row tahun ajaran dalam cakupan, urut berdasarkan ID menaik: `FOR SHARE` untuk operasi keuangan biasa; `FOR UPDATE` untuk perubahan lifecycle, settlement yang mengubah kesiapan, pembatalan/finalisasi proses, tutup/buka tahun.
2. Semua rekening dalam cakupan, urut berdasarkan ID menaik: `FOR UPDATE` bila operasi memeriksa atau mengubah ledger/status/kesiapan rekening.
3. Row siswa: `FOR UPDATE` bila memeriksa atau mengubah status, placement, atau proses pindah.
4. Row proses pindah dan row workflow lain, urut berdasarkan ID: `FOR UPDATE` terakhir.

Transaksi hanya mengambil awalan/subset urutan tersebut. Tidak boleh memegang row proses/permintaan lalu meminta lock tahun, rekening, atau siswa. Setelah lock diperoleh, baca ulang dan validasi ulang seluruh status serta himpunan rekening. Baris yang belum ada tidak dapat menjadi mutex: row siswa yang selalu ada menjadi gerbang serialisasi untuk start/cancel/finalize; partial unique index tetap menjadi pertahanan terakhir.

**Batas bisnis yang belum ditetapkan:** cakupan rekening kewajiban pindah lintas tahun, apakah pembuatan rekening baru diizinkan selama proses pindah aktif, tanggal efektif/backdate, serta tindakan terhadap rekening settled dengan kewajiban penyerahan yang belum selesai. Protokol tidak menetapkan jawabannya.

## Inventaris lock aktual

| Alur dan lokasi | Batas transaksi | Urutan lock eksplisit | Lock implisit / interaksi | Catatan |
|---|---|---|---|---|
| `StudentService.Leave` — `internal/service/student_service.go` (`Leave`) | UOW tunggal | Siswa `FOR UPDATE`; kemudian update riwayat placement | Trigger perubahan/penutupan riwayat mengunci tahun `FOR SHARE`; UPDATE mengambil row-write lock | Urutan efektif siswa → tahun. Bertentangan dengan pola tahun → siswa yang dibutuhkan finalisasi. |
| `StudentService.Reactivate` — `internal/service/student_service.go` (`Reactivate`) | UOW tunggal | Siswa `FOR UPDATE` → tahun placement `FOR UPDATE` | Insert riwayat memicu lock tahun `FOR SHARE` (compatible dengan lock tahun yang telah dipegang) | Membentuk inversi nyata dengan operasi tahun-dahulu yang kemudian menunggu siswa. |
| Setor/tarik — `internal/service/transaction_service.go` (`Deposit`, `Withdrawal`, `lockOpenTransactionAccount`, `requireActiveSavingsStudent`) | UOW tunggal per mutasi | Tahun `FOR SHARE` → rekening `FOR UPDATE` → siswa `FOR UPDATE` | INSERT transaksi/FK melindungi row induk dengan key-share; trigger pembatalan transaksi dapat mengambil rekening `FOR SHARE` | Lock akun menyerialkan ledger, namun urutan lama berlawanan dengan Reactivate. |
| `SavingsAccountService.Create` — `internal/service/savings_account_service.go` | Insert dan audit dalam UOW; precheck siswa/tahun dilakukan sebelum UOW | Service tidak mengambil row lock eksplisit. Trigger eligibility: tahun `FOR SHARE` → siswa `FOR SHARE` | INSERT dan FK; unique `(student_id, academic_year_id)` menahan duplikasi akun | Precheck service bukan sinkronisasi. Trigger adalah pelindung eligibility database yang nyata. |
| `SettleStudentLeaving` — `internal/service/settlement_service_settle.go` | UOW tunggal | Tahun `FOR SHARE` → rekening `FOR UPDATE`; siswa dibaca tanpa row lock | Update rekening, settlement/audit dan FK | Kode lama mengharuskan siswa `LEFT`; kontrak MQ-08-02 menggantinya dengan proses sebelum finalisasi. Implementasi berikutnya perlu lock siswa setelah akun. |
| `SettleYearEnd` — file yang sama | UOW tunggal | Tahun `FOR SHARE` → rekening `FOR UPDATE` | Update rekening/insert settlement dan audit | Tidak mengunci siswa. Tetap urutan tahun → akun. |
| `ConfirmHandover` — `internal/service/settlement_service_handover.go` | UOW tunggal | Tahun `FOR SHARE` → rekening `FOR UPDATE` | Insert handover/return dan audit; pembacaan siswa akhir biasa | Lock rekening menserialkan konfirmasi pada rekening yang sama; tidak ada lock settlement eksplisit. |
| `LeavingTransactionRequestService.Create` — `internal/service/leaving_transaction_service.go` | UOW tunggal | Tahun `FOR SHARE` → rekening `FOR UPDATE` → siswa `FOR UPDATE` | Insert request/FK dan audit | Selaras dengan urutan tahun → akun → siswa; kontrak status lama perlu diselaraskan pada implementasi proses baru. |
| `LeavingTransactionRequestService.Execute` — file yang sama | UOW tunggal | Tahun `FOR SHARE` → rekening `FOR UPDATE` → siswa `FOR UPDATE` → late request `FOR UPDATE` | Insert transaksi, update request, audit/FK | Urutan sudah mengikuti protokol; pertahankan row request di posisi terakhir. |
| `LeavingTransactionRequestService.Decide` — file yang sama | UOW tunggal | Late request `FOR UPDATE` | Update status/audit; tidak meminta lock tahun/akun/siswa sesudahnya | Request-first saat ini belum membentuk siklus sendiri karena tidak meminta resource berperingkat lebih awal. Bila perlu memvalidasi lifecycle, ubah menjadi tahun → akun → siswa → request. |
| Start/close tahun — `internal/service/academic_year_service.go` (`StartClosing`, `Close`) | UOW tunggal | Tahun `FOR UPDATE` | Pemeriksaan blocker; update status memicu trigger close yang memeriksa akun | Tahun menjadi gerbang yang kompatibel dengan writer tahun `FOR SHARE`. Hindari mengunci request blocker setelah membaca tahun. |
| Reopen tahun — `internal/service/academic_year_service.go` (`Reopen`) | UOW tunggal | Saat ini baca biasa, lalu UPDATE tahun (row-write lock implisit) | Validasi dilakukan sebelum lock update | Tidak membentuk siklus terbukti karena tidak meminta resource lain setelah UPDATE, tetapi validasi dapat stale. Ambil tahun `FOR UPDATE`, lalu baca/validasi ulang. |
| Reopen settlement — `internal/service/settlement_service_reopen.go` | UOW tunggal per aksi | Request: tahun `FOR UPDATE` → akun `FOR UPDATE`; decide: tahun `FOR UPDATE` → reopen request `FOR UPDATE`; reopen: tahun → akun → reopen request | Update akun memanggil trigger yang meminta tahun `FOR SHARE` (transaksi sendiri sudah memegang lock tahun) | Pertahankan tahun dahulu; jangan memperoleh request sebelum tahun/akun. |
| Pembuatan tahun — `AcademicYearService.Create` dan trigger migration 000011 | UOW | Daftar tahun `ORDER BY id FOR UPDATE` | Trigger mengunci semua tahun existing dalam urutan ID dan memeriksa successor/reopen | Ini pola urut global untuk banyak tahun; gunakan urutan deterministik, tanpa mengunci tahun yang tidak relevan bila tidak perlu. |
| `StudentLeavingRequestRepository` — `internal/repository/postgres/student_leaving_request_repository.go` | Mengikuti UOW caller | `GetByIDForUpdate`: row proses ada → `FOR UPDATE`; `GetInProgressByStudentForUpdate`: hanya row aktif yang ditemukan → `FOR UPDATE` | Create bergantung partial unique index untuk satu `IN_PROGRESS` per siswa | SELECT atas row yang tidak ditemukan tidak mengunci gap atau mencegah INSERT. Kunci siswa lebih dahulu. |
| Permintaan transfer kelas — `internal/service/class_transfer_service.go` | UOW per aksi | Request: siswa `FOR UPDATE`, lalu insert transfer; approve/reject mengunci transfer request | Trigger transfer mengambil tahun `FOR SHARE` | Request saat ini siswa → tahun lewat trigger. Reorder request menjadi tahun → siswa → insert. Approve/reject jangan setelah lock request meminta tahun/siswa. |

### Trigger dan lock implisit

| Mekanisme | Lock/efek | Dampak desain |
|---|---|---|
| Trigger eligibility rekening, migration 000006 | Tahun `FOR SHARE`, lalu siswa `FOR SHARE`; validasi status | Finalizer yang memegang tahun `FOR UPDATE` membuat pembuatan akun menunggu. Trigger harus mempertahankan tahun dahulu; kebijakan akun saat proses aktif harus eksplisit. |
| Trigger batas riwayat placement, migration 000006/000007 | Tahun `FOR SHARE` saat insert/update riwayat | Leave/finalize harus memperoleh tahun sebelum siswa agar tidak membuat inversi dengan penulis tahun-dahulu. |
| Trigger buka kembali rekening setelah tahun tutup, migration 000006 | Tahun `FOR SHARE` | Lock tahun lifecycle menserialkan reopen rekening. |
| Trigger transfer kelas, migration 000006 | Tahun `FOR SHARE` pada insert/perubahan status | Request transfer harus memperoleh tahun sebelum siswa; finalizer memeriksa blocker tanpa mengunci row transfer setelah lock tahun. |
| Trigger close tahun | Memeriksa rekening pada transisi `CLOSED` | Pertahankan tahun `FOR UPDATE` sebagai gerbang; finalizer/settlement harus selesai sebelum close lolos. |
| FK dan DML | FK insert/update dapat mengambil `KEY SHARE` pada parent; UPDATE mengambil row-write lock; unique check menunggu transaksi pesaing | Ini bukan pengganti urutan lock layanan atau validasi ulang. Indeks unique adalah backstop untuk duplikasi, bukan serialisasi seluruh proses. |

## Inversi terkonfirmasi dan risiko hipotetis

### Inversi yang terkonfirmasi dari urutan source

Deposit/withdrawal memegang tahun `FOR SHARE` dan akun `FOR UPDATE`, lalu meminta siswa `FOR UPDATE`. Reactivate memegang siswa `FOR UPDATE`, lalu meminta tahun `FOR UPDATE`. Interleaving deposit (memegang tahun, menunggu siswa) dan Reactivate (memegang siswa, menunggu tahun) membentuk siklus tunggu yang dapat dipilih PostgreSQL sebagai deadlock victim. Ini **terkonfirmasi sebagai inversi urutan lock dalam kode**, bukan deadlock yang direproduksi pada DB dalam audit ini.

Leave dan class-transfer Request juga memegang siswa sebelum trigger meminta tahun `FOR SHARE`; keduanya dapat berinversi dengan finalisasi/tutup yang mengambil tahun `FOR UPDATE` lalu menunggu siswa. SavingsAccount.Create trigger sendiri sudah mengambil tahun sebelum siswa, tetapi service precheck di luar UOW bukan perlindungan.

### Risiko belum dibuktikan runtime

- Start/finalize baru versus akun baru, pembatalan, Siswa Kembali, atau transfer kelas dapat mengalami wait/deadlock jika implementasi mengambil row request/student/account dalam urutan berbeda dari protokol.
- Dua Start bergantung pada unique index serta gate siswa; SELECT FOR UPDATE atas proses yang belum ada tidak cukup.
- Dua ConfirmHandover saat ini seharusnya diserialkan oleh lock rekening; perlu tes untuk memastikan hasil/idempotensi sesuai kontrak.
- Reopen tahun memvalidasi sebelum UPDATE lock; stale decision adalah risiko konsistensi, bukan deadlock yang terkonfirmasi.
- Trigger, FK, unique constraint, dan operasi transfer dapat menambah lock implisit; hasil konkurensi belum diuji pada baseline ini.

## Protokol yang direkomendasikan

### Akuisisi dan validasi

1. Tentukan cakupan tahun dari identitas objek awal yang dibaca biasa; jangan menganggap pembacaan awal sebagai validasi final.
2. Mulai UOW, kunci semua tahun dalam cakupan dalam urutan ID menaik. Gunakan `FOR SHARE` bagi transaksi ledger biasa dan `FOR UPDATE` bagi settlement/lifecycle yang perlu mengecualikan perubahan lain.
3. Setelah lock tahun didapat, enumerasi ulang rekening dalam cakupan. Kunci semua rekening dalam urutan ID menaik dengan `FOR UPDATE` bila menilai/mengubah rekening. Bandingkan dengan set yang akan diproses; jika berubah, batalkan dan ulangi dengan cakupan yang benar. Jangan mengandalkan lock rekening untuk melindungi rekening yang belum ada.
4. Kunci siswa `FOR UPDATE`; validasi ulang `status`, placement, izin, dan tanggal sesuai kontrak.
5. Baca/lock proses pindah dan workflow request terakhir. Untuk Start, setelah lock siswa periksa ulang tidak ada `IN_PROGRESS`, lalu insert. Unique partial index tetap menangani caller yang tidak mematuhi gate.
6. Validasi ulang ledger, settlement, handover, late request, transfer blocker, reopen blocker, status tahun, serta kesiapan. Baru lakukan perubahan dan audit dalam UOW yang sama.
7. Semua perubahan gagal harus rollback sebagai satu unit. Jangan menulis audit sukses sebelum seluruh syarat lulus.

Untuk Start/cancel/finalize yang mengubah lifecycle siswa gunakan tahun `FOR UPDATE` agar menyerialkan trigger tahun `FOR SHARE`, transaksi biasa, dan pembuatan rekening. Operasi biasa/settlement yang telah ada dapat memakai `FOR SHARE` tahun lalu `FOR UPDATE` rekening; semua operasi yang menyentuh siswa mengambil lock siswa setelah rekening. `FOR UPDATE` tahun yang lebih kuat boleh dipegang transaksi lifecycle sampai commit.

### Himpunan rekening dan row yang belum ada

`FOR UPDATE` pada row yang tidak ada tidak membuat gap lock di PostgreSQL. Gerbang stabil adalah row siswa yang sudah ada, tetapi agar pembuatan rekening tidak menyelinap saat finalisasi, finalizer juga mengunci tahun yang relevan `FOR UPDATE`. Trigger create rekening meminta tahun `FOR SHARE`; karena itu insert yang mulai belakangan menunggu. Setelah finalizer commit sebagai LEFT, trigger/service harus menolak insert berdasarkan status yang telah berubah. Jika insert lebih dulu mendapat share lock, finalizer menunggu, lalu menghitung rekening baru setelah lock didapat.

Jika proses belum ada, serialisasi Start memakai row siswa. Jika rekening belum ada, jangan menciptakan rekening sebagai syarat pindah; tetap kunci tahun dan siswa serta evaluasi tanpa rekening. Partial unique index satu proses aktif merupakan pertahanan duplikasi, bukan pengganti lock siswa dan validasi service.

### Cakupan banyak tahun/rekening

Untuk beberapa tahun, kunci seluruh row tahun yang disetujui terlebih dahulu, urut ID menaik; baru kemudian seluruh rekening urut ID menaik; kemudian siswa dan request. Jangan mengunci tahun kedua setelah sudah memegang akun/tahun pertama. Cakupan lintas tahun **belum diputuskan**. Skema `student_leaving_requests` mengikat satu `academic_year_id`, sedangkan kontrak menyebut seluruh kewajiban yang relevan. Sebelum coding, owner/supervisor harus menetapkan cakupan. Jika lintas tahun dipilih, kumpulkan set tahun kandidat, lock semuanya secara terurut, enumerasi ulang akun, lalu abort/retry bila set berubah.

## Dampak dan patch implementasi berurutan

1. Kunci cakupan bisnis dan perilaku akun/transaksi selama proses aktif; jangan mengkode asumsi lintas tahun atau backdate.
2. Implementasikan Start/Cancel/Finalize dengan UOW, tahun → rekening → siswa → process row, gate siswa untuk row proses yang belum ada, validasi ulang dan audit atomik.
3. Reorder `StudentService.Leave` menjadi mekanisme finalisasi yang sama; reorder `Reactivate` menjadi tahun → akun yang relevan → siswa, lalu baca ulang placement/status.
4. Reorder class-transfer Request menjadi tahun → siswa sebelum insert; pastikan approve/reject tidak meminta lock tahun setelah transfer request.
5. Update `SettleStudentLeaving` agar memeriksa proses aktif dan lock siswa setelah akun, dengan status prasyarat sesuai kontrak baru. Pertahankan settle/handover sebagai mutasi wali; Admin tetap tidak memperoleh mutasi keuangan.
6. Selaraskan `SavingsAccountService.Create` precheck ke UOW atau pertahankan trigger sebagai perlindungan akhir; tambahkan validasi proses hanya setelah keputusan akun selama proses aktif disetujui.
7. Selaraskan `LeavingTransactionRequestService` status/tanggal proses; bila `Decide` membutuhkan validasi lintas resource, ubah urutan agar tidak memegang request sebelum tahun/akun/siswa.
8. Ubah `AcademicYearService.Reopen` agar lock tahun `FOR UPDATE` diperoleh sebelum validasi. Pertahankan StartClosing/Close dan settlement reopen dengan tahun dahulu.
9. Tambahkan integration tests konkurensi sebelum menyatakan tahap lifecycle selesai; jangan mengubah trigger tanpa membuktikan kebutuhan, dan jangan menambahkan lock trigger process-row sebelum siswa.

Perubahan terkecil yang tidak dapat dihindari adalah membalik urutan Reactivate, Leave/finalize, dan class-transfer Request yang kini student-first; menambah lock siswa sesudah akun pada settlement pindah; membuat precheck rekening/create ikut transaksi/protokol; serta mengunci tahun sebelum validasi Reopen. Perubahan trigger hanya diperlukan bila kebijakan akun ketika proses aktif perlu ditegakkan di database.

## Matriks uji konkurensi yang diwajibkan

Semua skenario dijalankan kelak pada PostgreSQL 18 disposable, dua koneksi transaksi nyata, port loopback khusus, role/database task-owned, fixture yang diketahui, batas waktu per langkah, dan cleanup terverifikasi. Gunakan barrier eksplisit (misalnya kanal/test advisory barrier di harness) untuk menghentikan transaksi A setelah lock tertentu; amati `pg_blocking_pids()`/`pg_stat_activity` untuk membuktikan transaksi B memang menunggu lock yang dimaksud. Setiap test harus mengakhiri transaksi, memeriksa state final dan audit, serta memastikan tidak ada sesi tertinggal. Audit ini tidak menjalankan database.

| Kasus | Kondisi awal dan interleaving deterministik | Hasil yang boleh | Invariant yang diperiksa |
|---|---|---|---|
| A. Start vs Reactivate | ACTIVE; A Start lock tahun, B Reactivate mencoba tahun/siswa, barrier sebelum commit | Salah satu memperoleh urutan; setelah lock, loser membaca ulang dan menolak/berhasil hanya sesuai status aktual; tanpa deadlock | Satu status siswa, satu placement valid, maksimal satu proses aktif, tidak ada audit/placement parsial |
| B. Start vs CreateAccount | ACTIVE, akun belum ada; jalankan Start dan trigger INSERT dengan barrier pada year/student locks | Serialisasi terbukti. Keputusan allow/deny Create selama proses aktif harus mengikuti kontrak yang belum ditetapkan; jangan klaim hasil produk | Tidak ada akun yang lolos dari himpunan readiness final; tidak ada akun wajib paksa hanya untuk pindah |
| C. Settlement vs deposit/withdrawal | Akun OPEN; pause setelah transaksi mendapat tahun share/akun update; transaksi lain bersaing | Mutasi yang menang lebih dulu tercermin dalam saldo; mutasi setelah SETTLED ditolak; tanpa saldo/settlement ganda | Ledger kronologis, saldo = transaksi aktif, satu settlement yang sah, audit konsisten |
| D. Settlement vs Cancel | Proses IN_PROGRESS; pause settlement setelah lock; cancel meminta tahun update | Salah satu menang dan yang lain membaca ulang. Pembatalan hanya menurut kontrak; jangan memperluas izin saat handover wajib masih tertunda | Tidak ada settlement dibatalkan otomatis, uang/handover historis tetap, proses tidak LEFT tanpa finalisasi |
| E. Finalize vs CreateAccount | Siswa aktif dan rekening belum ada; finalizer memperoleh tahun update, create trigger meminta share; ulangi dengan create lebih dulu | Create yang lebih dulu selesai harus masuk enumerasi final; jika finalizer lebih dulu, create menunggu dan setelah commit LEFT ditolak | Tidak ada rekening yang terlewat dari clearance; siswa tanpa rekening tetap dapat diproses |
| F. Finalize vs transaksi | Rekening OPEN; barrier transaksi setelah year share dan finalizer meminta year update, lalu skenario dibalik | Transaksi yang commit lebih dulu dihitung; transaksi yang menunggu setelah finalisasi memvalidasi ulang dan ditolak | Tidak ada transaksi pasca-LEFT; ledger, status rekening, audit atomik |
| G. Siswa Kembali vs settlement historis | Ada settlement/akun historis; jalankan operasi lifecycle kembali berhadapan dengan settlement | Urutan tahun→akun→siswa menyerialkan; hasil mengikuti kontrak return/multi-cycle yang disetujui. Jangan menganggap schema multi-cycle sudah tersedia | Settlement lama immutable; identitas siswa/placement tidak duplikat; tidak ada siklus OPEN ganda |
| H. Close/reopen tahun vs operasi | Tahun OPEN/CLOSING; barrier antara year lock dan account/student lock untuk setor, settle, finalize, create, reopen | Lifecycle year lock membuat operasi menunggu; setelah lock, validasi status tahun terbaru menentukan hasil | Status tahun, settlement/handover, akun, audit konsisten; tidak ada deadlock atau update parsial |
| I. Dua Start | Tidak ada row IN_PROGRESS; dua transaksi start berhenti setelah mencoba gate siswa | Satu proses aktif dibuat; transaksi kedua melihat proses/conflict; unique index menjadi backstop | Tepat satu IN_PROGRESS; tidak ada process/audit duplikat |
| J. Row proses/rekening absen | Siswa/tahun ada, request dan/atau akun tidak ada; satu transaksi finalisasi/start, satu mencoba insert | Pembuktian bahwa row-missing `FOR UPDATE` tidak memblokir; year/student gate dan trigger/unique constraint menyediakan serialisasi | Tidak ada phantom account/process yang dilewatkan; nol rekening diperbolehkan; rollback bersih |

Setiap kasus harus mengamati deadlock (`40P01`), lock timeout, blocking PID, status transaksi akhir, nilai saldo/ledger, row settlement/handover/process/placement, serta audit. Hasil yang hanya menunjukkan “tidak error” tanpa membuktikan blocking/interleaving bukan bukti concurrency.

## Keputusan bisnis yang masih terbuka

- Apakah kewajiban pindah mencakup rekening tahun aktif saja atau semua tahun siswa? Dokumen proses saat ini memiliki satu `academic_year_id`.
- Apakah Wali boleh membuat rekening baru selama proses pindah aktif? Apakah transaksi reguler tetap diperbolehkan sampai settlement dimulai?
- Tanggal peristiwa pindah versus tanggal finalisasi administratif dan kebijakan backdate.
- Pembatalan setelah settlement disetujui hanya bila settlement/handover telah tuntas; tidak ada izin baru untuk pembatalan saat penyerahan positif tertunda.
- Siklus tabungan baru sesudah pembatalan/Siswa Kembali belum didukung skema multi-cycle; saldo awal siklus baru masih rancangan, bukan keputusan.
- Penanganan rekening tahun tertutup, settlement historis, atau reopen saat clearance pindah.

## Batas verifikasi

Audit pada HEAD yang tercantum hanya membaca source, migration, kontrak, dan tes terkait. Tidak ada kode/migration/test yang diubah, database diakses, atau tes dijalankan. Inversi lock yang dilaporkan diturunkan dari urutan kode; deadlock belum direproduksi. Matriks di atas adalah rencana tes untuk tahap implementasi berikutnya, bukan bukti lulus.
