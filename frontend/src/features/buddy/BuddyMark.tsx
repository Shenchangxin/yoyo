// 本文件拥有 agent 的小字形：有图标时把清洗过的 SVG 当遮罩，跟着文字颜色着色；没有图标时画首字母，外面一圈细线。
// 不拥有图标从哪来：后端下载、清洗后以 data URI 给出。Buddy 的侧栏、总览和详情都用它。

import type { CSSProperties } from "react";
import type { BuddyView } from "../../lib/client";
import { cn } from "../../lib/utils";

/** maskStyle 把 data URI 变成遮罩样式。图标居中、等比缩放。 */
function maskStyle(uri: string): CSSProperties {
  const image = `url("${uri}")`;
  return {
    maskImage: image,
    WebkitMaskImage: image,
    maskSize: "contain",
    WebkitMaskSize: "contain",
    maskRepeat: "no-repeat",
    WebkitMaskRepeat: "no-repeat",
    maskPosition: "center",
    WebkitMaskPosition: "center",
  };
}

/** BuddyMark 画 v 的字形，默认 size-5。className 可改尺寸和字号。只认 data:image/svg+xml 的图标。 */
export function BuddyMark({ v, className }: { v: Pick<BuddyView, "name" | "icon">; className?: string }) {
  if (v.icon?.startsWith("data:image/svg+xml")) {
    return <span aria-hidden className={cn("inline-block size-5 shrink-0 bg-current", className)} style={maskStyle(v.icon)} />;
  }
  const letter = (v.name || "?").trim().charAt(0).toUpperCase() || "?";
  return (
    <span
      aria-hidden
      className={cn("grid size-5 shrink-0 place-items-center rounded-full border border-border text-ui-badge font-semibold leading-none", className)}
    >
      {letter}
    </span>
  );
}
