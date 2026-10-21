import { useEffect, useMemo, useState, type KeyboardEvent as ReactKeyboardEvent } from "react";
import { COPY, detectLang, htmlLang, type GateKind, type Lang, type Scene } from "./copy";
import { detectOs, FALLBACK_VERSION, fallbackAssets, hydrateRelease, RELEASES, type Asset } from "./release";
import { SpaceStill } from "./space-still";
import { Workstation, type VideoTab } from "./workstation";

const SCENES: Scene[] = ["ask", "diff", "drama", "trust"];
const REPO = "https://github.com/Shenchangxin/yoyo";

type Theme = "light" | "dark";
type RefKey = "staging" | "canary" | "active";

function initialTheme(): Theme {
  const query = new URLSearchParams(window.location.search).get("theme");
  if (query === "dark" || query === "light") return query;
  return document.documentElement.dataset.theme === "dark" ? "dark" : "light";
}

export function App() {
  const [lang, setLang] = useState<Lang>(() => detectLang());
  const [theme, setTheme] = useState<Theme>(() => initialTheme());
  const [menu, setMenu] = useState(false);
  const [zone, setZone] = useState("workstation");
  const [stage, setStage] = useState<Scene>(() => {
    const query = new URLSearchParams(window.location.search).get("scene");
    if (query === "ask" || query === "diff" || query === "drama" || query === "trust") return query;
    return "ask";
  });
  const [gate, setGate] = useState<GateKind>("once");
  const [promoted, setPromoted] = useState(false);
  const [refKey, setRefKey] = useState<RefKey>("canary");
  const [hunks, setHunks] = useState<[boolean, boolean]>([true, true]);
  const [episode, setEpisode] = useState("02");
  const [filmMode, setFilmMode] = useState(1);
  const filmTab: VideoTab = filmMode === 0 ? "create" : filmMode === 2 ? "canvas" : "video";
  const [version, setVersion] = useState(FALLBACK_VERSION);
  const [assets, setAssets] = useState<Asset[]>(() => fallbackAssets(FALLBACK_VERSION));
  const [os, setOs] = useState<Asset["os"]>("windows");
  const t = COPY[lang];
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
    const nodes = [...document.querySelectorAll<HTMLElement>("main > section:not(.hero), .foot")];
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

  useEffect(() => {
    const onKey = (event: globalThis.KeyboardEvent) => {
      if (event.metaKey || event.ctrlKey || event.altKey) return;
      if (menu && event.key === "Escape") {
        setMenu(false);
        return;
      }
      const el = event.target as HTMLElement | null;
      if (el?.closest("input, textarea, select")) return;
      if (stage !== "ask") return;
      const next = ({ "1": "once", "2": "session", "3": "always", Escape: "deny" } as const)[event.key];
      if (next) setGate(next);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [menu, stage]);

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

  const onTabKey = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    const index = SCENES.indexOf(stage);
    const next =
      event.key === "ArrowRight"
        ? SCENES[(index + 1) % SCENES.length]
        : event.key === "ArrowLeft"
          ? SCENES[(index - 1 + SCENES.length) % SCENES.length]
          : event.key === "Home"
            ? SCENES[0]
            : event.key === "End"
              ? SCENES[SCENES.length - 1]
              : null;
    if (!next) return;
    event.preventDefault();
    setStage(next);
    window.requestAnimationFrame(() => document.getElementById(`scene-${next}`)?.focus());
  };

  const checkout = () => {
    setPromoted(true);
    setRefKey("active");
  };

  const caption = t.stage[stage];
  const sceneLabel: Record<Scene, string> = {
    ask: t.demo.agent,
    diff: t.demo.review,
    drama: t.demo.video,
    trust: t.nav.trust,
  };
  const refNote =
    promoted && refKey === "active"
      ? t.trust.done
      : refKey === "staging"
        ? t.trust.stagingHint
        : refKey === "canary"
          ? t.trust.canaryHint
          : t.trust.activeHint;
  const layers = [
    ["l1", t.model.l1, t.model.l1Body],
    ["l2", t.model.l2, t.model.l2Body],
    ["l3", t.model.l3, t.model.l3Body],
    ["l4", t.model.l4, t.model.l4Body],
  ] as const;
  const refs = [
    ["staging", t.trust.staging, t.trust.stagingHint],
    ["canary", t.trust.canary, t.trust.canaryHint],
    ["active", t.trust.active, t.trust.activeHint],
  ] as const;

  const closeMenu = () => setMenu(false);

  return (
    <>
      <a className="skip" href="#download">
        {t.nav.download}
      </a>
      <header className="nav">
        <a className="brand" href="#top" onClick={closeMenu}>
          <Mark />
          <span>yoyo</span>
        </a>
        <nav className="nav-links" aria-label="On this page">
          <a href="#top" aria-current={zone === "workstation" ? "true" : undefined}>
            {t.nav.workstation}
          </a>
          <a href="#video" aria-current={zone === "video" ? "true" : undefined}>
            {t.nav.video}
          </a>
          <a href="#model" aria-current={zone === "model" ? "true" : undefined}>
            {t.nav.model}
          </a>
          <a href="#trust" aria-current={zone === "trust" ? "true" : undefined}>
            {t.nav.trust}
          </a>
          <a href="#doors" aria-current={zone === "doors" ? "true" : undefined}>
            {t.nav.doors}
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
        {menu ? (
          <button type="button" className="nav-backdrop" aria-label={t.nav.close} onClick={closeMenu} />
        ) : null}
        <div className="nav-panel" id="nav-panel" hidden={!menu}>
          <nav aria-label="On this page">
            <a href="#top" onClick={closeMenu}>
              {t.nav.workstation}
            </a>
            <a href="#video" onClick={closeMenu}>
              {t.nav.video}
            </a>
            <a href="#model" onClick={closeMenu}>
              {t.nav.model}
            </a>
            <a href="#trust" onClick={closeMenu}>
              {t.nav.trust}
            </a>
            <a href="#doors" onClick={closeMenu}>
              {t.nav.doors}
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
        <section className="hero" id="top" data-zone="workstation">
          <div className="wrap hero-grid">
            <div className="hero-copy">
              <p className="kicker">
                yoyo · {version}
              </p>
              <Lines as="h1" text={t.hero.title} />
              <div className="hero-lower">
                <div className="hero-lead">
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
                <div className="hero-sculpture" aria-hidden="true">
                  <SpaceStill theme={theme} />
                </div>
              </div>
            </div>
          </div>
          <div className="wrap hero-stage">
              <div id="stage" role="tabpanel" aria-labelledby={`scene-${stage}`} aria-describedby="stage-note">
                <Workstation
                  t={t}
                  scene={stage}
                  onScene={setStage}
                  gate={gate}
                  onGate={setGate}
                  promoted={promoted}
                  onCheckout={checkout}
                  refKey={refKey}
                  onRef={setRefKey}
                  hunks={hunks}
                  onHunk={(index) => setHunks((prev) => (index === 0 ? [!prev[0], prev[1]] : [prev[0], !prev[1]]))}
                  episode={episode}
                  onEpisode={setEpisode}
                />
              </div>
              <div className="scene-tabs" role="tablist" aria-label={t.nav.workstation} onKeyDown={onTabKey}>
                {SCENES.map((id) => (
                  <button
                    key={id}
                    id={`scene-${id}`}
                    type="button"
                    role="tab"
                    aria-selected={stage === id}
                    tabIndex={stage === id ? 0 : -1}
                    onClick={() => setStage(id)}
                  >
                    {sceneLabel[id]}
                  </button>
                ))}
              </div>
              <p className="stage-note" id="stage-note">
                {caption}
                {stage === "drama" ? (
                  <a className="text-link" href="#video">
                    {t.nav.video}
                  </a>
                ) : null}
              </p>
          </div>
        </section>

        <section className="section section-stack wrap" id="work" data-zone="workstation">
          <header className="section-head">
            <p className="kicker">01</p>
            <Lines text={t.turn.title} />
            <p className="dek">{t.turn.dek}</p>
          </header>
          <ol className="steps">
            {t.turn.steps.map((step, index) => (
              <li key={step.title}>
                <span className="step-n">{String(index + 1).padStart(2, "0")}</span>
                <h3>{step.title}</h3>
                <p>{step.body}</p>
              </li>
            ))}
          </ol>
        </section>

        <section className="section section-stack wrap" id="ability" data-zone="workstation">
          <header className="section-head">
            <p className="kicker">02</p>
            <Lines text={t.pitch.title} />
            <p className="dek">{t.pitch.dek}</p>
          </header>
          <div className="ability-grid">
            {t.pitch.cards.map((card, index) => (
              <article key={card.title}>
                <span className="kicker">{String(index + 1).padStart(2, "0")}</span>
                <h3>{card.title}</h3>
                <p>{card.body}</p>
              </article>
            ))}
          </div>
        </section>

        <section className="section section-stack wrap" id="video" data-zone="video">
          <header className="section-head">
            <p className="kicker">03</p>
            <Lines text={t.film.title} />
            <p className="dek">{t.film.dek}</p>
          </header>
          <div className="film-grid">
            <Workstation
              t={t}
              scene="drama"
              embed
              onScene={() => undefined}
              videoTab={filmTab}
              onVideoTab={(tab) => setFilmMode(tab === "create" ? 0 : tab === "canvas" ? 2 : 1)}
              gate={gate}
              onGate={setGate}
              promoted={promoted}
              onCheckout={checkout}
              refKey={refKey}
              onRef={setRefKey}
              hunks={hunks}
              onHunk={(index) => setHunks((prev) => (index === 0 ? [!prev[0], prev[1]] : [prev[0], !prev[1]]))}
              episode={episode}
              onEpisode={setEpisode}
            />
            <div className="film-modes" role="tablist" aria-label={t.nav.video}>
              {t.film.modes.map((mode, index) => (
                <button key={mode.title} type="button" role="tab" aria-selected={filmMode === index} onClick={() => setFilmMode(index)}>
                  <b>{mode.title}</b>
                  <span>{mode.body}</span>
                </button>
              ))}
            </div>
          </div>
          <h3 className="phases-title">{t.film.phasesTitle}</h3>
          <ol className="phases">
            {t.film.phases.map((phase, index) => (
              <li key={phase.title}>
                <span className="step-n">{String(index + 1).padStart(2, "0")}</span>
                <h3>{phase.title}</h3>
                <p>{phase.body}</p>
              </li>
            ))}
          </ol>
          <p className="apart">{t.film.apart}</p>
        </section>

        <section className="section split wrap" id="model" data-zone="model">
          <div className="section-copy">
            <p className="kicker">04</p>
            <Lines text={t.model.title} />
            <p className="dek">{t.model.dek}</p>
          </div>
          <div className="section-body">
            <p className="weights">
              <b>{t.model.modelName}</b>
              <span>{t.model.weightsStill}</span>
            </p>
            <ul className="layers">
              {layers.map(([id, name, body]) => (
                <li key={id} className="layer" data-layer={id}>
                  <b>{name}</b>
                  <p>{body}</p>
                </li>
              ))}
            </ul>
            <p className="fp">
              <span>{t.model.fp}</span>
              <code>refs/models/&lt;fingerprint&gt;/active</code>
            </p>
            <p className="fp-body">{t.model.fpBody}</p>
            <p className="more">
              <a href={`${REPO}/blob/master/docs/architecture/invariants.md`}>{t.foot.invariants}</a>
            </p>
          </div>
        </section>

        <section className="section split wrap" id="trust" data-zone="trust">
          <div className="section-copy">
            <p className="kicker">05</p>
            <Lines id="trust-title" text={t.trust.title} />
            <p className="dek">{t.trust.dek}</p>
          </div>
          <div className="section-body">
            <div className="track" role="radiogroup" aria-labelledby="trust-title">
              {refs.map(([id, label]) => (
                <button key={id} type="button" role="radio" data-ref={id} aria-checked={refKey === id} onClick={() => setRefKey(id)}>
                  <b>{label}</b>
                  <i>{id}</i>
                </button>
              ))}
            </div>
            <p className="ref-note" aria-live="polite">
              {refNote}
            </p>
            <button type="button" className="btn btn-ink" disabled={promoted || refKey !== "canary"} onClick={checkout}>
              {t.trust.checkout}
            </button>
          </div>
        </section>

        <section className="section section-stack wrap" id="doors" data-zone="doors">
          <header className="section-head">
            <p className="kicker">06</p>
            <Lines text={t.doors.title} />
            <p className="dek">{t.doors.dek}</p>
          </header>
          <ol className="door-grid">
            <li>
              <h3>{t.doors.desktop}</h3>
              <p>{t.doors.desktopBody}</p>
            </li>
            <li>
              <h3>{t.doors.browser}</h3>
              <p>{t.doors.browserBody}</p>
              <code>yoyo serve</code>
            </li>
            <li>
              <h3>{t.doors.cli}</h3>
              <p>{t.doors.cliBody}</p>
              <code>yoyo run</code>
            </li>
          </ol>
          <div className="aside">
            <article>
              <h3>{t.doors.skills}</h3>
              <p>
                <InlineCode text={t.doors.skillsBody} />
              </p>
            </article>
            <article>
              <h3>{t.doors.budget}</h3>
              <p>{t.doors.budgetBody}</p>
            </article>
          </div>
        </section>

        <section className="section split wrap" id="limits" data-zone="doors">
          <div className="section-copy">
            <p className="kicker">07</p>
            <Lines text={t.limits.title} />
            <p className="dek">{t.limits.dek}</p>
          </div>
          <ul className="limits">
            {t.limits.items.map((item) => (
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
            <a href={`${REPO}/blob/master/docs/architecture/invariants.md`}>{t.foot.invariants}</a>
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

function Lines({ text, as: Tag = "h2", id }: { text: string; as?: "h1" | "h2"; id?: string }) {
  return (
    <Tag id={id}>
      {text.split("\n").map((line, index) => (
        <span className="ln" key={index}>
          {line}
        </span>
      ))}
    </Tag>
  );
}

function InlineCode({ text }: { text: string }) {
  return (
    <>
      {text.split("`").map((part, index) => (index % 2 ? <code key={index}>{part}</code> : <span key={index}>{part}</span>))}
    </>
  );
}

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

function DownloadIcon() {
  return (
    <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
      <path d="M8 2.2v7.2M5.2 7.2 8 10l2.8-2.8M3.2 13.2h9.6" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function SunIcon() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
      <circle cx="12" cy="12" r="3.5" fill="none" stroke="currentColor" strokeWidth="1.6" />
      <path d="M12 3v2.2M12 18.8V21M3 12h2.2M18.8 12H21M5.4 5.4l1.6 1.6M17 17l1.6 1.6M18.6 5.4 17 7M7 17l-1.6 1.6" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    </svg>
  );
}

function MoonIcon() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
      <path d="M15.2 3.4A7.4 7.4 0 1 0 20.6 14 5.8 5.8 0 0 1 15.2 3.4Z" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
    </svg>
  );
}

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
