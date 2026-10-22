// Builds catalog.json and copies avatars from the local eidolon-sc character pack.
// The Spine files themselves stay on the ShinyColors CDN; only the default outfit id is kept.
import { copyFileSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const root = join(here, "../../../../eidolon-sc/characters");
const keep = new Set(["mano", "hiori", "meguru"]);
const circles = [
  ["sc-illumination", "illumination STARS", ["mano", "hiori", "meguru"]],
];
const circleOf = new Map();
for (const [id, , members] of circles) for (const m of members) circleOf.set(m, id);

const toml = readFileSync(join(root, "characters.toml"), "utf8");
const characters = [];
let cur = null;
for (const line of toml.split(/\n/)) {
  const t = line.trim();
  if (!t || t.startsWith("#")) continue;
  if (t === "[[characters]]") {
    if (cur) characters.push(cur);
    cur = {};
    continue;
  }
  const m = t.match(/^([A-Za-z0-9_]+)\s*=\s*(.*)$/);
  if (!m || !cur) continue;
  cur[m[1]] = parseValue(m[2].trim());
}
if (cur) characters.push(cur);

const avatars = join(here, "avatars");
mkdirSync(avatars, { recursive: true });
const idols = [];
for (const ch of characters) {
  if (!ch.id || !keep.has(ch.id)) continue;
  const sheet = readMaybe(join(root, ch.sheet || ""));
  const facts = readMaybe(join(root, ch.facts || ""));
  const avatar = String(ch.avatar || "");
  let file = "";
  if (avatar) {
    const src = join(root, avatar);
    try {
      file = `${ch.id}.jpg`;
      copyFileSync(src, join(avatars, file));
    } catch {
      file = "";
    }
  }
  idols.push({
    id: `sc-${ch.id}`,
    name: ch.name_zh || ch.name,
    name_orig: ch.name || "",
    intro: ch.intro || "",
    color: ch.color || "#c47b8a",
    circle_id: circleOf.get(ch.id) || "",
    liked_gifts: Array.isArray(ch.liked_gifts) ? ch.liked_gifts : [],
    sheet,
    facts,
    avatar: file,
    greeting: ch.greeting_subtitle || stripCue(ch.greeting || ""),
    stand: typeof ch.default_outfit === "string" ? ch.default_outfit : "",
  });
}

const catalog = {
  circles: circles.map(([id, name]) => ({ id, name })),
  idols,
};
writeFileSync(join(here, "catalog.json"), JSON.stringify(catalog));
console.log(`idols ${idols.length}, with faces ${idols.filter((i) => i.avatar).length}`);

function readMaybe(path) {
  if (!path) return "";
  try {
    return readFileSync(path, "utf8");
  } catch {
    return "";
  }
}

function stripCue(s) {
  return String(s).replace(/\[[^\]]+\]\s*/g, "").trim();
}

function parseValue(s) {
  if (s.startsWith("[")) return JSON.parse(cutArray(s).replace(/'/g, '"'));
  if (s.startsWith('"')) return scanString(s);
  const n = s.split(/\s+#/)[0].trim();
  if (n === "true") return true;
  if (n === "false") return false;
  if (/^-?\d+$/.test(n)) return Number(n);
  return n;
}

function cutArray(s) {
  let depth = 0;
  let q = false;
  let esc = false;
  for (let i = 0; i < s.length; i++) {
    const c = s[i];
    if (q) {
      if (esc) { esc = false; continue; }
      if (c === "\\") { esc = true; continue; }
      if (c === '"') q = false;
      continue;
    }
    if (c === '"') { q = true; continue; }
    if (c === "[") depth++;
    if (c === "]" && --depth === 0) return s.slice(0, i + 1);
  }
  return s;
}

function scanString(s) {
  let out = "";
  for (let i = 1; i < s.length; i++) {
    const c = s[i];
    if (c === "\\") {
      out += s[i + 1] || "";
      i++;
      continue;
    }
    if (c === '"') return out;
    out += c;
  }
  return out;
}
