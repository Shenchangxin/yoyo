// @ts-nocheck
import { motion, useMotionValue, useReducedMotion, useSpring, useTransform, type MotionValue } from "motion/react";
import { forwardRef, useEffect, useRef, useState, type CSSProperties, type MouseEvent, type ReactNode } from "react";
import { Tooltip } from "antd";

import { cn } from "@yingce/lib/utils";
import { aceternityMotion } from "@yingce/lib/aceternity-motion";

export type FloatingDockCommand = {
    kind?: "command";
    id: string;
    label: string;
    displayLabel?: string;
    badge?: ReactNode;
    icon: ReactNode;
    wide?: boolean;
    quiet?: boolean;
    onClick?: (event: MouseEvent<HTMLButtonElement>) => void;
    active?: boolean;
    disabled?: boolean;
    danger?: boolean;
    /** 面板展开型工具——使用 aria-expanded 而非 aria-pressed */
    expands?: boolean;
};

export type FloatingDockSwitchOption = {
    id: string;
    label: string;
    icon: ReactNode;
    value: string;
    displayLabel?: string;
};

export type FloatingDockSwitch = {
    kind: "switch";
    id: string;
    label: string;
    value: string;
    options: FloatingDockSwitchOption[];
    onChange: (value: string) => void;
};

export type FloatingDockEntry = FloatingDockCommand | FloatingDockSwitch | { kind: "separator"; id: string };

type FloatingDockProps = {
    items: FloatingDockEntry[];
    size?: "default" | "compact";
    embedded?: boolean;
    className?: string;
    style?: CSSProperties;
    ariaLabel?: string;
    showLabels?: boolean;
    tooltipPlacement?: "top" | "bottom";
};

type DockMetrics = {
    base: number;
    magnified: number;
    icon: number;
    iconMagnified: number;
    distance: number;
};

// 桌面 dock 收紧常态与悬浮尺寸；触屏尺寸单独保留以保证点击目标。
const DOCK_METRICS: Record<NonNullable<FloatingDockProps["size"]>, DockMetrics> = {
    default: { base: 28, magnified: 28, icon: 15, iconMagnified: 15, distance: 0 },
    compact: { base: 28, magnified: 28, icon: 15, iconMagnified: 15, distance: 0 },
};

// `window` 存在不代表 `matchMedia` 存在：测试与 renderToString 下它可能是 undefined，
// 渲染期直接取用会抛 TypeError。指针能力是纯增强，取不到就按"非触屏"处理。
function coarsePointerQuery(): MediaQueryList | undefined {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") return undefined;
    return window.matchMedia("(pointer: coarse)");
}

const TOUCH_DOCK_METRICS: Record<NonNullable<FloatingDockProps["size"]>, DockMetrics> = {
    default: { base: 40, magnified: 40, icon: 18, iconMagnified: 18, distance: 0 },
    compact: { base: 36, magnified: 36, icon: 16, iconMagnified: 16, distance: 0 },
};

function uniqueDockItems(items: FloatingDockEntry[]): FloatingDockEntry[] {
    const seen = new Set<string>();
    const next: FloatingDockEntry[] = [];
    for (const item of items) {
        if (item.kind === "separator") {
            next.push(item);
            continue;
        }
        if (seen.has(item.id)) continue;
        seen.add(item.id);
        next.push(item);
    }
    return next;
}

export const FloatingDock = forwardRef<HTMLDivElement, FloatingDockProps>(function FloatingDock({ items, size = "default", embedded = false, className, style, ariaLabel = "画布工具", showLabels = false, tooltipPlacement = "top" }, forwardedRef) {
    const mouseX = useMotionValue(Number.POSITIVE_INFINITY);
    const [coarsePointer, setCoarsePointer] = useState(() => coarsePointerQuery()?.matches ?? false);

    useEffect(() => {
        const media = coarsePointerQuery();
        if (!media) return;
        const update = () => setCoarsePointer(media.matches);
        update();
        media.addEventListener("change", update);
        return () => media.removeEventListener("change", update);
    }, []);

    const motionEnabled = false;
    const metrics = coarsePointer ? TOUCH_DOCK_METRICS[size] : DOCK_METRICS[size];

    return (
        <motion.div
            ref={forwardedRef}
            role="toolbar"
            aria-label={ariaLabel}
            className={cn(
                "aceternity-floating-dock flex items-center overflow-x-auto overflow-y-hidden [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden",
                embedded ? "shadow-none" : "border backdrop-blur-2xl",
                showLabels
                    ? embedded
                        ? size === "compact"
                            ? "h-9 gap-0.5 px-0.5"
                            : "h-10 gap-0.5 px-0.5"
                        : size === "compact"
                          ? "h-10 gap-0.5 rounded-[var(--dock-radius-compact)] px-1.5"
                          : "h-11 gap-0.5 rounded-[var(--dock-radius-tight)] px-2"
                    : coarsePointer
                      ? embedded
                          ? size === "compact"
                              ? "h-10 gap-1 px-0.5"
                              : "h-11 gap-1 px-0.5"
                          : size === "compact"
                            ? "h-11 gap-1 rounded-[var(--dock-radius-tight)] px-1.5"
                            : "h-12 gap-1 rounded-[var(--panel-radius)] px-2"
                      : embedded
                        ? size === "compact"
                            ? "h-8 gap-0.5 px-0.5"
                            : "h-9 gap-0.5 px-0.5"
                        : size === "compact"
                          ? "h-8 gap-0.5 rounded-[var(--r-lg)] px-1"
                          : "h-9 gap-0.5 rounded-[var(--dock-radius)] px-1.5",
                className,
            )}
            style={style}
            onPointerMove={(event) => {
                if (motionEnabled) mouseX.set(event.clientX);
            }}
            onPointerLeave={() => mouseX.set(Number.POSITIVE_INFINITY)}
        >
            {renderDockItems(uniqueDockItems(items), { mouseX, metrics, motionEnabled: motionEnabled && !showLabels, compact: size === "compact", showLabel: showLabels, tooltipPlacement })}
        </motion.div>
    );
});

type DockItemRenderProps = {
    mouseX: MotionValue<number>;
    metrics: DockMetrics;
    motionEnabled: boolean;
    compact: boolean;
    showLabel: boolean;
    tooltipPlacement: "top" | "bottom";
};

/**
 * 渲染 Dock 条目，将连续的 danger 命令包裹在 is-danger-group 容器中实现视觉隔离。
 * 满足"危险操作必须与常规按钮隔离"的硬约束。
 */
function renderDockItems(items: FloatingDockEntry[], props: DockItemRenderProps) {
    const result: ReactNode[] = [];
    let dangerGroup: FloatingDockCommand[] = [];
    let index = 0;

    const flushDangerGroup = () => {
        if (!dangerGroup.length) return;
        const groupKey = `danger-group-${index}`;
        result.push(
            <span key={groupKey} className="aceternity-dock-danger-group flex shrink-0 items-center gap-px">
                {dangerGroup.map((command) => (
                    <DockCommandButton key={command.id} command={command} mouseX={props.mouseX} metrics={props.metrics} motionEnabled={props.motionEnabled} compact={props.compact} showLabel={props.showLabel} tooltipPlacement={props.tooltipPlacement} />
                ))}
            </span>,
        );
        dangerGroup = [];
    };

    for (const item of items) {
        if (item.kind === "separator") {
            flushDangerGroup();
            result.push(<DockSeparator key={item.id} compact={props.compact} labeled={props.showLabel} />);
            continue;
        }
        if (item.kind === "switch") {
            flushDangerGroup();
            result.push(<DockSwitch key={item.id} entry={item} compact={props.compact} showLabel={props.showLabel} motionEnabled={props.motionEnabled} metrics={props.metrics} tooltipPlacement={props.tooltipPlacement} />);
            index += 1;
            continue;
        }
        if (item.danger) {
            dangerGroup.push(item);
            index += 1;
            continue;
        }
        flushDangerGroup();
        result.push(<DockCommandButton key={item.id} command={item} mouseX={props.mouseX} metrics={props.metrics} motionEnabled={props.motionEnabled} compact={props.compact} showLabel={props.showLabel} tooltipPlacement={props.tooltipPlacement} />);
        index += 1;
    }
    flushDangerGroup();
    return result;
}

function DockCommandButton({ command, mouseX, metrics, motionEnabled, compact: _compact, showLabel, tooltipPlacement }: { command: FloatingDockCommand; mouseX: MotionValue<number>; metrics: DockMetrics; motionEnabled: boolean; compact: boolean; showLabel: boolean; tooltipPlacement: "top" | "bottom" }) {
    const ref = useRef<HTMLSpanElement>(null);
    const distance = useTransform(mouseX, (value) => {
        const bounds = ref.current?.getBoundingClientRect();
        if (!bounds || !Number.isFinite(value)) return Number.POSITIVE_INFINITY;
        return value - bounds.left - bounds.width / 2;
    });
    const itemTarget = useTransform(distance, (value) => proximitySize(value, metrics.base, metrics.magnified, metrics.distance, motionEnabled));
    const iconTarget = useTransform(distance, (value) => proximitySize(value, metrics.icon, metrics.iconMagnified, metrics.distance, motionEnabled));
    const itemSize = useSpring(itemTarget, aceternityMotion.spring.dock);
    const iconSize = useSpring(iconTarget, aceternityMotion.spring.dock);

    if (showLabel) {
        return (
            <motion.span ref={ref} className="relative block h-8 shrink-0">
                <motion.button
                    type="button"
                    aria-label={command.label}
                    aria-expanded={command.expands ? command.active || undefined : undefined}
                    aria-pressed={command.expands ? undefined : command.active || undefined}
                    disabled={command.disabled}
                    className={cn(
                        "aceternity-dock-command is-labeled group inline-flex h-8 items-center justify-center gap-1.5 whitespace-nowrap rounded-[var(--dock-item-radius)] border-0 px-2.5 outline-none",
                        command.active && "is-active",
                        command.danger && "is-danger",
                    )}
                    whileTap={!command.disabled ? { scale: 0.96 } : undefined}
                    transition={aceternityMotion.spring.dock}
                    onClick={command.onClick}
                >
                    <span className="grid size-3.5 shrink-0 place-items-center">{command.icon}</span>
                    <span className="inline-flex h-4 items-center text-[var(--fs-label)] font-medium leading-none">{command.displayLabel || command.label}</span>
                    {command.badge !== undefined ? <span className="aceternity-dock-command-badge inline-flex min-w-4 items-center justify-center rounded-full px-1 text-[var(--fs-micro)] font-bold leading-4">{command.badge}</span> : null}
                </motion.button>
            </motion.span>
        );
    }

    return (
        <motion.span ref={ref} className={cn("relative block shrink-0", command.wide && "min-w-[var(--dock-precision-width)]")} style={{ width: itemSize, height: itemSize }}>
            <Tooltip title={command.label} placement={tooltipPlacement} mouseEnterDelay={0.15}>
            <motion.button
                type="button"
                aria-label={command.label}
                aria-expanded={command.expands ? command.active || undefined : undefined}
                aria-pressed={command.expands ? undefined : command.active || undefined}
                disabled={command.disabled}
                className={cn("aceternity-dock-command group relative grid size-full place-items-center rounded-[var(--dock-item-radius)] border-0 outline-none", command.quiet && "is-quiet", command.active && "is-active", command.danger && "is-danger")}
                whileTap={motionEnabled && !command.disabled ? { scale: 0.92 } : undefined}
                transition={aceternityMotion.spring.dock}
                onClick={command.onClick}
            >
                <motion.span className={cn("grid place-items-center", command.wide && "w-full")} style={command.wide ? { height: iconSize } : { width: iconSize, height: iconSize }}>
                    {command.icon}
                </motion.span>
            </motion.button>
            </Tooltip>
        </motion.span>
    );
}

function DockSwitch({ entry, compact, showLabel, motionEnabled, metrics, tooltipPlacement }: { entry: FloatingDockSwitch; compact: boolean; showLabel: boolean; motionEnabled: boolean; metrics: DockMetrics; tooltipPlacement: "top" | "bottom" }) {
    const reducedMotion = useReducedMotion();
    const selectedIndex = Math.max(0, entry.options.findIndex((option) => option.value === entry.value));
    const touch = metrics.base >= 40;
    const labeled = showLabel || entry.options.some((option) => option.displayLabel);
    const slot = labeled ? (touch ? 68 : compact ? 58 : 64) : touch ? 32 : 24;
    const slotHeight = labeled ? Math.min(metrics.base, 24) : slot;
    const gap = labeled ? (touch ? 4 : 3) : 1;
    const padX = labeled ? (touch ? 5 : 4) : touch ? 4 : 2;

    return (
        <span
            role="radiogroup"
            aria-label={entry.label}
            className={cn("aceternity-dock-switch relative flex shrink-0 items-center", labeled && "is-labeled")}
            onKeyDown={(event) => {
                if (event.key !== "ArrowRight" && event.key !== "ArrowLeft") return;
                event.preventDefault();
                const direction = event.key === "ArrowRight" ? 1 : -1;
                const next = entry.options[(selectedIndex + direction + entry.options.length) % entry.options.length];
                if (next) entry.onChange(next.value);
            }}
        >
            <span
                className="aceternity-dock-switch-track relative inline-flex items-center"
                style={{ gap, padding: `${touch ? 4 : 3}px ${padX}px` }}
            >
                <motion.span
                    aria-hidden
                    className={cn("aceternity-dock-switch-thumb pointer-events-none absolute top-1/2", labeled ? "rounded-[var(--dock-item-radius)]" : "rounded-[6px]")}
                    initial={false}
                    animate={{ x: selectedIndex * (slot + gap), y: "-50%" }}
                    transition={reducedMotion || !motionEnabled ? { duration: 0 } : aceternityMotion.spring.dock}
                    style={{ width: slot, height: slotHeight, left: padX }}
                />
                {entry.options.map((option) => {
                    const checked = option.value === entry.value;
                    const button = (
                        <button
                            type="button"
                            role="radio"
                            aria-checked={checked}
                            aria-label={option.label}
                            className={cn("aceternity-dock-switch-option relative z-[1] inline-flex items-center justify-center border-0 outline-none", labeled ? "gap-1 rounded-[var(--dock-item-radius)] px-2" : "rounded-[6px]")}
                            style={{ width: slot, height: slotHeight }}
                            onClick={() => {
                                if (!checked) entry.onChange(option.value);
                            }}
                        >
                            <span className="grid size-[15px] shrink-0 place-items-center">{option.icon}</span>
                            {labeled ? <span className="shrink-0 whitespace-nowrap text-[length:var(--fs-micro)] font-semibold leading-none">{option.displayLabel || option.label}</span> : null}
                        </button>
                    );
                    return (
                        <span key={option.id} className="relative inline-flex shrink-0">
                            {labeled ? button : <Tooltip title={option.label} placement={tooltipPlacement} mouseEnterDelay={0.15}>{button}</Tooltip>}
                        </span>
                    );
                })}
            </span>
        </span>
    );
}

function DockSeparator({ compact, labeled }: { compact: boolean; labeled: boolean }) {
    return <span aria-hidden className={cn("aceternity-dock-separator shrink-0 self-center", labeled ? "mx-1.5 h-3 w-px" : compact ? "mx-1 h-3 w-px" : "mx-1.5 h-3 w-px")} />;
}

function proximitySize(distance: number, base: number, magnified: number, range: number, enabled: boolean) {
    if (!enabled || !Number.isFinite(distance)) return base;
    const proximity = 1 - Math.min(Math.abs(distance) / range, 1);
    return base + (magnified - base) * proximity * proximity;
}
