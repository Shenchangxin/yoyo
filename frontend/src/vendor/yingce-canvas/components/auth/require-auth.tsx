// @ts-nocheck
import type { ReactNode } from "react";
import { Navigate, useLocation } from "react-router";

import { FullScreenLoader } from "@yingce/components/ui/aceternity/full-screen-loader";
import { useUserStore } from "@yingce/stores/use-user-store";

export function RequireAuth({ children }: { children: ReactNode }) {
    const location = useLocation();
    const hydrated = useUserStore((state) => state.hydrated);
    const user = useUserStore((state) => state.user);
    const hosted = typeof window !== "undefined" && Boolean((window as Window & { __YOYO_CANVAS_HOST__?: boolean }).__YOYO_CANVAS_HOST__);

    if (!hydrated) return <FullScreenLoader />;
    if (!user) {
        // Hosted island has no /login route. Keep waiting for the local session
        // instead of navigating into a Not Found error page.
        if (hosted) return <FullScreenLoader label="正在连接本地工作区" detail="恢复创作账号与模型目录" />;
        return <Navigate to={`/login?next=${encodeURIComponent(location.pathname + location.search)}`} replace />;
    }
    return children;
}
