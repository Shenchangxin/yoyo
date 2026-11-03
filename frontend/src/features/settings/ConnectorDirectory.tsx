/**
 * 拥有：连接应用的分栏和搜索。点本地项不新建账户。
 * 不拥有：令牌和 MCP 进程。那些仍在上面的表单里。
 */

import { useMemo, useState } from "react";
import { useCopy, getLocale } from "../../lib/i18n";
import { str } from "../../lib/normalize";
import { cn } from "../../lib/utils";
import { connectorBlurb, connectorName, resolvedAccess } from "./connector-directory";

export type CatalogRow = {
  provider: string;
  label: string;
  platform: string;
  access: string;
  kind: string;
};

const tabs = ["windows", "mac", "linux", "ios", "android"] as const;
type Tab = (typeof tabs)[number];

/** hostPlatform 从浏览器判断这台桌面。认不出时当作 Windows。 */
export function hostPlatform(): Tab {
  try {
    const probe = `${navigator.platform || ""} ${navigator.userAgent || ""}`;
    if (/mac/i.test(probe)) return "mac";
    if (/linux/i.test(probe) && !/android/i.test(probe)) return "linux";
  } catch {
    /* 读不到平台时用 Windows。 */
  }
  return "windows";
}

/** ConnectorDirectory 按系统列出连接。已连接的账户只影响按钮上的字。 */
export function ConnectorDirectory(props: {
  catalog: CatalogRow[];
  connected: string[];
  onPick: (row: CatalogRow) => void;
}) {
  const copy = useCopy();
  const locale = getLocale();
  const current = hostPlatform();
  const [tab, setTab] = useState<Tab>(current);
  const [query, setQuery] = useState("");
  const names: Record<Tab, string> = {
    windows: copy.settings.appsWindows,
    mac: copy.settings.appsMac,
    linux: copy.settings.appsLinux,
    ios: copy.settings.appsIos,
    android: copy.settings.appsAndroid,
  };
  const needle = query.trim().toLowerCase();
  const rows = useMemo(() => props.catalog.filter((row) => {
    const platform = row.platform || "";
    const onTab = tab === "ios" || tab === "android" ? platform === tab : platform === tab || platform === "";
    if (!onTab) return false;
    if (!needle) return true;
    const blurb = connectorBlurb(row.provider, locale);
    const name = connectorName(row.provider, row.label, locale);
    return `${name} ${row.label} ${row.provider} ${blurb}`.toLowerCase().includes(needle);
  }), [props.catalog, tab, needle, locale]);
  return (
    <div className="border-t border-border px-3 py-3">
      <div className="flex items-center justify-end gap-3">
        <input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={copy.settings.appsSearch}
          aria-label={copy.settings.appsSearch}
          className="h-8 w-40 rounded-full border border-border bg-background px-3 text-[12px] outline-none"
        />
      </div>
      <div role="tablist" aria-label={copy.settings.appsTitle} className="mt-3 flex flex-wrap gap-1">
        {tabs.map((item) => (
          <button
            key={item}
            type="button"
            role="tab"
            aria-selected={tab === item}
            className={cn("h-7 rounded-full px-3 text-[12px]", tab === item ? "bg-foreground text-background" : "text-muted hover:text-foreground")}
            onClick={() => setTab(item)}
          >
            {item === current ? copy.settings.appsCurrent.replace("{name}", names[item]) : names[item]}
          </button>
        ))}
      </div>
      <div className="mt-3 grid grid-cols-2 gap-2">
        {rows.map((row) => {
          const access = resolvedAccess(row.provider, row.access);
          const blurb = connectorBlurb(row.provider, locale) || fallback({ ...row, access }, copy.settings, tab === current);
          const here = access === "local" && row.platform === current;
          const on = props.connected.includes(row.provider);
          return (
            <button
              key={row.provider}
              type="button"
              className="flex min-w-0 items-start justify-between gap-3 rounded-[16px] border border-border bg-background px-3 py-3 text-left hover:border-foreground/20"
              onClick={() => props.onPick(row)}
            >
              <span className="min-w-0">
                <span className="block truncate text-[13px] font-medium">{connectorName(row.provider, row.label, locale)}</span>
                <span className="mt-1 block text-[12px] leading-5 text-muted">{blurb}</span>
              </span>
              <span className="max-w-16 shrink-0 text-right text-[11px] leading-4 text-muted">{here ? copy.settings.appsThisMachine : on ? copy.rail.connected : "+"}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function fallback(row: CatalogRow, settings: ReturnType<typeof useCopy>["settings"], here: boolean): string {
  if (row.access === "local") return here ? settings.appsLocalHint : settings.appsOtherDevice;
  if (row.access === "source") return settings.appsWorkSource;
  if (row.access === "oauth") return settings.appsAccount;
  if (row.platform === "ios" || row.platform === "android") return settings.appsDeviceHint;
  return settings.appsMcpHint;
}

/** catalogRows 把目录接口收成稳定字段。 */
export function catalogRows(raw: unknown[]): CatalogRow[] {
  return raw.map((row) => ({
    provider: str((row as { provider?: string; Provider?: string }).provider || (row as { Provider?: string }).Provider),
    label: str((row as { label?: string; Label?: string }).label || (row as { Label?: string }).Label),
    platform: str((row as { platform?: string; Platform?: string }).platform || (row as { Platform?: string }).Platform),
    access: str((row as { access?: string; Access?: string }).access || (row as { Access?: string }).Access),
    kind: str((row as { kind?: string; Kind?: string }).kind || (row as { Kind?: string }).Kind),
  })).filter((row) => row.provider && row.label);
}
