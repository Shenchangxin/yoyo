// @ts-nocheck
import { Link } from "react-router";
import { CanvasWorkspaceAssetPanel } from "./canvas-workspace-asset-panel";
import { CanvasWorkspaceHistoryPanel } from "./canvas-workspace-history-panel";
import type { InsertAssetPayload } from "./asset-picker-modal";
import { useDeferredValue, useState } from "react";
import { Button, Grid } from "antd";
import { Home, PanelLeftClose, PanelLeftOpen, Layers, Images, ListChecks, History, X } from "lucide-react";
import { isYoyoCanvasHost, requestHostNavigation } from "@/features/video/canvas-host/design-system";
import { useCopy } from "@/lib/i18n";
import { AppDrawer } from "@yingce/components/ui/product/app-drawer";
import { searchCanvasNodes } from "@yingce/lib/canvas/canvas-node-search";
import { useEffectiveConfig } from "@yingce/stores/use-config-store";
import { canvasDockStyle } from "@yingce/lib/canvas/canvas-aceternity-style";
import { canvasThemes } from "@yingce/lib/canvas-theme";
import { useActiveTheme } from "@yingce/stores/canvas/use-canvas-theme-store";
import type { CanvasNodeData } from "@yingce/types/canvas";
import type { GenerationTask } from "@yingce/services/api/task-center";
import { useCanvasWorkspaceTasks } from "@yingce/pages/canvas/use-canvas-workspace-tasks";
import { CanvasWorkspaceNodeListPanel } from "./canvas-workspace-node-list-panel";
import { CanvasWorkspaceTaskPanel } from "./canvas-workspace-task-panel";
import "./canvas-workspace-panel.css";

export function CanvasWorkspacePanel({
    projectId,
    nodes,
    selectedNodeIds,
    onClose,
    open,
    onOpen,
    onInsertAssets,
    onFocus,
    onAssets,
    onProjectAssets,
    onCancelTask,
}: {
    projectId: string;
    nodes: CanvasNodeData[];
    selectedNodeIds: Set<string>;
    onClose: () => void;
    open: boolean;
    onOpen: () => void;
    onInsertAssets: (payloads: InsertAssetPayload[]) => Promise<unknown>;
    onFocus: (id: string) => void;
    onAssets: () => void;
    onProjectAssets?: () => void;
    onCancelTask: (task: GenerationTask) => void;
}) {
    const [tab, setTab] = useState("nodes");
    const board = useCopy().video.canvasBoard;
    const hosted = isYoyoCanvasHost();
    const theme = canvasThemes[useActiveTheme()];
    const [query, setQuery] = useState("");
    const history = tab === "history";
    const deferredQuery = useDeferredValue(query);
    const config = useEffectiveConfig();
    const screens = Grid.useBreakpoint();
    const tasks = useCanvasWorkspaceTasks(projectId, open && (tab === "tasks" || history));
    const content = (
        <div className="canvas-workspace-panel flex h-full min-h-0 flex-col bg-background text-foreground" data-canvas-no-zoom data-canvas-wheel-scroll onWheel={(e) => e.stopPropagation()} onPointerDown={(e) => e.stopPropagation()}>
            <header className="flex h-8 items-center justify-between border-b border-border px-3">
                <strong className="sg-eyebrow m-0">{board.workspace}</strong>
                <button aria-label={board.closeWorkspace} onClick={onClose}>
                    <X className="size-4" />
                </button>
            </header>
            {tab === "nodes" && (
                <CanvasWorkspaceNodeListPanel nodes={nodes} config={config} results={searchCanvasNodes(nodes, deferredQuery, nodes.length, config)} query={query} deferredQuery={deferredQuery} selectedNodeIds={selectedNodeIds} onQueryChange={setQuery} onFocus={onFocus} />
            )}
            {tab === "assets" && <CanvasWorkspaceAssetPanel onInsert={onInsertAssets} onManage={onAssets} onProjectAssets={onProjectAssets} />}
            {(tab === "tasks" || history) && (
                <>
                    {tasks.error ? (
                        <div role="alert" className="p-4">
                            {board.taskLoadFailed}<Button onClick={() => void tasks.refetch()}>{board.retry}</Button>
                        </div>
                    ) : tasks.loading ? (
                        <p className="p-4">{board.loadingTasks}</p>
                    ) : history ? (
                        <CanvasWorkspaceHistoryPanel tasks={tasks.allTasks} refreshing={tasks.refreshing} onRefresh={() => void tasks.refetch()} onCancelTask={onCancelTask} />
                    ) : (
                        <CanvasWorkspaceTaskPanel
                            tasks={history ? tasks.allTasks : tasks.tasks.filter((t) => t.status === "queued" || t.status === "running")}
                            refreshing={tasks.refreshing}
                            onRefresh={() => void tasks.refetch()}
                            onCancelTask={onCancelTask}
                        />
                    )}
                </>
            )}
        </div>
    );
    const entries = [
        { id: "nodes", label: board.nodes, icon: Layers },
        { id: "assets", label: board.assets, icon: Images },
        { id: "tasks", label: board.tasks, icon: ListChecks },
        { id: "history", label: board.history, icon: History },
    ];
    return (
        <>
            <nav className="canvas-workspace-rail" style={{ ...canvasDockStyle(theme), boxShadow: "none" }} aria-label={board.workspace} data-canvas-no-zoom onWheel={(event) => event.stopPropagation()} onPointerDown={(event) => event.stopPropagation()}>
                {hosted ? null : (
                    <>
                <Link
                    to="/"
                    className="canvas-workspace-rail-button"
                    aria-label={board.home}
                    title={board.home}
                    onClick={(event) => {
                        if (requestHostNavigation("/")) event.preventDefault();
                    }}
                >
                    <Home />
                    <span>{board.home}</span>
                </Link>
                <div className="canvas-workspace-rail-divider" aria-hidden="true" />
                    </>
                )}
                <div className="canvas-workspace-rail-items">
                    {entries.map(({ id, label, icon: Icon }) => (
                        <button
                            key={id}
                            type="button"
                            className="canvas-workspace-rail-button"
                            title={open && tab === id ? board.collapsePanel : label}
                            aria-pressed={open && tab === id}
                            onClick={() => {
                                if (open && tab === id) onClose();
                                else {
                                    setTab(id);
                                    onOpen();
                                }
                            }}
                        >
                            <Icon />
                            <span>{label}</span>
                        </button>
                    ))}
                </div>
                <button
                    className="canvas-workspace-rail-button canvas-workspace-rail-toggle"
                    type="button"
                    onClick={open ? onClose : onOpen}
                    aria-expanded={open}
                    aria-label={open ? board.collapsePanel : board.expandPanel}
                    title={open ? board.collapsePanel : board.expandPanel}
                >
                    {open ? <PanelLeftClose /> : <PanelLeftOpen />}
                    <span>{open ? board.collapsePanel : board.expandPanel}</span>
                </button>
            </nav>
            {open &&
                (screens.lg ? (
                    <aside className="w-80 shrink-0 border-r border-border min-h-0" aria-label={board.workspace}>
                        {content}
                    </aside>
                ) : (
                    <AppDrawer flush open placement="left" title={null} closable={false} size={320} onClose={onClose}>
                        {content}
                    </AppDrawer>
                ))}
        </>
    );
}
