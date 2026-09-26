import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import wails from "@wailsio/runtime/plugins/vite";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dir = path.dirname(fileURLToPath(import.meta.url));
const embedKeep = path.join(dir, "dist", ".keep");
const embedKeepBody = "go embed stub\n";

function keepGoEmbed() {
  return {
    name: "keep-go-embed",
    closeBundle() {
      mkdirSync(path.dirname(embedKeep), { recursive: true });
      if (!existsSync(embedKeep)) writeFileSync(embedKeep, embedKeepBody);
    },
  };
}

const wailsReady = existsSync("./bindings/github.com/wailsapp/wails");
const pkg = JSON.parse(readFileSync(path.join(dir, "package.json"), "utf8")) as { version?: string };
const appVersion = process.env.npm_package_version?.trim() || pkg.version || "0.0.0";

export default defineConfig({
  resolve: {
    alias: {
      "@": path.join(dir, "src"),
      "@yingce": path.join(dir, "src/vendor/yingce-canvas"),
    },
    dedupe: ["react", "react-dom"],
  },
  optimizeDeps: {
    exclude: ["@ffmpeg/core", "@ffmpeg/ffmpeg"],
  },
  define: {
    __APP_VERSION__: JSON.stringify(appVersion),
    __APP_CHANGELOG__: JSON.stringify(""),
    "import.meta.env.VITE_APP_VERSION": JSON.stringify(appVersion),
    "import.meta.env.VITE_E2E": JSON.stringify(process.env.VITE_E2E || ""),
    "import.meta.env.VITE_CANVAS_BACKEND_URL": JSON.stringify("/api"),
  },
  server: {
    // Wails loads http://localhost:9245. Binding only 127.0.0.1 leaves ::1
    // dead on Windows, so WebView2 connects and paints a blank window.
    host: true,
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  plugins: [tailwindcss(), react(), keepGoEmbed(), ...(wailsReady ? [wails("./bindings")] : [])],
});
