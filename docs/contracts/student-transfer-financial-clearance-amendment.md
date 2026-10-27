# Amendment — Penyelesaian Keuangan sebelum Pengesahan Pindah Sekolah

**Status aturan bisnis:** disetujui pemilik aplikasi untuk MQ-08-CONTRACT-02.
**Status implementasi:** belum diimplementasikan.
**Cakupan:** urutan proses pindah sekolah, penyelesaian rekening, konfirmasi penyerahan, dan finalisasi status siswa.

Amendment ini menggantikan urutan `LEFT`-sebelum-settlement pada [kontrak FASE 19](fase-19-student-http-contract.md), [amendment alur settlement pindah](student-leaving-settlement-ui-amendment.md), [amendment transaksi terlambat](late-transaction-request-amendment.md), dan ringkasan aturan lama di [BUSINESS_RULES.md](../BUSINESS_RULES.md). Dokumen-dokumen tersebut tetap menjadi bukti kontrak dan implementasi pada masanya; perubahan ini tidak menyatakan bahwa kode lama telah memenuhi aturan baru. Laporan audit historis tidak ditulis ulang.

## 1. Keputusan pemilik yang sudah disetujui

1. Siswa hanya boleh berubah menjadi `LEFT` setelah seluruh kewajiban tabungan relevan selesai, rekening yang perlu diselesaikan sudah `SETTLED`, dan semua penyerahan uang yang diwajibkan telah dikonfirmasi.
2. Admin memulai proses pindah; siswa tetap belum `LEFT` selama Wali menyelesaikan kewajiban tabungan.
3. Wali yang memiliki scope menyelesaikan rekening. Saldo positif harus diselesaikan dan penyerahannya dikonfirmasi; saldo nol harus diselesaikan tanpa penyerahan fisik.
4. Backend memeriksa kembali semua kewajiban sebelum Admin mengesahkan pindah, mengubah status menjadi `LEFT`, dan menutup riwayat penempatan kelas.
5. Siswa tanpa rekening tidak dipaksa membuat rekening hanya untuk pindah.
6. Riwayat kelas, transaksi, settlement, penyerahan, dan audit yang ada tidak dihapus atau ditulis ulang. Admin tidak memperoleh hak mutasi keuangan.
7. Memulai proses dan mengesahkan pindah adalah dua tindakan berbeda.
8. `PENDING_LEAVE` tidak disetujui sebagai nilai baru `students.status`. Proses pindah harus dimodelkan terpisah kecuali pemilik menyetujui perubahan status siswa kemudian.

## 2. Audit baseline: fakta implementasi yang belum sesuai

Baseline `e211f21ee7a86179a7c60910d0e94724b047e7d1` memiliki perilaku berikut:

- [`StudentService.Leave`](../../internal/service/student_service.go#L809-L883) adalah tindakan Admin satu tahap: di dalam UOW ia mengunci siswa, memanggil `Student.LeaveOn`, menutup riwayat kelas pada tanggal yang sama, menyimpan siswa, lalu commit. Method ini tidak memeriksa settlement atau handover sebelum status menjadi `LEFT`.
- [`Student.LeaveOn`](../../internal/domain/student.go#L64-L80) mengubah status ke `LEFT` dan menyimpan tanggal yang diberikan sebagai `LeavingDate`.
- [`SettlementService.PreviewStudentLeaving`](../../internal/service/settlement_service_read.go#L128-L173) dan [`SettleStudentLeaving`](../../internal/service/settlement_service_settle.go#L158-L284) mensyaratkan siswa sudah `LEFT`; settlement menolak rekening selain `OPEN`, tahun `CLOSED`, dan permintaan transaksi terlambat yang masih menghambat.
- [`ConfirmHandover`](../../internal/service/settlement_service_handover.go#L12-L98) hanya mengizinkan Wali dalam scope menyelesaikan penyerahan settlement positif. Kontrak FASE 23 menetapkan saldo nol tidak memerlukan handover.
- [`LeavingTransactionRequestService`](../../internal/service/leaving_transaction_service.go#L25-L75) serta jalur eksekusinya ([Execute](../../internal/service/leaving_transaction_service.go#L176-L250)) mengaitkan transaksi terlambat dengan siswa `LEFT` dan `students.leaving_date`; tanggal transaksi harus sebelum tanggal tersebut. `PENDING`/`APPROVED` menghambat settlement.
- [`TransactionService`](../../internal/service/transaction_service.go#L61-L229) mengizinkan setoran/penarikan biasa hanya untuk siswa `ACTIVE` pada titik tulis yang diserialisasi. [`SavingsAccountService.Create`](../../internal/service/savings_account_service.go#L14-L105) juga mensyaratkan siswa aktif, tahun terbuka, dan scope Wali.
- Migration 13 menambahkan `students.leaving_date` dan tabel permintaan transaksi terlambat ([up](../../migrations/000013_late_transaction_requests.up.sql#L1-L36)). Trigger eligibility pembuatan rekening mengambil lock tahun lalu lock siswa dan memeriksa tahun `OPEN` serta siswa `ACTIVE` ([migration 000006](../../migrations/000006_fase_16_8_contract_enforcement.up.sql#L150-L189)). Tidak ada status proses pindah terpisah di baseline.
- Riwayat kelas memakai rentang tanggal dengan batas akhir eksklusif ([schema 000001](../../migrations/000001_init_schema.up.sql#L64-L81)); service `Leave` saat ini menutup riwayat dengan tanggal pindah yang dipasok Admin.

Karena itu baseline tidak sesuai dengan keputusan baru. Ini temuan kontrak/kode; bukan bukti bahwa data produksi sudah bermasalah.

## 3. Model proses dan state machine

### Rekomendasi teknis

Gunakan satu entitas proses pindah terpisah dari `students.status`, misalnya tabel `student_leaving_requests` dengan tepat satu proses aktif per siswa. Jangan menambahkan `PENDING_LEAVE` ke enum status siswa. Simpan tanggal kejadian pindah (`effective_leaving_date`) saat Admin memulai proses, student ID, tahun/placement rujukan yang relevan, pembuat/waktu, serta metadata finalisasi atau pembatalan. Relasi settlement dan permintaan transaksi terlambat ke proses pindah perlu eksplisit dan nullable untuk kompatibilitas histori lama.

State yang ditampilkan dapat diturunkan dari data proses dan kewajiban, bukan disimpan sebagai banyak status yang mudah basi:

| Keadaan tampilan | Sumber kebenaran yang disarankan | Tindakan berikutnya |
|---|---|---|
| Belum mengajukan pindah | Tidak ada proses aktif/riwayat final yang relevan | Admin memulai proses |
| Proses pindah berlangsung / menunggu rekening | Proses berstatus `IN_PROGRESS`; masih ada rekening relevan `OPEN` atau permintaan transaksi terlambat `PENDING`/`APPROVED` | Wali menyelesaikan permintaan lalu settlement setiap rekening yang perlu diselesaikan |
| Menunggu konfirmasi penyerahan | Ada settlement positif dengan nilai bersih yang belum direkonsiliasi | Wali mengonfirmasi penyerahan melalui alur settlement yang berlaku |
| Siap disahkan | Pemeriksaan server saat ini menemukan seluruh kewajiban lolos | Admin mengesahkan pindah |
| Pindah disahkan | Proses `FINALIZED`, siswa `LEFT`, tanggal/riwayat kelas telah ditutup | Baca histori |
| Proses dibatalkan | Proses `CANCELLED`, siswa tidak berubah menjadi `LEFT` | Tampilkan histori proses dan dampak settlement yang sudah terjadi |

Persistensi minimum yang direkomendasikan adalah status proses `IN_PROGRESS`, `FINALIZED`, `CANCELLED`; status “menunggu rekening”, “menunggu penyerahan”, dan “siap” sebaiknya dihitung. `READY` tidak boleh menjadi flag otoritatif yang stale. Tabel permintaan pindah yang berdiri sendiri cukup untuk lifecycle administratif; tidak perlu tabel lifecycle-event generik pada tahap ini. Kebutuhan event audit khusus tetap harus ditinjau saat desain implementasi.

### Alur FE yang direkomendasikan

1. **Admin — detail siswa:** tindakan **Mulai Proses Pindah** meminta tanggal kejadian pindah dan alasan/referensi administratif yang ditetapkan implementasi. Setelah disimpan, status siswa tetap aktif; halaman memperlihatkan proses dan daftar kewajiban per tahun/rekening.
2. **Wali — Tabungan:** siswa dalam proses pindah diberi penanda tugas. Wali membuka setiap rekening dalam scope, menyelesaikan permintaan transaksi terlambat yang sah, memeriksa buku dan preview, lalu melakukan settlement. Untuk saldo positif, Wali lanjut ke detail settlement dan mengonfirmasi penyerahan; saldo nol menampilkan penyelesaian tanpa langkah serah uang.
3. **Admin — proses pindah:** halaman menampilkan ringkasan readiness terkini dan alasan setiap blocker. Tombol **Sahkan Pindah** hanya tersedia bila backend menyatakan siap pada request saat ini. Server mengulang semua pemeriksaan saat submit; tampilan FE bukan otoritas.
4. **Hasil:** sukses menampilkan status `LEFT`, tanggal peristiwa, waktu pengesahan, dan placement yang ditutup. Pembatalan menampilkan dampak settlement yang tetap permanen serta kebijakan lanjutan yang disetujui.

### Pembatalan

Pembatalan hanya berlaku sebelum finalisasi dan tidak boleh menghapus atau membalikkan transaksi, settlement, handover, audit, atau riwayat kelas. Jika rekening sudah `SETTLED`, jangan membuka rekening otomatis. Karena account `SETTLED` tidak dapat menerima transaksi biasa dan dukungan multi-cycle belum diimplementasikan, kemampuan membatalkan setelah settlement adalah **keputusan tambahan yang belum tersedia**. Rekomendasi keselamatan adalah menolak pembatalan setelah settlement pertama, kecuali ada alur pemulihan terpisah yang disetujui. Pemilik perlu mengonfirmasi aturan ini sebelum implementasi.

## 4. Tanggal kejadian dan tanggal pengesahan

`effective_leaving_date` adalah tanggal peristiwa pindah yang sebenarnya dan ditetapkan ketika proses dimulai; tanggal itu dipakai untuk penutupan placement dan batas transaksi yang benar-benar terjadi sebelum siswa pindah. `finalized_at` adalah waktu administrasi saat Admin mengesahkan proses. Keduanya tidak boleh disamakan.

Dalam rancangan kompatibilitas, `students.leaving_date` tetap bermakna tanggal peristiwa pindah terakhir yang telah disahkan, bukan waktu finalisasi. Isi nilainya dari tanggal kejadian tersimpan saat finalisasi. Proses yang dibatalkan tidak mengubahnya. Histori lama tidak ditebak atau di-backfill dari tanggal finalisasi. Apakah `leaving_date` kelak didepresiasi demi tabel event merupakan keputusan teknis lanjutan.

## 5. Kontrak settlement dan penyerahan

`SettleStudentLeaving` harus menerima hanya rekening yang terkait dengan proses pindah sah berstatus `IN_PROGRESS`, untuk siswa yang masih belum `LEFT`. Menghapus `student.IsLeft()` saja tidak cukup. Service perlu memvalidasi:

1. Role Wali Kelas dan assignment/scope historis yang sah untuk rekening dan tahun.
2. Proses pindah aktif terikat pada siswa, tanggal kejadian, dan rekening/tahun yang diselesaikan.
3. Tahun belum `CLOSED`, rekening masih `OPEN`, dan tidak ada permintaan terlambat `PENDING`/`APPROVED` yang belum selesai.
4. Saldo dihitung ulang dari ledger aktif sesudah lock; nilai preview/expected balance tetap diperiksa agar stale preview ditolak.
5. Penyelesaian rekening, settlement, dan audit dilakukan atomik dalam UOW. Settlement menyimpan referensi proses pindah dan hanya dapat dibuat sekali per rekening/siklus.
6. Settlement positif menunggu konfirmasi penyerahan Wali melalui mekanisme rekonsiliasi yang ada. Finalisasi diblokir sampai nilai bersih yang wajib diserahkan selesai direkonsiliasi. Saldo nol tetap menghasilkan settlement, tetapi tidak membuat handover fisik.
7. Setiap kegagalan menggagalkan satu UOW sehingga status rekening, settlement, handover, audit, dan proses pindah tidak menjadi parsial.

Konfirmasi handover tetap aksi Wali, bukan Admin. Gunakan saldo bersih dan movement yang sudah didukung kontrak FASE 23, termasuk koreksi/return yang sah. Penutupan tahun tetap mengikuti kontrak FASE 23 dan terpisah dari pengesahan pindah.

## 6. Transaksi biasa, transaksi terlambat, dan rekening baru

### Transaksi biasa dan saldo final

Selama proses berlangsung, rekomendasi awal adalah membolehkan setoran/penarikan biasa hanya pada rekening `OPEN` selama tahun dan scope mengizinkan. Wali boleh menyelesaikan transaksi terakhir sebelum memilih settlement; saldo final dihitung saat settlement di bawah lock rekening dan tahun. Setelah rekening berubah menjadi `SETTLED`, transaksi biasa ditolak seperti saat ini. Ini menghindari penghentian transaksi terlalu dini ketika proses baru dimulai.

Finalisasi harus mengunci ulang himpunan rekening relevan secara deterministik dan menghitung ulang kewajiban sebelum menandai siswa `LEFT`. Pembuatan rekening/siklus baru harus ditolak selama proses aktif. Agar tidak ada rekening baru yang lolos bersamaan dengan finalisasi, pemeriksaan proses dan insert rekening harus terserialisasi di backend dan trigger/database dengan protokol lock yang sama. Detail mekanisme dan urutan lock wajib direview sebelum implementasi; pernyataan ini adalah kebutuhan, bukan klaim race sudah tertangani.

Jika produk menghendaki semua transaksi rutin berhenti segera saat proses dimulai, itu akan memerlukan perubahan aturan yang belum disetujui. Belum dipilih di amendment ini.

### Transaksi terlambat

Transaksi yang benar-benar terjadi sebelum tanggal pindah tetapi belum masuk ledger tetap mengikuti kontrol permintaan, alasan, bukti, dan persetujuan Admin pada [amendment transaksi terlambat](late-transaction-request-amendment.md). Agar dapat diselesaikan sebelum `LEFT`, permintaan baru harus terkait proses pindah aktif, menggunakan `effective_leaving_date`, dan dieksekusi oleh Wali hanya saat rekening masih `OPEN` dan sebelum settlement rekening. `PENDING` atau `APPROVED` memblokir settlement dan finalisasi; transaksi `COMPLETED` menjadi bagian ledger dan saldo settlement. Jangan menerima pencatatan setelah rekening settled/uang diserahkan melalui jalan tersembunyi.

Perubahan ini memerlukan amendment/implementasi terkoordinasi pada lifecycle request yang sekarang mengharuskan status `LEFT`. Data historis dari kontrak lama tetap memakai `students.leaving_date` dan referensi lama/null; jangan menulis ulang data historis. Mekanisme membaca dan menampilkan kedua bentuk histori perlu diuji.

### Siswa tanpa rekening / banyak rekening

Siswa tanpa rekening dapat disahkan pindah tanpa membuat rekening. Bila ada rekening, backend harus memeriksa seluruh rekening yang relevan, bukan hanya account pilihan halaman. Pada baseline saat ini unique `(student_id, academic_year_id)` membatasi satu rekening per tahun. Dukungan beberapa siklus per tahun adalah pekerjaan MQ-08-CONTRACT-01 yang belum diimplementasikan; proses baru tidak boleh bergantung pada fitur multi-cycle yang belum ada.

## 7. Siswa kembali dan multi-siklus

Keputusan MQ-08-CONTRACT-01 tetap berlaku: identitas siswa dipertahankan; penempatan kembali membuat histori baru; settlement lama immutable; Admin menangani lifecycle/kelas; Wali menangani keuangan. Amendment ini hanya mengubah kapan `LEFT` ditetapkan, dan tidak menyelesaikan rincian multi-cycle.

Jika proses pindah dibatalkan sebelum settlement, siswa tetap pada status/placement aktif dan kewajiban keuangan berlanjut seperti biasa. Jika pembatalan setelah settlement dipilih/diizinkan kemudian, rekening yang settled tidak dibuka otomatis; dibutuhkan kebijakan siklus baru dan keputusan tentang saldo awal yang hingga kini belum diputuskan. Jangan menafsirkan rekomendasi saldo awal `Rp. 0`, agregasi laporan, atau tabel lifecycle-event di [kontrak Siswa Kembali](student-return-and-savings-cycle-contract.md) sebagai keputusan final.

## 8. Konkurensi dan kebutuhan urutan lock

Laporan [MQ-08-REVIEW-01](../reviews/MQ-08_CONCURRENCY_SETTLEMENT_REVIEW.md) mengidentifikasi risiko lock inversion: Reactivate saat ini mengambil lock student lalu year, sedangkan transaksi rutin memakai year → account → student; pembuatan rekening juga memiliki trigger year → student. Settlement membaca student tanpa lock pada baseline yang diaudit. Risiko itu belum dinyatakan sebagai race yang telah direproduksi.

### Arah desain yang harus dibuktikan

Pilih urutan global konsisten. Kandidat untuk direview: tahun yang relevan dalam urutan ID → semua rekening terkait dalam urutan ID → siswa → proses pindah/permintaan. Setiap operasi harus mengambil subset sumber daya sesuai urutan tersebut; trigger eligibility dan operasi lifecycle harus sejalan. Jangan langsung menerapkan urutan kandidat ini tanpa memeriksa kemungkinan deadlock dengan year close/reopen, class transfer, transaksi, create-account, settlement, dan return.

| Interaksi | Risiko yang perlu ditutup | Invariant hasil |
|---|---|---|
| Mulai proses vs setoran/penarikan | Transaksi dapat lolos dengan status/proses stale | Transaksi yang commit sebelum start tetap di ledger; setelah proses dibuat, perilaku sesuai kebijakan; settlement menghitung ledger terbaru |
| Settlement vs transaksi biasa | Ledger berubah sesudah saldo dihitung | Lock dan status `SETTLED` menyerialisasi mutasi; nominal settlement sama dengan saldo ledger yang disetujui |
| Settlement vs pembatalan | Proses bisa batal saat settlement mengikat rekening | Tepat satu hasil terserialisasi; tidak ada auto-reopen atau efek parsial |
| Finalisasi `LEFT` vs setoran/penarikan | Akun masih dianggap aktif pada satu pihak | Operasi berjalan sebelum finalisasi atau ditolak sesudahnya sesuai keputusan; tidak ada transaksi commit dengan status dan proses yang bertentangan |
| Finalisasi vs pembuatan rekening/siklus | Satu rekening baru dapat luput dari pemeriksaan kewajiban | Set account yang diperiksa tidak dapat bertambah di antara validasi dan commit |
| Siswa Kembali vs settlement/proses pindah | Perubahan lifecycle beradu dengan kewajiban finansial | Status akhir mengikuti serialisasi yang disetujui; history/settlement tidak parsial |
| Dua pembuatan siklus | Siklus/nomor ganda atau dua account `OPEN` | Constraint dan lock menegakkan satu siklus terbuka sesuai kontrak yang kelak disetujui |

Semua race perlu integration test PostgreSQL disposable yang deterministik, dua transaksi, barrier/lock yang dikendalikan, timeout, dan pemeriksaan status, ledger, audit serta hasil commit/rollback. Jangan mengandalkan `time.Sleep` saja. Test juga trigger eligibility dan year-close/reopen. Jangan menyatakan risiko tertutup sebelum tes dijalankan.

## 9. Perubahan dokumentasi yang terkait

- [Kontrak FASE 19](fase-19-student-http-contract.md) menjelaskan perilaku lama; keputusan baru ini mengganti urutan leave/settlement-nya.
- [Amendment UI settlement pindah](student-leaving-settlement-ui-amendment.md) dan [amendment transaksi terlambat](late-transaction-request-amendment.md) menjelaskan workflow lama `LEFT`-dahulu; keduanya perlu dianggap superseded untuk urutan tersebut.
- [Aturan Bisnis](../BUSINESS_RULES.md) menunjuk kontrak ini sebagai aturan terbaru.
- [FASE 23](fase-23-year-end-settlement-contract.md) tetap menjadi sumber perilaku settlement/handover, saldo nol, audit, dan year-close yang tidak bertentangan dengan urutan baru.
- [FASE 21](fase-21-transaction-http-contract.md) tetap mengatur transaksi rutin dan autorisasi; perlakuan transaksi ketika proses pindah aktif perlu amendment implementasi.
- [Audit FASE 16.5](fase-16-5-financial-contract-audit.md) dan laporan MQ-08 sebelumnya tetap snapshot historis; rekomendasi/proposisi lama di dalamnya tidak berarti implementasi terkini.
- [Kontrak Siswa Kembali](student-return-and-savings-cycle-contract.md) dan [rancangan MQ-08](../reviews/MQ-08_STUDENT_RETURN_DESIGN.md) tetap berlaku untuk topik kembali/multi-siklus sejauh tidak menyimpulkan urutan `LEFT`-sebelum-settlement.

## 10. Rencana implementasi bertahap

1. **Review kontrak teknis:** supervisor meninjau model proses, pembatalan setelah settlement, scope rekening relevan, perlakuan transaksi rutin selama proses, data historis, dan lock protocol.
2. **Schema dan migrasi:** rancang tabel proses pindah serta FK nullable dari settlement/permintaan terlambat; verifikasi upgrade/rollback pada database PostgreSQL disposable berisi dan kosong. Tidak ada perubahan data historis otomatis atau tebakan tanggal.
3. **Domain/repository/service:** buat start, cancel, readiness, finalize; ubah settlement dan late-request agar merujuk proses aktif; otorisasi Admin lifecycle dan Wali finansial tetap terpisah.
4. **Serialisasi dan database guard:** selaraskan semua lock order termasuk trigger; cek account set secara atomik pada finalisasi; uji race start/transaction, settlement/transaction, cancellation, finalize/create-account, Reactivate, year close/reopen, dan multi-cycle.
5. **HTTP/FE:** Admin memulai proses dan melihat hambatan per rekening; Wali membuka rekening terkait, menangani late request, settlement, dan handover; Admin hanya mendapat tombol finalisasi saat readiness dihitung siap. Semua aksi dilindungi CSRF serta otorisasi server.
6. **Histori dan kompatibilitas:** tampilkan proses baru bersama siswa `LEFT` historis berdasarkan data lama tanpa mengubah record. Pastikan laporan, histori, detail rekening, dan transaksi terlambat membedakan tanggal kejadian dari waktu finalisasi.
7. **Acceptance/release:** unit, integration, concurrency, HTTP/template, serta E2E browser; migration, rollback, data historis, audit, dan pemulihan diuji di disposable. Perbarui kontrak/TESTING/HANDOVER dan deploy melalui review workflow.

## 11. Keputusan tambahan sebelum implementasi

Keputusan berikut belum ditetapkan oleh pemilik dan harus dipastikan sebelum tahap yang memerlukannya:

1. Apakah proses boleh dibatalkan setelah ada satu atau lebih rekening settled? Rekomendasi: tidak, kecuali tersedia alur eksplisit siklus baru/pemulihan.
2. Rekening mana yang termasuk “kewajiban relevan”: rekomendasi audit adalah semua siklus/rekening siswa, termasuk tahun sebelumnya; perlakuan rekening `OPEN` pada tahun `CLOSED` belum disetujui dan tidak boleh auto-reopen.
3. Apakah `effective_leaving_date` wajib dimasukkan saat start, dan apakah boleh backdate; validasi terhadap placement aktif/historis perlu diputuskan.
4. Apakah transaksi biasa tetap boleh pada rekening OPEN setelah proses dimulai sampai Wali settle akun itu? Rekomendasi dokumen ini: ya; saldo dihitung ulang ketika settlement. Jika tidak, dibutuhkan aturan pembekuan yang eksplisit.
5. Apakah permintaan transaksi terlambat baru boleh dibuat setelah proses mulai dan tetap mensyaratkan persetujuan Admin serta bukti? Rekomendasi: ya, memakai kontrol existing; apakah peninjau Admin boleh menyetujui sebelum siswa LEFT perlu dipastikan dari amendment implementasi.
6. Apa yang dilakukan bila saldo negatif/tidak seimbang, ada transaksi tidak teratribusi, akun berada di tahun tertutup, atau settlement positif memiliki handover/reopen yang belum direkonsiliasi?
7. Apakah finalisasi mengunci semua tahun/rekening siswa atau hanya proses/request lalu memeriksa ulang melalui mekanisme lain? Detail harus disepakati lewat review concurrency.
8. Setelah proses pindah dibatalkan, kapan dan bagaimana Wali boleh melanjutkan siklus bila rekening sudah settled? Kebijakan saldo awal tetap belum diputuskan.

## 12. Batas verifikasi

Dokumen ini berdasarkan pembacaan source, schema, trigger, tests, dan kontrak pada baseline yang disebut di atas. Tidak ada kode, migration, database, fixture, atau tes yang diubah maupun dijalankan. State machine dan lock order adalah rancangan; tidak membuktikan sistem saat ini memenuhi aturan baru atau race telah terselesaikan.
