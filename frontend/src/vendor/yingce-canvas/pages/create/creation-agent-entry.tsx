// @ts-nocheck
import { useRef, useState } from "react";
import { App, Button } from "antd";
import { ArrowUpRight, FolderOpen, Clapperboard } from "lucide-react";
import { useNavigate } from "react-router";
import { openHostedVideoHref } from "@/features/video/canvas-host/host-nav";
import { useCopy } from "@/lib/i18n";
import { useCanvasStore } from "@yingce/stores/canvas/use-canvas-store";
import { useUserStore } from "@yingce/stores/use-user-store";
import { createCanvasProjectWithRemoteSync, hasRemoteUserDataSyncSession, saveRemoteUserDataNow } from "@yingce/services/user-data-sync";

/** Agent 依赖真实画布 ID，先完成本机保存，再进入同一个画布 Agent。 */
export function CreationAgentEntry() {
    const navigate = useNavigate();
    const { message } = App.useApp();
    const create = useCopy().video.create;
    const hydrated = useCanvasStore((state) => state.hydrated);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState("");
    const lock = useRef(false);
    const created = useRef<{ id: string; userId: string } | null>(null);
    const start = async () => {
        if (lock.current) return;
        lock.current = true;
        setBusy(true);
        setError("");
        const userId = useUserStore.getState().user?.id;
        try {
            if (!userId || !hasRemoteUserDataSyncSession()) throw new Error(create.agentNeedSession);
            if (!hydrated) throw new Error(create.agentRestoring);
            if (created.current?.userId !== userId) created.current = null;
            if (!created.current) {
                const result = await createCanvasProjectWithRemoteSync(create.agentTitle);
                if (!result.id) throw new Error(create.agentCreateFailed);
                created.current = { id: result.id, userId };
                if (result.syncError) throw new Error(create.agentSyncFailed);
            } else {
                await saveRemoteUserDataNow();
            }
            if (useUserStore.getState().user?.id !== userId) return;
            const href = `/canvas/${encodeURIComponent(created.current.id)}?agent=1`;
            if (!openHostedVideoHref(href)) navigate(href);
        } catch (cause) {
            if (useUserStore.getState().user?.id === userId) {
                const detail = cause instanceof Error ? cause.message : create.agentStartFailed;
                setError(detail);
                message.error(detail);
            }
        } finally { lock.current = false; setBusy(false); }
    };
    return <section className="creation-agent-entry" aria-label={create.modeAgent} data-testid="creation-agent-entry">
        <div className="creation-agent-copy">
            <span className="creation-agent-eyebrow">{create.modeAgent}</span>
            <h2>{create.agentTitle}</h2>
            <p>{create.agentBody}</p>
        </div>
        <div className="creation-agent-actions">
            <Button type="primary" icon={<Clapperboard />} loading={busy} disabled={!hydrated} onClick={() => void start()}>
                {created.current ? create.agentRetry : create.agentStart}<ArrowUpRight />
            </Button>
            <Button icon={<FolderOpen />} disabled={busy} onClick={() => { const href = "/canvas?agent=1"; if (!openHostedVideoHref(href)) navigate(href); }}>{create.agentContinue}</Button>
        </div>
        {error ? <p className="creation-agent-error" role="alert">{error}</p> : null}
    </section>;
}
