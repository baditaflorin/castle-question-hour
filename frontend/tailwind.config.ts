import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        night: "#1a1f2e",
        candle: "#e9d8a6",
        bone: "#f5efe0",
        slate: "#3b4356",
        moss: "#6b8e23",
        ember: "#ca6f1e",
      },
      fontFamily: {
        sans: ["ui-sans-serif", "system-ui", "-apple-system", "Segoe UI", "Inter", "sans-serif"],
        serif: ["ui-serif", "Georgia", "Cambria", "serif"],
      },
    },
  },
  plugins: [],
} satisfies Config;
