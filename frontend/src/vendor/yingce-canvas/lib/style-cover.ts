/** Resolve cover URLs for vendored 影策 style stills shipped in frontend/public. */

const PALETTES: Record<string, [string, string, string]> = {
  "cyberpunk-neon": ["#14081f", "#ff2f9e", "#2ce0ff"],
  "retro-hong-kong": ["#2c1408", "#e07a3a", "#f2c14e"],
  "suspense-noir": ["#0c0c0e", "#6b6b72", "#d8d6cf"],
  "fantasy-3d": ["#1a1230", "#7c5cff", "#f0c27a"],
  "storybook-fantasy": ["#1d2a18", "#7dae5a", "#f3e2b0"],
  "urban-live-action": ["#1c1814", "#8a7460", "#e8dcc8"],
  "future-tech": ["#07141c", "#1f6feb", "#7ee0c8"],
  "nature-healing": ["#122016", "#4f8f62", "#d7eccf"],
  "real-life": ["#241c14", "#c4894c", "#f0e0c8"],
  "black-white-noir": ["#111111", "#8a8a8a", "#efefef"],
  "period-live-action": ["#2a1c12", "#b07940", "#e8d2a8"],
  "space-opera": ["#070b18", "#3d5cff", "#c9d4ff"],
  "clay-stop-motion": ["#2a1a10", "#e09a4a", "#f3d7a8"],
  "ink-narrative": ["#14110c", "#4a4338", "#efe6d6"],
  "chinese-2d": ["#1a1220", "#c45c4a", "#e8c9a0"],
  "surreal-dream": ["#1c1430", "#c48ad4", "#8ec8e8"],
  "comic-pop": ["#1a1020", "#ee4b3c", "#f5d24a"],
  "three-d-cartoon": ["#20140c", "#e07a3a", "#f4d7a4"],
  "warm-interior": ["#2a1c10", "#c47a3a", "#ead2a4"],
};

function keyFromSrc(src: string) {
  const file = src.split("/").pop() || "cover";
  return file.replace(/\.(jpg|jpeg|png|webp|svg)$/i, "");
}

function hashPalette(key: string): [string, string, string] {
  let h = 0;
  for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) | 0;
  const hue = Math.abs(h) % 360;
  return [`hsl(${hue} 28% 10%)`, `hsl(${(hue + 28) % 360} 62% 52%)`, `hsl(${(hue + 48) % 360} 70% 72%)`];
}

function svgPoster(key: string) {
  const [a, b, c] = PALETTES[key] || hashPalette(key);
  const label = key.replace(/-/g, " ");
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 400" preserveAspectRatio="xMidYMid slice">
    <defs>
      <linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0" stop-color="${a}"/>
        <stop offset=".55" stop-color="${b}"/>
        <stop offset="1" stop-color="${c}"/>
      </linearGradient>
    </defs>
    <rect width="640" height="400" fill="url(#g)"/>
    <rect x="36" y="248" width="180" height="8" rx="4" fill="#fff" fill-opacity=".28"/>
    <rect x="36" y="268" width="280" height="8" rx="4" fill="#fff" fill-opacity=".16"/>
    <text x="36" y="232" fill="#fff" fill-opacity=".88" font-family="ui-sans-serif,system-ui,sans-serif" font-size="22" font-weight="600">${label}</text>
  </svg>`;
}

export function styleCoverFallback(src: string) {
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svgPoster(keyFromSrc(src || "cover")))}`;
}

export function styleCoverSrc(src: string) {
  if (!src) return styleCoverFallback("cover");
  return src;
}
