// @ts-nocheck
import type { ReactNode } from "react";
import { useEffect } from "react";

import { getAuthSession, invalidateAuthSessionCache, type AuthSessionPayload } from "@yingce/services/api/auth";
import { FullScreenLoader } from "@yingce/components/ui/aceternity/full-screen-loader";
import { preloadWorkspaceRoute } from "@yingce/lib/workspace-route-modules";
import { useUserStore } from "@yingce/stores/use-user-store";
import { recordDiagnosticEvent } from "@yingce/services/diagnostics/client-diagnostics";

export function AuthSessionHydrator({ children }: { children: ReactNode }) {
    const hydrated = useUserStore((state) => state.hydrated);

    useEffect(() => {
        let cancelled = false;
        const startedAt = performance.now();
        const hosted = typeof window !== "undefined" && Boolean((window as Window & { __YOYO_CANVAS_HOST__?: boolean }).__YOYO_CANVAS_HOST__);
        recordDiagnosticEvent({ category: "navigation", level: "info", code: "startup.auth_session_started", message: "开始恢复认证会话" });

        const load = (attempt = 0) => {
            if (hosted) invalidateAuthSessionCache();
            getAuthSession()
                .then(async (payload) => {
                    if (cancelled) return;
                    recordDiagnosticEvent({
                        category: "navigation",
                        level: "info",
                        code: "startup.auth_session_ready",
                        message: payload.user ? "认证会话已恢复" : "匿名会话已确认",
                        durationMs: performance.now() - startedAt,
                    });
                    if (!payload.user) {
                        if (hosted && attempt < 8) {
                            window.setTimeout(() => { if (!cancelled) load(attempt + 1); }, 400 * (attempt + 1));
                            return;
                        }
                        applyAnonymousSession(payload);
                        recordDiagnosticEvent({
                            category: "navigation",
                            level: "info",
                            code: "startup.anonymous_ready",
                            message: "匿名页面已解除启动阻塞",
                            durationMs: performance.now() - startedAt,
                        });
                        return;
                    }
                    const { applyUserSession } = await import("@yingce/lib/user-session");
                    if (cancelled) return;
                    await applyUserSession(payload);
                    recordDiagnosticEvent({
                        category: "navigation",
                        level: "info",
                        code: "startup.workspace_ready",
                        message: "登录工作区已解除启动阻塞",
                        durationMs: performance.now() - startedAt,
                    });
                    preloadWorkspaceRoute(window.location.pathname);
                })
                .catch(() => {
                    if (cancelled) return;
                    if (hosted && attempt < 8) {
                        window.setTimeout(() => { if (!cancelled) load(attempt + 1); }, 400 * (attempt + 1));
                        return;
                    }
                    applyAnonymousSession({ user: null });
                    recordDiagnosticEvent({
                        category: "navigation",
                        level: "warning",
                        code: "startup.auth_session_failed",
                        message: "认证会话恢复失败，已降级为匿名页面",
                        durationMs: performance.now() - startedAt,
                    });
                });
        };
        load();
        return () => {
            cancelled = true;
        };
    }, []);

    return hydrated ? children : <FullScreenLoader />;
}

function applyAnonymousSession(payload: AuthSessionPayload) {
    const store = useUserStore.getState();
    store.clearSession();
    store.setRuntimeLimits(payload.runtimeLimits);
    store.setDrawingEngine(payload.drawingEngine);
    store.setFeatures(payload.features);
    store.setHydrated(true);
}
