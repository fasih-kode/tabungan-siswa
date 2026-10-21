const { test, expect } = require("@playwright/test");

const testPassword = "E2E-Only-Fase23-2026";

async function login(page, username) {
  await loginWithPassword(page, username, testPassword);
}

async function loginWithPassword(page, username, password) {
  await page.goto("/login");
  await page.getByLabel("Username").fill(username);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page.getByText(username, { exact: true })).toBeVisible();
}

test("Wali menyelesaikan settlement dan Admin menutup tahun", async ({ page }) => {
  await login(page, "e2e-fase23-wali");

  await page.goto("/savings");
  await page.getByRole("link", { name: "Proses Tutup Buku Tabungan" }).click();
  await expect(page.getByText("Belum settlement 1")).toBeVisible();
  await expect(page.getByText("Rp 2.500")).toBeVisible();

  await page.getByRole("link", { name: "Proses settlement" }).click();
  await expect(page.getByRole("heading", { name: "Konfirmasi Settlement" })).toBeVisible();
  await expect(page.getByText("Rp 2.500")).toBeVisible();
  await page.getByRole("button", { name: "Konfirmasi Settlement" }).click();
  await expect(page.getByText("Settlement berhasil dicatat.")).toBeVisible();

  await page.getByRole("button", { name: "Konfirmasi Rekonsiliasi Uang" }).click();
  await expect(page.getByText("Rp 2.500 · Terkonfirmasi")).toBeVisible();
  await expect(page.getByText("Penyerahan uang berhasil dikonfirmasi.")).toBeVisible();

  await page.goto("/savings?view=settlement");
  await expect(page.getByText("Belum settlement 0")).toBeVisible();
  await expect(page.getByText("Menunggu penyerahan 0")).toBeVisible();
  await expect(page.getByText("Selesai 1")).toBeVisible();
  await expect(page.getByText("Cakupan Anda siap")).toBeVisible();

  await page.getByRole("button", { name: "Keluar" }).click();
  await login(page, "e2e-fase23-admin");
  await page.goto("/savings");
  await expect(page.getByRole("heading", { name: "Tabungan Siswa" })).toBeVisible();
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
  await row.locator('input[name="username"]').fill(renamedUsername);
  await row.getByRole("button", { name: "Simpan" }).click();
  row = page.getByRole("row").filter({ hasText: renamedUsername });
  await expect(row).toContainText("Aktif");

  await page.once("dialog", (dialog) => dialog.accept());
  await row.getByRole("button", { name: "Nonaktifkan" }).click();
  row = page.getByRole("row").filter({ hasText: renamedUsername });
  await expect(row).toContainText("Nonaktif");

  await page.once("dialog", (dialog) => dialog.accept());
  await row.getByRole("button", { name: "Aktifkan" }).click();
  row = page.getByRole("row").filter({ hasText: renamedUsername });
  await expect(row).toContainText("Aktif");

  await row.locator('input[name="username"]').fill(originalUsername);
  await row.getByRole("button", { name: "Simpan" }).click();
  row = page.getByRole("row").filter({ hasText: originalUsername });
  await page.once("dialog", (dialog) => dialog.accept());
  await row.getByRole("button", { name: "Reset password" }).click();
  const temporaryPassword = await page.locator("[data-credential-reveal] code").textContent();
  expect(temporaryPassword).toBeTruthy();

  await page.getByRole("button", { name: "Keluar" }).click();
  await loginWithPassword(page, originalUsername, temporaryPassword.trim());
  await expect(page.getByText("Login pertama menggunakan password sementara.")).toBeVisible();
  const newPassword = "E2E-Fase24-Recovered-2026";
  await page.getByLabel("Password saat ini").fill(temporaryPassword.trim());
  await page.getByLabel("Password baru", { exact: true }).fill(newPassword);
  await page.getByLabel("Konfirmasi password baru").fill(newPassword);
  await page.getByRole("button", { name: "Simpan Password Baru" }).click();
  await expect(page.getByText("Kata sandi berhasil diperbarui.")).toBeVisible();
});
