// @ts-nocheck
import { motion, useReducedMotion } from "motion/react";
import { useEffect, useLayoutEffect, useRef, useState, type CSSProperties, type ReactNode, type RefObject } from "react";
import { createPortal } from "react-dom";
import { Clapperboard, Image as ImageIcon, List, Music2, Pencil, Table2, Video, Settings2, Workflow as WorkflowIcon } from "lucide-react";

import { canvasOverlayHost } from "@/features/video/canvas-host/design-system/popup-container";
import { useCanvasOverlayLayer } from "@yingce/components/canvas/canvas-overlay-layer";
import { canvasThemes } from "@yingce/lib/canvas-theme";
import { aceternityMotion } from "@yingce/lib/aceternity-motion";
import { subscribeCanvasGraphicsViewportPreview, subscribeCanvasNodeDragPreview, subscribeCanvasViewportPreview } from "@yingce/lib/canvas/canvas-live-viewport";
import { useActiveTheme } from "@yingce/stores/canvas/use-canvas-theme-store";
import { CanvasNodeType, type CanvasNodeData, type ConnectionHandle, type Position, type ViewportTransform } from "@yingce/types/canvas";

export type PendingConnectionCreate = {
    connection: ConnectionHandle;
    position: Position;
    quick?: boolean;
    batchSourceNodeIds?: string[];
};

export function CanvasSelectionToolbar({ anchorRef, containerRef, count, children }: { anchorRef: RefObject<HTMLDivElement | null>; containerRef: RefObject<HTMLDivElement | null>; count: number; children: ReactNode }) {
    const theme = canvasThemes[useActiveTheme()];
    const reducedMotion = useReducedMotion();
    const toolbarRef = useRef<HTMLDivElement>(null);
    const [anchor, setAnchor] = useState<{ left: number; top: number; placement: "above" | "below" } | null>(null);

    useLayoutEffect(() => {
        const element = anchorRef.current;
        const container = containerRef.current;
        if (!element || !container) {
            setAnchor(null);
            return;
        }

        const update = () => {
            const container = containerRef.current;
            if (!container) {
                setAnchor(null);
                return;
            }
            const containerBounds = container.getBoundingClientRect();
            const toolbarHeight = toolbarRef.current?.offsetHeight || 40;
            const left = containerBounds.width / 2;
            const top = Math.max(12, containerBounds.height - toolbarHeight - 12);
            if (toolbarRef.current) {
                toolbarRef.current.style.left = `${left}px`;
                toolbarRef.current.style.top = `${top}px`;
                toolbarRef.current.classList.remove("-translate-y-full");
            }
            setAnchor((current) => (current?.left === left && current.top === top && current.placement === "below" ? current : { left, top, placement: "below" }));
        };

        update();
        const resizeObserver = new ResizeObserver(update);
        resizeObserver.observe(element);
        resizeObserver.observe(container);
        if (toolbarRef.current) resizeObserver.observe(toolbarRef.current);
        const viewportLayer = element.parentElement;
        const mutationObserver = new MutationObserver(update);
        if (viewportLayer) mutationObserver.observe(viewportLayer, { attributes: true, attributeFilter: ["style"] });
        const unsubscribeViewport = subscribeCanvasViewportPreview(container, update);
        window.addEventListener("resize", update);
        return () => {
            resizeObserver.disconnect();
            mutationObserver.disconnect();
            unsubscribeViewport();
            window.removeEventListener("resize", update);
        };
    }, [anchorRef, containerRef, count]);

    if (!anchor) return null;
    return (
        <div
            ref={toolbarRef}
            data-canvas-selection-toolbar
            data-canvas-no-zoom
            className="absolute z-[var(--z-toolbar)] max-w-[calc(100%_-_24px)] -translate-x-1/2"
            style={{ left: anchor.left, top: anchor.top, color: theme.node.text, transformOrigin: anchor.placement === "above" ? "bottom center" : "top center" }}
            onMouseDown={(event) => event.stopPropagation()}
            onPointerDown={(event) => event.stopPropagation()}
        >
            <motion.div
                initial={reducedMotion ? { opacity: 0 } : { opacity: 0, scale: 0.9, y: anchor.placement === "above" ? 8 : -8 }}
                animate={{ opacity: 1, scale: 1, y: 0 }}
                transition={aceternityMotion.spring.panel}
                className="flex items-center gap-2"
            >
                <span className="canvas-selection-count shrink-0 tabular-nums">
                    已选 {count}
                </span>
                <div className="max-w-[min(920px,calc(100vw-48px))] overflow-x-auto overflow-y-visible">{children}</div>
            </motion.div>
        </div>
    );
}

export function CanvasNodePanelOverlay({
    node,
    viewport,
    containerRef,
    panelWidth,
    panelHeight = 190,
    dragOffset,
    isDragging = false,
    allowOverflow = false,
    onBackgroundPointerDown,
    children,
}: {
    node: CanvasNodeData;
    viewport: ViewportTransform;
    containerRef: RefObject<HTMLDivElement | null>;
    panelWidth?: number;
    panelHeight?: number;
    dragOffset?: Position | null;
    isDragging?: boolean;
    allowOverflow?: boolean;
    onBackgroundPointerDown?: () => void;
    children: ReactNode;
}) {
    const panelRef = useRef<HTMLDivElement>(null);
    const { bringToFront, zIndex } = useCanvasOverlayLayer(`node-panel:${node.id}`, "var(--z-modal-overlay)");
    const bounds = overlayBoundsRect(containerRef.current);
    const initialWidth = resolveNodePanelWidth(node, viewport, panelWidth, bounds?.width || 0);
    const initialPosition = fallbackViewportPosition(node, viewport, bounds, initialWidth, panelHeight, dragOffset);

    useLayoutEffect(() => {
        bringToFront();
    }, [bringToFront]);

    useLayoutEffect(() => {
        const container = containerRef.current;
        const panel = panelRef.current;
        if (!container || !panel) return;
        let liveViewport = viewport;
        const update = (nextViewport: ViewportTransform) => {
            liveViewport = nextViewport;
            const safe = safeOverlayRect(overlayBoundsRect(container));
            const maxWidth = Math.max(280, safe.right - safe.left);
            const nextWidth = Math.min(resolveNodePanelWidth(node, nextViewport, panelWidth, safe.right - safe.left + OVERLAY_MARGIN * 2), maxWidth);
            panel.style.position = "fixed";
            panel.style.width = `${Math.round(nextWidth)}px`;
            panel.style.minWidth = "0px";
            panel.style.maxWidth = `${Math.round(maxWidth)}px`;
            panel.style.height = "max-content";
            panel.style.maxHeight = "none";
            panel.style.overflow = "visible";
            panel.style.pointerEvents = "none";
            panel.style.right = "auto";
            panel.style.bottom = "auto";
            panel.style.transform = "none";
            const anchor = nodeAnchorElement(container, node.id);
            const shellRect = anchor?.shell.getBoundingClientRect();
            if (!anchor || !shellRect || shellRect.width < 2 || shellRect.height < 2) {
                panel.style.visibility = "hidden";
                return;
            }
            const aboveAnchorTop = nodeAboveAnchorTop(anchor.nodeElement, shellRect);
            alignFixedPanel(panel, shellRect, aboveAnchorTop, safe);
            alignFixedPanel(panel, shellRect, aboveAnchorTop, safe);
        };
        update(viewport);
        const handleWindowResize = () => update(liveViewport);
        const resizeObserver = new ResizeObserver(handleWindowResize);
        resizeObserver.observe(container);
        resizeObserver.observe(panel);
        const anchor = nodeAnchorElement(container, node.id);
        if (anchor) resizeObserver.observe(anchor.shell);
        const unsubscribeViewport = subscribeCanvasGraphicsViewportPreview(container, update);
        const unsubscribeDrag = subscribeCanvasNodeDragPreview(container, () => update(liveViewport));
        window.addEventListener("resize", handleWindowResize);
        return () => {
            resizeObserver.disconnect();
            unsubscribeViewport();
            unsubscribeDrag();
            window.removeEventListener("resize", handleWindowResize);
        };
    }, [allowOverflow, containerRef, dragOffset?.x, dragOffset?.y, isDragging, node.height, node.id, node.position.x, node.position.y, node.width, panelHeight, panelWidth, viewport]);

    useEffect(() => {
        if (!onBackgroundPointerDown) return;
        const handlePointerDown = (event: PointerEvent) => {
            if (event.button !== 0) return;
            const target = event.target instanceof Element ? event.target : null;
            if (target?.closest(PANEL_DISMISS_IGNORE_SELECTOR)) return;
            if (target?.closest("[data-node-id]")) return;
            const content = panelRef.current?.querySelector<HTMLElement>("[data-canvas-panel-content]");
            if (content && target && content.contains(target)) return;
            if (content && pointInRect(event.clientX, event.clientY, content.getBoundingClientRect(), 2)) return;
            const container = containerRef.current;
            if (!container) return;
            if (!pointInRect(event.clientX, event.clientY, overlayBoundsRect(container), 0)) return;
            onBackgroundPointerDown();
        };
        window.addEventListener("pointerdown", handlePointerDown, true);
        return () => window.removeEventListener("pointerdown", handlePointerDown, true);
    }, [containerRef, onBackgroundPointerDown]);

    if (typeof document === "undefined") return null;

    return createPortal(
        <div
            ref={panelRef}
            data-canvas-node-panel
            className="pointer-events-none overflow-visible"
            style={{ position: "fixed", left: initialPosition.left, top: initialPosition.top, width: initialWidth, minWidth: 0, maxWidth: "calc(100vw - 24px)", height: "max-content", maxHeight: "none", overflow: "visible", pointerEvents: "none", visibility: "hidden", zIndex }}
        >
            <div
                data-canvas-panel-content
                data-canvas-no-zoom
                className="w-full min-w-0"
                style={{ pointerEvents: "auto", height: "max-content", width: "100%" }}
                onMouseDown={(event) => event.stopPropagation()}
                onPointerDownCapture={bringToFront}
                onFocusCapture={bringToFront}
                onPointerDown={(event) => event.stopPropagation()}
            >
                {children}
            </div>
        </div>,
        canvasOverlayHost(),
    );
}

const NODE_PROMPT_PANEL_MIN_WIDTH = 720;

function resolveNodePanelWidth(node: CanvasNodeData, viewport: ViewportTransform, requestedWidth?: number, viewportWidth = 0) {
    if (requestedWidth) return requestedWidth;
    const maxWidth = viewportWidth > 0 ? Math.max(320, viewportWidth - 24) : 920;
    const scaled = Math.round(node.width * Math.max(viewport.k, 0.05) * 1.5);
    const preferred = Math.max(NODE_PROMPT_PANEL_MIN_WIDTH, scaled);
    return clamp(preferred, Math.min(NODE_PROMPT_PANEL_MIN_WIDTH, maxWidth), Math.min(920, maxWidth));
}

export function CanvasConnectionCreateMenu({
    pending,
    viewport,
    viewportSize,
    containerRef,
    canCreateDrawing,
    getDisabledReason,
    onCreate,
    onClose,
}: {
    pending: PendingConnectionCreate;
    viewport: ViewportTransform;
    viewportSize: { width: number; height: number };
    containerRef: RefObject<HTMLDivElement | null>;
    canCreateDrawing: boolean;
    getDisabledReason: (
        type: CanvasNodeType.Image | CanvasNodeType.Text | CanvasNodeType.Script | CanvasNodeType.BatchTable | CanvasNodeType.Video | CanvasNodeType.Audio | CanvasNodeType.Drawing | CanvasNodeType.Config | CanvasNodeType.MediaConversion,
        provider?: "runninghub",
    ) => string;
    onCreate: (
        type: CanvasNodeType.Image | CanvasNodeType.Text | CanvasNodeType.Script | CanvasNodeType.BatchTable | CanvasNodeType.Video | CanvasNodeType.Audio | CanvasNodeType.Drawing | CanvasNodeType.Config | CanvasNodeType.MediaConversion,
        provider?: "runninghub",
    ) => void;
    onClose: () => void;
}) {
    const theme = canvasThemes[useActiveTheme()];
    const reducedMotion = useReducedMotion();
    const menuRef = useRef<HTMLDivElement>(null);
    const [activeOption, setActiveOption] = useState<string | null>(null);
    const lastPointerRef = useRef<Position | null>(null);
    const { bringToFront, zIndex } = useCanvasOverlayLayer("connection-create-menu", "var(--z-modal-overlay)");
    const menuWidth = Math.min(288, viewportSize.width - 24);
    const menuHeight = canCreateDrawing ? 448 : 404;
    const gap = 12;
    const initialPosition = getConnectionMenuPosition(pending.position, viewport, viewportSize, menuWidth, menuHeight, gap);

    useLayoutEffect(() => {
        bringToFront();
    }, [bringToFront]);

    useLayoutEffect(() => {
        const container = containerRef.current;
        const menu = menuRef.current;
        if (!container || !menu) return;
        const update = (nextViewport: ViewportTransform) => {
            const containerBounds = container.getBoundingClientRect();
            const position = getConnectionMenuPosition(pending.position, nextViewport, { width: containerBounds.width, height: containerBounds.height }, menu.offsetWidth || menuWidth, menu.offsetHeight || menuHeight, gap);
            menu.style.left = `${position.left}px`;
            menu.style.top = `${position.top}px`;
        };
        update(viewport);
        return subscribeCanvasViewportPreview(container, update);
    }, [containerRef, pending.position, viewport, viewportSize.height, viewportSize.width]);

    return (
        <motion.div
            ref={menuRef}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: aceternityMotion.duration.instant, ease: aceternityMotion.easing.enter }}
            className="thin-scrollbar absolute origin-top-left overflow-x-hidden overflow-y-auto rounded-[var(--r-2xl)] border p-2"
            data-canvas-no-zoom
            data-connection-create-menu
            aria-label="创建下一步"
            onKeyDown={(event) => {
                if (event.key === "Escape") {
                    event.stopPropagation();
                    onClose();
                }
            }}
            style={{ width: menuWidth, maxHeight: Math.max(120, viewportSize.height - 108), left: initialPosition.left, top: initialPosition.top, zIndex, background: theme.spatial.elevated, borderColor: theme.toolbar.border, color: theme.node.text }}
            onMouseDown={(event) => event.stopPropagation()}
            onPointerDownCapture={bringToFront}
            onFocusCapture={(event) => {
                bringToFront();
                if (event.target.matches(":focus-visible")) setActiveOption(event.target.closest<HTMLElement>("[data-create-option]")?.dataset.createOption || null);
            }}
            onBlurCapture={(event) => {
                if (!event.currentTarget.contains(event.relatedTarget)) setActiveOption(null);
            }}
            onPointerMove={(event) => {
                if (event.pointerType === "touch") return;
                const previous = lastPointerRef.current;
                // Layout changes can retarget a stationary pointer; only real movement selects a new row.
                if (previous?.x === event.clientX && previous.y === event.clientY) return;
                lastPointerRef.current = { x: event.clientX, y: event.clientY };
                const option = (event.target as Element).closest<HTMLElement>("[data-create-option]")?.dataset.createOption;
                if (option) setActiveOption(option);
            }}
            onPointerLeave={() => {
                lastPointerRef.current = null;
                setActiveOption(null);
            }}
            onPointerDown={(event) => event.stopPropagation()}
        >
            <div className="grid min-w-0 grid-cols-1 gap-1">
                <ConnectionCreateOption
                    expanded={activeOption === "文本生成"}
                    motionEnabled={!reducedMotion}
                    icon={<List className="size-4" />}
                    title="文本生成"
                    description="引用当前内容，生成或改写文本"
                    disabledReason={getDisabledReason(CanvasNodeType.Text)}
                    onClick={() => onCreate(CanvasNodeType.Text)}
                />
                <ConnectionCreateOption
                    expanded={activeOption === "分镜脚本"}
                    motionEnabled={!reducedMotion}
                    icon={<Clapperboard className="size-4" />}
                    title="分镜脚本"
                    description="根据剧情拆解镜头，编排分镜脚本"
                    disabledReason={getDisabledReason(CanvasNodeType.Script)}
                    onClick={() => onCreate(CanvasNodeType.Script)}
                />
                <ConnectionCreateOption
                    expanded={activeOption === "批量创作表"}
                    motionEnabled={!reducedMotion}
                    icon={<Table2 className="size-4" />}
                    title="批量创作表"
                    description="汇总多张图片，批量执行换装或创意生图"
                    disabledReason={getDisabledReason(CanvasNodeType.BatchTable)}
                    onClick={() => onCreate(CanvasNodeType.BatchTable)}
                />
                <ConnectionCreateOption
                    expanded={activeOption === "图片生成"}
                    motionEnabled={!reducedMotion}
                    icon={<ImageIcon className="size-4" />}
                    title="图片生成"
                    description="结合提示词和参考图，生成新的画面"
                    disabledReason={getDisabledReason(CanvasNodeType.Image)}
                    onClick={() => onCreate(CanvasNodeType.Image)}
                />
                <ConnectionCreateOption
                    expanded={activeOption === "生成配置"}
                    motionEnabled={!reducedMotion}
                    icon={<WorkflowIcon className="size-4" />}
                    title="生成配置"
                    description="选择模型，或使用已启用的工作流插件"
                    disabledReason={getDisabledReason(CanvasNodeType.Config)}
                    onClick={() => onCreate(CanvasNodeType.Config)}
                />
                {canCreateDrawing ? (
                    <ConnectionCreateOption
                        expanded={activeOption === "绘图"}
                        motionEnabled={!reducedMotion}
                        icon={<Pencil className="size-4" />}
                        title="绘图"
                        description="以参考图片为底图，自由绘制和标注"
                        disabledReason={getDisabledReason(CanvasNodeType.Drawing)}
                        onClick={() => onCreate(CanvasNodeType.Drawing)}
                    />
                ) : null}
                <ConnectionCreateOption
                    expanded={activeOption === "视频生成"}
                    motionEnabled={!reducedMotion}
                    icon={<Video className="size-4" />}
                    title="视频生成"
                    description="结合提示词与参考素材，生成动态视频"
                    disabledReason={getDisabledReason(CanvasNodeType.Video)}
                    onClick={() => onCreate(CanvasNodeType.Video)}
                />
                <ConnectionCreateOption
                    expanded={activeOption === "音频参考"}
                    motionEnabled={!reducedMotion}
                    icon={<Music2 className="size-4" />}
                    title="音频参考"
                    description="连接文本或角色卡，创建音频生成节点"
                    disabledReason={getDisabledReason(CanvasNodeType.Audio)}
                    onClick={() => onCreate(CanvasNodeType.Audio)}
                />
                <ConnectionCreateOption
                    expanded={activeOption === "转换"}
                    motionEnabled={!reducedMotion}
                    icon={<Settings2 className="size-4" />}
                    title="转换"
                    description="本地处理图片或视频"
                    disabledReason={getDisabledReason(CanvasNodeType.MediaConversion)}
                    onClick={() => onCreate(CanvasNodeType.MediaConversion)}
                />
            </div>
        </motion.div>
    );
}

function ConnectionCreateOption({
    expanded,
    motionEnabled,
    icon,
    title,
    description,
    disabledReason,
    onClick,
}: {
    expanded: boolean;
    motionEnabled: boolean;
    icon: ReactNode;
    title: string;
    description: string;
    disabledReason?: string;
    onClick: () => void;
}) {
    const theme = canvasThemes[useActiveTheme()];
    return (
        <button
            type="button"
            aria-disabled={Boolean(disabledReason)}
            aria-label={title}
            aria-description={disabledReason || description}
            data-create-option={title}
            data-expanded={expanded}
            data-motion={motionEnabled ? "enabled" : "reduced"}
            className="canvas-connection-create-option group flex min-h-10 w-full cursor-pointer items-start gap-2 rounded-[var(--dock-item-radius)] px-2 py-1.5 text-left outline-none focus-visible:ring-2 aria-disabled:cursor-not-allowed aria-disabled:opacity-40"
            style={{ color: theme.node.text, "--tw-ring-color": theme.node.muted, background: expanded ? theme.toolbar.itemHover : undefined } as CSSProperties}
            onClick={() => {
                if (!disabledReason) onClick();
            }}
        >
            <span className="grid size-7 shrink-0 place-items-center rounded-[var(--r-md)] opacity-65 transition-opacity group-hover:opacity-100 [&_svg]:size-3.5" style={{ background: theme.toolbar.itemHover }}>
                {icon}
            </span>
            <span className="min-w-0 flex-1 pt-1.5">
                <span className="flex items-center gap-2 text-[var(--fs-tiny)] font-semibold leading-4">{title}</span>
                <span aria-hidden="true" className="canvas-connection-create-description" style={{ color: theme.node.muted }}>
                    <span className="min-h-0 overflow-hidden">
                        <span className="block pt-1 whitespace-normal break-words text-[var(--fs-micro)] leading-relaxed">{disabledReason || description}</span>
                    </span>
                </span>
            </span>
        </button>
    );
}

const OVERLAY_GAP = 10;
const OVERLAY_MARGIN = 12;
const PANEL_DISMISS_IGNORE_SELECTOR = [
    "[data-canvas-main-dock]",
    ".canvas-node-toolbar",
    "[data-canvas-selection-toolbar]",
    ".canvas-topbar",
    ".canvas-agent-panel",
    ".canvas-mini-map",
    "[data-canvas-chrome]",
    "[data-connection-create-menu]",
    ".ant-modal",
    ".ant-modal-wrap",
    ".ant-popover",
    ".ant-dropdown",
    ".ant-select-dropdown",
    ".ant-picker-dropdown",
    ".yc-popover",
    ".yc-dropdown",
    ".yc-select-dropdown",
    ".sg-popover",
    ".sg-menu",
    ".canvas-generation-settings-popover",
    ".canvas-model-picker-popover",
    ".canvas-preset-picker-popover",
    ".canvas-composer-popover-surface",
    ".canvas-node-toolbar-menu",
    ".canvas-prompt-optimizer-popover-surface",
].join(",");

function clamp(value: number, min: number, max: number) {
    return Math.min(Math.max(value, min), max);
}

function pointInRect(x: number, y: number, rect: DOMRect, pad = 0) {
    return x >= rect.left - pad && x <= rect.right + pad && y >= rect.top - pad && y <= rect.bottom + pad;
}

function overlayBoundsRect(container: HTMLElement | null) {
    if (!container) return new DOMRect(0, 0, 0, 0);
    const stage = container.closest("[data-canvas-stage]");
    return (stage instanceof HTMLElement ? stage : container).getBoundingClientRect();
}

type OverlaySafeRect = { left: number; top: number; right: number; bottom: number };

function safeOverlayRect(stage: DOMRect): OverlaySafeRect {
    let left = Math.max(OVERLAY_MARGIN, stage.left + OVERLAY_MARGIN);
    let top = Math.max(OVERLAY_MARGIN, stage.top + OVERLAY_MARGIN);
    let right = Math.min(window.innerWidth - OVERLAY_MARGIN, stage.right - OVERLAY_MARGIN);
    let bottom = Math.min(window.innerHeight - OVERLAY_MARGIN, stage.bottom - OVERLAY_MARGIN);
    const dock = document.querySelector("[data-canvas-main-dock], .canvas-node-toolbar, [data-canvas-selection-toolbar]");
    if (dock instanceof HTMLElement) {
        const dockRect = dock.getBoundingClientRect();
        if (dockRect.width > 8 && dockRect.height > 8 && dockRect.top < bottom && dockRect.bottom > top) bottom = Math.min(bottom, dockRect.top - 8);
    }
    if (right < left) right = left;
    if (bottom < top) bottom = top;
    return { left, top, right, bottom };
}

function nodeAnchorElement(container: HTMLElement, nodeId: string) {
    const nodeElement = container.querySelector<HTMLElement>(`[data-node-id="${CSS.escape(nodeId)}"]`);
    if (!nodeElement) return null;
    return { nodeElement, shell: nodeElement.querySelector<HTMLElement>(".canvas-node-shell") || nodeElement };
}

function nodeAboveAnchorTop(nodeElement: HTMLElement, shellRect: DOMRect) {
    const header = nodeElement.querySelector(".canvas-node-external-header");
    if (!(header instanceof HTMLElement)) return shellRect.top;
    const headerRect = header.getBoundingClientRect();
    if (headerRect.height < 2 || headerRect.width < 2) return shellRect.top;
    return Math.min(shellRect.top, headerRect.top);
}

function panelVisualElement(panel: HTMLElement) {
    return panel.querySelector<HTMLElement>(".canvas-node-composer") || panel.querySelector<HTMLElement>("[data-canvas-panel-content]") || panel;
}

function fallbackViewportPosition(node: CanvasNodeData, viewport: ViewportTransform, bounds: DOMRect, panelWidth: number, panelHeight: number, dragOffset?: Position | null) {
    const local = getNodePanelPosition(node, viewport, { width: bounds.width, height: bounds.height }, panelWidth, panelHeight, dragOffset);
    return { left: bounds.left + local.left, top: bounds.top + local.top };
}

function alignFixedPanel(panel: HTMLElement, nodeRect: DOMRect, aboveAnchorTop: number, safe: OverlaySafeRect) {
    const visual = panelVisualElement(panel);
    const content = panel.querySelector<HTMLElement>("[data-canvas-panel-content]");
    const available = Math.max(160, safe.bottom - safe.top);
    if (content) {
        const natural = content.scrollHeight || visual.getBoundingClientRect().height;
        if (natural > available + 1) {
            content.style.maxHeight = `${Math.floor(available)}px`;
            content.style.overflowY = "auto";
        } else {
            content.style.maxHeight = "none";
            content.style.overflowY = "visible";
        }
    }
    const measured = visual.getBoundingClientRect();
    const width = Math.max(1, measured.width || panel.offsetWidth || 1);
    const height = Math.max(1, measured.height || panel.offsetHeight || 1);
    const desiredLeft = clamp(nodeRect.left + nodeRect.width / 2 - width / 2, safe.left, Math.max(safe.left, safe.right - width));
    const below = nodeRect.bottom + OVERLAY_GAP;
    const above = aboveAnchorTop - OVERLAY_GAP - height;
    const fitsBelow = below + height <= safe.bottom + 0.5;
    const fitsAbove = above >= safe.top - 0.5;
    const preferredTop = fitsBelow || !fitsAbove && safe.bottom - nodeRect.bottom >= nodeRect.top - safe.top ? below : above;
    const desiredTop = clamp(preferredTop, safe.top, Math.max(safe.top, safe.bottom - height));
    const dx = desiredLeft - measured.left;
    const dy = desiredTop - measured.top;
    if (Math.abs(dx) >= 0.5) panel.style.left = `${Math.round((parseFloat(panel.style.left) || 0) + dx)}px`;
    if (Math.abs(dy) >= 0.5) panel.style.top = `${Math.round((parseFloat(panel.style.top) || 0) + dy)}px`;
    panel.style.visibility = "visible";
}

function getConnectionMenuPosition(position: Position, viewport: ViewportTransform, viewportSize: { width: number; height: number }, menuWidth: number, menuHeight: number, gap: number) {
    const screenX = viewport.x + position.x * viewport.k;
    const screenY = viewport.y + position.y * viewport.k;
    return {
        left: clamp(screenX, gap, Math.max(gap, viewportSize.width - menuWidth - gap)),
        top: clamp(screenY, 52, Math.max(52, viewportSize.height - menuHeight - Math.max(gap, 56))),
    };
}

function placePanelVertically(nodeTop: number, nodeBottom: number, viewportHeight: number, panelHeight: number) {
    const gap = OVERLAY_GAP;
    const marginTop = 12;
    const marginBottom = 56;
    const height = Math.min(Math.max(panelHeight, 80), Math.max(80, viewportHeight - marginTop - marginBottom));
    const belowTop = nodeBottom + gap;
    const aboveTop = nodeTop - gap - height;
    const minTop = marginTop;
    const maxTop = Math.max(minTop, viewportHeight - height - marginBottom);
    if (belowTop <= maxTop) return { top: Math.max(minTop, belowTop), placement: "below" as const };
    if (aboveTop >= minTop) return { top: aboveTop, placement: "above" as const };
    return { top: clamp(belowTop, minTop, maxTop), placement: "dock" as const };
}

export function getNodePanelPosition(node: CanvasNodeData, viewport: ViewportTransform, viewportSize: { width: number; height: number }, panelWidth: number, panelHeight = 240, dragOffset?: Position | null) {
    const marginX = 12;
    const offsetX = dragOffset?.x || 0;
    const offsetY = dragOffset?.y || 0;
    const width = Math.min(panelWidth, Math.max(160, viewportSize.width - marginX * 2));
    const nodeCenterX = viewport.x + (node.position.x + offsetX + node.width / 2) * viewport.k;
    const nodeBottom = viewport.y + (node.position.y + offsetY + node.height) * viewport.k;
    const nodeTop = viewport.y + (node.position.y + offsetY) * viewport.k;
    const left = clamp(nodeCenterX - width / 2, marginX, Math.max(marginX, viewportSize.width - width - marginX));
    const vertical = placePanelVertically(nodeTop, nodeBottom, viewportSize.height, panelHeight);
    return { left, top: vertical.top, placement: vertical.placement };
}
