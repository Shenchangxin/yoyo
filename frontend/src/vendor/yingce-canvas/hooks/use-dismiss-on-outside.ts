// @ts-nocheck
import { useEffect, type RefObject } from "react";

/** 点击这些浮层内部时不要关掉当前菜单（含级联、分割器、选择器）。 */
export const CANVAS_OVERLAY_KEEP_SELECTOR = [
    ".ant-dropdown:not(.ant-dropdown-hidden)",
    ".ant-popover:not(.ant-popover-hidden)",
    ".ant-select-dropdown",
    ".ant-picker-dropdown",
    ".ant-tooltip",
    ".canvas-grid-split-picker",
    ".sg-menu",
    ".sg-popover",
    "[data-canvas-overlay]",
    "[data-canvas-context-menu]",
].join(",");

/**
 * 画布会在 pointerdown 上 stopPropagation，Ant Design 的冒泡关闭失效。
 * 用捕获阶段在 document 上关闭，点空白/画布即可收起。
 */
export function useDismissOnOutside(
    open: boolean,
    onClose: () => void,
    triggerRef?: RefObject<Element | null>,
    keepSelector: string = CANVAS_OVERLAY_KEEP_SELECTOR,
) {
    useEffect(() => {
        if (!open) return;
        const onPointerDown = (event: PointerEvent) => {
            const target = event.target;
            if (!(target instanceof Node)) return;
            if (triggerRef?.current?.contains(target)) return;
            if (target instanceof Element && target.closest(keepSelector)) return;
            onClose();
        };
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key === "Escape") onClose();
        };
        document.addEventListener("pointerdown", onPointerDown, true);
        document.addEventListener("keydown", onKeyDown);
        return () => {
            document.removeEventListener("pointerdown", onPointerDown, true);
            document.removeEventListener("keydown", onKeyDown);
        };
    }, [keepSelector, onClose, open, triggerRef]);
}
