# FASE 17.2 — HTTP Feature Contract

Status: DRAFT FOR IMPLEMENTATION
Fase: 17 — HTTP Feature Foundation
Stack: Go, net/http, html/template, HTMX, Tailwind CSS, PostgreSQL

## 1. Tujuan

Menetapkan kontrak HTTP bersama yang digunakan oleh vertical slice berikutnya.
Kontrak ini harus menjaga konsistensi handler, response, error mapping,
query parameter, pagination, dan presentation.

Dokumen ini tidak mengimplementasikan fitur domain.

## 2. Prinsip Arsitektur

1. Router mendaftarkan route dan menghubungkan middleware dengan handler.
2. Handler menangani HTTP request, memanggil service, dan membentuk response.
3. Service bertanggung jawab atas use case, business logic, validasi domain,
   authorization, dan transaction boundary sesuai kontrak service.
4. Repository bertanggung jawab atas akses data dan tidak menangani presentation.
5. Template menggunakan Go html/template, bukan templ.
6. HTMX merupakan progressive enhancement. HTML penuh tetap menjadi baseline.
7. Gunakan standard library dan abstraksi yang sudah tersedia sebelum
   memperkenalkan komponen baru.
8. Perubahan harus minimal dan mempertahankan kontrak domain yang sudah ada.

## 3. Handler dan Dependency Wiring

1. Handler menerima dependency yang dibutuhkan melalui mekanisme wiring aplikasi.
2. Dependency wajib harus tersedia sebelum handler digunakan.
3. Handler tidak boleh mengimplementasikan business logic.
4. Handler tidak boleh melewati service untuk mengakses repository secara langsung.
5. Handler bertanggung jawab mengubah request menjadi input use case,
   memanggil service, dan menentukan HTTP response.
6. Pola wiring yang sudah ada dipertahankan jika memenuhi kebutuhan tersebut.
7. Jangan membuat factory, interface, atau lapisan baru tanpa kebutuhan nyata.

## 4. View Model

1. View model merupakan data presentation untuk template.
2. View model tidak menjadi tempat business logic.
3. Model domain tidak diubah hanya untuk memenuhi kebutuhan tampilan.
4. Setiap halaman boleh memiliki view model khusus sesuai kebutuhannya.
5. Data umum seperti judul halaman, pesan status, dan pagination hanya
   ditambahkan apabila diperlukan oleh halaman.
6. Template harus mempertahankan contextual escaping dari html/template.
7. Hindari satu view model besar yang mencampurkan seluruh fitur aplikasi.

## 5. HTTP Error Mapping

Source saat ini memiliki pemetaan umum pada
internal/http/httperror/http_error.go dan pemetaan khusus authentication pada
internal/http/autherror/authentication_error.go.

Kategori dan status HTTP yang menjadi acuan:

| Kategori | Status HTTP |
|---|---:|
| Kesalahan request atau validation | 400 Bad Request |
| Unauthorized | 401 Unauthorized |
| Forbidden | 403 Forbidden |
| Not found | 404 Not Found |
| Conflict | 409 Conflict |
| Internal error | 500 Internal Server Error |

Ketentuan:

1. Pertahankan pemetaan berbasis errors.Is dan identitas error yang sudah ada.
2. Status ditentukan berdasarkan semantik error dan kontrak endpoint.
3. Kesalahan parsing atau format request dibedakan dari validation pada
   service/domain, meskipun keduanya dapat menghasilkan status 400.
4. Jangan memetakan semua error domain secara otomatis sebagai Bad Request.
5. Error yang tidak dikenal menjadi internal error.
6. Response tidak boleh membocorkan detail internal, query, credential,
   atau informasi sensitif.
7. Perilaku keamanan authentication, termasuk Cache-Control: no-store yang
   sudah digunakan, harus dipertahankan sesuai konteks endpoint.
8. Jangan menggabungkan httperror dan autherror tanpa bukti bahwa perubahan
   tersebut memperbaiki konsistensi tanpa menghilangkan perilaku yang ada.
9. Setiap perubahan mapping harus disertai test untuk kategori terdampak.

## 6. POST, Service, Redirect, GET

1. Handler memeriksa metode dan format request, lalu menyiapkan input use case.
2. Service menjalankan use case, business logic, validation, dan authorization
   sesuai kontraknya.
3. Handler menerjemahkan hasil service menjadi HTTP response.
4. Untuk submission form HTML biasa yang berhasil, gunakan pola
   Post/Redirect/Get.
5. Status 303 See Other menjadi status standar untuk redirect setelah POST
   form berhasil.
6. Login berhasil mengarahkan browser ke `/protected` dengan status 303.
7. Logout berhasil mengarahkan browser ke `/login` dengan status 303 setelah
   service berhasil mencabut sesi dan cookie sesi dibersihkan.
8. Logout tanpa cookie sesi tetap idempoten: cookie dibersihkan bila perlu,
   lalu browser diarahkan ke `/login` dengan status 303.
9. Jika service logout gagal, handler mengembalikan error sesuai pemetaan
   HTTP yang berlaku dan tidak mengirim redirect sukses atau menghapus cookie.
10. Pengecualian terhadap PRG hanya diperbolehkan jika kontrak endpoint secara
    eksplisit menetapkan response lain, misalnya partial response untuk HTMX.
11. Validation error dapat menampilkan kembali form dengan input aman dan
    pesan yang sesuai.
12. Error lain mengikuti kontrak HTTP dan kebutuhan halaman.
13. Refresh halaman hasil redirect tidak boleh mengulangi operasi tulis.
14. Authorization dan validasi domain tetap dilakukan oleh service, bukan
    hanya oleh handler.

## 7. Query Parameter

1. Query parameter dibaca melalui mekanisme standar net/http.
2. Parsing dan validasi format parameter dilakukan sebelum memanggil service.
3. Parameter yang tidak valid harus ditangani secara konsisten.
4. Default parameter harus dinyatakan secara eksplisit dalam kontrak fitur.
5. Search, filter, dan sort hanya diteruskan sebagai kriteria yang didukung
   oleh service atau repository.
6. Jangan menerima nama kolom atau ekspresi SQL bebas dari request.
7. Nama parameter dan perilaku default mengikuti kebutuhan fitur dan kontrak
   service yang sudah ada.

## 8. Pagination

Source repository telah memiliki sejumlah operasi list berpaginasi. Namun,
kontrak HTTP umum belum boleh diasumsikan sudah tersedia atau seragam.

Ketentuan:

1. Kontrak HTTP pagination harus kompatibel dengan service dan repository
   yang digunakan oleh fitur terkait.
2. Page, jika digunakan, harus bernilai positif.
3. Page size harus dibatasi untuk mencegah permintaan data berlebihan.
4. Nilai default dan batas maksimum ditentukan setelah signature, perilaku,
   dan test operasi list terkait diperiksa.
5. Jangan menetapkan default global hanya berdasarkan asumsi bahwa semua
   repository menggunakan pola pagination yang sama.
6. Handler mem-parsing parameter dan meneruskan input yang sesuai kepada service.
7. View model hanya menyediakan metadata yang benar-benar diperlukan template.
8. Semantik total count, halaman kosong, dan halaman di luar rentang harus
   mengikuti kontrak service yang digunakan.
9. Search dan filter yang aktif dipertahankan pada navigasi pagination hanya
   jika parameter tersebut memang didukung oleh endpoint.
10. Jangan mengubah kontrak repository atau menambah query hanya untuk memenuhi
    presentation apabila kontrak yang ada sudah memadai.

## 9. Search dan Filter

Belum ada bukti bahwa semua operasi list memiliki dukungan search dan filter
umum. Dukungan harus ditentukan per fitur berdasarkan kemampuan service dan
repository aktual.

Ketentuan:

1. Search dan filter bersifat opsional kecuali kontrak fitur menetapkan
   sebaliknya.
2. Handler mem-parsing dan memvalidasi format parameter request.
3. Hanya kriteria yang didukung service yang boleh diteruskan.
4. Nama kolom, ekspresi SQL, atau urutan SQL bebas tidak boleh diterima
   dari request.
5. Parameter search dan filter tidak boleh memperluas authorization scope.
6. Authorization tetap mengikuti kontrak service dan actor yang terautentikasi.
7. Nama parameter, normalisasi, dan perilaku default ditetapkan saat fitur
   terkait dibangun.
8. Jangan menambahkan search atau filter generik yang tidak didukung kebutuhan
   dan kontrak data aktual.

## 10. Success, Error, dan Validation Presentation

Login dan dashboard telah memiliki view model masing-masing. Namun, source
saat ini belum membuktikan adanya satu kontrak presentation message umum
untuk semua fitur.

Ketentuan:

1. Presentation membedakan pesan keberhasilan, error umum, dan validation
   apabila konteks halaman membutuhkannya.
2. Validation per field digunakan bila form dan use case memerlukannya.
3. Pesan presentation bukan sumber kebenaran business rule.
4. Pesan tidak boleh memuat secret atau detail internal.
5. Mekanisme pesan setelah redirect harus mengikuti fasilitas yang benar-benar
   tersedia dan sesuai kontrak keamanan aplikasi.
6. Jangan mengasumsikan flash message atau penyimpanan pesan dalam session
   sudah tersedia.
7. Jangan menambah mekanisme flash message baru sebelum kebutuhan, session,
   middleware, dan dampak keamanan diperiksa.
8. Kesalahan yang memerlukan pengisian ulang form harus mempertahankan hanya
   input yang aman dan relevan.
9. Bentuk view model ditentukan berdasarkan kebutuhan halaman, tanpa memaksa
   semua halaman menggunakan satu struktur besar.

## 11. Full-Page dan Partial Response

HTML penuh tetap menjadi baseline aplikasi. Roadmap menetapkan partial response
sebagai kontrak untuk kebutuhan HTMX mendatang, bukan implementasi seluruh
interaksi HTMX pada FASE 17.

Ketentuan:

1. Halaman utama tetap berfungsi tanpa HTMX.
2. Partial response hanya dikirim ketika endpoint dan request memiliki kontrak
   partial response yang eksplisit.
3. Partial response menggunakan template atau fragment yang sesuai.
4. Partial response tidak boleh melewati authentication, authorization,
   validation, atau business logic service.
5. Error dan validation pada partial response harus sesuai konteks tampilan.
6. Perbedaan full-page dan partial response tidak boleh mengubah hasil use case
   atau aturan authorization.
7. FASE 17 menetapkan kontrak dasar yang diperlukan. Implementasi interaksi
   HTMX secara luas tetap menjadi cakupan FASE 26.

## 12. Keamanan dan Kompatibilitas

1. Authentication dan authorization tetap mengikuti kontrak keamanan aplikasi.
2. Handler tidak boleh menganggap parameter request sebagai bukti authorization.
3. Template menggunakan html/template dan tidak menandai input pengguna
   sebagai HTML tepercaya tanpa alasan yang sah.
4. Error response tidak membocorkan informasi internal.
5. Perubahan tidak boleh menonaktifkan middleware atau pengamanan yang ada.
6. Integration test tetap harus menggunakan database test yang diizinkan.
7. Database development tidak boleh menjadi target integration test.

## 13. Strategi Pengujian dan Kriteria Penerimaan

Kontrak menjadi acuan implementasi FASE 17. Kesiapan implementasi dan status
LOCK fase harus dibuktikan melalui perubahan source dan hasil verifikasi.

Kriteria penerimaan implementasi:

1. Handler tidak mengandung business logic dan menggunakan service sesuai kontrak.
2. Dependency wajib divalidasi melalui mekanisme wiring yang berlaku.
3. HTTP error mapping mempertahankan perilaku yang ada dan konsisten untuk
   kategori error yang didukung.
4. Pola POST/Redirect/GET diuji untuk form HTML yang menggunakan pola tersebut.
5. Parsing query parameter dan pagination diuji pada endpoint yang menerapkannya.
6. Presentation mengikuti kebutuhan halaman dan tidak mengasumsikan mekanisme
   flash message yang belum tersedia.
7. Full-page HTML tetap berfungsi tanpa HTMX.
8. Partial response hanya diterapkan pada endpoint dengan kontrak eksplisit.
9. Focused test yang relevan lulus.
10. Perubahan lolos formatting, pemeriksaan diff, dan review.
11. Tidak ada perubahan kontrak domain atau implementasi fitur di luar cakupan.
12. Commit tersedia sebelum FASE 17 dinyatakan LOCK.

## 14. Batas Cakupan

Dokumen ini tidak mengimplementasikan:

- CRUD Academic Management.
- Student Management.
- Savings Account dan Passbook.
- Transaction Engine.
- Transaction Correction dan Audit.
- Year-End Settlement.
- User Administration.
- Dashboard dan Reporting.
- Seluruh interaksi HTMX.
- Security Audit menyeluruh.
- UAT atau deployment.

Fitur tersebut tetap dikerjakan pada fase masing-masing sesuai roadmap resmi.

## 15. Workflow Perubahan

Semua implementasi mengikuti urutan:

Inspect
→ Contract
→ Implement
→ Format
→ Test
→ Verify
→ Review
→ Commit
→ Lock

File baru dibuat menggunakan Heredoc-based File Generation.
Perubahan kecil pada file yang sudah ada menggunakan Python-based
Deterministic Text Replacement.

Jangan menyatakan test PASS, commit berhasil, atau fase LOCK sebelum ada
bukti yang mendukung status tersebut.
