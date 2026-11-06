/**
 * 本文件拥有短片工程、镜头、目录和工序的类型，以及镜头是否就绪的判断。
 * 不拥有制作界面。它不请求服务端，也不改服务端状态。
 */
/** StudioGate 是设定、剧本、声音、画面、合成、交付六步。 */
export type StudioGate = "setup" | "script" | "audio" | "visuals" | "compose" | "delivery";

/** STUDIO_GATES 按界面顺序列出六步工序。 */
export const STUDIO_GATES: StudioGate[] = ["setup", "script", "audio", "visuals", "compose", "delivery"];

/** StudioShot 是一条镜头上的图、视频、字幕、时长和生成任务。 */
export type StudioShot = {
  id?: string;
  image_hash?: string;
  video_hash?: string;
  caption?: string;
  duration?: number;
  motion?: string;
  fit?: string;
  prompt?: string;
  visual?: string;
  stale?: boolean;
  job_id?: string;
  media_url?: string;
};

/** StudioSegment 是剧本里的一段，带种类、口播和画面说明。 */
export type StudioSegment = {
  id?: string;
  kind?: string;
  narration?: string;
  visual?: string;
  duration?: number;
};

/** StudioIdentity 是短片要保住的角色、风格和参考图。 */
export type StudioIdentity = {
  character_id?: string;
  style_id?: string;
  name?: string;
  preserve?: string;
  prohibit?: string;
  ref_hash?: string;
};

/** StudioSnapshot 是一次成片版本的序号和哈希。 */
export type StudioSnapshot = {
  n: number;
  result_hash?: string;
  poster_hash?: string;
  cover_hash?: string;
  note?: string;
  created_at?: string;
};

/** StudioDelivery 是交付包的状态、字幕、封面和文案。 */
export type StudioDelivery = {
  status?: string;
  video_hash?: string;
  subtitle_hash?: string;
  cover_hash?: string;
  copy_hash?: string;
  copy_text?: string;
  note?: string;
  files?: { kind?: string; hash?: string; bytes?: number }[];
};

/** StudioActivity 是短片上的一条活动记录。 */
export type StudioActivity = { id?: string; at?: string; kind?: string; message?: string };
/** StudioIssue 是某道工序上的一条问题。 */
export type StudioIssue = { id?: string; gate?: string; level?: string; message?: string };

/** StudioProject 是短片在制作界面上用到的字段。 */
export type StudioProject = {
  id: string;
  title: string;
  topic_id?: string;
  aspect?: string;
  style?: string;
  preset_id?: string;
  template_id?: string;
  platform?: string;
  tone?: string;
  audience?: string;
  target_duration?: number;
  gate?: string;
  furthest_gate?: string;
  script?: string;
  script_status?: string;
  hook?: string;
  cta?: string;
  segments?: StudioSegment[];
  identity?: StudioIdentity;
  skip_audio?: boolean;
  voice?: string;
  voice_speed?: number;
  voice_engine?: string;
  storyboard?: { shots?: StudioShot[]; size?: string; fit?: string; bgm_volume?: number };
  shots?: StudioShot[];
  narration_hash?: string;
  subtitle_hash?: string;
  bgm_hash?: string;
  /** bgm_name 是当前配乐的名字。 */
  bgm_name?: string;
  /** no_bgm 为真表示操作者选了不用配乐，一键成片不会再配。 */
  no_bgm?: boolean;
  result_hash?: string;
  poster_hash?: string;
  cover_hash?: string;
  covers?: string[];
  cover_urls?: { hash?: string; url?: string }[];
  copy_text?: string;
  versions?: StudioSnapshot[];
  delivery?: StudioDelivery;
  activities?: StudioActivity[];
  issues?: StudioIssue[];
  status?: string;
  error?: string;
  auto_phase?: string;
  auto_error?: string;
  /** task 是正在后台跑的配音或合成，没有就不在。 */
  task?: { kind?: "speak" | "assemble" | string; started_at?: string } | null;
  subtitle_estimated?: boolean;
  media_url?: string;
  poster_url?: string;
  narration_url?: string;
  cover_url?: string;
};

/** StudioCatalog 是设定步能选的角色、风格、预设、模板、声音、平台和语气。 */
export type StudioCatalog = {
  characters?: { id: string; name: string; hint?: string; preserve?: string; prohibit?: string }[];
  styles?: { id: string; name: string; hint?: string }[];
  presets?: { id: string; name: string; hint?: string; aspect?: string }[];
  templates?: { id: string; title: string; hint?: string; tags?: string[] }[];
  voices?: { id: string; name: string }[];
  /** speech 是现在配音会用哪家。route 为空表示还没有能用的语音。 */
  speech?: { route?: string; vendor?: string };
  platforms?: string[];
  tones?: string[];
};

/** gateIndex 返回工序在 STUDIO_GATES 里的位置。认不出时落到 script 的下标。g 空着当 script，再在 STUDIO_GATES 里找下标，找不到就用 1。 */
export function gateIndex(g: string | undefined) {
  const i = STUDIO_GATES.indexOf((g || "script") as StudioGate);
  return i < 0 ? 1 : i;
}

/** shotReady 看镜头有没有图哈希或视频哈希。sh 看有没有 image_hash 或 video_hash。 */
export function shotReady(sh: StudioShot) {
  return !!(sh.image_hash || sh.video_hash);
}

/** shotsForSave 是写回镜头时要送的样子。界面手里的图、视频哈希可能早于刚落地的生图，送回去会被当成手动换画面，所以只给新镜头（没有 id）和 replaceAt 那一镜留画面；过期标记、任务编号和播放地址由服务端管，一律不送。shots 是界面上的镜头。replaceAt 是这次手动换画面的那一镜，没有就传 -1。 */
export function shotsForSave(shots: StudioShot[], replaceAt = -1): StudioShot[] {
  return shots.map(/** sh 去掉服务端管的字段；不是新镜头也不是 replaceAt 那一镜时再去掉画面。i 和 replaceAt 比较。 */ (sh, i) => {
    const { stale: _stale, job_id: _job, media_url: _url, ...rest } = sh;
    if (!sh.id || i === replaceAt) return rest;
    const { image_hash: _image, video_hash: _video, ...words } = rest;
    return words;
  });
}

/** visualsReady 和服务端的 VisualsReady 一样：至少有一镜，每一镜都有图或视频，并且没有过期。shots 逐镜检查。 */
export function visualsReady(shots: StudioShot[]) {
  return shots.length > 0 && shots.every(/** sh 要有画面且没过期。 */ (sh) => shotReady(sh) && !sh.stale);
}

/** projectShots 优先用工程上的 shots，否则用分镜里的镜头。p 没有就返回空数组；有 shots 就用 shots，否则用 storyboard.shots。 */
export function projectShots(p: StudioProject | null | undefined): StudioShot[] {
  if (!p) return [];
  if (p.shots?.length) return p.shots;
  return p.storyboard?.shots || [];
}
