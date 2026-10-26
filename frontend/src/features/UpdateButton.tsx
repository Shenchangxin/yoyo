// 本文件拥有标题栏上的更新按钮和确认应用。不拥有更新包怎么下载。调用方把它放进标题栏。

import { useEffect, useState } from "react";
import { Download } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../components/ui/button";
import { Tooltip } from "../components/ui/tooltip";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import * as api from "../lib/client";
import { unwrapEvent } from "../lib/stream";
import { ConfirmDialog } from "./ConfirmDialog";

type UpdateSnap = {
  phase?: string;
  percent?: number;
  newer?: boolean;
  downloaded?: boolean;
  staged?: boolean;
  error?: string;
  manifest?: { version?: string };
};

/** versionOf 从快照的 manifest.version 取版本，没有就是空字符串。s 读取 manifest，为空时不读。 */
function versionOf(s: UpdateSnap | null): string {
  return String(s?.manifest?.version || "").trim();
}

/** visible phase 为 ready 或 applying，或者已经 downloaded，才显示按钮。没有快照不显示。s 为假时返回假。 */
function visible(s: UpdateSnap | null): boolean {
  if (!s) return false;
  const phase = String(s.phase || "idle");
  if (phase === "ready" || phase === "applying") return true;
  return !!s.downloaded;
}

// UpdateButton 不可见时不渲染。百分比夹在 0 到 100。ready、downloading、checking 时每 1500 毫秒拉一次状态，否则每 4000 毫秒。
export function UpdateButton() {
  const copy = useCopy();
  const [ota, setOta] = useState<UpdateSnap | null>(null);
  const [applyOpen, setApplyOpen] = useState(false);
  const phase = String(ota?.phase || "idle");
  const percent = Math.max(0, Math.min(100, Number(ota?.percent || 0)));
  const ready = phase === "ready" || !!ota?.downloaded;
  const ver = versionOf(ota);
  const label = ready
    ? copy.titlebar.updateReady.replace("{version}", ver || copy.settings.updateReady)
    : phase === "downloading"
      ? copy.titlebar.updateDownloading.replace("{percent}", String(percent))
      : ver
        ? copy.titlebar.updateAvailable.replace("{version}", ver)
        : copy.titlebar.update;

  useEffect(() => {
    let stop = false;
    const pull = () => {
      void api.updateStatus().then((s) => {
        if (!stop) setOta(s || null);
      }).catch(() => {});
    };
    pull();
    const ms = ready || phase === "downloading" || phase === "checking" ? 1500 : 4000;
    const id = window.setInterval(pull, ms);
    return () => {
      stop = true;
      window.clearInterval(id);
    };
  }, [phase]);

  // VITE_E2E 时不订 yoyo:pulse。收到 kind 为 update 的脉冲就再拉一次状态。
  useEffect(() => {
    if (import.meta.env.VITE_E2E) return;
    let off: (() => void) | undefined;
    let alive = true;
    void import("@wailsio/runtime").then((mod: any) => {
      const Events = mod.Events;
      if (!alive || !Events?.On) return;
      off = Events.On("yoyo:pulse", (raw: any) => {
        const p = unwrapEvent(raw);
        if (String(p?.kind || "") === "update") {
          void api.updateStatus().then(setOta).catch(() => {});
        }
      });
    });
    return () => {
      alive = false;
      off?.();
    };
  }, []);

  if (!visible(ota)) return null;

  // applying 时只提示当前文案，不打开确认。其余阶段打开确认，确认后调用 applyUpdate。
  async function onClick() {
    if (phase === "applying") {
      toast.message(label);
      return;
    }
    setApplyOpen(true);
  }

  return (
    <>
      <Tooltip content={label}>
        <Button
          variant="ghost"
          size="icon"
          data-testid="update-download"
          aria-label={label}
          onClick={() => { void onClick(); }}
          className={cn("relative shrink-0", ready && "bg-lift text-foreground")}
        >
          <Download className="size-4" aria-hidden />
          {phase === "downloading" ? (
            <span className="absolute -right-0.5 -top-0.5 min-w-3.5 rounded-full bg-foreground px-1 text-[9px] text-background">{percent}%</span>
          ) : ready ? (
            <span className="absolute right-1 top-1 size-1.5 rounded-full bg-foreground" aria-hidden />
          ) : null}
        </Button>
      </Tooltip>
      <ConfirmDialog
        open={applyOpen}
        title={copy.settings.applyUpdate}
        body={copy.settings.applyUpdateConfirm}
        danger
        confirmLabel={copy.settings.applyUpdate}
        onCancel={() => setApplyOpen(false)}
        onConfirm={async () => {
          setApplyOpen(false);
          try {
            await api.applyUpdate();
            toast.success(copy.app.stagedUpdate);
          } catch (e) {
            toast.error(String(e));
          }
        }}
      />
    </>
  );
}
