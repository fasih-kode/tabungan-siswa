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
