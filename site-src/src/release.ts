const REPO = "Shenchangxin/yoyo";
export const RELEASES = `https://github.com/${REPO}/releases/latest`;
export const FALLBACK_VERSION = "0.4.2";

export type Asset = {
  os: "windows" | "mac" | "linux" | "deb" | "rpm";
  label: string;
  name: string;
  href: string;
  size: string;
};

function named(ver: string, file: string) {
  return `https://github.com/${REPO}/releases/download/v${ver}/${file}`;
}

function formatBytes(n: number) {
  if (!n) return "";
  if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB`;
  return `${(n / (1024 * 1024)).toFixed(0)} MB`;
}

export function fallbackAssets(ver: string): Asset[] {
  return [
    { os: "windows", label: "Windows", name: `Yoyo-${ver}-windows-x64-setup.exe`, href: named(ver, `Yoyo-${ver}-windows-x64-setup.exe`), size: "" },
    { os: "mac", label: "macOS", name: `Yoyo-${ver}-darwin-universal.dmg`, href: named(ver, `Yoyo-${ver}-darwin-universal.dmg`), size: "" },
    { os: "linux", label: "Linux", name: `Yoyo-${ver}-linux-amd64.AppImage`, href: named(ver, `Yoyo-${ver}-linux-amd64.AppImage`), size: "" },
    { os: "deb", label: "Debian", name: `Yoyo-${ver}-linux-amd64.deb`, href: named(ver, `Yoyo-${ver}-linux-amd64.deb`), size: "" },
    { os: "rpm", label: "RPM", name: `Yoyo-${ver}-linux-amd64.rpm`, href: named(ver, `Yoyo-${ver}-linux-amd64.rpm`), size: "" },
  ];
}

export function detectOs(): Asset["os"] {
  const ua = navigator.userAgent;
  if (/Mac OS X|Macintosh/.test(ua)) return "mac";
  if (/Linux/.test(ua) && !/Android/.test(ua)) return "linux";
  return "windows";
}

function pick(assets: { name: string; size: number; browser_download_url: string }[], tests: RegExp[]) {
  for (const test of tests) {
    const hit = assets.find((a) => test.test(a.name));
    if (hit) return hit;
  }
  return null;
}

export async function hydrateRelease(): Promise<{ version: string; assets: Asset[] }> {
  const base = fallbackAssets(FALLBACK_VERSION);
  try {
    const res = await fetch(`https://api.github.com/repos/${REPO}/releases/latest`);
    if (!res.ok) throw new Error("release");
    const data = (await res.json()) as {
      tag_name?: string;
      assets?: { name: string; size: number; browser_download_url: string }[];
    };
    const version = String(data.tag_name || FALLBACK_VERSION).replace(/^v/, "");
    const raw = data.assets || [];
    const mapped = fallbackAssets(version);
    const win = pick(raw, [/windows-x64-setup\.exe$/i]);
    const mac = pick(raw, [/darwin-universal\.dmg$/i, /\.dmg$/i]);
    const linux = pick(raw, [/linux-amd64\.AppImage$/i, /\.AppImage$/i]);
    const deb = pick(raw, [/\.deb$/i]);
    const rpm = pick(raw, [/\.rpm$/i]);
    const overlay = [win, mac, linux, deb, rpm];
    return {
      version,
      assets: mapped.map((item, i) => {
        const hit = overlay[i];
        if (!hit) return item;
        return { ...item, name: hit.name, href: hit.browser_download_url, size: formatBytes(hit.size) };
      }),
    };
  } catch {
    return { version: FALLBACK_VERSION, assets: base };
  }
}
