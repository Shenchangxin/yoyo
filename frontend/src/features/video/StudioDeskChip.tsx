import { Sparkles } from "lucide-react";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { useStudioSelection } from "./workshop-store";

const ghost =
  "h-6 min-w-0 max-w-[9.5rem] gap-1 rounded-md border-transparent bg-transparent px-1.5 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground disabled:pointer-events-none disabled:opacity-30 [&_svg]:size-3 [&_svg]:opacity-70";

export function StudioDeskChip(props: { disabled?: boolean }) {
  const copy = useCopy();
  const openStudioDesk = useUI((s) => s.openStudioDesk);

  return (
    <button
      type="button"
      data-testid="studio-open-desk"
      className={cn(ghost, "inline-flex items-center text-foreground")}
      disabled={props.disabled}
      onClick={() => {
        useStudioSelection.getState().setStudioId("");
        openStudioDesk();
      }}
    >
      <Sparkles className="size-3 shrink-0" aria-hidden />
      <span className="truncate">{copy.video.shellStudio}</span>
    </button>
  );
}
