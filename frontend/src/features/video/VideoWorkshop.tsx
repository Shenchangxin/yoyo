import { PenLine } from "lucide-react";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { isYingcePane, type VideoProject } from "../../lib/protocol";
import { CanvasStudio } from "./CanvasStudio";
import { DramaStudio } from "./DramaStudio";
import { StudioWorkbench } from "./StudioWorkbench";
import { VideoHistory } from "./VideoHistory";
import { useStudioSelection } from "./workshop-store";

export function VideoWorkshop(props: {
  sessionId?: string;
  onNeedSession: () => void | Promise<string | void>;
  onClose?: () => void;
  projects?: VideoProject[];
  canvasProjectId?: string;
  dramaId?: string;
  onOpenProject?: (p: VideoProject) => void;
  onOpenCut?: (id: string) => void;
  onRefreshHistory?: () => void;
}) {
  const copy = useCopy();
  const pane = useUI((s) => s.videoPane);
  const videoBoard = useUI((s) => s.videoBoard);
  const setVideoBoard = useUI((s) => s.setVideoBoard);
  const studioId = useStudioSelection((s) => s.studioId);
  const showBoard = videoBoard;
  const showYingce = isYingcePane(pane) || showBoard;

  return (
    <div
      className="@container flex h-full min-h-0 flex-col"
      data-testid="video-workshop"
      data-pane={showBoard ? "drama-board" : pane}
    >
      {pane === "studio" ? (
        <StudioWorkbench
          sessionId={props.sessionId}
          projects={props.projects}
          onNeedSession={props.onNeedSession}
          onOpenProject={props.onOpenProject}
          onRefreshHistory={props.onRefreshHistory}
          onClose={props.onClose}
        />
      ) : pane === "tasks" && !showBoard ? (
        <div className="flex h-full min-h-0 flex-col" data-testid="canvas-studio-shell" data-pane="tasks">
          <VideoHistory
            projects={props.projects || []}
            canvasProjectId={props.canvasProjectId}
            dramaId={props.dramaId}
            studioId={studioId}
            sessionId={props.sessionId}
            onOpenProject={props.onOpenProject}
            onOpenCut={props.onOpenCut}
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
      ) : (
        <EmptyState
          icon={<PenLine className="size-5" />}
          title={copy.video.workshopUnavailable}
          body={copy.video.workshopUnavailableHint}
          action={
            <Button size="sm" variant="lift" onClick={() => useUI.getState().openVideoPane("create")}>
              {copy.video.workshopBackCreate}
            </Button>
          }
        />
      )}
    </div>
  );
}
