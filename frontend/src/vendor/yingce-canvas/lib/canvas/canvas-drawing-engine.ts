// @ts-nocheck
import type { CanvasNodeData } from "@yingce/types/canvas";

export type CanvasDrawingEngine = "tldraw" | "excalidraw";

export type CanvasDrawingEngineSetting = {
    defaultEngine: CanvasDrawingEngine;
    tldrawLicenseKey?: string;
    configured?: boolean;
    updatedBy?: string;
    createdAt?: string;
    updatedAt?: string;
};

export const DEFAULT_DRAWING_ENGINE: CanvasDrawingEngine = "excalidraw";

export function drawingEngineForNode(node?: Pick<CanvasNodeData, "metadata"> | null): CanvasDrawingEngine {
    // 旧绘图节点没有引擎标记，其快照只能由 tldraw 读取。
    return node?.metadata?.drawingEngine === "excalidraw" ? "excalidraw" : "tldraw";
}

export function resolveTldrawLicenseKey(configuredLicenseKey?: string) {
    return configuredLicenseKey?.trim() || import.meta.env.VITE_TLDRAW_LICENSE_KEY?.trim() || undefined;
}

export function isDrawingEngineAvailable(engine: CanvasDrawingEngine, configuredLicenseKey?: string) {
    return engine === "excalidraw" || Boolean(resolveTldrawLicenseKey(configuredLicenseKey));
}

function isLoopbackHost(hostname: string) {
    const host = hostname.toLowerCase().replace(/^\[|\]$/g, "");
    return host === "localhost" || host === "::1" || /^127(?:\.\d{1,3}){3}$/.test(host);
}

/** 与 tldraw LicenseManager 的开发环境判断一致。这些环境没有 Key 也会保留编辑器。 */
export function tldrawRunsWithoutLicense() {
    if (!import.meta.env.PROD) return true;
    if (typeof window === "undefined") return false;
    const protocol = window.location.protocol;
    return !["https:", "vscode-webview:"].includes(protocol) || isLoopbackHost(window.location.hostname);
}

export function canUseDrawingEngine(engine: CanvasDrawingEngine, configuredLicenseKey?: string) {
    return isDrawingEngineAvailable(engine, configuredLicenseKey) || (engine === "tldraw" && tldrawRunsWithoutLicense());
}

export function drawingEngineForCreate(preferred: CanvasDrawingEngine, configuredLicenseKey?: string): CanvasDrawingEngine {
    return canUseDrawingEngine(preferred, configuredLicenseKey) ? preferred : "excalidraw";
}

export function drawingEngineLabel(engine: CanvasDrawingEngine) {
    return engine === "excalidraw" ? "Excalidraw" : "tldraw";
}