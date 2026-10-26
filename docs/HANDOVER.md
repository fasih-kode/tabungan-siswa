# Handover Proyek

> Dokumen operasional ini harus diperbarui berdasarkan status Git dan bukti pemeriksaan terbaru sebelum pekerjaan diakhiri. Jangan mengandalkan tanggal atau klaim lama tanpa verifikasi.

## Snapshot sebelum Project Memory

- Branch: `main`.
- HEAD lokal: `55492cb` — `docs(fase-23): lock year-end settlement contract`.
- Ref `origin/main` lokal yang terlihat: `fcc9398`; status remote belum diverifikasi melalui fetch.
- Working tree sebelum pembuatan Project Memory: 24 file terlacak berubah dan 6 file belum terlacak. Perubahan berpusat pada implementasi FASE 23, termasuk migrasi `000011`, template, route, service, repository, dan tes integrasi.
- Perubahan tersebut belum di-commit. Jangan menimpa atau mengasumsikan semuanya berasal dari tugas yang sama.

## Status fase yang tercatat

- FASE 21 memiliki implementation lock commit `fe21e78`.
- Kontrak FASE 22 menyatakan implementasi terverifikasi, namun owner review dan implementation lock record masih tertunda.
- FASE 23: kontrak dikunci; implementasi pada working tree berstatus berjalan, review dan verifikasi masih tertunda. Kehadiran file bukan bukti seluruh acceptance criteria terpenuhi.
- FASE 24–33 belum dinyatakan selesai oleh bukti yang ditinjau pada snapshot audit.

Rujuk kontrak fase untuk status terbarunya sebelum melanjutkan.

## Pemeriksaan audit dan Project Memory

- Audit repository dilakukan read-only sebelum dokumentasi dibuat.
- `git diff --check` dijalankan saat audit dan tidak melaporkan masalah.
- Setelah pembuatan Project Memory, `git diff --check` kembali lulus; pemeriksaan
  whitespace dan tautan lokal pada sembilan file Markdown terpilih juga lulus.
- Tes aplikasi tidak dijalankan pada audit maupun pembuatan Project Memory.
- Migrasi `000011` tidak dijalankan; status penerapan database tidak diketahui.
- Project Memory menambahkan `AGENTS.md`, `docs/PRD.md`, `docs/ARCHITECTURE.md`, `docs/BUSINESS_RULES.md`, `docs/TESTING.md`, `docs/DECISIONS.md`, dan dokumen ini. README diberi tautan ke dokumen tersebut.
- Diff akhir tetap perlu ditinjau sebelum dokumentasi dianggap siap.

## Pekerjaan berikutnya

1. Tinjau Project Memory dan diff dokumentasi; pastikan batas sumber otoritatif dan status FASE 23 tetap akurat.
2. Verifikasi implementasi FASE 23 terhadap acceptance criteria kontraknya, termasuk migration up/down pada database disposable dan tes terproteksi.
3. Perbarui status FASE 22/23 hanya setelah bukti review dan verifikasi tersedia.
4. Perbarui handover ini dengan hash commit, status tes, blocker, dan langkah berikut terbaru sebelum sesi berakhir.

## Risiko dan batasan yang diketahui

- README sebelumnya hanya menyebut prerequisite migrasi FASE 19 dan 20; belum ada satu prosedur migrasi lengkap untuk lingkungan baru.
- Test suite service/PostgreSQL membutuhkan database tes yang di-allowlist; gunakan `make test-integration`.
- Tidak ada bukti dari audit ini bahwa migrasi working tree telah diterapkan, tes FASE 23 lulus, atau aplikasi siap produksi.

## Handover setelah review implementasi FASE 23

- Branch/HEAD awal pemeriksaan: `main` / `55492cb` (`docs(fase-23): lock year-end settlement contract`). Lihat catatan terbaru di bagian akhir untuk commit implementation lock.
- Perbaikan review: detail settlement kini memuat history akun dan permintaan reopen terbaru (termasuk status final); implementasi repository `GetLatestReopenRequest` tersedia. Halaman preview menampilkan waktu hitung saldo. Overview tidak lagi menawarkan preview pada tahun ajaran yang belum berstatus `CLOSING`, mengarahkan ke daftar transfer tertunda untuk tahun terpilih, dan memberi label readiness sesuai cakupan Wali.
- Form konfirmasi penyerahan tidak lagi meminta referensi bukti opsional karena formulir tersebut tidak diwajibkan kontrak. Detail permintaan reopen menampilkan waktu/alasan keputusan Admin bila tersedia.
- Pemeriksaan yang dijalankan pada sesi ini: `git diff --check` lulus; `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...` lulus; `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...` lulus; `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/domain -count=1` lulus; `npm run test:instant-search` lulus (1 tes).
- `go test ./internal/http ./internal/domain -count=1` belum lulus penuh: tes HTTP yang memakai `httptest.NewServer` gagal karena sandbox menolak bind ke loopback (`operation not permitted`); domain tetap lulus. Perintah `go test ./internal/http -run 'TestSettlement' -count=1` berhasil dijalankan tetapi tidak menemukan tes settlement (`no tests to run`).
- `make test-integration` tidak dapat berjalan karena `psql` gagal terhubung ke database uji (pesan error kosong dari skrip lingkungan). Migrasi `000011` tidak dijalankan. Karena itu alur database, konkurensi, integritas migration up/down, dan integrasi end-to-end FASE 23 belum tervalidasi.
- Pekerjaan tersisa sebelum FASE 23 dapat dinyatakan terverifikasi: siapkan akses ke DB disposable yang di-allowlist dan ulangi `make test-integration`; tambahkan tes HTTP khusus halaman settlement; review kontrak vs perilaku lebih lanjut, terutama keterbacaan identitas pelaku di halaman detail (saat ini UUID) dan representasi riwayat settlement yang dikembalikan service tetapi belum ditampilkan.

## Verifikasi lanjutan setelah izin lingkungan tersedia

- Izin jaringan sesi diaktifkan untuk loopback dan akses PostgreSQL.
- Tes HTTP dan domain berhasil: `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/http ./internal/domain -count=1`.
- Penyebab percobaan `make test-integration` sebelumnya adalah Go mencoba menulis cache ke `/home/fasih/.cache/go-build`, yang read-only di sandbox. Dengan cache di `/tmp`, perintah `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration` berhasil untuk seluruh package; guard mengonfirmasi database `tabungan_siswa_fase168_clean_test`.
- Race/integration suite juga berhasil: `GOCACHE=/tmp/tabungan-siswa-gocache GOFLAGS=-race make test-integration`, semua package lulus.
- Suite tersebut berjalan dengan keadaan schema test yang tersedia. Tidak ada migration yang dijalankan dalam sesi ini, jadi hasil ini tidak membuktikan jalur upgrade/rollback migration `000011`.

## Migration FASE 23 pada database disposable

- Tindak lanjut pemeriksaan migration menemukan metadata `schema_migrations` awal pada `tabungan_siswa_fase168_clean_test` adalah versi 9 bersih, tetapi objek versi 10/11 sudah ada: fungsi trigger telah berisi logika FASE 23, dua tabel FASE 23 tersedia, dan constraint audit telah mencantumkan entitas reopen request.
- Percobaan migration `up` normal berhenti pada migration 10 karena fungsi `enforce_academic_year_create_after_previous_closed` sudah ada. Tidak ada DDL migration 10 yang diterapkan oleh percobaan tersebut; metadata sempat menjadi `10 dirty`, lalu dikembalikan ke `9 clean` (kondisi yang tercatat sebelum percobaan).
- Sebelum siklus rollback, kedua tabel FASE 23 diperiksa dan kosong. Metadata diselaraskan ke versi 11 agar migration 11 dapat diuji terhadap objek yang sudah ada. `migrate down 1` berhasil (versi 10 bersih), dilanjutkan `migrate up` yang berhasil (versi 11 bersih); kedua tabel kembali tersedia. Database ditinggalkan pada versi 11.
- Setelah siklus tersebut, `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration` dan `GOCACHE=/tmp/tabungan-siswa-gocache GOFLAGS=-race make test-integration` lulus seluruh package; `git diff --check` lulus.
- Batas verifikasi: migration 11 rollback/reapply berhasil. Migration 10 belum diuji dari versi 9 yang bersih karena objeknya telah ada sebelum tugas ini; jangan menganggap metadata awal mencerminkan schema awal yang sebenarnya. Database disposable sekarang tercatat versi 11 bersih.

## Penyelesaian tindak lanjut review

- Siklus migration tambahan berhasil di database tes disposable: dari versi 11 dilakukan `down 1` dua kali hingga versi 9 bersih, lalu `up 1` dua kali kembali ke versi 11 bersih. Sebelum rollback, tabel FASE 23 terverifikasi kosong. Status akhir `schema_migrations`: database `tabungan_siswa_fase168_clean_test`, versi 11, `dirty=false`.
- Catatan penting: database itu awalnya melaporkan versi 9 tetapi objek versi 10/11 sudah ada. Karena itu metadata diselaraskan ke versi 11 setelah objek diverifikasi, kemudian siklus migrasi 10 dan 11 diuji turun/naik. Ini bukan bukti tentang database development atau produksi.
- Halaman detail settlement sekarang menampilkan nama username pelaksana/konfirmator jika tersedia, riwayat settlement akun, serta pelaku keputusan reopen. UUID tetap menjadi fallback bila nama pelaku tidak tersedia. Tes baru memverifikasi rendering halaman detail dan template aplikasi.
- Verifikasi akhir: `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration`, `GOCACHE=/tmp/tabungan-siswa-gocache GOFLAGS=-race make test-integration`, `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...`, `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...`, `npm run test:instant-search`, dan `git diff --check` semuanya lulus. Tes frontend mencatat 7 kasus lulus.
- Tidak ada perubahan ke database development/produksi dan tidak ada commit. Perubahan tetap berada di working tree untuk ditinjau.

## Review diff akhir dan keputusan status FASE 23

- Review final menemukan dan memperbaiki perilaku preview stale: ketika transaksi berubah sesudah preview, endpoint settlement mengembalikan 409 dengan saldo serta waktu pratinjau terbaru dan form konfirmasi baru. Regresi dicakup oleh tes HTTP dan tes render template.
- Detail settlement kini menampilkan riwayat settlement akun dan username pelaku (UUID menjadi fallback); tes handler dan template mencakup tampilan tersebut.
- Status review: **belum lock-ready**. Integrasi utama mencakup settlement/handover/tutup, reopen, koreksi/re-settlement, rekonsiliasi, dan race dengan tahun penerus; namun matriks tes HTTP dan service di §13 kontrak belum seluruhnya tersedia. Daftar gap dan bukti review ada di [DECISIONS.md](DECISIONS.md).
- Tidak ada status kontrak/roadmap yang diubah untuk menyatakan FASE 23 selesai, dan tidak ada commit yang dibuat.

## Tes acceptance tambahan setelah review checkpoint

- Ditambahkan tes unit/service untuk readiness Admin/Wali, cakupan wali dan transfer blocker, akun tanpa atribusi, penolakan siswa/Admin, scope historis, stale preview, saldo nol, settlement duplikat, dan blocker audit.
- Tes integration diperluas: settlement bersamaan dengan penarikan dan duplicate settle, settlement/handover versus year close, handover student-leaving positif, approval dan rejection reopen, self-approval ditolak, Admin correction ditolak, koreksi/re-settlement, dan successor race. Fixture dibersihkan setelah tes.
- Tes HTTP/template diperluas untuk overview Admin/Wali, unresolved group, safe error, stale preview 409 dengan CSRF dan saldo baru, tampilan kosong/siap/blocked, detail pelajar read-only, history settlement, dan nama pelaku.
- Pemeriksaan akhir setelah penambahan tes: seluruh integration suite, race suite, `go vet`, build, frontend (7/7), dan `git diff --check` lulus.
- Review teknis saat itu tidak menemukan blocker; tindak lanjut dan status terbarunya dicatat di bagian akhir dokumen ini.

## Verifikasi akhir sebelum implementation lock — 2026-10-20

- Atas instruksi owner, tes acceptance dijalankan ulang sebelum penutupan FASE 23.
- `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration` lulus.
- `GOCACHE=/tmp/tabungan-siswa-gocache GOFLAGS=-race make test-integration` lulus.
- `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...` dan `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...` lulus.
- `npm run test:instant-search` lulus (7 tes); `git diff --check` lulus.
- Migration 10 dan 11 sebelumnya telah diuji upgrade/rollback pada database disposable; database akhir versi 11 bersih. Database development/produksi tidak disentuh.
- Bukti acceptance yang dijalankan lulus. Browser E2E tidak dijalankan. Commit implementation dan lock record dibuat; status akhir dicatat di bawah.

## FASE 23 ditutup — 2026-10-20

- Implementation commit: `9a0a0d0` (`feat(fase-23): implement year-end settlement workflows`). Catatan lock: [fase-23-implementation-lock.md](contracts/fase-23-implementation-lock.md).
- Acceptance checks akhir lulus: tes integrasi PostgreSQL, tes integrasi dengan race detector, `go vet ./...`, `go build ./...`, tes frontend (7/7), dan `git diff --check`.
- Migration 10 dan 11 telah diuji upgrade/rollback pada database disposable; database uji ditinggalkan pada versi 11 bersih. Tidak ada perubahan pada database development/produksi.
- Tidak ada browser E2E; aplikasi belum dinyatakan siap produksi hanya berdasarkan tes ini.
- Status FASE 23: **implementation lock dibuat**. Perubahan lanjutan yang mengubah perilaku FASE 23 perlu kontrak/amendment baru dan tes terkait.

## E2E browser manual setelah implementation lock — 2026-10-20

- Repository tidak menyediakan harness Playwright/Selenium; skenario dijalankan melalui browser lokal dengan server memakai database `tabungan_siswa_fase168_clean_test`.
- Wali berhasil login, membuka overview settlement, melihat satu akun OPEN dengan saldo Rp 2.500, membuka preview, membuat settlement, dan mengonfirmasi penyerahan. Overview berubah menjadi `Belum settlement 0`, `Menunggu penyerahan 0`, `Selesai 1`, dan `Cakupan Anda siap`.
- Admin berhasil login, melihat detail tahun, menjalankan `Tutup Tahun Ajaran`, dan UI menampilkan status `Ditutup`.
- Setelah E2E, server dihentikan dan hanya fixture dengan ID `f2300000-...` yang dibuat untuk skenario ini dibersihkan dari database tes. Verifikasi menunjukkan fixture tidak tersisa; dua akun uji yang ada sebelumnya tetap ada. Database development/produksi tidak disentuh.
- Hasil: seluruh alur manual browser yang dicoba lulus. Ini tidak menciptakan regression suite browser otomatis; tes HTTP, template, integration, dan concurrency otomatis tetap menjadi bukti berulang utama.

## Harness E2E browser otomatis — 2026-10-20

- Ditambahkan Playwright Test dengan Chromium, konfigurasi server lokal otomatis, tes alur Wali→settlement→penyerahan→readiness→Admin→tutup tahun, dan fixture Go ber-guard.
- Jalur pengguna: `npm install`, `npm run test:e2e:install`, lalu `npm run test:e2e`. Petunjuk target database dan pembersihan ada di [TESTING.md](TESTING.md).
- Verifikasi: `GOCACHE=/tmp/tabungan-siswa-gocache PLAYWRIGHT_BROWSERS_PATH=/tmp/tabungan-siswa-playwright-browsers npm run test:e2e` lulus (1/1). Fixture dibersihkan otomatis dari `tabungan_siswa_fase168_clean_test`; development/produksi tidak disentuh.
- `git diff --check` lulus. Harness ini berjalan serial dan tidak boleh dijalankan paralel dengan Go integration suite pada database tes yang sama.

## FASE 24 — implementasi administrasi akun berjalan — 2026-10-20

- Owner menyetujui rancangan kebijakan dalam [kontrak FASE 24](contracts/fase-24-user-administration-contract.md). Kontrak menjelaskan pengelolaan akun, role tetap, bootstrap Admin, larangan nonaktifkan diri/Admin, dan pencabutan sesi saat nonaktif/reset.
- Implementasi menambahkan status akun aktif, pemeriksaan status saat autentikasi dan setiap request, perubahan username, aktivasi/nonaktif, serta reset sementara yang mewajibkan ganti password.
- Migration `000012` diterapkan pada database disposable `tabungan_siswa_fase168_clean_test`; status schema yang diverifikasi setelahnya versi 12, bersih. Database development/produksi tidak disentuh.
- `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration` lulus seluruh package setelah migration 12. Tes mencakup perubahan status, pencabutan sesi saat reset/nonaktif, larangan perubahan role, penolakan sesi akun nonaktif, dan PostgreSQL rollback nyata saat pencabutan sesi sengaja gagal.
- Harness browser diperbarui untuk menolak schema selain v12. Playwright memeriksa alur settlement FASE 23 dan alur Admin ubah username, nonaktifkan/aktifkan, reset password, login dengan password sementara, dan ganti password; hasil 2/2 lulus.
- Verifikasi FASE 24: `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration` lulus; `GOCACHE=/tmp/tabungan-siswa-gocache GOFLAGS=-race make test-integration` lulus; `go vet ./...`, `go build ./...`, `npm run test:instant-search` (7/7), dan `git diff --check` lulus.
- Migration 12 diterapkan dan diuji pada DB disposable `tabungan_siswa_fase168_clean_test`, versi 12 bersih. Fixture browser dihapus otomatis.
- Siklus migration 12 `down 1` lalu `up` berhasil pada database disposable. Tepat sebelum siklus, backup status aktif/nonaktif sementara mencatat 0 akun nonaktif; backup diverifikasi lalu dihapus. Schema akhir versi 12 bersih. Tidak ada data development/produksi yang disentuh.
- Owner menerima implementasi FASE 24 pada 2026-10-21. Setelah review dan verifikasi akhir, fase dikunci pada implementation commit `e233883`; catatan lock ada di [fase-24-implementation-lock.md](contracts/fase-24-implementation-lock.md).

## Perbaikan login Admin lokal — 2026-10-20

- Laporan login Admin menampilkan HTTP 500. Pemeriksaan read-only pada `.env` memastikan aplikasi memakai database `tabungan_siswa_dev`, yang masih di schema versi 11 dan belum memiliki kolom `users.is_active` yang dibaca kode FASE 24.
- Migration `000012_user_activation` berhasil diterapkan ke database development tersebut; verifikasi sesudahnya menunjukkan versi 12, `dirty=false`, dan kolom `users.is_active` tersedia. Migration menambahkan kolom dengan default `TRUE` serta indeks sesi aktif.
- GET `/login` pada server lokal merespons HTTP 200. Screenshot berikutnya menunjukkan Admin berhasil login dan telah melewati kewajiban ganti password; database kini mencatat `password_change_required=false` untuk akun `admin`.
- Database tes disposable dan database produksi tidak disentuh oleh perbaikan login tersebut. Development database kini berada pada schema versi 12; database produksi tidak disentuh.

## Laporan akses Pengawasan Tabungan — 2026-10-20

- Screenshot menunjukkan GET `/savings` membalas 403 `Forbidden` untuk Admin. Kode handler saat ini mengizinkan `ADMIN`; query read-only juga mengonfirmasi akun `admin` ber-role `ADMIN` dan tahun ajaran `2026/2027` berstatus `OPEN`.
- Akar masalah ditemukan: halaman mengirim ID tahun aktif tetapi tanpa ID kelas ke `StudentService.List` untuk Admin. Layanan menganggap kombinasi filter itu sebagai `ErrScopeViolation`, yang tampil sebagai 403.
- Handler kini meminta roster global untuk Admin (tanpa filter tahun/kelas); tahun yang dipilih tetap digunakan untuk membaca akun tabungan. Wali tetap memakai roster dengan scope tahun/penugasan.
- Tes unit memastikan Admin meminta roster global. E2E memverifikasi Admin membuka `/savings` ketika tahun ajaran uji masih terbuka; pemeriksaan ditempatkan sebelum Admin menutup tahun. E2E `GOCACHE=/tmp/tabungan-siswa-gocache PLAYWRIGHT_BROWSERS_PATH=/tmp/tabungan-siswa-playwright-browsers npm run test:e2e` lulus (2/2), fixture dibersihkan otomatis dari DB disposable.
- Percobaan E2E awal yang dilakukan setelah tahun uji tertutup tidak menguji jalur tahun aktif; assertion tersebut kini dipindahkan ke skenario sebelum tahun ditutup.

## Root URL mengarah ke halaman login — 2026-10-21

- Penyebab: router hanya mendaftarkan `GET /login`; tidak ada handler untuk path root `/`, sehingga `http.ServeMux` mengembalikan 404.
- Ditambahkan route exact `GET /` yang mengalihkan ke `/login` dengan status 303, serta tes router yang memverifikasi status dan lokasi redirect.
- `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/http -run 'TestNewRouterPublicLoginRoute$' -count=1` lulus; `git diff --check` lulus.

## FASE 24 implementation lock — 2026-10-21

- Owner menyatakan seluruh alur lokal sudah berfungsi dan menginstruksikan commit serta push.
- Implementation commit: `e233883` (`feat(fase-24): administer user accounts`). Catatan lock: [fase-24-implementation-lock.md](contracts/fase-24-implementation-lock.md).
- Verifikasi sebelum lock lulus: integration tests normal dan `-race`, `go vet ./...`, `go build ./...`, frontend (7/7), Playwright E2E (2/2), tes router root redirect, dan `git diff --check`.
- E2E menggunakan database tes disposable versi 12 dan menghapus fixture otomatis. Browser server dihentikan setelah tes. Database development ada pada schema v12; produksi tidak diubah.

## FASE 25 — implementasi dashboard & reporting (belum lock) — 2026-10-21

- Kontrak disetujui owner berdasarkan tiga keputusan: memakai satu tahun ajaran aktif saja, setor/tarik dihitung sebagai total nominal transaksi `ACTIVE` (transaksi batal dikecualikan), dan laporan berupa tabel web tanpa ekspor. Tahun `CLOSING` tetap menjadi tahun aktif selama belum `CLOSED`, agar dashboard/laporan tersedia dalam workflow settlement. Rincian ada di [kontrak FASE 25](contracts/fase-25-dashboard-reporting-contract.md).
- Dashboard placeholder diganti metrik per role dan aktivitas terbaru Siswa. Empat laporan ditambahkan: siswa, kelas, tahun ajaran, dan settlement. Query reporting terpisah menerapkan scope Admin/Wali/Siswa server-side dan menghitung total dari satu tahun saja. Tidak ada migration.
- Tes repository menguji agregasi ledger aktif serta scope query; tes service mencakup tahun `CLOSING`, keadaan tanpa tahun aktif, dan larangan SISWA membuka direktori laporan.
- Verifikasi 2026-10-21: `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration` lulus seluruh package; `GOCACHE=/tmp/tabungan-siswa-gocache GOFLAGS=-race make test-integration` lulus; `go build ./...`, `go vet ./...`, `npm run test:instant-search` (7/7), Playwright E2E (2/2), dan `git diff --check` lulus. Playwright mencakup dashboard dan empat route laporan sebelum alur settlement; fixture dibersihkan otomatis dari `tabungan_siswa_fase168_clean_test`.
- Satu percobaan E2E awal gagal karena tahun `CLOSING` dianggap tidak aktif. Definisi kontrak dan service diperbaiki; E2E berikutnya lulus. Server test berhenti setelah script selesai. DB development/produksi tidak disentuh.
- Status: acceptance otomatis yang dijalankan lulus. Owner review akhir, implementation lock record, commit, dan push belum dilakukan. Working tree berisi perubahan FASE 25; jangan menyatakan fase terkunci.
- Langkah berikutnya: review visual halaman laporan/dashboard bersama owner, lakukan owner acceptance, lalu buat implementation lock dan commit setelah instruksi eksplisit.

## Klarifikasi UX FASE 25: Penutupan, histori, dan laporan — 2026-10-21

- Owner menyetujui pemisahan fungsi Admin: **Penutupan Tahun** untuk hambatan kesiapan per Wali/Kelas; **Histori Tabungan** untuk membuka rekening berdasarkan tahun ajaran dan kelas, dengan filter hanya siswa pemilik rekening; **Laporan** sebagai indeks empat laporan. Addendum disimpan di [kontrak FASE 25](contracts/fase-25-dashboard-reporting-contract.md).
- Menu Admin kini memisahkan Penutupan Tahun dan Histori Tabungan. Overview penutupan menampilkan status “Menghambat penutupan” pada grup dengan rekening belum settlement atau penyerahan belum selesai, beserta rincian yang dapat ditelusuri. Histori Admin memakai penempatan siswa pada tahun terpilih, pilihan kelas, serta filter pemilik rekening. Admin tetap baca-saja.
- `/reports` menampilkan indeks laporan Siswa, Kelas, Tahun Ajaran, dan Settlement. Filter laporan tetap terbatas pada tahun aktif; cakupan histori tidak mengubah kontrak laporan.
- Tes yang dijalankan pada sesi ini: `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration` lulus dan guard memverifikasi `tabungan_siswa_fase168_clean_test`; `go vet ./...`, `go build ./...`, tes HTTP termasuk filter histori dan indeks laporan, `npm run test:instant-search` (7/7), Playwright E2E (2/2), dan `git diff --check` lulus. E2E menghapus fixture dan menghentikan server otomatis. Tidak ada migration atau perubahan database development/produksi.
- Saat percobaan E2E pertama, port 8080 dipakai server lokal lain. Sesuai izin owner sebelumnya, proses `web` PID 43328 dihentikan agar harness menjalankan server aplikasi yang benar; E2E kemudian lulus.
- FASE 25 belum dikunci dan belum di-commit. Working tree tetap memuat implementasi dashboard/laporan sebelumnya beserta klarifikasi UX ini; pertahankan seluruh perubahan tersebut saat sesi berikutnya.

## Format uang Indonesia pada semua halaman — 2026-10-21

- Atas permintaan owner, formatter bersama `formatRupiah` kini menampilkan prefiks `Rp. ` dan pemisah ribuan titik; contoh `Rp. 10.000`. Semua tampilan saldo, transaksi, settlement/penyerahan, dashboard, dan laporan menggunakannya. Nilai yang dikirim pada form tetap angka integer agar validasi dan pengiriman transaksi tidak berubah; input nominal saat itu masih memakai kontrol angka native.
- Tes formatter mencakup nol, ribuan, dan jutaan; ekspektasi tes tabungan, transaksi, settlement, serta rendering template diperbarui.
- Verifikasi: `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/http ./internal/app -count=1`, `go build ./...`, Playwright E2E (2/2), dan `git diff --check` lulus. E2E memakai database disposable dan membersihkan fixture.
- Untuk menjalankan E2E, port 8080 kembali dipakai server `web` PID 48162. Sesuai izin owner sebelumnya, proses itu dihentikan; server yang diluncurkan harness berhenti otomatis setelah E2E. Tidak ada migration atau perubahan database development/produksi.
- Perubahan tetap di working tree tanpa commit/push, bersama perubahan FASE 25 sebelumnya.


## Pemisah ribuan otomatis pada input uang — 2026-10-21

- Input setoran dan penarikan pada halaman detail tabungan serta input koreksi transaksi kini menampilkan pemisah ribuan titik saat diketik (contoh `20.000`). JavaScript mengubah nilai tampilan menjadi input teks dan mengirim angka integer tanpa pemisah melalui input tersembunyi. Bila JavaScript tidak tersedia, kontrol angka native tetap dipakai.
- Formatter mempertahankan posisi caret saat mengetik/mengedit, menerima input tempel bertanda baca, dan membatasi input sampai 13 digit sesuai batas transaksi yang sudah berlaku. Tidak ada perubahan pada aturan nominal atau validasi server.
- Tes baru: unit JS untuk pengelompokan, paste, dan batas digit; Playwright untuk format tampilan dan payload integer; tes rendering template memastikan halaman memuat script dan menandai input terkait.
- Verifikasi sesi: `npm run test:rupiah-input` lulus (3/3); `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/http ./internal/app -count=1` lulus; Playwright E2E lulus (3/3); `git diff --check` lulus. E2E memakai database disposable dan membersihkan fixture.
- Perubahan tetap di working tree bersama implementasi FASE 25; belum di-commit atau di-push.

## Permintaan transaksi terlewat sebelum siswa pindah — 2026-10-21

- Owner menyetujui alur Wali mengajukan rincian transaksi yang benar-benar terjadi sebelum pindah, Admin menyetujui/menolak dengan alasan, lalu Wali mencatat rincian yang disetujui satu kali. Batas tahap ini: rekening belum diselesaikan dan tahun belum `CLOSED`; saldo keluar tetap lewat settlement. Kebijakan: [amendment permintaan transaksi terlewat](contracts/late-transaction-request-amendment.md).
- Ditambahkan migration 13 untuk `students.leaving_date`, tabel permintaan dengan state/approval/audit, dan `transactions.late_request_id` yang unik. Tanggal perpindahan baru disimpan saat Admin menandai siswa `LEFT`; data `LEFT` lama dibiarkan tanpa tanggal (tidak ditebak dari riwayat kelas), sehingga belum dapat mengajukan sampai tanggal terverifikasi tersedia.
- FE: Wali mengajukan di detail buku tabungan, Admin meninjau di menu **Transaksi Terlewat**, Wali mengeksekusi permintaan approved dari inbox yang sama. Backend mengunci tahun/rekening/siswa/permintaan, memeriksa role/scope/status/tanggal/balance kronologis, merekam audit, dan menghalangi settlement rekening serta penutupan tahun saat ada request `PENDING`/`APPROVED`.
- Migration 13 diterapkan pada database disposable `tabungan_siswa_fase168_clean_test` setelah memverifikasi schema 12 bersih, lalu diverifikasi schema 13 bersih. Setelah memastikan `.env` menunjuk ke `tabungan_siswa_dev` versi 12 bersih, migration additif yang sama diterapkan di sana dan diverifikasi versi 13 bersih. Produksi tidak disentuh. Baris siswa `LEFT` lama tetap memiliki `leaving_date=NULL` agar tanggal tidak ditebak; alur request menolaknya sampai tanggal terverifikasi tersedia.
- Verifikasi terakhir: `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration` lulus semua package termasuk PostgreSQL dan lifecycle request; target diverifikasi `tabungan_siswa_fase168_clean_test`. `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...`, `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...`, `npm run test:instant-search` (7/7), `npm run test:rupiah-input` (3/3), E2E Playwright (3/3), dan `git diff --check` lulus. Browser E2E memakai DB disposable dan membersihkan fixture otomatis.
- Referensi bukti yang sama kini ditolak untuk request PENDING, APPROVED, maupun COMPLETED; request REJECTED dapat diajukan ulang. Tes integrasi memastikan bukti yang sudah selesai tidak dapat digunakan kembali.
- Working tree tetap mencakup FASE 25 dan perbaikan sebelumnya yang belum di-commit. Jangan menghapus atau meng-commit perubahan tersebut tanpa instruksi.

## Petunjuk Wali saat penutupan belum dimulai — 2026-10-21

- Halaman Proses Tutup Buku Tabungan kini menampilkan petunjuk untuk Wali ketika tahun yang dipilih masih `OPEN`: minta Admin membuka Tahun Ajaran, memilih tahun terkait, dan klik **Mulai Penutupan**; setelah status `CLOSING`, Wali kembali dan tautan settlement tersedia per rekening.
- Petunjuk ini tidak mengubah hak akses atau alur settlement. Tes rendering template memverifikasi pesan untuk Wali.
- Verifikasi lulus: `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/app -count=1` dan `git diff --check`.

## Settlement tabungan siswa pindah sekolah melalui web — 2026-10-21

- Owner menyetujui amendment alur: Admin mencatat siswa `LEFT`; Wali menangani rekening melalui menu Tabungan, memeriksa buku transaksi dan pratinjau saldo, lalu mengonfirmasi settlement. Settlement positif memerlukan konfirmasi penyerahan; saldo nol tidak memerlukan langkah penyerahan. Admin memantau hasilnya dari Penutupan Tahun. Detail keputusan ada di [amendment](contracts/student-leaving-settlement-ui-amendment.md).
- Implementasi menambahkan pratinjau khusus settlement pindah saat tahun `OPEN` atau `CLOSING`, konfirmasi saldo yang diperiksa, deteksi preview kedaluwarsa, tautan “Perlu penyelesaian” pada direktori/detail Tabungan Wali, dan pemberitahuan setelah Admin mencatat siswa pindah. Settlement dan penyerahan ditolak pada tahun `CLOSED`; tidak ada migration.
- Tes HTTP/template dan integrasi mencakup pratinjau/konfirmasi saat tahun `OPEN`, handover, serta saldo berubah antara preview dan konfirmasi. Verifikasi yang dijalankan: `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/http ./internal/app -count=1`, `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration`, `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...`, `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...`, dan `git diff --check`; semuanya lulus. `make test-integration` memverifikasi database disposable `tabungan_siswa_fase168_clean_test`. Browser E2E tidak dijalankan.
- Working tree juga memuat perubahan FASE 25, format uang/input, dan petunjuk penutupan dari pekerjaan sebelumnya. Semua perubahan tersebut dipertahankan. FASE 25 belum dikunci; tidak ada commit/push pada sesi ini.
- Penyelarasan akses yang disetujui owner setelah audit: backend kini membatasi setoran/penarikan ke Wali dan mengunci/memeriksa status siswa `ACTIVE`; Wali masih dapat mengoreksi transaksi lama sesuai FASE 22 sebelum settlement. Jalur settlement `STUDENT_LEAVING` khusus untuk `LEFT`; `GRADUATED` diarahkan ke settlement penutupan tahun reguler. Admin tetap baca-saja.
- Antarmuka menyembunyikan form keuangan Admin, menghentikan form setoran/penarikan untuk siswa nonaktif, memberi petunjuk settlement khusus `LEFT`/`GRADUATED`, dan tidak menautkan settlement pindah untuk tahun `CLOSED`. HTTP, service, integrasi PostgreSQL, dan template mencakup penolakan Admin, siswa `LEFT`/`GRADUATED`, tahun ditutup, serta alur settlement `LEFT`.
- Verifikasi setelah penyelarasan: `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration`, `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/http ./internal/app -count=1`, `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...`, `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...`, dan `git diff --check` lulus. Browser E2E tidak dijalankan; database development/produksi tidak disentuh.

## Pelokalan antarmuka ke Bahasa Indonesia — 2026-10-24

- Istilah yang dilihat pengguna diseragamkan: **Dashboard** menjadi **Ringkasan**, **Username** menjadi **Nama pengguna**, **Password** menjadi **Kata sandi**, **Transfer Kelas** menjadi **Pindah Kelas**, dan **Settlement** menjadi **Penyelesaian Tabungan**. Nilai status seperti `OPEN`, `CLOSING`, `PENDING`, `APPROVED`, `SETTLED`, dan `COMPLETED` ditampilkan dengan padanan Indonesia; nilai internal, route, dan kontrak data tidak berubah.
- Label rekap penutupan membedakan rekening yang selesai diselesaikan dari penyerahan uang yang selesai. Pesan HTTP umum dan token/akses kini memakai pesan Indonesia.
- Pemeriksaan teks template untuk istilah Inggris umum tidak menemukan istilah tersebut sebagai teks yang terlihat; kemunculan tersisa berada pada nama route, atribut form, dan kode internal.
- Verifikasi: `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/http ./internal/app -count=1`, `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...`, `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...`, `npm run test:instant-search` (7/7), `npm run test:rupiah-input` (3/3), `PLAYWRIGHT_BROWSERS_PATH=/tmp/tabungan-siswa-playwright-browsers npm run test:e2e` (3/3), dan `git diff --check` lulus. E2E membuat dan membersihkan fixture di database disposable; database development dan produksi tidak disentuh. Tidak ada migration.
- Perubahan berada di working tree bersama pekerjaan sebelumnya yang belum di-commit; tidak ada commit atau push.

## Identitas visual dan favicon — 2026-10-24

- Ikon merek SVG baru menggambarkan buku tabungan terbuka dengan koin bertanda Rp. Ikon dipakai pada navigasi samping dan di bagian atas halaman masuk; favicon SVG dipasang pada seluruh 16 template halaman.
- Tes statis memeriksa bahwa kedua SVG dilayani sebagai `image/svg+xml`. Tes browser memeriksa logo dan favicon pada halaman masuk.
- Verifikasi: `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/http ./internal/app -count=1`, `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...`, `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...`, E2E Playwright (3/3), dan `git diff --check` lulus. E2E membuat dan membersihkan fixture di database disposable; database development/produksi dan migration tidak disentuh.
- Port 8080 sebelumnya dipakai oleh `make dev`; sesuai izin owner terdahulu untuk menghentikan server lokal saat tes, proses tersebut dihentikan. Server E2E berhenti otomatis setelah tes. Tidak ada commit/push.

## Kartu ikon pada Ringkasan — 2026-10-24

- Kartu metrik Ringkasan mengikuti pola Card with Icon TailAdmin: kotak ikon sederhana di atas label dan nilai, grid responsif, serta dukungan tema gelap. Setiap metrik memakai ikon relevan. Tidak ditambahkan indikator naik/turun karena data perbandingan periode belum tersedia.
- Pemetaan ikon disimpan bersama view model metrik; tes unit memeriksa ikon untuk peran Admin dan Siswa. Tes E2E memastikan enam kartu metrik dan ikonnya tampil di Ringkasan Admin.
- Verifikasi: `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/http ./internal/app -count=1`, `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...`, `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...`, E2E Playwright (3/3), dan `git diff --check` lulus. E2E membersihkan fixture database disposable; database development/produksi tidak disentuh.
- Port 8080 dipakai `make dev`; proses dihentikan sesuai izin owner sebelumnya agar E2E dapat dijalankan. Server tes berhenti otomatis. Tidak ada migration atau commit/push.

## Tab Laporan pada Ringkasan — 2026-10-24

- Empat tautan Laporan tahun aktif sekarang disusun sebagai navigasi tab bergaris bawah dengan ikon (siswa, kelas, kalender, penyelesaian tabungan), mengikuti referensi Tabs with underline and icon TailAdmin. Tautan tetap menuju halaman laporan terpisah dan tetap dapat dibuka langsung.
- E2E memastikan navigasi memiliki empat tautan dan empat ikon. Verifikasi: `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/http ./internal/app -count=1`, `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...`, `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...`, E2E Playwright (3/3), dan `git diff --check` lulus. E2E membersihkan fixture database disposable; database development/produksi tidak disentuh.
- Port 8080 dipakai `make dev`; proses dihentikan sesuai izin owner sebelumnya agar E2E dapat dijalankan. Server tes berhenti otomatis. Tidak ada migration atau commit/push.

## Kartu ikon pada indeks Laporan — 2026-10-24

- Empat pilihan laporan pada `/reports` kini memakai kartu Card with Icon: ikon untuk siswa, kelas, tahun ajaran, dan penyelesaian tabungan; judul, deskripsi, dan tujuan tiap tautan dipertahankan. Referensi visual: [TailAdmin Cards](https://demo.tailadmin.com/cards).
- Tes HTTP memeriksa pemetaan ikon dan tujuan link; tes template serta E2E memastikan empat kartu dan ikon dirender. Verifikasi: `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/http ./internal/app -count=1`, `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...`, `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...`, E2E Playwright (3/3), dan `git diff --check` lulus. E2E membersihkan fixture database disposable; database development/produksi tidak disentuh.
- Port 8080 dipakai `make dev`; proses dihentikan sesuai izin owner sebelumnya agar E2E dapat berjalan. Server E2E berhenti otomatis. Tidak ada migration atau commit/push.

## Ikon di depan tombol di seluruh aplikasi — 2026-10-24

- Atas klarifikasi owner, pola tombol berikon diterapkan ke seluruh halaman, bukan hanya indeks Laporan. Partial bersama `button-leading-icon` menyediakan ikon SVG konsisten untuk aksi tambah, cari/tampilkan, simpan, batal/tolak, impor, kata sandi, salin, keluar, dan konfirmasi. Tombol teks dan tautan navigasi yang memakai gaya tombol kini menampilkan ikon di depan; tombol tutup dialog yang berupa ikon saja tetap dipertahankan. Pagination memakai ikon di depan dan label Bahasa Indonesia.
- Variasi tindakan yang sudah ada (utama, sekunder, dan berbahaya) serta perilaku/endpoint tidak berubah. Query cache stylesheet diperbarui agar browser memuat CSS yang baru.
- Tes rendering aplikasi lulus; tes Playwright memverifikasi ikon tombol Masuk dan tiga aksi pada baris pengguna. Verifikasi akhir: `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/app -count=1`, `npm run test:instant-search` (7/7), `npm run test:rupiah-input` (3/3), `npm run test:e2e` (3/3), `npm run css:build`, dan `git diff --check` lulus. Percobaan E2E pertama gagal sebelum browser diluncurkan karena Chromium belum terpasang; setelah menjalankan `npm run test:e2e:install`, E2E lulus. Fixture dibersihkan dari database disposable secara otomatis.
- Tidak ada migration, perubahan database development/produksi, commit, atau push. Working tree tetap memuat perubahan terdahulu yang belum di-commit; pertahankan semuanya.

## Menu aksi titik tiga pada daftar Pengguna — 2026-10-24

- Kolom Aksi pada daftar Pengguna kini memakai pemicu titik tiga horizontal; menu menampung penyuntingan nama pengguna, aktivasi/nonaktivasi, dan reset kata sandi. Menu memakai elemen `<details>/<summary>` agar dapat dibuka dengan keyboard; klik di luar, Escape, dan memilih aksi menutupnya. Posisi panel dihitung terhadap viewport agar tidak terpotong oleh pembungkus tabel. Aksi, CSRF, konfirmasi, serta pencabutan sesi tetap menggunakan endpoint yang sudah ada.
- CSS menu dan mode gelap ditambahkan ke stylesheet bersama, cache CSS/JS diperbarui, dan E2E memeriksa pembukaan menu, tiga pilihan, Escape, serta keberhasilan alur ubah/nonaktifkan/aktifkan/reset akun.
- Verifikasi akhir: `npm run css:build`, `node --check static/js/app.js`, `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/app -count=1`, `npm run test:e2e` (3/3), dan `git diff --check` lulus. Percobaan E2E awal mendeteksi server `make dev` aktif di port 8080; sesuai izin owner sebelumnya, proses itu dihentikan. E2E kemudian lulus dan membersihkan fixture database disposable otomatis.
- Tidak ada perubahan aturan bisnis, migration, database development/produksi, commit, atau push. Working tree tetap memuat pekerjaan terdahulu yang belum di-commit.

## Perapian daftar Pengguna dan ikon halaman — 2026-10-24

- Aksi **Ubah nama pengguna** pada menu titik tiga kini membuka modal tersendiri; field edit tidak lagi memenuhi dropdown. Modal memiliki label aksesibel, nilai awal, token CSRF, serta tombol Batal/Simpan. Aksi aktivasi dan reset tetap berada pada menu dengan konfirmasi yang sama.
- Screenshot login dan Ringkasan menunjukkan ikon Masuk/Keluar membesar dan tata letak tab Laporan tidak mengikuti gaya terbaru. Akar masalahnya adalah beberapa halaman (layout masuk, Ringkasan, dan Laporan) memakai `app.css` tanpa versi cache, sehingga browser dapat menggabungkan HTML baru dengan stylesheet lama. Semua referensi CSS template diberi versi cache baru, dan SVG ikon tombol diberi atribut ukuran eksplisit agar tetap kecil walau utility CSS belum termuat.
- Tes template menguji pemicu modal dan isi edit; E2E menguji ukuran ikon Masuk/Keluar, pembukaan modal, perubahan nama, aktivasi/nonaktivasi, reset kata sandi, dan penutupan menu via Escape. Verifikasi: `npm run css:build`, `node --check static/js/app.js`, `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/app -count=1`, `npm run test:e2e` (3/3), dan `git diff --check` lulus. E2E membersihkan fixture database disposable. Port 8080 digunakan `make dev`; proses server dihentikan sesuai izin owner sebelumnya agar E2E memakai build terkini.
- Tidak ada perubahan aturan bisnis, migration, database development/produksi, commit, atau push. Perubahan terdahulu di working tree tetap dipertahankan.

## Perapian tampilan tabel Pengguna — 2026-10-24

- Pada screenshot terbaru, kartu filter dan tabel tampak menyatu karena tidak ada jarak vertikal; label kolom Aksi tidak sejajar dengan tombol titik tiga; baris yang longgar mendorong pagination melewati area bawah layar. Daftar kini memberi jarak antar kartu, meratakan label Aksi ke kanan, dan mengurangi padding vertikal pada baris agar tabel lebih ringkas.
- Verifikasi: `npm run css:build`, `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/app -count=1`, dan `git diff --check` lulus. E2E tidak diulang untuk perubahan tata letak ini.
- Perubahan hanya memengaruhi presentasi daftar Pengguna; aturan bisnis, database, dan endpoint tidak berubah. Tidak ada commit/push; perubahan lain di working tree dipertahankan.

## MQ-TEST-02-FIX02 — assertion pesan error terlokalisasi — 2026-10-26

- Assertion lama di `internal/http/autherror/authentication_error_test.go` diperbarui agar mengharapkan pesan umum Bahasa Indonesia yang sama dengan implementasi. Tes juga secara eksplisit memastikan detail error sintetis tidak muncul pada body; status 500 dan `Cache-Control: no-store` tetap diperiksa. Tidak ada perubahan perilaku aplikasi.
- Pemeriksaan terfokus lulus: `go test ./internal/http/autherror -count=1`, `go test ./internal/http/httperror -count=1`, `go test ./internal/http -count=1`, `go build ./...`, `go vet ./...`, dan `git diff --check`.
- Integration test dijalankan dengan `make test-integration` dari snapshot sementara pada PostgreSQL 18.6 disposable yang dibuat tanpa container, menggunakan direktori data privat dan port loopback 38841. Database `tabungan_siswa_fase168_clean_test`, role khusus tes, dan skema versi 13 (`dirty=false`) diverifikasi sebelum tes. Semua 11 package bertes lulus; 3 package command tidak memiliki tes. Tes mencakup package service dan repository PostgreSQL. E2E dan race detector tidak dijalankan.
- Cluster disposable telah dihentikan dan port tes tidak lagi listen; layanan systemd PostgreSQL pada port 5432 tetap aktif dan tidak disentuh. Direktori kerja/cluster sementara `/tmp/mq-test-02-fix01.mCv7Yh` masih tertinggal karena automatic review menolak penghapusan direktori. Isinya berada di direktori privat milik pengguna dan belum dihapus; pemilik perlu menanganinya secara manual. Tidak ada kredensial yang dicatat di sini.
- Working tree berisi hanya perubahan assertion tes dan catatan handover ini. Belum ada commit/push pada saat catatan ditulis.

## MQ-ROADMAP-01 — roadmap maintainability — 2026-10-26

- Dibuat [roadmap maintainability](reviews/MAINTAINABILITY_ROADMAP.md) berdasarkan audit MQ-AUDIT-00, kode pada baseline `2475fe524c817a713657059d84454d9b4e9f0112`, serta hasil MQ-TEST-02-FIX02. Roadmap memuat MQ-01 sampai MQ-04 dengan tujuan, file sasaran, risiko, prasyarat, tes, dan kriteria PASS; empat MQ belum dimulai.
- Backlog dicatat terpisah sebagai BUG-01 (`LateRequestID` tidak dibaca kembali), BUG-02 (constraint integer migration 000013), SEC-01 (evaluasi timing autentikasi), dan TEST-BACKLOG (E2E, race detector, cleanup resource FIX01). Backlog tidak otomatis menghalangi maintenance berisiko rendah.
- Pemeriksaan kode hanya-baca mengonfirmasi target gofmt, repository transaksi terlambat yang padat, field relasi yang tidak dipilih/di-scan, constraint migration, jalur autentikasi, dan tanggung jawab service settlement. Tidak ada kode atau migration yang diubah, tes tidak dijalankan, dan database tidak diakses.
- Hasil MQ-TEST-02-FIX02 tetap menjadi bukti tes terbaru: tes terfokus, build, vet, dan integration test pada PostgreSQL disposable lulus; E2E dan race detector belum diverifikasi pada baseline ini. Cluster FIX01 telah dihentikan, tetapi direktori sementara masih tertinggal; lihat catatan FIX02 di atas.
- Tidak ada perubahan selain dokumen roadmap dan catatan handover. MQ-01 belum dimulai; tunggu review supervisor.

## MQ-01 — format tes service autentikasi — 2026-10-26

- `gofmt -w` memformat hanya `internal/service/authentication_service_unit_test.go`. Diff satu baris menyelaraskan spasi field `name`; `git diff -w` kosong, `gofmt -l` tidak lagi mencantumkan target, dan tidak ada perubahan assertion, fixture, atau perilaku.
- Lulus: `go build ./...`, `go vet ./...`, `go test ./internal/domain ./internal/security ./internal/http/... -count=1`, dan `git diff --check`.
- Integration test PostgreSQL tidak diulang atas pengecualian eksplisit supervisor: perubahan terverifikasi whitespace-only dan `make test-integration` pada PostgreSQL disposable lulus di MQ-TEST-02-FIX02 sebelumnya. `internal/service` tidak dites langsung tanpa database. Tidak ada perubahan perilaku aplikasi.
- Perubahan MQ-01 terbatas pada file tes dan catatan ini; menunggu review supervisor sebelum MQ-02.

## MQ-02 — keterbacaan repository permintaan transaksi terlambat — 2026-10-26

- Dirapikan hanya `internal/repository/postgres/leaving_transaction_request_repository.go`: deklarasi, daftar kolom, query, argumen pemanggilan DB, dan pemetaan `Scan` kini ditata vertikal agar mudah ditinjau. Signature, kondisi SQL, placeholder dan urutan argumen, kolom/mapping, lock `FOR UPDATE`, serta cabang dan kontrak error dipertahankan.
- Bukti kesetaraan: perbandingan token pada seluruh 14 fragmen SQL mentah sebelum/sesudah identik, termasuk literal dan urutan token. Peninjauan diff mengonfirmasi urutan 15 argumen `Create`, 6 argumen `Update`, argumen query/filter, urutan 15 tujuan `Scan`, dan penanganan `sql.ErrNoRows` tidak berubah. Tidak ada migration, interface, service, atau aturan bisnis yang diubah.
- Pemeriksaan lulus: `gofmt -l internal/repository/postgres/leaving_transaction_request_repository.go` kosong, `go build ./...`, `go vet ./...`, `git diff --check`, dan `make test-integration` (semua paket; paket repository PostgreSQL dan service lulus). Integration test dijalankan dari snapshot sementara HEAD `3e57c6c3` dengan file target yang dirapikan, bukan dari working tree utama.
- PostgreSQL pengujian memakai cluster sementara `/tmp/mq-test-02-fix01.mCv7Yh/pgdata`, port loopback `38841`, database `tabungan_siswa_fase168_clean_test`, role khusus `tabungan_siswa_mqtest02`, skema versi 13. `data_directory`, port, database, dan role diverifikasi. Cluster dihentikan setelah tes; port sudah tidak mendengarkan. Direktori sementara dibiarkan utuh. Sesudah tes, tabel siswa, rekening, transaksi, penyelesaian, dan permintaan tidak memiliki baris; tersisa 2 baris pengguna dan 1 sesi, sehingga asal baris tersebut dari keadaan awal atau fixture pengujian tidak dapat dipastikan.
- Risiko yang tersisa: verifikasi adalah integration test pada skema disposable; tidak mengubah atau menguji ulang aturan bisnis. BUG-01, BUG-02, SEC-01, E2E, dan race detector tetap backlog terpisah. Perubahan MQ-02 menunggu review supervisor.

## MQ-03 — review error handling repository transaksi terlambat — 2026-10-26

- Audit read-only atas repository permintaan transaksi terlambat, translator error settlement, interface repository, pemanggil service, handler HTTP, dan tes terkait dicatat di [review MQ-03](reviews/MQ-03_ERROR_HANDLING_REVIEW.md).
- Tidak ditemukan perubahan implementasi yang wajib untuk menjaga kontrak saat ini. Usulan default adalah mempertahankan error semantics; peningkatan konteks `%w` hanya opsional dan memerlukan persetujuan karena mengubah teks error. Matriks mencatat celah tes untuk konflik unique konkuren, not-found/update nol baris khusus repository ini, dan jalur scan/rows.
- Tidak ada source code, migration, database, atau test yang diubah/dijalankan. `git diff --check` dan pemeriksaan tautan/referensi dokumen dijalankan sebelum commit. Implementasi lanjutan menunggu keputusan supervisor.
