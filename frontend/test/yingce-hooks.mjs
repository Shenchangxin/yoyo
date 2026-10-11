import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const frontend = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

function withExtension(base) {
  if (fs.existsSync(base) && fs.statSync(base).isFile()) return base;
  for (const ext of [".ts", ".tsx", ".js", ".mjs"]) {
    if (fs.existsSync(base + ext)) return base + ext;
  }
  for (const ext of [".ts", ".tsx", ".js"]) {
    const index = path.join(base, "index" + ext);
    if (fs.existsSync(index)) return index;
  }
  return base;
}

function mapSpecifier(specifier) {
  if (specifier.startsWith("@yingce/")) return withExtension(path.join(frontend, "src/vendor/yingce-canvas", specifier.slice("@yingce/".length)));
  if (specifier.startsWith("@/")) return withExtension(path.join(frontend, "src", specifier.slice(2)));
  return "";
}

export async function resolve(specifier, context, nextResolve) {
  const mapped = mapSpecifier(specifier);
  if (mapped) return nextResolve(pathToFileURL(mapped).href, context);
  if (specifier.startsWith(".") && !path.extname(specifier)) {
    const parent = context.parentURL ? fileURLToPath(context.parentURL) : "";
    if (parent) {
      const resolved = withExtension(path.resolve(path.dirname(parent), specifier));
      if (fs.existsSync(resolved)) return nextResolve(pathToFileURL(resolved).href, context);
    }
  }
  return nextResolve(specifier, context);
}
