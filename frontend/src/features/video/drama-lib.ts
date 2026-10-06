export type Drama = {
  id: string;
  title: string;
  style: string;
  aspect_ratio: string;
  episode_count: number;
};

export type Episode = {
  id: string;
  drama_id: string;
  episode_number?: number;
  title: string;
  content: string;
  script_content: string;
  resolution: string;
  pipeline: string;
  image_provider_id: string;
  video_provider_id: string;
  image_model?: string;
  video_model?: string;
  tts_provider_id?: string;
  tts_model?: string;
  video_url?: string;
  poster_url?: string;
};

export type Asset = {
  id: string;
  name?: string;
  location?: string;
  role?: string;
  image_url?: string;
  image_hash?: string;
  linked?: boolean;
  final_prompt?: string;
  appearance?: string;
  styling?: string;
  prompt?: string;
  lighting?: string;
  description?: string;
  time_of_day?: string;
};

export type Shot = {
  id: string;
  shot_number: number;
  title: string;
  description: string;
  video_prompt: string;
  duration: number;
  status: string;
  video_url?: string;
  poster_url?: string;
  character_ids?: string[];
  scene_id?: string;
  prop_ids?: string[];
};

export type Job = {
  id: string;
  type: string;
  status: string;
  error?: string;
  episode_id?: string;
  storyboard_id?: string;
  character_id?: string;
  scene_id?: string;
  prop_id?: string;
  media_url?: string;
  poster_url?: string;
};

export type PlanEpisode = {
  n: number;
  title: string;
  logline?: string;
  hook?: string;
  start: number;
  end: number;
  character_ids?: string[];
  target_seconds?: number;
  episode_id?: string;
};

export type EpisodePlan = {
  id?: string;
  drama_id?: string;
  status?: string;
  source_id?: string;
  episodes?: PlanEpisode[];
  queue_kind?: string;
  queue_index?: number;
};

export type Bundle = {
  drama: Drama;
  episode: Episode;
  characters: Asset[];
  scenes: Asset[];
  props: Asset[];
  shots: Shot[];
  jobs: Job[];
  plan: { chars: number; target_seconds: number; segment_count: number };
  status?: { ffmpeg?: boolean; missing?: string[]; ffmpeg_install?: { phase?: string; percent?: number; error?: string } };
};

export type Phase = "script" | "cast" | "board" | "cut";

export type PhaseStep = "rewrite" | "extract" | "prompts" | "stills" | "storyboard" | "video_prompts" | "clips" | "merge";

export function isAgentStep(step: PhaseStep) {
  return step === "rewrite" || step === "extract" || step === "prompts" || step === "storyboard" || step === "video_prompts";
}

export function fill(template: string, vars: Record<string, string | number>) {
  return template.replace(/\{(\w+)\}/g, (_, k) => String(vars[k] ?? ""));
}

const SERIES_PLACEHOLDER = /^(untitled(\s+(series|drama|short\s*drama))?|new\s+(series|short\s*drama|drama)|新短剧|未命名(剧集|短剧|系列)?)$/i;
const EPISODE_PLACEHOLDER = /^(untitled(\s+episode)?|new\s+episode|episode\s*\d+|第\s*\d+\s*集|未命名集)$/i;

export function isPlaceholderSeries(title: string | undefined) {
  const s = (title || "").trim();
  return !s || SERIES_PLACEHOLDER.test(s);
}

export function isPlaceholderEpisode(title: string | undefined) {
  const s = (title || "").trim();
  return !s || EPISODE_PLACEHOLDER.test(s);
}

export function shownSeriesTitle(title: string | undefined, untitled: string) {
  return isPlaceholderSeries(title) ? untitled : String(title).trim();
}

export function shownEpisodeTitle(title: string | undefined, n: number | undefined, episodeN: string) {
  const num = Math.max(1, Number(n) || 1);
  if (isPlaceholderEpisode(title)) return fill(episodeN, { n: num });
  return String(title).trim();
}

export function commitSeriesTitle(typed: string) {
  const t = typed.trim();
  return isPlaceholderSeries(t) ? "" : t;
}

export function commitEpisodeTitle(typed: string) {
  const t = typed.trim();
  return isPlaceholderEpisode(t) ? "" : t;
}

export function pipeStatus(raw: string, stage: string) {
  try {
    const m = JSON.parse(raw || "{}");
    return String(m?.[stage]?.status || "");
  } catch {
    return "";
  }
}

export function selectedClipIds(shots: Shot[], sel: Record<string, boolean>) {
  return shots.filter((s) => s.video_url && sel[s.id] !== false).map((s) => s.id);
}

export function parseModels(raw: string): string[] {
  const s = (raw || "").trim();
  if (!s) return [];
  if (s.startsWith("[")) {
    try {
      const v = JSON.parse(s);
      if (Array.isArray(v)) return v.map(String).filter(Boolean);
    } catch {
      /* fall through */
    }
  }
  return s.split(",").map((x) => x.trim()).filter(Boolean);
}

export function serviceKind(raw: unknown): string {
  const s = String(raw || "").toLowerCase().trim();
  if (s === "speech" || s === "audio") return "tts";
  if (s === "llm" || s === "text") return "chat";
  return s;
}

export function providerKey(id: string): string {
  return String(id || "").trim().replace(/^ch-/, "").replace(/--(text|image|video|audio)$/i, "");
}

export function catalogOf(p: { model?: string; models?: string } | undefined, selected = ""): string[] {
  const out: string[] = [];
  const add = (v: string) => {
    const s = (v || "").trim();
    if (s && !out.includes(s)) out.push(s);
  };
  add(String(p?.model || ""));
  add(selected);
  for (const m of parseModels(String(p?.models || ""))) add(m);
  return out;
}

export function isNarrator(a: Asset) {
  const t = `${a.name || ""} ${a.role || ""}`.toLowerCase();
  return ["旁白", "画外音", "narrator", "voice-over", "voiceover"].some((k) => t.includes(k));
}

export function assetName(a: Asset, kind: string) {
  if (kind === "scene") return a.location || "";
  return a.name || "";
}

export function linkedAssets(items: Asset[]) {
  return items.filter((a) => a.linked !== false);
}

export function phaseDone(ep: Episode | undefined, bundle: Bundle | null, phase: Phase): boolean {
  if (!ep) return false;
  if (phase === "script") {
    return pipeStatus(ep.pipeline, "rewrite") === "done" || !!(ep.script_content || "").trim();
  }
  if (phase === "cast") {
    if (pipeStatus(ep.pipeline, "assets") === "done") return true;
    const looks = [...(bundle?.characters || []), ...(bundle?.scenes || []), ...(bundle?.props || [])].filter((a) => a.linked !== false && !isNarrator(a));
    return looks.length > 0 && looks.every((a) => !!a.image_url);
  }
  if (phase === "board") {
    if (pipeStatus(ep.pipeline, "gen") === "done") return true;
    const shots = bundle?.shots || [];
    return shots.length > 0 && shots.some((s) => !!s.video_url);
  }
  return pipeStatus(ep.pipeline, "merge") === "done" || !!ep.video_url;
}

export function jobMoving(status: string) {
  return status === "queued" || status === "running" || status === "polling";
}

export function firstLook(bundle: Bundle | null): { kind: string; id: string } | null {
  if (!bundle) return null;
  const ch = linkedAssets(bundle.characters)[0];
  if (ch) return { kind: "character", id: ch.id };
  const sc = linkedAssets(bundle.scenes)[0];
  if (sc) return { kind: "scene", id: sc.id };
  const pr = linkedAssets(bundle.props)[0];
  if (pr) return { kind: "prop", id: pr.id };
  return null;
}

export function looksLikeSeries(text: string) {
  const n = Array.from((text || "").trim()).length;
  if (n >= 2000) return true;
  const chapters = (text || "").match(/第[0-9一二三四五六七八九十百千零两]+[章节回]|Chapter\s+\d+/gi);
  return !!chapters && chapters.length >= 2 && n >= 400;
}

export function phaseQueue(phase: Phase, bundle: Bundle | null): PhaseStep[] {
  if (phase === "script") return ["rewrite"];
  if (phase === "cast") {
    const looks = [
      ...(bundle?.characters || []),
      ...(bundle?.scenes || []),
      ...(bundle?.props || []),
    ].filter((a) => a.linked !== false && !isNarrator(a));
    const steps: PhaseStep[] = [];
    if (looks.length === 0) steps.push("extract");
    if (looks.length === 0 || looks.some((a) => !(a.final_prompt || "").trim())) steps.push("prompts");
    if (looks.length === 0 || looks.some((a) => !a.image_url)) steps.push("stills");
    return steps.length ? steps : ["stills"];
  }
  if (phase === "board") {
    const shots = bundle?.shots || [];
    const steps: PhaseStep[] = [];
    if (!shots.length) steps.push("storyboard");
    if (!shots.length || shots.some((s) => !(s.video_prompt || "").trim())) steps.push("video_prompts");
    if (!shots.length || shots.some((s) => !s.video_url)) steps.push("clips");
    return steps.length ? steps : ["clips"];
  }
  return ["merge"];
}
