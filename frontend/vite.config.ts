import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import wails from "@wailsio/runtime/plugins/vite";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import os from "node:os";
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

function characterCastPackDir() {
  const home = process.env.YOYO_HOME?.trim()
    ? path.resolve(process.env.YOYO_HOME.trim())
    : path.join(os.homedir(), ".yoyo");
  return path.join(home, "video", "packs", "character-cast");
}

/** Dev server only. Production stills are served by the Go asset middleware. */
function characterCastDevPack() {
  return {
    name: "yoyo-character-cast",
    configureServer(server: {
      middlewares: {
        use: (
          fn: (
            req: { url?: string; method?: string },
            res: {
              statusCode: number;
              setHeader: (name: string, value: string) => void;
              end: (body?: string | Uint8Array) => void;
            },
            next: () => void,
          ) => void,
        ) => void;
      };
    }) {
      const root = path.resolve(characterCastPackDir());
      server.middlewares.use((req, res, next) => {
        const method = req.method || "GET";
        if (method !== "GET" && method !== "HEAD") {
          next();
          return;
        }
        let urlPath = "";
        try {
          urlPath = decodeURIComponent((req.url || "").split("?")[0] || "");
        } catch {
          res.statusCode = 400;
          res.end();
          return;
        }
        const prefix = "/character-cast/";
        if (!urlPath.startsWith(prefix)) {
          next();
          return;
        }
        const name = urlPath.slice(prefix.length);
        if (!/^[a-z0-9]+(?:-[a-z0-9]+)*\.jpg$/.test(name)) {
          res.statusCode = 404;
          res.end();
          return;
        }
        const file = path.resolve(root, name);
        const rel = path.relative(root, file);
        if (rel.startsWith("..") || path.isAbsolute(rel) || !existsSync(file)) {
          next();
          return;
        }
        const body = readFileSync(file);
        res.setHeader("Content-Type", "image/jpeg");
        res.setHeader("Cache-Control", "public, max-age=3600");
        res.setHeader("X-Content-Type-Options", "nosniff");
        if (method === "HEAD") {
          res.end();
          return;
        }
        res.end(body);
      });
    },
  };
}

function standHomeDir() {
  const home = process.env.YOYO_HOME?.trim()
    ? path.resolve(process.env.YOYO_HOME.trim())
    : path.join(os.homedir(), ".yoyo");
  return path.join(home, "idols", "stands");
}

/** Dev server only. The desktop build serves the same cache from Go. */
function standDevPack() {
  const cdn = "https://cf-static.shinycolors.moe/spine/idols/";
  const files: Record<string, string> = {
    "data.json": "application/json; charset=utf-8",
    "data.atlas": "text/plain; charset=utf-8",
    "data.png": "image/png",
  };
  return {
    name: "yoyo-stand",
    configureServer(server: {
      middlewares: {
        use: (
          fn: (
            req: { url?: string; method?: string },
            res: {
              statusCode: number;
              setHeader: (name: string, value: string) => void;
              end: (body?: string | Uint8Array) => void;
            },
            next: () => void,
          ) => void,
        ) => void;
      };
    }) {
      server.middlewares.use((req, res, next) => {
        const method = req.method || "GET";
        if (method !== "GET" && method !== "HEAD") {
          next();
          return;
        }
        let urlPath = "";
        try {
          urlPath = decodeURIComponent((req.url || "").split("?")[0] || "");
        } catch {
          next();
          return;
        }
        const match = urlPath.match(/^\/idol-stand\/(stand(?:_costume)?)\/(\d+)\/(data\.(?:json|atlas|png))$/);
        if (!match) {
          next();
          return;
        }
        const outfit = `${match[1]}/${match[2]}`;
        const name = match[3];
        const root = path.resolve(standHomeDir());
        const file = path.resolve(root, outfit, name);
        const rel = path.relative(root, file);
        if (rel.startsWith("..") || path.isAbsolute(rel)) {
          res.statusCode = 404;
          res.end();
          return;
        }
        void (async () => {
          try {
            let body = existsSync(file) ? readFileSync(file) : null;
            if (!body || !standBytesOk(name, body)) {
              const fetched = await fetch(`${cdn}${outfit}/${name}`);
              if (!fetched.ok) {
                res.statusCode = 404;
                res.end();
                return;
              }
              body = Buffer.from(await fetched.arrayBuffer());
              if (!standBytesOk(name, body)) {
                res.statusCode = 404;
                res.end();
                return;
              }
              if (name === "data.atlas") body = Buffer.from(pointAtlas(body.toString("utf8")));
              mkdirSync(path.dirname(file), { recursive: true });
              writeFileSync(file, body);
            } else if (name === "data.atlas") {
              body = Buffer.from(pointAtlas(body.toString("utf8")));
            }
            const webp = body.length >= 12 && body.toString("ascii", 0, 4) === "RIFF" && body.toString("ascii", 8, 12) === "WEBP";
            res.setHeader("Content-Type", name === "data.png" && webp ? "image/webp" : files[name]);
            res.setHeader("Cache-Control", "public, max-age=31536000, immutable");
            if (method === "HEAD") {
              res.end();
              return;
            }
            res.end(body);
          } catch {
            if (!res.statusCode || res.statusCode === 200) res.statusCode = 404;
            res.end();
          }
        })();
      });
    },
  };
}

function standBytesOk(name: string, body: Buffer): boolean {
  const text = body.toString("utf8", 0, Math.min(body.length, 32)).trimStart();
  if (name === "data.json") return text.startsWith("{");
  if (name === "data.png") {
    const png = body.length >= 8 && body[0] === 0x89 && body.toString("ascii", 1, 4) === "PNG";
    const webp = body.length >= 12 && body.toString("ascii", 0, 4) === "RIFF" && body.toString("ascii", 8, 12) === "WEBP";
    return png || webp;
  }
  return text.length > 0 && !text.startsWith("<");
}

function pointAtlas(text: string): string {
  const lines = text.split("\n");
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].trim() !== "date.png") continue;
    const next = lines.slice(i + 1).map((line) => line.trim()).find((line) => line);
    if (next?.startsWith("size:")) lines[i] = lines[i].replace("date.png", "data.png");
  }
  return lines.join("\n");
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
  plugins: [standDevPack(), characterCastDevPack(), streamdownHighlight(), tailwindcss(), react(), keepGoEmbed(), ...(wailsReady ? [wails("./bindings")] : [])],
});
