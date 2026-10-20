const { test, expect } = require("@playwright/test");

const testPassword = "E2E-Only-Fase23-2026";

async function login(page, username) {
  await page.goto("/login");
  await page.getByLabel("Username").fill(username);
  await page.getByLabel("Password").fill(testPassword);
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
  await page.goto("/academic-years");

  const yearRow = page.getByRole("row").filter({ hasText: "E2E Tahun 2025/2026" });
  await yearRow.getByRole("link", { name: "Lihat" }).click();
  await page.getByRole("button", { name: "Tutup Tahun Ajaran" }).click();
  await expect(page.getByText("Ditutup", { exact: true })).toBeVisible();
});
