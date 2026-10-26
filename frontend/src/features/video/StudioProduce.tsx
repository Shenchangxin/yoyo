/**
 * 本文件拥有短片从设定到交付的步骤界面。不拥有工作台的选题和发布列表。
 * 套用、口播、出图、合成、封面和打包会改服务端。未完成镜头每 2000 毫秒重读，自动阶段每 1500 毫秒重读，不是另一条模型循环。
 */
import { useEffect, useRef, useState, type ReactNode } from "react";
import { Check, ChevronRight, ImagePlus, LoaderCircle, Video } from "lucide-react";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import {
  gateIndex,
  projectShots,
  shotReady,
  STUDIO_GATES,
  type StudioCatalog,
  type StudioGate,
  type StudioProject,
  type StudioShot,
} from "./studio-model";
import { requestSendToCut } from "./open-cut";

const ASPECTS = ["9:16", "16:9", "1:1"];
const LENGTHS = [15, 30, 45, 60];
const AUTO_PHASES = ["script", "voice", "visuals", "compose"] as const;

/** 自动阶段是剧本、声音、画面或合成时算正在跑。 */
function autoRunning(phase?: string) {
  return phase === "script" || phase === "voice" || phase === "visuals" || phase === "compose";
}

/** StudioProduceProps 是制作步从工作台接来的当前短片、目录和保存动作。 */
export type StudioProduceProps = {
  current: StudioProject;
  catalog: StudioCatalog;
  busy: string;
  mediaSrc: (hash?: string) => string;
  onChange: (next: StudioProject) => void;
  run: (label: string, fn: () => Promise<void>) => Promise<void>;
  saveProject: (patch: Record<string, unknown>) => Promise<void>;
  wrapTitle: string;
  setWrapTitle: (v: string) => void;
  remotionReady: boolean;
  onWrap: () => Promise<void>;
  onPublish?: () => void;
};

/** StudioProduce 按工序显示设定到交付。没有剧本且未通过时停在设定。 */
export function StudioProduce(props: StudioProduceProps) {
  const copy = useCopy();
  const s = copy.video.studio;
  const p = s.desk;
  const cur = props.current;
  const furthest = gateIndex(cur.furthest_gate);
  const [step, setStep] = useState<StudioGate>(() => {
    if (!String(cur.script || "").trim() && cur.script_status !== "approved") return "setup";
    return (cur.gate as StudioGate) || "script";
  });
  const shots = projectShots(cur);
  const onChangeRef = useRef(props.onChange);
  onChangeRef.current = props.onChange;
  const pendingKey = shots.filter((sh) => sh.job_id && (!shotReady(sh) || sh.stale)).map((sh) => sh.job_id).join(",");

  useEffect(() => {
    if (!String(cur.script || "").trim() && cur.script_status !== "approved") {
      setStep("setup");
      return;
    }
    const next = (cur.gate as StudioGate) || "script";
    if (gateIndex(next) <= furthest) setStep(next);
    // 只在短片换了的时候重置看得见的工序。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cur.id]);

  useEffect(() => {
    if (!pendingKey || !cur.id) return;
    const t = window.setInterval(() => {
      void api.video.getStudio(cur.id).then((row) => onChangeRef.current(row as StudioProject)).catch(() => {});
    }, 2000);
    return () => window.clearInterval(t);
  }, [cur.id, pendingKey]);

  useEffect(() => {
    if (!autoRunning(cur.auto_phase) || !cur.id) return;
    const t = window.setInterval(() => {
      void api.video.getStudio(cur.id).then((row) => {
        const next = row as StudioProject;
        onChangeRef.current(next);
        if (next.auto_phase === "done" && next.result_hash) setStep("delivery");
      }).catch(() => {});
    }, 1500);
    return () => window.clearInterval(t);
  }, [cur.id, cur.auto_phase]);

  /** 没越过已到达的下一步就切换工序，并记下当前工序。 */
  const go = (g: StudioGate) => {
    if (gateIndex(g) > furthest + 1) return;
    setStep(g);
    void props.saveProject({ gate: g });
  };

  return (
    <div className="flex flex-col gap-3">
      <OneClickBar
        current={cur}
        busy={props.busy}
        run={props.run}
        onChange={props.onChange}
      />
      <nav className="flex flex-wrap items-center gap-0.5" aria-label={p.setup} data-testid="studio-step-nav">
        {STUDIO_GATES.map((g, i) => {
          const locked = i > furthest;
          const done = i < furthest;
          const active = step === g;
          return (
            <span key={g} className="flex items-center">
              {i > 0 ? <span className="mx-0.5 text-[11px] text-muted/35" aria-hidden>/</span> : null}
              <button
                type="button"
                data-testid={`studio-step-${g}`}
                disabled={locked}
                title={locked ? p.locked : p[`${g}Hint` as "setupHint"]}
                aria-current={active ? "step" : undefined}
                className={cn(
                  "inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-[12px]",
                  active ? "bg-foreground text-background" : locked ? "text-muted/50" : done ? "text-foreground hover:bg-lift/70" : "text-muted hover:bg-lift/70 hover:text-foreground",
                )}
                onClick={() => go(g)}
              >
                <span className="tabular-nums text-[11px]">{done ? <Check className="size-3" aria-hidden /> : i + 1}</span>
                {p[g]}
              </button>
            </span>
          );
        })}
      </nav>
      <p className="text-[12px] text-muted">{p[`${step}Hint` as "setupHint"]}</p>
      {cur.error ? <p className="text-[12px] text-danger">{cur.error}</p> : null}
      {(cur.issues || []).length ? (
        <ul className="space-y-1" data-testid="studio-gate-issues">
          {(cur.issues || []).slice(-3).map((iss) => (
            <li key={iss.id} className={cn("text-[12px]", iss.level === "warn" ? "text-warning" : "text-danger")}>{iss.message}</li>
          ))}
        </ul>
      ) : null}
      {step === "setup" ? <SetupStep current={cur} catalog={props.catalog} busy={props.busy} run={props.run} onChange={props.onChange} saveProject={props.saveProject} onContinue={() => go("script")} /> : null}
      {step === "script" ? <ScriptStep current={cur} catalog={props.catalog} busy={props.busy} onChange={props.onChange} run={props.run} saveProject={props.saveProject} onContinue={() => go("audio")} /> : null}
      {step === "audio" ? <AudioStep current={cur} catalog={props.catalog} busy={props.busy} onChange={props.onChange} run={props.run} saveProject={props.saveProject} onContinue={() => go("visuals")} /> : null}
      {step === "visuals" ? <VisualsStep {...props} shots={shots} onContinue={() => go("compose")} /> : null}
      {step === "compose" ? <ComposeStep {...props} shots={shots} onContinue={() => go("delivery")} /> : null}
      {step === "delivery" ? <DeliveryStep {...props} onContinue={() => props.onPublish?.()} /> : null}
    </div>
  );
}

/** 一键跑完的进度。正在跑时不能再点。 */
function OneClickBar(props: {
  current: StudioProject;
  busy: string;
  run: StudioProduceProps["run"];
  onChange: StudioProduceProps["onChange"];
}) {
  const p = useCopy().video.studio.desk;
  const cur = props.current;
  const running = autoRunning(cur.auto_phase);
  const phaseLabel: Record<string, string> = {
    script: p.phaseScript,
    voice: p.phaseVoice,
    visuals: p.phaseVisuals,
    compose: p.phaseCompose,
    done: p.phaseDone,
    failed: p.phaseFailed,
  };
  const label = running ? p.oneClickRunning : cur.auto_phase === "failed" ? p.oneClickRetry : cur.result_hash ? p.oneClickAgain : p.oneClick;
  return (
    <section className="rounded-xl border border-border/70 bg-card px-3 py-3" data-testid="studio-auto">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <p className="text-[13px] text-foreground">{p.oneClick}</p>
          <p className="mt-1 text-[12px] leading-[1.45] text-muted">{p.oneClickHint}</p>
        </div>
        <Button
          type="button"
          size="sm"
          disabled={!!props.busy || running}
          onClick={() => void props.run("auto", async () => {
            props.onChange(await api.video.autoStudio(cur.id) as StudioProject);
          })}
        >
          {running ? <LoaderCircle className="mr-1 size-3.5 animate-spin" aria-hidden /> : null}
          {label}
        </Button>
      </div>
      <ol className="mt-3 flex flex-wrap gap-1.5" aria-label={p.oneClickRunning}>
        {AUTO_PHASES.map((id) => {
          const active = cur.auto_phase === id;
          const at = AUTO_PHASES.indexOf(cur.auto_phase as typeof id);
          const done = cur.auto_phase === "done" || (at >= 0 && AUTO_PHASES.indexOf(id) < at);
          return (
            <li
              key={id}
              className={cn(
                "rounded-full px-2 py-0.5 text-[11px]",
                active ? "bg-foreground text-background" : done ? "bg-lift text-foreground" : "text-muted",
              )}
              aria-current={active ? "step" : undefined}
            >
              {phaseLabel[id]}
            </li>
          );
        })}
      </ol>
      {cur.auto_error ? <p className="mt-2 text-[12px] text-danger" data-testid="studio-auto-error">{cur.auto_error}</p> : null}
      <p className="mt-2 text-[11px] text-muted">{p.oneClickManual}</p>
    </section>
  );
}

/** 设定步：预设、模板、角色和风格。 */
function SetupStep(props: {
  current: StudioProject;
  catalog: StudioCatalog;
  busy: string;
  run: StudioProduceProps["run"];
  onChange: StudioProduceProps["onChange"];
  saveProject: StudioProduceProps["saveProject"];
  onContinue: () => void;
}) {
  const copy = useCopy();
  const s = copy.video.studio;
  const p = s.desk;
  const cur = props.current;
  const cat = props.catalog;
  const look = (cat.templates || []).length + (cat.characters || []).length + (cat.styles || []).length > 0;
  return (
    <div className="flex flex-col gap-3">
      <section className="overflow-hidden rounded-xl border border-border/70 bg-card" data-testid="studio-brief">
        <FormRow label={s.aspect}>
          <ChipRow
            items={ASPECTS.map((id) => ({ id, label: id }))}
            value={cur.aspect || "9:16"}
            onPick={(id) => void props.saveProject({ aspect: id })}
          />
        </FormRow>
        <FormRow label={p.duration} hint={s.durationHint}>
          <div className="flex flex-wrap items-center gap-1.5">
            <ChipRow
              items={LENGTHS.map((n) => ({ id: String(n), label: String(n) }))}
              value={cur.target_duration ? String(cur.target_duration) : ""}
              onPick={(id) => void props.saveProject({ target_duration: Number(id) })}
            />
            <label className="flex items-center gap-1">
              <Input
                className="h-7 w-14"
                type="number"
                min={5}
                max={180}
                value={cur.target_duration || ""}
                data-testid="studio-brief-duration"
                aria-label={p.duration}
                onChange={(e) => props.onChange({ ...cur, target_duration: Number(e.target.value) || 0 })}
                onBlur={(e) => void props.saveProject({ target_duration: Number(e.target.value) || 0 })}
              />
              <span className="text-[11px] text-muted">{copy.video.durationLabel}</span>
            </label>
          </div>
        </FormRow>
        <FormRow label={p.audience} hint={s.audienceHint}>
          <Input
            value={cur.audience || ""}
            placeholder={s.audiencePh}
            data-testid="studio-brief-audience"
            aria-label={p.audience}
            onChange={(e) => props.onChange({ ...cur, audience: e.target.value })}
            onBlur={(e) => void props.saveProject({ audience: e.target.value })}
          />
        </FormRow>
        {(cat.presets || []).length ? (
          <FormRow label={p.preset}>
            <ChipRow
              items={cat.presets!.map((t) => ({ id: t.id, label: t.name }))}
              value={cur.preset_id || ""}
              onPick={(id) => void props.run("preset", async () => {
                props.onChange(await api.video.applyStudioPreset(cur.id, id) as StudioProject);
              })}
            />
          </FormRow>
        ) : null}
      </section>
      {look ? (
        <div className="flex flex-col gap-3">
          {(cat.templates || []).length ? (
            <Field label={p.template}>
              <ChipRow
                items={cat.templates!.map((t) => ({ id: t.id, label: t.title }))}
                value={cur.template_id || ""}
                onPick={(id) => void props.run("template", async () => {
                  props.onChange(await api.video.applyStudioTemplate(cur.id, id) as StudioProject);
                })}
              />
            </Field>
          ) : null}
          {(cat.characters || []).length ? (
            <Field label={p.character}>
              <ChipRow
                items={cat.characters!.map((t) => ({ id: t.id, label: t.name }))}
                value={cur.identity?.character_id || ""}
                onPick={(id) => void props.run("identity", async () => {
                  props.onChange(await api.video.setStudioIdentity(cur.id, id, cur.identity?.style_id || "") as StudioProject);
                })}
              />
            </Field>
          ) : null}
          {(cat.styles || []).length ? (
            <Field label={p.style}>
              <ChipRow
                items={cat.styles!.map((t) => ({ id: t.id, label: t.name }))}
                value={cur.identity?.style_id || cur.style || ""}
                onPick={(id) => void props.run("style", async () => {
                  props.onChange(await api.video.setStudioIdentity(cur.id, cur.identity?.character_id || "", id) as StudioProject);
                })}
              />
            </Field>
          ) : null}
        </div>
      ) : null}
      {cur.identity?.preserve ? (
        <p className="text-[12px] leading-[1.5] text-muted">
          <span className="text-foreground/85">{p.identity}</span> · {cur.identity.preserve}
          {cur.identity.prohibit ? ` · ${cur.identity.prohibit}` : ""}
        </p>
      ) : null}
      <ContinueButton label={s.continueTo.replace("{step}", p.script)} onClick={props.onContinue} />
    </div>
  );
}

/** 剧本步：正文、平台、语气、钩子和行动号召。通过后进入声音。 */
function ScriptStep(props: {
  current: StudioProject;
  catalog: StudioCatalog;
  busy: string;
  onChange: StudioProduceProps["onChange"];
  run: StudioProduceProps["run"];
  saveProject: StudioProduceProps["saveProject"];
  onContinue: () => void;
}) {
  const copy = useCopy();
  const s = copy.video.studio;
  const p = s.desk;
  const cur = props.current;
  const approved = cur.script_status === "approved";
  return (
    <div className="flex flex-col gap-2">
      <label className="flex flex-col gap-1 text-[12px] font-medium">
        {s.script}
        <textarea
          className="no-drag min-h-28 w-full rounded-lg border border-border bg-background px-2.5 py-2 text-[13px] leading-[1.55] outline-none focus-visible:border-accent/45 focus-visible:ring-2 focus-visible:ring-accent/20"
          value={cur.script || ""}
          onChange={(e) => props.onChange({ ...cur, script: e.target.value })}
          onBlur={(e) => void props.saveProject({ script: e.target.value })}
        />
      </label>
      <section className="overflow-hidden rounded-xl border border-border/70 bg-card" data-testid="studio-lines">
        <FormRow label={p.platform} hint={s.platformHint}>
          <ChipRow
            items={(props.catalog.platforms || ["douyin", "xhs", "bilibili"]).map((id) => ({ id, label: platLabel(id, s) }))}
            value={cur.platform || "douyin"}
            onPick={(id) => void props.saveProject({ platform: id })}
          />
        </FormRow>
        <FormRow label={p.tone} hint={s.toneHint}>
          <ChipRow
            items={(props.catalog.tones || ["natural", "professional", "energetic", "storytelling"]).map((id) => ({
              id, label: (p as Record<string, string>)[id] || id,
            }))}
            value={cur.tone || "natural"}
            onPick={(id) => void props.saveProject({ tone: id })}
          />
        </FormRow>
        <FormRow label={p.hook} hint={s.hookHint}>
          <Input
            value={cur.hook || ""}
            placeholder={s.hookPh}
            data-testid="studio-lines-hook"
            aria-label={p.hook}
            onChange={(e) => props.onChange({ ...cur, hook: e.target.value })}
            onBlur={(e) => void props.saveProject({ hook: e.target.value })}
          />
        </FormRow>
        <FormRow label={p.cta} hint={s.ctaHint}>
          <Input
            value={cur.cta || ""}
            placeholder={s.ctaPh}
            data-testid="studio-lines-cta"
            aria-label={p.cta}
            onChange={(e) => props.onChange({ ...cur, cta: e.target.value })}
            onBlur={(e) => void props.saveProject({ cta: e.target.value })}
          />
        </FormRow>
      </section>
      {(cur.segments || []).length ? (
        <ul className="divide-y divide-border/70 border-y border-border/70">
          {cur.segments!.map((seg) => (
            <li key={seg.id || seg.kind} className="py-1.5 text-[12px] leading-[1.45]">
              <span className="text-muted">{seg.kind || "beat"}</span>
              {seg.narration ? <span className="ml-2 text-foreground/90">{seg.narration}</span> : null}
            </li>
          ))}
        </ul>
      ) : null}
      <div className="flex flex-wrap items-center gap-2">
        <Button size="sm" variant="default" disabled={!!props.busy || !String(cur.script || "").trim() || approved} onClick={() => void props.run("approve", async () => {
          const next = await api.video.approveStudio(cur.id) as StudioProject;
          props.onChange(next);
          props.onContinue();
        })}>{p.approve}</Button>
        {approved ? (
          <>
            <span className="self-center text-[12px] text-success">{s.ready}</span>
            <ContinueButton label={s.continueTo.replace("{step}", p.audio)} onClick={props.onContinue} />
          </>
        ) : null}
      </div>
    </div>
  );
}

/** 声音步：音色、语速、口播或跳过。 */
function AudioStep(props: {
  current: StudioProject;
  catalog: StudioCatalog;
  busy: string;
  onChange: StudioProduceProps["onChange"];
  run: StudioProduceProps["run"];
  saveProject: StudioProduceProps["saveProject"];
  onContinue: () => void;
}) {
  const copy = useCopy();
  const s = copy.video.studio;
  const p = s.desk;
  const cur = props.current;
  const ready = !!(cur.narration_hash || cur.skip_audio);
  return (
    <div className="flex flex-col gap-2">
      <Preflight id={cur.id} gate="audio" />
      {(props.catalog.voices || []).length ? (
        <Field label={p.voice}>
          <ChipRow
            items={props.catalog.voices!.map((v) => ({ id: v.id, label: v.name }))}
            value={cur.voice || "alloy"}
            onPick={(id) => void props.saveProject({ voice: id })}
          />
        </Field>
      ) : null}
      {cur.narration_url ? <audio className="w-full" src={cur.narration_url} controls /> : null}
      {cur.voice_engine === "edge" ? <p className="text-[12px] text-muted">{p.edgeVoice}</p> : null}
      <Field label={p.speed}>
        <ChipRow
          items={[{ id: "0.8", label: "0.8×" }, { id: "1", label: p.speedNormal }, { id: "1.2", label: "1.2×" }]}
          value={String(cur.voice_speed || 1)}
          onPick={(id) => void props.saveProject({ voice_speed: Number(id) })}
        />
      </Field>
      <div className="flex flex-wrap gap-2">
        <Button size="sm" variant="default" disabled={!!props.busy || cur.script_status !== "approved" || !String(cur.script || "").trim()} onClick={() => void props.run("speak", async () => {
          const next = await api.video.speakStudio(cur.id, cur.script || "") as StudioProject;
          props.onChange(next);
          if (next.narration_hash) props.onContinue();
        })}>{s.speak}</Button>
        <Button size="sm" variant="ghost" disabled={!!props.busy} onClick={() => void props.run("skip", async () => {
          const next = await api.video.skipStudioAudio(cur.id) as StudioProject;
          props.onChange(next);
          props.onContinue();
        })}>{p.skipAudio}</Button>
      </div>
      <details>
        <summary className="cursor-pointer text-[12px] text-muted">{s.moreVoice}</summary>
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <Button size="sm" variant="ghost" disabled={!!props.busy || !cur.narration_hash} onClick={() => void props.run("timing", async () => {
            props.onChange(await api.video.studioTiming(cur.id) as StudioProject);
          })}>{p.applyTiming}</Button>
        </div>
      </details>
      {ready ? <ContinueButton label={s.continueTo.replace("{step}", p.visuals)} onClick={props.onContinue} /> : null}
    </div>
  );
}

/** 画面步：补齐镜头、上传、改词和按地址导入。 */
function VisualsStep(props: {
  current: StudioProject;
  busy: string;
  mediaSrc: (hash?: string) => string;
  onChange: (next: StudioProject) => void;
  run: StudioProduceProps["run"];
  saveProject: StudioProduceProps["saveProject"];
  shots: StudioShot[];
  onContinue: () => void;
}) {
  const copy = useCopy();
  const s = copy.video.studio;
  const p = s.desk;
  const cur = props.current;
  const shots = props.shots;
  const [ingestUrl, setIngestUrl] = useState("");

  /** 改一条镜头。persist 为真时写回短片。 */
  async function patchShot(i: number, extra: Partial<StudioShot>, persist = false) {
    const next = shots.map((sh, j) => (j === i ? { ...sh, ...extra } : sh));
    props.onChange({ ...cur, storyboard: { ...(cur.storyboard || {}), shots: next }, shots: next });
    if (persist) await props.saveProject({ shots: next, storyboard: { ...(cur.storyboard || {}), shots: next } });
  }

  /** 把文件登记成媒体。index 小于 0 时追加镜头，否则替换那一条。 */
  async function upload(kind: "image" | "video", file: File, index = -1) {
    const data = await readDataURL(file);
    const b64 = data.includes(",") ? data.slice(data.indexOf(",") + 1) : data;
    await props.run("upload", async () => {
      const ref = await api.video.putStudioMedia(b64, file.type, kind);
      const hash = String(ref?.cas_hash || "");
      if (!hash) return;
      const shot: StudioShot = kind === "video" ? { video_hash: hash, duration: 3, motion: "ken-burns" } : { image_hash: hash, duration: 3, motion: "ken-burns" };
      const next = index >= 0 ? shots.map((sh, j) => (j === index ? { ...sh, ...shot, stale: false } : sh)) : [...shots, shot];
      const saved = await api.video.updateStudio({ id: cur.id, shots: next, storyboard: { ...(cur.storyboard || {}), shots: next } }) as StudioProject;
      props.onChange(saved);
    });
  }

  return (
    <div className="flex flex-col gap-2">
      <Preflight id={cur.id} gate="visuals" />
      <Field label={p.fit}>
        <ChipRow
          items={[{ id: "cover", label: p.fitCover }, { id: "contain", label: p.fitContain }]}
          value={cur.storyboard?.fit === "contain" ? "contain" : "cover"}
          onPick={(id) => void props.saveProject({ storyboard: { ...(cur.storyboard || {}), fit: id } })}
        />
        <p className="text-[11px] leading-[1.4] text-muted">{p.fitHint}</p>
      </Field>
      <div className="flex flex-wrap items-center gap-2">
        <Button size="sm" variant="default" disabled={!!props.busy} onClick={() => void props.run("gen", async () => {
          await api.video.generateStudioMissing(cur.id);
          const next = await api.video.getStudio(cur.id) as StudioProject;
          props.onChange(next);
        })}>{p.generateMissing}</Button>
        <label className="inline-flex h-7 cursor-pointer items-center gap-1 rounded-md px-2 text-[12px] text-muted hover:bg-lift hover:text-foreground">
          <ImagePlus className="size-3.5" aria-hidden />
          {s.addStill}
          <input type="file" accept="image/*" className="hidden" onChange={(e) => {
            const f = e.target.files?.[0];
            e.target.value = "";
            if (f) void upload("image", f);
          }} />
        </label>
        <label className="inline-flex h-7 cursor-pointer items-center gap-1 rounded-md px-2 text-[12px] text-muted hover:bg-lift hover:text-foreground">
          <Video className="size-3.5" aria-hidden />
          {s.addClip}
          <input type="file" accept="video/*" className="hidden" onChange={(e) => {
            const f = e.target.files?.[0];
            e.target.value = "";
            if (f) void upload("video", f);
          }} />
        </label>
      </div>
      {shots.length === 0 ? <p className="text-[12px] text-muted">{s.needShots}</p> : null}
      <ul className="flex flex-col">
        {shots.map((sh, i) => {
          const src = sh.media_url || props.mediaSrc(sh.image_hash || sh.video_hash);
          const running = !!sh.job_id && (!shotReady(sh) || sh.stale);
          return (
            <li key={sh.id || i} className="flex gap-3 border-t border-border/50 py-2.5" data-testid="studio-shot-row">
              <div className="relative h-16 w-16 shrink-0 overflow-hidden rounded-md bg-lift">
                {src ? (
                  sh.video_hash ? <video className="h-full w-full object-cover" src={src} muted /> : <img className="h-full w-full object-cover" src={src} alt="" />
                ) : (
                  <span className="grid h-full place-items-center text-muted">{running ? <LoaderCircle className="size-4 animate-spin" /> : i + 1}</span>
                )}
              </div>
              <div className="grid min-w-0 flex-1 gap-1.5">
                <textarea
                  className="no-drag min-h-12 w-full rounded-md border border-border bg-background px-2 py-1 text-[12px] outline-none"
                  placeholder={p.visual}
                  value={sh.visual || sh.prompt || ""}
                  onChange={(e) => void patchShot(i, { visual: e.target.value, prompt: e.target.value })}
                  onBlur={(e) => void patchShot(i, { visual: e.target.value, prompt: e.target.value }, true)}
                />
                <Input
                  className="h-7"
                  placeholder={s.caption}
                  value={sh.caption || ""}
                  onChange={(e) => void patchShot(i, { caption: e.target.value })}
                  onBlur={(e) => void patchShot(i, { caption: e.target.value }, true)}
                />
                <div className="flex flex-wrap items-center gap-1.5">
                  <select
                    className="h-7 rounded-md border border-border bg-background px-1.5 text-[11px]"
                    value={sh.motion === "static" ? "static" : "ken-burns"}
                    onChange={(e) => {
                      const motion = e.target.value;
                      void patchShot(i, motion === "static" ? { motion, fit: "contain" } : { motion, fit: "" }, true);
                    }}
                    aria-label={p.motion}
                  >
                    <option value="ken-burns">{p.motionPush}</option>
                    <option value="static">{p.motionStill}</option>
                  </select>
                  <Input className="h-7 w-16" type="number" min={1} value={sh.duration || 3} aria-label={copy.video.shotDuration} onChange={(e) => void patchShot(i, { duration: Number(e.target.value) || 3 })} onBlur={(e) => void patchShot(i, { duration: Number(e.target.value) || 3 }, true)} />
                  <Button size="sm" variant="ghost" className="h-7 text-[11px]" disabled={!!props.busy} onClick={() => void props.run("regen", async () => {
                    await api.video.generateStudioShot(cur.id, sh.id || "");
                    props.onChange(await api.video.getStudio(cur.id) as StudioProject);
                  })}>{p.regen}</Button>
                  {sh.stale ? <span className="text-[11px] text-warning">{p.stale}</span> : null}
                  {running ? <span className="text-[11px] text-accent">{p.generating}</span> : null}
                </div>
              </div>
            </li>
          );
        })}
      </ul>
      <details>
        <summary className="cursor-pointer text-[12px] text-muted">{p.ingest}</summary>
        <p className="mt-1 text-[11px] leading-[1.45] text-muted">{p.ingestHint}</p>
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <Input className="h-8 min-w-48 flex-1" placeholder={p.ingestUrl} value={ingestUrl} onChange={(e) => setIngestUrl(e.target.value)} />
          <Button size="sm" variant="ghost" disabled={!!props.busy || !ingestUrl.trim()} onClick={() => void props.run("ingest", async () => {
            const next = await api.video.ingestStudio(cur.id, ingestUrl.trim()) as StudioProject;
            props.onChange(next);
            setIngestUrl("");
          })}>{p.ingest}</Button>
        </div>
      </details>
      {shots.some(shotReady) ? <ContinueButton label={s.continueTo.replace("{step}", p.compose)} onClick={props.onContinue} /> : null}
    </div>
  );
}

/** 合成步：背景音乐、出片、改画幅、烧字幕和送到剪辑。 */
function ComposeStep(props: {
  current: StudioProject;
  busy: string;
  shots: StudioShot[];
  run: StudioProduceProps["run"];
  onChange: StudioProduceProps["onChange"];
  wrapTitle: string;
  setWrapTitle: (v: string) => void;
  remotionReady: boolean;
  onWrap: () => Promise<void>;
  onContinue: () => void;
}) {
  const copy = useCopy();
  const s = copy.video.studio;
  const cur = props.current;
  return (
    <div className="flex flex-col gap-2">
      <Preflight id={cur.id} gate="compose" />
      <BgmRow current={cur} busy={props.busy} run={props.run} onChange={props.onChange} />
      {cur.media_url ? <video className="max-h-72 w-full rounded-xl bg-black" src={cur.media_url} poster={cur.poster_url || undefined} controls /> : null}
      <Button size="sm" variant="default" className="self-start" disabled={!!props.busy || !props.shots.some(shotReady)} onClick={() => void props.run("assemble", async () => {
        const next = await api.video.assembleStudio(cur.id) as StudioProject;
        props.onChange(next);
        if (next.result_hash) props.onContinue();
      })}>{s.assembleBtn}</Button>
      {cur.result_hash ? (
        <div className="flex flex-col gap-2" data-testid="studio-cut-ops">
          <p className="text-[12px] text-muted">{s.desk.cutHint}</p>
          <div className="flex flex-wrap items-center gap-2">
            <Button size="sm" variant="ghost" disabled={!!props.busy} onClick={() => void props.run("reframe", async () => {
              const r = await api.video.studioReframe(cur.result_hash || "", cur.aspect || "9:16");
              if (r?.cas_hash) {
                props.onChange(await api.video.updateStudio({ id: cur.id, result_hash: r.cas_hash }) as StudioProject);
              }
            })}>{s.reframe}</Button>
            {cur.subtitle_hash ? (
              <Button size="sm" variant="ghost" disabled={!!props.busy} onClick={() => void props.run("burnin", async () => {
                const r = await api.video.studioBurnin(cur.result_hash || "", cur.subtitle_hash || "");
                if (r?.cas_hash) {
                  props.onChange(await api.video.updateStudio({ id: cur.id, result_hash: r.cas_hash }) as StudioProject);
                }
              })}>{s.burnin}</Button>
            ) : null}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Input className="h-8 min-w-40 flex-1" placeholder={s.wrap} value={props.wrapTitle} onChange={(e) => props.setWrapTitle(e.target.value)} />
            <Button size="sm" variant="ghost" disabled={!!props.busy} onClick={() => void props.onWrap()}>{s.wrap}</Button>
          </div>
          {!props.remotionReady ? <p className="text-[11px] text-muted">{s.wrapHint}</p> : null}
          <Button
            size="sm"
            variant="default"
            className="self-start"
            data-testid="studio-send-to-cut"
            disabled={!!props.busy || !cur.result_hash}
            onClick={() => requestSendToCut({ hash: cur.result_hash, title: cur.title || copy.video.conductor.cutTitle })}
          >{copy.video.conductor.sendToCut}</Button>
        </div>
      ) : null}
      {cur.result_hash ? <ContinueButton label={s.continueTo.replace("{step}", s.desk.delivery)} onClick={props.onContinue} /> : null}
    </div>
  );
}

/** 交付步：封面、文案、打包和版本恢复。可发布后才继续。 */
function DeliveryStep(props: {
  current: StudioProject;
  busy: string;
  catalog: StudioCatalog;
  onChange: (next: StudioProject) => void;
  run: StudioProduceProps["run"];
  onContinue?: () => void;
}) {
  const copy = useCopy();
  const s = copy.video.studio;
  const p = s.desk;
  const cur = props.current;
  const ready = cur.delivery?.status === "publishable";
  const activeVer = String((cur.versions || []).find((v) => v.result_hash === cur.result_hash)?.n || "");
  return (
    <div className="flex flex-col gap-3" data-testid="studio-delivery">
      <Preflight id={cur.id} gate="delivery" />
      <p className={cn("text-[12px]", ready ? "text-success" : "text-warning")}>{ready ? p.publishable : (cur.delivery?.note || p.draftPack)}</p>
      {cur.media_url ? <video className="max-h-56 w-full rounded-xl bg-black" src={cur.media_url} poster={cur.cover_url || cur.poster_url || undefined} controls /> : null}
      {cur.result_hash ? (
        <Button
          size="sm"
          variant="default"
          className="self-start"
          data-testid="studio-delivery-send-to-cut"
          disabled={!!props.busy}
          onClick={() => requestSendToCut({ hash: cur.result_hash, title: cur.title || copy.video.conductor.cutTitle })}
        >{copy.video.conductor.sendToCut}</Button>
      ) : null}
      {cur.copy_text ? <pre className="whitespace-pre-wrap rounded-lg bg-lift/50 px-3 py-2 text-[12px] leading-[1.55]">{cur.copy_text}</pre> : null}
      {(cur.delivery?.files || []).length ? (
        <p className="text-[12px] text-muted">{cur.delivery!.files!.map((f) => f.kind).filter(Boolean).join(" · ")}</p>
      ) : null}
      <div className="flex flex-wrap gap-1.5">
        {(cur.cover_urls || []).map((c) => (
          <button key={c.hash} type="button" className={cn("overflow-hidden rounded-md border", cur.cover_hash === c.hash ? "border-foreground" : "border-border")} onClick={() => void props.run("cover", async () => {
            props.onChange(await api.video.pickStudioCover(cur.id, c.hash || "") as StudioProject);
          })} title={p.pickCover}>
            {c.url ? <img src={c.url} alt="" className="h-16 w-16 object-cover" /> : null}
          </button>
        ))}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button size="sm" variant="ghost" disabled={!!props.busy} onClick={() => void props.run("cover", async () => {
          await api.video.generateStudioCover(cur.id);
          props.onChange(await api.video.getStudio(cur.id) as StudioProject);
        })}>{p.covers}</Button>
        <Button size="sm" variant="ghost" disabled={!!props.busy} onClick={() => void props.run("copy", async () => {
          props.onChange(await api.video.writeStudioCopy(cur.id, cur.platform || "") as StudioProject);
        })}>{p.writeCopy}</Button>
        <Button size="sm" variant="default" disabled={!!props.busy || !cur.result_hash} onClick={() => void props.run("pack", async () => {
          const next = await api.video.packageStudio(cur.id) as StudioProject;
          props.onChange(next);
          if (next.delivery?.status === "publishable") props.onContinue?.();
        })}>{p.package}</Button>
      </div>
      {ready ? <ContinueButton label={s.continueTo.replace("{step}", s.publish)} onClick={() => props.onContinue?.()} /> : null}
      {(cur.versions || []).length > 0 ? (
        <Field label={p.versions}>
          <ChipRow
            items={(cur.versions || []).map((v) => ({ id: String(v.n), label: `V${v.n}` }))}
            value={activeVer}
            onPick={(id) => void props.run("restore", async () => {
              props.onChange(await api.video.restoreStudio(cur.id, Number(id)) as StudioProject);
            })}
          />
        </Field>
      ) : null}
    </div>
  );
}

/** 读这一道工序的预检，失败时显示缺少适配器。 */
function Preflight(props: { id: string; gate: string }) {
  const p = useCopy().video.studio.desk;
  const [pf, setPf] = useState<{ ok?: boolean; spends?: string[]; missing?: string[]; message?: string } | null>(null);
  const [pfErr, setPfErr] = useState("");
  useEffect(() => {
    if (!props.id) return;
    setPfErr("");
    void api.video.studioPreflight(props.id, props.gate).then((row) => setPf(row as typeof pf)).catch((e) => {
      setPf(null);
      setPfErr(api.errMessage(e));
    });
  }, [props.id, props.gate]);
  if (!props.id) return null;
  if (pfErr) {
    return <p className="text-[12px] text-warning" data-testid="studio-preflight">{p.missingAdapters}</p>;
  }
  if (!pf) return <p className="text-[12px] text-muted" data-testid="studio-preflight">{p.preflight}</p>;
  const spend = (pf.spends || []).join(" · ");
  return (
    <p className={cn("text-[12px]", pf.ok ? "text-muted" : "text-warning")} data-testid="studio-preflight">
      {p.preflight}{spend ? ` · ${spend}` : ""}
      {pf.missing?.length ? ` · ${p.missingAdapters}` : ""}
    </p>
  );
}

/** 列出背景音乐，可以随机、关掉或点名使用。 */
function BgmRow(props: {
  current: StudioProject;
  busy: string;
  run: StudioProduceProps["run"];
  onChange: StudioProduceProps["onChange"];
}) {
  const p = useCopy().video.studio.desk;
  const cur = props.current;
  const [list, setList] = useState<{ dir?: string; items?: { name: string }[] } | null>(null);
  useEffect(() => {
    void api.video.listStudioBGM().then((row) => setList(row as { dir?: string; items?: { name: string }[] })).catch(() => {});
  }, [cur.id, cur.bgm_hash]);
  const items = list?.items || [];
  return (
    <section className="flex flex-col gap-1.5" data-testid="studio-bgm">
      <span className="text-[12px] text-muted">{p.bgm}</span>
      <p className="text-[11px] leading-[1.4] text-muted">{p.bgmHint}</p>
      {list?.dir ? <p className="break-all text-[11px] text-muted">{list.dir}</p> : null}
      <div className="flex flex-wrap gap-1.5">
        <Button size="sm" variant="ghost" className="h-7 text-[11px]" disabled={!!props.busy || items.length === 0} onClick={() => void props.run("bgm", async () => {
          props.onChange(await api.video.useStudioBGM(cur.id, "random") as StudioProject);
        })}>{p.bgmRandom}</Button>
        <Button size="sm" variant="ghost" className="h-7 text-[11px]" disabled={!!props.busy || !cur.bgm_hash} onClick={() => void props.run("bgm", async () => {
          props.onChange(await api.video.clearStudioBGM(cur.id) as StudioProject);
        })}>{p.bgmOff}</Button>
        {items.map((it) => (
          <Button key={it.name} size="sm" variant="ghost" className="h-7 text-[11px]" disabled={!!props.busy} onClick={() => void props.run("bgm", async () => {
            props.onChange(await api.video.useStudioBGM(cur.id, it.name) as StudioProject);
          })}>{it.name}</Button>
        ))}
      </div>
      {list && items.length === 0 ? <p className="text-[11px] text-muted">{p.bgmEmpty}</p> : null}
    </section>
  );
}

/** 带右箭头的继续按钮。 */
function ContinueButton(props: { label: string; disabled?: boolean; onClick: () => void }) {
  return (
    <Button type="button" variant="default" className="mt-1 self-start gap-1" disabled={props.disabled} onClick={props.onClick}>
      {props.label}
      <ChevronRight className="size-3.5" aria-hidden />
    </Button>
  );
}

/** 一行带说明的表单。 */
function FormRow(props: { label: string; hint?: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5 border-t border-border/50 px-3 py-2.5 first:border-t-0">
      <div>
        <span className="text-[13px] text-foreground/85">{props.label}</span>
        {props.hint ? <p className="mt-0.5 text-[11px] leading-[1.4] text-muted">{props.hint}</p> : null}
      </div>
      {props.children}
    </div>
  );
}

/** 一块带标签的字段。 */
function Field(props: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-[12px] text-muted">{props.label}</span>
      {props.children}
    </div>
  );
}

/** 一排可切换的圆角选项。 */
function ChipRow(props: { items: { id: string; label: string }[]; value: string; onPick: (id: string) => void }) {
  return (
    <div className="flex flex-wrap gap-1">
      {props.items.map((it) => (
        <button
          key={it.id}
          type="button"
          aria-pressed={props.value === it.id}
          className={cn("rounded-full px-2 py-0.5 text-[11px]", props.value === it.id ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70")}
          onClick={() => props.onPick(it.id)}
        >
          {it.label}
        </button>
      ))}
    </div>
  );
}

/** 把平台 id 换成界面名字，没有就用原 id。 */
function platLabel(id: string, s: ReturnType<typeof useCopy>["video"]["studio"]) {
  const p = s.desk;
  const map: Record<string, string> = {
    douyin: s.trendDouyin,
    bilibili: s.trendBilibili,
    zhihu: s.trendZhihu,
    xhs: p.xhs,
    channels: p.channels,
    kuaishou: p.kuaishou,
  };
  return map[id] || id;
}

/** 把文件读成 data URL。 */
function readDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(String(r.result || ""));
    r.onerror = () => reject(r.error);
    r.readAsDataURL(file);
  });
}
