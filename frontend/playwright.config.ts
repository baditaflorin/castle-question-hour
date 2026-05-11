import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./test/e2e",
  timeout: 30_000,
  fullyParallel: false,
  retries: 0,
  reporter: "line",
  use: {
    baseURL: process.env.BASE_URL || "http://127.0.0.1:14173",
    headless: true,
    trace: "off",
    viewport: { width: 390, height: 844 },
  },
  projects: [
    { name: "phone-chromium", use: { ...devices["Pixel 7"] } },
  ],
});
