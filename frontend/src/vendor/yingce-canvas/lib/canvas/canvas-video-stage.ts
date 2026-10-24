import { CANVAS_GRAPHICS_VIEWPORT_PREVIEW_EVENT, CANVAS_NODE_DRAG_PREVIEW_EVENT } from "./canvas-live-viewport.ts";

export type StageBox = { left: number; top: number; width: number; height: number };

export type CanvasVideoStageStyle = {
    left: string;
    top: string;
    width: string;
    height: string;
    borderRadius: string;
    visibility: "hidden" | "visible";
};

/** Screen-space box for a video that must not inherit the canvas world transform. */
export function canvasVideoStageStyle(anchor: StageBox, viewport: StageBox, radius: string, layoutWidth: number): CanvasVideoStageStyle {
    const width = Math.max(0, anchor.width);
    const height = Math.max(0, anchor.height);
    const scale = layoutWidth > 0 ? width / layoutWidth : 1;
    const hidden = width < 1 || height < 1
        || anchor.left >= viewport.left + viewport.width
        || anchor.top >= viewport.top + viewport.height
        || anchor.left + width <= viewport.left
        || anchor.top + height <= viewport.top;
    return {
        left: `${anchor.left - viewport.left}px`,
        top: `${anchor.top - viewport.top}px`,
        width: `${width}px`,
        height: `${height}px`,
        borderRadius: scaleRadius(radius, scale),
        visibility: hidden ? "hidden" : "visible",
    };
}

/**
 * Keeps `stage` aligned with `anchor`'s on-screen rect and outside the scaled
 * world layer. The canvas pans with transforms; a video inside that layer is
 * resampled every frame and drops frames on Windows.
 */
export function trackCanvasVideoStage(anchor: HTMLElement, stage: HTMLElement) {
    const viewport = anchor.closest<HTMLElement>("[data-canvas-viewport]");
    if (!viewport) return () => {};
    if (stage.parentElement !== viewport) viewport.appendChild(stage);
    stage.style.position = "absolute";
    stage.style.zIndex = "6";
    let last = "";
    const place = () => {
        const next = canvasVideoStageStyle(
            boxOf(anchor.getBoundingClientRect()),
            boxOf(viewport.getBoundingClientRect()),
            getComputedStyle(anchor).borderRadius,
            anchor.offsetWidth,
        );
        const key = `${next.left}|${next.top}|${next.width}|${next.height}|${next.borderRadius}|${next.visibility}`;
        if (key === last) return;
        last = key;
        stage.style.left = next.left;
        stage.style.top = next.top;
        stage.style.width = next.width;
        stage.style.height = next.height;
        stage.style.borderRadius = next.borderRadius;
        stage.style.visibility = next.visibility;
    };
    place();
    const observer = typeof ResizeObserver !== "undefined" ? new ResizeObserver(place) : null;
    observer?.observe(anchor);
    observer?.observe(viewport);
    viewport.addEventListener(CANVAS_GRAPHICS_VIEWPORT_PREVIEW_EVENT, place);
    viewport.addEventListener(CANVAS_NODE_DRAG_PREVIEW_EVENT, place);
    window.addEventListener("resize", place);
    return () => {
        observer?.disconnect();
        viewport.removeEventListener(CANVAS_GRAPHICS_VIEWPORT_PREVIEW_EVENT, place);
        viewport.removeEventListener(CANVAS_NODE_DRAG_PREVIEW_EVENT, place);
        window.removeEventListener("resize", place);
    };
}

function boxOf(rect: DOMRect): StageBox {
    return { left: rect.left, top: rect.top, width: rect.width, height: rect.height };
}

function scaleRadius(radius: string, scale: number) {
    if (!radius) return "0px";
    if (!Number.isFinite(scale) || Math.abs(scale - 1) < 0.001) return radius;
    return radius.split("/").map((part) => part.trim().split(/\s+/).map((token) => {
        if (!token.endsWith("px")) return token;
        const value = Number.parseFloat(token);
        if (!Number.isFinite(value)) return token;
        return `${Math.round(value * scale * 100) / 100}px`;
    }).join(" ")).join(" / ");
}
