import { EXTRA_DRAMA_STYLES } from "./drama-style-catalog.ts";

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
  video_hash?: string;
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
  video_hash?: string;
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

export const PRESET_STYLE_GROUP_IDS = ["anime", "volume", "paint", "east", "live"] as const;

export type PresetStyleGroupId = (typeof PRESET_STYLE_GROUP_IDS)[number];

const CORE_PRESET_STYLES = [
  { value: "3d", name: "3D cinematic", group: "volume" },
  { value: "pixar", name: "Pixar volume", group: "volume" },
  { value: "disney", name: "Storybook 3D", group: "volume" },
  { value: "clay", name: "Stop-motion clay", group: "volume" },
  { value: "chibi", name: "Chibi", group: "volume" },
  { value: "anime", name: "Anime", group: "anime" },
  { value: "ghibli", name: "Ghibli", group: "anime" },
  { value: "shinkai", name: "Luminous city", group: "anime" },
  { value: "shoujo", name: "Shojo", group: "anime" },
  { value: "webtoon", name: "Webtoon", group: "anime" },
  { value: "dongman", name: "Donghua 2D", group: "anime" },
  { value: "retro80", name: "80s cel", group: "anime" },
  { value: "watercolor", name: "Watercolor", group: "paint" },
  { value: "painterly", name: "Painterly", group: "paint" },
  { value: "comic", name: "Graphic novel", group: "paint" },
  { value: "noir", name: "Noir ink", group: "paint" },
  { value: "sketch", name: "Pencil sketch", group: "paint" },
  { value: "pixel", name: "Pixel", group: "paint" },
  { value: "guofeng", name: "Guofeng 2.5D", group: "east" },
  { value: "ink", name: "Ink wash", group: "east" },
  { value: "xianxia", name: "Xianxia", group: "east" },
  { value: "ukiyo", name: "Ukiyo-e", group: "east" },
  { value: "live", name: "Live-action", group: "live" },
  { value: "kdrama", name: "K-drama", group: "live" },
  { value: "wes", name: "Pastel symmetry", group: "live" },
  { value: "cyber", name: "Neon noir", group: "live" },
  { value: "mecha", name: "Mecha", group: "volume" },
  { value: "lego", name: "Brick build", group: "volume" },
  { value: "lowpoly", name: "Low poly", group: "volume" },
  { value: "school", name: "Campus anime", group: "anime" },
  { value: "sports", name: "Sports anime", group: "anime" },
  { value: "action", name: "Ink action", group: "anime" },
  { value: "papercut", name: "Paper-cut", group: "paint" },
  { value: "stained", name: "Stained glass", group: "paint" },
  { value: "gouache", name: "Gouache", group: "paint" },
  { value: "vaporwave", name: "Vaporwave", group: "paint" },
  { value: "wuxia", name: "Wuxia", group: "east" },
  { value: "period", name: "Period court", group: "east" },
  { value: "fable", name: "Animal fable", group: "east" },
  { value: "horror", name: "Horror", group: "live" },
  { value: "pastoral", name: "Pastoral", group: "live" },
  { value: "space", name: "Space opera", group: "live" },
  { value: "steampunk", name: "Steampunk", group: "live" },
  { value: "isometric", name: "Isometric", group: "live" },
  { value: "documentary", name: "Documentary", group: "live" },
  { value: "felt", name: "Felt puppet", group: "volume" },
  { value: "risograph", name: "Risograph", group: "paint" },
  { value: "crayon", name: "Crayon", group: "paint" },
  { value: "collage", name: "Collage", group: "paint" },
  { value: "blueprint", name: "Blueprint", group: "live" },
  { value: "seinen", name: "Seinen", group: "anime" },
  { value: "manhua", name: "Manhua color", group: "east" },
  { value: "gothic", name: "Gothic", group: "live" },
  { value: "western", name: "Western", group: "live" },
  { value: "underwater", name: "Underwater", group: "live" },
  { value: "citypop", name: "City pop", group: "anime" },
  { value: "lofi", name: "Lo-fi room", group: "paint" },
  { value: "origami", name: "Origami", group: "volume" },
] as const;

export const PRESET_STYLES = [
  ...CORE_PRESET_STYLES,
  ...EXTRA_DRAMA_STYLES.map((row) => ({ value: row.value, name: row.name, group: row.group })),
] as const;

export function presetStyleGroup(value: string): PresetStyleGroupId | "" {
  return PRESET_STYLES.find((row) => row.value === value)?.group || "";
}

export function mergeDramaStyles(api: Array<{ value?: string; name?: string }> | null | undefined) {
  const map = new Map<string, { value: string; name: string }>();
  for (const row of PRESET_STYLES) map.set(row.value, { value: row.value, name: row.name });
  for (const row of api || []) {
    const value = String(row?.value || "").trim();
    if (!value) continue;
    map.set(value, { value, name: String(row.name || map.get(value)?.name || value) });
  }
  const order = PRESET_STYLES.map((row) => row.value);
  const extras = [...map.keys()].filter((value) => !order.includes(value as (typeof order)[number]));
  return [...order, ...extras].map((value) => map.get(value)!).filter(Boolean);
}

const PRESET_STYLE_COVERS = new Set<string>(PRESET_STYLES.map((style) => style.value));

export function isPresetDramaStyle(value: string) {
  return PRESET_STYLE_COVERS.has(String(value || "").trim());
}

export function dramaStyleCover(value: string, _label = "") {
  const key = String(value || "").trim();
  if (!key || !isPresetDramaStyle(key)) return "";
  return `/drama-styles/${key}.jpg`;
}

export function styleLabel(names: object | undefined, value: string, fallback = "") {
  if (names && value in names) {
    const label = (names as Record<string, string>)[value];
    if (label) return label;
  }
  return fallback || value;
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
