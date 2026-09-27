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

/**
 * streamdown React.lazy()s ./highlighted-body-*.js. Vite 8's dep optimizer
 * rewrites that to a hashed file under .vite/deps that is never written, so
 * WebView2 on wails.localhost fails the dynamic import.
 *
 * Do not optimizeDeps.exclude streamdown — native ESM of that graph never
 * mounts in WebView2 and leaves the Wails background (28,29,31) on screen.
 */
function streamdownHighlight() {
  const shim =
    'export { HighlightedCodeBlockBody } from "/src/lib/streamdown-highlight-body.tsx";\n';
  const isMissingDep = (url: string) =>
    /(?:^|\/)node_modules\/\.vite\/deps\/highlighted-body-[^/?]+\.js(?:$|\?)/.test(url);

  return {
    name: "yoyo-streamdown-highlight",
    configureServer(server: {
      middlewares: {
        use: (
          fn: (
            req: { url?: string },
            res: { setHeader: (k: string, v: string) => void; end: (b: string) => void },
            next: () => void,
          ) => void,
        ) => void;
      };
    }) {
      server.middlewares.use((req, res, next) => {
        const url = req.url || "";
        if (!isMissingDep(url)) {
          next();
          return;
        }
        const name = (url.split("?")[0] || "").split("/").pop() || "";
        if (name && existsSync(path.join(dir, "node_modules/.vite/deps", name))) {
          next();
          return;
        }
        res.setHeader("Content-Type", "application/javascript; charset=utf-8");
        res.setHeader("Cache-Control", "no-cache");
        res.end(shim);
      });
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
    include: ["streamdown"],
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
    cors: true,
  },
  plugins: [streamdownHighlight(), tailwindcss(), react(), keepGoEmbed(), ...(wailsReady ? [wails("./bindings")] : [])],
});
