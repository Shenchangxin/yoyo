import { LayoutGrid, PenLine } from "lucide-react";
import { EmptyState } from "../../components/ui/empty-state";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { DramaStudio } from "./DramaStudio";

export function VideoWorkshop(props: { sessionId?: string; onNeedSession: () => void }) {
  const copy = useCopy();
  const mode = useUI((s) => s.videoMode);

  return (
    <div className="@container flex h-full min-h-0 flex-col" data-testid="video-workshop">
      {mode === "drama" ? (
        <DramaStudio sessionId={props.sessionId} onNeedSession={props.onNeedSession} />
      ) : (
        <EmptyState
          icon={mode === "canvas" ? <LayoutGrid className="size-5" /> : <PenLine className="size-5" />}
          title={mode === "canvas" ? copy.video.canvas : copy.video.creative}
          body={copy.video.later}
        />
      )}
    </div>
  );
}
