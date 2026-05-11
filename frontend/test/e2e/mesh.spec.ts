import { test, expect, chromium } from "@playwright/test";

const API_BASE = process.env.API_BASE || "http://127.0.0.1:18080";
const APP_URL = process.env.APP_URL || "http://127.0.0.1:14173/castle-question-hour/";

// Two-peer mesh integration test.
//
// Boots two completely separate browser contexts (no shared IndexedDB,
// BroadcastChannel, or session storage), joins both to the same castle, has
// peer A post an answer, then asserts peer B's local "N answer(s)" counter
// reflects the new answer within a reasonable timeout. This is the
// load-bearing claim of the project — answers ride the WebRTC mesh — so it
// gets its own test.
test("two peers see each other's answers via the WebRTC mesh", async () => {
  const browser = await chromium.launch();
  const ctxA = await browser.newContext();
  const ctxB = await browser.newContext();
  const pageA = await ctxA.newPage();
  const pageB = await ctxB.newPage();

  try {
    const castle = `mesh-${Date.now().toString(36)}`;
    const joinURL = `${APP_URL}#castle=${castle}&api=${encodeURIComponent(API_BASE)}`;

    await pageA.goto(joinURL);
    await pageB.goto(joinURL);

    // Both peers should auto-join via the #castle= hash and skip the JoinScreen.
    // The HourScreen shows the header "castle: <code>".
    await expect(pageA.getByText(`castle:`)).toBeVisible({ timeout: 8000 });
    await expect(pageB.getByText(`castle:`)).toBeVisible({ timeout: 8000 });

    // Peer A posts an answer.
    await pageA.getByRole("textbox").fill("A shared whisper from peer A.");
    await pageA.getByRole("button", { name: /^share$/i }).click();
    await expect(pageA.getByText(/anonymously/i)).toBeVisible();

    // Peer B should reflect at least 1 answer via mesh sync.
    await expect(pageB.getByText(/1 answer so far/i)).toBeVisible({ timeout: 15000 });
  } finally {
    await ctxA.close();
    await ctxB.close();
    await browser.close();
  }
});
