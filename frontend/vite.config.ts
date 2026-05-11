import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const basePath = process.env.BASE_PATH || "/castle-question-hour/";

export default defineConfig({
  base: basePath,
  plugins: [react()],
  build: {
    outDir: "../docs",
    // IMPORTANT: docs/ also holds markdown (ADRs, runbook, etc.) — never empty it.
    // The Makefile build target removes only the known build artefacts before invoking us.
    emptyOutDir: false,
    sourcemap: false,
    target: "es2022",
    rollupOptions: {
      output: {
        // Function form — Vite 8 / rolldown requires this shape.
        manualChunks(id: string): string | undefined {
          if (
            id.includes("node_modules/yjs/") ||
            id.includes("node_modules/y-webrtc/") ||
            id.includes("node_modules/y-indexeddb/") ||
            id.includes("node_modules/lib0/")
          ) {
            return "yjs";
          }
          return undefined;
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api/v1/signal": {
        target: "ws://localhost:8080",
        ws: true,
      },
      "/api": "http://localhost:8080",
      "/healthz": "http://localhost:8080",
      "/readyz": "http://localhost:8080",
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    include: ["test/**/*.test.ts"],
    exclude: ["test/e2e/**", "node_modules", "../docs"],
  },
});
