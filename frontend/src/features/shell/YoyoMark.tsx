// 本文件拥有小写 yoyo 字标和空状态用的静置印记。不拥有旁边的导航或标题。左上角传 live，焦点在两个 yo 之间移动。ThreadRail、BootSkeleton、SkillsWorkspace 用 YoyoMark；About 和 Transcript 用 MarkWell。

import { useLayoutEffect, useRef, useState } from "react";
import { cn } from "../../lib/utils";
import TrueFocus from "./TrueFocus";

const MARK_SCALE_X = 1.72;
const MARK_SCALE_Y = 1.08;

type MarkProps = {
  className?: string;
  // compact 给窄边栏用，字号 15px。页面印记不传它，字号 17px，仍然是裁切过的字标。
  compact?: boolean;
  // live 只给左上角：把 yoyo 拆成两个 yo，焦点在其间移动。空状态印记不传。
  live?: boolean;
};

/** YoyoMark 用界面字体画小写 yoyo。className 传给 cn。compact 为真时字号更小。live 为真时在两个 yo 之间跑焦点。 */
export function YoyoMark({ className, compact, live }: MarkProps) {
  if (live) return <LiveMark className={className} />;
  return (
    <span
      className={cn(
        "yoyo-mark inline-flex shrink-0 items-center font-semibold tracking-[-0.04em] text-foreground",
        compact ? "text-[15px] leading-none" : "text-[17px] leading-none",
        className,
      )}
    >
      yoyo
    </span>
  );
}

/** LiveMark 把字标横向拉开，高度只加一点，并按拉开后的尺寸占位，避免两个 yo 叠在一起。 */
function LiveMark({ className }: { className?: string }) {
  const innerRef = useRef<HTMLSpanElement>(null);
  const [box, setBox] = useState({ w: 0, h: 0 });

  useLayoutEffect(() => {
    const el = innerRef.current;
    if (!el) return;
    const measure = () => setBox({ w: el.offsetWidth, h: el.offsetHeight });
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  return (
    <span
      className={cn("yoyo-mark yoyo-mark-live shrink-0 text-foreground", className)}
      role="img"
      aria-label="yoyo"
      style={box.w > 0 ? { width: Math.ceil(box.w * MARK_SCALE_X), height: Math.ceil(box.h * MARK_SCALE_Y) } : undefined}
    >
      <span
        ref={innerRef}
        className="yoyo-mark-live-inner"
        style={{ transform: `scaleX(${MARK_SCALE_X}) scaleY(${MARK_SCALE_Y})` }}
      >
        <TrueFocus
          sentence="yo yo"
          manualMode={false}
          focusStyle="brackets"
          easing="smooth"
          order="sequential"
          blurAmount={0.7}
          focusScale={1}
          unfocusedOpacity={0.5}
          borderColor="var(--accent)"
          glowColor="color-mix(in srgb, var(--accent) 40%, transparent)"
          cornerSize={5}
          borderWidth={1.25}
          frameGap={3}
          animationDuration={0.48}
          pauseBetweenAnimations={1.15}
        />
      </span>
    </span>
  );
}

/** MarkWell 是空状态里的静置印记，不是海报。它对读屏隐藏。className 传给 cn。markClassName 写到 YoyoMark 的 className。 */
export function MarkWell({ className, markClassName }: { className?: string; markClassName?: string }) {
  return (
    <span className={cn("mark-well inline-flex", className)} aria-hidden>
      <span className="mark-well-core">
        <YoyoMark className={markClassName} />
      </span>
    </span>
  );
}
