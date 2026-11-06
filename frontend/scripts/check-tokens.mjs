/**
 * 拥有：功能代码里字号、圆角、渐变和投影只能走令牌的守卫。
 * 不拥有：令牌本身。刻度在 src/styles.css 的 @theme，规则在 docs/uiux-plan.md §6。
 *
 * 旧的写法记在 token-baseline.json 里，按文件计数。一个文件的计数只能变少，新文件必须是零。
 * 清掉一批旧写法之后，用 `node scripts/check-tokens.mjs --update` 把基线收紧。
 */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const src = path.join(here, "..", "src");
const baselinePath = path.join(here, "token-baseline.json");

const rules = {
  text: /\btext-\[\d+(?:\.\d+)?px\]/g,
  radius: /\brounded(?:-(?:t|b|l|r|tl|tr|bl|br|s|e|ss|se|es|ee))?-(?:\[\d+(?:\.\d+)?(?:px|rem)\]|2xl|3xl|4xl)\b/g,
  gradient: /\bbg-(?:\[(?:linear|radial|conic)-gradient|gradient-to-|linear-to-|radial\b)/g,
  shadow: /\bshadow-\[(?!var\(--shadow-)/g,
};

function walk(dir, out) {
  for (const name of fs.readdirSync(dir)) {
    const full = path.join(dir, name);
    const rel = path.relative(src, full).replace(/\\/g, "/");
    if (fs.statSync(full).isDirectory()) {
      if (rel === "vendor") continue;
      walk(full, out);
    } else if (/\.tsx?$/.test(name) && !/\.test\.ts$/.test(name)) {
      out.push(rel);
    }
  }
  return out;
}

function count(text) {
  const out = {};
  for (const [key, re] of Object.entries(rules)) {
    const n = (text.match(re) || []).length;
    if (n) out[key] = n;
  }
  return out;
}

const current = {};
for (const rel of walk(src, []).sort()) {
  const hits = count(fs.readFileSync(path.join(src, rel), "utf8"));
  if (Object.keys(hits).length) current[rel] = hits;
}

if (process.argv.includes("--update")) {
  fs.writeFileSync(baselinePath, `${JSON.stringify(current, null, 2)}\n`);
  const total = Object.values(current).reduce((n, row) => n + Object.values(row).reduce((a, b) => a + b, 0), 0);
  console.log(`token baseline: ${Object.keys(current).length} files, ${total} legacy uses`);
  process.exit(0);
}

const baseline = fs.existsSync(baselinePath) ? JSON.parse(fs.readFileSync(baselinePath, "utf8")) : {};
const problems = [];
for (const [rel, hits] of Object.entries(current)) {
  for (const [key, n] of Object.entries(hits)) {
    const allowed = baseline[rel]?.[key] || 0;
    if (n > allowed) problems.push(`${rel}: ${key} ${n} (baseline ${allowed})`);
  }
}

if (problems.length) {
  console.error("Use the ui-* type scale, the radius scale, and --shadow-* tokens instead of raw values:");
  for (const line of problems) console.error(`  ${line}`);
  console.error("Scale: text-ui-{badge,caption,footnote,body,body-l,title-s,title,display}; rounded-{xs,sm,md,lg,xl,full}.");
  process.exit(1);
}
console.log("tokens ok");
