/**
 * 拥有：通话情绪到立绘心情、点头方向和桌面宠物表情编号的对照。
 * 不拥有：Spine 片段选择、画布绘制和通话请求。这里不调用模型。本地对照，不改服务端。
 */

const MOODS = new Set(["calm", "happy", "shy", "surprised", "sad", "angry", "thinking"]);

/** standMood 只接受 calm、happy、shy、surprised、sad、angry、thinking，和立绘用的是同一组，其余收成 calm。 */
export function standMood(raw: string | undefined): string {
  const s = String(raw || "").trim().toLowerCase();
  return MOODS.has(s) ? s : "calm";
}

/** nodFor 按情绪点头：angry、sad、surprised 摇头，thinking 不动，其余点头。 */
export function nodFor(mood: string): "yes" | "no" | null {
  const m = standMood(mood);
  if (m === "thinking") return null;
  if (m === "angry" || m === "sad" || m === "surprised") return "no";
  return "yes";
}

/** petFace 没有立绘的桌面宠物所用的表情编号。听或拨号朝向一侧，思考用思考脸，说话跟着心情，其余用静息脸。 */
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
