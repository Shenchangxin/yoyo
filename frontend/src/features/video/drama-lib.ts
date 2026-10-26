/**
 * 本文件拥有短剧工程、集、资产、镜头和工序的类型，以及标题、风格和阶段的纯判断。
 * 不拥有看板。它不请求服务端，也不改服务端状态。
 */
import { EXTRA_DRAMA_STYLES } from "./drama-style-catalog.ts";

/** Drama 是一条系列的 id、标题、风格、画幅和集数。 */
export type Drama = {
  id: string;
  title: string;
  style: string;
  aspect_ratio: string;
  episode_count: number;
};

/** Episode 是一集的正文、剧本、分辨率、管线和成片地址。 */
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

/** Asset 是角色、场景或道具上界面会读的字段。 */
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

/** Shot 是一条分镜的序号、描述、视频提示、时长和成片。 */
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

/** Job 是一条生成任务的类型、状态，以及它挂在哪一集或哪个资产上。 */
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

/** PlanEpisode 是分集方案里的一集，带起止和目标秒数。 */
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

/** EpisodePlan 是一系列的分集方案和队列位置。 */
export type EpisodePlan = {
  id?: string;
  drama_id?: string;
  status?: string;
  source_id?: string;
  episodes?: PlanEpisode[];
  queue_kind?: string;
  queue_index?: number;
};

/** Bundle 是打开一集时一起拿到的系列、资产、镜头、任务和计划。 */
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

/** Phase 是剧本、定妆、分镜、成片四段。 */
export type Phase = "script" | "cast" | "board" | "cut";

/** PhaseStep 是某一段里可以排队的具体步骤。 */
export type PhaseStep = "rewrite" | "extract" | "prompts" | "stills" | "storyboard" | "video_prompts" | "clips" | "merge";

/** isAgentStep 判断这一步是不是改写、抽取、提示词、分镜或视频提示。step 和 rewrite、extract、prompts、storyboard、video_prompts 比较。 */
export function isAgentStep(step: PhaseStep) {
  return step === "rewrite" || step === "extract" || step === "prompts" || step === "storyboard" || step === "video_prompts";
}

/** fill 把模板里的 {名字} 换成给出的值，缺了就换成空字符串。template 把花括号里的名字换成 vars 里的值。vars 按捕获的名字取值，没有就换成空字符串。 */
export function fill(template: string, vars: Record<string, string | number>) {
  return template.replace(/\{(\w+)\}/g, /** k 当作 vars 的键。 */ (_, k) => String(vars[k] ?? ""));
}

const SERIES_PLACEHOLDER = /^(untitled(\s+(series|drama|short\s*drama))?|new\s+(series|short\s*drama|drama)|新短剧|未命名(剧集|短剧|系列)?)$/i;
const EPISODE_PLACEHOLDER = /^(untitled(\s+episode)?|new\s+episode|episode\s*\d+|第\s*\d+\s*集|未命名集)$/i;

/** isPlaceholderSeries 看标题是不是空，或像未命名系列。title 去掉空白后看是不是空，或匹不匹配未命名系列。 */
export function isPlaceholderSeries(title: string | undefined) {
  const s = (title || "").trim();
  return !s || SERIES_PLACEHOLDER.test(s);
}

/** isPlaceholderEpisode 看标题是不是空，或像未命名集、第 n 集。title 去掉空白后看是不是空，或匹不匹配未命名集。 */
export function isPlaceholderEpisode(title: string | undefined) {
  const s = (title || "").trim();
  return !s || EPISODE_PLACEHOLDER.test(s);
}

/** shownSeriesTitle 占位标题换成界面给的未命名文案。title 交给 isPlaceholderSeries。untitled 在标题是占位时原样返回。 */
export function shownSeriesTitle(title: string | undefined, untitled: string) {
  return isPlaceholderSeries(title) ? untitled : String(title).trim();
}

/** shownEpisodeTitle 占位标题换成带集数的文案，否则用原标题。title 是占位时不用它，否则去掉空白后返回。n 收成至少 1 的集数。episodeN 作为模板，把 n 填进去。 */
export function shownEpisodeTitle(title: string | undefined, n: number | undefined, episodeN: string) {
  const num = Math.max(1, Number(n) || 1);
  if (isPlaceholderEpisode(title)) return fill(episodeN, { n: num });
  return String(title).trim();
}

/** commitSeriesTitle 占位标题返回空字符串。typed 去掉空白后，是占位就返回空字符串，否则返回去掉空白的文本。 */
export function commitSeriesTitle(typed: string) {
  const t = typed.trim();
  return isPlaceholderSeries(t) ? "" : t;
}

/** commitEpisodeTitle 占位标题返回空字符串。typed 去掉空白后，是占位就返回空字符串，否则返回去掉空白的文本。 */
export function commitEpisodeTitle(typed: string) {
  const t = typed.trim();
  return isPlaceholderEpisode(t) ? "" : t;
}

/** pipeStatus 从管线 JSON 里取出某一阶段的状态。解析失败就是空。raw 当 JSON 解析，空的当 {}。stage 用来取这一阶段的 status。 */
export function pipeStatus(raw: string, stage: string) {
  try {
    const m = JSON.parse(raw || "{}");
    return String(m?.[stage]?.status || "");
  } catch {
    return "";
  }
}

/** selectedClipIds 留下有视频且没有被明确取消的镜头 id。shots 留下有 video_url 且没有被明确取消的，再取出 id。sel 用镜头 id 查，值不是 false 才留下。 */
export function selectedClipIds(shots: Shot[], sel: Record<string, boolean>) {
  return shots.filter(/** s 留下有 video_url 且 sel 里没有明确取消的镜头。 */ (s) => s.video_url && sel[s.id] !== false).map(/** s 取出 id。 */ (s) => s.id);
}

/** parseModels 把 JSON 数组或逗号分隔的模型串收成列表。raw 去掉空白，以 [ 开头就按 JSON 数组解析，否则按逗号拆。 */
export function parseModels(raw: string): string[] {
  const s = (raw || "").trim();
  if (!s) return [];
  if (s.startsWith("[")) {
    try {
      const v = JSON.parse(s);
      if (Array.isArray(v)) return v.map(String).filter(Boolean);
    } catch {
      /* 继续按逗号拆 */
    }
  }
  return s.split(",").map(/** x 去掉空白，空的丢掉。 */ (x) => x.trim()).filter(Boolean);
}

/** serviceKind 把 speech、audio 收成 tts，把 llm、text 收成 chat。raw 转成小写并去掉空白后归类，对不上就返回这个小写串。 */
export function serviceKind(raw: unknown): string {
  const s = String(raw || "").toLowerCase().trim();
  if (s === "speech" || s === "audio") return "tts";
  if (s === "llm" || s === "text") return "chat";
  return s;
}

/** providerKey 去掉 id 上的 ch- 前缀和结尾的通道种类。 */
export function providerKey(id: string): string {
  return String(id || "").trim().replace(/^ch-/, "").replace(/--(text|image|video|audio)$/i, "");
}

/** catalogOf 合并提供方自己的模型、当前选中和模型列表，去掉重复。p 读 model 和 models。selected 放进列表。 */
export function catalogOf(p: { model?: string; models?: string } | undefined, selected = ""): string[] {
  const out: string[] = [];
  /** 非空且未出现过的模型才放进列表。v 去掉空白，非空且没出现过才放进列表。 */
  const add = (v: string) => {
    const s = (v || "").trim();
    if (s && !out.includes(s)) out.push(s);
  };
  add(String(p?.model || ""));
  add(selected);
  for (const m of parseModels(String(p?.models || ""))) add(m);
  return out;
}

/** isNarrator 看名字或角色里有没有旁白、画外音或 narrator。a 用 name 和 role 拼成小写再查找。 */
export function isNarrator(a: Asset) {
  const t = `${a.name || ""} ${a.role || ""}`.toLowerCase();
  return ["旁白", "画外音", "narrator", "voice-over", "voiceover"].some(/** k 看拼出来的文本里有没有它。 */ (k) => t.includes(k));
}

/** assetName 场景用地点，其他用名字。a 在 kind 为 scene 时取 location，否则取 name。kind 用来决定取地点还是名字。 */
export function assetName(a: Asset, kind: string) {
  if (kind === "scene") return a.location || "";
  return a.name || "";
}

/** linkedAssets 留下没有被标成未关联的资产。items 留下 linked 不是 false 的。 */
export function linkedAssets(items: Asset[]) {
  return items.filter(/** a 留下 linked 不是 false 的。 */ (a) => a.linked !== false);
}

/** phaseDone 按管线状态，或按剧本、定妆图、镜头视频判断这一段是否做完。ep 没有就返回假；剧本看 rewrite 或 script_content，成片看 merge 或 video_url。bundle 在定妆和分镜时提供角色、场景、道具或镜头。phase 决定按哪一段判断。 */
export function phaseDone(ep: Episode | undefined, bundle: Bundle | null, phase: Phase): boolean {
  if (!ep) return false;
  if (phase === "script") {
    return pipeStatus(ep.pipeline, "rewrite") === "done" || !!(ep.script_content || "").trim();
  }
  if (phase === "cast") {
    if (pipeStatus(ep.pipeline, "assets") === "done") return true;
    const looks = [...(bundle?.characters || []), ...(bundle?.scenes || []), ...(bundle?.props || [])].filter(/** a 留下仍关联且不是旁白的定妆。 */ (a) => a.linked !== false && !isNarrator(a));
    return looks.length > 0 && looks.every(/** a 看有没有 image_url。 */ (a) => !!a.image_url);
  }
  if (phase === "board") {
    if (pipeStatus(ep.pipeline, "gen") === "done") return true;
    const shots = bundle?.shots || [];
    return shots.length > 0 && shots.some(/** s 看有没有 video_url。 */ (s) => !!s.video_url);
  }
  return pipeStatus(ep.pipeline, "merge") === "done" || !!ep.video_url;
}

/** jobMoving 看任务是不是排队、运行或轮询。status 和 queued、running、polling 比较。 */
export function jobMoving(status: string) {
  return status === "queued" || status === "running" || status === "polling";
}

/** firstLook 按角色、场景、道具的顺序取第一个仍关联的资产。bundle 没有就返回空，否则按角色、场景、道具取第一个仍关联的。 */
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

/** PRESET_STYLE_GROUP_IDS 是风格分组：动画、体积、绘画、东方、真人。 */
export const PRESET_STYLE_GROUP_IDS = ["anime", "volume", "paint", "east", "live"] as const;

/** PresetStyleGroupId 是预设风格分组的 id。 */
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

/** PRESET_STYLES 是内置风格加上额外风格，带值和分组。 */
export const PRESET_STYLES = [
  ...CORE_PRESET_STYLES,
  ...EXTRA_DRAMA_STYLES.map(/** row 只留下 value、name 和 group。 */ (row) => ({ value: row.value, name: row.name, group: row.group })),
] as const;

/** presetStyleGroup 返回这个风格值的分组，不在表里就是空。value 在预设表里找分组，没有就返回空。 */
export function presetStyleGroup(value: string): PresetStyleGroupId | "" {
  return PRESET_STYLES.find(/** row 用 value 和传入的风格值比对。 */ (row) => row.value === value)?.group || "";
}

/** mergeDramaStyles 用服务端列表覆盖同名风格，预设顺序在前，多出来的排在后面。api 每一项用 value 覆盖同名风格，空 value 跳过，name 空着就留原来的或用 value。 */
export function mergeDramaStyles(api: Array<{ value?: string; name?: string }> | null | undefined) {
  const map = new Map<string, { value: string; name: string }>();
  for (const row of PRESET_STYLES) map.set(row.value, { value: row.value, name: row.name });
  for (const row of api || []) {
    const value = String(row?.value || "").trim();
    if (!value) continue;
    map.set(value, { value, name: String(row.name || map.get(value)?.name || value) });
  }
  const order = PRESET_STYLES.map(/** row 取出 value。 */ (row) => row.value);
  const extras = [...map.keys()].filter(/** value 不在预设顺序里才留下。 */ (value) => !order.includes(value as (typeof order)[number]));
  return [...order, ...extras].map(/** value 从对照表取出风格。 */ (value) => map.get(value)!).filter(Boolean);
}

const PRESET_STYLE_COVERS = new Set<string>(PRESET_STYLES.map(/** style 取出 value。 */ (style) => style.value));

/** isPresetDramaStyle 看这个值是不是预设风格。value 去掉空白后看在不在预设封面集合里。 */
export function isPresetDramaStyle(value: string) {
  return PRESET_STYLE_COVERS.has(String(value || "").trim());
}

/** dramaStyleCover 预设风格返回 /drama-styles/ 下的 jpg 路径，其他返回空。value 去掉空白后当路径里的风格名，空的或不是预设就返回空。 */
export function dramaStyleCover(value: string, _label = "") {
  const key = String(value || "").trim();
  if (!key || !isPresetDramaStyle(key)) return "";
  return `/drama-styles/${key}.jpg`;
}

/** styleLabel 优先用名字表里的中文，没有就用备用名或原值。names 有这份表且 value 是其中的键时取中文。value 当作键，没有中文就退回 fallback 或它自己。fallback 在没有中文名时返回，空的再用 value。 */
export function styleLabel(names: object | undefined, value: string, fallback = "") {
  if (names && value in names) {
    const label = (names as Record<string, string>)[value];
    if (label) return label;
  }
  return fallback || value;
}

/** looksLikeSeries 正文达到 2000 字，或至少两处章节且不少于 400 字，就算系列。text 去掉空白后数字数，并找章节标题。 */
export function looksLikeSeries(text: string) {
  const n = Array.from((text || "").trim()).length;
  if (n >= 2000) return true;
  const chapters = (text || "").match(/第[0-9一二三四五六七八九十百千零两]+[章节回]|Chapter\s+\d+/gi);
  return !!chapters && chapters.length >= 2 && n >= 400;
}

/** phaseQueue 按这一段还缺什么，排出要跑的步骤。成片段只有合并。phase 是剧本时只返回改写，是成片时只返回合并。bundle 在定妆时看角色、场景、道具，在分镜时看镜头。 */
export function phaseQueue(phase: Phase, bundle: Bundle | null): PhaseStep[] {
  if (phase === "script") return ["rewrite"];
  if (phase === "cast") {
    const looks = [
      ...(bundle?.characters || []),
      ...(bundle?.scenes || []),
      ...(bundle?.props || []),
    ].filter(/** a 留下仍关联且不是旁白的定妆。 */ (a) => a.linked !== false && !isNarrator(a));
    const steps: PhaseStep[] = [];
    if (looks.length === 0) steps.push("extract");
    if (looks.length === 0 || looks.some(/** a 看 final_prompt 去掉空白后是不是空的。 */ (a) => !(a.final_prompt || "").trim())) steps.push("prompts");
    if (looks.length === 0 || looks.some(/** a 看有没有 image_url。 */ (a) => !a.image_url)) steps.push("stills");
    return steps.length ? steps : ["stills"];
  }
  if (phase === "board") {
    const shots = bundle?.shots || [];
    const steps: PhaseStep[] = [];
    if (!shots.length) steps.push("storyboard");
    if (!shots.length || shots.some(/** s 看 video_prompt 去掉空白后是不是空的。 */ (s) => !(s.video_prompt || "").trim())) steps.push("video_prompts");
    if (!shots.length || shots.some(/** s 看有没有 video_url。 */ (s) => !s.video_url)) steps.push("clips");
    return steps.length ? steps : ["clips"];
  }
  return ["merge"];
}
