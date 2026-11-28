/**
 * 拥有：设置里的「聊天软件」页：机器人列表、加机器人和改机器人的弹框，以及谁能叫醒它的规则。
 * 不拥有：平台协议。目录、状态和密钥都来自本机的消息运行时。
 * 机器人一行一个，开关就地生效；点一行在弹框里改，加机器人先选平台再填。规则改了就存，没有保存按钮。
 */
import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { ArrowLeft, Plus } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Switch } from "../../components/ui/switch";
import * as api from "../../lib/client";
import { writeClipboard } from "../../lib/clipboard";
import { getLocale, useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { ConfirmDialog } from "../ConfirmDialog";
import { FormDialog } from "../FormDialog";
import { SettingDisclosure, SettingEmpty, SettingRow, SettingSection } from "./SettingChrome";

type Phrase = { zh: string; en: string };
type Field = { key: string; label: Phrase; help?: Phrase; secret?: boolean; default?: string };
type Platform = { id: string; name: Phrase; transport: string; summary: Phrase; fields?: Field[] };
type Account = {
  id: string;
  platform: string;
  label: string;
  enabled: boolean;
  config?: Record<string, string>;
  has_secret?: Record<string, boolean>;
};
type Status = { id: string; state: string; detail?: string };
type Snap = {
  catalog?: Platform[];
  settings?: Record<string, unknown>;
  accounts?: Account[];
  statuses?: Status[];
  hook_base?: string;
};
type PolicyForm = {
  wake: string;
  friend: boolean;
  ignoreSelf: boolean;
  ignoreAtAll: boolean;
  admins: string;
  users: string;
  groups: string;
  rate: string;
  mention: boolean;
  segmented: boolean;
  host: string;
  port: string;
};
type Draft = {
  id: string;
  platform: string;
  label: string;
  enabled: boolean;
  config: Record<string, string>;
  secrets: Record<string, string>;
};
type Kind = "running" | "error" | "stopped" | "unsaved";

const MODES: Record<string, string[]> = {
  lark: ["webhook", "socket"],
  dingtalk: ["stream", "webhook"],
  wecom_ai_bot: ["long", "webhook"],
  slack: ["socket", "webhook"],
};

function phrase(p: Phrase | undefined): string {
  if (!p) return "";
  return getLocale() === "zh-CN" ? p.zh || p.en : p.en || p.zh;
}

function listText(v: unknown): string {
  return Array.isArray(v) ? v.map((x) => String(x).trim()).filter(Boolean).join(", ") : "";
}

function splitList(s: string): string[] {
  return s.split(/[,，\n]/).map((x) => x.trim()).filter(Boolean);
}

function policyFrom(raw: Record<string, unknown> | undefined): PolicyForm {
  return {
    wake: listText(raw?.wake_prefixes) || "/",
    friend: !!raw?.friend_needs_wake,
    ignoreSelf: raw?.ignore_self !== false,
    ignoreAtAll: !!raw?.ignore_at_all,
    admins: listText(raw?.admins),
    users: listText(raw?.allow_users),
    groups: listText(raw?.allow_groups),
    rate: String(raw?.rate_per_minute ?? 20),
    mention: !!raw?.reply_mention,
    segmented: !!raw?.segmented,
    host: String(raw?.listen_host || "127.0.0.1"),
    port: String(raw?.listen_port || 8787),
  };
}

/** policyBody 把表单收成运行时要的形状。数字填坏了时退回默认值。 */
function policyBody(p: PolicyForm) {
  const port = Number(p.port);
  const rate = Number(p.rate);
  return {
    wake_prefixes: splitList(p.wake),
    friend_needs_wake: p.friend,
    ignore_self: p.ignoreSelf,
    ignore_at_all: p.ignoreAtAll,
    admins: splitList(p.admins),
    allow_users: splitList(p.users),
    allow_groups: splitList(p.groups),
    rate_per_minute: Number.isFinite(rate) ? rate : 20,
    reply_mention: p.mention,
    segmented: p.segmented,
    listen_host: p.host.trim(),
    listen_port: Number.isFinite(port) ? port : 8787,
  };
}

function defaultsFor(p: Platform | undefined): Record<string, string> {
  const config: Record<string, string> = {};
  for (const field of p?.fields || []) {
    if (!field.secret && field.default) config[field.key] = field.default;
  }
  return config;
}

function needsHook(p: Platform | undefined, config: Record<string, string>): boolean {
  if (!p) return false;
  if (p.transport === "webhook") return true;
  const mode = (config.connection_mode || "").toLowerCase();
  if (mode === "webhook") return true;
  if (!mode && p.id === "lark") return true;
  return false;
}

function errText(e: unknown): string {
  if (e instanceof Error && e.message) return e.message;
  return String(e || "");
}

function markLetter(name: string, id: string): string {
  const chars = name.replace(/\s/g, "");
  return (chars[0] || id[0] || "?").toUpperCase();
}

type MessagingCopy = ReturnType<typeof useCopy>["settings"]["messaging"];

function modeLabel(t: MessagingCopy, mode: string): string {
  const known = t.modes as Record<string, string>;
  return known[mode] || mode;
}

function transportLabel(t: MessagingCopy, transport: string): string {
  const known = t.transports as Record<string, string>;
  return known[transport] || transport;
}

function stateKind(state: string | undefined): Kind {
  if (state === "running") return "running";
  if (state === "error") return "error";
  if (state === "stopped") return "stopped";
  return "unsaved";
}

function stateText(t: MessagingCopy, kind: Kind): string {
  if (kind === "running") return t.stateRunning;
  if (kind === "error") return t.stateError;
  if (kind === "stopped") return t.stateStopped;
  return t.unsaved;
}

/** MessagingSettings 是「聊天软件」整页：上面是机器人，有机器人之后下面才出现唤醒规则。 */
export function MessagingSettings() {
  const copy = useCopy();
  const t = copy.settings.messaging;
  const [snap, setSnap] = useState<Snap | null>(null);
  const [policy, setPolicy] = useState<PolicyForm | null>(null);
  // savedPolicy 是上一次存下的规则，没变就不再提交，免得失焦就弹一次「已保存」。
  const savedPolicy = useRef("");
  const [dialog, setDialog] = useState<"pick" | "edit" | null>(null);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [query, setQuery] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [confirmRemove, setConfirmRemove] = useState(false);
  const [testChat, setTestChat] = useState("");
  const [testText, setTestText] = useState("");

  useEffect(() => {
    let stop = false;
    const load = async () => {
      try {
        const next = (await api.imSnapshot()) as Snap;
        if (stop) return;
        setSnap(next);
        setError("");
        if (!savedPolicy.current) {
          const form = policyFrom(next.settings);
          savedPolicy.current = JSON.stringify(policyBody(form));
          setPolicy(form);
        }
      } catch (e) {
        if (!stop) setError(errText(e));
      }
    };
    void load();
    const timer = window.setInterval(() => void load(), 3000);
    return () => {
      stop = true;
      window.clearInterval(timer);
    };
  }, []);

  const catalog = snap?.catalog || [];
  const accounts = snap?.accounts || [];
  const statuses = snap?.statuses || [];
  const platform = catalog.find((p) => p.id === draft?.platform);
  const status = statuses.find((s) => s.id === draft?.id);
  const saved = accounts.find((a) => a.id === draft?.id);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return catalog;
    return catalog.filter((p) =>
      [p.id, p.name.zh, p.name.en, p.summary.zh, p.summary.en].join(" ").toLowerCase().includes(q),
    );
  }, [catalog, query]);

  function draftOf(account: Account, list: Platform[] = catalog): Draft {
    const spec = list.find((p) => p.id === account.platform);
    return {
      id: account.id,
      platform: account.platform,
      label: account.label,
      enabled: account.enabled,
      config: { ...defaultsFor(spec), ...(account.config || {}) },
      secrets: {},
    };
  }

  function openAccount(account: Account) {
    setDraft(draftOf(account));
    setTestChat("");
    setTestText("");
    setDialog("edit");
  }

  function openPicker() {
    setQuery("");
    setDraft(null);
    setDialog("pick");
  }

  function choose(spec: Platform) {
    setDraft({ id: "", platform: spec.id, label: "", enabled: true, config: defaultsFor(spec), secrets: {} });
    setDialog("edit");
  }

  function closeDialog() {
    if (busy === "account" || busy === "remove") return;
    setDialog(null);
    setDraft(null);
  }

  async function saveAccount() {
    if (!draft || !platform) return;
    setBusy("account");
    try {
      const out = (await api.imSaveAccount(
        { id: draft.id, platform: draft.platform, label: draft.label, enabled: draft.enabled, config: draft.config },
        draft.secrets,
      )) as Account;
      const next = (await api.imSnapshot()) as Snap;
      setSnap(next);
      toast.success(t.saved);
      const fresh = (next.accounts || []).find((a) => a.id === out.id) || out;
      // 新建的回调机器人要把回调地址抄到平台上，弹框留着让人复制；其余存完就关。
      if (!draft.id && needsHook(platform, fresh.config || draft.config)) {
        setDraft(draftOf(fresh, next.catalog || catalog));
      } else {
        setDialog(null);
        setDraft(null);
      }
    } catch (e) {
      toast.error(errText(e));
    } finally {
      setBusy("");
    }
  }

  async function removeAccount() {
    if (!draft?.id) return;
    setBusy("remove");
    try {
      await api.imDeleteAccount(draft.id);
      setConfirmRemove(false);
      setDialog(null);
      setDraft(null);
      toast.success(t.removed);
      setSnap((await api.imSnapshot()) as Snap);
    } catch (e) {
      toast.error(errText(e));
    } finally {
      setBusy("");
    }
  }

  /** toggleAccount 在列表里直接开关一个机器人。密钥留空，运行时保留原来的。 */
  async function toggleAccount(account: Account, enabled: boolean) {
    setBusy(`toggle:${account.id}`);
    try {
      await api.imSaveAccount(
        { id: account.id, platform: account.platform, label: account.label, enabled, config: account.config || {} },
        {},
      );
      setSnap((await api.imSnapshot()) as Snap);
    } catch (e) {
      toast.error(errText(e));
    } finally {
      setBusy("");
    }
  }

  async function sendTest() {
    if (!draft?.id) return;
    setBusy("send");
    try {
      await api.imSend(draft.id, testChat, testText);
      toast.success(t.sent);
    } catch (e) {
      toast.error(errText(e));
    } finally {
      setBusy("");
    }
  }

  /** savePolicy 在开关拨动或输入框失焦时存规则。和上次存的一样就不提交。 */
  async function savePolicy(next: PolicyForm) {
    const body = policyBody(next);
    const key = JSON.stringify(body);
    if (key === savedPolicy.current) return;
    try {
      await api.imSaveSettings(body);
      savedPolicy.current = key;
      toast.success(t.saved, { id: "im-policy-saved" });
    } catch (e) {
      toast.error(errText(e));
    }
  }

  function commit(next: PolicyForm) {
    setPolicy(next);
    void savePolicy(next);
  }

  const hook = draft?.id && needsHook(platform, draft.config) ? `${snap?.hook_base || ""}${draft.id}` : "";

  return (
    <div className="flex w-full min-w-0 flex-col">
      {error ? <p className="mb-3 px-1 text-ui-footnote leading-[1.5] text-danger" role="alert">{error}</p> : null}

      <SettingSection
        id="messaging-accounts"
        title={copy.settings.sections.messagingAccounts}
        description={t.intro}
        headerActions={
          <Button size="sm" variant="ghost" disabled={!catalog.length} onClick={openPicker}>
            <Plus aria-hidden />
            {t.add}
          </Button>
        }
      >
        {accounts.length === 0 ? (
          <SettingEmpty compact>{t.noAccounts}</SettingEmpty>
        ) : (
          accounts.map((account) => (
            <BotRow
              key={account.id}
              t={t}
              account={account}
              platform={catalog.find((p) => p.id === account.platform)}
              status={statuses.find((s) => s.id === account.id)}
              busy={busy === `toggle:${account.id}`}
              onOpen={() => openAccount(account)}
              onToggle={(on) => void toggleAccount(account, on)}
            />
          ))
        )}
      </SettingSection>

      {policy && accounts.length > 0 ? (
        <SettingSection
          id="messaging-policy"
          title={copy.settings.sections.messagingPolicy}
          description={t.policyHint}
          footnote={t.autosave}
        >
          <PolicyInput
            title={t.wakePrefixes}
            description={t.wakePrefixesHint}
            value={policy.wake}
            onChange={(v) => setPolicy({ ...policy, wake: v })}
            onCommit={() => void savePolicy(policy)}
          />
          <SettingRow title={t.friendNeedsWake}>
            <Switch checked={policy.friend} aria-label={t.friendNeedsWake} onCheckedChange={(v) => commit({ ...policy, friend: v })} />
          </SettingRow>
          <SettingDisclosure
            title={t.access}
            description={t.accessHint}
            defaultOpen={!!(policy.admins || policy.users || policy.groups)}
          >
            <PolicyInput
              title={t.admins}
              description={t.adminsHint}
              value={policy.admins}
              onChange={(v) => setPolicy({ ...policy, admins: v })}
              onCommit={() => void savePolicy(policy)}
            />
            <PolicyInput
              title={t.allowUsers}
              description={t.allowUsersHint}
              value={policy.users}
              onChange={(v) => setPolicy({ ...policy, users: v })}
              onCommit={() => void savePolicy(policy)}
            />
            <PolicyInput
              title={t.allowGroups}
              description={t.allowGroupsHint}
              value={policy.groups}
              onChange={(v) => setPolicy({ ...policy, groups: v })}
              onCommit={() => void savePolicy(policy)}
            />
          </SettingDisclosure>
          <SettingDisclosure title={t.advanced} description={t.advancedHint}>
            <SettingRow title={t.ignoreSelf}>
              <Switch checked={policy.ignoreSelf} aria-label={t.ignoreSelf} onCheckedChange={(v) => commit({ ...policy, ignoreSelf: v })} />
            </SettingRow>
            <SettingRow title={t.ignoreAtAll}>
              <Switch checked={policy.ignoreAtAll} aria-label={t.ignoreAtAll} onCheckedChange={(v) => commit({ ...policy, ignoreAtAll: v })} />
            </SettingRow>
            <PolicyInput
              numeric
              title={t.rate}
              description={t.rateHint}
              value={policy.rate}
              onChange={(v) => setPolicy({ ...policy, rate: v })}
              onCommit={() => void savePolicy(policy)}
            />
            <SettingRow title={t.replyMention}>
              <Switch checked={policy.mention} aria-label={t.replyMention} onCheckedChange={(v) => commit({ ...policy, mention: v })} />
            </SettingRow>
            <SettingRow title={t.segmented}>
              <Switch checked={policy.segmented} aria-label={t.segmented} onCheckedChange={(v) => commit({ ...policy, segmented: v })} />
            </SettingRow>
            <PolicyInput
              title={t.listenHost}
              description={t.listenHint}
              value={policy.host}
              onChange={(v) => setPolicy({ ...policy, host: v })}
              onCommit={() => void savePolicy(policy)}
            />
            <PolicyInput
              numeric
              title={t.listenPort}
              value={policy.port}
              onChange={(v) => setPolicy({ ...policy, port: v })}
              onCommit={() => void savePolicy(policy)}
            />
          </SettingDisclosure>
        </SettingSection>
      ) : null}

      <FormDialog open={dialog === "pick"} title={t.pickTitle} description={t.pickHint} onClose={closeDialog}>
        <PlatformPicker t={t} query={query} platforms={filtered} onQuery={setQuery} onChoose={choose} />
      </FormDialog>

      <FormDialog
        open={dialog === "edit" && !!draft && !!platform && !confirmRemove}
        title={draft?.label || phrase(platform?.name)}
        description={phrase(platform?.summary)}
        onClose={closeDialog}
      >
        {draft && platform ? (
          <AccountEditor
            t={t}
            cancelLabel={copy.dialog.cancel}
            draft={draft}
            platform={platform}
            status={status}
            saved={saved}
            hook={hook}
            busy={busy}
            testChat={testChat}
            testText={testText}
            onChange={setDraft}
            onTestChat={setTestChat}
            onTestText={setTestText}
            onSave={() => void saveAccount()}
            onRemove={() => setConfirmRemove(true)}
            onBack={draft.id ? undefined : openPicker}
            onCancel={closeDialog}
            onSend={() => void sendTest()}
            onCopy={async () => {
              await writeClipboard(hook);
              toast.success(t.copied);
            }}
          />
        ) : null}
      </FormDialog>

      <ConfirmDialog
        open={confirmRemove}
        danger
        title={t.removeTitle}
        body={t.removeBody}
        confirmLabel={t.remove}
        onCancel={() => setConfirmRemove(false)}
        onConfirm={() => void removeAccount()}
      />
    </div>
  );
}

/** BotRow 是列表里的一个机器人：点左边打开弹框改，右边开关就地生效。 */
function BotRow(props: {
  t: MessagingCopy;
  account: Account;
  platform?: Platform;
  status?: Status;
  busy: boolean;
  onOpen: () => void;
  onToggle: (on: boolean) => void;
}) {
  const { t, account } = props;
  const platformName = phrase(props.platform?.name) || account.platform;
  const name = account.label || platformName;
  const kind = stateKind(props.status?.state || "stopped");
  const line = account.label ? `${stateText(t, kind)} · ${platformName}` : stateText(t, kind);
  return (
    <div className="flex min-h-8 w-full min-w-0 items-center gap-3" data-testid={`im-bot-${account.id}`}>
      <button
        type="button"
        onClick={props.onOpen}
        title={kind === "error" ? props.status?.detail : undefined}
        className="group flex min-w-0 flex-1 items-center gap-3 text-left"
      >
        <Mark name={platformName} id={account.platform} />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-ui-body-l leading-[1.45] text-foreground transition-colors group-hover:text-accent">{name}</span>
          <span className={cn("mt-0.5 flex items-center gap-1.5 text-ui-footnote", kind === "error" ? "text-danger" : "text-muted")}>
            <StatusDot kind={kind} />
            <span className="truncate">{line}</span>
          </span>
        </span>
      </button>
      <Switch checked={account.enabled} disabled={props.busy} aria-label={`${t.enabled} ${name}`} onCheckedChange={props.onToggle} />
    </div>
  );
}

/** PolicyInput 是规则里的一格文字：失焦或回车时存。numeric 为真时弹数字键盘。 */
function PolicyInput(props: {
  title: string;
  description?: string;
  value: string;
  numeric?: boolean;
  onChange: (v: string) => void;
  onCommit: () => void;
}) {
  return (
    <SettingRow title={props.title} description={props.description}>
      <Input
        className="h-8 w-full text-ui-body"
        aria-label={props.title}
        inputMode={props.numeric ? "numeric" : undefined}
        value={props.value}
        onChange={(e) => props.onChange(e.target.value)}
        onBlur={props.onCommit}
        onKeyDown={(e) => {
          if (e.key === "Enter") e.currentTarget.blur();
        }}
      />
    </SettingRow>
  );
}

function StatusDot({ kind }: { kind: Kind }) {
  return (
    <span
      className={cn(
        "size-1.5 shrink-0 rounded-full",
        kind === "running" && "bg-success",
        kind === "error" && "bg-danger",
        kind === "stopped" && "bg-muted",
        kind === "unsaved" && "bg-accent",
      )}
      aria-hidden
    />
  );
}

function Mark({ name, id }: { name: string; id: string }) {
  return (
    <span className="grid size-8 shrink-0 place-items-center rounded-lg bg-lift text-ui-footnote font-medium text-foreground">
      {markLetter(name, id)}
    </span>
  );
}

/** PlatformPicker 是加机器人的第一步：搜一下，点一个平台。 */
function PlatformPicker(props: {
  t: MessagingCopy;
  query: string;
  platforms: Platform[];
  onQuery: (q: string) => void;
  onChoose: (p: Platform) => void;
}) {
  const { t } = props;
  return (
    <div className="mt-4 flex flex-col gap-3">
      <Input
        autoFocus
        value={props.query}
        placeholder={t.searchPlatforms}
        aria-label={t.searchPlatforms}
        onChange={(e) => props.onQuery(e.target.value)}
      />
      {props.platforms.length === 0 ? (
        <p className="py-6 text-ui-footnote text-muted">{t.noMatch}</p>
      ) : (
        <div className="-mx-2 flex max-h-[22rem] flex-col overflow-y-auto">
          {props.platforms.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => props.onChoose(p)}
              className="flex items-start gap-3 rounded-lg px-2 py-2 text-left transition-colors hover:bg-lift"
            >
              <Mark name={phrase(p.name)} id={p.id} />
              <span className="min-w-0 flex-1">
                <span className="flex items-center gap-2">
                  <span className="truncate text-ui-body font-medium text-foreground">{phrase(p.name)}</span>
                  <span className="shrink-0 text-ui-caption text-muted">{transportLabel(t, p.transport)}</span>
                </span>
                <span className="mt-0.5 block text-ui-footnote leading-5 text-muted">{phrase(p.summary)}</span>
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

/** AccountEditor 是弹框里的表单：状态和启用、名称、平台字段、回调地址、发测试，最后是操作。 */
function AccountEditor(props: {
  t: MessagingCopy;
  cancelLabel: string;
  draft: Draft;
  platform: Platform;
  status?: Status;
  saved?: Account;
  hook: string;
  busy: string;
  testChat: string;
  testText: string;
  onChange: (next: Draft) => void;
  onTestChat: (v: string) => void;
  onTestText: (v: string) => void;
  onSave: () => void;
  onRemove: () => void;
  onBack?: () => void;
  onCancel: () => void;
  onSend: () => void;
  onCopy: () => void;
}) {
  const { t, draft, platform } = props;
  const kind: Kind = draft.id ? stateKind(props.status?.state || "stopped") : "unsaved";
  return (
    <div className="mt-4 flex flex-col gap-3">
      <div className="flex items-center justify-between gap-3 rounded-lg bg-lift px-3 py-2">
        <span className="inline-flex items-center gap-1.5 text-ui-footnote text-muted">
          <StatusDot kind={kind} />
          {stateText(t, kind)}
        </span>
        <label className="flex items-center gap-2 text-ui-footnote text-muted">
          {t.enabled}
          <Switch checked={draft.enabled} onCheckedChange={(v) => props.onChange({ ...draft, enabled: v })} />
        </label>
      </div>
      {props.status?.state === "error" && props.status.detail ? (
        <p className="rounded-lg bg-danger/10 px-3 py-2 text-ui-footnote leading-5 text-danger">{props.status.detail}</p>
      ) : null}
      <FieldLabel title={t.label}>
        <Input value={draft.label} placeholder={phrase(platform.name)} onChange={(e) => props.onChange({ ...draft, label: e.target.value })} />
      </FieldLabel>
      {(platform.fields || []).map((field) => (
        <ConfigField key={field.key} t={t} field={field} draft={draft} platform={platform} saved={props.saved} onChange={props.onChange} />
      ))}
      {props.hook ? (
        <div className="rounded-lg bg-lift px-3 py-2.5">
          <div className="text-ui-footnote font-medium text-foreground">{t.hook}</div>
          <div className="mt-1.5 flex items-center gap-2">
            <code className="min-w-0 flex-1 truncate font-mono text-ui-footnote text-muted">{props.hook}</code>
            <Button type="button" size="sm" variant="outline" onClick={props.onCopy}>{t.copy}</Button>
          </div>
        </div>
      ) : needsHook(platform, draft.config) && !draft.id ? (
        <p className="text-ui-footnote leading-5 text-muted">{t.hookAfterSave}</p>
      ) : null}
      {draft.id ? (
        <details className="setting-disclosure rounded-lg border border-border px-3 py-2">
          <summary className="cursor-pointer list-none text-ui-footnote font-medium text-foreground">{t.testSend}</summary>
          <p className="mt-1 text-ui-footnote leading-5 text-muted">{t.testHint}</p>
          <div className="mt-2 flex flex-col gap-2">
            <FieldLabel title={t.testChat}>
              <Input value={props.testChat} onChange={(e) => props.onTestChat(e.target.value)} />
            </FieldLabel>
            <FieldLabel title={t.testText}>
              <Input value={props.testText} onChange={(e) => props.onTestText(e.target.value)} />
            </FieldLabel>
            <div className="flex justify-end">
              <Button type="button" size="sm" variant="outline" disabled={props.busy === "send"} onClick={props.onSend}>{t.testSend}</Button>
            </div>
          </div>
        </details>
      ) : null}
      <div className="dialog-actions">
        {draft.id ? (
          <Button type="button" size="sm" variant="ghost" className="mr-auto text-danger hover:bg-danger/10" onClick={props.onRemove}>
            {t.remove}
          </Button>
        ) : props.onBack ? (
          <Button type="button" size="sm" variant="ghost" className="mr-auto" onClick={props.onBack}>
            <ArrowLeft aria-hidden />
            {t.backToPlatforms}
          </Button>
        ) : null}
        <Button type="button" size="sm" variant="lift" onClick={props.onCancel}>{props.cancelLabel}</Button>
        <Button type="button" size="sm" disabled={props.busy === "account"} onClick={props.onSave}>{t.saveAccount}</Button>
      </div>
    </div>
  );
}

function ConfigField(props: {
  t: MessagingCopy;
  field: Field;
  draft: Draft;
  platform: Platform;
  saved?: Account;
  onChange: (next: Draft) => void;
}) {
  const { t, field, draft, platform } = props;
  const modes = field.key === "connection_mode" ? MODES[platform.id] : undefined;
  const kept = !!props.saved?.has_secret?.[field.key];
  const hint = [phrase(field.help), field.secret && kept ? t.secretKeep : ""].filter(Boolean).join(" ");
  const value = modes ? draft.config[field.key] || field.default || modes[0] : "";
  return (
    <FieldLabel title={phrase(field.label)} hint={hint}>
      {modes ? (
        <div className="flex gap-1 rounded-lg bg-lift p-0.5" role="radiogroup" aria-label={phrase(field.label)}>
          {modes.map((mode) => {
            const on = value === mode;
            return (
              <button
                key={mode}
                type="button"
                role="radio"
                aria-checked={on}
                onClick={() => props.onChange({ ...draft, config: { ...draft.config, [field.key]: mode } })}
                className={cn(
                  "h-7 flex-1 rounded-md text-ui-footnote",
                  on ? "bg-background text-foreground shadow-[var(--shadow-card)]" : "text-muted hover:text-foreground",
                )}
              >
                {modeLabel(t, mode)}
              </button>
            );
          })}
        </div>
      ) : (
        <Input
          type={field.secret ? "password" : "text"}
          value={field.secret ? draft.secrets[field.key] || "" : draft.config[field.key] || ""}
          placeholder={field.secret && kept ? "••••" : field.default || ""}
          onChange={(e) => {
            if (field.secret) {
              props.onChange({ ...draft, secrets: { ...draft.secrets, [field.key]: e.target.value } });
              return;
            }
            props.onChange({ ...draft, config: { ...draft.config, [field.key]: e.target.value } });
          }}
        />
      )}
    </FieldLabel>
  );
}

function FieldLabel({ title, hint, children }: { title: string; hint?: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-ui-footnote font-medium text-foreground">{title}</span>
      {hint ? <span className="text-ui-footnote leading-5 text-muted">{hint}</span> : null}
      {children}
    </div>
  );
}
