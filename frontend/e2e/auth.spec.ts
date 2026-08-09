import { expect, test } from "@playwright/test";

test("user can register, sign in, view a profile, and sign out", async ({ page }, testInfo) => {
  const projectSuffix = testInfo.project.name.replace(/[^a-z]/g, "_");
  const username = `e2e_${projectSuffix}_${Date.now()}`;
  const password = "e2e-password-123";

  await page.goto("/auth/signup");
  await page.getByLabel("Username").fill(username);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page.getByLabel("Confirm password").fill(password);
  await page.getByRole("button", { name: "Create account" }).click();

  await expect(page).toHaveURL(/\/auth\/login$/);
  await expect(page.getByText("Account created. You can now sign in.")).toBeVisible();

  await page.getByLabel("Username").fill(username);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await expect(page).toHaveURL(/\/profile$/);
  await expect(page.getByRole("heading", { name: "Profile" })).toBeVisible();
  await expect(page.getByText(username, { exact: true }).first()).toBeVisible();

  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/auth\/login$/);
});
