// @ts-nocheck
import { Tooltip } from "@yingce/components/ui/base/tooltip";
import type { ReactNode } from "react";
import { motion, useReducedMotion } from "motion/react";
import { X, ZoomIn, ZoomOut } from "lucide-react";


import { aceternityMotion } from "@yingce/lib/aceternity-motion";
import { canvasThemes } from "@yingce/lib/canvas-theme";
import { useActiveTheme } from "@yingce/stores/canvas/use-canvas-theme-store";

type CanvasFocusModeBarProps = {
    versionsOpen: boolean;
    onToggleVersions: () => void;
    syncStatus?: ReactNode;
    dockRevealed: boolean;
    zoomPercent: number;
    onToggleDock: () => void;
    onExit: () => void;
    onZoomIn: () => void;
    onZoomOut: () => void;
    onFit: () => void;
};

export function CanvasFocusModeBar({ zoomPercent, onExit, onZoomIn, onZoomOut, onFit }: CanvasFocusModeBarProps) {
    const theme = canvasThemes[useActiveTheme()];
    const reducedMotion = useReducedMotion();

    return (
        <motion.div
            initial={reducedMotion ? { opacity: 0 } : { opacity: 0, y: -12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={reducedMotion ? { duration: 0 } : aceternityMotion.spring.panel}
            className="pointer-events-auto absolute left-1/2 top-2 z-[var(--z-toolbar)] -translate-x-1/2"
            role="toolbar"
            aria-label="专注模式工具栏"
        >
            <div className="flex items-center gap-0.5 rounded-xl p-1" style={{ background: theme.spatial.elevated, color: theme.node.text, boxShadow: "0 0 0 0.5px color-mix(in srgb, var(--foreground) 12%, transparent), 0 10px 28px color-mix(in srgb, black 16%, transparent)" }}>
                <Tooltip title="退出专注模式（Esc）">
                    <button
                        type="button"
                        onClick={onExit}
                        className="grid size-8 place-items-center rounded-md transition hover:bg-black/5"
                        style={{ color: theme.node.text }}
                        aria-label="退出专注模式"
                    >
                        <X className="size-4" />
                    </button>
                </Tooltip>
                <span className="mx-0.5 h-4 w-px" style={{ background: theme.toolbar.border }} />
                <Tooltip title="缩小">
                    <button type="button" onClick={onZoomOut} className="grid size-8 place-items-center rounded-md transition hover:bg-black/5" style={{ color: theme.node.text }} aria-label="缩小">
                        <ZoomOut className="size-4" />
                    </button>
                </Tooltip>
                <button
                    type="button"
                    onClick={onFit}
                    className="grid h-8 min-w-14 place-items-center rounded-md px-2 text-[11px] font-medium tabular-nums transition hover:bg-black/5"
                    style={{ color: theme.node.text }}
                    title="适应画布"
                >
                    {Math.round(zoomPercent * 100)}%
                </button>
                <Tooltip title="放大">
                    <button type="button" onClick={onZoomIn} className="grid size-8 place-items-center rounded-md transition hover:bg-black/5" style={{ color: theme.node.text }} aria-label="放大">
                        <ZoomIn className="size-4" />
                    </button>
                </Tooltip>
            </div>
        </motion.div>
    );
}
