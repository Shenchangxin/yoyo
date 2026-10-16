import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { Clapperboard, Mic, Scissors, Sparkles, PanelsTopLeft } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { useUI } from "../../lib/store";

type CutClip = {
  id?: string;
  hash?: string;
  start?: number;
  end?: number;
  title?: string;
  scores?: Record<string, number>;
  media_url?: string;
};

type CutProject = {
  id: string;
  title?: string;
  source_hash?: string;
  result_hash?: string;
  duration?: number;
  media_url?: string;
  source_url?: string;
  result_url?: string;
  poster_url?: string;
  breakdown?: { scenes?: number[]; silences?: { start: number; end: number }[]; clips?: CutClip[] };
  stems?: { original?: string; voice?: string; dub?: string; music?: string };
  captions?: { cues?: { start: number; end: number; text: string }[]; srt_hash?: string };
  clips?: CutClip[];
};

export function FootageDesk(props: {
  sessionId?: string;
  sourceHash?: string;
  compact?: boolean;
  onNeedSession?: () => void | Promise<string | void>;
}) {
  const copy = useCopy();
  const c = copy.video.conductor;
  const openVideoPane = useUI((s) => s.openVideoPane);
  const setFootageOpen = useUI((s) => s.setFootageOpen);
  const [proj, setProj] = useState<CutProject | null>(null);
  const [busy, setBusy] = useState("");
  const fileRef = useRef<HTMLInputElement>(null);

  const load = useCallback(async (id?: string) => {
    try {
      if (id) {
        const p = await api.video.cutGet(id) as CutProject;
        setProj(p);
        return p;
      }
      if (props.sessionId) {
        const p = await api.video.cutSession(props.sessionId) as CutProject | Record<string, never>;
        if (p && (p as CutProject).id) {
          setProj(p as CutProject);
          return p as CutProject;
        }
      }
      const list = await api.video.cutList() as CutProject[];
      if (list?.[0]) {
        setProj(list[0]);
        return list[0];
      }
    } catch {
      /* empty desk */
    }
    return null;
  }, [props.sessionId]);

  useEffect(() => { void load(); }, [load]);

  useEffect(() => {
    if (!props.sourceHash || proj?.source_hash === props.sourceHash) return;
    void run("ingest", async () => {
      const p = await api.video.cutIngest(proj?.id || "", props.sourceHash || "", proj?.title || "") as CutProject;
      setProj(p);
    });
  }, [props.sourceHash]);

  useEffect(() => {
    let off: (() => void) | undefined;
    void import("@wailsio/runtime").then((mod: any) => {
      const Events = mod.Events;
      if (!Events?.On) return;
      off = Events.On("yoyo:item", (ev: any) => {
        const kind = String(ev?.item_kind || ev?.ItemKind || "");
        const payload = ev?.payload || ev?.Payload || {};
        if (kind === "cut_state" && payload.id) void load(String(payload.id));
        if (kind === "video_route") {
          const pane = String(payload.pane || "");
          if (pane === "studio" || pane === "drama" || pane === "canvas") openVideoPane(pane as any);
        }
      });
    }).catch(() => {});
    return () => { off?.(); };
  }, [load, openVideoPane]);

  async function ensureSession() {
    if (props.sessionId) return props.sessionId;
    const id = await props.onNeedSession?.();
    return typeof id === "string" ? id : "";
  }

  async function run(label: string, fn: () => Promise<void>) {
    setBusy(label);
    try {
      await fn();
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  async function onFile(file: File) {
    const data = await new Promise<string>((resolve, reject) => {
      const r = new FileReader();
      r.onload = () => resolve(String(r.result || ""));
      r.onerror = () => reject(r.error);
      r.readAsDataURL(file);
    });
    const b64 = data.includes(",") ? data.slice(data.indexOf(",") + 1) : data;
    await run("upload", async () => {
      const ref = await api.video.putStudioMedia(b64, file.type, "video");
      const hash = String(ref?.cas_hash || ref?.hash || "");
      if (!hash) throw new Error(c.needHash);
      const sessionId = await ensureSession();
      const p = await api.video.cutCreate(file.name.replace(/\.[^.]+$/, ""), hash, sessionId) as CutProject;
      setProj(p);
    });
  }

  async function route(dest: "studio" | "drama" | "canvas" | "cut") {
    if (dest === "studio" || dest === "drama" || dest === "canvas") {
      setFootageOpen(false);
      openVideoPane(dest);
    }
    const sessionId = await ensureSession();
    if (!sessionId || dest === "drama") return;
    await run("route", async () => {
      await api.video.route(sessionId, dest, proj?.id || "", proj?.title || "");
    });
  }

  const clips = proj?.clips || proj?.breakdown?.clips || [];
  const cues = proj?.captions?.cues || [];
  const preview = proj?.result_url || proj?.source_url || proj?.media_url || "";

  return (
    <div className="footage-desk flex h-full min-h-0 flex-col overflow-hidden" data-testid="footage-desk" data-compact={props.compact ? "true" : undefined}>
      <header className="flex shrink-0 items-start justify-between gap-3 border-b border-border/70 px-4 py-3">
        <div className="min-w-0">
          <p className="text-[11px] font-medium text-muted">{c.label}</p>
          <h1 className="truncate text-[16px] font-semibold tracking-[-0.03em]">{proj?.title || c.title}</h1>
          {proj?.duration ? <p className="text-[12px] text-muted">{proj.duration.toFixed(1)}s</p> : null}
        </div>
        <input ref={fileRef} type="file" accept="video/*" className="hidden" onChange={(e) => {
          const f = e.target.files?.[0];
          if (f) void onFile(f);
          e.currentTarget.value = "";
        }} />
        <Button size="sm" variant="ghost" disabled={!!busy} onClick={() => fileRef.current?.click()}>{c.drop}</Button>
      </header>

      {!proj?.source_hash ? (
        <div className="flex min-h-0 flex-1 flex-col justify-center gap-4 px-5 py-8" data-testid="footage-empty">
          <p className="max-w-[36ch] text-[13px] leading-[1.55] text-muted">{c.empty}</p>
          <div className="flex flex-wrap gap-2">
            <Chip icon={Scissors} label={c.clipThis} onClick={() => fileRef.current?.click()} />
            <Chip icon={Mic} label={c.dub} onClick={() => void route("cut")} />
            <Chip icon={Sparkles} label={c.makeShort} onClick={() => void route("studio")} />
            <Chip icon={Clapperboard} label={c.makeDrama} onClick={() => void route("drama")} />
            <Chip icon={PanelsTopLeft} label={c.goCanvas} onClick={() => void route("canvas")} />
          </div>
        </div>
      ) : (
        <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
          {preview ? (
            <video className="mb-3 max-h-56 w-full bg-black" src={preview} poster={proj.poster_url || undefined} controls />
          ) : null}
          <div className="mb-3 flex flex-wrap gap-2">
            <Button size="sm" variant="ghost" disabled={!!busy} onClick={() => void run("analyze", async () => { setProj(await api.video.cutAnalyze(proj.id) as CutProject); })}>{c.analyze}</Button>
            <Button size="sm" variant="ghost" disabled={!!busy} onClick={() => void run("propose", async () => { setProj(await api.video.cutPropose(proj.id, 3, "15-30") as CutProject); })}>{c.propose}</Button>
            <Button size="sm" variant="ghost" disabled={!!busy || clips.length === 0} onClick={() => void run("apply", async () => { setProj(await api.video.cutApply(proj.id) as CutProject); })}>{c.apply}</Button>
            <Button size="sm" variant="ghost" disabled={!!busy} onClick={() => void run("mix", async () => { setProj(await api.video.cutMix(proj.id, true) as CutProject); })}>{c.mix}</Button>
            <Button size="sm" variant="ghost" disabled={!!busy} onClick={() => void run("asr", async () => { setProj(await api.video.cutTranscribe(proj.id) as CutProject); })}>{c.transcribe}</Button>
          </div>
          <Section title={c.scenes}>{(proj.breakdown?.scenes || []).length ? `${(proj.breakdown?.scenes || []).length}` : "—"}</Section>
          <Section title={c.clips}>
            {clips.length === 0 ? "—" : (
              <ul className="mt-1 flex flex-col gap-1">
                {clips.map((clip, i) => (
                  <li key={clip.id || i} className="flex items-center justify-between gap-2 text-[12px]">
                    <span>{clip.title || `clip ${i + 1}`}</span>
                    <span className="font-mono text-muted">{(clip.start || 0).toFixed(1)}–{(clip.end || 0).toFixed(1)} · e{clip.scores?.energy ?? "—"}</span>
                  </li>
                ))}
              </ul>
            )}
          </Section>
          <Section title={c.stems}>
            <p className="font-mono text-[11px] text-muted">
              vo {proj.stems?.dub || proj.stems?.voice || "—"} · bg {proj.stems?.music || "—"}
            </p>
          </Section>
          <Section title={c.captions}>
            {cues.length === 0 ? "—" : (
              <ul className="mt-1 flex flex-col gap-1">
                {cues.slice(0, 8).map((cue, i) => (
                  <li key={i} className="text-[12px] leading-[1.45]">{cue.text}</li>
                ))}
              </ul>
            )}
          </Section>
        </div>
      )}
    </div>
  );
}

function Chip(props: { icon: typeof Scissors; label: string; onClick: () => void }) {
  const Icon = props.icon;
  return (
    <button
      type="button"
      className="inline-flex items-center gap-1.5 rounded-full border border-border/80 px-3 py-1.5 text-[13px] text-muted hover:border-border hover:bg-lift hover:text-foreground"
      onClick={props.onClick}
    >
      <Icon className="size-3.5 opacity-70" aria-hidden />
      {props.label}
    </button>
  );
}

function Section(props: { title: string; children: ReactNode }) {
  return (
    <section className="mb-3 border-t border-border/50 pt-2">
      <h2 className="text-[11px] font-medium text-muted">{props.title}</h2>
      <div className="mt-1 text-[13px]">{props.children}</div>
    </section>
  );
}
