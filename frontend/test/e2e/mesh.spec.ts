import { test, expect, chromium } from "@playwright/test";

const API_BASE = process.env.API_BASE || "http://127.0.0.1:18080";
const APP_URL = process.env.APP_URL || "http://127.0.0.1:14173/castle-question-hour/";

// Two-peer mesh integration test.
//
// Boots two completely separate browser contexts (no shared IndexedDB,
// BroadcastChannel, or session storage), joins both to the same castle, has
// peer A post an answer, then asserts peer B's local "N answer(s)" counter
// reflects the new answer.
//
// Gating order matters: we first wait until both peers see each other in
// awareness ("with N other(s)" badge), then post. ICE gathering on a cold
// WebRTC stack can take 10-20s the first time STUN servers are queried, so
// gating on actual connectivity avoids the test racing the WebRTC handshake.
//
// WHY OPT-IN: headless Chromium's WebRTC ICE gathering against the default
// y-webrtc STUN servers is flaky in CI-like environments — it sometimes
// times out the peer handshake even though the signaling exchange itself
// is fine. The signaling protocol (which is the load-bearing part on the
// backend) is proven by the unit tests in
// backend/internal/signaling/signaling_test.go. This e2e exists for cases
// when you actually want to test the whole chain end-to-end, including the
// browser's real RTCPeerConnection, against a fresh build. Run with:
//
//   ENABLE_MESH_E2E=1 make smoke
//
// or directly:
//
//   cd frontend && ENABLE_MESH_E2E=1 npx playwright test test/e2e/mesh.spec.ts
test.skip(!process.env.ENABLE_MESH_E2E, "set ENABLE_MESH_E2E=1 to run (flaky WebRTC in headless)");
test.setTimeout(120_000);

test("two peers see each other's answers via the WebRTC mesh", async () => {
  const browser = await chromium.launch();
  const ctxA = await browser.newContext();
  const ctxB = await browser.newContext();
  const pageA = await ctxA.newPage();
  const pageB = await ctxB.newPage();

  try {
    const castle = `mesh-${Date.now().toString(36)}`;
    const joinURL = `${APP_URL}#castle=${castle}&api=${encodeURIComponent(API_BASE)}`;

    await Promise.all([pageA.goto(joinURL), pageB.goto(joinURL)]);

    // Both peers should auto-join via #castle= and skip JoinScreen.
    await Promise.all([
      expect(pageA.getByText(`castle:`)).toBeVisible({ timeout: 10_000 }),
      expect(pageB.getByText(`castle:`)).toBeVisible({ timeout: 10_000 }),
    ]);

    // Wait until both peers see each other via Y.js awareness. The header
    // shows "with 1 other" once the mesh is up.
    await Promise.all([
      expect(pageA.getByText(/with 1 other/i)).toBeVisible({ timeout: 45_000 }),
      expect(pageB.getByText(/with 1 other/i)).toBeVisible({ timeout: 45_000 }),
    ]);

    // Peer A posts an answer.
    await pageA.getByRole("textbox").fill("A shared whisper from peer A.");
    await pageA.getByRole("button", { name: /^share$/i }).click();
    await expect(pageA.getByText(/anonymously/i)).toBeVisible();

    // Peer B should reflect at least 1 answer via mesh sync.
    await expect(pageB.getByText(/1 answer so far/i)).toBeVisible({ timeout: 15_000 });
  } finally {
    await ctxA.close();
    await ctxB.close();
    await browser.close();
  }
});
