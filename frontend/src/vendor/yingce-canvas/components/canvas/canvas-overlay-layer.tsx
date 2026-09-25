// @ts-nocheck
import { createContext, useCallback, useContext, useMemo, useState, type MouseEventHandler, type PointerEventHandler, type ReactNode, type WheelEventHandler } from "react";

export type CanvasOverlayKind = "chrome" | "hud" | "menu" | "popover" | "dialog";

type CanvasOverlayLayerContextValue = {
    activeOverlayId: string | null;
    exclusive: { kind: "menu" | "popover" | "hud"; id: string } | null;
    bringToFront: (overlayId: string) => void;
    setExclusive: (kind: "menu" | "popover" | "hud", id: string) => void;
    clearExclusive: (id?: string) => void;
};

const CanvasOverlayLayerContext = createContext<CanvasOverlayLayerContextValue | null>(null);

export function CanvasOverlayLayerProvider({ children }: { children: ReactNode }) {
    const [activeOverlayId, setActiveOverlayId] = useState<string | null>(null);
    const [exclusive, setExclusiveState] = useState<{ kind: "menu" | "popover" | "hud"; id: string } | null>(null);
    const bringToFront = useCallback((overlayId: string) => {
        setActiveOverlayId((current) => (current === overlayId ? current : overlayId));
    }, []);
    const setExclusive = useCallback((kind: "menu" | "popover" | "hud", id: string) => {
        setExclusiveState({ kind, id });
    }, []);
    const clearExclusive = useCallback((id?: string) => {
        setExclusiveState((current) => (id && current?.id !== id ? current : null));
    }, []);
    const value = useMemo(() => ({ activeOverlayId, exclusive, bringToFront, setExclusive, clearExclusive }), [activeOverlayId, exclusive, bringToFront, setExclusive, clearExclusive]);

    return <CanvasOverlayLayerContext.Provider value={value}>{children}</CanvasOverlayLayerContext.Provider>;
}

export function useCanvasOverlayLayer(overlayId: string, fallbackZIndex: string, kind: CanvasOverlayKind = "chrome") {
    const context = useContext(CanvasOverlayLayerContext);
    const bringToFrontFromContext = context?.bringToFront;
    const bringToFront = useCallback(() => {
        if (kind === "chrome" || kind === "hud") return;
        bringToFrontFromContext?.(overlayId);
    }, [bringToFrontFromContext, overlayId, kind]);
    const raised = kind !== "chrome" && kind !== "hud" && context?.activeOverlayId === overlayId;
    const zIndex = raised ? "var(--z-canvas-overlay-active)" : fallbackZIndex;

    return { bringToFront, zIndex, exclusive: context?.exclusive, setExclusive: context?.setExclusive, clearExclusive: context?.clearExclusive };
}

export function CanvasOverlayLayerContainer({
    overlayId,
    fallbackZIndex,
    kind = "chrome",
    className,
    children,
    onMouseDown,
    onPointerDown,
    onWheel,
}: {
    overlayId: string;
    fallbackZIndex: string;
    kind?: CanvasOverlayKind;
    className?: string;
    children: ReactNode;
    onMouseDown?: MouseEventHandler<HTMLDivElement>;
    onPointerDown?: PointerEventHandler<HTMLDivElement>;
    onWheel?: WheelEventHandler<HTMLDivElement>;
}) {
    const { bringToFront, zIndex } = useCanvasOverlayLayer(overlayId, fallbackZIndex, kind);

    return (
        <div data-canvas-no-zoom className={className} style={{ zIndex }} onPointerDownCapture={bringToFront} onFocusCapture={bringToFront} onMouseDown={onMouseDown} onPointerDown={onPointerDown} onWheel={onWheel}>
            {children}
        </div>
    );
}
