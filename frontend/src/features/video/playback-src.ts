/**
 * 本文件拥有把媒体地址改写成当前页面可播路径的规则。不拥有播放器，也不上传文件。
 * 它只改地址字符串，不改服务端。
 */
/** playbackSrc 在 https 或 wails 页面上，把 /media/ 与 /resource/ 收成站内路径。 */
export function playbackSrc(url?: string, page?: { protocol: string; hostname: string }): string {
  const raw = String(url || "").trim();
  if (!raw) return "";
  const loc = page ?? (typeof window === "undefined" ? undefined : {
    protocol: window.location.protocol,
    hostname: window.location.hostname,
  });
  if (!loc) return raw;
  let parsed: URL;
  try {
    parsed = new URL(raw);
  } catch {
    return raw;
  }
  if (!parsed.pathname.startsWith("/media/") && !parsed.pathname.startsWith("/resource/")) return raw;
  const embedded = loc.hostname === "wails.localhost" || loc.hostname.endsWith(".wails.localhost");
  if (loc.protocol === "https:" || embedded) return parsed.pathname + parsed.search;
  return raw;
}
