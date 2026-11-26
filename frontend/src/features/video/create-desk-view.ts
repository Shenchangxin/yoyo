/**
 * 本文件拥有创作桌此刻该画哪一种视图。只看四个事实：在不在创作桌、有没有绑定的剪辑、成片稿和翻拍稿是否打开。
 * 不拥有这些事实从哪来，也不改任何状态。
 */

/** CreateDeskView 是创作桌的视图：不在创作桌、起步、剪辑、成片、翻拍。 */
export type CreateDeskView = "off" | "launch" | "cut" | "film" | "remake";

/** createDeskView 按优先级给出视图：绑定的剪辑先于成片稿，成片稿先于翻拍稿，都没有时是起步。s 的 createDesk 为假时返回 off。 */
export function createDeskView(s: { createDesk: boolean; cutBound: boolean; filmOpen: boolean; remakeOpen: boolean }): CreateDeskView {
  if (!s.createDesk) return "off";
  if (s.cutBound) return "cut";
  if (s.filmOpen) return "film";
  if (s.remakeOpen) return "remake";
  return "launch";
}