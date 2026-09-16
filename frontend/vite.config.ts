import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import wails from "@wailsio/runtime/plugins/vite";
import { existsSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dir = path.dirname(fileURLToPath(import.meta.url));
const wailsReady = existsSync("./bindings/github.com/wailsapp/wails");

export default defineConfig({
  resolve: {
    alias: { "@": path.join(dir, "src") },
  },
  define: {
    "import.meta.env.VITE_E2E": JSON.stringify(process.env.VITE_E2E || ""),
  },
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  plugins: [tailwindcss(), react(), ...(wailsReady ? [wails("./bindings")] : [])],
});
