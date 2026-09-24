import { PenLine } from "lucide-react";
import { EmptyState } from "../../components/ui/empty-state";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { DramaStudio } from "./DramaStudio";
import { CanvasStudio } from "./CanvasStudio";

export function VideoWorkshop(props: { sessionId?: string; onNeedSession: () => void; onClose?: () => void }) {
  const copy = useCopy();
  const mode = useUI((s) => s.videoMode);

  return (
    <div className="@container flex h-full min-h-0 flex-col" data-testid="video-workshop">
      {mode === "drama" ? (
        <DramaStudio sessionId={props.sessionId} onNeedSession={props.onNeedSession} onClose={props.onClose} />
      ) : mode === "canvas" ? (
        <CanvasStudio sessionId={props.sessionId} onNeedSession={props.onNeedSession} />
      ) : (
        <EmptyState
          icon={<PenLine className="size-5" />}
          title={copy.video.creative}
          body={copy.video.creativeHint}
        />
      )}
    </div>
  );
}
