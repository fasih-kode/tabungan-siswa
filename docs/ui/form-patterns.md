# Form UI Patterns

## Tujuan

Form UI Patterns adalah kontrak visual dan struktur dasar seluruh form
di aplikasi Tabungan Siswa.

Pattern digunakan oleh halaman fitur tanpa membawa business logic.

## Label

```html
<label class="form-label" for="field">
    Label
</label>
```

Label wajib memiliki atribut `for` yang menunjuk ke `id` field.

## Required

Gunakan atribut HTML `required`.

```html
<input
    id="name"
    name="name"
    type="text"
    required
    class="form-input"
>
```

## Input

```html
<input
    id="field"
    name="field"
    type="text"
    class="form-input"
>
```

## Input Error

Gunakan `aria-invalid`, `aria-describedby`, dan `form-input-error`.

```html
<input
    id="field"
    name="field"
    type="text"
    aria-invalid="true"
    aria-describedby="field-error"
    class="form-input form-input-error"
>
```

## Help Text

```html
<p class="form-help">
    Keterangan tambahan untuk field.
</p>
```

## Error Message

```html
<p id="field-error" class="form-error">
    Nilai tidak valid.
</p>
```

## Select

```html
<select
    id="field"
    name="field"
    class="form-select"
>
    <option value="">Pilih...</option>
</select>
```

## Textarea

```html
<textarea
    id="field"
    name="field"
    class="form-textarea"
></textarea>
```

## Form Actions

```html
<div class="form-actions">
    <button type="submit" class="form-button-primary">
        Simpan
    </button>

    <a href="/previous" class="form-button-secondary">
        Batal
    </a>
</div>
```

## State

Pattern mendukung:

- normal
- focus
- error
- disabled

State error menggunakan:

- `aria-invalid="true"`
- `aria-describedby`
- `form-input-error`
- `form-error`

State disabled menggunakan atribut HTML `disabled`.

## Prinsip

- HTML tetap menjadi sumber perilaku form.
- Tailwind menjadi sumber styling.
- Tidak membutuhkan JavaScript untuk perilaku dasar.
- HTMX hanya digunakan ketika form memang membutuhkan progressive enhancement.
- Pattern tidak mengandung business logic.
- Pattern tidak menentukan endpoint.
- Pattern tidak menentukan validasi domain.

## Pencarian dan filter instan

Pencarian kata kunci menggunakan `type="search"` dan diproses 300 ms setelah
pengguna berhenti mengetik. Filter pilihan langsung memperbarui hasil ketika
nilainya berubah. Semua hasil tetap berasal dari endpoint GET yang melakukan
validasi dan otorisasi di server.

Gunakan satu pembungkus ber-ID tetap dengan atribut `data-instant-region`.
Form GET di dalamnya memakai `data-instant-filter`. Bungkus tabel beserta
paginasinya dengan `data-instant-results`, terpisah dari form agar kotak
pencarian tetap fokus. Sediakan `data-instant-status` dengan `role="status"`
dan `aria-live="polite"` untuk pengumuman proses serta kesalahan koneksi.
Muat `/static/js/instant-search.js` pada halaman tersebut.

- `data-reset-filter="class_id"` pada filter tahun mengosongkan kelas lama
  sebelum meminta pilihan kelas untuk tahun yang baru.
- `data-instant-summary` menyediakan ringkasan jumlah baris yang ikut diperbarui.
- `data-instant-sync` pada dialog kelas/penugasan menyelaraskan pilihan modal
  yang sedang tertutup dengan tahun ajaran hasil filter.
- Tautan paginasi di dalam hasil memakai endpoint GET yang sama dan diperbarui
  tanpa reload. Tautan detail tetap berfungsi sebagai navigasi biasa.
- Permintaan lama dibatalkan dan respons kedaluwarsa diabaikan. URL menyimpan
  filter aktif; Back/Forward memulihkan kontrol dan hasil yang sesuai.
- Tombol submit hanya ditampilkan di dalam `<noscript>` sebagai fallback.
  Form penyimpanan/aksi POST tetap menggunakan tombol dan perlindungan CSRF.

Toolbar siswa menyatu dengan kartu tabel: pencarian di kiri, diikuti status,
kelas, tahun ajaran, jenis tampilan, dan ringkasan hasil. Pada layar kecil,
kontrol turun menjadi beberapa baris.

Pemeriksaan JavaScript: `npm run test:instant-search`.

## Pagination tabel

Seluruh tabel menggunakan **10 baris per halaman**. Ini menggantikan ukuran
halaman 20 dan pilihan ukuran `limit` pada kontrak UI sebelumnya. Nilai `limit`
lama yang valid tetap diterima untuk kompatibilitas tautan, tetapi tidak
mengubah jumlah baris tabel. Daftar pilihan pada modal tetap memuat seluruh
pilihan yang tersedia dan tidak ikut dibatasi ukuran tabel.

Gunakan `{{template "pagination" .Pagination}}` di bawah setiap tabel, di dalam
`data-instant-results`. Komponen menampilkan Previous/Next dengan ikon panah,
nomor halaman aktif berwarna lembut, dan elipsis di antara rentang nomor yang
berjauhan. Kontrol yang tidak tersedia dinonaktifkan; `aria-current="page"`
menandai halaman aktif.

Jumlah halaman dihitung dari total data dengan filter dan cakupan akses yang
sama dengan daftar. Service meminta total hanya saat `Options.IncludeTotal`
aktif. Repository `ListTotals` menjalankan query count tanpa memuat seluruh
baris. Penghitungan direktori siswa memakai kondisi SQL yang sama dengan
query daftar agar pencarian, tahun, kelas, status, dan belum ditempatkan
menghasilkan jumlah halaman yang konsisten.

Tautan mempertahankan filter/pencarian, menghapus ukuran halaman lama serta
notifikasi hasil POST, dan dapat dibuka langsung. Jika halaman sudah tidak
tersedia setelah data berkurang, server mengarahkan ke halaman terakhir yang
masih tersedia. Pencarian atau perubahan filter kembali ke halaman pertama.
