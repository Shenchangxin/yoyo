/**
 * 拥有：每个表面的左侧栏该画哪一块。
 * 不拥有：侧栏里的按钮和列表。那些在各自的导航组件。
 */

import type { Surface } from "./protocol";

/** RailBody 是侧栏中间那一块的种类。sessions 才是会话列表。 */
export type RailBody = "sessions" | "video" | "settings" | "learn" | "skills" | "harness" | "idol";

/** railBody 按表面选出侧栏。会话列表只属于会话表面。surface 为 video 时返回 video，为 settings 时返回 settings，为 pages 时返回 learn，为 skills 时返回 skills，为 harness 时返回 harness，为 idol 时返回 idol，否则返回 sessions。 */
export function railBody(surface: Surface): RailBody {
  switch (surface) {
    case "video":
      return "video";
    case "settings":
      return "settings";
    case "pages":
      return "learn";
    case "skills":
      return "skills";
    case "harness":
      return "harness";
    case "idol":
      return "idol";
    default:
      return "sessions";
  }
}
