// 本文件拥有小写 yoyo 字标和空状态用的静置印记。不拥有旁边的导航或标题。ThreadRail、BootSkeleton、SkillsWorkspace 用 YoyoMark；About 和 Transcript 用 MarkWell。

import { cn } from "../../lib/utils";

type MarkProps = {
  className?: string;
  // compact 给窄边栏用，字号 15px。页面印记不传它，字号 17px，仍然是裁切过的字标。
  compact?: boolean;
};

/** YoyoMark 用界面字体画小写 yoyo。className 传给 cn。compact 为真时才继续。 */
export function YoyoMark({ className, compact }: MarkProps) {
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
