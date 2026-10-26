/**
 * 拥有：桌面开发服务器的 Vite 配置，以及几个只在开发时插进的中间件。
 * 不拥有：生产包怎么由 Go 提供静态文件。
 */

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

/** keepGoEmbed 在打包结束时写 dist/.keep。目录里没有这个文件时，Go 的 embed 会拒绝空目录。文件已经在就不覆盖。 */
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
 * streamdownHighlight 补上 streamdown 用 React.lazy 去加载的 highlighted-body。
 * Vite 8 的依赖优化会把这个文件改写成 .vite/deps 下一个从未写出的带哈希名字，WebView2 在 wails.localhost 上动态导入就会失败。
 * 不要把 streamdown 放进 optimizeDeps.exclude。那一图的原生 ESM 在 WebView2 里挂不上，窗口会停在 Wails 的背景色 (28,29,31)。
 * 哈希文件若已经写在 node_modules/.vite/deps 里，就交给后面的中间件。
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

/** characterCastPackDir 是角色包目录。设了 YOYO_HOME 就用它，否则用用户主目录下的 .yoyo。路径是 video/packs/character-cast。 */
function characterCastPackDir() {
  const home = process.env.YOYO_HOME?.trim()
    ? path.resolve(process.env.YOYO_HOME.trim())
    : path.join(os.homedir(), ".yoyo");
  return path.join(home, "video", "packs", "character-cast");
}

/** characterCastDevPack 只给开发服务器用。生产环境的静帧由 Go 的资源中间件提供。只接受 /character-cast/ 下小写连字符的 jpg，路径跳出包目录就交给后面的中间件。 */
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

/** standHomeDir 是立绘缓存目录。设了 YOYO_HOME 就用它，否则用用户主目录下的 .yoyo/idols/stands。 */
function standHomeDir() {
  const home = process.env.YOYO_HOME?.trim()
    ? path.resolve(process.env.YOYO_HOME.trim())
    : path.join(os.homedir(), ".yoyo");
  return path.join(home, "idols", "stands");
}

/** standDevPack 只给开发服务器用。桌面安装包由 Go 提供同一份缓存。缺文件时向 cf-static.shinycolors.moe 拉取，校验通过才写入缓存。 */
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

/** standBytesOk 拒绝明显不是立绘文件的响应。data.json 必须以 { 开头。data.png 接受 PNG 或 WEBP 文件头。其余文本不能以 < 开头，避免把 HTML 错误页存进缓存。 */
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

/** pointAtlas 把图集里单独一行的 date.png 改成 data.png。后面要有一行以 size: 开头才改，避免误伤正文里的同名文字。 */
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
    // Wails 打开的是 http://localhost:9245。只绑 127.0.0.1 时，Windows 上的 ::1 没有人听，WebView2 连上后会画出空白窗口。
    host: true,
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
    cors: true,
  },
  plugins: [standDevPack(), characterCastDevPack(), streamdownHighlight(), tailwindcss(), react(), keepGoEmbed(), ...(wailsReady ? [wails("./bindings")] : [])],
});
