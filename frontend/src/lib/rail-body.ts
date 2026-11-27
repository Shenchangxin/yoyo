/**
 * 拥有：每个表面的左侧栏该画哪一块。
 * 不拥有：侧栏里的按钮和列表。那些在各自的导航组件。技能不是表面，它是会话上的滑出面板，没有自己的侧栏。
 */

import type { Surface } from "./protocol";

/** RailBody 是侧栏中间那一块的种类。sessions 才是会话列表。home 把列表收进历史。 */
export type RailBody = "home" | "sessions" | "video" | "settings" | "learn" | "harness" | "idol";

/** railBody 按表面选出侧栏。会话表面用 home，历史默认收起。surface 为 video 时返回 video，为 settings 时返回 settings，为 pages 时返回 learn，为 harness 时返回 harness，为 idol 时返回 idol。 */
export function railBody(surface: Surface): RailBody {
  switch (surface) {
    case "agent":
      return "home";
    case "video":
      return "video";
    case "settings":
      return "settings";
    case "pages":
      return "learn";
    case "harness":
      return "harness";
    case "idol":
      return "idol";
    default:
      return "sessions";
  }
}
