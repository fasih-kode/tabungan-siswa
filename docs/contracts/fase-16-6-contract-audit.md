# FASE 16.6 — CONTRACT AUDIT

Status: AUDIT COMPLETE
Phase: 16.6
Parent: FASE 16 — CONTRACT HARDENING
Scope: Cross-audit FASE 16.0 through FASE 16.5
Source baseline: `1a680c81d71cb6a8dbc6485612b1677a528283c4`
Source archive: `tabungan-siswa.tar(20261009-002538).gz`

---

# 1. Tujuan

FASE 16.6 memeriksa konsistensi silang dokumen FASE 16.0–16.5.

Audit mencakup:

1. Konsistensi status dan klasifikasi temuan.
2. Duplikasi ID dan hubungan antar-temuan.
3. Konflik kontrak domain, database, lifecycle, authorization, dan keuangan.
4. Ownership penyelesaian.
5. Keputusan bisnis yang belum dikunci.
6. Kesiapan register keputusan FASE 16.7.

Fase ini hanya mengonsolidasikan temuan. Source code, migration, skema database, API service, dan business rule tidak diubah.

## 1.1 Batas klaim

`AUDIT COMPLETE` berarti audit silang dan pencatatan temuan selesai. Status ini tidak berarti semua kontrak telah dikunci, implementation gap telah diselesaikan, atau seluruh test telah dijalankan.

## 1.2 Dokumen sumber

- `docs/contracts/fase-16-0-source-baseline.md`
- `docs/contracts/fase-16-1-domain-contract-audit.md`
- `docs/contracts/fase-16-2-database-contract-audit.md`
- `docs/contracts/fase-16-3-lifecycle-contract-audit.md`
- `docs/contracts/fase-16-4-authorization-contract-audit.md`
- `docs/contracts/fase-16-5-financial-contract-audit.md`

---

# 2. Klasifikasi Kanonis

| Status | Makna |
|---|---|
| `MATCH` | Implementasi sesuai dengan kontrak yang sudah dikunci dalam scope audit. |
| `GAP - IMPLEMENTATION` | Kontrak jelas, tetapi implementasi belum memenuhinya. |
| `GAP - CONTRACT DECISION` | Keputusan bisnis belum ditetapkan. |
| `CONFLICT` | Implementasi atau dua kontrak saling bertentangan. |
| `DEFERRED` | Penyelesaian ditunda ke fase pemiliknya; temuan tetap terbuka. |

Temuan implementasi yang bergantung pada keputusan bisnis belum boleh dianggap siap dikerjakan.

Temuan `MATCH` pada satu scope tidak membuktikan bahwa seluruh sistem sudah memenuhi kontrak serupa.

---

# 3. Ringkasan Temuan Silang

| ID | Area | Kesimpulan |
|---|---|---|
| CA-01 | Tanggal transaksi | Status kontrak belum konsisten antar-dokumen. |
| CA-02 | Periode tahun ajaran | Overlap periode merupakan keputusan yang belum dikunci. |
| CA-03 | Penempatan kelas awal | Satu implementation gap tercatat dengan beberapa ID. |
| CA-04 | Batas periode history | Aturan `StudentClassHistory` dan `TeacherClassAssignment` perlu dibedakan. |
| CA-05 | Audit log | Audit keuangan tidak menutup gap audit coverage global. |
| CA-06 | Settlement lineage | Kewajiban relasi replacement belum diputuskan secara konsisten. |
| CA-07 | Reopen settlement | Eligibility reopen dan workflow approval merupakan dua persoalan berbeda. |
| CA-08 | Transfer request | Kebijakan multiple pending requests belum diputuskan. |
| CA-09 | Pembuatan rekening | Status siswa dan status tahun ajaran merupakan keputusan terpisah. |
| CA-10 | Mutasi transaksi | Gap implementasi harus dipisahkan dari kebijakan tanggal dan saldo historis. |
| CA-11 | Authorization | Scope akses wali kelas terhadap history siswa belum diputuskan. |
| CA-12 | Ownership | Setiap temuan perlu satu jalur penyelesaian yang jelas. |

---

# 4. CA-01 — Transaction Date Boundary

Referensi:

- `GAP-02`
- `GAP-TRANSACTION-DATE-01`
- `GAP-DB-TRANSACTION-01`
- `GAP-LIFE-TRANSACTION-01`
- `FIN-EDIT-03`

Target aturan yang dicatat:

`academic_year.start_date <= transaction_date < academic_year.end_date`

Masalah konsistensi: dokumen awal menyebut aturan ini sebagai kontrak yang masih perlu dibekukan, sedangkan FASE 16.5 memperlakukannya sebagai invariant yang akan ditegakkan pada FASE 21.

**Klasifikasi konsolidasi:** `GAP - CONTRACT DECISION`.

FASE 16.7 harus memutuskan apakah aturan tersebut final, apakah transaksi bertanggal lampau diperbolehkan, dan bagaimana validitas saldo historis dipertahankan.

Ownership keputusan: FASE 16.7. Ownership enforcement: FASE 21.

---

# 5. CA-02 — Academic Year Period Overlap

Referensi:

- `GAP-04`
- `GAP-AY-01`
- `GAP-DB-AY-01`

Constraint `start_date < end_date` dan pembatasan jumlah tahun ajaran `OPEN` tidak dengan sendirinya melarang periode tahun ajaran yang saling overlap.

Ketiga referensi tersebut harus diperlakukan sebagai satu kelompok keputusan, bukan tiga implementation task independen.

**Klasifikasi konsolidasi:** `GAP - CONTRACT DECISION`.

FASE 16.7 harus menetapkan apakah periode tahun ajaran wajib tidak overlap sebelum constraint atau validasi baru diimplementasikan.

Ownership keputusan: FASE 16.7. Implementasi menunggu keputusan tersebut.

---

# 6. CA-03 — Initial Student Class Assignment

Referensi:

- `GAP-01`
- `GAP-STUDENT-CLASS-01`
- `GAP-LIFE-STUDENT-CLASS-01`

Audit sebelumnya mencatat bahwa use case penempatan kelas awal siswa belum tersedia secara eksplisit.

Ketiga ID mengacu pada satu masalah utama, bukan tiga implementation gap terpisah.

**ID kanonis:** `GAP-STUDENT-CLASS-01`.

**Klasifikasi:** `GAP - IMPLEMENTATION`.

Ownership: FASE 19 — Student Management.

Penyelesaian harus menyediakan use case penempatan kelas awal dan menjaga konsistensi `StudentClassHistory`. Jangan membuat use case duplikat hanya untuk menghilangkan ID temuan yang berulang.

---

# 7. CA-04 — History Period Boundary

Referensi: `GAP-DB-HISTORY-01`.

Kebijakan batas periode `StudentClassHistory` dan `TeacherClassAssignment` berhubungan, tetapi satu invariant tidak otomatis membuktikan invariant lainnya.

Gunakan subfinding konsolidasi:

- `GAP-DB-HISTORY-01A`: batas periode `StudentClassHistory`.
- `GAP-DB-HISTORY-01B`: batas periode `TeacherClassAssignment`.

**Klasifikasi:** `GAP - CONTRACT DECISION`.

FASE 16.7 harus memutuskan apakah kedua periode wajib berada di dalam tahun ajaran terkait dan apakah keduanya memakai aturan yang sama.

Jangan melakukan penggantian ID massal pada dokumen terdahulu. Pertahankan ID historis dan gunakan subfinding ini pada register konsolidasi.

---

# 8. CA-05 — Audit Coverage

Referensi:

- `GAP-03`
- `FIN-AUDIT-01`
- `FIN-AUDIT-02`

FASE 16.5 menemukan audit pada operasi keuangan utama. Temuan itu hanya berlaku dalam scope yang diperiksa dan tidak membuktikan bahwa seluruh mutasi bisnis sudah diaudit.

**Klasifikasi konsolidasi:**

- Audit keuangan yang diperiksa: `MATCH` dalam scope tersebut.
- Audit coverage global: `DEFERRED`.

Ownership global: FASE 22 — Transaction Correction & Audit.

FASE 22 harus memeriksa cakupan audit pada seluruh mutation yang diwajibkan kontrak. `GAP-03` tidak boleh ditutup hanya berdasarkan temuan audit keuangan.

---

# 9. CA-06 — Settlement Replacement Lineage

Referensi:

- `GAP-DB-SETTLEMENT-01`
- `FIN-REOPEN-03`

FASE 16.5 menemukan bahwa alur settlement yang diperiksa belum mengisi `ReplacesSettlementID`, sedangkan FASE 16.2 menyatakan bahwa kontrak lineage belum ditetapkan.

Sebelum memutuskan bahwa kondisi ini merupakan implementation gap, tetapkan dahulu apakah hubungan replacement memang diwajibkan.

**Klasifikasi konsolidasi:** `GAP - CONTRACT DECISION`.

FASE 16.7 harus memutuskan:

1. Apakah settlement baru wajib merujuk settlement yang digantikan.
2. Apakah replacement harus berasal dari rekening yang sama.
3. Apakah satu settlement dapat memiliki lebih dari satu replacement.
4. Bagaimana lineage diperlakukan setelah reopen berulang.

Ownership keputusan: FASE 16.7. Ownership implementasi: fase yang ditetapkan setelah keputusan, termasuk FASE 22 jika kontrak mewajibkannya.

---

# 10. CA-07 — Settlement Reopen

Referensi:

- `GAP-05`
- `GAP-SETTLEMENT-REOPEN-01`
- `GAP-LIFE-SETTLEMENT-01`
- `FIN-REOPEN-04`

Pisahkan dua persoalan:

**A. Workflow request/approval**

Workflow permintaan wali kelas dan persetujuan admin belum tersedia dan belum menjadi kontrak yang dikunci.

Status: `DEFERRED`, sampai workflow tersebut diputuskan secara eksplisit.

**B. Reopen setelah tahun ajaran CLOSED**

Belum diputuskan apakah rekening pada tahun ajaran `CLOSED` boleh dibuka kembali.

Status: `GAP - CONTRACT DECISION`.

FASE 16.7 harus memutuskan kelayakan reopen setelah tahun ajaran ditutup, apakah transaksi baru tetap dilarang, apakah alasan reopen wajib dicatat, dan apakah workflow approval memang diperlukan. Keputusan-keputusan tersebut tidak boleh digabung menjadi satu aturan.

---

# 11. CA-08 — Multiple Pending Transfer Requests

Referensi:

- `GAP-DB-TRANSFER-02`
- `GAP-LIFE-TRANSFER-02`

Kedua temuan mengacu pada satu keputusan: apakah beberapa transfer request berstatus `PENDING` boleh ada dalam scope siswa dan tahun ajaran yang relevan.

**ID kanonis:** `GAP-DB-TRANSFER-02`.

**Klasifikasi:** `GAP - CONTRACT DECISION`.

FASE 16.7 harus menetapkan scope aturan sebelum unique constraint atau validasi baru dibuat. Jangan mengasumsikan satu siswa hanya boleh memiliki satu permintaan secara global.

---

# 12. CA-09 — Savings Account Creation

Referensi:

- `GAP-LIFE-ACCOUNT-01`
- `GAP-LIFE-ACCOUNT-02`
- `FIN-ACCOUNT-03`

Dua pertanyaan harus tetap terpisah:

1. Apakah siswa `LEFT` boleh memiliki rekening baru?
2. Apakah rekening boleh dibuat pada tahun ajaran `CLOSED`?

**Klasifikasi keduanya:** `GAP - CONTRACT DECISION`.

FASE 16.7 harus memutuskan masing-masing aturan. Kebijakan transaksi pada rekening `SETTLED` tidak otomatis menjawab kelayakan pembuatan rekening baru.

---

# 13. CA-10 — Transaction Mutation

Referensi:

- `FIN-BALANCE-02`
- `FIN-BALANCE-03`
- `FIN-EDIT-01`
- `FIN-EDIT-03`
- `FIN-CANCEL-02`
- `FIN-CANCEL-03`
- `FIN-SETTLEMENT-05`
- `GAP-TRANSACTION-DATE-01`

Kelompokkan temuan menjadi:

**Keputusan kontrak**
- Kebijakan tanggal transaksi dan saldo historis: `FIN-BALANCE-03` dan referensi boundary tanggal terkait.

**Implementation gap**
- `FIN-EDIT-01`: validasi nominal positif pada edit.
- `FIN-CANCEL-02`: pembatalan belum memeriksa status rekening settled.
- `FIN-CANCEL-03`: pembatalan belum memvalidasi hasil saldo.

**Referensi dampak**
- `FIN-SETTLEMENT-05` menggambarkan dampak mutasi rekening settled terhadap settlement dan berhubungan dengan `FIN-CANCEL-02`.

FASE 16.7 harus mengunci kebijakan tanggal efektif dan saldo historis. Implementation gap yang bergantung pada keputusan itu baru siap dikerjakan setelah kontraknya jelas.

Ownership: FASE 16.8 untuk gap kontrak tingkat implementasi; FASE 21 untuk enforcement boundary tanggal transaksi.

---

# 14. CA-11 — Authorization Scope

Referensi: `GAP-AUTH-HISTORY-01`.

Belum diputuskan apakah `WALI_KELAS` boleh mengakses:

- Hanya siswa pada kelas saat ini; atau
- Siswa yang mempunyai riwayat kelas dalam tahun ajaran terkait.

**Klasifikasi:** `GAP - CONTRACT DECISION`.

FASE 16.7 harus menetapkan aturan scope dan tanggal efektif yang benar. Jangan memperluas akses hanya karena riwayat kelas tersedia, dan jangan mengubah scope siswa terhadap datanya sendiri.

---

# 15. CA-12 — Ownership Penyelesaian

Ownership konsolidasi:

| Area | Pemilik keputusan | Pemilik implementasi |
|---|---|---|
| Academic year overlap | FASE 16.7 | Setelah kontrak dikunci |
| Student class history period | FASE 16.7 | Setelah kontrak dikunci |
| Teacher assignment period | FASE 16.7 | Setelah kontrak dikunci |
| Initial student class assignment | Kontrak yang ada, jika tetap berlaku | FASE 19 |
| Transaction date boundary | FASE 16.7 | FASE 21 |
| Global audit coverage | Sesuai kontrak audit | FASE 22 |
| Settlement lineage | FASE 16.7 | Setelah keputusan; FASE 22 bila relevan |
| Multiple pending transfers | FASE 16.7 | Setelah kontrak dikunci |
| Account creation rules | FASE 16.7 | Setelah kontrak dikunci |
| Reopen after year close | FASE 16.7 | Setelah kontrak dikunci |
| Wali class-history scope | FASE 16.7 | Setelah kontrak dikunci |
| Edit amount validation | Sesuai kontrak nominal | FASE 16.8 |
| Cancel on settled account | Sesuai kontrak lifecycle | FASE 16.8 |

Satu keputusan kontrak dapat memengaruhi lebih dari satu fase implementasi. Fase implementasi tidak boleh mendefinisikan ulang aturan bisnisnya sendiri.

---

# 16. Canonical Finding Register

| Finding kanonis | Referensi terkait | Klasifikasi | Owner |
|---|---|---|---|
| Academic year overlap | `GAP-04`, `GAP-AY-01`, `GAP-DB-AY-01` | Contract decision | FASE 16.7 |
| Initial class assignment | `GAP-01`, `GAP-STUDENT-CLASS-01`, `GAP-LIFE-STUDENT-CLASS-01` | Implementation gap | FASE 19 |
| Transaction date boundary | `GAP-02`, `GAP-TRANSACTION-DATE-01`, `GAP-DB-TRANSACTION-01`, `GAP-LIFE-TRANSACTION-01`, `FIN-EDIT-03` | Contract decision | FASE 16.7, lalu FASE 21 |
| Student history period | `GAP-DB-HISTORY-01A` | Contract decision | FASE 16.7 |
| Teacher assignment period | `GAP-DB-HISTORY-01B` | Contract decision | FASE 16.7 |
| Global audit coverage | `GAP-03` | Deferred | FASE 22 |
| Transfer rejection reason | `GAP-DB-TRANSFER-01` | Implementation gap | Fase pemilik sesuai roadmap |
| Multiple pending transfers | `GAP-DB-TRANSFER-02`, `GAP-LIFE-TRANSFER-02` | Contract decision | FASE 16.7 |
| Student LEFT account creation | `GAP-LIFE-ACCOUNT-01` | Contract decision | FASE 16.7 |
| Closed-year account creation | `GAP-LIFE-ACCOUNT-02` | Contract decision | FASE 16.7 |
| Reopen after year close | `GAP-LIFE-SETTLEMENT-01`, `FIN-REOPEN-04` | Contract decision | FASE 16.7 |
| Reopen approval workflow | `GAP-05`, `GAP-SETTLEMENT-REOPEN-01` | Deferred pending decision | FASE 16.7 |
| Settlement lineage | `GAP-DB-SETTLEMENT-01`, `FIN-REOPEN-03` | Contract decision | FASE 16.7 |
| Wali history authorization | `GAP-AUTH-HISTORY-01` | Contract decision | FASE 16.7 |
| Money overflow | `FIN-MONEY-02` | Implementation gap; numeric contract must be respected | FASE 16.8 |
| Deposit historical balance | `FIN-BALANCE-02` | Implementation gap; depends on date policy | FASE 16.8 |
| Edit amount validation | `FIN-EDIT-01` | Implementation gap | FASE 16.8 |
| Cancel settled account | `FIN-CANCEL-02`, `FIN-SETTLEMENT-05` | Implementation gap | FASE 16.8 |
| Cancel balance validity | `FIN-CANCEL-03` | Implementation gap; depends on date policy | FASE 16.8 |

### Aturan register

1. ID lama tetap menjadi referensi historis.
2. ID terkait tidak otomatis berarti task implementasi terpisah.
3. Parent finding tidak boleh ditutup sebelum subfinding wajib selesai.
4. `DEFERRED` bukan `RESOLVED`.
5. Gap yang bergantung pada keputusan belum siap diimplementasikan.
6. Perubahan klasifikasi pada FASE 16.7 harus mencatat alasan dan ID yang terdampak.

---

# 17. Cross-Contract Invariants

Aturan berikut harus konsisten di seluruh domain, service, repository, dan database:

1. Mutasi keuangan dan audit terkait harus atomik.
2. Saldo dihitung dari transaksi aktif, bukan diedit langsung.
3. Transaksi harus berada dalam periode tahun ajaran yang berlaku.
4. Akses wali kelas harus mengikuti aturan scope siswa yang dikunci.
5. Settlement tidak boleh menghapus riwayat transaksi.
6. Perubahan status harus mengikuti lifecycle yang telah disepakati.
7. Gap yang membutuhkan keputusan tidak boleh diimplementasikan sebelum keputusan dikunci.

# 18. Interpretasi yang Dilarang

1. `DEFERRED` tidak berarti disetujui atau selesai.
2. Temuan audit tidak otomatis menjadi izin mengubah kontrak.
3. Kolom database yang tersedia tidak otomatis berarti workflow terkait sudah disepakati.
4. Jangan mengubah migration atau source code hanya untuk membuat tes lolos.
5. Jangan menggabungkan beberapa keputusan berbeda menjadi satu keputusan tanpa alasan tertulis.

# 19. Register Keputusan FASE 16.7

| ID | Keputusan |
|---|---|
| DEC-16-01 | Apakah periode tahun ajaran boleh tumpang tindih? |
| DEC-16-02 | Apakah riwayat kelas harus berada dalam periode tahun ajaran? |
| DEC-16-03 | Apakah tahun ajaran boleh CLOSED ketika rekening masih OPEN? |
| DEC-16-04 | Apakah siswa LEFT harus menutup riwayat kelas aktif? |
| DEC-16-05 | Apakah rekening boleh dibuat untuk siswa LEFT? |
| DEC-16-06 | Apakah rekening boleh dibuat pada tahun ajaran CLOSED? |
| DEC-16-07 | Apakah transfer request hanya boleh dibuat atau diproses ketika tahun ajaran OPEN? |
| DEC-16-08 | Apakah satu siswa boleh memiliki beberapa transfer request PENDING dalam scope yang sama? |
| DEC-16-09 | Apakah rekening boleh dibuka kembali setelah tahun ajaran CLOSED? |
| DEC-16-10 | Apakah wali kelas hanya boleh mengakses siswa pada kelas aktif atau juga riwayat kelas dalam tahun ajaran? |
| DEC-16-11 | Apakah workflow request/approval diperlukan untuk reopen settlement? |
| DEC-16-12 | Bagaimana aturan lineage settlement pengganti harus diterapkan? |
| DEC-16-13 | Apakah alasan penolakan transfer wajib berdasarkan status dan ditegakkan di database? |
| DEC-16-14 | Bagaimana kebijakan tanggal transaksi historis diterapkan dalam batas tahun ajaran? |

Keputusan harus mencatat: pilihan final, alasan, dampak pada domain/database/service, dan fase implementasi jika diperlukan.

# 20. Implementation Gaps

Gap berikut tidak boleh dianggap sebagai keputusan kontrak:

- Penempatan kelas awal siswa: FASE 19.
- Validasi batas tanggal transaksi: FASE 21, setelah keputusan tanggal dikunci.
- Validasi jumlah dan tanggal transaksi sebelum mutasi.
- Validasi pembatalan transaksi dan saldo.
- Pemeriksaan konsistensi audit pada seluruh mutasi.
- Pemeriksaan overflow operasi uang.
- Implementasi constraint atau validasi yang telah disetujui melalui FASE 16.7.

# 21. Batas Perubahan FASE 16

FASE 16.6 hanya menyatukan hasil audit.

Tidak ada perubahan source code, migration, schema database, atau business behavior pada tahap ini.

Keputusan kontrak dilakukan di FASE 16.7. Implementasi hanya dilakukan setelah keputusan terkait jelas.

# 22. Kriteria Selesai

- Temuan lintas audit telah digabung dan duplikasi ditandai.
- Setiap temuan memiliki ID kanonis, klasifikasi, dan owner atau dependensi.
- Keputusan kontrak dipisahkan dari gap implementasi.
- Tidak ada keputusan yang dianggap selesai tanpa persetujuan eksplisit.
- Register keputusan DEC-16-01 sampai DEC-16-14 tersedia.
- Tidak ada perubahan source code atau migration.

# 23. Status Akhir

FASE 16.6 menghasilkan konsolidasi audit kontrak domain, database, lifecycle, authorization, dan finansial.

FASE 16.6 belum mengunci keputusan bisnis yang masih terbuka.

Langkah berikutnya: **FASE 16.7 — Decision Resolution**. Putuskan gap kontrak satu per satu, catat keputusan final, lalu tetapkan perubahan implementasi yang benar-benar diperlukan.

