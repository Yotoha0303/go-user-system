import { expect, test } from "@playwright/test";

test("user can register, restore two tabs, and revoke access on sign out", async ({ context, page }, testInfo) => {
  const browserErrors: string[] = [];
  const captureConsoleError = (message: import("@playwright/test").ConsoleMessage) => {
    if (message.type() !== "error") return;
    const isExpectedAnonymousRefresh =
      message.text().includes("status of 401") &&
      message.location().url.includes("/api/v1/auth/refresh");
    if (!isExpectedAnonymousRefresh) browserErrors.push(message.text());
  };
  page.on("console", captureConsoleError);
  page.on("pageerror", (error) => browserErrors.push(error.message));
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
  const loginResponsePromise = page.waitForResponse(
    (response) =>
      response.url().endsWith("/api/v1/auth/login") &&
      response.request().method() === "POST"
  );
  await page.getByRole("button", { name: "Sign in" }).click();
  const loginResponse = await loginResponsePromise;
  const loginBody = await loginResponse.json();
  let currentAccessToken = loginBody.data.access_token as string;

  await expect(page).toHaveURL(/\/profile$/);
  await expect(page.getByRole("heading", { name: "Profile" })).toBeVisible();
  await expect(
    page
      .getByRole("region", { name: "Account details" })
      .getByText(username, { exact: true })
      .first()
  ).toBeVisible();

  const secondTab = await context.newPage();
  secondTab.on("console", captureConsoleError);
  secondTab.on("pageerror", (error) => browserErrors.push(error.message));
  const pageRefreshPromise = page.waitForResponse(
    (response) =>
      response.url().endsWith("/api/v1/auth/refresh") &&
      response.request().method() === "POST"
  );
  await Promise.all([page.reload(), secondTab.goto("/profile")]);
  const pageRefreshBody = await (await pageRefreshPromise).json();
  currentAccessToken = pageRefreshBody.data.access_token as string;
  await expect(page.getByRole("heading", { name: "Profile" })).toBeVisible();
  await expect(secondTab.getByRole("heading", { name: "Profile" })).toBeVisible();

  const openNavigationButton = page.getByRole("button", {
    name: "Open navigation",
  });
  if (await openNavigationButton.isVisible()) {
    await openNavigationButton.click();
    await expect(openNavigationButton).toHaveAttribute("aria-expanded", "true");
  }
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/auth\/login$/);
  expect(browserErrors).toEqual([]);

  const revokedAccessResponse = await page.request.get("/api/v1/users/me", {
    headers: { Authorization: `Bearer ${currentAccessToken}` },
  });
  expect(revokedAccessResponse.status()).toBe(401);
});
