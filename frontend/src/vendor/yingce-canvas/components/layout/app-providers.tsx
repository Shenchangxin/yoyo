// @ts-nocheck
import type { ReactNode } from "react";
import { lazy, Suspense, useLayoutEffect } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { App, ConfigProvider } from "antd";
import zhCN from "antd/locale/zh_CN";

import { AuthSessionHydrator } from "@yingce/components/auth/auth-session-hydrator";
import { FullScreenLoader } from "@yingce/components/ui/aceternity/full-screen-loader";
import { getAntThemeConfig } from "@yingce/lib/app-theme";
import { yoyoHostAntTheme } from "@/features/video/canvas-host/yoyo-ant-theme";
import { appQueryClient } from "@yingce/lib/query-client";
import { isIsolatedDirectorRepro } from "@yingce/lib/dev-repro";
import { useActiveTheme } from "@yingce/stores/canvas/use-canvas-theme-store";
import { useAppearanceStore } from "@yingce/stores/use-appearance-store";
import { useUserStore } from "@yingce/stores/use-user-store";

const ClientRootInit = lazy(() => import("@yingce/components/layout/client-root-init").then((module) => ({ default: module.ClientRootInit })));

function ClientRootBoundary({ children }: { children: ReactNode }) {
    const authenticated = useUserStore((state) => Boolean(state.user));
    if (!authenticated) return children;
    return <Suspense fallback={<FullScreenLoader label="正在准备创作环境" detail="连接本地能力与模型配置" />}><ClientRootInit>{children}</ClientRootInit></Suspense>;
}

export function AppProviders({ children }: { children: ReactNode }) {
    const theme = useActiveTheme();
    const dark = theme === "dark";
    const appearance = useAppearanceStore((state) => state.appearance);

    useLayoutEffect(() => {
        const island = document.querySelector(".yingce-island");
        if (island) {
            island.classList.toggle("dark", dark);
            island.classList.toggle("light", !dark);
            (island as HTMLElement).style.colorScheme = theme;
        }
    }, [appearance, dark, theme]);

    // DEV 复现台必须是同源本地确定性场景：AuthSessionHydrator 会打 /api/auth/session，
    // ClientRootInit 会打 /api/model-catalog，没有后端时产生真实 502，与导演台无关却会污染判据。
    // 与启动入口共用精确路径边界；生产构建中 import.meta.env.DEV 为 false，始终不启用隔离。
    const isolateDevRepro = typeof window !== "undefined" && isIsolatedDirectorRepro(import.meta.env.DEV, window.location.pathname);

    return (
        <ConfigProvider locale={zhCN} prefixCls="yc" theme={typeof window !== "undefined" && (window as Window & { __YOYO_CANVAS_HOST__?: boolean }).__YOYO_CANVAS_HOST__ ? yoyoHostAntTheme(dark) : getAntThemeConfig(dark, appearance.activeSkin)} wave={{ disabled: true }}>
            <App message={{ duration: 3, maxCount: 3 }} notification={{ duration: 4.5, maxCount: 3, placement: "topRight" }}>
                <QueryClientProvider client={appQueryClient}>
                    {isolateDevRepro ? (
                        children
                    ) : (
                        <AuthSessionHydrator>
                            <ClientRootBoundary>{children}</ClientRootBoundary>
                        </AuthSessionHydrator>
                    )}
                </QueryClientProvider>
            </App>
        </ConfigProvider>
    );
}
