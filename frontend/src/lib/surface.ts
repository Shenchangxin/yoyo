import type { HarnessTab, Lab, Surface } from "./protocol";

/**
 * 拥有：实验室、Harness 页签和表面之间的对应。
 * 不拥有：这些表面的界面，以及路由历史。
 */

/** HARNESS_TABS Harness 四个页签的固定顺序。 */
export const HARNESS_TABS: readonly HarnessTab[] = ["overview", "propose", "prove", "promote"];

/** surfaceForLab 智能体实验室对应智能体表面，其余实验室对应 Harness 表面。lab 为 agent 时返回 agent，否则返回 harness。 */
export function surfaceForLab(lab: Lab): Surface {
  return lab === "agent" ? "agent" : "harness";
}

/** tabFromLab 把实验室收成页签。harbor 为证明，evolve 为提议，harness 为提升，其余为总览。lab 为 harbor 时返回 prove，为 evolve 时返回 propose，为 harness 时返回 promote，否则返回 overview。 */
export function tabFromLab(lab: Lab): HarnessTab {
  if (lab === "harbor") return "prove";
  if (lab === "evolve") return "propose";
  if (lab === "harness") return "promote";
  return "overview";
}

/** labFromTab 证明页回到 harbor，提议页回到 evolve，其余页签回到 harness 实验室。tab 为 prove 时返回 harbor，为 propose 时返回 evolve，否则返回 harness。 */
export function labFromTab(tab: HarnessTab): Lab {
  if (tab === "prove") return "harbor";
  if (tab === "propose") return "evolve";
  return "harness";
}

/** isHarnessTab 判断字符串是不是合法页签。v 传给 includes。 */
export function isHarnessTab(v: string): v is HarnessTab {
  return (HARNESS_TABS as readonly string[]).includes(v);
}
