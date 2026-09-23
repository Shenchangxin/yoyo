import { Check, ChevronDown, Clapperboard, LayoutGrid, PenLine } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import type { VideoMode } from "../../lib/protocol";

const ghost =
  "h-6 min-w-0 max-w-[9.5rem] gap-1 rounded-lg border-transparent bg-transparent px-1.5 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground disabled:pointer-events-none disabled:opacity-30 [&_svg]:size-3 [&_svg]:opacity-70";

export function VideoModeSwitch(props: { disabled?: boolean }) {
  const copy = useCopy();
  const mode = useUI((s) => s.videoMode);
  const setMode = useUI((s) => s.setVideoMode);
  const modes: { id: VideoMode; label: string; hint: string; icon: typeof Clapperboard; ready: boolean }[] = [
    { id: "drama", label: copy.video.drama, hint: copy.video.dramaHint, icon: Clapperboard, ready: true },
    { id: "canvas", label: copy.video.canvas, hint: copy.video.canvasHint, icon: LayoutGrid, ready: false },
    { id: "creative", label: copy.video.creative, hint: copy.video.creativeHint, icon: PenLine, ready: false },
  ];
  const current = modes.find((m) => m.id === mode) || modes[0];
  const Icon = current.icon;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          data-testid="video-mode-switch"
          className={cn(ghost, "inline-flex items-center text-foreground")}
          aria-label={copy.video.modes}
          disabled={props.disabled}
        >
          <Icon className="size-3 shrink-0" aria-hidden />
          <span className="truncate">{current.label}</span>
          <ChevronDown className="size-3 shrink-0 opacity-70" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" side="top" className="w-64">
        {modes.map((m) => (
          <DropdownMenuItem
            key={m.id}
            className="items-start"
            onSelect={() => setMode(m.id)}
          >
            <m.icon className="mt-0.5 size-3.5 shrink-0 opacity-70" aria-hidden />
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="flex items-center gap-1.5">
                {m.label}
                {!m.ready ? <span className="text-[10.5px] font-normal text-muted/70">{copy.video.soon}</span> : null}
              </span>
              <span className="text-[11px] text-muted">{m.hint}</span>
            </span>
            {mode === m.id ? <Check className="mt-0.5 size-3.5 shrink-0" /> : null}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
