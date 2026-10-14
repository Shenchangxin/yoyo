import type { MoodMate } from "./types";

export type PresenceMotion = "celebrate" | "flourish" | "bounce" | "nudge" | "spin";

/** One-shot body accent when a face first lands. Keep this sparse so work faces stay still. */
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
