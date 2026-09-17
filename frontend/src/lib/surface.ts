import type { HarnessTab, Lab, Surface } from "./protocol";

export const HARNESS_TABS: readonly HarnessTab[] = ["overview", "propose", "prove", "promote"];

export function surfaceForLab(lab: Lab): Surface {
  return lab === "agent" ? "agent" : "harness";
}

export function tabFromLab(lab: Lab): HarnessTab {
  if (lab === "harbor") return "prove";
  if (lab === "evolve") return "propose";
  if (lab === "harness") return "promote";
  return "overview";
}

export function labFromTab(tab: HarnessTab): Lab {
  if (tab === "prove") return "harbor";
  if (tab === "propose") return "evolve";
  return "harness";
}

export function isHarnessTab(v: string): v is HarnessTab {
  return (HARNESS_TABS as readonly string[]).includes(v);
}
