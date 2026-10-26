/** 拥有表情刚落下时的一次性身体动作。干活的脸保持不动，所以这里很疏。不拥有情绪球，也不另开模型循环。本地选择，不改服务端。 */
import type { MoodMate } from "./types";

/** PresenceMotion 是庆祝、挥手、跳、轻推或转。 */
export type PresenceMotion = "celebrate" | "flourish" | "bounce" | "nudge" | "spin";

/** entryMotion 在一张脸刚落下时给一次性动作，干活的脸不加。id 用来做分支。 */
export function entryMotion(id: string): PresenceMotion | null {
  switch (id) {
    case "33":
      return "celebrate";
    case "47":
    case "10":
    case "23":
      return "flourish";
    case "31":
    case "13":
    case "07":
    case "01":
    case "08":
    case "28":
    case "49":
      return "bounce";
    case "19":
    case "25":
    case "29":
    case "48":
    case "22":
      return "nudge";
    case "21":
    case "34":
    case "38":
      return "nudge";
    default:
      return null;
  }
}

/** playMotion 把动作交到情绪球上，缺方法时用跳或转顶上。mate 读取 flourish、nudge，调用 celebrate，调用 flourish，调用 nudge，调用 spin，调用 bounce，为假时走另一支。motion 为假时走另一支，用来做分支。 */
export function playMotion(mate: MoodMate | null | undefined, motion: PresenceMotion | null | undefined): void {
  if (!mate || !motion) return;
  switch (motion) {
    case "celebrate":
      mate.celebrate(1);
      return;
    case "flourish":
      if (typeof mate.flourish === "function") mate.flourish();
      else {
        mate.nudge?.();
        mate.spin(1);
      }
      return;
    case "bounce":
      mate.bounce();
      return;
    case "nudge":
      if (typeof mate.nudge === "function") mate.nudge();
      else mate.bounce();
      return;
    case "spin":
      mate.spin(1);
  }
}
