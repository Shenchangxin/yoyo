export type StudioGate = "setup" | "script" | "audio" | "visuals" | "compose" | "delivery";

export const STUDIO_GATES: StudioGate[] = ["setup", "script", "audio", "visuals", "compose", "delivery"];

export type StudioShot = {
  id?: string;
  image_hash?: string;
  video_hash?: string;
  caption?: string;
  duration?: number;
  motion?: string;
  prompt?: string;
  visual?: string;
  stale?: boolean;
  job_id?: string;
  media_url?: string;
};

export type StudioSegment = {
  id?: string;
  kind?: string;
  narration?: string;
  visual?: string;
  duration?: number;
};

export type StudioIdentity = {
  character_id?: string;
  style_id?: string;
  name?: string;
  preserve?: string;
  prohibit?: string;
  ref_hash?: string;
};

export type StudioSnapshot = {
  n: number;
  result_hash?: string;
  poster_hash?: string;
  cover_hash?: string;
  note?: string;
  created_at?: string;
};

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

export type StudioActivity = { id?: string; at?: string; kind?: string; message?: string };
export type StudioIssue = { id?: string; gate?: string; message?: string };

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
  storyboard?: { shots?: StudioShot[]; size?: string };
  shots?: StudioShot[];
  narration_hash?: string;
  subtitle_hash?: string;
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
  media_url?: string;
  poster_url?: string;
  narration_url?: string;
  cover_url?: string;
};

export type StudioCatalog = {
  characters?: { id: string; name: string; hint?: string; preserve?: string; prohibit?: string }[];
  styles?: { id: string; name: string; hint?: string }[];
  presets?: { id: string; name: string; hint?: string; aspect?: string }[];
  templates?: { id: string; title: string; hint?: string; tags?: string[] }[];
  voices?: { id: string; name: string }[];
  platforms?: string[];
  tones?: string[];
};

export function gateIndex(g: string | undefined) {
  const i = STUDIO_GATES.indexOf((g || "script") as StudioGate);
  return i < 0 ? 1 : i;
}

export function shotReady(sh: StudioShot) {
  return !!(sh.image_hash || sh.video_hash);
}

export function projectShots(p: StudioProject | null | undefined): StudioShot[] {
  if (!p) return [];
  if (p.shots?.length) return p.shots;
  return p.storyboard?.shots || [];
}
