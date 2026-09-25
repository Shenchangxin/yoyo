// @ts-nocheck
import { scopedLocalStorage } from "@yingce/lib/user-scope";

import type { ToolbarId, ToolbarPrefs } from "./tool-definition";

const STORAGE_VERSION = "v3";
const storageKey = (toolbar: ToolbarId) => `canvas-toolbar-prefs-${toolbar}-${STORAGE_VERSION}`;

export const CANVAS_MODE_TOOL_ID = "tool-canvas-mode";
export const LEGACY_CANVAS_MODE_TOOL_IDS = ["tool-move", "tool-box-select"] as const;

/**
 * 读取工具栏偏好。缺失或解析失败时返回 null（由调用方决定是否用默认值）
 */
export function readToolbarPrefs(toolbar: ToolbarId): ToolbarPrefs | null {
    try {
        const stored = scopedLocalStorage.getItem(storageKey(toolbar));
        if (!stored) return null;
        const parsed = JSON.parse(stored) as unknown;
        if (!isPrefsShape(parsed)) return null;
        const prefs: ToolbarPrefs = { order: Array.isArray(parsed.order) ? parsed.order.filter((id) => typeof id === "string") : [], hidden: Array.isArray(parsed.hidden) ? parsed.hidden.filter((id) => typeof id === "string") : [] };
        return migrateToolbarPrefs(toolbar, prefs);
    } catch {
        return null;
    }
}

export function persistToolbarPrefs(toolbar: ToolbarId, prefs: ToolbarPrefs) {
    try {
        scopedLocalStorage.setItem(storageKey(toolbar), JSON.stringify(prefs));
    } catch {
        // 浏览器禁用本地存储时保留当前会话内的选择
    }
}

export function clearToolbarPrefs(toolbar: ToolbarId) {
    try {
        scopedLocalStorage.removeItem(storageKey(toolbar));
    } catch {
        // 忽略
    }
}

function isPrefsShape(value: unknown): value is Partial<ToolbarPrefs> {
    return typeof value === "object" && value !== null;
}

function uniqueIds(ids: string[]): string[] {
    const seen = new Set<string>();
    const next: string[] = [];
    for (const id of ids) {
        if (!id || seen.has(id)) continue;
        seen.add(id);
        next.push(id);
    }
    return next;
}

/** 把旧的抓手/框选两个独立按钮偏好合并成一个开关项，并去掉重复 id */
export function migrateToolbarPrefs(toolbar: ToolbarId, prefs: ToolbarPrefs): ToolbarPrefs {
    let order = uniqueIds(prefs.order);
    let hidden = uniqueIds(prefs.hidden);
    if (toolbar !== "main") return { order, hidden };

    const legacyIds = new Set<string>(LEGACY_CANVAS_MODE_TOOL_IDS);
    const hasLegacy = order.some((id) => legacyIds.has(id)) || hidden.some((id) => legacyIds.has(id));
    if (!hasLegacy) return { order, hidden };

    order = order.filter((id) => id !== CANVAS_MODE_TOOL_ID && !legacyIds.has(id));
    const firstLegacyIndex = prefs.order.findIndex((id) => legacyIds.has(id));
    const insertAt = firstLegacyIndex >= 0 ? Math.min(firstLegacyIndex, order.length) : 0;
    order.splice(insertAt, 0, CANVAS_MODE_TOOL_ID);

    const hiddenLegacy = LEGACY_CANVAS_MODE_TOOL_IDS.filter((id) => hidden.includes(id));
    hidden = hidden.filter((id) => id !== CANVAS_MODE_TOOL_ID && !legacyIds.has(id));
    if (hiddenLegacy.length === LEGACY_CANVAS_MODE_TOOL_IDS.length) hidden.push(CANVAS_MODE_TOOL_ID);

    return { order: uniqueIds(order), hidden: uniqueIds(hidden) };
}
