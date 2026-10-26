# Kontrak Siswa Kembali dan Siklus Tabungan

**Status keputusan bisnis:** disetujui pemilik aplikasi untuk MQ-08-CONTRACT-01.
**Status implementasi:** belum diimplementasikan. Dokumen ini menetapkan perilaku produk; rancangan schema/API rinci tetap rekomendasi sampai mendapat review teknis.

## 1. Keputusan bisnis yang disetujui

1. Identitas siswa tidak berubah ketika siswa pindah lalu kembali. Jangan membuat siswa baru hanya untuk memulai ulang tabungan.
2. Setiap penempatan kembali membuat baris `StudentClassHistory` baru dengan tahun, kelas, dan tanggal efektif yang dipilih. Riwayat lama tidak dibuka kembali atau ditimpa.
3. Siswa `LEFT` yang kembali pada tahun yang sama boleh memakai reaktivasi biasa selama rekening tahun tersebut belum `SETTLED`.
4. Jika rekening tahun yang sama sudah `SETTLED`, siswa yang kembali memakai alur khusus **Siswa Kembali**. Alur tersebut membuat penempatan baru dan dapat diikuti siklus tabungan baru jika diperlukan.
5. Jika siswa kembali pada tahun ajaran berbeda, buat penempatan pada tahun itu dan gunakan siklus tabungan untuk tahun tersebut. Rekening tahun lain tetap terpisah.
6. Riwayat kelas, rekening, transaksi, settlement, penyerahan uang, dan audit yang sudah ada harus dipertahankan. Settlement lama tidak otomatis dibatalkan, dibuka kembali, diubah, atau dipindahkan ke rekening baru.
7. Admin mengelola status siswa dan penempatan kelas. Admin tidak melakukan mutasi keuangan. Wali Kelas menangani pembuatan rekening/siklus dan transaksi keuangan dalam kelas/tahun yang menjadi scope-nya. Siswa hanya membaca tabungannya sendiri.

### Rekomendasi yang belum menjadi keputusan bisnis

- Jangan membawa saldo lama otomatis ke siklus baru. Karena settlement lama tidak dibatalkan atau diubah, rekomendasi awal adalah siklus baru mulai `Rp. 0` dan hanya transaksi baru yang membentuk saldonya. Pemilik perlu mengonfirmasi aturan saldo awal sebelum implementasi; kontrak belum menyetujui transaksi pembukaan saldo atau pemindahan uang.

## 2. Alur bisnis per kasus

Dalam tabel berikut, “riwayat lama tetap” berarti semua kelas, rekening, transaksi, settlement, arus penyerahan, dan audit terdahulu tetap dapat ditinjau dan tidak diedit oleh proses kembali.

| Kasus | Status siswa dan riwayat kelas | Rekening/siklus dan saldo | Riwayat lama dan audit | Hak akses |
|---|---|---|---|---|
| **A. Kembali pada tahun yang sama; rekening belum `SETTLED`** | Admin mengubah `LEFT → ACTIVE` melalui reaktivasi biasa dan membuat penempatan baru mulai tanggal efektif. Periode penempatan yang lama tetap tertutup; rentang tanggal tidak boleh tumpang tindih. | Pakai kembali rekening tahun itu yang masih `OPEN`; jangan membuat siklus kedua. Saldo tetap hasil transaksi aktif pada rekening yang sama. Bila rekening belum pernah dibuat, Admin menyelesaikan reaktivasi/penempatan; Wali dapat membuat rekening pertama lewat alur keuangan yang berwenang. | Transaksi lama tetap pada rekening yang sama. Catat aktor, tanggal efektif, kelas/tahun, serta perubahan status dalam jejak audit yang sesuai. Kebutuhan audit event baru perlu implementasi dan review teknis. | Admin menjalankan reaktivasi. Wali melihat buku dan melanjutkan kegiatan keuangan sesuai scope. Admin tetap read-only untuk keuangan. |
| **B. Kembali pada tahun yang sama; rekening `SETTLED`, belum meminta tabungan baru** | Admin memakai alur **Siswa Kembali**; status menjadi `ACTIVE`, penempatan baru dibuat. | Rekening lama tetap `SETTLED`; tidak ada rekening baru dan saldo siklus baru belum ada. | Settlement lama tidak dibatalkan. Jika penyerahan masih menunggu, statusnya tetap menunggu dan harus terlihat terpisah. Catat keputusan kembali dan referensi ke rekening/settlement lama. | Admin hanya mengelola lifecycle/penempatan. Wali melanjutkan penyerahan settlement lama jika menjadi tanggung jawabnya; Admin tidak mengonfirmasi atau memindahkan uang. |
| **C. Kembali pada tahun yang sama; rekening `SETTLED`, membutuhkan tabungan baru** | Admin menjalankan **Siswa Kembali** dan menambahkan riwayat penempatan baru. | Settlement lama tetap final. Wali membuat rekening/siklus baru untuk tahun yang sama hanya bila diperlukan dan setelah penempatan/scope tersedia. Saldo awal siklus baru mengikuti keputusan produk yang belum dikonfirmasi; rekomendasi teknis adalah `Rp. 0` dan hanya transaksi baru masuk rekening baru. | Semua transaksi dan settlement lama tetap terkait rekening lama. Siklus baru memiliki ID rekening baru dan audit pembuatan tersendiri. Tidak ada pembatalan settlement otomatis. | Admin tidak membuat rekening. Wali membuat siklus melalui layanan keuangan dengan pengecekan status, scope kelas/tahun, dan pencegahan siklus OPEN ganda. |
| **D. Kembali pada tahun ajaran berbeda** | Admin membuat penempatan baru untuk tahun yang dipilih dan mengaktifkan siswa sesuai aturan lifecycle. Riwayat tahun lama tidak berubah. | Gunakan rekening yang sudah ada pada tahun tujuan bila masih `OPEN`; jika belum ada, Wali dapat membuat siklus tahun itu. Setiap tahun mempunyai ledger sendiri. Saldo awal siklus baru mengikuti keputusan saldo yang belum ditetapkan. Bila rekening tahun tujuan sudah settled, terapkan alur khusus kasus C. | Tidak menggabungkan transaksi atau settlement antar tahun. Audit mengidentifikasi tahun/kelas efektif dan aktor. | Admin mengelola penempatan; Wali menangani siklus dan keuangan pada assignment yang berlaku; siswa hanya melihat data sendiri. |
| **E. Pindah dan kembali beberapa kali di tahun yang sama** | Setiap keluar menutup periode penempatan; setiap kembali membuat periode kelas baru dengan tanggal efektif. Identitas student tetap satu. Periode harus tidak saling tumpang tindih. | Jika rekening masih OPEN, semua kembali menggunakan rekening itu. Jika settlement telah dibuat, kembali menggunakan **Siswa Kembali**; setiap siklus baru hanya dibuat Wali setelah siklus sebelumnya settled. Dalam satu tahun dapat ada beberapa rekening/siklus, tetapi maksimal satu siklus OPEN per siswa/tahun direkomendasikan sebagai invariant teknis. | Seluruh periode kelas dan siklus finansial tetap terpisah. Peristiwa leave/return harus dapat ditelusuri; apakah memerlukan tabel lifecycle-event khusus masih rekomendasi teknis. | Admin mengelola setiap transisi dan penempatan; Wali menangani ledger/siklus. Akses Wali tetap mengikuti scope historis/aktif yang ditentukan kebijakan keuangan. |
| **F. Siswa kembali setelah menerima penyerahan settlement** | Admin membuat penempatan baru melalui alur sesuai status rekening tahun itu. | Siklus settled lama tidak dibuka lagi. Jika perlu tabungan, Wali membuat rekening/siklus baru; saldo awalnya mengikuti keputusan saldo yang belum ditetapkan (rekomendasi teknis: `Rp. 0`). | Baris settlement dan arus penyerahan yang selesai tetap utuh. Siklus baru tidak mengubah bukti uang lama telah diserahkan. Audit menghubungkan catatan kembali dengan siswa dan referensi siklus sebelumnya. | Sama seperti kasus C: lifecycle/kelas oleh Admin; keuangan oleh Wali. |

### Keadaan yang memerlukan klarifikasi produk

- Nilai saldo awal siklus baru belum diputuskan; rekomendasi teknis adalah `Rp. 0` dan tidak ada pemindahan nominal settlement lama tanpa workflow khusus yang disetujui.
- Keputusan yang diberikan belum menyatakan apakah Wali boleh membuat siklus baru saat settlement positif sebelumnya masih menunggu penyerahan/rekonsiliasi. Sistem wajib menampilkan tunggakan tersebut; usulan untuk menahan pembuatan siklus sampai rekonsiliasi adalah rekomendasi keamanan operasional, bukan aturan yang sudah disetujui.
- Untuk kembali pada tahun berbeda yang sudah `CLOSED`, kontrak ini tidak menetapkan penempatan retroaktif atau pembukaan siklus. Perilaku harus mengikuti aturan tahun ajaran dan reopen yang sudah berlaku; jangan membuka tahun secara implisit dari alur kembali.
- Tanggal efektif kembali harus mengikuti validasi rentang tahun dan kelas yang berlaku. Aturan jika tanggal efektif lebih awal dari `leaving_date` terakhir perlu ditetapkan sebelum menerima backdate.

## 3. Audit implementasi saat ini

Fakta dari baseline MQ-08:

- [`StudentService.Reactivate`](../../internal/service/student_service.go#L885-L933) hanya menerima Admin, mengunci siswa, mengubah `LEFT → ACTIVE`, membuat placement melalui `studentPlacementAtDate`, lalu menyimpan student dan history dalam UOW. Fungsi tidak memeriksa status rekening atau settlement.
- `studentPlacementAtDate` mengunci tahun `FOR UPDATE`, memvalidasi kelas/tanggal, lalu membuat history sampai akhir tahun ([student_service.go](../../internal/service/student_service.go#L935-L963)). Riwayat lama tidak dibuka kembali.
- [`Student.Reactivate`](../../internal/domain/student.go#L83-L91) hanya mengubah status dan `UpdatedAt`; ia tidak menghapus `LeavingDate`. `Student.LeaveOn` menimpa `LeavingDate` saat keluar berikutnya ([student.go](../../internal/domain/student.go#L64-L80)). Oleh karena itu kolom tunggal `students.leaving_date` bukan histori lengkap untuk beberapa siklus keluar/kembali.
- Schema mempunyai satu `savings_accounts` row per siswa/tahun melalui `savings_accounts_student_year_unique` ([000001_init_schema.up.sql](../../migrations/000001_init_schema.up.sql#L130-L150)). Account ID adalah induk transaksi dan settlement ([000001_init_schema.up.sql](../../migrations/000001_init_schema.up.sql#L153-L188)). Satu rekening saat ini adalah satu unit ledger.
- [`SavingsAccountService.Create`](../../internal/service/savings_account_service.go#L14-L58) mewajibkan role Wali, siswa ACTIVE, tahun OPEN, dan scope assignment. Pembuatan akun mencatat audit CREATE ([savings_account_service.go](../../internal/service/savings_account_service.go#L84-L105)). Ini selaras dengan aturan Admin read-only keuangan ([financial-supervision-access.md](financial-supervision-access.md#L5-L10)).
- Lookup `GetByStudentAndAcademicYear` mengasumsikan satu hasil, dan dipakai oleh direktori tabungan serta dashboard siswa ([savings_account_repository.go](../../internal/repository/postgres/savings_account_repository.go#L94-L118), [savings_page.go](../../internal/http/savings_page.go#L201-L231), [reporting_service.go](../../internal/service/reporting_service.go#L45-L72)). Laporan memakai join rekening per siswa/tahun dan saat ini menampilkan satu account row dalam hasil siswa ([reporting_repository.go](../../internal/repository/postgres/reporting_repository.go#L24-L56), [reporting_repository.go](../../internal/repository/postgres/reporting_repository.go#L83-L118)).
- Settlement pindah mensyaratkan siswa `LEFT`, account `OPEN`, dan tahun bukan `CLOSED`; settlement, perubahan status rekening, dan audit ditulis bersama ([settlement_service_settle.go](../../internal/service/settlement_service_settle.go#L158-L284)).
- Route Reactivate saat ini merupakan POST Admin di detail siswa ([student_page.go](../../internal/http/student_page.go#L618-L650)). Halaman Tabungan menawarkan pembuatan rekening kepada Wali untuk siswa aktif tanpa rekening dan hanya sesuai scope/tahun terbuka ([savings_page.go](../../internal/http/savings_page.go#L193-L231)). Belum ada route atau UI multi-siklus.

Ini adalah deskripsi implementasi yang diperiksa, bukan klaim bahwa kontrak baru sudah diterapkan.

## 4. Hak akses dan audit

- **Admin:** mengubah status siswa dan penempatan kelas melalui alur kembali. Tidak membuat rekening, menyetor, menarik, mengoreksi, menyelesaikan, atau merekonsiliasi uang.
- **Wali Kelas:** membaca histori keuangan dalam scope yang diizinkan; membuat siklus baru dan melakukan transaksi melalui layanan yang memvalidasi penugasan kelas/tahun. Penyerahan settlement dilakukan oleh role dan alur yang sudah ditentukan kontrak settlement terkait.
- **Siswa:** baca-saja untuk akun sendiri; tidak membuat siklus atau mengubah data.
- Tidak boleh ada jalur UI yang menggantikan otorisasi service/backend.
- Semua operasi baru harus mempertahankan audit transaksi keuangan yang ada. **Rekomendasi teknis:** tambah event audit lifecycle untuk reaktivasi/siswa kembali dengan actor, waktu, student ID tetap, status sebelum/sesudah, tahun, kelas, tanggal efektif, serta ID siklus/settlement terkait jika ada. Kebutuhan skema dan tipe audit baru perlu persetujuan implementasi.

## 5. Batas siklus tabungan

1. Setiap siklus mempunyai identitas rekening unik. Semua deposit/withdrawal tetap merujuk ke rekening siklus itu; settlement tetap merujuk ke rekening yang diselesaikan.
2. Saldo dihitung hanya dari transaksi aktif pada account/cycle tersebut. Saldo awal siklus baru belum diputuskan; rekomendasi teknis awal adalah nol tanpa pemindahan nominal settlement lama.
3. Settlement menjadi riwayat finansial permanen. Alur siswa kembali tidak memodifikasi status, jumlah, actor, tanggal, atau cash movement settlement lama.
4. Reopen hanya melalui workflow reopen yang ada dan kebijakannya; tidak dipicu otomatis oleh reaktivasi.
5. Dalam satu tahun, siswa boleh memiliki lebih dari satu siklus berurutan akibat settlement, tetapi sebaiknya hanya satu siklus OPEN pada satu waktu. Pembuatan paralel harus terserialisasi.
6. Tidak boleh ada transaksi atau settlement yang dipindahkan dari account lama ke account baru.

## 6. Pengujian penerimaan bisnis

Implementasi kelak harus membuktikan sedikitnya:

- Skenario A menggunakan account OPEN yang sama, saldo/ledger tidak berubah hanya karena reaktivasi, dan history placement baru dibuat.
- Skenario B tidak membuat account baru; settlement lama serta status handover tetap persis.
- Skenario C membuat account baru hanya oleh Wali yang berwenang; saldo awal nol dan semua transaksi baru menunjuk account baru.
- Skenario D memisahkan laporan dan saldo menurut tahun; tahun lama tidak berubah.
- Skenario E mempertahankan semua placement dan siklus; tidak ada rentang tanggal tumpang tindih atau dua siklus OPEN.
- Skenario F mempertahankan settlement dan bukti handover yang sudah selesai; account baru terpisah.
- Admin ditolak pada seluruh endpoint mutasi finansial; Wali di luar assignment ditolak; siswa hanya bisa membaca miliknya sendiri.
- Gagal di tengah alur tidak meninggalkan status student, history, account, audit, atau transaksi yang hanya sebagian tersimpan.

## 7. Keputusan teknis yang belum disetujui

Hal-hal berikut adalah rekomendasi/pertanyaan implementasi, bukan bagian yang dianggap sudah diputuskan pemilik:

- Format model multi-cycle: gunakan row `savings_accounts` sebagai satu cycle dengan nomor siklus eksplisit, atau buat tabel induk `savings_cycles` dan pertahankan accounts sebagai ledger. Rekomendasi awal di dokumen desain memilih alternatif row-per-cycle dengan nomor eksplisit karena transaksi/settlement sudah berelasi ke account ID.
- Kebijakan pembuatan cycle saat settlement penyerahan lama masih pending.
- Apakah perlu lifecycle event table terpisah untuk setiap leave/return, mengingat `leaving_date` hanya satu kolom pada Student.
- Ubah API/queries yang kini mengembalikan satu rekening per siswa/tahun menjadi pemilihan current/open cycle dan daftar seluruh cycles.
- Urutan lock global lintas leave, reactivate, settlement, transaksi biasa, dan create-cycle untuk menghilangkan deadlock yang tercatat di [MQ-08 review](../reviews/MQ-08_CONCURRENCY_SETTLEMENT_REVIEW.md).
- Kebijakan penempatan efektif pada tahun yang CLOSED dan aturan backdate.

Migration, kode, route, dan interface belum diubah oleh kontrak ini.
