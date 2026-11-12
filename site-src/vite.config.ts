import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const root = dirname(fileURLToPath(import.meta.url));

/** productVersion 从 internal/version/version.go 读出产品版本。读不到文件，或文件里没有 Version，返回 0.0.0。 */
function productVersion(): string {
  try {
    const source = readFileSync(resolve(root, "../internal/version/version.go"), "utf8");
    return source.match(/var Version = "([^"]+)"/)?.[1] ?? "0.0.0";
  } catch {
    return "0.0.0";
  }
}

export default defineConfig({
  plugins: [react()],
  base: "./",
  publicDir: "public",
  // 构建时把产品版本写进页面。拉不到 GitHub 最新发布时，下载地址用这个版本拼。
  define: {
    __YOYO_VERSION__: JSON.stringify(productVersion()),
  },
  build: {
    outDir: resolve(root, "../site"),
    emptyOutDir: true,
    assetsDir: "assets",
    sourcemap: false,
  },
});
