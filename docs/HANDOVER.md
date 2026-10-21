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
