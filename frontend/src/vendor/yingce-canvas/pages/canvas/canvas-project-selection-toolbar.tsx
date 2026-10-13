// @ts-nocheck
import type { RefObject } from "react";
import { Ellipsis } from "lucide-react";
import { Tooltip } from "antd";

import { CanvasSelectionToolbar } from "@yingce/components/canvas/canvas-workspace-overlays";
import { CanvasDropdown } from "@yingce/components/ui/canvas-overlay";
import { FloatingDock } from "@yingce/components/ui/aceternity/floating-dock";
import { canvasThemes } from "@yingce/lib/canvas-theme";
import { canvasDockStyle } from "@yingce/lib/canvas/canvas-aceternity-style";
import { defaultToolbarPrefs, readToolbarPrefs, resolveToolbarEntries, type ToolContext, type ToolbarHandlers } from "@yingce/lib/canvas/tool-registry";
import type { CanvasAlignmentMode } from "@yingce/lib/canvas/canvas-layout";
import { useCanvasThemeStore } from "@yingce/stores/canvas/use-canvas-theme-store";
import { isYoyoCanvasHost } from "@/features/video/canvas-host/design-system";
import { useCopy } from "@/lib/i18n";

type CanvasProjectSelectionToolbarProps = {
    anchorRef: RefObject<HTMLDivElement | null>;
    containerRef: RefObject<HTMLDivElement | null>;
    count: number;
    selectedVideoCount: number;
    mergingVideos: boolean;
    onAlign: (mode: CanvasAlignmentMode) => void;
    onArrange: (mode: "row" | "column" | "grid" | "flow") => void;
    onCreateStoryboard: () => void;
    onCreateReferenceGroup: () => void;
    onBatchConnect: () => void;
    onMergeVideos: () => void;
    onSendSelectionToAgent: () => void;
};

export function CanvasProjectSelectionToolbar({ anchorRef, containerRef, count, selectedVideoCount, mergingVideos, onAlign, onArrange, onCreateStoryboard, onCreateReferenceGroup, onBatchConnect, onMergeVideos, onSendSelectionToAgent }: CanvasProjectSelectionToolbarProps) {
    const theme = canvasThemes[useCanvasThemeStore((state) => state.theme)];
    const board = useCopy().video.canvasBoard;
    const hosted = isYoyoCanvasHost();

    const handlers = {
        onAlign, onArrange, onCreateStoryboard, onCreateReferenceGroup, onBatchConnect, onMergeVideos, onSendSelectionToAgent,
    } as Partial<ToolbarHandlers> as ToolbarHandlers;

    const ctx: ToolContext = {
        selectedCount: count,
        selectedNodeTypes: new Set(),
        selectedVideoCount,
        canvasTool: "move",
        workspaceMode: "professional",
        isProjectLinked: false,
        canUndo: false,
        canRedo: false,
        extractingVideoFrames: false,
        extractingAudio: false,
        trimmingVideo: false,
        mergingVideos,
        addPanelOpen: false,
        appearancePanelOpen: false,
        settingsPanelOpen: false,
        handlers,
    };

    const prefs = readToolbarPrefs("selection") ?? defaultToolbarPrefs("selection");
    const items = resolveToolbarEntries("selection", ctx, prefs);
    const primaryIds = new Set(["selection-create-storyboard", "selection-send-to-agent"]);
    const primary = items.filter((item) => item.kind !== "separator" && item.kind !== "switch" && primaryIds.has(item.id));
    const overflow = items.filter((item) => item.kind !== "separator" && item.kind !== "switch" && !primaryIds.has(item.id));

    return (
        <CanvasSelectionToolbar anchorRef={anchorRef} containerRef={containerRef} count={count}>
            {hosted ? (
                <div className="sg-glass-group canvas-selection-hud flex items-center" role="toolbar" aria-label={board.selectionHud} style={canvasDockStyle(theme)} data-testid="canvas-selection-hud">
                    {primary.map((tool) => (
                        <Tooltip key={tool.id} title={tool.label} placement="top" mouseEnterDelay={0.15}>
                            <button
                                type="button"
                                className="sg-ghost canvas-topbar-action"
                                style={{ color: theme.node.text }}
                                aria-label={tool.label}
                                disabled={tool.disabled}
                                onClick={tool.onClick}
                            >
                                <span className="grid size-4 place-items-center">{tool.icon}</span>
                            </button>
                        </Tooltip>
                    ))}
                    {overflow.length ? (
                        <CanvasDropdown
                            trigger={["click"]}
                            placement="topRight"
                            autoAdjustOverflow
                            menu={{
                                items: overflow.map((tool) => ({
                                    key: tool.id,
                                    icon: tool.icon,
                                    label: tool.label,
                                    disabled: tool.disabled,
                                    onClick: tool.onClick,
                                })),
                            }}
                        >
                            <Tooltip title={board.more} placement="top" mouseEnterDelay={0.15}>
                                <button type="button" className="sg-ghost canvas-topbar-action" style={{ color: theme.node.text }} aria-label={board.more}>
                                    <Ellipsis className="size-4" />
                                </button>
                            </Tooltip>
                        </CanvasDropdown>
                    ) : null}
                </div>
            ) : (
                <FloatingDock items={items} size="compact" className="canvas-floating-dock" style={canvasDockStyle(theme)} ariaLabel={board.selectionHud} tooltipPlacement="top" />
            )}
        </CanvasSelectionToolbar>
    );
}
