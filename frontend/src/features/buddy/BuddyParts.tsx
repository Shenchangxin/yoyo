/**
 * 拥有：Buddy 页里几处都要用的小件：头像位、按状态给出的主按钮、更多菜单、要人看清的命令块、
 * 就地的确认卡（装到系统、卸载、打不开终端时的登录命令），以及密钥、模型、审批三个输入。样子用今天栏那一套（day-ui）。
 * 不拥有：按钮背后的动作（buddy-actions.ts），也不判断该出现哪些按钮（buddy-model.ts）。
 * 密钥输入只进不出：已保存的只显示「已保存」，保存后由表单清空。装到系统必须先把完整命令给人看，人点了才运行。
 */

import { useEffect, useRef, type ReactNode } from "react";
import { Loader2, MoreHorizontal } from "lucide-react";
import { toast } from "sonner";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import type { BuddySecret, BuddyView } from "../../lib/client";
import { writeClipboard } from "../../lib/clipboard";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";
import { DAY_INPUT, DayCard, DayInput, DayPill, DayRound, DaySwitch, DayText } from "../superagent/day-ui";
import { BuddyMark } from "./BuddyMark";
import { buddyDo, buddySystemInstall, buddyUninstall } from "./buddy-actions";
import { DEFAULT_MODEL, type KeyDraft } from "./buddy-fields";
import { actionLabel, buddyMenu, buddyPrimary, fill, type BuddyAction, type BuddyCopy } from "./buddy-model";
import { useBuddy } from "./buddy-store";

/** AgentAvatar 是一枚浅底微描边的圆，里面是 agent 的字形。large 用在详情页的头部。 */
export function AgentAvatar({ v, large, quiet }: { v: Pick<BuddyView, "name" | "icon">; large?: boolean; quiet?: boolean }) {
  return (
    <span className={cn("grid shrink-0 place-items-center rounded-full border border-border bg-lift/75", large ? "size-11" : "size-9", quiet ? "text-muted" : "text-foreground/90")} aria-hidden>
      <BuddyMark v={v} className={cn("border-0", large ? "size-5 text-ui-title-s" : "size-4 text-ui-footnote")} />
    </span>
  );
}

/** tone 是主按钮的轻重：要人做决定的（安装、连接、登录、配置）用深底；重试、取消、测试是次要的。 */
function tone(action: BuddyAction): "solid" | "soft" {
  return action === "install" || action === "installSystem" || action === "login" || action === "configure" ? "solid" : "soft";
}

/** BuddyPrimary 是按状态给出的主按钮。这一项有操作在等后端时转圈并禁用，取消除外。hideTest 为真时不放「测试」。 */
export function BuddyPrimary({ v, small, hideTest }: { v: BuddyView; small?: boolean; hideTest?: boolean }) {
  const c = useCopy().buddy;
  const pending = useBuddy((s) => s.pending[v.id] || "");
  const action = buddyPrimary(v);
  if (action === "none" || (hideTest && action === "test")) return null;
  const waiting = !!pending && action !== "cancel";
  return (
    <DayPill small={small} tone={tone(action)} busy={waiting} onClick={() => void buddyDo(v, action)}>
      {waiting ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : null}
      {actionLabel(action, v, c)}
    </DayPill>
  );
}

/** BuddyMore 是更多菜单。卸载和移除放在最后，前面隔一条线。没有可用项时不画。 */
export function BuddyMore({ v }: { v: BuddyView }) {
  const c = useCopy().buddy;
  const items = buddyMenu(v);
  if (!items.length) return null;
  const danger = items.filter((i) => i === "uninstall" || i === "remove");
  const rest = items.filter((i) => i !== "uninstall" && i !== "remove");
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <DayRound quiet label={fill(c.action.more, { name: v.name })} className="size-7">
          <MoreHorizontal className="size-4" aria-hidden />
        </DayRound>
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

/** CommandBlock 是一段要人看清楚的命令，整段显示、不截断，下面可以复制。 */
export function CommandBlock({ command, c }: { command: string; c: BuddyCopy }) {
  return (
    <div className="day-well mt-3 px-3.5 py-3">
      <code className="block whitespace-pre-wrap break-all font-mono text-ui-footnote leading-5 text-foreground" data-testid="buddy-command">
        {command}
      </code>
      <div className="mt-1.5 flex justify-end">
        <DayText
          onClick={() => {
            void writeClipboard(command).then((ok) => ok && toast(c.detail.copied));
          }}
        >
          {c.confirm.copyCommand}
        </DayText>
      </div>
    </div>
  );
}

/** ConfirmCard 是就地的一张确认卡：标题、说明、可选的命令，下面取消和确认。出现时滚到看得见的地方并把焦点给它。 */
function ConfirmCard(props: { title: string; body: ReactNode; confirm: string; danger?: boolean; onCancel: () => void; onConfirm: () => void | Promise<void>; cancel: string }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    ref.current?.scrollIntoView({ block: "nearest" });
    ref.current?.focus();
  }, []);
  return (
    <div
      ref={ref}
      role="alertdialog"
      aria-label={props.title}
      tabIndex={-1}
      className="day-card mt-4 px-5 py-5 outline-none ring-1 ring-border"
      data-testid="buddy-confirm"
      onKeyDown={(e) => {
        if (e.key === "Escape") props.onCancel();
      }}
    >
      <h2 className="text-ui-title-s font-semibold">{props.title}</h2>
      <div className="mt-1.5 text-ui-body leading-6 text-muted">{props.body}</div>
      <div className="mt-4 flex justify-end gap-2">
        <DayPill tone="soft" onClick={props.onCancel}>{props.cancel}</DayPill>
        <DayPill tone={props.danger ? "danger" : "solid"} onClick={() => void props.onConfirm()}>{props.confirm}</DayPill>
      </div>
    </div>
  );
}

/**
 * BuddyConfirms 画出仓库里开着的那张确认卡：装到系统（先显示完整命令）、卸载或移除、打不开终端时的登录命令。
 * only 给了编号时只画属于这一项的，详情页用；没给时都画，列表页用。
 */
export function BuddyConfirms({ only }: { only?: string }) {
  const c = useCopy().buddy;
  const dialog = useBuddy((s) => s.dialog);
  const list = useBuddy((s) => s.list);
  const openDialog = useBuddy((s) => s.openDialog);
  if (!dialog || !("id" in dialog) || (only && dialog.id !== only)) return null;
  const v = list.find((x) => x.id === dialog.id);
  const close = () => openDialog(null);
  if (!v) return null;
  if (dialog.kind === "system") {
    return (
      <ConfirmCard
        title={fill(c.confirm.systemTitle, { name: v.name })}
        body={
          <>
            <p>{fill(c.confirm.systemBody, { name: v.name })}</p>
            <CommandBlock command={v.systemInstall?.command || ""} c={c} />
          </>
        }
        confirm={c.confirm.systemRun}
        cancel={c.action.cancel}
        onCancel={close}
        onConfirm={async () => {
          close();
          await buddySystemInstall(v);
        }}
      />
    );
  }
  if (dialog.kind === "uninstall") {
    return (
      <ConfirmCard
        danger
        title={fill(v.custom ? c.confirm.removeTitle : c.confirm.uninstallTitle, { name: v.name })}
        body={fill(v.custom ? c.confirm.removeBody : v.install.kind === "system" ? c.confirm.uninstallSystemBody : c.confirm.uninstallBody, { name: v.name })}
        confirm={v.custom ? c.action.remove : c.action.uninstall}
        cancel={c.action.cancel}
        onCancel={close}
        onConfirm={async () => {
          await buddyUninstall(v);
          close();
          if (v.custom && useUI.getState().buddyFocus === v.id) useUI.getState().setBuddyFocus("");
        }}
      />
    );
  }
  if (dialog.kind === "loginCopy") {
    const command = dialog.command;
    return (
      <ConfirmCard
        title={c.detail.login}
        body={
          <>
            <p>{c.detail.loginCopy}</p>
            <CommandBlock command={command} c={c} />
          </>
        }
        confirm={c.confirm.copyCommand}
        cancel={c.action.cancel}
        onCancel={close}
        onConfirm={async () => {
          if (await writeClipboard(command)) toast(c.detail.copied);
          close();
        }}
      />
    );
  }
  return null;
}

/** KeyField 是一把密钥：名字、已保存的标记和删除、密码输入框；yoyo 有同类钥匙时多一个复用开关，并写明会把钥匙交给它。 */
export function KeyField(props: { s: BuddySecret; draft: KeyDraft; onDraft: (next: KeyDraft) => void; c: BuddyCopy }) {
  const { s, draft, c } = props;
  const id = `buddy-key-${s.name}`;
  const saved = s.hasKey && !draft.cleared[s.name];
  const reuse = !!draft.reuse[s.name];
  const label = s.label || s.name;
  return (
    <div className="space-y-1.5">
      <div className="flex min-w-0 items-center gap-2 px-1">
        <label htmlFor={id} className="min-w-0 flex-1 truncate text-ui-footnote text-muted">
          <span className="text-foreground">{label}</span>
          {s.label ? <span className="ml-1.5 font-mono text-ui-caption">{s.name}</span> : null}
          {s.optional ? <span className="ml-1.5">{c.detail.optional}</span> : null}
        </label>
        {saved ? <span className="shrink-0 text-ui-caption text-status-done">{c.detail.keySaved}</span> : null}
        {saved ? (
          <DayText className="h-6 px-2" danger onClick={() => props.onDraft({ ...draft, cleared: { ...draft.cleared, [s.name]: true } })}>
            {c.detail.keyClear}
          </DayText>
        ) : null}
      </div>
      <DayInput
        id={id}
        type="password"
        name={`buddy-${s.name}`}
        autoComplete="new-password"
        data-testid={id}
        disabled={reuse}
        value={draft.values[s.name] ?? ""}
        placeholder={saved ? c.detail.keyReplace : c.detail.keyPlaceholder}
        onChange={(e) => props.onDraft({ ...draft, values: { ...draft.values, [s.name]: e.target.value } })}
      />
      {s.reusable ? (
        <div className="flex items-start justify-between gap-3 px-1 pt-1">
          <p className="min-w-0 text-ui-footnote leading-5 text-muted">
            <span className="block text-foreground">{c.detail.reuse}</span>
            {fill(c.detail.reuseNote, { name: label })}
          </p>
          <DaySwitch on={reuse} label={c.detail.reuse} onChange={(on) => props.onDraft({ ...draft, reuse: { ...draft.reuse, [s.name]: on } })} />
        </div>
      ) : null}
    </div>
  );
}

/** ModelField 是模型：探测到列表、而且当前值就在列表里时用下拉；否则用输入框，好填 yoyo 的模型或别的名字。 */
export function ModelField(props: { v: BuddyView; value: string; onChange: (model: string) => void; c: BuddyCopy; id: string }) {
  const models = props.v.config.models ?? [];
  const known = !props.value || models.some((m) => m.id === props.value);
  if (models.length && known) {
    return (
      <Select value={props.value || DEFAULT_MODEL} onValueChange={(m) => props.onChange(m === DEFAULT_MODEL ? "" : m)}>
        <SelectTrigger id={props.id} className={DAY_INPUT} aria-label={props.c.detail.model}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={DEFAULT_MODEL}>{props.c.detail.modelDefault}</SelectItem>
          {models.map((m) => (
            <SelectItem key={m.id} value={m.id}>
              {m.name || m.id}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );
  }
  const listId = `${props.id}-models`;
  return (
    <>
      <DayInput
        id={props.id}
        list={models.length ? listId : undefined}
        value={props.value}
        placeholder={props.c.detail.modelPlaceholder}
        spellCheck={false}
        onChange={(e) => props.onChange(e.target.value)}
      />
      {models.length ? (
        <datalist id={listId}>
          {models.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name || m.id}
            </option>
          ))}
        </datalist>
      ) : null}
    </>
  );
}

/** ApprovalField 是审批下拉：yoyo 决定（默认）或每次问我。没有会降低监督的选项。 */
export function ApprovalField(props: { value: "yoyo" | "ask"; onChange: (v: "yoyo" | "ask") => void; c: BuddyCopy; id: string }) {
  return (
    <Select value={props.value} onValueChange={(v) => props.onChange(v === "ask" ? "ask" : "yoyo")}>
      <SelectTrigger id={props.id} className={DAY_INPUT} aria-label={props.c.detail.approval}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="yoyo">{props.c.detail.approvalYoyo}</SelectItem>
        <SelectItem value="ask">{props.c.detail.approvalAsk}</SelectItem>
      </SelectContent>
    </Select>
  );
}

/** BuddyNote 是卡片里的一句提示，比如保存成功或出错。 */
export function BuddyNote({ text, ok }: { text: string; ok?: boolean }) {
  if (!text) return null;
  return (
    <span role="status" className={cn("text-ui-footnote", ok ? "text-status-done" : "text-danger")}>
      {text}
    </span>
  );
}

/** BuddyCard 是 Buddy 页里带小标题的一张卡。 */
export function BuddyCard(props: { title?: string; children: ReactNode; testId?: string; className?: string }) {
  return (
    <DayCard className={cn("mt-4", props.className)} testId={props.testId}>
      {props.title ? <h2 className="mb-3 text-ui-title-s font-semibold">{props.title}</h2> : null}
      {props.children}
    </DayCard>
  );
}
