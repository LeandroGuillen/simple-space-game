import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

import { defineConfig } from "vitest/config";
import solid from "vite-plugin-solid";

const version = readFileSync(
  fileURLToPath(new URL("../VERSION", import.meta.url)),
  "utf-8",
).trim();

export default defineConfig({
  define: {
    "import.meta.env.VITE_APP_VERSION": JSON.stringify(version),
  },
  plugins: [solid()],
  test: {
    environment: 'jsdom',
    globals: true,
  },
})
