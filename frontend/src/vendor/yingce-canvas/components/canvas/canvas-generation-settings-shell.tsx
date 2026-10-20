// @ts-nocheck
import { canvasOverlayHost } from "@/features/video/canvas-host/design-system/popup-container";
import { createPortal } from "react-dom";
import { useLayoutEffect, useState, type CSSProperties, type ReactNode, type RefObject } from "react";

export type GenerationSettingsPlacement = "topLeft" | "top" | "topRight" | "bottomLeft" | "bottom" | "bottomRight";

export function generationSettingsHost() {
    return canvasOverlayHost();
}

export function generationSettingsStyle(
    buttonRect: DOMRect,
    placement: GenerationSettingsPlacement | undefined,
    width: number,
    estimatedHeight = 320,
    measuredHeight?: number,
): CSSProperties {
    const gap = 8;
    const margin = 12;
    const maxWidth = Math.min(width, window.innerWidth - margin * 2);
    const alignRight = placement?.endsWith("Right");
    const alignCenter = placement === "top" || placement === "bottom";
    const left = alignCenter ? buttonRect.left + buttonRect.width / 2 - maxWidth / 2 : alignRight ? buttonRect.right - maxWidth : buttonRect.left;
    const preferredAbove = Boolean(placement?.startsWith("top"));
    const topSpace = Math.max(0, buttonRect.top - gap - margin);
    const bottomSpace = Math.max(0, window.innerHeight - buttonRect.bottom - gap - margin);
    const needed = measuredHeight || estimatedHeight;
    const placeAbove = topSpace >= needed
        ? preferredAbove || bottomSpace < Math.min(needed, 200)
        : bottomSpace >= needed
            ? false
            : topSpace >= bottomSpace;
    const available = placeAbove ? topSpace : bottomSpace;
    const maxHeight = Math.max(120, available);
    return {
        position: "fixed",
        zIndex: 200,
        boxSizing: "border-box",
        width: maxWidth,
        minWidth: 0,
        left: Math.max(margin, Math.min(window.innerWidth - maxWidth - margin, left)),
        padding: 12,
        ...(placeAbove
            ? { bottom: window.innerHeight - buttonRect.top + gap, top: "auto", maxHeight }
            : { top: buttonRect.bottom + gap, bottom: "auto", maxHeight }),
        overflowX: "hidden",
        overflowY: "auto",
        overscrollBehavior: "contain",
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
    const [measuredHeight, setMeasuredHeight] = useState<number | undefined>();
    const style = generationSettingsStyle(buttonRect, placement, width, estimatedHeight, measuredHeight);

    useLayoutEffect(() => {
        const panel = panelRef.current;
        if (!panel) return;
        const intrinsic = panel.scrollHeight;
        setMeasuredHeight((current) => (current === intrinsic ? current : intrinsic));
    }, [buttonRect, children, estimatedHeight, panelRef, placement, width]);

    return createPortal(
        <div
            ref={panelRef}
            className="sg-popover canvas-generation-settings-popover thin-scrollbar"
            style={style}
            onPointerDown={(event) => event.stopPropagation()}
            onMouseDown={(event) => event.stopPropagation()}
            onClick={(event) => event.stopPropagation()}
            onWheel={(event) => event.stopPropagation()}
        >
            {children}
        </div>,
        generationSettingsHost(),
    );
}
