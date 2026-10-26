/**
 * 拥有：立绘方向到身体、脸、嘴、点头和眼睛片段名的选择。片段名与 eidolon-sc 同一套。
 * 不拥有：片段播放、画布和通话请求。这里不调用模型。本地选择，不改服务端。
 */

/** Emotion 立绘心情，与通话情绪同一组：平静、高兴、害羞、惊讶、难过、生气、思考。 */
export type Emotion = "calm" | "happy" | "shy" | "surprised" | "sad" | "angry" | "thinking";

/** Nod 一次点头或摇头。id 用来同一下只播一遍。 */
export type Nod = { kind: "yes" | "no"; id: number };

/** StandDirection 这一帧要呈现的心情、是否在说、是否在看，以及点头。 */
export type StandDirection = {
  mood: Emotion;
  speaking: boolean;
  looking: boolean;
  nod: Nod | null;
};

/** CALM 默认方向：平静、不说、不看、不点头。 */
export const CALM: StandDirection = { mood: "calm", speaking: false, looking: false, nod: null };

/** TrackAnimations 身体、脸、嘴、点头和眼睛各自要播的片段。嘴和点头可以空。 */
export type TrackAnimations = {
  body: string;
  face: string;
  mouth: { name: string; open: boolean } | null;
  nod: { name: string; id: number } | null;
  eye: string | null;
};

const BODY: Record<Emotion, string[]> = {
  calm: ["wait"],
  happy: ["smile1", "smile2"],
  shy: ["shy1", "shy2", "smile2"],
  surprised: ["surp1", "surp2"],
  sad: ["sad1", "sad2"],
  angry: ["anger1", "anger2"],
  thinking: ["think", "think1", "think2", "wait4", "wait"],
};

const FACE: Record<Emotion, string[]> = {
  calm: ["face_wait"],
  happy: ["face_smile", "face_smile1", "face_smile2"],
  shy: ["face_shy", "face_shy2"],
  surprised: ["face_surp"],
  sad: ["face_sad"],
  angry: ["face_anger", "face_anger1", "face_anger2"],
  thinking: ["face_serious"],
};

const LIP: Record<Emotion, string> = {
  calm: "wait",
  happy: "smile",
  shy: "shy",
  surprised: "surp",
  sad: "sad",
  angry: "anger",
  thinking: "serious",
};

/** pickAnimations 从已有片段里按心情挑身体、脸和嘴。减少动效时身体固定 wait。点头按 id 给一次。在看且有 eye_front 才给眼睛。 */
export function pickAnimations(
  direction: StandDirection,
  animations: ReadonlyMap<string, number>,
  reducedMotion: boolean,
): TrackAnimations {
  /** 按候选名取第一条这套服装里有的片段。 */
  const pick = (names: string[]) => names.find((name) => animations.has(name));
  const { mood, nod } = direction;
  const k = LIP[mood];
  const mouth = pick([`lip_${k}`, `lip_${k}1`, `lip_${k}2`, "lip_smile", "lip_wait"]) ?? anyMouth(animations);
  const nodding = nod && pick([nod.kind, `${nod.kind}2`]);
  return {
    body: reducedMotion ? "wait" : (pick(BODY[mood]) ?? "wait"),
    face: pick(FACE[mood]) ?? "blank",
    mouth: mouth ? { name: mouth, open: direction.speaking } : null,
    nod: nod && nodding ? { name: nodding, id: nod.id } : null,
    eye: direction.looking && animations.has("eye_front") ? "eye_front" : null,
  };
}

/** 找第一条时长大于 0、不是口型序列也不是 lip_a 的 lip_ 片段。 */
function anyMouth(animations: ReadonlyMap<string, number>): string | undefined {
  for (const [name, duration] of animations) {
    if (name.startsWith("lip_") && !/_s\d*$/.test(name) && name !== "lip_a" && duration > 0) return name;
  }
  return undefined;
}
