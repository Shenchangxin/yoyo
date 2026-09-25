// @ts-nocheck
import { Dropdown, Popover, type DropdownProps, type PopoverProps } from "antd";
import { useCallback, useRef, useState, type ReactNode } from "react";

import { useDismissOnOutside } from "@yingce/hooks/use-dismiss-on-outside";

/**
 * 画布里的 Ant Dropdown：点空白/画布即可收起（捕获阶段），浮层挂到 body 避免被裁切。
 */
export function CanvasDropdown({ children, open: openProp, onOpenChange, getPopupContainer, ...rest }: DropdownProps) {
    const triggerRef = useRef<HTMLSpanElement>(null);
    const [uncontrolledOpen, setUncontrolledOpen] = useState(false);
    const isControlled = openProp !== undefined;
    const open = isControlled ? openProp : uncontrolledOpen;
    const setOpen = useCallback(
        (next: boolean) => {
            if (!isControlled) setUncontrolledOpen(next);
            onOpenChange?.(next);
        },
        [isControlled, onOpenChange],
    );
    const close = useCallback(() => setOpen(false), [setOpen]);
    useDismissOnOutside(Boolean(open), close, triggerRef);

    return (
        <Dropdown {...rest} open={open} onOpenChange={setOpen} getPopupContainer={getPopupContainer ?? (() => document.body)}>
            <span ref={triggerRef} className="inline-flex min-w-0 max-w-full align-middle">
                {children as ReactNode}
            </span>
        </Dropdown>
    );
}

/** 画布里的 Ant Popover：同样支持点空白收起。 */
export function CanvasPopover({ children, open: openProp, onOpenChange, getPopupContainer, ...rest }: PopoverProps) {
    const triggerRef = useRef<HTMLSpanElement>(null);
    const [uncontrolledOpen, setUncontrolledOpen] = useState(false);
    const isControlled = openProp !== undefined;
    const open = isControlled ? openProp : uncontrolledOpen;
    const setOpen = useCallback(
        (next: boolean) => {
            if (!isControlled) setUncontrolledOpen(next);
            onOpenChange?.(next);
        },
        [isControlled, onOpenChange],
    );
    const close = useCallback(() => setOpen(false), [setOpen]);
    useDismissOnOutside(Boolean(open), close, triggerRef);

    return (
        <Popover {...rest} open={open} onOpenChange={setOpen} getPopupContainer={getPopupContainer ?? (() => document.body)}>
            <span ref={triggerRef} className="inline-flex min-w-0 max-w-full align-middle">
                {children as ReactNode}
            </span>
        </Popover>
    );
}
