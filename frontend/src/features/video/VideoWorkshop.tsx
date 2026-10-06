import { PenLine } from "lucide-react";
import { EmptyState } from "../../components/ui/empty-state";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { isYingcePane, type VideoProject } from "../../lib/protocol";
import { CanvasStudio } from "./CanvasStudio";
import { DramaStudio } from "./DramaStudio";
import { VideoHistory } from "./VideoHistory";

export function VideoWorkshop(props: {
  sessionId?: string;
  onNeedSession: () => void | Promise<string | void>;
  onClose?: () => void;
  projects?: VideoProject[];
  canvasProjectId?: string;
  dramaId?: string;
  onOpenProject?: (p: VideoProject) => void;
  onRefreshHistory?: () => void;
}) {
  const copy = useCopy();
  const pane = useUI((s) => s.videoPane);
  const mode = useUI((s) => s.videoMode);
  const videoBoard = useUI((s) => s.videoBoard);
  const setVideoBoard = useUI((s) => s.setVideoBoard);
  const showBoard = videoBoard;
  const showYingce = isYingcePane(pane) || showBoard;

  return (
    <div
      className="@container flex h-full min-h-0 flex-col"
      data-testid="video-workshop"
      data-pane={showBoard ? "drama-board" : pane}
    >
      {pane === "tasks" && !showBoard ? (
        <div className="flex h-full min-h-0 flex-col" data-testid="canvas-studio-shell" data-pane="tasks">
          <VideoHistory
            projects={props.projects || []}
            canvasProjectId={props.canvasProjectId}
            dramaId={props.dramaId}
            sessionId={props.sessionId}
            onOpenProject={props.onOpenProject}
            onRefresh={props.onRefreshHistory}
          />
        </div>
      ) : showYingce ? (
        <div
          className={showBoard ? "drama-desk" : "flex h-full min-h-0 flex-col"}
          data-testid={showBoard ? "drama-desk" : undefined}
        >
          <div className={showBoard ? "drama-desk-chat" : "flex min-h-0 flex-1 flex-col overflow-hidden"}>
            <CanvasStudio sessionId={props.sessionId} onNeedSession={props.onNeedSession} onClose={props.onClose} />
          </div>
          {showBoard ? (
            <div className="drama-desk-board">
              <DramaStudio sessionId={props.sessionId} onNeedSession={props.onNeedSession} onClose={() => setVideoBoard(false)} />
            </div>
          ) : null}
        </div>
      ) : mode === "creative" ? (
        <EmptyState
          icon={<PenLine className="size-5" />}
          title={copy.video.creative}
          body={copy.video.creativeHint}
        />
      ) : (
        <EmptyState
          icon={<PenLine className="size-5" />}
          title={copy.video.creative}
          body={copy.video.later}
        />
      )}
    </div>
  );
}
