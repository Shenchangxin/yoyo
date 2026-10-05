// @ts-nocheck
import { Button } from "antd";
import { Home, RefreshCw } from "lucide-react";
import { useNavigate, useRouteError } from "react-router";

import { WorkspaceSignalIcon } from "@yingce/components/ui/aceternity/workspace-signal-icon";
import { reloadAfterChunkFailure } from "@yingce/lib/chunk-recovery";
import { requestHostNavigation } from "@/features/video/canvas-host/design-system";

function routeErrorMessage(error: unknown) {
    if (error instanceof Error && error.message) return error.message;
    if (typeof error === "string" && error.trim()) return error;
    if (error && typeof error === "object") {
        const record = error as { message?: unknown; statusText?: unknown; data?: unknown; error?: { message?: unknown } };
        const nested = [record.message, record.statusText, record.error?.message, record.data]
            .map((value) => (typeof value === "string" ? value.trim() : ""))
            .find(Boolean);
        if (nested) return nested;
        try {
            const dumped = JSON.stringify(error);
            if (dumped && dumped !== "{}") return dumped;
        } catch {
            /* ignore */
        }
    }
    return "页面暂时无法显示";
}

export default function RouteErrorPage() {
    const error = useRouteError();
    const navigate = useNavigate();
    const message = routeErrorMessage(error);
    if (typeof console !== "undefined") console.error("yingce route error", error);

    return (
        <main className="app-workspace-page grid h-dvh place-items-center px-6 text-foreground">
            <section className="w-full max-w-md text-center">
                <WorkspaceSignalIcon variant="error" size="lg" className="mx-auto" />
                <p className="text-xs font-medium text-muted-foreground">页面运行异常</p>
                <h1 className="mt-3 text-2xl font-semibold">当前页面没有正常加载</h1>
                <p className="mt-3 break-words text-sm leading-6 text-muted-foreground">{message}</p>
                <div className="mt-6 flex justify-center gap-3">
                    <Button icon={<RefreshCw className="size-4" />} onClick={reloadAfterChunkFailure}>重新加载</Button>
                    <Button type="primary" icon={<Home className="size-4" />} onClick={() => { if (!requestHostNavigation("/")) navigate("/"); }}>返回主页</Button>
                </div>
            </section>
        </main>
    );
}
