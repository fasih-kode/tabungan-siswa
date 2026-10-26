# MQ-08-REVIEW-01 — Audit Konkurensi Reaktivasi dan Settlement

**Status:** audit read-only; belum ada perubahan implementasi atau keputusan aturan bisnis.
**Baseline:** `db5ddcccb6067471b4f9e7bf17e3ef13204e1ad1` pada branch `review`.
**Kesimpulan ringkas:** kode menunjukkan interleaving yang dapat menghasilkan siswa `ACTIVE` dengan rekening tahun tersebut `SETTLED`. Ini adalah celah invariant yang perlu diputuskan, belum terbukti sebagai bug karena kontrak tidak menetapkan apakah reaktivasi setelah settlement harus ditolak atau bagaimana rekeningnya ditangani. Pada pasangan langsung ini belum tampak siklus deadlock karena settlement membaca siswa tanpa lock. Namun menambahkan lock siswa ke settlement dengan urutan saat ini dapat menciptakan deadlock dengan `Reactivate`. Ada pula siklus lock potensial terpisah antara `Reactivate` dan setoran/penarikan biasa.

## 1. Batas audit dan fakta kontrak

Audit membaca implementasi service, repository PostgreSQL, Unit of Work, kontrak lifecycle/settlement, serta test yang relevan. Tidak ada source, test, migration, konfigurasi, atau data database yang diubah; tidak ada test yang dijalankan.

Fakta kontrak:

- Admin boleh mengaktifkan kembali siswa `LEFT`; proses membuat riwayat kelas baru dan mengubah status menjadi `ACTIVE` dalam satu transaksi. Kontrak tidak menyebut pemeriksaan rekening yang sudah `SETTLED` ([student-class-placement-workflow.md](../contracts/student-class-placement-workflow.md#L25-L29)).
- Settlement pindah sekolah berlaku untuk siswa `LEFT`, rekening terbuka, dan tahun `OPEN` atau `CLOSING`; settlement serta konfirmasi penyerahan merupakan langkah keuangan tersendiri ([student-leaving-settlement-ui-amendment.md](../contracts/student-leaving-settlement-ui-amendment.md#L44-L59), [BUSINESS_RULES.md](../BUSINESS_RULES.md#L55-L57)).
- Kontrak menyatakan transaksi biasa memerlukan siswa aktif di titik tulis yang terserialisasi, tetapi tidak menetapkan aturan reaktivasi terhadap settlement yang sudah terjadi ([student-leaving-settlement-ui-amendment.md](../contracts/student-leaving-settlement-ui-amendment.md#L44-L51)).
- Satu rekening per siswa per tahun dijamin oleh unique constraint `savings_accounts_student_year_unique` ([000001_init_schema.up.sql](../../migrations/000001_init_schema.up.sql#L139-L140)). Reaktivasi tidak membuat atau membuka kembali rekening.

Karena itu, “`ACTIVE` + rekening tahun yang sama `SETTLED`” adalah keadaan yang dihasilkan/dimungkinkan oleh implementasi, tetapi apakah keadaan tersebut dilarang merupakan **keputusan kontrak yang belum tercatat**, bukan bug terkonfirmasi.

## 2. Peta transaksi dan lock

`PostgresUnitOfWorkManager.Begin` memakai `BeginTx(ctx, nil)` dan membuat seluruh repository memakai transaksi SQL yang sama ([unit_of_work.go](../../internal/repository/postgres/unit_of_work.go#L37-L63)). Tidak ada isolation level eksplisit. PostgreSQL memakai `READ COMMITTED` sebagai default; SELECT biasa melihat snapshot committed saat statement dimulai dan tidak menunggu row lock yang menahan perubahan belum commit ([dokumentasi isolasi transaksi PostgreSQL](https://www.postgresql.org/docs/18/transaction-iso.html)).

| Alur | Urutan repository/lock yang relevan | Pembacaan dan perubahan | Penyelesaian transaksi |
|---|---|---|---|
| `Reactivate` | `Students.GetByIDForUpdate` (**student `FOR UPDATE`**) → `AcademicYears.GetByIDForUpdate` (**year `FOR UPDATE`**) melalui `studentPlacementAtDate` → baca kelas → buat riwayat → update student | Status dibaca di bawah student lock; domain `Reactivate()` mengubah `LEFT → ACTIVE`; menambahkan riwayat kelas baru. Tidak membaca atau mengunci rekening. Lihat [`Reactivate`](../../internal/service/student_service.go#L885-L933), [`studentPlacementAtDate`](../../internal/service/student_service.go#L935-L955), dan SQL [`GetByIDForUpdate`](../../internal/repository/postgres/student_repository.go#L86-L104). | Satu UOW; rollback pada semua jalur gagal; commit setelah history dan student update. |
| `SettleStudentLeaving` | baca rekening tanpa lock untuk mendapat year ID → year **`FOR SHARE`** → account **`FOR UPDATE`** → baca student tanpa lock (`GetByID`) → baca transaksi → update account ke `SETTLED` → insert settlement dan dua audit | Status student harus `LEFT`, tetapi pembacaan tidak mengunci student. Settlement tidak mengubah student. Lihat [`SettleStudentLeaving`](../../internal/service/settlement_service_settle.go#L158-L284), SQL year [`FOR SHARE`](../../internal/repository/postgres/academic_year_repository.go#L90-L110), dan account [`FOR UPDATE`](../../internal/repository/postgres/savings_account_repository.go#L121-L139). | UOW/nested transaction; settlement, account, audit dalam transaksi yang sama; commit di akhir jika method memiliki transaksi. |
| `Leave` | student **`FOR UPDATE`** → cek transfer tertunda → tutup riwayat kelas → update student | Mengubah `ACTIVE → LEFT`; tidak mengambil lock year di jalur yang diperiksa. Lihat [`Leave`](../../internal/service/student_service.go#L809-L883). | Satu UOW; commit setelah update student. |
| Setor/tarik biasa | baca account → year **`FOR SHARE`** → account **`FOR UPDATE`** → student **`FOR UPDATE`** melalui `requireActiveSavingsStudent` | Status active diperiksa setelah lock student; insert transaksi dan audit setelahnya. Lihat [`lockOpenTransactionAccount`](../../internal/service/transaction_service.go#L737-L773) dan [`requireActiveSavingsStudent`](../../internal/service/transaction_service.go#L775-L789). | Satu UOW; operasi finansial dan audit berada dalam transaksi yang sama. |

Lock `FOR UPDATE` pada year oleh reaktivasi berkonflik dengan `FOR SHARE` yang dipakai settlement/transaksi finansial. Pembacaan student biasa oleh settlement tidak berkonflik sebagai row lock dengan student `FOR UPDATE` milik reaktivasi.

## 3. Matriks interleaving

| Skenario | Urutan dan lock | Hasil yang mungkin | Deadlock / status bukti |
|---|---|---|---|
| Reaktivasi commit sebelum settlement membaca student | R mengunci student, lalu year `FOR UPDATE`; membuat history dan mengubah status; R commit. Setelah itu S mendapat year `FOR SHARE`, account `FOR UPDATE`, lalu membaca student `ACTIVE`. | R sukses; S ditolak `ErrInvalidTransition`; status `ACTIVE`, rekening tetap `OPEN`, settlement tidak dibuat. | Tidak tampak deadlock. Urutan dan pemeriksaan status dibuktikan kode; hasil interleaving belum diuji di PostgreSQL. |
| Settlement membaca `LEFT` sebelum reaktivasi commit | S memegang year `FOR SHARE` dan account `FOR UPDATE`; membaca student sebagai `LEFT`. R memperoleh student `FOR UPDATE`, mengubah row dalam transaksi, lalu meminta year `FOR UPDATE` dan menunggu S. S tidak meminta lock student; ia dapat menyelesaikan settlement dan commit. Setelah itu R memperoleh year lock dan commit. | Kedua operasi dapat sukses berurutan secara commit; status akhir dapat `ACTIVE`, rekening `SETTLED`, settlement tercatat. History reaktivasi ikut tercatat. | Tidak ada siklus lock langsung pada pasangan ini karena S tidak menunggu student. Interleaving mungkin menurut call/lock yang terlihat; belum direproduksi. Apakah state akhir dilarang adalah celah kontrak. |
| R menahan student ketika S sudah mengunci year/account | R mengambil student `FOR UPDATE`. S mengambil year `FOR SHARE`, account `FOR UPDATE`, lalu menjalankan SELECT student tanpa lock. SELECT melihat versi committed sebelumnya, lazimnya `LEFT`, tanpa menunggu perubahan R. R selanjutnya menunggu year `FOR UPDATE` milik S. | S dapat settle berdasarkan nilai committed `LEFT`; R menunggu hingga S commit/rollback. Jika S commit, R dapat melanjutkan lalu mengaktifkan siswa. | Tidak ada deadlock antara R dan S dengan lock aktual. Ini adalah variasi skenario sebelumnya; SELECT biasa tidak menjadi synchronization point. Perlu uji PostgreSQL untuk bukti runtime. |
| Settlement commit lebih dahulu, Reactivate menyusul | S selesai dengan account `SETTLED`; setelah lock year dilepas, R menjalankan reaktivasi yang hanya memeriksa student dan year. | Student dapat menjadi `ACTIVE`, sementara account tahun tersebut tetap `SETTLED`. Unique constraint mencegah rekening kedua untuk kombinasi siswa/tahun, dan Reactivate tidak membuka kembali rekening. | Bukan otomatis deadlock atau korupsi ledger; konsistensi bisnis belum ditentukan. Kontrak perlu menyatakan apakah reaktivasi dilarang setelah settlement, butuh proses pembukaan kembali, atau dapat diterima. |
| Salah satu transaksi rollback | Bila R rollback saat menunggu year, perubahan status/history R batal dan lock student dilepas; S melihat status committed sebelumnya dan dapat settle jika `LEFT`. Bila S rollback, perubahan account/settlement/audit dibatalkan bersama; R dapat memperoleh year lock lalu commit reaktivasi. | Operasi yang commit saja menentukan state; transaksi yang rollback tidak semestinya meninggalkan sebagian write karena satu UOW. | Atomicity didukung struktur UOW; rollback interleaving belum diuji. Error commit/rollback dan keadaan koneksi saat deadlock masih perlu test integrasi. |

### Risiko deadlock terpisah pada transaksi biasa

Walau pasangan `Reactivate`–`SettleStudentLeaving` saat ini tidak membentuk siklus, urutan lock `Reactivate` adalah **student → year**, sementara setor/tarik biasa adalah **year → account → student**. Interleaving yang mungkin: R memegang student dan menunggu year; transaksi finansial memegang year/account dan menunggu student. Ini membentuk siklus, sehingga PostgreSQL dapat membatalkan salah satu transaksi (umumnya dengan deadlock detected / SQLSTATE `40P01`). Belum ada reproduksi atau test untuk interleaving ini; ini adalah risiko potensial yang diturunkan dari lock order, bukan failure yang diamati.

Penambahan student `FOR UPDATE` ke settlement dengan urutannya saat ini juga akan menciptakan bentuk siklus yang sama: R memegang student lalu menunggu year, sementara S memegang year/account lalu menunggu student. Karena itu, menambahkan lock siswa sebagai patch tunggal **tidak aman**.

## 4. Risiko terbukti, dugaan, dan yang belum diketahui

**Terbukti dari kode:** Reactivate mengunci student lalu year; settlement mengunci year lalu account, tetapi membaca student biasa; settlement mengubah account dan membuat settlement/audit dalam UOW; Reactivate tidak memeriksa account/settlement; transaksi biasa mengunci year → account → student.

**Belum terbukti melalui runtime:** interleaving yang menghasilkan `ACTIVE + SETTLED`; adanya deadlock pada Reactivate vs setor/tarik; tingkat kemungkinan/produksi interleaving; apakah PostgreSQL membatalkan transaksi pada test konkret; apakah retry yang ada menangani SQLSTATE deadlock dengan aman.

**Belum ditetapkan oleh kontrak:** apakah reaktivasi setelah settlement tahun berjalan diizinkan; jika diizinkan, apakah rekening tetap settled, settlement harus dibuka kembali, atau rekening baru boleh dibuat; apakah tanggal efektif reaktivasi dapat berada sebelum settlement yang sudah terjadi; dan cara menjaga konsistensi histori kelas dengan histori keuangan.

**Inventaris pengujian yang terlihat:** ada unit test domain untuk transisi `Student.Reactivate`, unit test settlement yang menolak siswa aktif, dan integration test settlement pindah sekolah berurutan. Pencarian test tidak menemukan service/integration test Reactivate maupun test konkuren Reactivate–settlement. Tidak ada test yang dijalankan untuk audit ini.

## 5. Evaluasi alternatif solusi

| Alternatif | Dampak / trade-off | Penilaian |
|---|---|---|
| Tambah `Students.GetByIDForUpdate` dalam settlement pada urutan sekarang | Memastikan pemeriksaan status terserialisasi terhadap row student, tetapi membuat siklus dengan lock order Reactivate (student→year) ketika settlement memegang year/account. Dapat menyebabkan deadlock dan satu operasi gagal. Hak akses dan audit tidak berubah, tetapi ada perubahan hasil race. | Jangan lakukan sendiri; tidak aman tanpa perubahan order lintas workflow dan test. |
| Selaraskan urutan lock untuk lifecycle dan operasi keuangan | Dapat menghindari siklus jika semua jalur memakai urutan total yang sama dan mengunci status pada titik keputusan. Perlu mengubah Reactivate, kemungkinan Leave, settlement, serta transaksi biasa; perubahan dapat memengaruhi throughput, error, latency, dan serialisasi. Urutan year→account→student konsisten dengan jalur finansial, tetapi lifecycle perlu tahu/menjaga rekening terkait dan menghindari lock account→year. | Pilihan teknis paling kuat bila kontrak menuntut satu hasil serial. Perlu rancangan terkoordinasi serta review semua pemanggil/lock path. |
| Validasi ulang status setelah lock yang tepat | Pembacaan/validasi status harus terjadi setelah student row lock yang kompatibel dengan urutan global; hanya menambahkan pemeriksaan setelah SELECT biasa tidak menutup race. | Wajib sebagai bagian solusi serialisasi, bukan solusi berdiri sendiri. Otorisasi tetap pada service; audit tetap harus commit atomik dengan perubahan. |
| Guard reaktivasi jika ada rekening/settlement tahun terkait | Menutup state `ACTIVE + SETTLED` dengan menolak reaktivasi pada kasus tertentu; perlu menentukan cakupan tahun, rekening terbuka, settlement belum diserahkan, serta effective date. Guard harus lock data yang diperiksa dan mengikuti urutan global, atau race check-then-act tetap ada. Tidak mengubah histori keuangan bila ditolak. | Perubahan aturan produk; perlu keputusan owner sebelum implementasi. Bisa menjadi perubahan terkecil jika aturan dipilih “settlement final memblokir reaktivasi”. |
| Terima `ACTIVE + SETTLED` sebagai keadaan sah | Tidak perlu perubahan transaksi untuk state itu, tetapi antarmuka/aturan harus menjelaskan bahwa siswa aktif kembali tidak memperoleh rekening baru pada tahun sama (unique constraint) dan settlement tetap final. | Pilihan paling kecil secara kode, tetapi konsekuensi operasional harus diterima secara eksplisit. |

Semua alternatif harus mempertahankan transaksi atomic, otorisasi Admin/Wali, audit settlement dan rekening, serta riwayat kelas. Audit ini tidak menemukan bukti bahwa audit settlement dapat commit terpisah dari perubahan account; keduanya memakai UOW yang sama.

## 6. Rancangan regression test konkuren

Jalankan kelak hanya pada PostgreSQL disposable dengan schema/migration dan fixture terisolasi. Hindari `time.Sleep` sebagai sinkronisasi. Pilihan yang disarankan: test-only repository/UOW decorator dengan channel barrier pada call tertentu, atau test harness dua koneksi yang memperoleh dan melepas row lock eksplisit; gunakan query observasi `pg_stat_activity`/`pg_blocking_pids` dengan deadline untuk memastikan sesi benar-benar menunggu pada lock yang ditargetkan. Jangan menambahkan test hook produksi hanya demi test tanpa review.

| Kasus | Sinkronisasi deterministik | Assertion minimum |
|---|---|---|
| Reaktivasi lebih dahulu | Barrier setelah R commit, baru panggil S. | R sukses; S ditolak karena status bukan `LEFT`; student `ACTIVE`, account `OPEN`, tidak ada settlement/audit settlement baru. |
| Settlement mendapat keputusan status dahulu | Tahan S setelah year/account lock dan setelah pembacaan student `LEFT` melalui decorator/barrier; mulai R dan pastikan R tertahan pada year lock; lepaskan S. | Rekam apakah kedua operasi sukses sesuai keputusan kontrak. Verifikasi student status, account status, tepat satu settlement, histori class, audit terkait. Jika hasil `ACTIVE + SETTLED` dilarang, test harus gagal sebelum perbaikan dan lulus setelah solusi. |
| Dua transaksi pada status siswa | Ambil student lock R dan year/account lock S secara terkontrol; amati blocking dengan PostgreSQL, lalu lepaskan tiap gate. | Tidak ada hang; tidak ada deadlock tak tertangani; hasil memenuhi aturan serial yang disepakati; account/settlement/audit konsisten. |
| Reactivate vs setor/tarik | Gunakan gate untuk membuat urutan student/year yang berpotensi membentuk siklus; pantau blocker dan SQLSTATE. | Setelah lock-order fix, tidak ada `40P01` pada skenario yang diulang; tepat satu hasil serial yang sah; tidak ada transaksi keuangan untuk student yang tidak aktif pada titik tulis. |
| Rollback/error pada masing-masing sisi | Paksa kegagalan terkontrol sebelum commit (mis. decorator yang mengembalikan error repository) setelah lock/write relevan. | Tidak ada perubahan parsial student/history/account/settlement/audit; operasi lain dapat melanjutkan; saldo/ledger tidak berubah di luar request sah. |

Setiap test perlu memakai context timeout sebagai pagar hang, tetapi timeout bukan sinkronisasi utama. Periksa row state dan hitungan record setelah kedua goroutine selesai, termasuk bahwa saldo serta transaksi ledger tidak berubah tanpa operasi yang diotorisasi. Test tidak boleh menerima error deadlock sebagai hasil lulus kecuali kontrak secara eksplisit menetapkan retry/penolakan yang dapat diprediksi; idealnya lock order mencegah siklus.

## 7. Rekomendasi perubahan terkecil yang aman

1. **Putuskan invariant produk terlebih dahulu:** apakah reaktivasi diizinkan jika rekening pada tahun itu telah `SETTLED` (termasuk settlement penyerahan belum direkonsiliasi)? Jika tidak, tetapkan perilaku penolakan/approval dan tahun yang diperiksa. Jika ya, jelaskan konsekuensi tidak adanya rekening kedua dan status settlement final.
2. Jika sistem harus mencegah transisi status/keuangan bersamaan, susun satu urutan lock global untuk `Leave`, `Reactivate`, setor/tarik, dan settlement. Kemudian validasi status student sambil memegang row lock pada titik commit yang sesuai. Jangan menambahkan row lock ke settlement saja.
3. Tambahkan test PostgreSQL disposable deterministik untuk dua race: Reactivate–SettleStudentLeaving dan Reactivate–Deposit/Withdrawal. Sertakan rollback dan audit checks. Jalankan suite integrasi serta `go test -race` yang relevan setelah implementasi disetujui.
4. Periksa kebutuhan retry SQLSTATE `40P01` berdasarkan pola retry yang benar-benar tersedia; jangan menambah retry otomatis sebelum memastikan operasi/idempotency dan batas transaksi aman.

Alternatif kode terkecil apabila keputusan produk adalah “settlement mengunci lifecycle siswa untuk tahun tersebut” kemungkinan berupa guard reaktivasi terhadap settlement terkait, tetapi guard itu tetap harus diambil secara terkunci dan dalam urutan global yang bebas deadlock. Detailnya membutuhkan desain implementasi setelah kontrak disetujui.

## 8. Keputusan untuk supervisor/pemilik

- Apakah siswa boleh diaktifkan kembali setelah settlement untuk tahun yang sama telah dibuat?
- Apakah status penyerahan uang (`belum diserahkan`, `sebagian`, `selesai`) memengaruhi kelayakan reaktivasi?
- Jika reaktivasi diizinkan, apakah siswa tetap tanpa rekening tabungan pada tahun tersebut karena settlement final dan unique constraint satu rekening/tahun?
- Haruskah tanggal efektif reaktivasi dibandingkan dengan tanggal settlement/penyerahan, dan aturan tanggal mana yang otoritatif?
- Apakah scope MQ berikutnya boleh mengubah lock order lintas `Leave`, `Reactivate`, transaksi biasa, dan settlement sekaligus, dengan test PostgreSQL konkuren wajib?

## 9. Batas verifikasi

- Audit hanya meninjau source dan dokumentasi pada baseline yang ditentukan; tidak menjalankan tes, mengakses PostgreSQL, atau membuktikan jadwal konkurensi saat runtime.
- Perilaku isolation dijelaskan berdasarkan `BeginTx(ctx, nil)` dan default PostgreSQL `READ COMMITTED`; konfigurasi deployment dapat berbeda dan belum diperiksa.
- Tidak ada data development/production yang diperiksa; laporan ini tidak menyatakan ada state bisnis yang sudah tidak konsisten.
- Bukti deadlock adalah analisis lock graph dari call-site yang ada, bukan deadlock yang direproduksi.
- Belum diputuskan apakah reaktivasi setelah settlement melanggar aturan bisnis.
