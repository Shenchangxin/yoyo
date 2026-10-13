// @ts-nocheck
import { useEffect, useRef, useState, type ReactNode } from "react";
import { Link, useNavigate } from "react-router";
import { Clapperboard, CloudDownload, CloudUpload, CopyPlus, Focus, FolderKanban, Gauge, Hand, History, Home, LayoutGrid, LayoutList, LoaderCircle, Menu, MousePointer2, Pencil, PanelsTopLeft, Plus, Redo2, Save, Search, Share2, Trash2, Undo2, Upload } from "lucide-react";
import { Tooltip } from "antd";

import { CanvasDropdown } from "@yingce/components/ui/canvas-overlay";
import { CanvasCreateMenu, type CanvasCreateCommand } from "@yingce/components/canvas/canvas-create-menu";
import { requestHostNavigation, isYoyoCanvasHost } from "@/features/video/canvas-host/design-system";
import { useCopy } from "@/lib/i18n";

import { WorkspaceCreditGiftMark } from "@yingce/components/layout/workspace-credit-gift-mark";
import { useWalletBalance } from "@yingce/hooks/use-wallet-balance";
import { openWorkspaceWallet } from "@yingce/lib/workspace-wallet";
import { canvasDockStyle } from "@yingce/lib/canvas/canvas-aceternity-style";
import type { CanvasContextSummary } from "@yingce/lib/canvas/canvas-context-summary";
import type { CanvasShortDramaProgress } from "@yingce/lib/canvas/canvas-short-drama";
import { canvasThemes } from "@yingce/lib/canvas-theme";
import { useCanvasThemeStore } from "@yingce/stores/canvas/use-canvas-theme-store";
import { useUserStore } from "@yingce/stores/use-user-store";
import type { CanvasMediaPerformanceMode, CanvasToolMode } from "@yingce/types/canvas";
import { CanvasShortcutsModal } from "./canvas-shortcuts-modal";

export type CanvasBoardView = "stage" | "list";

type CanvasTopBarProps = {
    syncStatus?: ReactNode;
    versionsOpen: boolean;
    onToggleVersions: () => void;
    title: string;
    titleDraft: string;
    isTitleEditing: boolean;
    onTitleDraftChange: (value: string) => void;
    onStartTitleEditing: () => void;
    onFinishTitleEditing: () => void;
    onCancelTitleEditing: () => void;
    canUndo: boolean;
    canRedo: boolean;
    onCreateProject: () => void;
    onDeleteProject: () => void;
    onSave: () => void | Promise<void>;
    onForceSave: () => void;
    onImportImage: () => void;
    onImportLibTV: () => void;
    onImportTapNow: () => void;
    onUndo: () => void;
    onRedo: () => void;
    onShare: () => void;
    shortcutRequestNonce: number;
    mediaPerformanceMode: CanvasMediaPerformanceMode;
    onMediaPerformanceModeChange: (mode: CanvasMediaPerformanceMode) => void;
    onOpenSearch: () => void;
    projectContext?: CanvasContextSummary & { projectId: string; projectName: string };
    onEnterFocusMode: () => void;
    shortDramaGuide?: { progress: CanvasShortDramaProgress; collapsed: boolean; onToggle: () => void };
    canvasTool?: CanvasToolMode;
    onToolChange?: (tool: CanvasToolMode) => void;
    createCommands?: CanvasCreateCommand[];
    boardView?: CanvasBoardView;
    onBoardViewChange?: (view: CanvasBoardView) => void;
};

export function CanvasTopBar({
    syncStatus,
    versionsOpen,
    onToggleVersions,
    title,
    titleDraft,
    isTitleEditing,
    onTitleDraftChange,
    onStartTitleEditing,
    onFinishTitleEditing,
    onCancelTitleEditing,
    canUndo,
    canRedo,
    onCreateProject,
    onDeleteProject,
    onSave,
    onForceSave,
    onImportImage,
    onImportLibTV,
    onImportTapNow,
    onUndo,
    onRedo,
    onShare,
    shortcutRequestNonce,
    mediaPerformanceMode: _mediaPerformanceMode,
    onMediaPerformanceModeChange,
    onOpenSearch,
    projectContext,
    onEnterFocusMode,
    shortDramaGuide,
    canvasTool,
    onToolChange,
    createCommands,
    boardView = "stage",
    onBoardViewChange,
}: CanvasTopBarProps) {
    const copy = useCopy().video.canvasBoard;
    const hosted = isYoyoCanvasHost();
    const theme = canvasThemes[useCanvasThemeStore((state) => state.theme)];
    const dockStyle = canvasDockStyle(theme, theme.node.text);
    const navigate = useNavigate();
    const user = useUserStore((state) => state.user);
    const creditsEnabled = useUserStore((state) => state.features.creditsEnabled);
    const { availableMicrocredits, refreshing } = useWalletBalance(user?.id, creditsEnabled);
    const titleRef = useRef<HTMLDivElement>(null);
    const [shortcutsOpen, setShortcutsOpen] = useState(false);
    const [addOpen, setAddOpen] = useState(false);

    const handleShortDramaGuideToggle = () => {
        shortDramaGuide?.onToggle();
    };

    useEffect(() => {
        if (shortcutRequestNonce > 0) setShortcutsOpen(true);
    }, [shortcutRequestNonce]);

    useEffect(() => {
        if (!isTitleEditing) return;
        const close = (event: PointerEvent) => {
            if (!titleRef.current?.contains(event.target as Node)) onFinishTitleEditing();
        };
        document.addEventListener("pointerdown", close, true);
        return () => document.removeEventListener("pointerdown", close, true);
    }, [isTitleEditing, onFinishTitleEditing]);

    const wrappedCreateCommands = (createCommands || []).map((command) => ({
        ...command,
        onClick: () => {
            command.onClick();
            setAddOpen(false);
        },
    }));

    const projectMenuItems = [
        hosted ? null : { key: "home", icon: <Home className="size-4" />, label: copy.home, onClick: () => { if (!requestHostNavigation("/")) navigate("/"); } },
        { key: "projects", icon: <LayoutGrid className="size-4" />, label: copy.boards, onClick: () => { if (!requestHostNavigation("/canvas")) navigate("/canvas"); } },
        { type: "divider" },
        { key: "new", icon: <Plus className="size-4" />, label: copy.newBoard, onClick: onCreateProject },
        { key: "delete", danger: true, icon: <Trash2 className="size-4" />, label: copy.deleteBoard, onClick: onDeleteProject },
        { key: "save", icon: <Save className="size-4" />, label: <MenuLabel text={copy.save} shortcut="⌘ S" />, onClick: () => void onSave() },
        { key: "force-save", icon: <CloudUpload className="size-4" />, label: copy.forceSave, onClick: onForceSave },
        { type: "divider" },
        { key: "import", icon: <Upload className="size-4" />, label: copy.importAssets, onClick: onImportImage },
        { key: "search", icon: <Search className="size-4" />, label: <MenuLabel text={copy.search} shortcut="⌘ F" />, onClick: onOpenSearch },
        { key: "libtv", icon: <CopyPlus className="size-4" />, label: copy.importLibTV, onClick: onImportLibTV },
        { key: "tapnow", icon: <CloudDownload className="size-4" />, label: copy.importTapNow, onClick: onImportTapNow },
        { type: "divider" },
        { key: "undo", disabled: !canUndo, icon: <Undo2 className="size-4" />, label: <MenuLabel text={copy.undo} shortcut="⌘ Z" />, onClick: onUndo },
        { key: "redo", disabled: !canRedo, icon: <Redo2 className="size-4" />, label: <MenuLabel text={copy.redo} shortcut="⌘ ⇧ Z / ⌘ Y" />, onClick: onRedo },
        {
            key: "performance",
            icon: <Gauge className="size-4" />,
            label: copy.performance,
            children: [
                { key: "auto", label: copy.performanceAuto, onClick: () => onMediaPerformanceModeChange("auto") },
                { key: "quality", label: copy.performanceQuality, onClick: () => onMediaPerformanceModeChange("quality") },
                { key: "performance", label: copy.performanceSpeed, onClick: () => onMediaPerformanceModeChange("performance") },
            ],
        },
    ].filter(Boolean);

    const showCredits = Boolean(user && creditsEnabled && !hosted);
    const contextCount = projectContext
        ? projectContext.chapterLabel || copy.nodeCount.replace("{n}", String(projectContext.nodeCount))
        : "";
    const selectedLabel = projectContext?.selectedCount ? ` · ${copy.selectedCount.replace("{n}", String(projectContext.selectedCount))}` : "";

    return (
        <>
            <div className="canvas-topbar pointer-events-none absolute inset-x-0 top-0 z-[var(--z-toolbar)] h-[var(--canvas-topbar-h)] items-center gap-3" data-testid="canvas-topbar">
                <div className="canvas-topbar-cluster canvas-topbar-project-cluster pointer-events-auto flex min-w-0 items-center gap-2" style={dockStyle}>
                    <CanvasTopBarTooltip label={copy.menu}>
                        <CanvasDropdown
                            trigger={["click"]}
                            placement="bottomLeft"
                            autoAdjustOverflow
                            menu={{ items: projectMenuItems }}
                        >
                            <button type="button" className="sg-ghost canvas-topbar-action" style={{ color: theme.node.text }} aria-label={copy.menu} data-testid="canvas-board-menu">
                                <Menu className="size-4" />
                            </button>
                        </CanvasDropdown>
                    </CanvasTopBarTooltip>

                    <div ref={titleRef} className="canvas-topbar-title-block flex min-w-0 flex-auto flex-col items-start overflow-hidden">
                        {isTitleEditing ? (
                            <input
                                autoFocus
                                size={canvasTitleInputSize(titleDraft)}
                                value={titleDraft}
                                onChange={(event) => onTitleDraftChange(event.target.value)}
                                onBlur={onFinishTitleEditing}
                                onKeyDown={(event) => {
                                    if (event.key === "Enter") onFinishTitleEditing();
                                    if (event.key === "Escape") onCancelTitleEditing();
                                }}
                                className="h-8 w-auto min-w-12 max-w-[min(280px,42vw)] appearance-none border-0 bg-transparent p-0 text-left text-[13px] font-semibold tracking-tight outline-none ring-0 focus:outline-none focus:ring-0 focus-visible:outline-2 focus-visible:outline-offset-2"
                                style={{ color: theme.node.text, caretColor: theme.accent.primary, border: 0, boxShadow: "none", outline: "none" }}
                                aria-label={copy.rename}
                            />
                        ) : (
                            <div className="canvas-topbar-title-row flex min-w-0 items-center gap-0.5">
                                <button type="button" className="min-w-0 flex-1 truncate text-left text-[13px] font-semibold tracking-tight transition-opacity hover:opacity-75" onClick={onStartTitleEditing} title={copy.rename}>
                                    {title || copy.untitled}
                                </button>
                                <CanvasTopBarTooltip label={copy.rename}>
                                    <button
                                        type="button"
                                        className="canvas-topbar-action grid size-7 shrink-0 place-items-center rounded-md opacity-60 transition-opacity hover:opacity-100 focus-visible:outline-none focus-visible:ring-2"
                                        style={{ color: theme.node.text }}
                                        onClick={onStartTitleEditing}
                                        aria-label={copy.rename}
                                    >
                                        <Pencil className="size-3.5" />
                                    </button>
                                </CanvasTopBarTooltip>
                            </div>
                        )}
                        {projectContext && !isTitleEditing ? (
                            <div className="canvas-topbar-project-context mt-0.5 flex w-full min-w-0 items-center gap-1.5 overflow-hidden text-[var(--fs-tiny)]" style={{ color: theme.node.muted }}>
                                <Link to={`/projects/${projectContext.projectId}/overview`} className="inline-flex min-w-0 items-center gap-1 hover:underline" title={projectContext.projectName}>
                                    <FolderKanban className="size-3 shrink-0" />
                                    <span className="max-w-[120px] truncate">{projectContext.projectName}</span>
                                </Link>
                                <span aria-hidden>·</span>
                                <button type="button" className="min-w-0 truncate hover:underline" onClick={onOpenSearch} title={copy.search}>
                                    {contextCount}
                                    {projectContext.shotLabel ? ` · ${projectContext.shotLabel}` : ""}
                                    {selectedLabel}
                                </button>
                            </div>
                        ) : null}
                    </div>
                    {syncStatus}
                </div>

                <div className="canvas-topbar-cluster canvas-topbar-tools-cluster pointer-events-auto flex items-center" style={dockStyle} role="toolbar" aria-label={copy.hud}>
                    {onToolChange ? (
                        <>
                            <CanvasTopBarTooltip label={copy.select}>
                                <button
                                    type="button"
                                    className={`sg-ghost canvas-topbar-action ${canvasTool === "box-select" ? "is-active" : ""}`}
                                    style={{ color: theme.node.text }}
                                    aria-label={copy.select}
                                    aria-pressed={canvasTool === "box-select"}
                                    data-testid="canvas-tool-select"
                                    onClick={() => onToolChange("box-select")}
                                >
                                    <MousePointer2 className="size-4" />
                                </button>
                            </CanvasTopBarTooltip>
                            <CanvasTopBarTooltip label={copy.pan}>
                                <button
                                    type="button"
                                    className={`sg-ghost canvas-topbar-action ${canvasTool === "move" ? "is-active" : ""}`}
                                    style={{ color: theme.node.text }}
                                    aria-label={copy.pan}
                                    aria-pressed={canvasTool === "move"}
                                    data-testid="canvas-tool-pan"
                                    onClick={() => onToolChange("move")}
                                >
                                    <Hand className="size-4" />
                                </button>
                            </CanvasTopBarTooltip>
                        </>
                    ) : null}
                    <CanvasTopBarTooltip label={copy.undo}>
                        <button type="button" className="sg-ghost canvas-topbar-action" style={{ color: theme.node.text }} disabled={!canUndo} aria-label={copy.undo} onClick={onUndo}>
                            <Undo2 className="size-4" />
                        </button>
                    </CanvasTopBarTooltip>
                    <CanvasTopBarTooltip label={copy.redo}>
                        <button type="button" className="sg-ghost canvas-topbar-action" style={{ color: theme.node.text }} disabled={!canRedo} aria-label={copy.redo} onClick={onRedo}>
                            <Redo2 className="size-4" />
                        </button>
                    </CanvasTopBarTooltip>
                    {wrappedCreateCommands.length ? (
                        <CanvasTopBarTooltip label={copy.add}>
                            <CanvasDropdown
                                trigger={["click"]}
                                placement="bottom"
                                autoAdjustOverflow
                                open={addOpen}
                                onOpenChange={setAddOpen}
                                popupRender={() => (
                                    <div className="sg-menu max-h-[min(520px,calc(100vh-96px))] overflow-y-auto overflow-x-hidden p-1.5" style={{ color: theme.node.text }}>
                                        <CanvasCreateMenu commands={wrappedCreateCommands} />
                                    </div>
                                )}
                            >
                                <button type="button" className="sg-ghost canvas-topbar-action" style={{ color: theme.node.text }} aria-label={copy.add} aria-expanded={addOpen} data-testid="canvas-tool-add">
                                    <Plus className="size-4" />
                                </button>
                            </CanvasDropdown>
                        </CanvasTopBarTooltip>
                    ) : null}
                </div>

                <div className="canvas-topbar-cluster canvas-topbar-trailing-cluster pointer-events-auto flex items-center" style={dockStyle}>
                    {onBoardViewChange ? (
                        <div className="canvas-board-view-toggle" role="group" aria-label={copy.stage}>
                            <CanvasTopBarTooltip label={copy.stage}>
                                <button
                                    type="button"
                                    className={`sg-ghost canvas-topbar-action ${boardView === "stage" ? "is-active" : ""}`}
                                    style={{ color: theme.node.text }}
                                    aria-pressed={boardView === "stage"}
                                    data-testid="canvas-board-view-stage"
                                    onClick={() => onBoardViewChange("stage")}
                                >
                                    <PanelsTopLeft className="size-4" />
                                </button>
                            </CanvasTopBarTooltip>
                            <CanvasTopBarTooltip label={copy.list}>
                                <button
                                    type="button"
                                    className={`sg-ghost canvas-topbar-action ${boardView === "list" ? "is-active" : ""}`}
                                    style={{ color: theme.node.text }}
                                    aria-pressed={boardView === "list"}
                                    data-testid="canvas-board-view-list"
                                    onClick={() => onBoardViewChange("list")}
                                >
                                    <LayoutList className="size-4" />
                                </button>
                            </CanvasTopBarTooltip>
                        </div>
                    ) : null}
                    <CanvasTopBarTooltip label={copy.search}>
                        <button type="button" className="sg-ghost canvas-topbar-action" style={{ color: theme.node.text }} onClick={onOpenSearch} aria-label={copy.search}>
                            <Search className="size-4" />
                        </button>
                    </CanvasTopBarTooltip>
                    {!hosted ? (
                        <CanvasTopBarTooltip label={copy.importDisplay}>
                            <CanvasDropdown
                                trigger={["click"]}
                                placement="bottomRight"
                                autoAdjustOverflow
                                menu={{
                                    items: [
                                        { key: "libtv", icon: <CopyPlus className="size-4" />, label: copy.importLibTV, onClick: onImportLibTV },
                                        { key: "tapnow", icon: <CloudDownload className="size-4" />, label: copy.importTapNow, onClick: onImportTapNow },
                                        { type: "divider" },
                                        {
                                            key: "performance",
                                            icon: <Gauge className="size-4" />,
                                            label: copy.performance,
                                            children: [
                                                { key: "auto", label: copy.performanceAuto, onClick: () => onMediaPerformanceModeChange("auto") },
                                                { key: "quality", label: copy.performanceQuality, onClick: () => onMediaPerformanceModeChange("quality") },
                                                { key: "performance", label: copy.performanceSpeed, onClick: () => onMediaPerformanceModeChange("performance") },
                                            ],
                                        },
                                    ],
                                }}
                            >
                                <button type="button" className="sg-ghost canvas-topbar-action canvas-topbar-import-button" style={{ color: theme.node.text }} aria-label={copy.importDisplay}>
                                    <CloudDownload className="size-4" />
                                </button>
                            </CanvasDropdown>
                        </CanvasTopBarTooltip>
                    ) : null}
                    {showCredits ? (
                        <CanvasTopBarTooltip label={copy.workspace}>
                            <button
                                type="button"
                                className="sg-ghost canvas-topbar-action w-auto min-w-0 px-2 text-[11px] font-medium tabular-nums"
                                style={{ color: theme.node.text }}
                                aria-label={copy.workspace}
                                onClick={() => openWorkspaceWallet()}
                            >
                                {refreshing && availableMicrocredits === null ? <LoaderCircle className="size-3.5 animate-spin opacity-60" /> : <WorkspaceCreditGiftMark className="is-compact" />}
                            </button>
                        </CanvasTopBarTooltip>
                    ) : null}
                    <CanvasTopBarTooltip label={copy.focus}>
                        <button type="button" className="sg-ghost canvas-topbar-action" style={{ color: theme.node.text }} onClick={onEnterFocusMode} aria-label={copy.focus}>
                            <Focus className="size-4" />
                        </button>
                    </CanvasTopBarTooltip>
                    {shortDramaGuide ? (
                        <CanvasTopBarTooltip label={shortDramaGuide.collapsed ? copy.dramaGuideExpand : copy.dramaGuideCollapse}>
                            <button
                                type="button"
                                className={`sg-ghost canvas-topbar-action w-auto min-w-0 px-2 ${shortDramaGuide.collapsed ? "" : "is-active"}`}
                                style={{ color: theme.node.text }}
                                onClick={handleShortDramaGuideToggle}
                                aria-label={copy.dramaGuide}
                                aria-pressed={!shortDramaGuide.collapsed}
                            >
                                <Clapperboard className="size-4" />
                                <span className="ml-1 text-[11px] tabular-nums">{shortDramaGuide.progress.completedCount}/5</span>
                            </button>
                        </CanvasTopBarTooltip>
                    ) : null}
                    <CanvasTopBarTooltip label={copy.versions}>
                        <button
                            type="button"
                            className={`sg-ghost canvas-topbar-action canvas-topbar-version-button ${versionsOpen ? "is-active" : ""}`}
                            style={{ color: theme.node.text }}
                            aria-label={copy.versions}
                            aria-pressed={versionsOpen}
                            onClick={onToggleVersions}
                        >
                            <History className="size-4" />
                        </button>
                    </CanvasTopBarTooltip>
                    <CanvasTopBarTooltip label={copy.share}>
                        <button type="button" className="sg-ghost canvas-topbar-action" style={{ color: theme.node.text }} onClick={onShare} aria-label={copy.share}>
                            <Share2 className="size-4" />
                        </button>
                    </CanvasTopBarTooltip>
                </div>
            </div>
            <CanvasShortcutsModal open={shortcutsOpen} onClose={() => setShortcutsOpen(false)} />
        </>
    );
}

function CanvasTopBarTooltip({ label, children }: { label: string; children: ReactNode }) {
    return (
        <Tooltip title={label} placement="bottom" mouseEnterDelay={0.15}>
            <span className="relative inline-flex">{children}</span>
        </Tooltip>
    );
}

function MenuLabel({ text, shortcut }: { text: string; shortcut: string }) {
    return (
        <span className="flex min-w-36 items-center justify-between gap-8">
            <span>{text}</span>
            <span className="text-xs opacity-45">{shortcut}</span>
        </span>
    );
}

function canvasTitleInputSize(value: string) {
    const visualLength = Array.from(value || "Board").reduce((length, character) => length + (character.codePointAt(0)! > 0xff ? 2 : 1), 0);
    return Math.min(30, Math.max(5, visualLength));
}
