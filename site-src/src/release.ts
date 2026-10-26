/**
 * 拥有：产品站上的安装包地址。拉不到 GitHub 最新发布时，用 FALLBACK_VERSION 拼出文件名。
 * 不拥有：安装包怎么签名。
 */

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

/** named 拼出 v 加版本号下面的下载地址。 */
function named(ver: string, file: string) {
  return `https://github.com/${REPO}/releases/download/v${ver}/${file}`;
}

/** formatBytes 小于 1 MiB 时用 KB，否则用四舍五入到整数的 MB。0 返回空串。 */
function formatBytes(n: number) {
  if (!n) return "";
  if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB`;
  return `${(n / (1024 * 1024)).toFixed(0)} MB`;
}

/** fallbackAssets 按版本号拼出 Windows、macOS、Linux、deb 和 rpm 五条地址。尺寸留空，等 hydrateRelease 填。 */
export function fallbackAssets(ver: string): Asset[] {
  return [
    { os: "windows", label: "Windows", name: `Yoyo-${ver}-windows-x64-setup.exe`, href: named(ver, `Yoyo-${ver}-windows-x64-setup.exe`), size: "" },
    { os: "mac", label: "macOS", name: `Yoyo-${ver}-darwin-universal.dmg`, href: named(ver, `Yoyo-${ver}-darwin-universal.dmg`), size: "" },
    { os: "linux", label: "Linux", name: `Yoyo-${ver}-linux-amd64.AppImage`, href: named(ver, `Yoyo-${ver}-linux-amd64.AppImage`), size: "" },
    { os: "deb", label: "Debian", name: `Yoyo-${ver}-linux-amd64.deb`, href: named(ver, `Yoyo-${ver}-linux-amd64.deb`), size: "" },
    { os: "rpm", label: "RPM", name: `Yoyo-${ver}-linux-amd64.rpm`, href: named(ver, `Yoyo-${ver}-linux-amd64.rpm`), size: "" },
  ];
}

/** detectOs 用浏览器标识区分 mac、linux 和 windows。Android 不算 linux。认不出时返回 windows。 */
export function detectOs(): Asset["os"] {
  const ua = navigator.userAgent;
  if (/Mac OS X|Macintosh/.test(ua)) return "mac";
  if (/Linux/.test(ua) && !/Android/.test(ua)) return "linux";
  return "windows";
}

/** pick 按正则顺序找第一个文件名匹配的发布资产。都没有就返回 null。 */
function pick(assets: { name: string; size: number; browser_download_url: string }[], tests: RegExp[]) {
  for (const test of tests) {
    const hit = assets.find((a) => test.test(a.name));
    if (hit) return hit;
  }
  return null;
}

/** hydrateRelease 读取 GitHub 最新发布。失败或响应不是 200 时，返回 FALLBACK_VERSION 和空尺寸的备用地址。成功时用真实文件名、地址和大小盖住对应条目。 */
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
