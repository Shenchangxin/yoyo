// @ts-nocheck
import { Modal } from "antd";
import { Switch } from "@yingce/components/ui/base/switch";
import { GripVertical, RotateCcw, X } from "lucide-react";
import { motion, useReducedMotion } from "motion/react";
import { useEffect, useRef, useState } from "react";

import { canvasThemes } from "@yingce/lib/canvas-theme";
import { defaultToolbarPrefs, getToolbarTools, persistToolbarPrefs, readToolbarPrefs, type ToolbarId, type ToolbarPrefs, type ToolContext, type ToolDefinition } from "@yingce/lib/canvas/tool-registry";
import { useActiveTheme } from "@yingce/stores/canvas/use-canvas-theme-store";

type ToolbarSettingsModalProps = {
    open: boolean;
    onClose: () => void;
    toolbar: ToolbarId;
};

/** 设置面板用的最小化上下文——仅用于解析工具的 label/icon */
const settingsMockContext: ToolContext = {
    selectedCount: 0,
    selectedNodeTypes: new Set(),
    selectedVideoCount: 0,
    canvasTool: "move",
    workspaceMode: "professional",
    isProjectLinked: false,
    canUndo: false,
    canRedo: false,
    extractingVideoFrames: false,
    extractingAudio: false,
    trimmingVideo: false,
    mergingVideos: false,
    addPanelOpen: false,
    appearancePanelOpen: false,
    settingsPanelOpen: false,
    handlers: {} as ToolContext["handlers"],
};

type SettingsItem = {
    id: string;
    label: string;
    icon: React.ReactNode;
    visible: boolean;
};

export function ToolbarSettingsModal({ open, onClose, toolbar }: ToolbarSettingsModalProps) {
    const theme = canvasThemes[useActiveTheme()];
    const reducedMotion = useReducedMotion();
    const [items, setItems] = useState<SettingsItem[]>([]);
    const [toolbarId, setToolbarId] = useState<ToolbarId>(toolbar);
    const draggedItemIdRef = useRef<string | null>(null);
    const dragTargetIdRef = useRef<string | null>(null);
    const [draggedItemId, setDraggedItemId] = useState<string | null>(null);
    const visibleCount = items.filter((item) => item.visible).length;

    useEffect(() => {
        if (!open) return;
        setToolbarId(toolbar);
        const tools = getToolbarTools(toolbar);
        const prefs = readToolbarPrefs(toolbar) ?? defaultToolbarPrefs(toolbar);
        const hiddenSet = new Set(prefs.hidden);
        const orderIndex = new Map(prefs.order.map((id, index) => [id, index]));
        const sorted = [...tools].sort((a, b) => {
            const ai = orderIndex.has(a.id) ? orderIndex.get(a.id)! : Number.MAX_SAFE_INTEGER;
            const bi = orderIndex.has(b.id) ? orderIndex.get(b.id)! : Number.MAX_SAFE_INTEGER;
            if (ai !== bi) return ai - bi;
            return a.defaultOrder - b.defaultOrder;
        });
        setItems(sorted.map((tool) => ({
            id: tool.id,
            label: resolveLabel(tool, settingsMockContext),
            icon: resolveIcon(tool, settingsMockContext),
            visible: !hiddenSet.has(tool.id),
        })));
    }, [open, toolbar]);

    const handleDragStart = (id: string) => {
        draggedItemIdRef.current = id;
        dragTargetIdRef.current = id;
        setDraggedItemId(id);
    };

    const handleDragEnter = (targetId: string) => {
        const sourceId = draggedItemIdRef.current;
        if (!sourceId || dragTargetIdRef.current === targetId) return;
        dragTargetIdRef.current = targetId;

        setItems((current) => {
            const sourceIndex = current.findIndex((item) => item.id === sourceId);
            const targetIndex = current.findIndex((item) => item.id === targetId);
            if (sourceIndex < 0 || targetIndex < 0) return current;

            const next = [...current];
            const [movedItem] = next.splice(sourceIndex, 1);
            next.splice(targetIndex, 0, movedItem);
            persistCurrent(next);
            return next;
        });
    };

    const handleDragEnd = () => {
        draggedItemIdRef.current = null;
        dragTargetIdRef.current = null;
        setDraggedItemId(null);
    };

    const handleToggleVisible = (id: string, visible: boolean) => {
        setItems((prev) => {
            const next = prev.map((item) => item.id === id ? { ...item, visible } : item);
            persistCurrent(next);
            return next;
        });
    };

    const handleReset = () => {
        const defaults = defaultToolbarPrefs(toolbarId);
        const tools = getToolbarTools(toolbarId);
        const hiddenSet = new Set(defaults.hidden);
        setItems(tools.map((tool) => ({
            id: tool.id,
            label: resolveLabel(tool, settingsMockContext),
            icon: resolveIcon(tool, settingsMockContext),
            visible: !hiddenSet.has(tool.id),
        })));
        persistToolbarPrefs(toolbarId, defaults);
    };

    const persistCurrent = (currentItems: SettingsItem[]) => {
        const prefs: ToolbarPrefs = {
            order: currentItems.map((item) => item.id),
            hidden: currentItems.filter((item) => !item.visible).map((item) => item.id),
        };
        persistToolbarPrefs(toolbarId, prefs);
    };

    return (
        <Modal
            className="canvas-toolbar-settings-modal"
            open={open}
            onCancel={onClose}
            footer={null}
            closable={false}
            width={380}
            centered
            destroyOnClose
            getContainer={() => document.body}
            styles={{
                container: { padding: 0, background: theme.spatial.elevated, border: 0, boxShadow: "none" },
                body: { padding: 0, background: theme.spatial.elevated },
            }}
        >
            <div className="flex items-center justify-between gap-3 px-4 pb-2 pt-3.5">
                <div className="min-w-0">
                    <h2 className="text-[13px] font-semibold leading-none tracking-tight">工具栏设置</h2>
                    <p className="mt-1.5 text-[11px] leading-none" style={{ color: theme.node.muted }}>拖动排序，开关控制显示</p>
                </div>
                <button
                    type="button"
                    onClick={onClose}
                    className="grid size-7 shrink-0 place-items-center rounded-[7px] outline-none transition-colors hover:bg-black/5 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 dark:hover:bg-white/8"
                    style={{ color: theme.node.muted, outlineColor: theme.accent.primary }}
                    aria-label="关闭工具栏设置"
                >
                    <X className="size-3.5" strokeWidth={1.55} />
                </button>
            </div>
            <div className="flex items-center justify-between gap-3 px-4 pb-2">
                <span className="text-[10.5px] font-medium tabular-nums" style={{ color: theme.node.muted }}>已显示 {visibleCount}/{items.length}</span>
                <button
                    type="button"
                    onClick={handleReset}
                    className="inline-flex h-6 items-center gap-1 rounded-[7px] px-1.5 text-[10.5px] font-medium outline-none transition-colors hover:bg-black/5 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 dark:hover:bg-white/8"
                    style={{ color: theme.node.muted, outlineColor: theme.accent.primary }}
                    aria-label="恢复默认工具栏设置"
                >
                    <RotateCcw className="size-3" strokeWidth={1.55} />
                    恢复默认
                </button>
            </div>
            <div className="canvas-toolbar-settings-list max-h-[min(52vh,420px)] overflow-y-auto px-2.5 pb-3" aria-label="主工具栏顺序">
                {items.map((item) => (
                    <ToolbarSettingsItem
                        key={item.id}
                        item={item}
                        reducedMotion={Boolean(reducedMotion)}
                        theme={theme}
                        dragging={draggedItemId === item.id}
                        onToggleVisible={handleToggleVisible}
                        onDragStart={handleDragStart}
                        onDragEnter={handleDragEnter}
                        onDragEnd={handleDragEnd}
                    />
                ))}
            </div>
        </Modal>
    );
}

function ToolbarSettingsItem({ item, reducedMotion, theme, dragging, onToggleVisible, onDragStart, onDragEnter, onDragEnd }: { item: SettingsItem; reducedMotion: boolean; theme: (typeof canvasThemes)[keyof typeof canvasThemes]; dragging: boolean; onToggleVisible: (id: string, visible: boolean) => void; onDragStart: (id: string) => void; onDragEnter: (id: string) => void; onDragEnd: () => void }) {
    return (
        <motion.div
            layout={!reducedMotion}
            transition={reducedMotion ? { duration: 0 } : undefined}
            className={`canvas-toolbar-settings-row flex h-10 items-center gap-2 rounded-[8px] px-1.5 ${item.visible ? "" : "is-hidden"} ${dragging ? "is-dragging" : ""}`}
            style={{ color: theme.node.text }}
            onDragEnter={() => onDragEnter(item.id)}
            onDragOver={(event) => event.preventDefault()}
        >
            <button
                type="button"
                draggable
                className="grid size-6 shrink-0 touch-none cursor-grab place-items-center rounded-[6px] outline-none opacity-35 transition-opacity hover:opacity-70 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 active:cursor-grabbing"
                style={{ color: theme.node.muted, outlineColor: theme.accent.primary }}
                onDragStart={(event) => {
                    event.dataTransfer.effectAllowed = "move";
                    onDragStart(item.id);
                }}
                onDragEnd={onDragEnd}
                aria-label={`拖动调整${item.label}顺序`}
            >
                <GripVertical className="size-3.5" strokeWidth={1.55} />
            </button>
            <span className="grid size-6 shrink-0 place-items-center rounded-[6px]" style={{ color: theme.node.muted }}>
                <span className="grid size-[15px] place-items-center [&_svg]:size-[15px]">{item.icon}</span>
            </span>
            <span className="min-w-0 flex-1 truncate text-[12.5px] font-medium leading-none tracking-tight" title={item.label}>{item.label}</span>
            <Switch size="sm" checked={item.visible} onChange={(checked) => onToggleVisible(item.id, checked)} aria-label={`${item.visible ? "隐藏" : "显示"}${item.label}`} />
        </motion.div>
    );
}

function resolveLabel(tool: ToolDefinition, ctx: ToolContext): string {
    return typeof tool.label === "function" ? tool.label(ctx) : tool.label;
}

function resolveIcon(tool: ToolDefinition, ctx: ToolContext): React.ReactNode {
    return typeof tool.icon === "function" ? tool.icon(ctx) : tool.icon;
}
