const REPO = "Shenchangxin/yoyo";
const RELEASES = `https://github.com/${REPO}/releases/latest`;
const LANGS = ["en", "zh", "ja"];

function fallbackVer() {
  return document.querySelector("[data-version]")?.textContent?.trim() || "0.3.9";
}

function namedAsset(ver, file) {
  return {
    name: file,
    size: 0,
    browser_download_url: `https://github.com/${REPO}/releases/download/v${ver}/${file}`,
  };
}

const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];

function detectLang() {
  const saved = localStorage.getItem("yoyo-site-lang");
  if (LANGS.includes(saved)) return saved;
  const nav = (navigator.language || "en").toLowerCase();
  if (nav.startsWith("zh")) return "zh";
  if (nav.startsWith("ja")) return "ja";
  return "en";
}

function setLang(lang) {
  const next = LANGS.includes(lang) ? lang : "en";
  document.documentElement.dataset.lang = next;
  document.documentElement.lang = next === "zh" ? "zh-Hans" : next;
  localStorage.setItem("yoyo-site-lang", next);
  $$(".lang button").forEach((btn) => {
    btn.setAttribute("aria-pressed", String(btn.dataset.lang === next));
  });
}

function pickAsset(assets, tests) {
  for (const test of tests) {
    const hit = assets.find((a) => test.test(a.name));
    if (hit) return hit;
  }
  return null;
}

function formatBytes(n) {
  if (!n) return "";
  if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB`;
  return `${(n / (1024 * 1024)).toFixed(0)} MB`;
}

function bindDownload(el, asset, fallback) {
  if (!el) return;
  el.href = asset?.browser_download_url || fallback;
  const file = el.querySelector(".file");
  const size = el.querySelector(".size");
  if (file && asset) file.textContent = asset.name;
  if (size && asset) size.textContent = formatBytes(asset.size);
}

function applyFallbacks(ver) {
  bindDownload($('[data-os="windows"]'), namedAsset(ver, `Yoyo-${ver}-windows-x64-setup.exe`), RELEASES);
  bindDownload($('[data-os="mac"]'), namedAsset(ver, `Yoyo-${ver}-darwin-universal.dmg`), RELEASES);
  bindDownload($('[data-os="linux"]'), namedAsset(ver, `Yoyo-${ver}-linux-amd64.AppImage`), RELEASES);
  bindDownload($('[data-os="deb"]'), namedAsset(ver, `Yoyo-${ver}-linux-amd64.deb`), RELEASES);
  bindDownload($('[data-os="rpm"]'), namedAsset(ver, `Yoyo-${ver}-linux-amd64.rpm`), RELEASES);
  const primary = $("#download-primary");
  const ua = navigator.userAgent;
  let preferred = namedAsset(ver, `Yoyo-${ver}-windows-x64-setup.exe`);
  if (/Mac OS X|Macintosh/.test(ua)) preferred = namedAsset(ver, `Yoyo-${ver}-darwin-universal.dmg`);
  else if (/Linux/.test(ua) && !/Android/.test(ua)) preferred = namedAsset(ver, `Yoyo-${ver}-linux-amd64.AppImage`);
  if (primary) primary.href = preferred.browser_download_url;
}

async function hydrateRelease() {
  const primary = $("#download-primary");
  const versionEls = $$("[data-version]");
  const ver = fallbackVer();
  applyFallbacks(ver);
  try {
    const res = await fetch(`https://api.github.com/repos/${REPO}/releases/latest`);
    if (!res.ok) throw new Error("release");
    const data = await res.json();
    const tag = data.tag_name || "latest";
    const ver = String(tag).replace(/^v/, "");
    versionEls.forEach((el) => {
      el.textContent = ver;
    });
    document.title = `Yoyo ${ver}`;

    const assets = data.assets || [];
    const win = pickAsset(assets, [/windows-x64-setup\.exe$/i, /windows-amd64-setup\.exe$/i]);
    const mac = pickAsset(assets, [/darwin-universal\.dmg$/i, /\.dmg$/i]);
    const linux = pickAsset(assets, [/linux-amd64\.AppImage$/i, /\.AppImage$/i]);
    const deb = pickAsset(assets, [/\.deb$/i]);
    const rpm = pickAsset(assets, [/\.rpm$/i]);

    if (win) bindDownload($('[data-os="windows"]'), win, RELEASES);
    if (mac) bindDownload($('[data-os="mac"]'), mac, RELEASES);
    if (linux) bindDownload($('[data-os="linux"]'), linux, RELEASES);
    if (deb) bindDownload($('[data-os="deb"]'), deb, RELEASES);
    if (rpm) bindDownload($('[data-os="rpm"]'), rpm, RELEASES);

    const ua = navigator.userAgent;
    let preferred = win;
    if (/Mac OS X|Macintosh/.test(ua)) preferred = mac;
    else if (/Linux/.test(ua) && !/Android/.test(ua)) preferred = linux;
    if (primary) {
      primary.href = preferred?.browser_download_url || RELEASES;
    }
  } catch {
    versionEls.forEach((el) => {
      if (!el.textContent) el.textContent = "0.3.9";
    });
  }
}

function bindVeil() {
  const slider = $("#veil");
  const skin = $(".skin");
  if (!slider || !skin) return;
  const apply = () => skin.style.setProperty("--veil", String(Number(slider.value) / 100));
  slider.addEventListener("input", apply);
  apply();
}

function bindGates() {
  $$(".gates button").forEach((btn) => {
    btn.addEventListener("click", () => {
      $$(".gates button").forEach((b) => b.classList.remove("is-on"));
      btn.classList.add("is-on");
    });
  });
}

function bindScrollStage() {
  const hero = $(".hero");
  const windowEl = $(".window");
  if (!hero || !windowEl || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
  let raf = 0;
  const update = () => {
    raf = 0;
    const y = Math.min(1, Math.max(0, window.scrollY / Math.max(hero.offsetHeight, 1)));
    windowEl.style.setProperty("--pan-y", `${18 + y * 36}px`);
    windowEl.style.setProperty("--pan-x", `${y * -12}px`);
  };
  window.addEventListener(
    "scroll",
    () => {
      if (!raf) raf = requestAnimationFrame(update);
    },
    { passive: true },
  );
  update();
}

document.addEventListener("DOMContentLoaded", () => {
  setLang(detectLang());
  $$(".lang button").forEach((btn) => {
    btn.addEventListener("click", () => setLang(btn.dataset.lang));
  });
  bindVeil();
  bindGates();
  bindScrollStage();
  void hydrateRelease();
});
