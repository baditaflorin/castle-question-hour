import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const basePath = process.env.BASE_PATH || "/castle-question-hour/";

export default defineConfig({
  base: basePath,
  plugins: [react()],
  build: {
    outDir: "../docs",
    emptyOutDir: true,
    sourcemap: false,
    target: "es2022",
    rollupOptions: {
      output: {
        manualChunks: {
          yjs: ["yjs", "y-webrtc", "y-indexeddb"],
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
  },
});
