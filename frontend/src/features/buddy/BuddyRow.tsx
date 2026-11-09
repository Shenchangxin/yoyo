/**
 * 拥有：总览里的一行 agent：字形、名字、厂商、版本、状态句、主按钮和更多菜单；安装中时行底一条进度细线。也拥有详情页复用的更多菜单。
 * 不拥有：按钮背后的动作（buddy-actions.ts），也不判断该出现哪些按钮（buddy-model.ts）。
 */

import { Loader2, MoreHorizontal } from "lucide-react";
import { Button } from "../../components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { StatusDot } from "../../components/ui/status-dot";
import type { BuddyView } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { BuddyMark } from "./BuddyMark";
import { buddyDo } from "./buddy-actions";
import { actionLabel, buddyMenu, buddyPrimary, buddyStatusText, buddyTone, fill, progressBar, type BuddyAction } from "./buddy-model";
import { useBuddy } from "./buddy-store";

/** primaryVariant 是主按钮的样式：在线时的测试用 ghost，取消用 lift，其余用描边。 */
function primaryVariant(action: BuddyAction): "ghost" | "lift" | "outline" {
  if (action === "test") return "ghost";
  if (action === "cancel") return "lift";
  return "outline";
}

/** BuddyPrimary 是主按钮。这一项有操作在等后端时转圈并禁用，取消除外。 */
export function BuddyPrimary({ v, size = "sm" }: { v: BuddyView; size?: "sm" | "default" }) {
  const c = useCopy().buddy;
  const pending = useBuddy((s) => s.pending[v.id] || "");
  const action = buddyPrimary(v);
  if (action === "none") return null;
  const waiting = !!pending && action !== "cancel";
  return (
    <Button size={size} variant={primaryVariant(action)} disabled={waiting} aria-busy={waiting || undefined} onClick={() => void buddyDo(v, action)}>
      {waiting ? <Loader2 className="animate-spin" aria-hidden /> : null}
      {actionLabel(action, v, c)}
    </Button>
  );
}

/** BuddyMoreMenu 是更多菜单。卸载和移除放在最后，前面隔一条线。没有可用项时不画。 */
export function BuddyMoreMenu({ v }: { v: BuddyView }) {
  const c = useCopy().buddy;
  const items = buddyMenu(v);
  if (!items.length) return null;
  const danger = items.filter((i) => i === "uninstall" || i === "remove");
  const rest = items.filter((i) => i !== "uninstall" && i !== "remove");
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button size="icon" variant="ghost" className="size-7" aria-label={fill(c.action.more, { name: v.name })}>
          <MoreHorizontal aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {rest.map((item) => (
          <DropdownMenuItem key={item} onSelect={() => void buddyDo(v, item)}>
            {actionLabel(item, v, c)}
          </DropdownMenuItem>
        ))}
        {danger.length && rest.length ? <DropdownMenuSeparator /> : null}
        {danger.map((item) => (
          <DropdownMenuItem key={item} className="text-danger focus:text-danger" onSelect={() => void buddyDo(v, item)}>
            {actionLabel(item, v, c)}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** BuddyRow 是总览里的一行。点名字或状态句进详情。 */
export function BuddyRow({ v, onOpen }: { v: BuddyView; onOpen: (id: string) => void }) {
  const c = useCopy().buddy;
  const tone = buddyTone(v);
  const status = buddyStatusText(v, c);
  const bar = progressBar(v);
  const quiet = v.state === "unsupported";
  return (
    <div className="relative flex min-h-14 items-center gap-3 border-b border-border/60 px-1 py-2 last:border-b-0" data-testid={`buddy-row-${v.id}`}>
      <BuddyMark v={v} className={cn("size-6", quiet ? "text-muted" : "text-foreground/80")} />
      <button type="button" className="min-w-0 flex-1 cursor-pointer rounded-md text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/40" onClick={() => onOpen(v.id)}>
        <span className="flex min-w-0 items-baseline gap-2">
          <span className={cn("truncate text-ui-body font-medium", quiet ? "text-muted" : "text-foreground")}>{v.name}</span>
          {v.vendor ? <span className="hidden truncate text-ui-caption text-muted sm:inline">{v.vendor}</span> : null}
          {v.version ? <span className="shrink-0 text-ui-caption tabular-nums text-muted">{fill(c.detail.version, { v: v.version })}</span> : null}
        </span>
        <span className="mt-0.5 flex min-w-0 items-center gap-1.5 text-ui-footnote text-muted" title={status}>
          {quiet ? null : <StatusDot tone={tone.tone} still={tone.still} />}
          <span className="truncate" data-testid={`buddy-status-${v.id}`}>{status}</span>
        </span>
      </button>
      <div className="flex shrink-0 items-center gap-1">
        <BuddyPrimary v={v} />
        <BuddyMoreMenu v={v} />
      </div>
      {v.state === "installing" ? <ProgressHairline className="absolute inset-x-1 bottom-0" value={bar.value} indeterminate={bar.indeterminate} /> : null}
    </div>
  );
}
