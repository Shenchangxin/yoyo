import { useCallback, useEffect, useState } from "react";
import { Check, ChevronDown, LayoutGrid, Plus } from "lucide-react";
import { toast } from "sonner";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { videoCall } from "../../lib/client";
import { bindCanvasSession, useCanvasHost } from "./canvas-host/session";
import { useUI } from "../../lib/store";

type CanvasSummary = { id: string; title?: string };

const ghost =
  "h-6 min-w-0 max-w-[9.5rem] gap-1 rounded-md border-transparent bg-transparent px-1.5 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground disabled:pointer-events-none disabled:opacity-30 [&_svg]:size-3 [&_svg]:opacity-70";

export function CanvasProjectChip(props: { disabled?: boolean }) {
  const copy = useCopy();
  const projectId = useCanvasHost((s) => s.projectId);
  const title = useCanvasHost((s) => s.title);
  const sessionId = useUI((s) => s.videoThreadId);
  const [list, setList] = useState<CanvasSummary[]>([]);

  const load = useCallback(async () => {
    const env = await videoCall("canvas.http", { method: "GET", path: "canvas-projects" });
    const projects = env?.data?.projects || env?.projects || [];
    setList(Array.isArray(projects) ? projects : []);
  }, []);

  useEffect(() => {
    void load().catch(() => {});
  }, [load]);

  const current = list.find((p) => p.id === projectId);
  const label = current?.title || title || copy.video.canvas;

  async function createBoard() {
    try {
      const env = await videoCall("canvas.http", { method: "POST", path: "canvas-projects", body: { title: copy.video.canvas } });
      const project = env?.data?.project || env?.project;
      if (project?.id) {
        useCanvasHost.setState({ projectId: project.id, title: project.title || copy.video.canvas });
        if (sessionId) await bindCanvasSession(sessionId, project.id);
        await load();
      }
    } catch (e) {
      toast.error(String(e));
    }
  }

  async function pick(p: CanvasSummary) {
    useCanvasHost.setState({ projectId: p.id, title: p.title || copy.video.canvas });
    if (sessionId) void bindCanvasSession(sessionId, p.id);
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button type="button" data-testid="canvas-project-chip" className={cn(ghost, "inline-flex items-center text-foreground")} disabled={props.disabled}>
          <LayoutGrid className="size-3 shrink-0" aria-hidden />
          <span className="truncate">{label}</span>
          <ChevronDown className="size-3 shrink-0 opacity-70" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" side="top" className="w-64">
        <DropdownMenuLabel>{copy.video.canvas}</DropdownMenuLabel>
        {list.map((p) => (
          <DropdownMenuItem key={p.id} onSelect={() => void pick(p)}>
            <span className="min-w-0 flex-1 truncate">{p.title || p.id}</span>
            {p.id === projectId ? <Check className="size-3.5 shrink-0" /> : null}
          </DropdownMenuItem>
        ))}
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={() => void createBoard()}>
          <Plus className="size-3.5" />
          {copy.video.newDrama}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
