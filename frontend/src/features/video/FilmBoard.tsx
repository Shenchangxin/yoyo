import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { playbackSrc } from "./playback-src";
import { useFilmDesk, type FilmProject, type FilmWord } from "./film-desk";

const shapes = ["9:16", "16:9", "1:1"] as const;

function scriptLines(words: FilmWord[]) {
  const groups = new Map<string, FilmWord[]>();
  for (const word of words) {
    const key = word.line_id || word.id || word.text || "";
    if (!key) continue;
    const group = groups.get(key) || [];
    group.push(word);
    groups.set(key, group);
  }
  const lines: string[] = [];
  for (const group of groups.values()) {
    const parts = group.map((word) => String(word.text || "")).filter(Boolean);
    if (!parts.length) continue;
    const han = parts.some((part) => /[\u4e00-\u9fff]/.test(part));
    lines.push(parts.join(han ? "" : " "));
  }
  return lines.slice(0, 8);
}

export function FilmBoard(props: { onOpenCut: (id: string) => void }) {
  const copy = useCopy();
  const c = copy.video.film;
  const back = copy.video.conductor.backHome;
  const project = useFilmDesk((s) => s.project);
  const setProject = useFilmDesk((s) => s.setProject);
  const clear = useFilmDesk((s) => s.clear);
  const [host, setHost] = useState("");
  const [product, setProduct] = useState("");
  const [opening, setOpening] = useState("");
  const [ending, setEnding] = useState("");
  const [price, setPrice] = useState("");
  const [busy, setBusy] = useState("");
  const [mediaBase, setMediaBase] = useState("");

  useEffect(() => {
    let alive = true;
    void api.video.status().then((st) => {
      if (alive) setMediaBase(String(st?.media_base || "").replace(/\/+$/, ""));
    }).catch(() => {});
    return () => { alive = false; };
  }, []);

  useEffect(() => {
    setHost(project?.brief?.host || "");
    setProduct(project?.brief?.product || "");
    setOpening(project?.brief?.hook || "");
    setEnding(project?.brief?.cta || "");
    setPrice(project?.brief?.price || "");
  }, [project?.id, project?.brief?.host, project?.brief?.product, project?.brief?.hook, project?.brief?.cta, project?.brief?.price]);

  if (!project || !project.id) return null;
  const id = project.id;
  const words = project.words || [];
  const shots = project.shots || [];
  const lines = scriptLines(words);
  const failed = (project.needs || []).find((need) => need.error);
  const artifacts = project.artifacts || [];
  const picture = artifacts.find((a) => a.media_type?.startsWith("video/"))
    || artifacts.find((a) => a.media_type?.startsWith("image/"));
  const pictureSrc = picture?.hash && mediaBase ? playbackSrc(`${mediaBase}/media/${picture.hash}`) : "";
  const aspect = project.brief?.aspect || "9:16";
  const shapeLabel = aspect === "16:9" ? c.landscape : aspect === "1:1" ? c.square : c.portrait;
  const structure = project.structure === "ranking" ? c.ranking : project.structure === "podcast" ? c.podcast : project.structure === "interview" ? c.interview : c.title;

  async function run(label: string, fn: () => Promise<FilmProject>) {
    setBusy(label);
    try {
      const next = await fn();
      if (next?.id) setProject(next);
      return next;
    } catch (e) {
      toast.error(api.errMessage(e));
      return project;
    } finally {
      setBusy("");
    }
  }

  function patch(label: string, body: Record<string, string>) {
    const current = project?.brief || {};
    const same = Object.entries(body).every(([key, value]) => String((current as Record<string, string | undefined>)[key] || "") === value);
    if (same) return;
    void run(label, async () => api.video.filmSwap(id, body) as Promise<FilmProject>);
  }

  async function openCut() {
    setBusy("handoff");
    try {
      await api.video.filmSwap(id, { host, product });
      const cut = await api.video.filmHandoff(id, project?.title || "");
      const next = cut as { id?: string };
      if (!next?.id) throw new Error(c.needPicture);
      clear();
      props.onOpenCut(next.id);
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  return (
    <section className="creation-remake" data-testid="film-board" aria-label={c.title}>
      <button type="button" className="cut-back" onClick={() => clear()}>{back}</button>
      <header>
        <strong>{project.title || c.title}</strong>
        <small>
          {c.stats
            .replace("{structure}", structure)
            .replace("{words}", String(lines.length || words.length))
            .replace("{shots}", String(shots.length))}
          {" · "}
          {shapeLabel}
        </small>
      </header>
      {pictureSrc ? (
        picture?.media_type?.startsWith("video/")
          ? <video className="creation-film-preview" src={pictureSrc} controls />
          : <img className="creation-film-preview" src={pictureSrc} alt={project.title || c.preview} />
      ) : <p>{c.previewEmpty}</p>}
      {failed?.error ? <p>{c.failedStep}：{failed.error}</p> : null}
      <label>
        <span>{c.who}</span>
        <input value={host} onChange={(ev) => setHost(ev.target.value)} />
      </label>
      <label>
        <span>{c.what}</span>
        <input value={product} onChange={(ev) => setProduct(ev.target.value)} />
      </label>
      <div className="creation-remake-actions">
        {pictureSrc ? (
          <Button size="sm" variant="lift" disabled={!!busy} onClick={() => void openCut()}>{c.goEdit}</Button>
        ) : (
          <Button size="sm" variant="lift" disabled={!!busy} onClick={() => void run("build", async () => {
            await api.video.filmSwap(id, { host, product });
            return api.video.filmBuild(id) as Promise<FilmProject>;
          })}>{busy === "build" ? c.working : c.makePicture}</Button>
        )}
      </div>
      <details className="creation-more">
        <summary>{c.details}</summary>
        <span className="creation-field-label">{c.shape}</span>
        <div className="creation-shape" role="group" aria-label={c.shape}>
          {shapes.map((value) => (
            <button
              key={value}
              type="button"
              aria-pressed={(project.brief?.aspect || "9:16") === value}
              disabled={!!busy}
              onClick={() => patch("aspect", { aspect: value })}
            >
              {value === "16:9" ? c.landscape : value === "1:1" ? c.square : c.portrait}
            </button>
          ))}
        </div>
        <label>
          <span>{c.opening}</span>
          <input value={opening} onChange={(ev) => setOpening(ev.target.value)} onBlur={() => patch("hook", { hook: opening.trim() })} />
        </label>
        <label>
          <span>{c.ending}</span>
          <input value={ending} onChange={(ev) => setEnding(ev.target.value)} onBlur={() => patch("cta", { cta: ending.trim() })} />
        </label>
        {project.structure === "ranking" ? (
          <label>
            <span>{c.price}</span>
            <input value={price} onChange={(ev) => setPrice(ev.target.value)} onBlur={() => patch("price", { price: price.trim() })} />
          </label>
        ) : null}
        {lines.length ? (
          <>
            <span className="creation-field-label">{c.script}</span>
            <ol className="creation-film-line">
              {lines.map((line) => <li key={line}><strong>{line}</strong></li>)}
            </ol>
          </>
        ) : null}
        <div className="creation-remake-actions">
          <Button size="sm" variant="ghost" disabled={!!busy} onClick={() => {
            setBusy("bed");
            void api.pickFiles().then(async (picked) => {
              const path = picked[0];
              if (!path) return project;
              const ref = await api.video.putStudioMediaPath(path, "audio");
              const hash = String(ref?.cas_hash || ref?.hash || "");
              if (!hash) throw new Error(c.needPicture);
              return api.video.filmBed(id, hash) as Promise<FilmProject>;
            }).then((next) => {
              if (next?.id) setProject(next);
            }).catch((e) => toast.error(api.errMessage(e))).finally(() => setBusy(""));
          }}>{c.myVoice}</Button>
          <Button size="sm" variant="ghost" disabled={!!busy || !host.trim()} onClick={() => void run("fork", async () => api.video.filmFork(id, { host }) as Promise<FilmProject>)}>{c.another}</Button>
        </div>
      </details>
    </section>
  );
}
