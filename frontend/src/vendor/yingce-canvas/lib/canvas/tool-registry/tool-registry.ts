// @ts-nocheck
import { ScanSearch, Settings2 } from "lucide-react";
import { createElement } from "react";

import type { FloatingDockEntry } from "@yingce/components/ui/aceternity/floating-dock";
import { ART_CRITIQUE_NODE_TYPE } from "@yingce/lib/art-critique/contracts";
import { listCreatableNodeDefinitions } from "@yingce/lib/canvas/node-registry";

import type { AddNodeMenuCommand, AddNodeMenuContext, NodeToolbarGroup, ToolCategory, ToolContext, ToolDefinition, ToolbarId, ToolbarPrefs } from "./tool-definition";

/** 模块级注册表。Vite HMR 会重复执行 definition 模块，注册必须按 id 替换而不是追加。 */
const registry = new Map<ToolbarId, ToolDefinition[]>();
const addNodeMenuRegistry: AddNodeMenuCommand[] = [];

function uniqueById<T extends { id: string }>(items: T[], mode: "first" | "last" = "last"): T[] {
    const map = new Map<string, T>();
    for (const item of items) {
        if (mode === "first" && map.has(item.id)) continue;
        map.set(item.id, item);
    }
    return [...map.values()];
}

function sortByDefaultOrder<T extends { defaultOrder?: number }>(items: T[]): T[] {
    return [...items].sort((a, b) => (a.defaultOrder ?? Number.MAX_SAFE_INTEGER) - (b.defaultOrder ?? Number.MAX_SAFE_INTEGER));
}

function upsertById<T extends { id: string }>(list: T[], item: T): T[] {
    const next = [...list];
    const index = next.findIndex((entry) => entry.id === item.id);
    if (index >= 0) next[index] = item;
    else next.push(item);
    return next;
}

/** 批量注册工具到指定工具栏 */
export function registerToolbarTools(tools: ToolDefinition[]) {
    for (const tool of tools) {
        registry.set(tool.toolbar, upsertById(registry.get(tool.toolbar) ?? [], tool));
    }
}

/** 注册添加节点菜单命令 */
export function registerAddNodeMenuCommands(commands: AddNodeMenuCommand[]) {
    let next = addNodeMenuRegistry;
    for (const command of commands) next = upsertById(next, command);
    addNodeMenuRegistry.length = 0;
    addNodeMenuRegistry.push(...next);
}

/** 获取某工具栏全部已注册工具（按 defaultOrder 升序） */
export function getToolbarTools(toolbar: ToolbarId): ToolDefinition[] {
    return sortByDefaultOrder(uniqueById(registry.get(toolbar) ?? []));
}

/** 获取添加节点菜单全部命令（按 defaultOrder 升序） */
export function getAddNodeMenuCommands(): AddNodeMenuCommand[] {
    return sortByDefaultOrder(uniqueById(addNodeMenuRegistry));
}

/**
 * 将已注册、且创建菜单可见的画布节点转为命令。
 * 内置 curated 命令优先；此处只补齐 Markdown / 图表等扩展节点和插件节点。
 */
function getCreatableNodeMenuCommands(): AddNodeMenuCommand[] {
    return listCreatableNodeDefinitions().map((definition, index) => {
        const pluginId = definition.plugin?.pluginId;
        const FallbackIcon = definition.type === ART_CRITIQUE_NODE_TYPE ? ScanSearch : Settings2;
        return {
            id: definition.type,
            label: definition.label,
            icon: definition.icon || createElement(FallbackIcon, { "aria-hidden": true }),
            section: "node",
            defaultOrder: pluginId ? 1000 + index : 200 + index,
            applicable: pluginId ? (ctx: AddNodeMenuContext) => !ctx.enabledPluginIds || ctx.enabledPluginIds.has(pluginId) : undefined,
            run: (ctx: AddNodeMenuContext) => ctx.handlers.onAddExtensionNode(definition.type),
        };
    });
}

/** 默认偏好：全部工具按 defaultOrder 排列；defaultVisible 为 false 的进入 hidden */
export function defaultToolbarPrefs(toolbar: ToolbarId): ToolbarPrefs {
    const tools = getToolbarTools(toolbar);
    return {
        order: tools.map((tool) => tool.id),
        hidden: tools.filter((tool) => !tool.defaultVisible).map((tool) => tool.id),
    };
}

/**
 * 解析工具栏条目——核心函数
 *
 * 流程：过滤 applicable → 应用用户排序 → 过滤用户隐藏 → 生成 FloatingDockEntry（含 separator 分组）
 */
export function resolveToolbarEntries(toolbar: ToolbarId, ctx: ToolContext, prefs: ToolbarPrefs | null): FloatingDockEntry[] {
    const tools = resolveToolbarTools(toolbar, ctx, prefs);
    return buildEntriesWithSeparators(tools, ctx);
}

/**
 * 解析工具栏工具——返回过滤排序后的 ToolDefinition[]（不含 separator）。
 * 供需要后处理工具列表的场景使用（如节点悬停工具栏合并图片工具）。
 */
export function resolveToolbarTools(toolbar: ToolbarId, ctx: ToolContext, prefs: ToolbarPrefs | null): ToolDefinition[] {
    const allTools = getToolbarTools(toolbar);
    const applicableTools = uniqueById(allTools.filter((tool) => !tool.applicable || tool.applicable(ctx)));
    const effectivePrefs = prefs ?? defaultToolbarPrefs(toolbar);
    const hiddenSet = new Set(effectivePrefs.hidden);
    const visibleTools = applicableTools.filter((tool) => !hiddenSet.has(tool.id));
    const orderIndex = new Map(effectivePrefs.order.map((id, index) => [id, index]));
    return [...visibleTools].sort((a, b) => {
        const ai = orderIndex.has(a.id) ? orderIndex.get(a.id)! : Number.MAX_SAFE_INTEGER;
        const bi = orderIndex.has(b.id) ? orderIndex.get(b.id)! : Number.MAX_SAFE_INTEGER;
        if (ai !== bi) return ai - bi;
        return (a.defaultOrder ?? Number.MAX_SAFE_INTEGER) - (b.defaultOrder ?? Number.MAX_SAFE_INTEGER);
    });
}

/** 将节点工具定义解析为唯一的 Dock 展示层级和排序。 */
export function resolveNodeToolbarPlacement(tool: ToolDefinition, ctx: ToolContext): { group: NodeToolbarGroup; order: number } {
    const placement = tool.nodeToolbar;
    return {
        group: typeof placement?.group === "function" ? placement.group(ctx) : placement?.group || "more",
        order: typeof placement?.order === "function" ? placement.order(ctx) : placement?.order ?? tool.defaultOrder,
    };
}

/** 解析添加节点菜单命令——合并插件/扩展节点后按 applicable 过滤并排序。 */
export function resolveAddNodeMenuCommands(ctx: AddNodeMenuContext): AddNodeMenuCommand[] {
    const merged = uniqueById([...getAddNodeMenuCommands(), ...getCreatableNodeMenuCommands()], "first");
    return sortByDefaultOrder(merged.filter((command) => !command.applicable || command.applicable(ctx)));
}

/**
 * 将工具列表转为 FloatingDockEntry，按 category 边界自动插入 separator。
 * danger 类工具会被包裹在 is-danger-group 容器中实现视觉隔离。
 */
function buildEntriesWithSeparators(tools: ToolDefinition[], ctx: ToolContext): FloatingDockEntry[] {
    const entries: FloatingDockEntry[] = [];
    let prevCategory: ToolCategory | null = null;
    let separatorIndex = 0;
    for (const tool of uniqueById(tools)) {
        if (prevCategory !== null && prevCategory !== tool.category) {
            entries.push({ kind: "separator", id: `sep-${tool.toolbar}-${separatorIndex}` });
            separatorIndex += 1;
        }
        entries.push(toolToEntry(tool, ctx));
        prevCategory = tool.category;
    }
    return entries;
}

function toolToEntry(tool: ToolDefinition, ctx: ToolContext): FloatingDockEntry {
    if (tool.switchGroup) {
        return {
            kind: "switch",
            id: tool.id,
            label: resolveText(tool.label, ctx),
            value: tool.switchGroup.value(ctx),
            options: tool.switchGroup.options,
            onChange: (value) => tool.switchGroup?.onChange(ctx, value),
        };
    }
    return {
        kind: "command",
        id: tool.id,
        label: resolveText(tool.label, ctx),
        displayLabel: tool.displayLabel ? resolveText(tool.displayLabel, ctx) : undefined,
        icon: resolveIcon(tool.icon, ctx),
        active: tool.active?.(ctx),
        disabled: tool.disabled?.(ctx),
        danger: tool.danger,
        expands: tool.expands,
        onClick: (event) => tool.run(ctx, event),
    };
}

function resolveText(value: string | ((ctx: ToolContext) => string), ctx: ToolContext): string {
    return typeof value === "function" ? value(ctx) : value;
}

function resolveIcon(value: ToolDefinition["icon"], ctx: ToolContext) {
    return typeof value === "function" ? value(ctx) : value;
}
