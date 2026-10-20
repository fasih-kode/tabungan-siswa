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

- Branch/HEAD saat pemeriksaan: `main` / `55492cb` (`docs(fase-23): lock year-end settlement contract`). Tidak ada commit baru; working tree tetap memuat perubahan lokal Project Memory dan implementasi FASE 23.
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
- Review teknis saat ini tidak menemukan blocker terhadap implementasi berdasarkan bukti tes tersebut. FASE 23 belum ditandai terkunci karena implementation lock record menurut kontrak dibuat setelah implementation commit dan owner review; tidak ada commit dalam sesi ini.

## Verifikasi akhir sebelum implementation lock — 2026-10-20

- Atas instruksi owner, tes acceptance dijalankan ulang sebelum penutupan FASE 23.
- `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration` lulus.
- `GOCACHE=/tmp/tabungan-siswa-gocache GOFLAGS=-race make test-integration` lulus.
- `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...` dan `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...` lulus.
- `npm run test:instant-search` lulus (7 tes); `git diff --check` lulus.
- Migration 10 dan 11 sebelumnya telah diuji upgrade/rollback pada database disposable; database akhir versi 11 bersih. Database development/produksi tidak disentuh.
- Bukti acceptance yang dijalankan lulus. Browser E2E tidak dijalankan. Commit implementasi dan lock record menyusul sebagai langkah penutupan resmi.
