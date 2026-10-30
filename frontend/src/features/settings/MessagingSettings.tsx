/**
 * 拥有：设置里的消息工作台。左边是机器人，右边是接入配置，下面是唤醒规则。
 * 不拥有：平台协议。目录、状态和密钥都来自本机的消息运行时。
 */
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { ArrowLeft, Plus } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Switch } from "../../components/ui/switch";
import * as api from "../../lib/client";
import { writeClipboard } from "../../lib/clipboard";
import { getLocale, useCopy } from "../../lib/i18n";
import { useMedia } from "../../lib/media";
import { cn } from "../../lib/utils";
import { ConfirmDialog } from "../ConfirmDialog";
import { SettingRow, SettingSection } from "./SettingChrome";

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

function stateKind(state: string | undefined): "running" | "error" | "stopped" | "unsaved" {
  if (state === "running") return "running";
  if (state === "error") return "error";
  if (state === "stopped") return "stopped";
  return "unsaved";
}

/** MessagingSettings 是设置里的消息页。机器人工作台在上，唤醒规则在下。 */
export function MessagingSettings() {
  const copy = useCopy();
  const t = copy.settings.messaging;
  const narrow = useMedia("(max-width: 860px)");
  const [snap, setSnap] = useState<Snap | null>(null);
  const [policy, setPolicy] = useState<PolicyForm | null>(null);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [picking, setPicking] = useState(false);
  const [query, setQuery] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [confirmRemove, setConfirmRemove] = useState(false);
  const [testChat, setTestChat] = useState("");
  const [testText, setTestText] = useState("");

  useEffect(() => {
    let stop = false;
    const load = async (first: boolean) => {
      try {
        const next = (await api.imSnapshot()) as Snap;
        if (stop) return;
        setSnap(next);
        setError("");
        if (first) {
          setPolicy(policyFrom(next.settings));
          const firstAccount = next.accounts?.[0];
          if (firstAccount) applyAccount(firstAccount, next.catalog || []);
        }
      } catch (e) {
        if (!stop) setError(errText(e));
      }
    };
    void load(true);
    const timer = window.setInterval(() => void load(false), 3000);
    return () => {
      stop = true;
      window.clearInterval(timer);
    };
  }, []);

  const catalog = snap?.catalog || [];
  const accounts = snap?.accounts || [];
  const platform = catalog.find((p) => p.id === draft?.platform);
  const status = (snap?.statuses || []).find((s) => s.id === draft?.id);
  const saved = accounts.find((a) => a.id === draft?.id);
  const opening = picking || (!draft && accounts.length === 0 && catalog.length > 0);
  const showList = !narrow || (!draft && !opening);
  const showMain = !narrow || !!draft || opening;

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return catalog;
    return catalog.filter((p) =>
      [p.id, p.name.zh, p.name.en, p.summary.zh, p.summary.en].join(" ").toLowerCase().includes(q),
    );
  }, [catalog, query]);

  function applyAccount(account: Account, list: Platform[]) {
    const spec = list.find((p) => p.id === account.platform);
    setPicking(false);
    setDraft({
      id: account.id,
      platform: account.platform,
      label: account.label,
      enabled: account.enabled,
      config: { ...defaultsFor(spec), ...(account.config || {}) },
      secrets: {},
    });
  }

  function choose(spec: Platform) {
    setQuery("");
    setPicking(false);
    setDraft({
      id: "",
      platform: spec.id,
      label: "",
      enabled: false,
      config: defaultsFor(spec),
      secrets: {},
    });
  }

  async function savePolicy() {
    if (!policy) return;
    setBusy("policy");
    try {
      const port = Number(policy.port);
      const rate = Number(policy.rate);
      await api.imSaveSettings({
        wake_prefixes: splitList(policy.wake),
        friend_needs_wake: policy.friend,
        ignore_self: policy.ignoreSelf,
        ignore_at_all: policy.ignoreAtAll,
        admins: splitList(policy.admins),
        allow_users: splitList(policy.users),
        allow_groups: splitList(policy.groups),
        rate_per_minute: Number.isFinite(rate) ? rate : 20,
        reply_mention: policy.mention,
        segmented: policy.segmented,
        listen_host: policy.host.trim(),
        listen_port: Number.isFinite(port) ? port : 8787,
      });
      toast.success(t.saved);
    } catch (e) {
      toast.error(errText(e));
    } finally {
      setBusy("");
    }
  }

  async function saveAccount() {
    if (!draft) return;
    setBusy("account");
    try {
      const savedAccount = (await api.imSaveAccount(
        {
          id: draft.id,
          platform: draft.platform,
          label: draft.label,
          enabled: draft.enabled,
          config: draft.config,
        },
        draft.secrets,
      )) as Account;
      toast.success(t.saved);
      const next = (await api.imSnapshot()) as Snap;
      setSnap(next);
      const fresh = (next.accounts || []).find((a) => a.id === savedAccount.id) || savedAccount;
      applyAccount(fresh, next.catalog || []);
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
      setDraft(null);
      setPicking(false);
      setConfirmRemove(false);
      toast.success(t.removed);
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

  const hook = draft?.id && needsHook(platform, draft.config) ? `${snap?.hook_base || ""}${draft.id}` : "";
  const rows: Account[] = draft && !draft.id
    ? [{ id: "", platform: draft.platform, label: draft.label, enabled: draft.enabled }, ...accounts]
    : accounts;

  return (
    <div className="flex w-full min-w-0 flex-col">
      {error ? <p className="mb-3 text-[13px] text-danger" role="alert">{error}</p> : null}
      <section className="mb-6 w-full min-w-0" data-setting-section-highlight-target="messaging-accounts">
        <div className="mb-3 flex items-end justify-between gap-3">
          <div className="min-w-0">
            <h2 id="messaging-accounts" className="text-[13px] font-medium text-foreground">{copy.settings.sections.messagingAccounts}</h2>
            <p className="mt-1 max-w-2xl text-[12px] leading-[1.55] text-muted">{t.intro}</p>
          </div>
        </div>
        <div className="flex h-[min(36rem,calc(100dvh-10rem))] min-h-[22rem] overflow-hidden rounded-xl border border-border bg-card">
          {showList ? (
            <aside className={cn("flex min-h-0 min-w-0 flex-col", showMain ? "w-[15.5rem] shrink-0 border-r border-border" : "flex-1")}>
              <div className="flex items-center justify-between gap-2 px-3 py-2.5">
                <span className="text-[12px] font-medium text-muted">{t.listTitle}</span>
                <Button type="button" size="sm" variant="ghost" disabled={!catalog.length} onClick={() => { setPicking(true); setQuery(""); }}>
                  <Plus />
                  {t.add}
                </Button>
              </div>
              {rows.length === 0 ? (
                <div className="flex flex-1 flex-col items-start justify-center gap-3 px-4 pb-8">
                  <p className="text-[13px] leading-5 text-muted">{t.noAccounts}</p>
                  <Button type="button" size="sm" disabled={!catalog.length} onClick={() => setPicking(true)}>{t.add}</Button>
                </div>
              ) : (
                <div className="min-h-0 flex-1 overflow-y-auto px-2 pb-2" role="listbox" aria-label={t.listTitle}>
                  {rows.map((account) => {
                    const spec = catalog.find((p) => p.id === account.platform);
                    const st = account.id ? (snap?.statuses || []).find((s) => s.id === account.id) : undefined;
                    const kind = account.id ? stateKind(st?.state || "stopped") : "unsaved";
                    const active = !picking && (account.id ? draft?.id === account.id : !draft?.id);
                    const platformName = phrase(spec?.name);
                    const name = account.label || platformName || account.platform;
                    const statusLine = kind === "unsaved" ? t.unsaved : stateText(t, kind);
                    return (
                      <button
                        key={account.id || "draft"}
                        type="button"
                        role="option"
                        aria-selected={active}
                        onClick={() => account.id ? applyAccount(account, catalog) : setPicking(false)}
                        className={cn(
                          "flex w-full items-center gap-2.5 rounded-lg px-2 py-2 text-left",
                          active ? "bg-lift" : "hover:bg-lift/70",
                        )}
                      >
                        <Mark name={platformName} id={account.platform} />
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-[13px] font-medium text-foreground">{name}</span>
                          <span className="mt-0.5 flex items-center gap-1.5 text-[11px] text-muted">
                            <StatusDot kind={kind} />
                            <span className="truncate">{account.label ? `${statusLine} · ${platformName}` : statusLine}</span>
                          </span>
                        </span>
                      </button>
                    );
                  })}
                </div>
              )}
            </aside>
          ) : null}
          {showMain ? (
            <div className="flex min-h-0 min-w-0 flex-1 flex-col">
              {narrow ? (
                <button
                  type="button"
                  className="flex items-center gap-1.5 px-4 pt-3 text-[12px] text-muted hover:text-foreground"
                  onClick={() => {
                    if (picking && draft) {
                      setPicking(false);
                      return;
                    }
                    setDraft(null);
                    setPicking(false);
                  }}
                >
                  <ArrowLeft className="size-3.5" />
                  {t.back}
                </button>
              ) : null}
              {opening ? (
                <PlatformPicker
                  t={t}
                  query={query}
                  platforms={filtered}
                  onQuery={setQuery}
                  onChoose={choose}
                />
              ) : draft && platform ? (
                <AccountEditor
                  t={t}
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
                  onSend={() => void sendTest()}
                  onCopy={async () => {
                    await writeClipboard(hook);
                    toast.success(t.copied);
                  }}
                />
              ) : (
                <div className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center">
                  <p className="text-[13px] text-muted">{accounts.length ? t.selectHint : t.pickHint}</p>
                </div>
              )}
            </div>
          ) : null}
        </div>
      </section>

      <SettingSection
        id="messaging-policy"
        title={copy.settings.sections.messagingPolicy}
        description={t.policyHint}
        actions={
          <Button type="button" size="sm" disabled={!policy || busy === "policy"} onClick={() => void savePolicy()}>
            {t.savePolicy}
          </Button>
        }
      >
        {policy ? (
          <>
            <SettingRow title={t.wakePrefixes} description={t.wakePrefixesHint}>
              <Input value={policy.wake} onChange={(e) => setPolicy({ ...policy, wake: e.target.value })} />
            </SettingRow>
            <SettingRow title={t.friendNeedsWake}>
              <Switch checked={policy.friend} onCheckedChange={(v) => setPolicy({ ...policy, friend: v })} />
            </SettingRow>
            <SettingRow title={t.ignoreSelf}>
              <Switch checked={policy.ignoreSelf} onCheckedChange={(v) => setPolicy({ ...policy, ignoreSelf: v })} />
            </SettingRow>
            <SettingRow title={t.ignoreAtAll}>
              <Switch checked={policy.ignoreAtAll} onCheckedChange={(v) => setPolicy({ ...policy, ignoreAtAll: v })} />
            </SettingRow>
            <SettingRow title={t.admins} description={t.adminsHint}>
              <Input value={policy.admins} onChange={(e) => setPolicy({ ...policy, admins: e.target.value })} />
            </SettingRow>
            <SettingRow title={t.allowUsers} description={t.allowUsersHint}>
              <Input value={policy.users} onChange={(e) => setPolicy({ ...policy, users: e.target.value })} />
            </SettingRow>
            <SettingRow title={t.allowGroups} description={t.allowGroupsHint}>
              <Input value={policy.groups} onChange={(e) => setPolicy({ ...policy, groups: e.target.value })} />
            </SettingRow>
            <SettingRow title={t.rate} description={t.rateHint}>
              <Input inputMode="numeric" value={policy.rate} onChange={(e) => setPolicy({ ...policy, rate: e.target.value })} />
            </SettingRow>
            <SettingRow title={t.replyMention}>
              <Switch checked={policy.mention} onCheckedChange={(v) => setPolicy({ ...policy, mention: v })} />
            </SettingRow>
            <SettingRow title={t.segmented}>
              <Switch checked={policy.segmented} onCheckedChange={(v) => setPolicy({ ...policy, segmented: v })} />
            </SettingRow>
            <SettingRow title={t.listenHost} description={t.listenHint}>
              <Input value={policy.host} onChange={(e) => setPolicy({ ...policy, host: e.target.value })} />
            </SettingRow>
            <SettingRow title={t.listenPort}>
              <Input inputMode="numeric" value={policy.port} onChange={(e) => setPolicy({ ...policy, port: e.target.value })} />
            </SettingRow>
          </>
        ) : null}
      </SettingSection>

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

function stateText(t: MessagingCopy, kind: "running" | "error" | "stopped" | "unsaved"): string {
  if (kind === "running") return t.stateRunning;
  if (kind === "error") return t.stateError;
  if (kind === "stopped") return t.stateStopped;
  return t.unsaved;
}

function StatusDot({ kind }: { kind: "running" | "error" | "stopped" | "unsaved" }) {
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
    <span className="grid size-8 shrink-0 place-items-center rounded-lg bg-lift text-[12px] font-medium text-foreground">
      {markLetter(name, id)}
    </span>
  );
}

function PlatformPicker(props: {
  t: MessagingCopy;
  query: string;
  platforms: Platform[];
  onQuery: (q: string) => void;
  onChoose: (p: Platform) => void;
}) {
  const { t } = props;
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="shrink-0 px-4 pt-4">
        <h3 className="text-[15px] font-medium tracking-[-0.01em] text-foreground">{t.pickTitle}</h3>
        <p className="mt-1 text-[12px] leading-5 text-muted">{t.pickHint}</p>
        <Input className="mt-3" value={props.query} placeholder={t.searchPlatforms} onChange={(e) => props.onQuery(e.target.value)} autoFocus />
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-3">
        {props.platforms.length === 0 ? (
          <p className="py-8 text-[13px] text-muted">{t.noMatch}</p>
        ) : (
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            {props.platforms.map((p) => (
              <button
                key={p.id}
                type="button"
                onClick={() => props.onChoose(p)}
                className="flex items-start gap-3 rounded-xl border border-border bg-background px-3 py-3 text-left transition-colors hover:bg-lift"
              >
                <Mark name={phrase(p.name)} id={p.id} />
                <span className="min-w-0">
                  <span className="flex items-center gap-2">
                    <span className="truncate text-[13px] font-medium text-foreground">{phrase(p.name)}</span>
                    <span className="shrink-0 text-[11px] text-muted">{transportLabel(t, p.transport)}</span>
                  </span>
                  <span className="mt-1 block text-[12px] leading-5 text-muted">{phrase(p.summary)}</span>
                </span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function AccountEditor(props: {
  t: MessagingCopy;
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
  onSend: () => void;
  onCopy: () => void;
}) {
  const { t, draft, platform } = props;
  const kind = draft.id ? stateKind(props.status?.state || "stopped") : "unsaved";
  const platformName = phrase(platform.name);
  const title = draft.label || platformName;
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="flex shrink-0 items-center gap-3 border-b border-border px-4 py-3">
        <Mark name={platformName} id={platform.id} />
        <div className="min-w-0 flex-1">
          <div className="truncate text-[15px] font-medium tracking-[-0.01em] text-foreground">{title}</div>
          <div className="mt-0.5 flex items-center gap-2 text-[12px] text-muted">
            {draft.label ? <span className="truncate">{platformName}</span> : null}
            <span className={cn(
              "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium",
              kind === "running" && "bg-success/15 text-success",
              kind === "error" && "bg-danger/10 text-danger",
              kind === "stopped" && "bg-lift text-muted",
              kind === "unsaved" && "bg-accent/15 text-accent",
            )}>
              <StatusDot kind={kind} />
              {stateText(t, kind)}
            </span>
          </div>
        </div>
        <label className="flex shrink-0 items-center gap-2 text-[12px] text-muted">
          {t.enabled}
          <Switch checked={draft.enabled} onCheckedChange={(v) => props.onChange({ ...draft, enabled: v })} />
        </label>
      </header>
      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
        <p className="text-[12px] leading-5 text-muted">{phrase(platform.summary)}</p>
        {props.status?.state === "error" && props.status.detail ? (
          <p className="mt-3 rounded-lg bg-danger/10 px-3 py-2 text-[12px] leading-5 text-danger">{props.status.detail}</p>
        ) : null}
        <div className="mt-4 flex flex-col gap-3">
          <FieldLabel title={t.label}>
            <Input value={draft.label} onChange={(e) => props.onChange({ ...draft, label: e.target.value })} />
          </FieldLabel>
          {(platform.fields || []).map((field) => (
            <ConfigField key={field.key} t={t} field={field} draft={draft} platform={platform} saved={props.saved} onChange={props.onChange} />
          ))}
          {props.hook ? (
            <div className="rounded-lg bg-lift px-3 py-2.5">
              <div className="text-[12px] font-medium text-foreground">{t.hook}</div>
              <div className="mt-1.5 flex items-center gap-2">
                <code className="min-w-0 flex-1 truncate font-mono text-[12px] text-muted">{props.hook}</code>
                <Button type="button" size="sm" variant="outline" onClick={props.onCopy}>{t.copy}</Button>
              </div>
            </div>
          ) : needsHook(platform, draft.config) && !draft.id ? (
            <p className="text-[12px] leading-5 text-muted">{t.hookAfterSave}</p>
          ) : null}
          {draft.id ? (
            <details className="rounded-lg border border-border px-3 py-2">
              <summary className="cursor-pointer text-[12px] font-medium text-foreground">{t.testSend}</summary>
              <p className="mt-1 text-[12px] leading-5 text-muted">{t.testHint}</p>
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
        </div>
      </div>
      <footer className="flex shrink-0 items-center justify-between gap-2 border-t border-border px-4 py-3">
        {draft.id ? (
          <Button type="button" size="sm" variant="ghost" className="text-danger hover:bg-danger/10" onClick={props.onRemove}>{t.remove}</Button>
        ) : <span />}
        <Button type="button" size="sm" disabled={props.busy === "account"} onClick={props.onSave}>{t.saveAccount}</Button>
      </footer>
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
                  "h-7 flex-1 rounded-md text-[12px]",
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
      <span className="text-[12px] font-medium text-foreground">{title}</span>
      {hint ? <span className="text-[12px] leading-5 text-muted">{hint}</span> : null}
      {children}
    </div>
  );
}
