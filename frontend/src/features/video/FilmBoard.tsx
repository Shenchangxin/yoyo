/**
 * 本文件拥有成片台的简介、画幅、预览和去剪辑。不拥有剪辑时间线。
 * 保存简介、做画面、换床、分叉和交接都会改服务端。
 */
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { playbackSrc } from "./playback-src";
import { FilmStudio } from "./FilmStudio";
import { castFromProject, castPayload, isFilmMode, sameCast, type FilmCast, type FilmMode } from "./film-cast";
import { useFilmDesk, type FilmProject, type FilmWord } from "./film-desk";
import { subscribeVideoEvents } from "./studio-events";

const shapes = ["9:16", "16:9", "1:1"] as const;

/** 按行把词拼起来，汉字之间不加空格，最多 8 行。words 按 line_id、id 或 text 分组。 */
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
    const parts = group.map(/** word 取出 text。 */ (word) => String(word.text || "")).filter(Boolean);
    if (!parts.length) continue;
    const han = parts.some(/** part 含汉字就整行不加空格。 */ (part) => /[\u4e00-\u9fff]/.test(part));
    lines.push(parts.join(han ? "" : " "));
  }
  return lines.slice(0, 8);
}

/** FilmBoard 在排行榜、对谈和街采上改那几行字；其他结构仍改主持人和产品。有画面时去剪辑，否则做画面。props 的 onOpenCut 在交接出剪辑 id 后打开。 */
export function FilmBoard(props: { onOpenCut: (id: string) => void }) {
  const copy = useCopy();
  const c = copy.video.film;
  const back = copy.video.conductor.backHome;
  const project = useFilmDesk(/** s 取出 project。 */ (s) => s.project);
  const setProject = useFilmDesk(/** s 取出 setProject。 */ (s) => s.setProject);
  const clear = useFilmDesk(/** s 取出 clear。 */ (s) => s.clear);
  const [host, setHost] = useState("");
  const [product, setProduct] = useState("");
  const [opening, setOpening] = useState("");
  const [ending, setEnding] = useState("");
  const [price, setPrice] = useState("");
  const [busy, setBusy] = useState("");
  const [mediaBase, setMediaBase] = useState("");
  const [local, setLocal] = useState<FilmCast | null>(null);
  // 换了画幅后旧画面就不对了，要重新做；同步锁挡住连点两下打出两次请求。
  const [aspectStale, setAspectStale] = useState(false);
  const running = useRef(false);

  useEffect(() => {
    let alive = true;
    void api.video.status().then(/** st 取出 media_base 并去掉结尾斜杠。 */ (st) => {
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

  useEffect(() => {
    setLocal(null);
    setAspectStale(false);
  }, [project?.id]);

  useEffect(() => {
    /** Escape 和「回到创作」同一条路。没有打开成片时不动。 */
    const onLeave = () => {
      if (useFilmDesk.getState().project?.id) clear();
    };
    window.addEventListener("yoyo:leave-create-overlay", onLeave);
    return () => window.removeEventListener("yoyo:leave-create-overlay", onLeave);
  }, [clear]);

  const building = !!project?.building;
  useEffect(() => {
    // 做画面在后台跑：服务端做完推 film_state，收到就重读；事件丢了也每 10 秒兜底重读一次。
    const id = project?.id;
    if (!id || !building) return;
    const refresh = () => void api.video.filmGet(id).then(/** row 当成工程换上。 */ (row) => {
      const next = row as FilmProject;
      if (useFilmDesk.getState().project?.id === id) setProject(next);
      if (!next?.building && next?.note) toast.message(next.note);
    }).catch(() => {});
    const off = subscribeVideoEvents(["film_state"], /** pushed 是推过来的片子编号，是这一部才重读。 */ (_kind, pushed) => {
      if (pushed === id) refresh();
    });
    const t = window.setInterval(refresh, 10_000);
    return () => {
      off();
      window.clearInterval(t);
    };
  }, [project?.id, building, setProject]);

  if (!project || !project.id) return null;
  const id = project.id;
  const words = project.words || [];
  const shots = project.shots || [];
  const lines = scriptLines(words);
  const failed = (project.needs || []).find(/** need 有 error 才留下。 */ (need) => need.error);
  const artifacts = project.artifacts || [];
  const picture = artifacts.find(/** a 的 media_type 以 video/ 开头才留下。 */ (a) => a.media_type?.startsWith("video/"))
    || artifacts.find(/** a 的 media_type 以 image/ 开头才留下。 */ (a) => a.media_type?.startsWith("image/"));
  const pictureSrc = picture?.hash && mediaBase ? playbackSrc(`${mediaBase}/media/${picture.hash}`) : "";
  const aspect = project.brief?.aspect || "9:16";
  const shapeLabel = aspect === "16:9" ? c.landscape : aspect === "1:1" ? c.square : c.portrait;
  const structure = project.structure === "ranking" ? c.ranking : project.structure === "podcast" ? c.podcast : project.structure === "interview" ? c.interview : c.title;

  /** 跑一次成片请求，成功就换上返回的工程。已经有一个在跑就丢掉这次点击。label 写成忙碌标记。fn 执行，返回带 id 的工程就换上。 */
  async function run(label: string, fn: () => Promise<FilmProject>) {
    if (running.current) return project;
    running.current = true;
    setBusy(label);
    try {
      const next = await fn();
      if (next?.id) setProject(next);
      if (label === "build") setAspectStale(false);
      return next;
    } catch (e) {
      toast.error(api.errMessage(e));
      return project;
    } finally {
      running.current = false;
      setBusy("");
    }
  }

  /** 简介字段没变就不提交。改了画幅时旧画面作废，要重新做。label 交给 run。body 和当前简介逐项比较，没变就不提交，变了就交给 filmSwap。 */
  function patch(label: string, body: Record<string, string>) {
    const current = project?.brief || {};
    const same = Object.entries(body).every(/** key 用来读当前简介。value 和现有值比较。 */ ([key, value]) => String((current as Record<string, string | undefined>)[key] || "") === value);
    if (same) return;
    if ("aspect" in body && picture) setAspectStale(true);
    void run(label, async () => api.video.filmSwap(id, body) as Promise<FilmProject>);
  }

  /** 记下主持人和产品后，把成片交成一条剪辑并打开它。成片台的工程留着，从剪辑台回来还在。 */
  async function openCut() {
    if (running.current) return;
    running.current = true;
    setBusy("handoff");
    try {
      await api.video.filmSwap(id, { host, product });
      const cut = await api.video.filmHandoff(id, project?.title || "");
      const next = cut as { id?: string };
      if (!next?.id) throw new Error(c.needPicture);
      props.onOpenCut(next.id);
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      running.current = false;
      setBusy("");
    }
  }

  const format: FilmMode | null = project.structure && isFilmMode(project.structure) ? project.structure : null;
  const base = format ? castFromProject(project) : null;
  const cast = local ?? base;
  const dirty = !!cast && !!base && !sameCast(cast, base);
  const rebuild = !pictureSrc || dirty || aspectStale;
  const needTitle: Record<string, string> = { image: c.needImage, speech: c.needSpeech, video: c.needVideo, matting: c.needMatting, music: c.needMusic };

  /** 画面已是最新就去剪。否则先把字写回，再做出画面。 */
  async function commitFormat() {
    if (!cast) return;
    if (!rebuild) {
      await openCut();
      return;
    }
    const shaped = await run("build", async () => {
      const next = await api.video.filmShape(id, castPayload(cast)) as FilmProject;
      if (next?.id) setProject(next);
      return api.video.filmBuild(id) as Promise<FilmProject>;
    });
    if (shaped && shaped !== project) setLocal(null);
  }

  if (format && cast) {
    return (
      <section className="creation-remake film-open" data-testid="film-board" aria-label={c.title}>
        <button type="button" className="cut-back" onClick={() => clear()}>{back}</button>
        <FilmStudio
          mode={format}
          value={cast}
          busy={!!busy || building}
          commitLabel={building ? c.working : rebuild ? c.makePicture : c.goEdit}
          onChange={setLocal}
          onCommit={() => { void (rebuild ? commitFormat() : openCut()); }}
          failed={failed?.error ? `${c.failedStep}：${failed.error}` : !building && project.note ? project.note : ""}
          picture={pictureSrc ? { src: pictureSrc, video: !!picture?.media_type?.startsWith("video/"), title: project.title || c.preview } : undefined}
          needs={(project.needs || []).map(/** need 收成任务标题，种类换成中文叫法。 */ (need) => ({
            title: needTitle[need.kind || ""] || c.plan,
            detail: need.error || need.prompt,
          }))}
        />
        <details className="creation-more">
          <summary>{c.quiet}</summary>
          <span className="creation-field-label">{c.shape}</span>
          <div className="creation-shape" role="group" aria-label={c.shape}>
            {shapes.map(/** value 交给 patch 当画幅，并用来判断选中和文案。 */ (value) => (
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
          <div className="creation-remake-actions">
            <Button size="sm" variant="ghost" disabled={!!busy} onClick={() => {
              setBusy("bed");
              void api.pickFiles().then(/** picked 取第一项路径，登记成音频后换床。 */ async (picked) => {
                const path = picked[0];
                if (!path) return project;
                const ref = await api.video.putStudioMediaPath(path, "audio");
                const hash = String(ref?.cas_hash || ref?.hash || "");
                if (!hash) throw new Error(c.needPicture);
                return api.video.filmBed(id, hash) as Promise<FilmProject>;
              }).then(/** next 有 id 才换上工程。 */ (next) => {
                if (next?.id) setProject(next);
              }).catch(/** e 交给 errMessage 后弹出。 */ (e) => toast.error(api.errMessage(e))).finally(() => setBusy(""));
            }}>{c.bed}</Button>
            <Button size="sm" variant="ghost" disabled={!!busy || !cast.subject.trim()} onClick={() => void run("fork", async () => api.video.filmFork(id, { host: cast.subject }) as Promise<FilmProject>)}>{c.another}</Button>
          </div>
        </details>
      </section>
    );
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
      ) : <p>{building ? c.working : c.previewEmpty}</p>}
      {failed?.error ? <p>{c.failedStep}：{failed.error}</p> : null}
      {!building && project.note ? <p data-testid="film-note">{project.note}</p> : null}
      <label>
        <span>{c.who}</span>
        <input value={host} onChange={/** ev 取出 target.value 写进主持人。 */ (ev) => setHost(ev.target.value)} />
      </label>
      <label>
        <span>{c.what}</span>
        <input value={product} onChange={/** ev 取出 target.value 写进产品。 */ (ev) => setProduct(ev.target.value)} />
      </label>
      <div className="creation-remake-actions">
        {pictureSrc && !building && !aspectStale ? (
          <Button size="sm" variant="default" disabled={!!busy} onClick={() => void openCut()}>{c.goEdit}</Button>
        ) : (
          <Button size="sm" variant="default" disabled={!!busy || building} onClick={() => void run("build", async () => {
            await api.video.filmSwap(id, { host, product });
            return api.video.filmBuild(id) as Promise<FilmProject>;
          })}>{busy === "build" || building ? c.working : c.makePicture}</Button>
        )}
      </div>
      <details className="creation-more">
        <summary>{c.details}</summary>
        <span className="creation-field-label">{c.shape}</span>
        <div className="creation-shape" role="group" aria-label={c.shape}>
          {shapes.map(/** value 交给 patch 当画幅，并用来判断选中和文案。 */ (value) => (
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
          <input value={opening} onChange={/** ev 取出 target.value 写进开头。 */ (ev) => setOpening(ev.target.value)} onBlur={() => patch("hook", { hook: opening.trim() })} />
        </label>
        <label>
          <span>{c.ending}</span>
          <input value={ending} onChange={/** ev 取出 target.value 写进结尾。 */ (ev) => setEnding(ev.target.value)} onBlur={() => patch("cta", { cta: ending.trim() })} />
        </label>
        {project.structure === "ranking" ? (
          <label>
            <span>{c.price}</span>
            <input value={price} onChange={/** ev 取出 target.value 写进价格。 */ (ev) => setPrice(ev.target.value)} onBlur={() => patch("price", { price: price.trim() })} />
          </label>
        ) : null}
        {lines.length ? (
          <>
            <span className="creation-field-label">{c.script}</span>
            <ol className="creation-film-line">
              {lines.map(/** line 显示成一行。 */ (line) => <li key={line}><strong>{line}</strong></li>)}
            </ol>
          </>
        ) : null}
        <div className="creation-remake-actions">
          <Button size="sm" variant="ghost" disabled={!!busy} onClick={() => {
            setBusy("bed");
            void api.pickFiles().then(/** picked 取第一项路径，登记成音频后换床。 */ async (picked) => {
              const path = picked[0];
              if (!path) return project;
              const ref = await api.video.putStudioMediaPath(path, "audio");
              const hash = String(ref?.cas_hash || ref?.hash || "");
              if (!hash) throw new Error(c.needPicture);
              return api.video.filmBed(id, hash) as Promise<FilmProject>;
            }).then(/** next 有 id 才换上工程。 */ (next) => {
              if (next?.id) setProject(next);
            }).catch(/** e 交给 errMessage 后弹出。 */ (e) => toast.error(api.errMessage(e))).finally(() => setBusy(""));
          }}>{c.bed}</Button>
          <Button size="sm" variant="ghost" disabled={!!busy || !host.trim()} onClick={() => void run("fork", async () => api.video.filmFork(id, { host }) as Promise<FilmProject>)}>{c.another}</Button>
        </div>
      </details>
    </section>
  );
}
