/** How the stand should look, and which of an outfit's clips show it. Same tracks as eidolon-sc. */

export type Emotion = "calm" | "happy" | "shy" | "surprised" | "sad" | "angry" | "thinking";

export type Nod = { kind: "yes" | "no"; id: number };

export type StandDirection = {
  mood: Emotion;
  speaking: boolean;
  looking: boolean;
  nod: Nod | null;
};

export const CALM: StandDirection = { mood: "calm", speaking: false, looking: false, nod: null };

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

export function pickAnimations(
  direction: StandDirection,
  animations: ReadonlyMap<string, number>,
  reducedMotion: boolean,
): TrackAnimations {
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

function anyMouth(animations: ReadonlyMap<string, number>): string | undefined {
  for (const [name, duration] of animations) {
    if (name.startsWith("lip_") && !/_s\d*$/.test(name) && name !== "lip_a" && duration > 0) return name;
  }
  return undefined;
}
