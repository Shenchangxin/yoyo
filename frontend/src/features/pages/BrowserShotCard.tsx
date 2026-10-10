import { useUI } from "../../lib/store";
import { useCopy } from "../../lib/i18n";
import type { Item } from "../../lib/protocol";
import { toolArgs, toolName } from "../../lib/tool-summary";

export function isBrowserShotItem(item?: Item): boolean {
  if (!item) return false;
  const n = toolName(item);
  return n === "browser_screenshot" || n === "browser_snapshot";
}

export function BrowserShotCard(props: { call?: Item; result?: Item }) {
  const copy = useCopy();
  const item = props.result || props.call;
  if (!item) return null;
  const args = { ...toolArgs(props.call || item), ...toolArgs(props.result || item) };
  const path = String(args.path || args.file || item.payload?.path || "");
  const src = String(item.payload?.preview || item.payload?.url || "");
  return (
    <button
      type="button"
      className="block overflow-hidden rounded-xl border border-border/70 bg-lift/30 text-left"
      data-testid="browser-shot-card"
      onClick={() => {
        useUI.getState().setInspTab("browser");
        useUI.getState().setInspector(true);
      }}
    >
      {src ? (
        <img src={src} alt="" className="max-h-40 w-full object-cover" />
      ) : (
        <div className="px-3 py-6 text-center text-[12px] text-muted">{copy.pages.browserShot}</div>
      )}
      {path ? <div className="truncate px-2 py-1 text-[11px] text-muted">{path}</div> : null}
    </button>
  );
}
