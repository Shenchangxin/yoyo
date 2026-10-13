// @ts-nocheck
import { useEffect, useRef, useState, type RefObject } from "react";
import { Compass, Ellipsis, Eye, EyeOff, Focus, HelpCircle, LayoutTemplate, Minus, Plus } from "lucide-react";
import { Tooltip } from "antd";

import { CanvasDropdown } from "@yingce/components/ui/canvas-overlay";

import { canvasThemes } from "@yingce/lib/canvas-theme";
import { subscribeCanvasViewportPreview } from "@yingce/lib/canvas/canvas-live-viewport";
import { useActiveTheme } from "@yingce/stores/canvas/use-canvas-theme-store";
import { useCopy } from "@/lib/i18n";
import { GenerationSettingsPortal } from "./canvas-generation-settings-shell";

type CanvasZoomControlsProps = {
    scale: number;
    onScaleChange: (scale: number) => void;
    onFitContent: () => void;
    onAutoArrange?: () => void;
    hideNodeConnections?: boolean;
    onHideNodeConnectionsChange?: (value: boolean) => void;
    isMiniMapOpen: boolean;
    onToggleMiniMap: () => void;
    onOpenShortcuts: () => void;
    containerRef?: RefObject<HTMLDivElement | null>;
};

const QUICK_ZOOM_LEVELS = [0.25, 0.5, 1, 2] as const;

export function CanvasZoomControls({ scale, onScaleChange, onFitContent, onAutoArrange, hideNodeConnections = false, onHideNodeConnectionsChange, isMiniMapOpen, onToggleMiniMap, onOpenShortcuts, containerRef }: CanvasZoomControlsProps) {
    const copy = useCopy().video.canvasBoard;
    const theme = canvasThemes[useActiveTheme()];
    const rootRef = useRef<HTMLDivElement>(null);
    const panelRef = useRef<HTMLDivElement>(null);
    const liveScaleRef = useRef(scale);
    const rangeRef = useRef<HTMLInputElement>(null);
    const dockLabelRef = useRef<HTMLSpanElement>(null);
    const panelLabelRef = useRef<HTMLSpanElement>(null);
    const [precisionOpen, setPrecisionOpen] = useState(false);
    const [buttonRect, setButtonRect] = useState<DOMRect | null>(null);

    useEffect(() => updateScaleDisplay(scale), [scale]);

    useEffect(() => {
        const container = containerRef?.current;
        if (!container) return;
        return subscribeCanvasViewportPreview(container, (viewport) => updateScaleDisplay(viewport.k));
    }, [containerRef]);

    useEffect(() => {
        if (!precisionOpen) return;
        const syncPosition = () => setButtonRect(rootRef.current?.getBoundingClientRect() || null);
        const close = (event: PointerEvent) => {
            const target = event.target;
            if (!(target instanceof Node)) return;
            if (rootRef.current?.contains(target) || panelRef.current?.contains(target)) return;
            setPrecisionOpen(false);
        };
        const closeOnEscape = (event: KeyboardEvent) => {
            if (event.key === "Escape") setPrecisionOpen(false);
        };
        syncPosition();
        window.addEventListener("resize", syncPosition);
        document.addEventListener("pointerdown", close, true);
        document.addEventListener("keydown", closeOnEscape);
        return () => {
            window.removeEventListener("resize", syncPosition);
            document.removeEventListener("pointerdown", close, true);
            document.removeEventListener("keydown", closeOnEscape);
        };
    }, [precisionOpen]);

    function updateScaleDisplay(nextScale: number) {
        liveScaleRef.current = nextScale;
        const percent = String(Math.round(nextScale * 100));
        if (rangeRef.current) rangeRef.current.value = percent;
        if (dockLabelRef.current) dockLabelRef.current.textContent = percent;
        if (panelLabelRef.current) panelLabelRef.current.textContent = `${percent}%`;
    }

    function commitScale(nextScale: number) {
        const clampedScale = Math.min(2, Math.max(0.05, nextScale));
        updateScaleDisplay(clampedScale);
        onScaleChange(clampedScale);
    }

    return (
        <div ref={rootRef} data-canvas-no-zoom className="relative z-[var(--z-toolbar)]" onMouseDown={(event) => event.stopPropagation()} onPointerDown={(event) => event.stopPropagation()} onWheel={(event) => event.stopPropagation()}>
            {precisionOpen && buttonRect ? (
                <GenerationSettingsPortal buttonRect={buttonRect} panelRef={panelRef} placement="topLeft" width={220} estimatedHeight={180}>
                    <div className="p-2.5" style={{ color: theme.node.text }}>
                        <div className="flex items-center justify-between gap-3">
                            <span className="text-[13px] font-medium">{copy.zoomTitle}</span>
                            <span ref={panelLabelRef} className="text-[12.5px] font-medium tabular-nums" style={{ color: theme.accent.primary }}>
                                {Math.round(scale * 100)}%
                            </span>
                        </div>
                        <input
                            ref={rangeRef}
                            type="range"
                            min="5"
                            max="200"
                            step="1"
                            defaultValue={Math.round(scale * 100)}
                            className="aceternity-zoom-range mt-3 h-4 w-full"
                            style={{ accentColor: theme.accent.primary }}
                            onChange={(event) => commitScale(Number(event.target.value) / 100)}
                            aria-label={copy.zoomExact}
                        />
                        <div className="mt-2.5 grid grid-cols-4 gap-1">
                            {QUICK_ZOOM_LEVELS.map((level) => (
                                <button
                                    key={level}
                                    type="button"
                                    className="sg-ghost h-7 w-auto min-w-0 text-[11px] font-medium tabular-nums"
                                    onClick={() => commitScale(level)}
                                >
                                    {Math.round(level * 100)}%
                                </button>
                            ))}
                        </div>
                    </div>
                </GenerationSettingsPortal>
            ) : null}

            <div className="canvas-zoom-capsule sg-island pointer-events-auto inline-flex items-center" role="toolbar" aria-label={copy.zoom} style={{ color: theme.node.text }}>
                <Tooltip title={copy.zoomOut} placement="top" mouseEnterDelay={0.15}>
                <button type="button" className="sg-ghost" aria-label={copy.zoomOut} onClick={() => commitScale(liveScaleRef.current - 0.1)}>
                    <Minus strokeWidth={1.55} />
                </button>
                </Tooltip>
                <span className="canvas-zoom-sep" aria-hidden />
                <Tooltip title={copy.zoomExact} placement="top" mouseEnterDelay={0.15}>
                <button type="button" className="sg-ghost canvas-zoom-pct" aria-label={copy.zoomExact} aria-expanded={precisionOpen} onClick={() => setPrecisionOpen((value) => !value)}>
                    <span ref={dockLabelRef}>{Math.round(scale * 100)}</span>
                    <span className="ml-px text-[10px] font-medium opacity-45">%</span>
                </button>
                </Tooltip>
                <span className="canvas-zoom-sep" aria-hidden />
                <Tooltip title={copy.zoomIn} placement="top" mouseEnterDelay={0.15}>
                <button type="button" className="sg-ghost" aria-label={copy.zoomIn} onClick={() => commitScale(liveScaleRef.current + 0.1)}>
                    <Plus strokeWidth={1.55} />
                </button>
                </Tooltip>
                <span className="canvas-zoom-sep" aria-hidden />
                <CanvasDropdown
                    trigger={["click"]}
                    placement="topLeft"
                    autoAdjustOverflow
                    menu={{
                        items: [
                            { key: "fit", icon: <Focus strokeWidth={1.55} />, label: copy.fit, onClick: onFitContent },
                            ...(onAutoArrange ? [{ key: "arrange", icon: <LayoutTemplate strokeWidth={1.55} />, label: copy.arrange, onClick: onAutoArrange }] : []),
                            { key: "minimap", icon: <Compass strokeWidth={1.55} />, label: isMiniMapOpen ? copy.minimapOff : copy.minimapOn, onClick: onToggleMiniMap },
                            ...(onHideNodeConnectionsChange ? [{ key: "wires", icon: hideNodeConnections ? <Eye strokeWidth={1.55} /> : <EyeOff strokeWidth={1.55} />, label: hideNodeConnections ? copy.wiresOn : copy.wiresOff, onClick: () => onHideNodeConnectionsChange(!hideNodeConnections) }] : []),
                            { type: "divider" },
                            { key: "shortcuts", icon: <HelpCircle strokeWidth={1.55} />, label: copy.shortcuts, onClick: onOpenShortcuts },
                        ],
                    }}
                >
                    <Tooltip title={copy.zoom} placement="top" mouseEnterDelay={0.15}>
                    <button type="button" className="sg-ghost" aria-label={copy.zoom}>
                        <Ellipsis strokeWidth={1.55} />
                    </button>
                    </Tooltip>
                </CanvasDropdown>
            </div>
        </div>
    );
}
