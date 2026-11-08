/**
 * 拥有：产品站的页面，包括主题、语言、下载和六个真实模块的截图。
 * 不拥有：桌面应用本身。下载地址来自 release.ts。
 */

import { useEffect, useMemo, useState, type KeyboardEvent as ReactKeyboardEvent } from "react";
import { COPY, detectLang, htmlLang, ROOM_SHOT, ROOMS, type Lang, type Room } from "./copy";
import { detectOs, FALLBACK_VERSION, fallbackAssets, hydrateRelease, RELEASES, type Asset } from "./release";
import { PageField } from "./page-field";

const REPO = "https://github.com/Shenchangxin/yoyo";

type Theme = "light" | "dark";

/** shotSrc 拼出 public/shots 里的截图地址。base 为 ./ 时，GitHub Pages 子路径也能找到。 */
function shotSrc(room: Room) {
  return `${import.meta.env.BASE_URL}shots/${ROOM_SHOT[room]}`;
}

/** initialTheme 先看地址栏的 theme，再看文档根上已经写下的 data-theme。都不是 dark 就是 light。 */
function initialTheme(): Theme {
  const query = new URLSearchParams(window.location.search).get("theme");
  if (query === "dark" || query === "light") return query;
  return document.documentElement.dataset.theme === "dark" ? "dark" : "light";
}

/** initialRoom 只接受六个真实模块。地址栏没有，或写错了，就从今天开始。 */
function initialRoom(): Room {
  const query = new URLSearchParams(window.location.search).get("room");
  if (query && (ROOMS as readonly string[]).includes(query)) return query as Room;
  return "today";
}

/** App 渲染产品站。语言、主题、当前模块和发布资源都放在这里。 */
export function App() {
  const [lang, setLang] = useState<Lang>(() => detectLang());
  const [theme, setTheme] = useState<Theme>(() => initialTheme());
  const [menu, setMenu] = useState(false);
  const [zone, setZone] = useState("home");
  const [room, setRoom] = useState<Room>(() => initialRoom());
  const [version, setVersion] = useState(FALLBACK_VERSION);
  const [assets, setAssets] = useState<Asset[]>(() => fallbackAssets(FALLBACK_VERSION));
  const [os, setOs] = useState<Asset["os"]>("windows");
  const t = COPY[lang];
  const current = t.rooms.items.find((item) => item.id === room) ?? t.rooms.items[0];
  const preferred = useMemo(() => assets.find((item) => item.os === os) || assets[0], [assets, os]);

  useEffect(() => {
    document.documentElement.dataset.lang = lang;
    document.documentElement.lang = htmlLang(lang);
    document.title = `${t.meta.title} · ${version}`;
    document.querySelector('meta[name="description"]')?.setAttribute("content", t.meta.description);
    document.querySelector('meta[property="og:title"]')?.setAttribute("content", t.meta.title);
    document.querySelector('meta[property="og:description"]')?.setAttribute("content", t.meta.description);
  }, [lang, t.meta.description, t.meta.title, version]);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    document.documentElement.style.colorScheme = theme;
    document.querySelector('meta[name="theme-color"]')?.setAttribute("content", theme === "dark" ? "#121211" : "#f7f4ee");
  }, [theme]);

  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => {
      try {
        if (localStorage.getItem("yoyo-site-theme")) return;
        if (new URLSearchParams(window.location.search).get("theme")) return;
      } catch {
        return;
      }
      setTheme(media.matches ? "dark" : "light");
    };
    media.addEventListener("change", onChange);
    return () => media.removeEventListener("change", onChange);
  }, []);

  useEffect(() => {
    setOs(detectOs());
    void hydrateRelease().then((data) => {
      setVersion(data.version);
      setAssets(data.assets);
    });
  }, []);

  useEffect(() => {
    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const nodes = [...document.querySelectorAll<HTMLElement>("main > section:not(.hero-home), .foot")];
    if (reduce) {
      nodes.forEach((node) => node.classList.add("is-in"));
      return;
    }
    const io = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting || !(entry.target instanceof HTMLElement)) continue;
          entry.target.classList.add("is-in");
          io.unobserve(entry.target);
        }
      },
      { threshold: 0.14, rootMargin: "0px 0px -6% 0px" },
    );
    nodes.forEach((node) => io.observe(node));
    return () => io.disconnect();
  }, []);

  useEffect(() => {
    document.body.classList.toggle("menu-open", menu);
    return () => document.body.classList.remove("menu-open");
  }, [menu]);

  useEffect(() => {
    const jump = () => {
      const id = window.location.hash.replace(/^#/, "");
      if (!id) return;
      document.getElementById(id)?.scrollIntoView();
    };
    jump();
    window.addEventListener("hashchange", jump);
    return () => window.removeEventListener("hashchange", jump);
  }, []);

  useEffect(() => {
    const nodes = [...document.querySelectorAll<HTMLElement>("[data-zone]")];
    const io = new IntersectionObserver(
      (entries) => {
        const hit = entries
          .filter((entry) => entry.isIntersecting)
          .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0];
        if (hit?.target instanceof HTMLElement && hit.target.dataset.zone) setZone(hit.target.dataset.zone);
      },
      { rootMargin: "-30% 0px -45% 0px", threshold: [0.2, 0.45, 0.7] },
    );
    nodes.forEach((node) => io.observe(node));
    return () => io.disconnect();
  }, []);

  /** chooseLang 调用 setLang。next 传给 setLang，传给 localStorage.setItem。 */
  const chooseLang = (next: Lang) => {
    setLang(next);
    try {
      localStorage.setItem("yoyo-site-lang", next);
    } catch {
      /* ignore */
    }
  };

  const toggleTheme = () => {
    const next = theme === "dark" ? "light" : "dark";
    setTheme(next);
    try {
      localStorage.setItem("yoyo-site-theme", next);
    } catch {
      /* ignore */
    }
  };

  /** onRoomKey 在六个模块之间移动焦点。event 读取 key，调用 preventDefault。 */
  const onRoomKey = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    const index = ROOMS.indexOf(room);
    const next =
      event.key === "ArrowRight" || event.key === "ArrowDown"
        ? ROOMS[(index + 1) % ROOMS.length]
        : event.key === "ArrowLeft" || event.key === "ArrowUp"
          ? ROOMS[(index - 1 + ROOMS.length) % ROOMS.length]
          : event.key === "Home"
            ? ROOMS[0]
            : event.key === "End"
              ? ROOMS[ROOMS.length - 1]
              : null;
    if (!next) return;
    event.preventDefault();
    setRoom(next);
    window.requestAnimationFrame(() => document.getElementById(`room-${next}`)?.focus());
  };

  const closeMenu = () => setMenu(false);

  return (
    <>
      <PageField theme={theme} />
      <a className="skip" href="#download">
        {t.nav.download}
      </a>
      <header className="nav">
        <a className="brand" href="#top" onClick={closeMenu}>
          <Mark />
          <span>yoyo</span>
        </a>
        <nav className="nav-links" aria-label="On this page">
          <a href="#top" aria-current={zone === "home" ? "true" : undefined}>
            {t.nav.home}
          </a>
          <a href="#rooms" aria-current={zone === "rooms" ? "true" : undefined}>
            {t.nav.rooms}
          </a>
          <a href="#local" aria-current={zone === "local" ? "true" : undefined}>
            {t.nav.local}
          </a>
          <a href="#download" aria-current={zone === "download" ? "true" : undefined}>
            {t.nav.download}
          </a>
        </nav>
        <div className="nav-end">
          <LangSwitch lang={lang} onPick={chooseLang} />
          <button type="button" className="icon-btn theme-btn" aria-pressed={theme === "dark"} aria-label={theme === "dark" ? t.nav.themeToLight : t.nav.themeToDark} onClick={toggleTheme}>
            {theme === "dark" ? <SunIcon /> : <MoonIcon />}
          </button>
          <a className="nav-github" href={REPO}>
            GitHub
          </a>
          <a className="btn btn-ink nav-cta" href={preferred?.href || RELEASES} aria-label={t.nav.download}>
            <span className="nav-cta-label">{t.nav.download}</span>
            <span className="btn-orb" aria-hidden="true">
              <DownloadIcon />
            </span>
          </a>
          <button
            type="button"
            className="icon-btn menu-btn"
            aria-expanded={menu}
            aria-controls="nav-panel"
            aria-label={menu ? t.nav.close : t.nav.menu}
            onClick={() => setMenu((open) => !open)}
          >
            <MenuIcon open={menu} />
          </button>
        </div>
        {menu ? <button type="button" className="nav-backdrop" aria-label={t.nav.close} onClick={closeMenu} /> : null}
        <div className="nav-panel" id="nav-panel" hidden={!menu}>
          <nav aria-label="On this page">
            <a href="#top" onClick={closeMenu}>
              {t.nav.home}
            </a>
            <a href="#rooms" onClick={closeMenu}>
              {t.nav.rooms}
            </a>
            <a href="#local" onClick={closeMenu}>
              {t.nav.local}
            </a>
            <a href="#download" onClick={closeMenu}>
              {t.nav.download}
            </a>
            <a href={REPO}>{t.hero.github}</a>
          </nav>
          <div className="nav-panel-tools">
            <LangSwitch lang={lang} onPick={chooseLang} />
            <button type="button" className="text-link" onClick={toggleTheme}>
              {theme === "dark" ? t.nav.themeToLight : t.nav.themeToDark}
            </button>
          </div>
        </div>
      </header>

      <main>
        <section className="hero-home" id="top" data-zone="home">
          <div className="hero-copy">
            <p className="kicker">
              yoyo · {version}
            </p>
            <Lines as="h1" text={t.hero.title} />
            <p className="dek">{t.hero.dek}</p>
            <div className="hero-actions">
              <a className="btn btn-ink btn-hero" href={preferred?.href || RELEASES}>
                <span>
                  {t.hero.download} {version}
                  {preferred ? <span className="btn-os"> · {preferred.label}</span> : null}
                </span>
                <span className="btn-orb" aria-hidden="true">
                  <DownloadIcon />
                </span>
              </a>
              <a className="text-link" href="#download">
                {t.hero.builds}
              </a>
            </div>
            <p className="premise">{t.get.premise}</p>
            <ul className="ledger">
              {t.hero.chips.map((chip) => (
                <li key={chip}>{chip}</li>
              ))}
            </ul>
          </div>
          <figure className="hero-shot">
            <img src={shotSrc("today")} alt={t.hero.shotAlt} />
          </figure>
        </section>

        <section className="section section-stack wrap" id="rooms" data-zone="rooms">
          <header className="section-head">
            <p className="kicker">01</p>
            <Lines text={t.rooms.title} />
            <p className="dek">{t.rooms.dek}</p>
          </header>
          <div className="rooms">
            <div className="room-copy">
              <div className="room-list" role="tablist" aria-label={t.rooms.label} onKeyDown={onRoomKey}>
                {t.rooms.items.map((item) => (
                  <button
                    key={item.id}
                    id={`room-${item.id}`}
                    type="button"
                    role="tab"
                    aria-selected={room === item.id}
                    aria-controls="room-shot"
                    tabIndex={room === item.id ? 0 : -1}
                    onClick={() => setRoom(item.id)}
                  >
                    {item.name}
                  </button>
                ))}
              </div>
              <p className="room-line">{current.line}</p>
            </div>
            <figure className="room-shot" id="room-shot" role="tabpanel" aria-labelledby={`room-${room}`}>
              <img key={room} src={shotSrc(room)} alt={current.alt} />
            </figure>
          </div>
        </section>

        <section className="section split wrap" id="local" data-zone="local">
          <div className="section-copy">
            <p className="kicker">02</p>
            <Lines text={t.local.title} />
            <p className="dek">{t.local.dek}</p>
          </div>
          <ul className="limits">
            {t.local.items.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </section>

        <section className="section split wrap" id="download" data-zone="download">
          <div className="section-copy">
            <p className="kicker">
              {t.get.version} {version}
            </p>
            <Lines text={t.get.title} />
            <p className="dek">{t.get.dek}</p>
            <ol className="start">
              {t.get.steps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
          </div>
          <div className="section-body">
            <div className="downloads">
              {assets.map((asset) => {
                const yours = asset.os === os;
                return (
                  <a key={asset.os} className={yours ? "asset is-you" : "asset"} href={asset.href}>
                    <span className="asset-copy">
                      <b>{asset.label}</b>
                      <span className="asset-name">{asset.name}</span>
                    </span>
                    <span className="asset-meta">
                      {yours ? <em>{t.get.thisMachine}</em> : null}
                      {asset.size ? <span>{asset.size}</span> : null}
                    </span>
                  </a>
                );
              })}
            </div>
            <p className="note">{t.get.note}</p>
          </div>
        </section>
      </main>

      <footer className="foot">
        <div className="wrap foot-row">
          <a className="brand" href="#top">
            <Mark />
            <span>yoyo</span>
          </a>
          <nav>
            <a href={REPO}>GitHub</a>
            <a href={RELEASES}>Releases</a>
            <a href={`${REPO}/blob/master/docs/PRODUCT.md`}>{t.foot.product}</a>
            <a href={`${REPO}/issues`}>{t.foot.issues}</a>
            <a href={`${REPO}/blob/master/LICENSE`}>{t.foot.license}</a>
          </nav>
          <p>
            <span>{t.foot.stack}</span>
            <span>© 2026</span>
          </p>
        </div>
      </footer>
    </>
  );
}

/** Lines 按换行把标题拆成多行，每行一个 span。text 按换行切开。Tag 用作标签。 */
function Lines({ text, as: Tag = "h2" }: { text: string; as?: "h1" | "h2" }) {
  return (
    <Tag>
      {text.split("\n").map((line, index) => (
        <span className="ln" key={index}>
          {line}
        </span>
      ))}
    </Tag>
  );
}

/** LangSwitch 在英文、简体中文和日文之间切换。lang 与 id 比较是否相等。onPick 被调用，并传入 id。 */
function LangSwitch({ lang, onPick }: { lang: Lang; onPick: (lang: Lang) => void }) {
  return (
    <div className="lang" role="group" aria-label="Language">
      {(
        [
          ["en", "EN", "English"],
          ["zh", "中", "简体中文"],
          ["ja", "日", "日本語"],
        ] as const
      ).map(([id, label, name]) => (
        <button key={id} type="button" aria-label={name} aria-pressed={lang === id} onClick={() => onPick(id)}>
          {label}
        </button>
      ))}
    </div>
  );
}

/** Mark 是页眉上的标志，只作装饰。 */
function Mark() {
  return (
    <svg className="brand-mark" viewBox="0 0 32 32" width="18" height="18" aria-hidden="true">
      <rect width="32" height="32" rx="8" fill="currentColor" />
      <g fill="none" stroke="var(--mark-cut)" strokeLinecap="round">
        <circle cx="16" cy="10.2" r="4" strokeWidth="2.2" />
        <path d="M8.2 16.4 16 21.2 23.8 16.4" strokeWidth="2.4" />
        <path d="M16 21.2v5.2" strokeWidth="2.4" />
      </g>
    </svg>
  );
}

/** DownloadIcon 是下载按钮上的箭头。 */
function DownloadIcon() {
  return (
    <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
      <path d="M8 2.2v7.2M5.2 7.2 8 10l2.8-2.8M3.2 13.2h9.6" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

/** SunIcon 表示切到浅色。 */
function SunIcon() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
      <circle cx="12" cy="12" r="3.5" fill="none" stroke="currentColor" strokeWidth="1.6" />
      <path d="M12 3v2.2M12 18.8V21M3 12h2.2M18.8 12H21M5.4 5.4l1.6 1.6M17 17l1.6 1.6M18.6 5.4 17 7M7 17l-1.6 1.6" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    </svg>
  );
}

/** MoonIcon 表示切到深色。 */
function MoonIcon() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
      <path d="M15.2 3.4A7.4 7.4 0 1 0 20.6 14 5.8 5.8 0 0 1 15.2 3.4Z" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
    </svg>
  );
}

/** MenuIcon 在菜单打开时画叉，收起时画两条横线。open 为真时才继续。 */
function MenuIcon({ open }: { open: boolean }) {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
      {open ? (
        <path d="M6 6l12 12M18 6 6 18" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
      ) : (
        <path d="M4 8h16M4 16h16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
      )}
    </svg>
  );
}
