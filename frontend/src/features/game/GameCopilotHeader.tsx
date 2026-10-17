import { Gamepad2 } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { useGameSelection } from "../video/workshop-store";
import { copilotHint, deskPage, deskStarters, pageLabel } from "./game-desk";

export function GameCopilotHeader(props: {
  onPrompt?: (text: string) => void;
}) {
  const copy = useCopy();
  const g = copy.video.gameDesk;
  const title = useGameSelection((s) => s.title);
  const gameId = useGameSelection((s) => s.gameId);
  const focus = useGameSelection((s) => s.focus);
  const page = deskPage(null, focus || "play");
  const chips = deskStarters(page, { id: gameId, title, intent: "" }, g);

  return (
    <header className="flex shrink-0 flex-col gap-2 border-b border-border/70 px-3 py-2" data-testid="game-copilot-header">
      <div className="flex items-start gap-2">
        <Gamepad2 className="mt-0.5 size-3.5 shrink-0 opacity-70" aria-hidden />
        <div className="min-w-0 flex-1">
          <p className="text-[11px] font-medium text-muted">{copy.video.gameCopilot} · {pageLabel(page, g)}</p>
          <p className="truncate text-[13px] font-medium tracking-[-0.02em]">
            {title || g.copilotBound}
          </p>
          <p className="text-[12px] leading-[1.45] text-muted">{copilotHint(page, g)}</p>
        </div>
      </div>
      {props.onPrompt ? (
        <div className="flex flex-wrap gap-1.5 pl-6">
          {chips.map((chip) => (
            <button
              key={chip.label}
              type="button"
              className="rounded-md bg-lift px-2 py-1 text-[11px] text-foreground hover:bg-lift/80"
              data-testid={`game-copilot-chip-${chip.label}`}
              onClick={() => props.onPrompt?.(chip.text)}
            >
              {chip.label}
            </button>
          ))}
        </div>
      ) : null}
    </header>
  );
}
