import { lazy, Suspense, useEffect, useMemo, useState } from "react";
import {
  COPY,
  detectLang,
  htmlLang,
  type GateKind,
  type Lang,
} from "./copy";
import { MaskText, Rise } from "./motion";
import { Workstation } from "./workstation";

const StoneField = lazy(() => import("./stone-field").then((mod) => ({ default: mod.StoneField })));
import { detectOs, FALLBACK_VERSION, fallbackAssets, hydrateRelease, RELEASES, type Asset } from "./release";

type RefKey = "staging" | "canary" | "active";
type SeeBeat = "pin" | "diff" | "drama";
type Stage = "ask" | "diff" | "drama" | "trust";

export function App() {
  const [lang, setLang] = useState<Lang>(() => detectLang());
  const t = COPY[lang];
  const [zone, setZone] = useState("hero");
  const [beat, setBeat] = useState<SeeBeat>("diff");
  const [stage, setStage] = useState<Stage>("ask");
  const [gate, setGate] = useState<GateKind>("once");
  const [promoted, setPromoted] = useState(false);
  const [refKey, setRefKey] = useState<RefKey>("canary");
  const [hunks, setHunks] = useState<[boolean, boolean]>([true, true]);
  const [episode, setEpisode] = useState("02");
  const [version, setVersion] = useState(FALLBACK_VERSION);
  const [assets, setAssets] = useState<Asset[]>(() => fallbackAssets(FALLBACK_VERSION));
  const [os, setOs] = useState<Asset["os"]>("windows");
  const preferred = useMemo(() => assets.find((a) => a.os === os) || assets[0], [assets, os]);

  useEffect(() => {
    document.documentElement.dataset.lang = lang;
    document.documentElement.lang = htmlLang(lang);
    document.title = `${t.meta.title} ${version}`;
    const meta = document.querySelector('meta[name="description"]');
    if (meta) meta.setAttribute("content", t.meta.description);
    try {
      localStorage.setItem("yoyo-site-lang", lang);
    } catch {
      /* ignore */
    }
  }, [lang, t.meta.description, t.meta.title, version]);

  useEffect(() => {
    setOs(detectOs());
    void hydrateRelease().then((data) => {
      setVersion(data.version);
      setAssets(data.assets);
    });
  }, []);

  useEffect(() => {
    const root = document.documentElement;
    let frame = 0;
    const write = () => {
      const max = root.scrollHeight - window.innerHeight;
      root.style.setProperty("--scroll", max > 0 ? String(window.scrollY / max) : "0");
    };
    const onScroll = () => {
      window.cancelAnimationFrame(frame);
      frame = window.requestAnimationFrame(write);
    };
    write();
    window.addEventListener("scroll", onScroll, { passive: true });
    window.addEventListener("resize", onScroll);
    return () => {
      window.cancelAnimationFrame(frame);
      window.removeEventListener("scroll", onScroll);
      window.removeEventListener("resize", onScroll);
    };
  }, []);

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
          .filter((e) => e.isIntersecting)
          .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0];
        if (hit?.target instanceof HTMLElement && hit.target.dataset.zone) {
          setZone(hit.target.dataset.zone);
        }
      },
      { rootMargin: "-28% 0px -42% 0px", threshold: [0.2, 0.5, 0.75] },
    );
    nodes.forEach((n) => io.observe(n));
    return () => io.disconnect();
  }, []);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.metaKey || event.ctrlKey || event.altKey) return;
      const el = event.target as HTMLElement | null;
      if (el?.closest("input, textarea, select")) return;
      const next = ({ "1": "once", "2": "session", "3": "always", Escape: "deny" } as const)[event.key];
      if (next) setGate(next);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  useEffect(() => {
    const nodes = [...document.querySelectorAll<HTMLElement>("[data-reveal]")];
    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (reduce) {
      nodes.forEach((node) => node.classList.add("is-in"));
      return;
    }
    const io = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) entry.target.classList.add("is-in");
        });
      },
      { threshold: 0.16 },
    );
    nodes.forEach((node) => io.observe(node));
    return () => io.disconnect();
  }, [lang]);

  const navCurrent =
    zone === "trust" ? "trust" : zone === "model" ? "model" : zone === "get" || zone === "doors" ? "download" : "workstation";

  return (
    <>
      <Suspense fallback={null}>
        <StoneField zone={zone} />
      </Suspense>
      <div className="scroll-line" aria-hidden="true" />
      <a className="skip" href="#download">
        {t.nav.download}
      </a>
      <div className="grain" aria-hidden="true" />
      <header className="nav is-float">
        <a className="mark" href="#top">
          yoyo
        </a>
        <div className="nav-end">
          <nav className="nav-toc" aria-label="On this page">
            <a href="#top" aria-current={navCurrent === "workstation" ? "true" : undefined}>
              {t.nav.workstation}
            </a>
            <a href="#model" aria-current={navCurrent === "model" ? "true" : undefined}>
              {t.nav.model}
            </a>
            <a href="#trust" aria-current={navCurrent === "trust" ? "true" : undefined}>
              {t.nav.trust}
            </a>
            <a href="#download" aria-current={navCurrent === "download" ? "true" : undefined}>
              {t.nav.download}
            </a>
          </nav>
          <nav className="lang" aria-label="Language">
            {(["en", "zh", "ja"] as const).map((item) => (
              <button
                key={item}
                type="button"
                aria-label={item === "en" ? "English" : item === "zh" ? "简体中文" : "日本語"}
                aria-pressed={lang === item}
                onClick={() => setLang(item)}
              >
                {item === "en" ? "EN" : item === "zh" ? "中" : "日"}
              </button>
            ))}
          </nav>
          <a href="https://github.com/Shenchangxin/yoyo">GitHub</a>
          <a className="btn-ink" href="#download">
            {t.nav.download}
          </a>
        </div>
      </header>

      <main>
        <div className="hero-wrap" id="top" data-zone="hero">
        <section className="hero">
          <div className="hero-copy" data-reveal>
            <p className="kicker">yoyo</p>
            <MaskText as="h1" text={t.hero.title} />
            <Rise delay={180}>
              <p className="dek">{t.hero.dek}</p>
            </Rise>
            <Rise delay={280} className="hero-actions">
              <a className="btn btn-ink" href={preferred?.href || RELEASES}>
                <span>
                  {t.hero.download} {version}
                </span>
                <span className="btn-go" aria-hidden="true">
                  <svg viewBox="0 0 16 16" width="12" height="12">
                    <path d="M3.2 8h8.2M8.4 4.2 12.2 8 8.4 11.8" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                </span>
              </a>
              <a className="btn btn-ghost" href="https://github.com/Shenchangxin/yoyo">
                {t.hero.github}
              </a>
            </Rise>
          </div>
          <div className="hero-stage" data-reveal>
            <p className="live-line">
              <i />
              {stage === "ask" ? t.demo.threadLive : stage === "diff" ? t.see.diff : stage === "drama" ? t.demo.shotCap : t.trust.canary}
            </p>
            <Rise delay={120} className="bezel">
              <Workstation
                t={t}
                scene={stage}
                gate={gate}
                onGate={setGate}
                promoted={promoted}
                onCheckout={() => {
                  setPromoted(true);
                  setRefKey("active");
                }}
                refKey={refKey}
                onRef={setRefKey}
                hunks={hunks}
                onHunk={(i) => setHunks((prev) => (i === 0 ? [!prev[0], prev[1]] : [prev[0], !prev[1]]))}
                episode={episode}
                onEpisode={setEpisode}
              />
            </Rise>
          </div>
        </section>

        <div className="scene-picks" data-reveal role="tablist" aria-label={t.nav.workstation}>
          {(
            [
              ["ask", t.demo.agent],
              ["diff", t.demo.review],
              ["drama", t.demo.video],
              ["trust", t.nav.trust],
            ] as const
          ).map(([id, label]) => (
            <button key={id} type="button" role="tab" aria-selected={stage === id} className={stage === id ? "is-on" : undefined} onClick={() => setStage(id)}>
              {label}
            </button>
          ))}
        </div>
        </div>

        <ul className="spec">
          {t.hero.chips.map((chip) => (
            <li key={chip}>{chip}</li>
          ))}
        </ul>

        <section className="chapter" id="model" data-zone="model" data-reveal>
          <div className="chapter-copy">
            <MaskText text={t.model.title} />
            <Rise delay={140}>
              <p className="dek">{t.model.dek}</p>
            </Rise>
            <ul className="facts">
              <li>
                <b>{t.model.solver}</b>
                <span>{t.model.solverBody}</span>
              </li>
              <li>
                <b>{t.model.shape}</b>
                <span>{t.model.shapeBody}</span>
              </li>
              <li>
                <b>{t.model.evolve}</b>
                <span>{t.model.evolveBody}</span>
              </li>
              <li>
                <b>{t.model.fp}</b>
                <span>{t.model.fpBody}</span>
              </li>
            </ul>
          </div>
          <div className="bezel">
            <Workstation
              t={t}
              scene="model"
              gate={gate}
              onGate={setGate}
              promoted={promoted}
              onCheckout={() => {
                setPromoted(true);
                setRefKey("active");
              }}
              refKey={refKey}
              onRef={setRefKey}
              hunks={hunks}
              onHunk={(i) => setHunks((prev) => (i === 0 ? [!prev[0], prev[1]] : [prev[0], !prev[1]]))}
              episode={episode}
              onEpisode={setEpisode}
            />
          </div>
        </section>

        <figure className="still" data-reveal>
          <img src={`${import.meta.env.BASE_URL}desk-folder.jpg`} alt={t.see.folderAlt} />
          <figcaption>
            <b>{t.see.pin}</b>
            <span>{t.see.pinBody}</span>
          </figcaption>
        </figure>

        <section className="chapter chapter-see" id="see" data-zone="see" data-reveal>
          <div className="bezel">
            <Workstation
              t={t}
              scene={beat === "drama" ? "drama" : "diff"}
              gate={gate}
              onGate={setGate}
              promoted={promoted}
              onCheckout={() => {
                setPromoted(true);
                setRefKey("active");
              }}
              refKey={refKey}
              onRef={setRefKey}
              hunks={hunks}
              onHunk={(i) => setHunks((prev) => (i === 0 ? [!prev[0], prev[1]] : [prev[0], !prev[1]]))}
              episode={episode}
              onEpisode={setEpisode}
            />
          </div>
          <div className="chapter-copy">
            <MaskText text={t.see.title} />
            <Rise delay={140}>
              <p className="dek">{t.see.dek}</p>
            </Rise>
            <ul className="facts">
              <li className={beat === "diff" ? "is-on" : undefined}>
                <button type="button" onClick={() => setBeat("diff")}>
                  <b>{t.see.diff}</b>
                  <span>{t.see.diffBody}</span>
                </button>
              </li>
              <li className={beat === "drama" ? "is-on" : undefined}>
                <button type="button" onClick={() => setBeat("drama")}>
                  <b>{t.see.drama}</b>
                  <span>{t.see.dramaBody}</span>
                </button>
              </li>
            </ul>
          </div>
        </section>

        <section className="chapter" id="trust" data-zone="trust" data-reveal>
          <div className="chapter-copy">
            <MaskText text={t.trust.title} />
            <Rise delay={140}>
              <p className="dek">{t.trust.dek}</p>
            </Rise>
          </div>
          <div className="bezel">
            <Workstation
              t={t}
              scene="trust"
              gate={gate}
              onGate={setGate}
              promoted={promoted}
              onCheckout={() => {
                setPromoted(true);
                setRefKey("active");
              }}
              refKey={refKey}
              onRef={setRefKey}
              hunks={hunks}
              onHunk={(i) => setHunks((prev) => (i === 0 ? [!prev[0], prev[1]] : [prev[0], !prev[1]]))}
              episode={episode}
              onEpisode={setEpisode}
            />
          </div>
        </section>

        <section className="ability" id="ability" data-zone="doors" data-reveal>
          <div className="ability-head">
            <MaskText text={t.pitch.title} />
            <Rise delay={120}>
              <p className="dek">{t.pitch.dek}</p>
            </Rise>
          </div>
          <div className="ability-grid">
            {t.pitch.cards.map((card, index) => (
              <article key={card.title} className={index === 3 ? "ability-still" : undefined}>
                {index === 3 ? <img src={`${import.meta.env.BASE_URL}lamp-dusk.jpg`} alt={t.demo.shotCap} /> : null}
                <h3>{card.title}</h3>
                <p>{card.body}</p>
              </article>
            ))}
          </div>
        </section>

        <section className="band" id="doors" data-zone="doors" data-reveal>
          <div className="band-wide">
            <MaskText text={t.doors.title} />
            <Rise delay={120}>
              <p className="dek">{t.doors.dek}</p>
            </Rise>
            <ol className="door-list">
              <li>
                <b>{t.doors.desktop}</b>
                <span>{t.doors.desktopBody}</span>
              </li>
              <li>
                <b>{t.doors.browser}</b>
                <span>{t.doors.browserBody}</span>
                <code>yoyo serve</code>
              </li>
              <li>
                <b>{t.doors.cli}</b>
                <span>{t.doors.cliBody}</span>
                <code>yoyo run · eval · evolve</code>
              </li>
            </ol>
            <div className="aside">
              <div>
                <h3>{t.doors.skills}</h3>
                <p>
                  {t.doors.skillsBody.split("`").map((part, i) => (i % 2 ? <span className="code" key={i}>{part}</span> : <span key={i}>{part}</span>))}
                </p>
              </div>
              <div>
                <h3>{t.doors.budget}</h3>
                <p>{t.doors.budgetBody}</p>
                <div className="meter" aria-hidden="true">
                  <i />
                </div>
              </div>
            </div>
          </div>
        </section>

        <section className="band get" id="download" data-zone="get" data-reveal>
          <div className="band-narrow">
            <p className="kicker">
              {t.get.version} {version}
            </p>
            <MaskText text={t.get.title} />
            <Rise delay={120}>
              <p className="dek">{t.get.dek}</p>
            </Rise>
            <p className="premise">{t.get.premise}</p>
            <ol className="start">
              {t.get.steps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
            <div className="downloads">
              {(assets.length ? assets : []).map((asset) => (
                <a key={asset.os} data-os={asset.os} className={asset.os === os ? "is-you" : undefined} href={asset.href}>
                  <b>{asset.label}</b>
                  <span className="file">{asset.name}</span>
                  <small className="you">{t.get.thisMachine}</small>
                  <small className="size">{asset.size}</small>
                </a>
              ))}
            </div>
            <p className="note">{t.get.note}</p>
          </div>
        </section>
      </main>

      <footer className="foot">
        <a className="mark" href="#top">
          yoyo
        </a>
        <nav>
          <a href="https://github.com/Shenchangxin/yoyo">GitHub</a>
          <a href={RELEASES}>Releases</a>
          <a href="https://github.com/Shenchangxin/yoyo/blob/master/docs/PRODUCT.md">{t.foot.product}</a>
          <a href="https://github.com/Shenchangxin/yoyo/blob/master/docs/architecture/invariants.md">{t.foot.invariants}</a>
          <a href="https://github.com/Shenchangxin/yoyo/issues">{t.foot.issues}</a>
          <a href="https://github.com/Shenchangxin/yoyo/blob/master/LICENSE">{t.foot.license}</a>
        </nav>
        <span>© 2026</span>
      </footer>
    </>
  );
}
