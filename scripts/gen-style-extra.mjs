import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const src = fs.readFileSync(path.join(root, "frontend/src/features/video/drama-style-catalog.ts"), "utf8");
const zh = fs.readFileSync(path.join(root, "frontend/src/features/video/drama-style-names-zh.ts"), "utf8");
const re = /\{ value: "([^"]+)", name: "([^"]+)", group: "[^"]+", prompt: "([^"]+)", description: "([^"]+)" \}/g;
const rows = [];
for (const m of src.matchAll(re)) rows.push({ value: m[1], name: m[2], prompt: m[3], description: m[4] });
const zhKeys = [...zh.matchAll(/^\s+([a-z0-9]+):/gm)].map((m) => m[1]);
const missingZh = rows.filter((row) => !zhKeys.includes(row.value)).map((row) => row.value);
if (rows.length < 100) {
  console.error("parsed", rows.length);
  process.exit(1);
}
if (missingZh.length) {
  console.error("missing zh", missingZh.join(","));
  process.exit(1);
}
const esc = (s) => s.replaceAll("\\", "\\\\").replaceAll('"', '\\"');
const body = rows.map((r) => `\t\t{Value: "${esc(r.value)}", Name: "${esc(r.name)}", Prompt: "${esc(r.prompt)}", Description: "${esc(r.description)}"},`).join("\n");
fs.writeFileSync(path.join(root, "internal/video/style_extra.go"), `package video\n\nfunc extraPresetStyles() []Style {\n\treturn []Style{\n${body}\n\t}\n}\n`);
console.log("wrote", rows.length);
