// @ts-nocheck
import { AnimatePresence, motion } from "motion/react";
import { useEffect, useRef, useState, type MouseEvent as ReactMouseEvent } from "react";
import { Switch } from "@yingce/components/ui/base/switch";
import { Info } from "lucide-react";

import { FloatingDock } from "@yingce/components/ui/aceternity/floating-dock";
import { CanvasAppearanceControls } from "@yingce/components/canvas/canvas-appearance-controls";
import { useCanvasOverlayLayer } from "@yingce/components/canvas/canvas-overlay-layer";
import { CanvasCreateMenu, type CanvasCreateCommand } from "@yingce/components/canvas/canvas-create-menu";
import { useCanvasCreateCommands } from "@yingce/components/canvas/use-canvas-create-commands";
import { ToolbarSettingsModal } from "@yingce/components/canvas/toolbars/toolbar-settings-modal";
import { aceternityMotion } from "@yingce/lib/aceternity-motion";
import { canvasDockStyle } from "@yingce/lib/canvas/canvas-aceternity-style";
import type { CanvasAppearance } from "@yingce/lib/canvas/canvas-appearance";
import { canvasThemes, type CanvasBackgroundMode, type CanvasTheme } from "@yingce/lib/canvas-theme";
import { defaultToolbarPrefs, readToolbarPrefs, resolveToolbarEntries, type ToolContext, type ToolbarHandlers, type ToolbarPrefs } from "@yingce/lib/canvas/tool-registry";
import { useActiveTheme } from "@yingce/stores/canvas/use-canvas-theme-store";
import type { CanvasNodeTypeId, CanvasToolMode, CanvasWorkspaceMode } from "@yingce/types/canvas";

export function CanvasToolbar({
    selectedCount,
    workspaceMode,
    canvasTool,
    onToolChange,
    isProjectLinked,
    canUndo,
    canRedo,
    appearance,
    backgroundMode,
    showImageInfo,
    onAddImage,
    onAddVideo,
    onAddAudio,
    onAddText,
    onChooseStyle,
    onAddScript,
    onAddFrame,
    onAddFolder,
    onAddDrawing,
    onAddExtensionNode,
    onAddWorkflow,
    onOpenDirector,
    onUndo,
    onRedo,
    onUpload,
    onDelete,
    onClear,
    onDeselect,
    onAppearanceChange,
    onSaveAppearanceDefault,
    onBackgroundModeChange,
    onShowImageInfoChange,
    onOpenWorkspace,
    onOpenMyAssets,
    onOpenProjectCharacters,
}: {
    selectedCount: number;
    workspaceMode: CanvasWorkspaceMode;
    canvasTool: CanvasToolMode;
    onToolChange: (tool: CanvasToolMode) => void;
    isProjectLinked: boolean;
    canUndo: boolean;
    canRedo: boolean;
    appearance: CanvasAppearance;
    backgroundMode: CanvasBackgroundMode;
    showImageInfo: boolean;
    onAddImage: () => void;
    onAddVideo: () => void;
    onAddAudio: () => void;
    onAddText: () => void;
    onChooseStyle: () => void;
    onAddScript: () => void;
    onAddFrame: () => void;
    onAddFolder: () => void;
    onAddDrawing: () => void;
    onAddExtensionNode: (type: CanvasNodeTypeId) => void;
    onAddWorkflow: () => void;
    onOpenDirector: () => void;
    onUndo: () => void;
    onRedo: () => void;
    onUpload: () => void;
    onDelete: () => void;
    onClear: () => void;
    onDeselect: () => void;
    onAppearanceChange: (appearance: CanvasAppearance) => void;
    onSaveAppearanceDefault: (appearance: CanvasAppearance) => void;
    onBackgroundModeChange: (mode: CanvasBackgroundMode) => void;
    onShowImageInfoChange: (show: boolean) => void;
    onOpenWorkspace?: () => void;
    onOpenMyAssets: () => void;
    onOpenProjectCharacters: () => void;
}) {
    const rootRef = useRef<HTMLDivElement>(null);
    const { bringToFront, zIndex } = useCanvasOverlayLayer("main-toolbar", "var(--z-toolbar)", "chrome");
    const dockRef = useRef<HTMLDivElement>(null);
    const colorTheme = useActiveTheme();
    const theme = canvasThemes[colorTheme];
    const [addOpen, setAddOpen] = useState(false);
    const [appearanceOpen, setAppearanceOpen] = useState(false);
    const [settingsOpen, setSettingsOpen] = useState(false);
    const [panelX, setPanelX] = useState(0);
    const [panelParentWidth, setPanelParentWidth] = useState(() => (typeof window === "undefined" ? 1200 : window.innerWidth));
    const [prefs, setPrefs] = useState<ToolbarPrefs | null>(() => readToolbarPrefs("main"));

    useEffect(() => {
        if (addOpen || appearanceOpen) bringToFront();
    }, [addOpen, appearanceOpen, bringToFront]);

    // 设置面板关闭后重新读取偏好（用户可能调整了排序/显隐）
    useEffect(() => {
        if (!settingsOpen) setPrefs(readToolbarPrefs("main"));
    }, [settingsOpen]);

    const placePanel = (event: ReactMouseEvent<HTMLElement>) => {
        const dock = dockRef.current;
        const parent = dock?.parentElement;
        setPanelParentWidth(parent?.clientWidth || (typeof window === "undefined" ? 1200 : window.innerWidth));
        setPanelX(getPanelX(dock, event.currentTarget));
    };
    const runAddAction = (action: () => void) => {
        action();
        setAddOpen(false);
    };

    // 点击外部关闭浮层面板
    useEffect(() => {
        if (!addOpen && !appearanceOpen) return;
        const closeFloatingPanels = (event: PointerEvent) => {
            const target = event.target instanceof Node ? event.target : null;
            if (target && rootRef.current?.contains(target)) return;
            const element = event.target instanceof Element ? event.target : null;
            if (element?.closest(".ant-color-picker,.ant-popover")) return;
            setAddOpen(false);
            setAppearanceOpen(false);
        };
        document.addEventListener("pointerdown", closeFloatingPanels, true);
        return () => document.removeEventListener("pointerdown", closeFloatingPanels, true);
    }, [addOpen, appearanceOpen]);

    // 构建 handlers（主工具栏只需要部分回调，其余用 no-op 占位满足类型）
    const handlers: ToolbarHandlers = {
        onToolChange,
        onDeselect,
        onUndo,
        onRedo,
        onClear,
        onAddText,
        onAddImage,
        onAddVideo,
        onAddAudio,
        onAddScript,
        onAddFrame,
        onAddFolder,
        onAddDrawing,
        onAddExtensionNode,
        onAddWorkflow,
        onChooseStyle,
        onOpenDirector,
        onUpload,
        onOpenWorkspace,
    onOpenMyAssets,
        onOpenProjectCharacters,
        onBackgroundModeChange,
        onShowImageInfoChange,
        onToggleAddPanel: (event: ReactMouseEvent<HTMLElement>) => { placePanel(event); setAppearanceOpen(false); setSettingsOpen(false); setAddOpen((value) => !value); },
        onToggleAppearancePanel: (event: ReactMouseEvent<HTMLElement>) => { placePanel(event); setAddOpen(false); setSettingsOpen(false); setAppearanceOpen((value) => !value); },
        onToggleSettingsPanel: () => { setAddOpen(false); setAppearanceOpen(false); setSettingsOpen((value) => !value); },
        onDeleteSelected: onDelete,
        // 以下为多选/节点悬停工具栏回调，主工具栏不使用，用 no-op 占位
        onAlign: () => {}, onArrange: () => {}, onCreateStoryboard: () => {}, onCreateReferenceGroup: () => {}, onBatchConnect: () => {}, onMergeVideos: () => {}, onSendSelectionToAgent: () => {},
        onNodeInfo: () => {}, onNodeDelete: () => {}, onNodeRetry: () => {}, onNodeEditText: () => {}, onNodeDecreaseFont: () => {}, onNodeIncreaseFont: () => {},
        onNodeToggleDialog: () => {}, onNodeAnnotate: () => {}, onNodeGenerateImage: () => {}, onNodeUpload: () => {}, onNodeDownload: () => {}, onNodeSaveAsset: () => {},
        onNodeMaskEdit: () => {}, onNodeImageEdit: () => {}, onNodeRemoveBackground: () => {}, onNodeEmotion: () => {}, onNodePortraitTexture: () => {}, onNodeCrop: () => {}, onNodeSplit: () => {}, onNodeUpscale: () => {},
        onNodeSuperResolve: () => {}, onNodeAngle: () => {}, onNodeViewImage: () => {}, onNodeExtractVideoFrames: () => {}, onNodeExtractAudioFromVideo: () => {}, onNodeTrimVideoSegments: () => {}, onNodeSubtitles: () => {}, onNodeTimeline: () => {}, onNodeReversePrompt: () => {},
        onNodeToggleFreeResize: () => {}, onNodeToggleLocked: () => {}, onNodeCopyPrompt: () => {},
    } as ToolbarHandlers;

    const ctx: ToolContext = {
        selectedCount,
        selectedNodeTypes: new Set(),
        selectedVideoCount: 0,
        canvasTool,
        workspaceMode,
        isProjectLinked,
        canUndo,
        canRedo,
        extractingVideoFrames: false,
        extractingAudio: false,
        trimmingVideo: false,
        mergingVideos: false,
        addPanelOpen: addOpen,
        appearancePanelOpen: appearanceOpen,
        settingsPanelOpen: settingsOpen,
        handlers,
    };

    const items = resolveToolbarEntries("main", ctx, prefs ?? defaultToolbarPrefs("main"));

    // 中央空白起点与主工具栏共用同一份命令解析，避免素材类型和插件节点逐渐分叉。
    const createCommands = useCanvasCreateCommands(ctx, runAddAction);

    return (
        <div
            ref={rootRef}
            data-canvas-no-zoom
            data-canvas-main-dock
            className="pointer-events-none absolute bottom-[var(--canvas-toolbar-bottom,12px)] left-[var(--canvas-zoom-reserve,228px)] right-[var(--canvas-inset-x,12px)] flex justify-center"
            style={{ zIndex }}
            onPointerDownCapture={bringToFront}
            onFocusCapture={bringToFront}
        >
            <FloatingDock ref={dockRef} items={items} className="canvas-floating-dock pointer-events-auto max-w-full" style={canvasDockStyle(theme)} tooltipPlacement="top" />

            <AnimatePresence>
                {addOpen ? (
                    <AddNodeMenu
                        x={panelX}
                        parentWidth={panelParentWidth}
                        theme={theme}
                        commands={createCommands}
                    />
                ) : null}
            </AnimatePresence>

            <AnimatePresence>
                {appearanceOpen ? (
                    <motion.div initial={{ opacity: 0, scaleY: 0.96, y: 6 }} animate={{ opacity: 1, scaleY: 1, y: 0 }} exit={{ opacity: 0, scaleY: 0.96, y: 4 }} transition={{ duration: aceternityMotion.duration.panel, ease: aceternityMotion.easing.enter }} className="pointer-events-auto absolute bottom-[calc(100%+8px)] z-[var(--dock-z-popover)] w-[320px] max-w-[calc(100vw-24px)]" style={{ left: clampPanelX(panelX, 320, panelParentWidth), transformOrigin: "bottom center", x: "-50%" }}>
                        <motion.div className="sg-popover max-h-[min(420px,calc(100vh-96px))] overflow-y-auto overflow-x-hidden p-2.5" style={{ color: theme.toolbar.item }} onWheel={(event) => event.stopPropagation()}>
                            <div className="sg-eyebrow px-1">画布外观</div>
                            <CanvasAppearanceControls appearance={appearance} backgroundMode={backgroundMode} colorTheme={colorTheme} theme={theme} onAppearanceChange={onAppearanceChange} onSaveAppearanceDefault={onSaveAppearanceDefault} onBackgroundModeChange={onBackgroundModeChange} />
                            <div className="mt-2.5 flex items-center justify-between gap-2 rounded-[var(--dock-item-radius-labeled)] border px-2.5 py-2" style={{ background: theme.spatial.surface, borderColor: theme.toolbar.border }}>
                                <span className="inline-flex min-w-0 items-center gap-1.5 text-[var(--fs-tiny)] font-semibold"><Info className="size-3" />媒体信息</span>
                                <Switch size="sm" checked={showImageInfo} onChange={onShowImageInfoChange} />
                            </div>
                        </motion.div>
                    </motion.div>
                ) : null}
            </AnimatePresence>

            <ToolbarSettingsModal open={settingsOpen} onClose={() => setSettingsOpen(false)} toolbar="main" />
        </div>
    );
}

function AddNodeMenu({ x, parentWidth, theme, commands }: {
    x: number;
    parentWidth: number;
    theme: CanvasTheme;
    commands: CanvasCreateCommand[];
}) {
    return (
        <motion.div layoutId="studio-add-morph" initial={{ opacity: 0, scaleY: 0.96, y: 6 }} animate={{ opacity: 1, scaleY: 1, y: 0 }} exit={{ opacity: 0, scaleY: 0.96, y: 4 }} transition={{ duration: aceternityMotion.duration.panel, ease: aceternityMotion.easing.enter }} className="pointer-events-auto absolute bottom-[calc(100%+8px)] z-[var(--dock-z-popover)] w-[280px] max-w-[calc(100vw-24px)]" style={{ left: clampPanelX(x, 280, parentWidth), transformOrigin: "bottom center", x: "-50%" }}>
            <motion.div className="sg-menu max-h-[min(520px,calc(100vh-96px))] overflow-y-auto overflow-x-hidden p-1.5" style={{ color: theme.node.text }} onWheel={(event) => event.stopPropagation()}>
                <CanvasCreateMenu commands={commands} />
            </motion.div>
        </motion.div>
    );
}

function getPanelX(dock: HTMLDivElement | null, target: HTMLElement) {
    if (!dock) return 0;
    const rootBox = dock.parentElement?.getBoundingClientRect() || dock.getBoundingClientRect();
    const box = target.getBoundingClientRect();
    return box.left - rootBox.left + box.width / 2;
}

function clampPanelX(x: number, panelWidth: number, parentWidth = typeof window === "undefined" ? 1200 : window.innerWidth) {
    const half = panelWidth / 2;
    const min = half + 12;
    const max = Math.max(min, parentWidth - half - 12);
    if (!x) return parentWidth / 2;
    return Math.min(Math.max(x, min), max);
}
