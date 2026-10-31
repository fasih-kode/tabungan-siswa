# MQ-04 — Evaluasi Pemisahan Settlement Service

## Ringkasan dan rekomendasi

Baseline branch review, HEAD cb7add40796c81a508770c85b595fbfacecff8c2; working tree awal bersih. Review source/test read-only; tidak ada kode, migration, konfigurasi, atau database yang diubah/diakses.

settlement_service.go berisi 1.112 baris dan 21 fungsi/method. Semua terkait satu domain: preview/settlement, reopen, handover, readiness, otorisasi dan audit. Panjang file saja tidak membenarkan service atau abstraksi baru.

**Rekomendasi:** pertahankan satu settlementService, interface, Dependencies dan UOW. Bila supervisor menyetujui, pisahkan file internal menurut alur dalam package service tanpa mengubah method, transaksi, atau urutan operasi. Tahap pertama terkecil: pindahkan operasi baca/preview ke file baru.

**Risiko terpisah:** SettleStudentLeaving mengunci tahun/rekening, tetapi membaca status siswa lewat GetByID biasa. Reactivate mengunci row siswa tanpa lock rekening/tahun yang terlihat. Keduanya berpotensi berlomba tanpa lock row yang sama. Belum ada reproduksi runtime; tandai HIGH RISK/TEST GAP, pisahkan dari file split, minta keputusan supervisor dan uji interleaving PostgreSQL terpisah.

## Inventaris fungsi

Semua method memakai receiver settlementService dari Dependencies (RepositorySet + UnitOfWorkManager). Interface dan input/output di [contracts.go:855](../../internal/service/contracts.go#L855). Source: [settlement_service.go](../../internal/service/settlement_service.go).

| Fungsi/lokasi | Tanggung jawab; input → output | Dependency/UOW | Tes aktual |
|---|---|---|---|
| SettleYearEnd [L17](../../internal/service/settlement_service.go#L17) | Account/Wali/saldo preview → settlement+account; hitung ledger, cek lifecycle/blocker/replacement. | Year/account/transactions/request/scope/audit; UOW atomik. | Unit L419, L501-L579, L821, L970, L998; integration L16. |
| SettleStudentLeaving [L161](../../internal/service/settlement_service.go#L161) | Account/Wali/expected → settlement/account/student LEFT. | Year/account/scope/request/student/ledger/audit; UOW. | Unit L580/L672/L703; integration L16; race reaktivasi belum dites. |
| PreviewStudentLeaving [L288](../../internal/service/settlement_service.go#L288) | Account/Wali → saldo/waktu preview, read-only; settlement hitung ulang. | Account/year/scope/request/ledger. | OPEN L634; GRADUATED ditolak L655. |
| Get [L335](../../internal/service/settlement_service.go#L335) | ID/actor → detail settlement, account, movements, history, student, reopen, actor names. | Settlement/account/auth/movements/history/student/request/users; read. | Self/other siswa L731/L762; detail assertion parsial. |
| List [L405](../../internal/service/settlement_service.go#L405) | Account/actor/options → list settlement paginated. | Account/auth/settlement; read tanpa UOW. | Opsi L790. |
| RequestReopen [L441](../../internal/service/settlement_service.go#L441) | Wali ajukan reopen; validasi alasan/scope/state/successor. | Settlement/account/year/request/audit; UOW, year→account. | Integration L16, tanpa unit khusus. |
| DecideReopen [L519](../../internal/service/settlement_service.go#L519) | Admin berbeda requester approve/reject. | Request/year/update/audit; UOW, year→request. | Integration L16 self-approval/decision. |
| Reopen [L594](../../internal/service/settlement_service.go#L594) | Wali menjalankan reopen; settlement lama superseded. | Settlement/account/year/request/scope/audit; UOW, year→account→request. | Integration L16 reopen/correction/replacement/cash delta. |
| createReopenRequestAudit [L697](../../internal/service/settlement_service.go#L697) | Audit request before/after; JSON/AuditLogs dalam UOW caller; dipakai request/decision/reopen/completion. | Unit JSON L853; integration L16. |
| createAcademicYearAudit [L713](../../internal/service/settlement_service.go#L713) | Audit year before/after; JSON/AuditLogs dalam UOW; dipakai Reopen. | Integration L16; helper tidak dites langsung. |
| authorizeAccount [L730](../../internal/service/settlement_service.go#L730) | Admin valid, Wali scoped, siswa hanya diri sendiri; read-only. | Student/RequireStudentScope; dipakai Get/List/preview/leaving. | Self/other L731/L762; scope L541. |
| createSettlementAudit [L764](../../internal/service/settlement_service.go#L764) | Audit settlement before/after; JSON/AuditLogs dalam UOW; dipakai settle/reopen. | Rollback/JSON L821/L853; integration L16. |
| createSavingsAccountAudit [L797](../../internal/service/settlement_service.go#L797) | Audit account before/after; JSON/AuditLogs dalam UOW. | Dua audit L419; integration L16. |
| settlementPointers [L830](../../internal/service/settlement_service.go#L830) | Slice settlement menjadi pointer untuk List; helper murni. | Tanpa repo/UOW; tercakup tidak langsung L790. |
| PreviewYearEnd [L839](../../internal/service/settlement_service.go#L839) | Admin/Wali/account → balance/time; read-only, settle hitung ulang. | Account/year/request/scope/ledger/student. | Stale L970; integration L16. |
| ConfirmHandover [L883](../../internal/service/settlement_service.go#L883) | Settlement/Wali → movement handover/return sesuai delta. | Year/account/scope/movement/student/audit; UOW. | Year share→account update lock; integration L16. Race dua konfirmasi belum dites. |
| YearReadiness [L971](../../internal/service/settlement_service.go#L971) | Year/actor → counts/blockers; Admin global, Wali scoped; advisory read. | Year/readiness/late request/transfers; tanpa UOW. | Wali L910; Admin L939/L1041; student reject L962. |
| hasPendingLateRequest [L1040](../../internal/service/settlement_service.go#L1040) | Cek blocker request per account; repo nil = no blocker. | LeavingTransactionRequests; read; dipakai preview/settle. | Tidak langsung pada integration request. |
| authorizeResponsibleWali [L1047](../../internal/service/settlement_service.go#L1047) | Scope dari placement/assignment historis di akhir tahun. | History/teacher assignment; read tanpa lock. | Wali historis L541/L1070; reassignment race belum dites. |
| correctionContextForApprovedRequest [L1071](../../internal/service/settlement_service.go#L1071) | Batasi koreksi CLOSING ke approved scope → context. | Request/account/Wali; dipakai TransactionService pada UOW aktif. | Tidak langsung pada dua test target; integration L16 koreksi. |
| createSettlementMovementAudit [L1101](../../internal/service/settlement_service.go#L1101) | Audit movement + recipient student dalam UOW handover. | JSON/AuditLogs; dipakai ConfirmHandover. | Integration L16; recipient assertion tidak tampak di target tests. |

RepositorySet yang dipakai mencakup AcademicYears, SavingsAccounts, Students, Transactions, Settlements, StudentClassHistories, TeacherClassAssignments, LeavingTransactionRequests, ClassTransferRequests, ListTotals, Users dan AuditLogs; mutation memakai UOW. correctionContextForApprovedRequest juga dipakai TransactionService, jadi harus tetap tersedia dalam package service.

## Unit of Work dan urutan lock

beginTransaction memakai UOW dari context dan tidak membuat nested UOW; hanya pemilik commit/rollback. Caller luar bertanggung jawab jika service ikut transaksi. Sumber: [transaction_scope.go:97](../../internal/service/transaction_scope.go#L97), [UnitOfWork contract](../../internal/repository/contracts.go#L416). SQL locks: [year repository](../../internal/repository/postgres/academic_year_repository.go#L92), [account repository](../../internal/repository/postgres/savings_account_repository.go#L121), [request FOR UPDATE](../../internal/repository/postgres/settlement_workflow_repository.go#L209).

| Mutasi | Discovery reads tanpa lock | Row lock eksplisit berurutan | Tulis dalam UOW |
|---|---|---|---|
| SettleYearEnd | Account untuk year ID | Year FOR SHARE → account FOR UPDATE | Ledger/blocker; account update; settlement insert; optional reopen update; audit. Service L42-L55. |
| SettleStudentLeaving | Account untuk year ID | Year FOR SHARE → account FOR UPDATE; Student dibaca tanpa FOR UPDATE | Status/ledger; account update; settlement insert; audit. L186-L220. |
| RequestReopen | Settlement → account | Year FOR UPDATE → account FOR UPDATE | Successor/completed/request check; request create; audit. L459-L500. |
| DecideReopen | Request untuk year ID | Year FOR UPDATE → request FOR UPDATE | Successor/status/requester check; update; audit. L535-L567. Tak mengunci account. |
| Reopen | Settlement → account | Year FOR UPDATE → account FOR UPDATE → request FOR UPDATE | Update year, settlement lama, account dan audit. UPDATE settlement memperoleh write lock implisit. L610-L680. |
| ConfirmHandover | Settlement → account | Year FOR SHARE → account FOR UPDATE | Read movements; insert cash delta; student read; audit. L900-L961. |

Ini urutan call/SQL yang terlihat, bukan analisis penuh lock manager. Kontrak mensyaratkan saldo dibaca setelah lock, year-before-account, serialisasi reopen/successor, audit dan mutasi atomik: [FASE 23 §4-5, §12-13](../contracts/fase-23-year-end-settlement-contract.md#L86), [amendment pindah](../contracts/student-leaving-settlement-ui-amendment.md#L10). Integration membuktikan beberapa interleaving, bukan seluruh kemungkinan.

## Invariants yang wajib dijaga

| Invariant | Bukti |
|---|---|
| Admin monitoring/approval; Wali mengerjakan settlement/handover; siswa hanya baca diri. | RequireRole, authorizeAccount, authorizeResponsibleWali; FASE 23 §5. |
| Scope Wali memakai placement/assignment historis di tanggal akhir tahun. | authorizeResponsibleWali L1047-L1068; helper membaca tanpa lock eksplisit. |
| Regular settlement hanya year CLOSING/account OPEN; leaving hanya student LEFT dan year OPEN/CLOSING. | SettleYearEnd L57-L76; SettleStudentLeaving L192-L223; amendment pindah. |
| Saldo dari seluruh transaksi ACTIVE, dihitung lagi setelah locks; preview bukan otoritas; stale ExpectedBalance ditolak. | ListActiveBySavingsAccount + CalculateBalance + compare expected. |
| Saldo negatif ditolak, nol sah, account settled mencegah duplikasi. | Domain balance, unit L998, FASE 23 §3-4. |
| Pending late request memblokir settlement/readiness. | hasPendingLateRequest, YearReadiness, integration request flow. |
| Settlement positive dan handover terpisah; cash movement hanya delta; delta nol tidak membuat duplikat. | ConfirmHandover L929-L961, integration L16, FASE 23 §4/§14. |
| Reopen menyimpan settlement lama sebagai superseded, menghubungkan replacement, successor melarang reopen. | Reopen/SettleYearEnd, integration L16, FASE 23 §3/§9/§12. |
| State, movement dan audit atomik; jangan pecah menjadi UOW terpisah. | transaction_scope/defer rollback, unit audit rollback L821, integration. |
| Requester tidak menyetujui dirinya; successor dicek pada request, keputusan dan eksekusi. | DecideReopen/Reopen dan integration L16. |

### Risiko reaktivasi

SettleStudentLeaving membaca siswa via GetByID di settlement_service.go L217. Reactivate memakai GetByIDForUpdate lalu mengubah status dan membuat history di student_service.go L911-L926. Tidak tampak lock row yang sama. Dua transaksi mungkin melihat state berbeda dan commit bersamaan; belum direproduksi, jadi bukan bug terkonfirmasi. Tandai HIGH RISK + TEST GAP. Supervisor perlu menentukan hasil yang benar, lalu review lock order StudentService/SettlementService/TransactionService dan test race PostgreSQL. Jangan masukkan ke file split.

## Maintainability dan cakupan tes

Satu file memuat public workflows, read/preview, lifecycle reopen, audit helpers, authorization dan readiness. Beban navigasi nyata; duplikasi tanggung jawab atau kebutuhan beberapa service tidak terbukti. Dependencies bersifat luas karena RepositorySet dibagi antar service. Membuat service baru dapat menambah wiring dan membuat transaksi lintas service lebih sulit dilacak.

Test unit meng-cover status, saldo stale, role/scope, audit rollback/JSON, readiness, nol/duplikasi dan beberapa alur leaving. Satu integration test settlement workflow meng-cover settlement, handover, close, reopen, successor guard, koreksi dan sebagian interleaving. Test unit memakai fake sehingga tidak membuktikan SQL locking. Kekurangan yang relevan: race Reactivate vs SettleStudentLeaving; perubahan assignment historis vs otorisasi; dua ConfirmHandover bersamaan; unit khusus RequestReopen/DecideReopen. Cakupan HTTP role/status juga tidak menjadi bukti dari service tests.

Referensi: [unit tests](../../internal/service/settlement_service_unit_test.go), [workflow integration](../../internal/service/settlement_workflow_integration_test.go), [reactivation](../../internal/service/student_service.go#L911), [contract tests](../../internal/service/contracts.go#L855).

## Pilihan desain

| Opsi | Penilaian |
|---|---|
| A. Satu service/interface; beberapa file dalam package service | Direkomendasikan. Tidak mengubah wiring, receiver, API atau transaksi. Kelompokkan operasi berdasarkan alur dan pertahankan helper bersama di package. |
| B. Pecah menjadi beberapa service/collaborator | Belum direkomendasikan. Membutuhkan perubahan Dependencies/wiring dan harus membuktikan alur transaksi, helper lintas service serta urutan lock tetap jelas. Panjang file saja bukan manfaat yang cukup. |
| C. Hanya tambah komentar/header di satu file | Risiko paling rendah, tetapi tidak banyak mengurangi biaya navigasi. Bisa menjadi fallback bila file split tidak disetujui. |

### Usulan tahap implementasi (belum disetujui)

1. **MQ-04-IMPL-01: kelompok baca dan preview.** Pindahkan tanpa mengubah body `PreviewStudentLeaving`, `Get`, `List`, `PreviewYearEnd`, `YearReadiness`, dan `settlementPointers` ke `settlement_service_read.go`. Tidak mengubah nama, signature, statement, query atau dependency. Risiko rendah; tetap perlu build, vet dan unit tests service. Integration test tidak wajib untuk pemindahan murni yang diff-nya membuktikan hanya perpindahan, tetapi jalankan jika perubahan meluas ke perilaku.
2. **Kelompok reopen.** Jika tahap pertama lulus review, pindahkan `RequestReopen`, `DecideReopen`, `Reopen`, dan helper audit yang eksklusif untuk workflow tersebut. `correctionContextForApprovedRequest` harus tetap dapat dipanggil dari TransactionService di package yang sama. Wajib integration test reopen/successor/audit.
3. **Kelompok settlement dan penyerahan.** Pindahkan settlement tahun, settlement siswa pindah, konfirmasi penyerahan serta helper yang tepat. Jangan ubah lock order/UOW. Wajib integration test ledger, stale balance, blocker, role/scope, handover delta, rollback, dan concurrency yang relevan.

Setiap tahap perlu persetujuan supervisor tersendiri sebelum source code berubah. Bila review menemukan kebutuhan mengubah semantic, stop dan ajukan perubahan scope. Perubahan lock, aturan uang, role atau status memerlukan keputusan produk/teknis dan regression test tambahan.

## Risiko dan syarat PASS

Risiko utama ialah perubahan urutan lock atau cakupan transaksi saat pemisahan method/helper, serta kesan salah bahwa pemindahan file memperbaiki celah concurrency. Kriteria tahap file split: diff source hanya pemindahan kode; public signatures dan receiver tidak berubah; semua call/return/error/UOW/lock/audit identik; `gofmt`, build, vet, unit dan integration relevan lulus; tidak ada migration atau dependency baru.

Review ini tidak menjalankan test, membuka database, atau mengubah source. Sesuai ruang lingkup MQ-04, pemeriksaan yang perlu dijalankan pada dokumen saja adalah `git diff --check` dan validasi tautan/referensi. Tidak ada perubahan implementation yang disetujui atau dibuat.

## Sumber otoritatif

- [Settlement service](../../internal/service/settlement_service.go)
- [Settlement contract](../../internal/service/contracts.go#L855)
- [Transaction scope](../../internal/service/transaction_scope.go#L97)
- [Repository interfaces and UOW](../../internal/repository/contracts.go#L416)
- [FASE 23 settlement contract](../contracts/fase-23-year-end-settlement-contract.md)
- [Student leaving settlement amendment](../contracts/student-leaving-settlement-ui-amendment.md)
- [Maintainability roadmap](MAINTAINABILITY_ROADMAP.md)
