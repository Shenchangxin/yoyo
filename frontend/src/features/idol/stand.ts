/** Call emotions, the same set eidolon uses for a stand: calm, happy, shy, surprised, sad, angry, thinking. */
const MOODS = new Set(["calm", "happy", "shy", "surprised", "sad", "angry", "thinking"]);

export function standMood(raw: string | undefined): string {
  const s = String(raw || "").trim().toLowerCase();
  return MOODS.has(s) ? s : "calm";
}

/** Head nod. Angry, sad, and surprised shake; thinking holds still; the rest nod. */
export function nodFor(mood: string): "yes" | "no" | null {
  const m = standMood(mood);
  if (m === "thinking") return null;
  if (m === "angry" || m === "sad" || m === "surprised") return "no";
  return "yes";
}

/**
 * Mood-mate face for the desktop pet, who has no portrait.
 * Listening looks over, thinking uses the think face, speaking follows the mood.
 */
export function petFace(phase: string, mood: string): string {
  const m = standMood(mood);
  if (phase === "thinking") return "30";
  if (phase === "listening" || phase === "dialing") {
    if (m === "sad") return "12";
    if (m === "shy") return "08";
    if (m === "angry") return "38";
    return "35";
  }
  if (phase === "speaking") {
    switch (m) {
      case "happy": return "46";
      case "shy": return "08";
      case "surprised": return "35";
      case "sad": return "12";
      case "angry": return "38";
      case "thinking": return "30";
      default: return "39";
    }
  }
  return "02";
}
