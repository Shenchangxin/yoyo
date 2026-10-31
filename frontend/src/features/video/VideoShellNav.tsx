/**
 * 本文件拥有视频壳的主导航。不拥有各栏的内容。
 * 它只切换本地界面栏，不改服务端。
 */
import { BookOpen, Clapperboard, Folder, History, Images, PanelsTopLeft, Puzzle, Sparkles } from "lucide-react";
import { motion } from "motion/react";
import { useCopy } from "../../lib/i18n";
import { DURATION_SHELL, motionTransition, useMotionReduced } from "../../lib/motion";
import { useUI } from "../../lib/store";
import type { VideoPane } from "../../lib/protocol";
import { cn } from "../../lib/utils";

const PRIMARY: { id: Extract<VideoPane, "create" | "studio" | "drama" | "canvas">; icon: typeof Clapperboard }[] = [
  { id: "create", icon: Images },
  { id: "studio", icon: Sparkles },
  { id: "drama", icon: Clapperboard },
  { id: "canvas", icon: PanelsTopLeft },
];

const LIBRARY: { id: Extract<VideoPane, "assets" | "skills" | "plugins" | "tasks">; icon: typeof Clapperboard }[] = [
  { id: "assets", icon: Folder },
  { id: "skills", icon: BookOpen },
  { id: "plugins", icon: Puzzle },
  { id: "tasks", icon: History },
];

/** VideoShellNav 画出创作、短片、短剧、画布，以及素材、技能、插件。窄横条仍保留创作历史入口；宽侧栏的创作历史在项目列表上。props 的 layout 为 bar 时画横条，否则画侧栏；onNavigate 在真正切栏时调用。 */
export function VideoShellNav(props: { layout?: "rail" | "bar"; onNavigate?: () => void }) {
  const bar = props.layout === "bar";
  const copy = useCopy();
  const reduced = useMotionReduced();
  const rail = motionTransition(reduced, DURATION_SHELL);
  const pane = useUI(/** s 取出 videoPane。 */ (s) => s.videoPane);
  const videoBoard = useUI(/** s 取出 videoBoard。 */ (s) => s.videoBoard);
  const openVideoPane = useUI(/** s 取出 openVideoPane。 */ (s) => s.openVideoPane);
  const labels: Record<(typeof PRIMARY)[number]["id"] | (typeof LIBRARY)[number]["id"], string> = {
    create: copy.video.shellCreate,
    studio: copy.video.shellStudio,
    drama: copy.video.shellDrama,
    canvas: copy.video.shellCanvas,
    assets: copy.video.shellAssets,
    skills: copy.video.shellSkills,
    plugins: copy.video.shellPlugins,
    tasks: copy.video.shellTasks,
  };

  /** 短剧在看板打开时也算选中；其余栏只看当前栏。id 是短剧时还看看板是否打开，是短片时只认 studio，其余和当前栏比较。 */
  function active(id: VideoPane) {
    if (id === "drama") return pane === "drama" || videoBoard;
    if (id === "studio") return pane === "studio";
    return pane === id;
  }

  /** 画一行导航。已经停在这一栏就不再打开。item 用 id 判断选中和测试标记，用 icon 画图标。 */
  function row(item: { id: VideoPane; icon: typeof Clapperboard }) {
    const Icon = item.icon;
    const on = active(item.id);
    return (
      <button
        key={item.id}
        type="button"
        data-testid={`video-shell-${item.id}`}
        aria-current={on ? "page" : undefined}
        className={cn(
          "relative flex items-center gap-2 text-left text-[13px] transition-[background-color,color] duration-[var(--duration-fast)] ease-[var(--ease-out)]",
          bar
            ? "h-8 shrink-0 rounded-full px-3"
            : "mb-px min-h-8 w-full rounded-lg px-2.5 py-[6px]",
          on ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground",
        )}
        onClick={() => {
          if (on && item.id === pane) return;
          openVideoPane(item.id);
          props.onNavigate?.();
        }}
      >
        {on && !bar ? (
          <motion.span
            layoutId="video-shell-rail"
            className="absolute left-1 top-2 bottom-2 w-[2px] rounded-full bg-accent"
            transition={rail}
            aria-hidden
          />
        ) : null}
        <Icon className="size-3.5 shrink-0 opacity-70" aria-hidden />
        <span className="truncate">{labels[item.id]}</span>
      </button>
    );
  }

  const library = bar ? LIBRARY : LIBRARY.filter((item) => item.id !== "tasks");

  if (bar) {
    return (
      <nav aria-label={copy.video.shell} data-testid="video-shell-nav" className="flex shrink-0 gap-1 overflow-x-auto border-b border-border px-3 py-2 [scrollbar-width:none]">
        {PRIMARY.map(row)}
        <span aria-hidden className="mx-1 w-px shrink-0 self-stretch bg-border" />
        {library.map(row)}
      </nav>
    );
  }

  return (
    <nav aria-label={copy.video.shell} data-testid="video-shell-nav" className="relative shrink-0 px-1.5 pb-1">
      <div className="px-2.5 pb-1 pt-1.5 text-[11px] font-medium text-muted">{copy.video.shell}</div>
      {PRIMARY.map(row)}
      <div className="mx-2 my-1.5 h-px bg-border" />
      <div className="px-2.5 pb-1 text-[11px] font-medium text-muted">{copy.video.shellLibrary}</div>
      {library.map(row)}
    </nav>
  );
}
