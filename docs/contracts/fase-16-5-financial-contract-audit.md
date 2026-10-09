# FASE 16.5 — FINANCIAL CONTRACT AUDIT

Status: AUDIT COMPLETE
Phase: 16.5
Parent: FASE 16 — CONTRACT HARDENING
Scope: Money, balance, deposit, withdrawal, transaction mutation, cancellation, settlement, settlement reopening, financial audit consistency
Source baseline: commit `1a680c81d71cb6a8dbc6485612b1677a528283c4`
Audit basis: source snapshot `tabungan-siswa.tar(20261009-002538).gz`

---

# 1. Tujuan

FASE 16.5 melakukan audit terhadap kontrak keuangan aplikasi Tabungan Siswa.

Audit memeriksa konsistensi antara:

1. Domain `Money`.
2. Domain `Balance`.
3. Domain `Transaction`.
4. Domain `SavingsAccount`.
5. Domain `SavingsSettlement`.
6. Transaction service.
7. Savings account service.
8. Settlement service.
9. PostgreSQL repository.
10. Database constraints.
11. Unit of Work dan transaction boundary.
12. Audit log.
13. Lifecycle tahun ajaran.
14. Contract audit yang sudah disusun pada FASE 16.1 sampai FASE 16.4.

FASE 16.5 bukan fase implementasi.

Tidak boleh dilakukan dalam fase ini:

- Mengubah source code.
- Mengubah migration.
- Mengubah struktur tabel.
- Mengubah API service.
- Mengubah business rule yang belum diputuskan.
- Menambahkan workflow keuangan baru.
- Menghapus data transaksi atau settlement.
- Menganggap implementasi saat ini otomatis benar hanya karena unit test lulus.

Temuan harus diklasifikasikan dan diberikan ownership yang jelas.

---

# 2. Aturan Audit

## 2.1 Klasifikasi temuan

Gunakan klasifikasi berikut:

```text
MATCH
GAP - IMPLEMENTATION
GAP - CONTRACT DECISION
CONFLICT
DEFERRED
```

### MATCH

Implementasi yang ditemukan konsisten dengan kontrak yang telah dikunci.

### GAP - IMPLEMENTATION

Kontrak sudah jelas, tetapi implementasi belum memenuhi kontrak.

### GAP - CONTRACT DECISION

Ada aturan bisnis yang belum diputuskan. Perubahan implementasi harus menunggu keputusan.

### CONFLICT

Implementasi atau database bertentangan dengan kontrak yang sudah dikunci.

### DEFERRED

Temuan valid, tetapi ownership penyelesaiannya berada pada fase lain yang telah ditentukan.

## 2.2 Prinsip pembuktian

Setiap temuan harus menyebutkan:

1. ID temuan.
2. Status.
3. Source yang diperiksa.
4. Perilaku aktual.
5. Invariant yang harus dipenuhi.
6. Dampak.
7. Keputusan yang dibutuhkan, jika ada.
8. Fase pemilik penyelesaian.
9. Batas pekerjaan fase ini.

Temuan tidak boleh menyatakan suatu perilaku sebagai bug hanya karena perilaku tersebut belum tertulis. Periksa terlebih dahulu kontrak yang telah dikunci.

## 2.3 Batas klaim

Audit source code tidak membuktikan bahwa semua kondisi concurrency, seluruh jalur HTTP, dan semua kemungkinan kegagalan database sudah diuji.

Pernyataan dalam dokumen ini dibatasi pada source dan migration yang benar-benar diperiksa.

---

# 3. Source Inventory

## 3.1 Domain

Source utama:

```text
internal/domain/money.go
internal/domain/balance.go
internal/domain/transaction.go
internal/domain/savings_account.go
internal/domain/savings_settlement.go
internal/domain/academic_year.go
internal/domain/types.go
```

Test terkait:

```text
internal/domain/money_test.go
internal/domain/balance_test.go
internal/domain/transaction_test.go
internal/domain/savings_settlement_test.go
internal/domain/savings_account_test.go
```

## 3.2 Service

Source utama:

```text
internal/service/transaction_service.go
internal/service/savings_account_service.go
internal/service/settlement_service.go
internal/service/academic_year_service.go
internal/service/audit_service.go
```

Test terkait:

```text
internal/service/transaction_service_unit_test.go
internal/service/settlement_service_unit_test.go
internal/service/savings_account_service_unit_test.go
internal/service/service_transaction_integration_test.go
internal/service/audit_service_unit_test.go
```

## 3.3 Repository

Source utama:

```text
internal/repository/postgres/transaction_repository.go
internal/repository/postgres/savings_account_repository.go
internal/repository/postgres/settlement_repository.go
internal/repository/postgres/audit_repository.go
internal/repository/postgres/unit_of_work.go
```

## 3.4 Migration

Migration yang relevan:

```text
migrations/000001_init_schema.up.sql
migrations/000002_align_domain_contract.up.sql
```

Migration lain tetap menjadi bagian dari baseline database, tetapi tidak menjadi fokus utama audit keuangan ini.

## 3.5 Hasil inventory

Status:

```text
MATCH
```

Alasan:

Source domain, service, repository, test, dan migration yang diperlukan untuk audit tersedia pada snapshot.

Status ini hanya menyatakan kelengkapan inventory yang diperlukan untuk audit, bukan bahwa seluruh kontrak keuangan sudah benar.

---

# 4. Model Keuangan Saat Ini

Alur keuangan utama:

```text
SAVINGS ACCOUNT
       |
       v
  TRANSACTION
       |
       +----------------+
       |                |
       v                v
    DEPOSIT          WITHDRAWAL
       |                |
       +--------+-------+
                |
                v
       ACTIVE TRANSACTIONS
                |
                v
        CALCULATE BALANCE
                |
                v
           SETTLEMENT
                |
                v
        ACCOUNT = SETTLED
```

Pembatalan transaksi:

```text
ACTIVE TRANSACTION
        |
        v
      CANCEL
        |
        v
CANCELLED TRANSACTION
        |
        v
EXCLUDED FROM BALANCE
```

Pembukaan kembali settlement:

```text
COMPLETED SETTLEMENT
        |
        v
      REOPEN
        |
        +----------------------+
        |                      |
        v                      v
SETTLEMENT SUPERSEDED    ACCOUNT = OPEN
```

Saldo bukan kolom tersimpan pada tabel `savings_accounts`.

Saldo dihitung dari transaksi aktif melalui `domain.CalculateBalance`.

Settlement menyimpan nominal saldo pada saat settlement dilakukan.

---

# 5. Kontrak Uang

## 5.1 Representasi domain

Domain menggunakan:

```go
type Money int64
```

Representasi ini merupakan bilangan bulat bertanda 64-bit.

Domain menyediakan:

```text
IsNegative()
IsZero()
IsPositive()
Add()
Subtract()
```

`Subtract` menolak hasil negatif melalui `ErrNegativeBalance`.

## 5.2 Kontrak database

Tabel `transactions` menggunakan:

```sql
amount NUMERIC(15, 2) NOT NULL
```

Tabel `savings_settlements` menggunakan:

```sql
amount NUMERIC(15, 2) NOT NULL
```

Migration `000002_align_domain_contract` menambahkan constraint:

```sql
CHECK (amount = trunc(amount, 0))
```

Constraint tersebut membatasi nominal tersimpan menjadi bilangan bulat tanpa bagian pecahan.

Repository settlement juga menolak nilai pecahan ketika membaca nominal settlement.

## 5.3 Temuan FIN-MONEY-01: Presisi uang

Status:

```text
MATCH
```

Perilaku aktual:

- Domain menggunakan `int64`.
- Database menggunakan `NUMERIC(15,2)`.
- Database membatasi nominal agar tidak mempunyai bagian pecahan.
- Domain dan database dengan demikian menggunakan nominal bilangan bulat untuk transaksi dan settlement.

Kontrak yang dicatat:

```text
MONEY
- Bilangan bulat.
- Tidak menerima pecahan.
- Nominal transaksi harus positif.
- Nominal settlement tidak boleh negatif.
```

Batas interpretasi:

Dokumen ini tidak menetapkan apakah satu unit `Money` merupakan rupiah, sen, atau unit mata uang lain. Penamaan satuan mata uang harus konsisten pada kontrak produk dan presentation layer.

Tidak boleh mengubah representasi menjadi pecahan hanya karena tipe database `NUMERIC` mendukung pecahan.

## 5.4 Temuan FIN-MONEY-02: Batas representasi nominal

Status:

```text
GAP - IMPLEMENTATION
```

Perilaku aktual:

`Money.Add` melakukan penjumlahan integer langsung.

`Money.Subtract` melakukan pengurangan integer langsung.

Tidak ada pemeriksaan eksplisit overflow aritmetika `int64` pada kedua operasi tersebut.

Dampak:

Overflow pada nilai ekstrem dapat menghasilkan hasil aritmetika yang tidak mewakili nominal matematis sebenarnya.

Ownership:

```text
FASE 16.8 — Resolve contract-level gaps
```

Keputusan implementasi harus mempertimbangkan batas nominal bisnis dan konsistensi domain dengan batas maksimum database.

Batas pekerjaan FASE 16.5:

Tidak mengubah tipe `Money`, batas nominal, atau operasi aritmetika.

---

# 6. Kontrak Saldo

## 6.1 Perhitungan saldo

`domain.CalculateBalance` memproses transaksi dan:

1. Mengabaikan transaksi yang statusnya bukan `ACTIVE`.
2. Menambahkan nominal transaksi `DEPOSIT`.
3. Mengurangi nominal transaksi `WITHDRAWAL`.
4. Mengembalikan `ErrNegativeBalance` ketika saldo hasil perhitungan menjadi negatif.
5. Menghasilkan `Balance` dengan nominal akhir yang tidak negatif.

Repository `ListActiveBySavingsAccount` mengurutkan transaksi secara kronologis:

```sql
ORDER BY transaction_date ASC, created_at ASC, id ASC
```

Dengan demikian, urutan transaksi memengaruhi validitas saldo historis.

## 6.2 Kontrak saldo

Kontrak yang dituju:

```text
ACTIVE DEPOSIT
    → MENAMBAH SALDO

ACTIVE WITHDRAWAL
    → MENGURANGI SALDO

CANCELLED TRANSACTION
    → TIDAK MEMENGARUHI SALDO

SALDO NEGATIF
    → DITOLAK
```

Saldo harus dihitung berdasarkan transaksi aktif yang tersimpan, bukan berdasarkan total nominal settlement atau hasil perhitungan sementara yang tidak dipersistenkan.

## 6.3 Temuan FIN-BALANCE-01: Saldo diturunkan dari transaksi aktif

Status:

```text
MATCH
```

Perilaku aktual:

Saldo dihitung dari daftar transaksi aktif. Transaksi berstatus `CANCELLED` tidak ikut dalam perhitungan.

Kontrak:

```text
BALANCE = CALCULATE(ACTIVE TRANSACTIONS)
```

Tidak ada kolom saldo permanen pada tabel `savings_accounts` yang perlu disinkronkan oleh setiap mutasi transaksi.

Batasan:

Kebenaran saldo tetap bergantung pada validitas semua transaksi aktif dan konsistensi urutan perhitungan.

## 6.4 Temuan FIN-BALANCE-02: Deposit tidak menghitung ulang saldo historis

Status:

```text
GAP - IMPLEMENTATION
```

Perilaku aktual:

`transactionService.Deposit`:

1. Mengunci rekening.
2. Memeriksa otorisasi.
3. Memastikan rekening berstatus `OPEN`.
4. Membuat transaksi deposit.
5. Menyimpan transaksi.
6. Menulis audit log.

Service tidak memanggil `CalculateBalance` terhadap seluruh transaksi aktif sebelum menyelesaikan deposit.

Contoh kondisi yang perlu ditangani:

```text
10 Januari: WITHDRAWAL 100
20 Januari: DEPOSIT 200
```

Jika transaksi penarikan 10 Januari sudah tersimpan tanpa saldo pendahulu yang cukup, penambahan deposit 20 Januari tidak memperbaiki saldo historis pada 10 Januari.

Saldo historis tetap dapat negatif saat seluruh transaksi dihitung dalam urutan tanggal.

Dampak:

- Deposit dapat tersimpan tanpa mendeteksi saldo historis yang tidak valid.
- Pembacaan saldo berikutnya dapat gagal ketika `CalculateBalance` menemukan saldo negatif.
- Perilaku deposit berbeda dengan validasi yang dilakukan pada withdrawal dan edit.

Ownership:

```text
FASE 16.8 — Resolve contract-level gaps
```

Kontrak yang perlu ditegakkan adalah bahwa transaksi aktif yang tersimpan tidak boleh menghasilkan saldo historis negatif.

Batas pekerjaan FASE 16.5:

Tidak mengubah service deposit atau menambahkan validasi saldo.

## 6.5 Temuan FIN-BALANCE-03: Urutan transaksi dan tanggal efektif

Status:

```text
GAP - CONTRACT DECISION
```

Perilaku aktual:

Saldo dihitung mengikuti urutan `transaction_date`, kemudian `created_at`, lalu `id`.

Konsekuensi:

Transaksi yang dicatat belakangan tetapi memiliki tanggal transaksi lebih awal dapat mengubah validitas saldo historis.

Keputusan yang dibutuhkan:

1. Apakah tanggal transaksi merupakan tanggal efektif yang menentukan urutan saldo?
2. Apakah transaksi bertanggal lampau diperbolehkan?
3. Apakah penarikan harus valid terhadap saldo pada tanggal transaksi, bukan hanya terhadap saldo akhir?
4. Apakah saldo historis negatif selalu dilarang?

Audit tidak menganggap tanggal transaksi harus sama dengan tanggal input karena aturan tersebut belum dinyatakan sebagai kontrak.

Ownership:

```text
FASE 16.7 — Gap Resolution Decision
```

Batas pekerjaan FASE 16.5:

Tidak mengubah algoritma pengurutan atau kebijakan tanggal efektif.

---

# 7. Deposit

## 7.1 Perilaku aktual

`transactionService.Deposit` melakukan:

```text
AUTHORIZE ROLE
    ↓
LOCK SAVINGS ACCOUNT
    ↓
AUTHORIZE ACCOUNT SCOPE
    ↓
REQUIRE ACCOUNT OPEN
    ↓
CREATE DEPOSIT
    ↓
PERSIST TRANSACTION
    ↓
WRITE AUDIT LOG
    ↓
COMMIT
```

Role yang dapat membuat transaksi adalah:

```text
ADMIN
WALI_KELAS
```

Wali kelas harus memenuhi scope rekening.

Pembuatan transaksi dan audit log berada dalam transaction boundary yang sama ketika service memiliki Unit of Work sendiri.

## 7.2 Temuan FIN-DEPOSIT-01: Nominal deposit

Status:

```text
MATCH
```

Perilaku aktual:

Pembuatan deposit menggunakan `domain.NewDeposit`, yang meneruskan validasi ke `domain.NewTransaction`.

`NewTransaction` menolak nominal yang tidak positif.

Database juga memiliki constraint:

```sql
CHECK (amount > 0)
```

Kontrak:

```text
DEPOSIT AMOUNT > 0
```

## 7.3 Temuan FIN-DEPOSIT-02: Rekening settled

Status:

```text
MATCH
```

Perilaku aktual:

Deposit menolak rekening yang bukan `OPEN`.

Kontrak:

```text
SETTLED ACCOUNT
    → TIDAK BOLEH MENERIMA DEPOSIT BARU
```

Temuan ini tidak menyelesaikan kebijakan pembukaan kembali settlement. Kebijakan tersebut dibahas pada bagian settlement.

## 7.4 Temuan FIN-DEPOSIT-03: Konsistensi audit

Status:

```text
MATCH
```

Perilaku aktual:

Deposit menulis audit log setelah transaksi disimpan. Kegagalan pembuatan audit menyebabkan operasi gagal dan, pada Unit of Work yang dimiliki service, transaksi database di-rollback.

Kontrak:

```text
TRANSACTION PERSISTENCE
+
AUDIT PERSISTENCE
=
SATU ATOMIC OPERATION
```

Batasan:

Jaminan ini bergantung pada penggunaan transaction boundary dan repository yang sama.

---

# 8. Withdrawal

## 8.1 Perilaku aktual

`transactionService.Withdrawal` melakukan:

```text
AUTHORIZE ROLE
    ↓
LOCK SAVINGS ACCOUNT
    ↓
AUTHORIZE ACCOUNT SCOPE
    ↓
REQUIRE ACCOUNT OPEN
    ↓
LOAD ACTIVE TRANSACTIONS
    ↓
VALIDATE EXISTING BALANCE
    ↓
CREATE WITHDRAWAL
    ↓
VALIDATE BALANCE WITH WITHDRAWAL
    ↓
PERSIST TRANSACTION
    ↓
WRITE AUDIT LOG
    ↓
COMMIT
```

## 8.2 Temuan FIN-WITHDRAWAL-01: Nominal withdrawal

Status:

```text
MATCH
```

Perilaku aktual:

Withdrawal menggunakan `domain.NewWithdrawal`, yang memvalidasi nominal positif melalui `domain.NewTransaction`.

Database juga membatasi nominal agar lebih besar dari nol.

Kontrak:

```text
WITHDRAWAL AMOUNT > 0
```

## 8.3 Temuan FIN-WITHDRAWAL-02: Pencegahan saldo negatif

Status:

```text
MATCH
```

Perilaku aktual:

Withdrawal menghitung saldo transaksi aktif sebelum dan sesudah transaksi withdrawal ditambahkan.

Jika perhitungan menghasilkan saldo negatif, transaksi baru tidak dilanjutkan ke persistence.

Kontrak:

```text
WITHDRAWAL
    → SALDO HARUS VALID
    → SALDO TIDAK BOLEH NEGATIF
```

Batasan:

Validasi ini tetap dipengaruhi urutan tanggal transaksi. Temuan FIN-BALANCE-03 mengatur keputusan terkait transaksi bertanggal lampau.

## 8.4 Temuan FIN-WITHDRAWAL-03: Concurrency

Status:

```text
MATCH
```

Perilaku aktual:

Service mengambil lock pada rekening melalui `GetByIDForUpdate` sebelum memeriksa dan membuat withdrawal.

Kontrak concurrency:

Mutasi saldo pada rekening yang sama harus diserialisasi melalui lock rekening agar dua withdrawal yang berjalan bersamaan tidak sama-sama menggunakan saldo awal yang sudah kedaluwarsa.

Batasan:

Temuan ini mengonfirmasi penggunaan mekanisme lock dalam source yang diperiksa. Audit ini tidak mengklaim bahwa seluruh skenario concurrency telah dibuktikan oleh stress test.

---

# 9. Edit Transaksi

## 9.1 Perilaku aktual

`transactionService.Edit` melakukan:

```text
LOCK ACCOUNT
    ↓
AUTHORIZE SCOPE
    ↓
REQUIRE ACCOUNT OPEN
    ↓
REQUIRE TRANSACTION ACTIVE
    ↓
RE-READ TRANSACTION
    ↓
REPLACE AMOUNT AND DATE
    ↓
RECALCULATE BALANCE
    ↓
PERSIST UPDATE
    ↓
WRITE AUDIT LOG
    ↓
COMMIT
```

Service menghitung ulang saldo sebelum menyimpan transaksi yang diubah.

## 9.2 Temuan FIN-EDIT-01: Validasi nominal pada service

Status:

```text
GAP - IMPLEMENTATION
```

Perilaku aktual:

Service mengisi `updated.Amount` langsung dari input.

Service tidak secara eksplisit memanggil validasi nominal positif sebelum perhitungan saldo dan persistence.

Constraint database tetap mensyaratkan nominal lebih besar dari nol, tetapi constraint database tidak menggantikan validasi domain pada batas service.

Dampak:

- Input nominal tidak valid tidak ditolak secara konsisten melalui kontrak domain.
- Kesalahan dapat baru muncul ketika database menerima update.
- Perilaku edit tidak sejelas pembuatan transaksi baru yang menggunakan `NewTransaction`.

Ownership:

```text
FASE 16.8 — Resolve contract-level gaps
```

Kontrak:

```text
EDITED TRANSACTION AMOUNT > 0
```

## 9.3 Temuan FIN-EDIT-02: Konsistensi saldo setelah edit

Status:

```text
MATCH
```

Perilaku aktual:

Service mengganti transaksi yang diedit di daftar transaksi aktif, menghitung ulang saldo, lalu menyimpan perubahan.

Jika saldo hasil perhitungan negatif, service menolak perubahan sebelum persistence.

Kontrak:

```text
EDIT TRANSACTION
    → RECOMPUTE BALANCE
    → REJECT INVALID BALANCE
    → PERSIST ONLY IF VALID
```

## 9.4 Temuan FIN-EDIT-03: Tanggal transaksi hasil edit

Status:

```text
DEFERRED
```

Perilaku aktual:

Service menerima tanggal transaksi dari input edit dan memperbarui `TransactionDate`.

Aturan batas tanggal transaksi terhadap tahun ajaran sudah dicatat sebagai `GAP-TRANSACTION-DATE-01` pada audit domain dan lifecycle sebelumnya.

Ownership:

```text
FASE 21 — TRANSACTION ENGINE
```

Batas pekerjaan:

Jangan menduplikasi temuan batas tanggal sebagai gap baru dalam FASE 16.5. Implementasi harus mengikuti keputusan kontrak yang akan ditegakkan pada fase pemiliknya.

---

# 10. Pembatalan Transaksi

## 10.1 Perilaku aktual

`transactionService.Cancel` melakukan:

```text
LOAD TRANSACTION
    ↓
LOCK RELATED ACCOUNT
    ↓
AUTHORIZE ACCOUNT SCOPE
    ↓
RE-READ TRANSACTION
    ↓
TRANSITION ACTIVE → CANCELLED
    ↓
PERSIST UPDATE
    ↓
WRITE AUDIT LOG
    ↓
COMMIT
```

Transaksi tidak dihapus dari database.

Perubahan status transaksi menjadi `CANCELLED` membuatnya dikeluarkan dari perhitungan saldo.

## 10.2 Temuan FIN-CANCEL-01: Pembatalan tidak menghapus transaksi

Status:

```text
MATCH
```

Perilaku aktual:

Pembatalan menggunakan perubahan status transaksi. Repository memperbarui row transaksi yang sudah ada.

Kontrak:

```text
CANCEL
    → RETAIN TRANSACTION
    → STATUS = CANCELLED
    → PRESERVE AUDIT HISTORY
```

Tidak boleh mengganti mekanisme ini dengan penghapusan fisik transaksi.

## 10.3 Temuan FIN-CANCEL-02: Rekening settled dapat terkena pembatalan

Status:

```text
GAP - IMPLEMENTATION
```

Perilaku aktual:

`Cancel` mengambil lock rekening dan memeriksa otorisasi scope. Namun, tidak melakukan pemeriksaan eksplisit `account.IsOpen()` sebelum mengubah status transaksi.

Berbeda dengan `Deposit`, `Withdrawal`, dan `Edit`, alur `Cancel` tidak secara eksplisit menolak rekening `SETTLED`.

Dampak:

- Transaksi yang mendasari settlement dapat berubah status setelah settlement selesai.
- Nominal settlement yang tersimpan dapat tidak lagi sama dengan saldo hasil perhitungan transaksi aktif.
- Riwayat settlement dapat merepresentasikan saldo yang berbeda dari saldo transaksi terkini.

Ownership:

```text
FASE 16.8 — Resolve contract-level gaps
```

Kontrak yang harus dipertahankan:

```text
SETTLED ACCOUNT
    → TRANSACTION MUTATION MUST BE RESTRICTED
```

Jika bisnis membutuhkan koreksi setelah settlement, koreksi harus melalui workflow yang secara eksplisit mengatur pembukaan kembali dan pencatatan riwayatnya.

## 10.4 Temuan FIN-CANCEL-03: Pembatalan dan validitas saldo

Status:

```text
GAP - IMPLEMENTATION
```

Perilaku aktual:

Service melakukan transisi transaksi ke `CANCELLED`, lalu menyimpan transaksi dan audit log. Service tidak menjalankan ulang `CalculateBalance` setelah pembatalan.

Dampak:

Pembatalan suatu deposit dapat mengubah validitas saldo historis, terutama apabila ada withdrawal setelah tanggal deposit tersebut.

Kontrak yang dituju:

```text
TRANSACTION CANCELLATION
    → RESULTING ACTIVE TRANSACTION HISTORY
    → MUST REMAIN FINANCIALLY VALID
```

Ownership:

```text
FASE 16.8 — Resolve contract-level gaps
```

Batas pekerjaan:

Tidak menambahkan validasi saldo pada service pembatalan dalam audit ini.

---

# 11. Kontrak Status Transaksi

Domain mendefinisikan:

```text
ACTIVE
CANCELLED
```

Transisi pembatalan yang valid:

```text
ACTIVE
   |
   v
CANCELLED
```

Domain menolak pembatalan ulang transaksi yang sudah `CANCELLED`.

Database mengizinkan nilai status yang telah ditentukan melalui constraint.

## 11.1 Temuan FIN-STATUS-01

Status:

```text
MATCH
```

Perilaku aktual:

Status transaksi dibatasi pada `ACTIVE` dan `CANCELLED`.

Pembatalan dilakukan melalui method domain, bukan dengan menghapus row.

## 11.2 Aturan mutasi

Kontrak yang harus konsisten:

- Transaksi aktif dapat diedit selama rekening memenuhi syarat.
- Transaksi aktif dapat dibatalkan selama rekening memenuhi syarat.
- Transaksi yang dibatalkan tidak dapat diedit sebagai transaksi aktif.
- Transaksi yang dibatalkan tidak dapat dibatalkan untuk kedua kalinya.
- Perubahan nominal atau tanggal harus mempertahankan validitas saldo.
- Setiap perubahan keuangan yang dipersistenkan harus memiliki audit log.

---

# 12. Kontrak Savings Account

## 12.1 State account

Domain mendefinisikan:

```text
OPEN
SETTLED
```

Transisi utama:

```text
OPEN
  |
  v
SETTLED
  |
  v
OPEN
```

Transisi terakhir hanya melalui alur reopen settlement yang dikontrol.

Database mengharuskan:

```text
OPEN
    → settled_at IS NULL

SETTLED
    → settled_at IS NOT NULL
```

## 12.2 Temuan FIN-ACCOUNT-01: Konsistensi status rekening

Status:

```text
MATCH
```

Perilaku aktual:

Domain `Settle` menolak rekening yang sudah settled.

Domain `Reopen` menolak rekening yang bukan settled.

Database menjaga konsistensi antara `status` dan `settled_at`.

## 12.3 Temuan FIN-ACCOUNT-02: Satu rekening per siswa per tahun ajaran

Status:

```text
MATCH
```

Perilaku aktual:

Database memiliki unique constraint:

```sql
UNIQUE (student_id, academic_year_id)
```

Kontrak:

```text
ONE STUDENT
    + ONE ACADEMIC YEAR
    → AT MOST ONE SAVINGS ACCOUNT
```

Constraint ini mencegah rekening ganda untuk kombinasi siswa dan tahun ajaran yang sama.

## 12.4 Temuan FIN-ACCOUNT-03: Pembuatan rekening pada tahun ajaran yang tidak sesuai

Status:

```text
GAP - CONTRACT DECISION
```

Perilaku aktual:

`SavingsAccountService.Create` memvalidasi role, ID siswa, scope wali kelas, dan membuat rekening. Pada jalur yang diperiksa, service tidak memeriksa status tahun ajaran untuk memastikan tahun ajaran masih `OPEN`.

Keputusan yang dibutuhkan:

1. Apakah rekening hanya boleh dibuat pada tahun ajaran `OPEN`?
2. Apakah rekening boleh dibuat pada tahun ajaran `CLOSING`?
3. Apakah rekening boleh dibuat pada tahun ajaran `CLOSED`?
4. Apakah status siswa `LEFT` memengaruhi kelayakan pembuatan rekening?

Pertanyaan tersebut juga beririsan dengan temuan lifecycle yang telah dicatat sebelumnya.

Ownership:

```text
FASE 16.7 — Gap Resolution Decision
```

Batas pekerjaan:

Tidak mengubah aturan pembuatan rekening sebelum keputusan kontrak dibuat.

---

# 13. Settlement

## 13.1 Jenis settlement

Domain mendefinisikan:

```text
REGULAR_YEAR_END
STUDENT_LEAVING
```

Status settlement:

```text
COMPLETED
SUPERSEDED
```

Nominal settlement tidak boleh negatif.

## 13.2 Settlement akhir tahun

`SettleYearEnd`:

1. Memvalidasi role.
2. Mengunci rekening.
3. Memeriksa otorisasi scope.
4. Memastikan rekening masih `OPEN`.
5. Mengambil tahun ajaran rekening.
6. Memastikan tahun ajaran berada pada status `CLOSING`.
7. Mengambil transaksi aktif.
8. Menghitung saldo.
9. Membuat settlement `REGULAR_YEAR_END`.
10. Mengubah rekening menjadi `SETTLED`.
11. Menyimpan settlement.
12. Menulis audit settlement dan rekening.
13. Melakukan commit.

## 13.3 Settlement siswa keluar

`SettleStudentLeaving`:

1. Memvalidasi role.
2. Mengunci rekening.
3. Memeriksa otorisasi scope.
4. Memastikan rekening masih `OPEN`.
5. Memuat siswa.
6. Memastikan status siswa `LEFT`.
7. Mengambil transaksi aktif.
8. Menghitung saldo.
9. Membuat settlement `STUDENT_LEAVING`.
10. Mengubah rekening menjadi `SETTLED`.
11. Menyimpan settlement.
12. Menulis audit settlement dan rekening.
13. Melakukan commit.

## 13.4 Temuan FIN-SETTLEMENT-01: Snapshot nominal settlement

Status:

```text
MATCH
```

Perilaku aktual:

Nominal settlement diambil dari hasil `CalculateBalance` pada transaksi aktif rekening.

Nominal settlement disimpan pada tabel `savings_settlements`.

Kontrak:

```text
SETTLEMENT AMOUNT
    = BALANCE OF ACTIVE TRANSACTIONS
      AT SETTLEMENT EXECUTION
```

Settlement bukan saldo dinamis yang otomatis berubah ketika transaksi berubah.

Karena itu, mutasi transaksi setelah settlement harus dibatasi oleh lifecycle rekening dan aturan koreksi.

## 13.5 Temuan FIN-SETTLEMENT-02: Settlement akhir tahun membutuhkan CLOSING

Status:

```text
MATCH
```

Perilaku aktual:

`SettleYearEnd` menolak tahun ajaran yang bukan `CLOSING`.

Kontrak:

```text
ACADEMIC YEAR CLOSING
    → ELIGIBLE FOR YEAR-END SETTLEMENT
```

Hal ini tidak berarti bahwa seluruh rekening harus sudah settled sebelum tahun ajaran dapat ditutup. Persyaratan global tersebut masih membutuhkan pemeriksaan dan keputusan tersendiri.

## 13.6 Temuan FIN-SETTLEMENT-03: Settlement siswa keluar

Status:

```text
MATCH
```

Perilaku aktual:

`SettleStudentLeaving` hanya berjalan jika status siswa `LEFT` dan rekening masih `OPEN`.

Kontrak:

```text
STUDENT LEFT
    + ACCOUNT OPEN
    → ELIGIBLE FOR STUDENT-LEAVING SETTLEMENT
```

Aturan apakah settlement siswa keluar boleh dilakukan ketika tahun ajaran sudah `CLOSED` belum diselesaikan oleh temuan ini.

## 13.7 Temuan FIN-SETTLEMENT-04: Atomicity settlement

Status:

```text
MATCH
```

Perilaku aktual:

Settlement, perubahan status rekening, dan audit log dikerjakan dalam transaction boundary yang sama pada jalur Unit of Work milik service.

Jika persistence settlement atau audit gagal, operasi tidak boleh dianggap berhasil.

Kontrak:

```text
SETTLEMENT
+
ACCOUNT STATE TRANSITION
+
AUDIT LOG
=
ONE ATOMIC OPERATION
```

## 13.8 Temuan FIN-SETTLEMENT-05: Rekening settled dan transaksi aktif

Status:

```text
GAP - IMPLEMENTATION
```

Temuan ini terkait dengan FIN-CANCEL-02.

Service deposit, withdrawal, dan edit memeriksa status rekening. Service cancel tidak melakukan pemeriksaan yang sama.

Dampak:

Settlement dapat tetap berstatus `COMPLETED` meskipun transaksi yang menjadi dasar nominal settlement kemudian dibatalkan.

Ownership:

```text
FASE 16.8 — Resolve contract-level gaps
```

---

# 14. Reopen Settlement

## 14.1 Perilaku aktual

`SettlementService.Reopen`:

1. Membatasi role menjadi `ADMIN`.
2. Memuat settlement.
3. Mengunci rekening terkait.
4. Memastikan rekening berstatus `SETTLED`.
5. Memuat settlement `COMPLETED` terkini.
6. Memastikan settlement yang diminta adalah settlement terkini.
7. Mengubah settlement menjadi `SUPERSEDED`.
8. Membuka kembali rekening.
9. Menyimpan perubahan settlement.
10. Menyimpan perubahan rekening.
11. Menulis audit untuk settlement dan rekening.
12. Melakukan commit.

## 14.2 Temuan FIN-REOPEN-01: Otorisasi reopen

Status:

```text
MATCH
```

Perilaku aktual:

Reopen settlement hanya diizinkan untuk `ADMIN`.

Kontrak:

```text
REOPEN SETTLEMENT
    → ADMIN ONLY
```

Tidak ada workflow permintaan reopen dari wali kelas ke admin dalam implementasi yang diperiksa.

Workflow permintaan dan persetujuan tidak boleh ditambahkan secara diam-diam dalam fase audit.

## 14.3 Temuan FIN-REOPEN-02: Riwayat settlement dipertahankan

Status:

```text
MATCH
```

Perilaku aktual:

Settlement lama tidak dihapus. Statusnya diubah menjadi `SUPERSEDED`, dan rekening dikembalikan ke `OPEN`.

Database membatasi satu settlement `COMPLETED` per rekening melalui partial unique index.

Kontrak:

```text
REOPEN
    → OLD SETTLEMENT SUPERSEDED
    → ACCOUNT OPEN
    → HISTORY RETAINED
```

## 14.4 Temuan FIN-REOPEN-03: Hubungan settlement pengganti

Status:

```text
GAP - IMPLEMENTATION
```

Perilaku aktual:

Tabel settlement memiliki:

```text
replaces_settlement_id
```

Namun, kedua alur pembuatan settlement yang diperiksa mengirimkan nilai `nil` untuk `ReplacesSettlementID`.

Akibatnya, setelah settlement dibuka kembali dan kemudian dibuat settlement baru, source saat ini tidak otomatis menghubungkan settlement baru ke settlement yang digantikan.

Dampak:

- Riwayat settlement tersedia.
- Status settlement lama masih dapat menunjukkan `SUPERSEDED`.
- Relasi eksplisit antara settlement pengganti dan settlement sebelumnya belum ditegakkan oleh alur pembuatan settlement yang diperiksa.

Ownership:

```text
FASE 16.8 — Resolve contract-level gaps
```

Kontrak yang perlu ditegakkan:

Jika `replaces_settlement_id` dimaksudkan sebagai lineage settlement, nilai tersebut harus mencerminkan hubungan penggantian yang benar.

Batas pekerjaan:

Tidak mengubah model settlement atau migration pada fase audit ini.

## 14.5 Temuan FIN-REOPEN-04: Reopen dan status tahun ajaran

Status:

```text
GAP - CONTRACT DECISION
```

Perilaku aktual:

`Reopen` memeriksa role, status rekening, dan settlement terkini. Source yang diperiksa tidak memeriksa status tahun ajaran rekening sebelum membuka kembali rekening.

Keputusan yang dibutuhkan:

1. Apakah rekening tahun ajaran `CLOSED` boleh dibuka kembali?
2. Apakah reopen harus ditolak setelah tahun ajaran ditutup?
3. Jika reopen diizinkan, apakah transaksi baru boleh dibuat pada tahun ajaran tersebut?
4. Apakah reopen membutuhkan alasan yang wajib dicatat?

Ownership:

```text
FASE 16.7 — Gap Resolution Decision
```

Temuan ini berhubungan dengan `GAP-SETTLEMENT-REOPEN-01` pada audit sebelumnya.

Tidak boleh mengimplementasikan pembatasan atau izin baru sebelum keputusan kontrak dikunci.

---

# 15. Kontrak Database Keuangan

## 15.1 Constraint transaksi

Migration menetapkan:

```text
type:
    DEPOSIT
    WITHDRAWAL

amount:
    NOT NULL
    > 0
    INTEGER VALUE

status:
    ACTIVE
    CANCELLED
```

Status:

```text
MATCH
```

Constraint database selaras dengan jenis transaksi, nominal positif, dan status yang didefinisikan domain.

## 15.2 Constraint settlement

Migration menetapkan:

```text
type:
    REGULAR_YEAR_END
    STUDENT_LEAVING

amount:
    NOT NULL
    >= 0
    INTEGER VALUE

status:
    COMPLETED
    SUPERSEDED
```

Database juga menetapkan:

```sql
CREATE UNIQUE INDEX savings_settlements_one_completed_idx
    ON savings_settlements (savings_account_id)
    WHERE status = 'COMPLETED';
```

Status:

```text
MATCH
```

Constraint ini membatasi jumlah settlement yang `COMPLETED` untuk satu rekening menjadi paling banyak satu pada satu waktu.

Constraint tersebut tidak dengan sendirinya membuktikan bahwa nominal settlement sama dengan saldo transaksi terkini.

## 15.3 Temuan FIN-DB-01: Saldo tidak disimpan sebagai kolom account

Status:

```text
MATCH
```

Perilaku aktual:

Tabel `savings_accounts` tidak memiliki kolom saldo.

Saldo diturunkan dari transaksi aktif.

Kontrak:

```text
BALANCE IS DERIVED DATA
```

Tidak diperlukan mekanisme sinkronisasi kolom saldo account pada setiap transaksi selama desain ini dipertahankan.

## 15.4 Temuan FIN-DB-02: Database tidak dapat membuktikan saldo historis dari CHECK sederhana

Status:

```text
MATCH
```

Perilaku aktual:

Constraint database memvalidasi nominal dan status transaksi, tetapi tidak menghitung ulang saldo kumulatif berdasarkan seluruh transaksi rekening.

Karena itu, invariant saldo nonnegatif tetap membutuhkan enforcement pada service atau mekanisme database yang memang dirancang untuk menjaga invariant lintas-row.

Audit ini tidak menyarankan trigger atau prosedur database baru tanpa kebutuhan kontrak dan analisis concurrency yang jelas.

---

# 16. Audit Log Keuangan

## 16.1 Jenis mutasi yang diperiksa

Operasi keuangan berikut menulis audit log:

```text
CREATE SAVINGS ACCOUNT
CREATE DEPOSIT
CREATE WITHDRAWAL
EDIT TRANSACTION
CANCEL TRANSACTION
SETTLE YEAR END
SETTLE STUDENT LEAVING
REOPEN SETTLEMENT
```

Untuk perubahan transaksi, audit mencatat data sebelum dan sesudah perubahan apabila tersedia.

Settlement dan perubahan status rekening juga memiliki audit masing-masing.

## 16.2 Temuan FIN-AUDIT-01: Audit pada operasi keuangan

Status:

```text
MATCH
```

Perilaku aktual:

Source yang diperiksa menyediakan audit untuk operasi keuangan utama yang disebutkan di atas.

Operasi create, edit, cancel, settle, dan reopen memanggil mekanisme audit yang menggunakan repository dalam transaction boundary terkait.

## 16.3 Temuan FIN-AUDIT-02: Atomicity audit

Status:

```text
MATCH
```

Kontrak:

```text
FINANCIAL MUTATION
    +
AUDIT RECORD
    → COMMIT TOGETHER
```

Audit yang gagal tidak boleh dibiarkan menghasilkan operasi keuangan yang sudah dianggap berhasil tanpa catatan audit.

## 16.4 Batasan audit

Keberadaan pemanggilan audit pada service tidak otomatis membuktikan bahwa setiap jalur kegagalan, rollback, dan concurrency sudah teruji.

Pengujian integrasi harus membuktikan perilaku tersebut pada fase pengujian kontrak dan implementasi terkait.

---

# 17. Cross-Entity Financial Consistency

Kontrak keuangan tidak berdiri sendiri. Mutasi harus konsisten dengan rekening, transaksi, tahun ajaran, dan settlement.

## 17.1 Rekening dan transaksi

Invariant:

```text
ACCOUNT OPEN
    → FINANCIAL TRANSACTION MUTATIONS MAY BE ALLOWED
       SUBJECT TO OTHER CONTRACTS

ACCOUNT SETTLED
    → FINANCIAL TRANSACTION MUTATIONS RESTRICTED
```

Temuan yang terkait:

```text
FIN-CANCEL-02
FIN-SETTLEMENT-05
```

## 17.2 Transaksi dan saldo

Invariant:

```text
ACTIVE TRANSACTION HISTORY
    → BALANCE MUST BE VALID
    → BALANCE MUST NOT BECOME NEGATIVE
```

Temuan yang terkait:

```text
FIN-BALANCE-02
FIN-BALANCE-03
FIN-CANCEL-03
FIN-EDIT-02
```

## 17.3 Transaksi dan tahun ajaran

Invariant yang telah dicatat sebelumnya:

```text
ACADEMIC_YEAR.START_DATE
    <= TRANSACTION_DATE
    < ACADEMIC_YEAR.END_DATE
```

Ownership:

```text
FASE 21 — TRANSACTION ENGINE
```

Referensi temuan sebelumnya:

```text
GAP-TRANSACTION-DATE-01
GAP-DB-TRANSACTION-01
GAP-LIFE-TRANSACTION-01
```

Temuan ini tidak dibuat ulang sebagai gap baru pada FASE 16.5.

## 17.4 Tahun ajaran dan settlement

Kontrak yang sudah ditemukan:

```text
REGULAR YEAR-END SETTLEMENT
    → ACADEMIC YEAR MUST BE CLOSING
```

Keputusan yang masih dibutuhkan:

- Apakah tahun ajaran boleh ditutup ketika masih ada rekening `OPEN`?
- Apakah settlement yang belum selesai menghalangi penutupan tahun ajaran?
- Apakah settlement dapat dibuka kembali setelah tahun ajaran `CLOSED`?

Referensi:

```text
GAP-LIFE-AY-01
GAP-SETTLEMENT-REOPEN-01
FIN-REOPEN-04
```

Ownership keputusan:

```text
FASE 16.7 — Gap Resolution Decision
```

---

# 18. Financial Gap Register

Daftar berikut merupakan register utama temuan FASE 16.5.

| ID | Status | Ringkasan | Ownership |
|---|---|---|---|
| FIN-MONEY-01 | MATCH | Representasi uang berupa integer dan database menolak pecahan | Selesai |
| FIN-MONEY-02 | GAP - IMPLEMENTATION | Operasi aritmetika `Money` tidak memeriksa overflow | FASE 16.8 |
| FIN-BALANCE-01 | MATCH | Saldo diturunkan dari transaksi aktif | Selesai |
| FIN-BALANCE-02 | GAP - IMPLEMENTATION | Deposit tidak memvalidasi saldo historis | FASE 16.8 |
| FIN-BALANCE-03 | GAP - CONTRACT DECISION | Kebijakan tanggal efektif dan saldo historis | FASE 16.7 |
| FIN-DEPOSIT-01 | MATCH | Deposit harus memiliki nominal positif | Selesai |
| FIN-DEPOSIT-02 | MATCH | Deposit menolak rekening settled | Selesai |
| FIN-DEPOSIT-03 | MATCH | Deposit dan audit berada dalam transaction boundary | Selesai |
| FIN-WITHDRAWAL-01 | MATCH | Withdrawal harus memiliki nominal positif | Selesai |
| FIN-WITHDRAWAL-02 | MATCH | Withdrawal memvalidasi saldo | Selesai |
| FIN-WITHDRAWAL-03 | MATCH | Withdrawal mengunci rekening sebelum memvalidasi saldo | Selesai |
| FIN-EDIT-01 | GAP - IMPLEMENTATION | Validasi nominal positif tidak dilakukan eksplisit pada service edit | FASE 16.8 |
| FIN-EDIT-02 | MATCH | Edit menghitung ulang saldo sebelum persistence | Selesai |
| FIN-EDIT-03 | DEFERRED | Batas tanggal transaksi hasil edit | FASE 21 |
| FIN-CANCEL-01 | MATCH | Pembatalan mempertahankan row transaksi | Selesai |
| FIN-CANCEL-02 | GAP - IMPLEMENTATION | Cancel tidak memeriksa status rekening settled | FASE 16.8 |
| FIN-CANCEL-03 | GAP - IMPLEMENTATION | Cancel tidak menghitung ulang validitas saldo | FASE 16.8 |
| FIN-STATUS-01 | MATCH | Status transaksi dibatasi pada ACTIVE dan CANCELLED | Selesai |
| FIN-ACCOUNT-01 | MATCH | Status account dan settled_at konsisten | Selesai |
| FIN-ACCOUNT-02 | MATCH | Satu rekening per siswa per tahun ajaran | Selesai |
| FIN-ACCOUNT-03 | GAP - CONTRACT DECISION | Status tahun ajaran untuk pembuatan rekening | FASE 16.7 |
| FIN-SETTLEMENT-01 | MATCH | Settlement menyimpan snapshot saldo transaksi aktif | Selesai |
| FIN-SETTLEMENT-02 | MATCH | Settlement akhir tahun mensyaratkan tahun ajaran CLOSING | Selesai |
| FIN-SETTLEMENT-03 | MATCH | Settlement siswa keluar mensyaratkan siswa LEFT | Selesai |
| FIN-SETTLEMENT-04 | MATCH | Settlement, account, dan audit dikerjakan atomik | Selesai |
| FIN-SETTLEMENT-05 | GAP - IMPLEMENTATION | Transaksi rekening settled dapat dibatalkan | FASE 16.8 |
| FIN-REOPEN-01 | MATCH | Reopen settlement hanya ADMIN | Selesai |
| FIN-REOPEN-02 | MATCH | Reopen mempertahankan riwayat settlement | Selesai |
| FIN-REOPEN-03 | GAP - IMPLEMENTATION | Relasi settlement pengganti belum ditegakkan | FASE 16.8 |
| FIN-REOPEN-04 | GAP - CONTRACT DECISION | Kebijakan reopen setelah tahun ajaran CLOSED | FASE 16.7 |
| FIN-DB-01 | MATCH | Saldo tidak disimpan pada kolom rekening | Selesai |
| FIN-DB-02 | MATCH | Constraint database tidak menghitung saldo lintas transaksi | Selesai |
| FIN-AUDIT-01 | MATCH | Operasi keuangan utama memiliki audit log | Selesai |
| FIN-AUDIT-02 | MATCH | Audit dan mutasi keuangan menggunakan transaction boundary | Selesai |

Catatan:

`FIN-SETTLEMENT-05` berkaitan dengan `FIN-CANCEL-02`. Kedua ID dipertahankan karena yang satu mengidentifikasi dampak pada settlement, sedangkan yang lain mengidentifikasi jalur pembatalan transaksi.

Implementasi penyelesaiannya tidak boleh diduplikasi tanpa alasan.

---

# 19. Keputusan yang Belum Boleh Diambil dalam FASE 16.5

Hal-hal berikut harus menunggu konsolidasi keputusan pada FASE 16.7.

## 19.1 Nominal maksimum dan overflow

Harus diputuskan batas nominal bisnis yang valid dan cara penegakannya.

## 19.2 Transaksi bertanggal lampau

Harus diputuskan apakah transaksi bertanggal lampau diizinkan dan bagaimana validitas saldo historis ditegakkan.

## 19.3 Pembuatan rekening

Harus diputuskan status tahun ajaran yang mengizinkan pembuatan rekening.

## 19.4 Penutupan tahun ajaran

Harus diputuskan apakah semua rekening wajib settled sebelum tahun ajaran dapat berstatus `CLOSED`.

## 19.5 Reopen settlement

Harus diputuskan apakah reopen diizinkan setelah tahun ajaran `CLOSED` dan apakah alasan reopen wajib dicatat.

Keputusan tersebut harus konsisten dengan audit domain, database, lifecycle, dan authorization.

---

# 20. Ownership Penyelesaian

## 20.1 FASE 16.7 — Gap Resolution Decision

Pemilik keputusan kontrak:

```text
FIN-BALANCE-03
FIN-ACCOUNT-03
FIN-REOPEN-04
```

Keputusan lintas fase yang berhubungan:

```text
GAP-LIFE-AY-01
GAP-SETTLEMENT-REOPEN-01
GAP-DB-AY-01
GAP-DB-HISTORY-01
```

## 20.2 FASE 16.8 — Resolve Contract-Level Gaps

Kandidat penyelesaian implementasi:

```text
FIN-MONEY-02
FIN-BALANCE-02
FIN-EDIT-01
FIN-CANCEL-02
FIN-CANCEL-03
FIN-SETTLEMENT-05
FIN-REOPEN-03
```

Penyelesaian tidak boleh dilakukan sebelum keputusan yang relevan dari FASE 16.7 dikunci.

## 20.3 FASE 21 — Transaction Engine

Ownership:

```text
FIN-EDIT-03
GAP-TRANSACTION-DATE-01
GAP-DB-TRANSACTION-01
GAP-LIFE-TRANSACTION-01
```

FASE 21 harus mengimplementasikan kontrak tanggal transaksi yang telah disepakati, termasuk pada operasi create dan edit.

---

# 21. Kontrak Keuangan yang Harus Dipertahankan

Kontrak berikut menjadi baseline untuk fase berikutnya.

## 21.1 Money

```text
- Money menggunakan representasi integer.
- Nominal transaksi harus lebih besar dari nol.
- Nominal settlement boleh nol.
- Nominal settlement tidak boleh negatif.
- Pecahan tidak diterima.
```

## 21.2 Balance

```text
- Saldo dihitung dari transaksi ACTIVE.
- Transaksi CANCELLED tidak memengaruhi saldo.
- Saldo tidak boleh negatif.
- Validitas saldo bergantung pada seluruh transaksi aktif.
```

## 21.3 Deposit

```text
- Hanya ADMIN dan WALI_KELAS yang dapat membuat deposit.
- Rekening harus OPEN.
- Nominal harus positif.
- Mutasi dan audit harus atomik.
```

## 21.4 Withdrawal

```text
- Hanya ADMIN dan WALI_KELAS yang dapat membuat withdrawal.
- Rekening harus OPEN.
- Nominal harus positif.
- Saldo harus valid setelah withdrawal.
- Mutasi dan audit harus atomik.
```

## 21.5 Edit

```text
- Rekening harus memenuhi syarat untuk dimutasi.
- Transaksi harus ACTIVE.
- Nominal hasil edit harus positif.
- Saldo harus dihitung ulang.
- Mutasi dan audit harus atomik.
```

## 21.6 Cancel

```text
- Pembatalan tidak menghapus transaksi.
- Status transaksi menjadi CANCELLED.
- Transaksi CANCELLED tidak memengaruhi saldo.
- Pembatalan harus menjaga validitas riwayat saldo.
- Mutasi dan audit harus atomik.
```

## 21.7 Settlement

```text
- Settlement menyimpan snapshot saldo transaksi aktif.
- Settlement akhir tahun membutuhkan tahun ajaran CLOSING.
- Settlement siswa keluar membutuhkan siswa LEFT.
- Rekening berubah dari OPEN menjadi SETTLED.
- Nominal settlement tidak negatif.
- Settlement, account state, dan audit harus atomik.
```

## 21.8 Reopen settlement

```text
- Reopen hanya boleh dilakukan ADMIN.
- Settlement lama dipertahankan.
- Settlement lama menjadi SUPERSEDED.
- Rekening kembali OPEN.
- Reopen dan audit harus atomik.
- Aturan reopen setelah tahun ajaran CLOSED menunggu keputusan kontrak.
```

---

# 22. Exit Criteria FASE 16.5

FASE 16.5 dianggap selesai jika:

- [x] Source domain keuangan diinventarisasi.
- [x] Service transaksi diperiksa.
- [x] Service rekening diperiksa.
- [x] Service settlement diperiksa.
- [x] Repository transaksi diperiksa.
- [x] Repository rekening diperiksa.
- [x] Repository settlement diperiksa.
- [x] Migration keuangan diperiksa.
- [x] Representasi uang dibandingkan dengan database.
- [x] Perhitungan saldo diperiksa.
- [x] Deposit dan withdrawal diperiksa.
- [x] Edit dan cancel diperiksa.
- [x] Settlement dan reopen diperiksa.
- [x] Audit log keuangan diperiksa.
- [x] Temuan diberi ID.
- [x] Temuan diklasifikasikan.
- [x] Ownership penyelesaian dicatat.
- [x] Keputusan yang belum tersedia tidak ditebak.
- [x] Tidak ada source code yang diubah.
- [x] Tidak ada migration yang diubah.

---

# 23. Status Akhir

```text
FASE 16.5 — FINANCIAL CONTRACT AUDIT

STATUS: AUDIT COMPLETE

SOURCE CODE CHANGED: NO
MIGRATION CHANGED: NO
BUSINESS CONTRACT CHANGED: NO

IMPLEMENTATION GAPS: RECORDED
CONTRACT DECISIONS: RECORDED
DEFERRED WORK: ASSIGNED
```

FASE 16.5 menghasilkan audit keuangan dan register temuan untuk konsolidasi kontrak.

Fase ini tidak menyatakan bahwa seluruh implementasi keuangan sudah memenuhi kontrak. Temuan `GAP - IMPLEMENTATION` dan `GAP - CONTRACT DECISION` tetap terbuka sampai diselesaikan pada fase yang ditetapkan.

Langkah berikutnya setelah dokumen dibuat dan diverifikasi:

```text
FASE 16.6 — AUDIT CONTRACT AUDIT
```
