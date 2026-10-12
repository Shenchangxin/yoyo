import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const root = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [react()],
  base: "./",
  publicDir: "public",
  build: {
    outDir: resolve(root, "../site"),
    emptyOutDir: true,
    assetsDir: "assets",
    sourcemap: false,
  },
});
