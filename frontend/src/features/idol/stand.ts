/**
 * 拥有：通话情绪到立绘心情、点头方向和桌面宠物表情编号的对照。
 * 不拥有：Spine 片段选择、画布绘制和通话请求。这里不调用模型。本地对照，不改服务端。
 */

const MOODS = new Set(["calm", "happy", "shy", "surprised", "sad", "angry", "thinking"]);

/** standMood 只接受 calm、happy、shy、surprised、sad、angry、thinking，和立绘用的是同一组，其余收成 calm。raw 为假时用空字符串，转成字符串，去掉首尾空白，转成小写后，作为分支结果，传给 MOODS.has。 */
export function standMood(raw: string | undefined): string {
  const s = String(raw || "").trim().toLowerCase();
  return MOODS.has(s) ? s : "calm";
}

/** 语气提示里的英文词根对应的情绪，按顺序找，先命中的算。和服务端 CueEmotion 是同一张表。 */
const CUE_MOODS: [string, string[]][] = [
  ["surprised", ["surpris", "shock", "gasp", "startl", "astonish", "panic"]],
  ["shy", ["shy", "embarrass", "bashful", "blush", "fluster", "timid", "sheepish"]],
  ["angry", ["angr", "annoy", "sulk", "pout", "irrit", "frustrat", "huff", "indignant"]],
  ["sad", ["sad", "disappoint", "worr", "lonel", "tear", "sigh", "gloom", "hurt", "upset"]],
  ["thinking", ["think", "hesita", "ponder", "wonder", "unsure", "mumbl", "puzzl"]],
  ["happy", ["happ", "excit", "cheer", "laugh", "giggl", "smil", "joy", "bright", "playful", "teas", "proud", "delight"]],
  ["calm", ["gentl", "soft", "calm", "whisper", "sleepy", "quiet", "warm"]],
];

/** cueMood 把她写的语气提示（例如 shyly）归到立绘的七种情绪；认不出或没有时返回空串，立绘保持原来的心情。 */
export function cueMood(cue: string | undefined): string {
  const c = String(cue || "").trim().toLowerCase();
  if (!c) return "";
  for (const [mood, roots] of CUE_MOODS) {
    if (roots.some((r) => c.includes(r))) return mood;
  }
  return "";
}

/** nodFor 按情绪点头：angry、sad、surprised 摇头，thinking 不动，其余点头。mood 传给 standMood。 */
export function nodFor(mood: string): "yes" | "no" | null {
  const m = standMood(mood);
  if (m === "thinking") return null;
  if (m === "angry" || m === "sad" || m === "surprised") return "no";
  return "yes";
}

/** petFace 没有立绘的桌面宠物所用的表情编号。听或拨号朝向一侧，思考用思考脸，说话跟着心情，其余用静息脸。phase 为 thinking 时返回 30。mood 传给 standMood。 */
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
