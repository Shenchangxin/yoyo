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
    const next = String(tag).replace(/^v/, "");
    versionEls.forEach((el) => {
      el.textContent = next;
    });
    document.title = `Yoyo ${next}`;

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
  const chips = $$(".chip[data-kind]");
  const buttons = $$(".gates button");
  const select = (kind) => {
    buttons.forEach((btn) => btn.classList.toggle("is-on", btn.dataset.kind === kind));
    chips.forEach((chip) => chip.classList.toggle("is-hot", chip.dataset.kind === kind));
    $$(".verdict > [data-kind]").forEach((line) => line.classList.toggle("is-on", line.dataset.kind === kind));
    $$(".agent-mini").forEach((card) => {
      card.classList.add("is-settled");
      card.classList.toggle("is-denied", kind === "deny");
    });
  };
  buttons.forEach((btn) => btn.addEventListener("click", () => select(btn.dataset.kind)));
  chips.forEach((chip) => {
    if (chip instanceof HTMLButtonElement) chip.addEventListener("click", () => select(chip.dataset.kind));
  });
  document.addEventListener("keydown", (event) => {
    if (event.metaKey || event.ctrlKey || event.altKey) return;
    if (event.target.closest("input, textarea, select")) return;
    const kind = { 1: "once", 2: "session", 3: "always", Escape: "deny" }[event.key];
    if (kind) select(kind);
  });
}

function bindTilt() {
  const hero = $(".hero");
  const windowEl = $(".hero .window");
  if (!hero || !windowEl) return;
  if (window.matchMedia("(prefers-reduced-motion: reduce), (pointer: coarse)").matches) return;
  hero.addEventListener("mousemove", (event) => {
    const rect = hero.getBoundingClientRect();
    const x = (event.clientX - rect.left) / rect.width - 0.5;
    const y = (event.clientY - rect.top) / rect.height - 0.5;
    windowEl.style.setProperty("--tilt-y", `${(x * 5).toFixed(2)}deg`);
    windowEl.style.setProperty("--tilt-x", `${(-y * 3.5).toFixed(2)}deg`);
  });
  hero.addEventListener("mouseleave", () => {
    windowEl.style.setProperty("--tilt-x", "0deg");
    windowEl.style.setProperty("--tilt-y", "0deg");
  });
}

const NAV = { agent: "reel", review: "reel", video: "video", harness: "harness" };

function markNav(zone) {
  const key = NAV[zone] || "";
  $$(".nav-toc a").forEach((link) => {
    if (link.dataset.nav === key) link.setAttribute("aria-current", "page");
    else link.removeAttribute("aria-current");
  });
}

function bindReel() {
  const reel = $(".reel");
  const articles = $$(".reel-story article");
  const frames = $$(".frame");
  if (!reel || !articles.length) return;
  frames.forEach((frame) => frame.toggleAttribute("inert", !frame.classList.contains("is-on")));
  const show = (scene) => {
    reel.dataset.scene = scene;
    document.documentElement.dataset.zone = scene;
    markNav(scene);
    articles.forEach((article) => article.classList.toggle("is-on", article.dataset.scene === scene));
    frames.forEach((frame) => {
      const on = frame.dataset.scene === scene;
      frame.classList.toggle("is-on", on);
      frame.toggleAttribute("inert", !on);
    });
  };
  const io = new IntersectionObserver(
    (entries) => {
      const hit = entries
        .filter((entry) => entry.isIntersecting)
        .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0];
      if (hit) show(hit.target.dataset.scene);
    },
    { rootMargin: "-20% 0px -35% 0px", threshold: [0.25, 0.55] },
  );
  articles.forEach((article) => io.observe(article));
}

function bindFrames() {
  const review = $(".review-mini");
  if (review) {
    const panes = $$(".rpane", review);
    const tabs = $$(".rtabs button", review);
    const hunks = $$(".hunk", review);
    const count = $(".hunk-count", review);
    const apply = $(".apply", review);
    const applied = $(".applied", review);
    const sync = () => {
      const n = hunks.filter((hunk) => hunk.classList.contains("is-on")).length;
      const num = apply?.querySelector("span");
      if (num) num.textContent = String(n);
      if (apply) apply.disabled = n === 0;
      if (count) count.textContent = n === 1 ? "1 hunk" : `${n} hunks`;
    };
    tabs.forEach((tab) => {
      tab.addEventListener("click", () => {
        tabs.forEach((item) => item.setAttribute("aria-selected", String(item === tab)));
        panes.forEach((pane) => pane.classList.toggle("is-on", pane.dataset.pane === tab.dataset.pane));
      });
    });
    hunks.forEach((hunk) => {
      hunk.addEventListener("click", () => {
        const on = !hunk.classList.contains("is-on");
        hunk.classList.toggle("is-on", on);
        hunk.setAttribute("aria-pressed", String(on));
        if (applied) applied.hidden = true;
        sync();
      });
    });
    apply?.addEventListener("click", () => {
      const n = hunks.filter((hunk) => hunk.classList.contains("is-on")).length;
      if (!n) return;
      hunks.forEach((hunk) => {
        hunk.classList.remove("is-on");
        hunk.setAttribute("aria-pressed", "false");
      });
      if (applied) {
        applied.hidden = false;
        applied.textContent = `Applied ${n}`;
      }
      sync();
    });
  }

  const video = $(".video-mini");
  if (video) {
    const modes = $$(".vseg button", video);
    const panels = $$(".vpanel", video);
    modes.forEach((mode) => {
      mode.addEventListener("click", () => {
        modes.forEach((item) => item.setAttribute("aria-selected", String(item === mode)));
        panels.forEach((panel) => panel.classList.toggle("is-on", panel.dataset.mode === mode.dataset.mode));
      });
    });
    $(".send-btn", video)?.addEventListener("click", () => {
      $(".create-still", video)?.classList.add("is-live");
    });
    const episodes = $$(".eps button", video);
    const still = $(".shot-still", video);
    const unshot = $(".unshot", video);
    const caption = still?.querySelector("figcaption");
    episodes.forEach((episode) => {
      episode.addEventListener("click", () => {
        episodes.forEach((item) => item.classList.toggle("live", item === episode));
        const shot = episode.dataset.ep === "02";
        if (still) still.hidden = !shot;
        if (unshot) unshot.hidden = shot;
        if (caption && shot) caption.textContent = "02 · The lamp";
      });
    });
  }

  const harness = $(".harness-mini");
  if (harness) {
    const refs = $$(".refs button", harness);
    const notes = $$(".harbor-note", harness);
    const checkout = $(".checkout", harness);
    const show = (ref) => {
      const promoted = harness.dataset.promoted === "1";
      const note = promoted && ref === "active" ? "done" : ref;
      refs.forEach((item) => item.setAttribute("aria-pressed", String(item.dataset.ref === ref)));
      notes.forEach((item) => {
        item.hidden = item.dataset.ref !== note;
      });
      if (checkout) checkout.disabled = promoted || ref !== "canary";
    };
    refs.forEach((ref) => ref.addEventListener("click", () => show(ref.dataset.ref)));
    checkout?.addEventListener("click", () => {
      harness.dataset.promoted = "1";
      show("active");
    });
    show("canary");
  }
}

function bindLayers() {
  $$(".layer").forEach((layer) => {
    layer.addEventListener("click", () => {
      $$(".layer").forEach((item) => item.classList.toggle("is-open", item === layer));
    });
  });
}

function bindDoors() {
  const tabs = $$(".door-tabs button");
  const doors = $$(".door");
  tabs.forEach((tab) => {
    tab.addEventListener("click", () => {
      const door = tab.dataset.door;
      tabs.forEach((item) => item.setAttribute("aria-selected", String(item === tab)));
      doors.forEach((panel) => panel.classList.toggle("is-on", panel.dataset.door === door));
    });
  });
}

function bindZones() {
  const zones = [
    ["#layers", "layers"],
    ["#surfaces", "surfaces"],
    ["#desk", "desk"],
    ["#wallpaper", "paper"],
    ["#download", "get"],
  ];
  const io = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (!entry.isIntersecting) return;
        document.documentElement.dataset.zone = entry.target.dataset.zone;
        markNav(entry.target.dataset.zone);
      });
    },
    { rootMargin: "-45% 0px -45% 0px", threshold: 0 },
  );
  zones.forEach(([sel, zone]) => {
    const el = $(sel);
    if (!el) return;
    el.dataset.zone = zone;
    io.observe(el);
  });
  const desk = $(".desk");
  if (!desk) return;
  const meter = new IntersectionObserver(
    (entries) => {
      if (entries.some((entry) => entry.isIntersecting)) desk.classList.add("is-in");
    },
    { threshold: 0.4 },
  );
  meter.observe(desk);
}

function markMachine() {
  const ua = navigator.userAgent;
  let os = "windows";
  if (/Mac OS X|Macintosh/.test(ua)) os = "mac";
  else if (/Linux/.test(ua) && !/Android/.test(ua)) os = "linux";
  $(`[data-os="${os}"]`)?.classList.add("is-you");
}

function bindScrollStage() {
  const hero = $(".hero");
  const windowEl = $(".window");
  if (!hero || !windowEl || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
  let raf = 0;
  const update = () => {
    raf = 0;
    const y = Math.min(1, Math.max(0, window.scrollY / Math.max(hero.offsetHeight, 1)));
    windowEl.style.setProperty("--pan-y", `${10 + y * 40}px`);
    windowEl.style.setProperty("--pan-x", `${y * -14}px`);
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

function bindHarnessPath() {
  const steps = $$(".path article");
  if (!steps.length || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
  const io = new IntersectionObserver(
    (entries) => {
      const visible = entries
        .filter((entry) => entry.isIntersecting)
        .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
      if (!visible.length) return;
      steps.forEach((step) => step.classList.remove("is-on"));
      visible[0].target.classList.add("is-on");
    },
    { rootMargin: "-35% 0px -45% 0px", threshold: 0.2 },
  );
  steps.forEach((step) => io.observe(step));
}

function bindClock() {
  const clock = $("[data-clock]");
  if (!clock || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
  let seconds = 12;
  window.setInterval(() => {
    seconds += 1;
    const m = Math.floor(seconds / 60);
    const s = String(seconds % 60).padStart(2, "0");
    clock.textContent = `${m}:${s}`;
  }, 1000);
}

document.addEventListener("DOMContentLoaded", () => {
  setLang(detectLang());
  $$(".lang button").forEach((btn) => {
    btn.addEventListener("click", () => setLang(btn.dataset.lang));
  });
  bindVeil();
  bindGates();
  bindTilt();
  bindScrollStage();
  bindReel();
  bindFrames();
  bindLayers();
  bindDoors();
  bindZones();
  bindClock();
  markMachine();
  document.documentElement.dataset.zone = "agent";
  void hydrateRelease();
});
