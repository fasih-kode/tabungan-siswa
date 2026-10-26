const { test, expect } = require("@playwright/test");

const testPassword = "E2E-Only-Fase23-2026";

async function login(page, username) {
  await loginWithPassword(page, username, testPassword);
}

async function loginWithPassword(page, username, password) {
  await page.goto("/login");
  await expect(page.locator('a[href="/login"] img[src="/static/img/tabungan-siswa.svg"]')).toBeVisible();
  await expect(page.locator('link[rel="icon"][href="/static/img/favicon.svg"]')).toHaveCount(1);
  await expect(page.locator('link[rel="stylesheet"][href*="app.css"]')).toHaveAttribute("href", /user-management-polish/);
  await page.getByLabel("Nama pengguna").fill(username);
  await page.getByLabel("Kata sandi").fill(password);
  await expect(page.getByRole("button", { name: "Masuk" }).locator("svg")).toHaveAttribute("width", "16");
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page.getByText(username, { exact: true })).toBeVisible();
}

test("input uang menampilkan pemisah ribuan dan mengirim angka utuh", async ({ page }) => {
  await page.setContent('<form><label for="amount">Jumlah</label><input id="amount" name="amount" type="number" min="1" max="9999999999999" data-rupiah-input required></form>');
  await page.addScriptTag({ path: "static/js/rupiah-input.js" });

  await page.getByLabel("Jumlah").fill("20000");
  await expect(page.getByLabel("Jumlah")).toHaveValue("20.000");
  await expect(page.locator('input[type="hidden"][name="amount"]')).toHaveValue("20000");
});

test("Wali menyelesaikan settlement dan Admin menutup tahun", async ({ page }) => {
  await login(page, "e2e-fase23-wali");

  await page.goto("/leaving-transaction-requests");
  await expect(page.getByRole("heading", { name: "Permintaan Transaksi Terlewat" })).toBeVisible();
  await expect(page.getByText("Tidak ada permintaan dengan status ini.")).toBeVisible();

  await page.goto("/protected");
  await expect(page.getByText("Tahun ajaran: E2E Tahun 2025/2026")).toBeVisible();
  await expect(page.getByText("Saldo").first()).toBeVisible();
  await expect(page.locator(".dashboard-metric-card")).toHaveCount(6);
  await expect(page.locator(".dashboard-metric-icon svg")).toHaveCount(6);
  const reportTabs = page.getByRole("navigation", { name: "Laporan tahun aktif" });
  await expect(reportTabs.getByRole("link")).toHaveCount(4);
  await expect(reportTabs.locator("svg")).toHaveCount(4);
  await page.goto("/reports/students");
  await expect(page.getByText("E2E Tahun 2025/2026")).toBeVisible();
  await expect(page.getByText("E2E Siswa")).toBeVisible();
  await page.goto("/reports/classes");
  await expect(page.getByText("E2E Kelas A")).toBeVisible();
  await page.goto("/reports/academic-year");
  await expect(page.getByRole("heading", { name: "Laporan Tahun Ajaran" })).toBeVisible();
  await page.goto("/reports/settlements");
  await expect(page.getByText("E2E Siswa")).toBeVisible();

  await page.goto("/savings");
  await page.getByRole("link", { name: "Proses Tutup Buku Tabungan" }).click();
  await expect(page.getByText("Belum diselesaikan 1")).toBeVisible();
  await expect(page.getByText("Rp. 2.500")).toBeVisible();

  await page.getByRole("link", { name: "Proses penyelesaian" }).click();
  await expect(page.getByRole("heading", { name: "Konfirmasi Penyelesaian Tabungan" })).toBeVisible();
  await expect(page.getByText("Rp. 2.500")).toBeVisible();
  await page.getByRole("button", { name: "Konfirmasi Penyelesaian Tabungan" }).click();
  await expect(page.getByText("Penyelesaian tabungan berhasil dicatat.")).toBeVisible();

  await page.getByRole("button", { name: "Konfirmasi Pencocokan Jumlah Uang" }).click();
  await expect(page.getByText("Rp. 2.500 · Terkonfirmasi")).toBeVisible();
  await expect(page.getByText("Penyerahan uang berhasil dikonfirmasi.")).toBeVisible();

  await page.goto("/savings?view=settlement");
  await expect(page.getByText("Belum diselesaikan 0")).toBeVisible();
  await expect(page.getByText("Menunggu penyerahan 0")).toBeVisible();
  await expect(page.getByText("Rekening selesai 1")).toBeVisible();
  await expect(page.getByText("Cakupan Anda siap")).toBeVisible();

  await expect(page.getByRole("button", { name: "Keluar" }).locator("svg")).toHaveAttribute("width", "16");
  await page.getByRole("button", { name: "Keluar" }).click();
  await login(page, "e2e-fase23-admin");
  await page.goto("/leaving-transaction-requests");
  await expect(page.getByRole("heading", { name: "Permintaan Transaksi Terlewat" })).toBeVisible();
  await page.goto("/reports");
  await expect(page.getByRole("heading", { name: "Laporan", exact: true })).toBeVisible();
  await expect(page.locator(".report-link-card")).toHaveCount(4);
  await expect(page.locator(".report-link-icon svg")).toHaveCount(4);
  await expect(page.locator('a[href="/reports/students"].form-button-primary')).toHaveText("Buka laporan");
  await expect(page.locator('a[href="/reports/classes"].form-button-primary')).toHaveText("Buka laporan");
  await expect(page.locator(".report-link-card .form-button-primary svg")).toHaveCount(4);

  await page.goto("/savings?view=settlement");
  await expect(page.getByRole("heading", { name: "Penutupan Tahun" })).toBeVisible();
  await page.locator("#settlement-year").selectOption({ label: "E2E Tahun 2025/2026" });
  await page.getByRole("button", { name: "Tampilkan" }).click();
  await expect(page.getByText("e2e-fase23-wali").first()).toBeVisible();

  await page.goto("/savings");
  await expect(page.getByRole("heading", { name: "Histori Tabungan" })).toBeVisible();
  await page.locator("#savings-year").selectOption({ label: "E2E Tahun 2025/2026" });
  await page.locator("#savings-class").selectOption({ label: "E2E Kelas A" });
  await page.getByLabel("Hanya siswa yang memiliki rekening").check();
  await expect(page.getByText("E2E Siswa")).toBeVisible();
  await expect(page.getByText("Belum dibuat")).toHaveCount(0);
  await page.goto("/academic-years");

  const yearRow = page.getByRole("row").filter({ hasText: "E2E Tahun 2025/2026" });
  await yearRow.getByRole("link", { name: "Lihat" }).click();
  await page.getByRole("button", { name: "Tutup Tahun Ajaran" }).click();
  await expect(page.getByText("Ditutup", { exact: true })).toBeVisible();
});

test("Admin mengubah, menonaktifkan, mengaktifkan, dan reset akun Wali", async ({ page }) => {
  await login(page, "e2e-fase23-admin");
  await page.goto("/users?role=WALI_KELAS");

  const originalUsername = "e2e-fase23-wali";
  const renamedUsername = "e2e-fase24-wali-edit";
  let row = page.getByRole("row").filter({ hasText: originalUsername });
  await expect(row).toContainText("Aktif");
  const actions = row.locator(`[data-row-action-toggle][aria-label="Aksi untuk ${originalUsername}"]`);
  await expect(actions).toBeVisible();
  await actions.click();
  await expect(row.locator(".row-action-panel")).toBeVisible();
  await expect(row.locator(".row-action-panel button")).toHaveCount(3);
  await page.keyboard.press("Escape");
  await expect(row.locator(".row-action-panel")).toBeHidden();
  await expect(actions).toBeFocused();
  await actions.click();
  await row.getByRole("button", { name: "Ubah nama pengguna" }).click();
  const editDialog = page.getByRole("dialog", { name: "Ubah Nama Pengguna" });
  await expect(editDialog).toBeVisible();
  await editDialog.getByLabel("Nama pengguna baru").fill(renamedUsername);
  await editDialog.getByRole("button", { name: "Simpan Perubahan" }).click();
  row = page.getByRole("row").filter({ hasText: renamedUsername });
  await expect(row).toContainText("Aktif");

  await row.locator(`[data-row-action-toggle][aria-label="Aksi untuk ${renamedUsername}"]`).click();
  await page.once("dialog", (dialog) => dialog.accept());
  await row.getByRole("button", { name: "Nonaktifkan akun" }).click();
  row = page.getByRole("row").filter({ hasText: renamedUsername });
  await expect(row).toContainText("Nonaktif");

  await row.locator(`[data-row-action-toggle][aria-label="Aksi untuk ${renamedUsername}"]`).click();
  await page.once("dialog", (dialog) => dialog.accept());
  await row.getByRole("button", { name: "Aktifkan akun" }).click();
  row = page.getByRole("row").filter({ hasText: renamedUsername });
  await expect(row).toContainText("Aktif");

  await row.locator(`[data-row-action-toggle][aria-label="Aksi untuk ${renamedUsername}"]`).click();
  await row.getByRole("button", { name: "Ubah nama pengguna" }).click();
  await editDialog.getByLabel("Nama pengguna baru").fill(originalUsername);
  await editDialog.getByRole("button", { name: "Simpan Perubahan" }).click();
  row = page.getByRole("row").filter({ hasText: originalUsername });
  await row.locator(`[data-row-action-toggle][aria-label="Aksi untuk ${originalUsername}"]`).click();
  await page.once("dialog", (dialog) => dialog.accept());
  await row.getByRole("button", { name: "Atur ulang kata sandi" }).click();
  const temporaryPassword = await page.locator("[data-credential-reveal] code").textContent();
  expect(temporaryPassword).toBeTruthy();

  await page.getByRole("button", { name: "Keluar" }).click();
  await loginWithPassword(page, originalUsername, temporaryPassword.trim());
  await expect(page.getByText("Saat masuk pertama kali, gunakan kata sandi sementara. Buat kata sandi baru sebelum membuka menu lainnya.")).toBeVisible();
  const newPassword = "E2E-Fase24-Recovered-2026";
  await page.getByLabel("Kata sandi saat ini").fill(temporaryPassword.trim());
  await page.getByLabel("Kata sandi baru", { exact: true }).fill(newPassword);
  await page.getByLabel("Konfirmasi kata sandi baru").fill(newPassword);
  await page.getByRole("button", { name: "Simpan Kata Sandi Baru" }).click();
  await expect(page.getByText("Kata sandi berhasil diperbarui.")).toBeVisible();
});
