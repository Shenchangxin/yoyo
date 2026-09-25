// @ts-nocheck
import { createPortal } from "react-dom";
import type { CSSProperties, ReactNode, RefObject } from "react";

export type GenerationSettingsPlacement = "topLeft" | "top" | "topRight" | "bottomLeft" | "bottom" | "bottomRight";

export function generationSettingsHost() {
    return document.body;
}

export function generationSettingsStyle(
    buttonRect: DOMRect,
    placement: GenerationSettingsPlacement | undefined,
    width: number,
    estimatedHeight = 320,
): CSSProperties {
    const gap = 8;
    const margin = 12;
    const maxWidth = Math.min(width, window.innerWidth - margin * 2);
    const alignRight = placement?.endsWith("Right");
    const alignCenter = placement === "top" || placement === "bottom";
    const left = alignCenter ? buttonRect.left + buttonRect.width / 2 - maxWidth / 2 : alignRight ? buttonRect.right - maxWidth : buttonRect.left;
    const topPlacement = placement?.startsWith("top");
    const topSpace = buttonRect.top - gap - margin;
    const bottomSpace = window.innerHeight - buttonRect.bottom - gap - margin;
    const placeAbove = topPlacement ? topSpace >= estimatedHeight || topSpace >= bottomSpace : bottomSpace < estimatedHeight && topSpace > bottomSpace;
    return {
        position: "fixed",
        zIndex: 200,
        width: maxWidth,
        left: Math.max(margin, Math.min(window.innerWidth - maxWidth - margin, left)),
        ...(placeAbove
            ? { bottom: window.innerHeight - buttonRect.top + gap, maxHeight: Math.max(260, topSpace) }
            : { top: buttonRect.bottom + gap, maxHeight: Math.max(260, bottomSpace) }),
        overflowY: "auto",
    };
}

export function GenerationSettingsPortal({
    buttonRect,
    panelRef,
    placement,
    width,
    estimatedHeight,
    children,
}: {
    buttonRect: DOMRect;
    panelRef: RefObject<HTMLDivElement | null>;
    placement?: GenerationSettingsPlacement;
    width: number;
    estimatedHeight?: number;
    children: ReactNode;
}) {
    return createPortal(
        <div
            ref={panelRef}
            className="sg-popover canvas-generation-settings-popover"
            style={generationSettingsStyle(buttonRect, placement, width, estimatedHeight)}
            onPointerDown={(event) => event.stopPropagation()}
            onMouseDown={(event) => event.stopPropagation()}
            onClick={(event) => event.stopPropagation()}
        >
            {children}
        </div>,
        generationSettingsHost(),
    );
}
