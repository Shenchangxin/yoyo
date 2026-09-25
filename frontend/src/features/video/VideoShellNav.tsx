import { Blocks, Clapperboard, History, Images, LibraryBig, PanelsTopLeft, WandSparkles } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import type { VideoPane } from "../../lib/protocol";
import { cn } from "../../lib/utils";

export function VideoShellNav() {
  const copy = useCopy();
  const pane = useUI((s) => s.videoPane);
  const videoBoard = useUI((s) => s.videoBoard);
  const openVideoPane = useUI((s) => s.openVideoPane);
  const items: { id: VideoPane; label: string; icon: typeof WandSparkles }[] = [
    { id: "create", label: copy.video.shellCreate, icon: WandSparkles },
    { id: "drama", label: copy.video.drama, icon: Clapperboard },
    { id: "canvas", label: copy.video.canvas, icon: PanelsTopLeft },
    { id: "assets", label: copy.video.shellAssets, icon: Images },
    { id: "skills", label: copy.video.shellSkills, icon: LibraryBig },
    { id: "plugins", label: copy.video.shellPlugins, icon: Blocks },
    { id: "tasks", label: copy.video.shellTasks, icon: History },
  ];

  return (
    <nav aria-label={copy.video.shell} data-testid="video-shell-nav" className="mt-2 space-y-0.5">
      {items.map((item, i) => {
        const Icon = item.icon;
        const on = item.id === "drama"
          ? pane === "drama" || videoBoard
          : pane === item.id;
        return (
          <div key={item.id}>
            {i === 3 ? <div className="my-1 h-px bg-border/60" /> : null}
            <button
              type="button"
              data-testid={`video-shell-${item.id}`}
              aria-current={on ? "page" : undefined}
              className={cn(
                "flex h-7 w-full items-center gap-2 rounded-lg px-2 text-left text-[12px] transition-colors",
                on ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground",
              )}
              onClick={() => openVideoPane(item.id)}
            >
              <Icon className="size-3.5 shrink-0 opacity-70" aria-hidden />
              <span className="truncate">{item.label}</span>
            </button>
          </div>
        );
      })}
    </nav>
  );
}
