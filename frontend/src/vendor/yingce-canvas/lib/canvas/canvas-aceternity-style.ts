// @ts-nocheck
import type { CSSProperties } from "react";

import type { CanvasTheme } from "@yingce/lib/canvas-theme";

export function canvasDockStyle(theme: CanvasTheme, color: string = theme.toolbar.item): CSSProperties {
    return {
        background: "var(--dock-surface)",
        borderColor: "var(--dock-border)",
        color,
        boxShadow: "var(--elevation-overlay)",
        "--dock-command-bg": theme.spatial.surface,
        "--dock-command-hover": theme.toolbar.itemHover,
        "--dock-command-active": theme.toolbar.activeBg,
        "--dock-command-active-text": theme.toolbar.activeText,
        "--dock-command-danger": theme.accent.danger,
        "--dock-tooltip-bg": theme.spatial.elevated,
        "--dock-tooltip-border": theme.toolbar.border,
        "--dock-switch-track": "#2c2621",
        "--dock-switch-thumb": "#f2ede6",
        "--dock-switch-thumb-text": "#1c1916",
    } as CSSProperties;
}
