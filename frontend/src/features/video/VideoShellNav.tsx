import { BookOpen, Clapperboard, Folder, Gamepad2, History, Images, PanelsTopLeft, Puzzle, Sparkles } from "lucide-react";
import { motion } from "motion/react";
import { useCopy } from "../../lib/i18n";
import { DURATION_SHELL, motionTransition, useMotionReduced } from "../../lib/motion";
import { useUI } from "../../lib/store";
import type { VideoPane } from "../../lib/protocol";
import { cn } from "../../lib/utils";
import { useGameSelection, useStudioSelection } from "./workshop-store";

const PRIMARY: { id: Extract<VideoPane, "create" | "studio" | "drama" | "canvas" | "game">; icon: typeof Clapperboard }[] = [
  { id: "create", icon: Images },
  { id: "studio", icon: Sparkles },
  { id: "drama", icon: Clapperboard },
  { id: "canvas", icon: PanelsTopLeft },
  { id: "game", icon: Gamepad2 },
];

const LIBRARY: { id: Extract<VideoPane, "assets" | "skills" | "plugins" | "tasks">; icon: typeof Clapperboard }[] = [
  { id: "assets", icon: Folder },
  { id: "skills", icon: BookOpen },
  { id: "plugins", icon: Puzzle },
  { id: "tasks", icon: History },
];

export function VideoShellNav() {
  const copy = useCopy();
  const reduced = useMotionReduced();
  const rail = motionTransition(reduced, DURATION_SHELL);
  const pane = useUI((s) => s.videoPane);
  const videoBoard = useUI((s) => s.videoBoard);
  const openVideoPane = useUI((s) => s.openVideoPane);
  const labels: Record<(typeof PRIMARY)[number]["id"] | (typeof LIBRARY)[number]["id"], string> = {
    create: copy.video.shellCreate,
    studio: copy.video.shellStudio,
    drama: copy.video.shellDrama,
    canvas: copy.video.shellCanvas,
    game: copy.video.shellGame,
    assets: copy.video.shellAssets,
    skills: copy.video.shellSkills,
    plugins: copy.video.shellPlugins,
    tasks: copy.video.shellTasks,
  };

  function active(id: VideoPane) {
    if (id === "drama") return pane === "drama" || videoBoard;
    if (id === "studio") return pane === "studio";
    return pane === id;
  }

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
          "relative mb-px flex min-h-8 w-full items-center gap-2 rounded-lg px-2.5 py-[6px] text-left text-[13px] transition-[background-color,color] duration-[var(--duration-fast)] ease-[var(--ease-out)]",
          on ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground",
        )}
        onClick={() => {
          if (item.id === "studio") useStudioSelection.getState().setStudioId("");
          if (item.id === "game" && pane !== "game") useGameSelection.getState().setGameId("");
          openVideoPane(item.id);
        }}
      >
        {on ? (
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

  return (
    <nav aria-label={copy.video.shell} data-testid="video-shell-nav" className="relative shrink-0 px-1.5 pb-1">
      <div className="px-2.5 pb-1 pt-1.5 text-[11px] font-medium text-muted">{copy.video.shell}</div>
      {PRIMARY.map(row)}
      <div className="mx-2 my-1.5 h-px bg-border" />
      <div className="px-2.5 pb-1 text-[11px] font-medium text-muted">{copy.video.shellLibrary}</div>
      {LIBRARY.map(row)}
    </nav>
  );
}
