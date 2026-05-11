import { test, expect } from "@playwright/test";

const API_BASE = process.env.API_BASE || "http://127.0.0.1:18080";

test("a phone joins a castle and sees the hour's question", async ({ page }) => {
  // Route the frontend at the local backend via the #api hash.
  await page.goto(`/#api=${encodeURIComponent(API_BASE)}`);

  // join screen
  await expect(page.getByRole("heading", { name: /castle of quiet questions/i })).toBeVisible();

  // enter a castle code and join
  await page.getByLabel("Castle code").fill("smoke-castle");
  await page.getByRole("button", { name: /enter the castle/i }).click();

  // expect the question text from the backend to appear
  await expect(page.getByRole("heading", { level: 1 })).not.toHaveText("…", { timeout: 5000 });

  // share an answer
  await page.getByPlaceholder(/anonymous|whatever|whisper/i).fill("A small kindness, unasked.");
  await page.getByRole("button", { name: /^share$/i }).click();
  await expect(page.getByText(/anonymously/i)).toBeVisible();
});
