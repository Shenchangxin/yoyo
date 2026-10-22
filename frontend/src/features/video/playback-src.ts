/** Play a CAS url on the app origin when the webview would block the loopback server. */
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
