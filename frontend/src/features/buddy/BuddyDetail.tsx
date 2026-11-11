/**
 * 拥有：选中一个 agent 时的详情：头部（字形、名字、厂商、版本、来源和路径）、状态和它手上的工单、能力、登录、行内配置表单和危险区。
 * 不拥有：按钮背后的动作（buddy-actions.ts）和对话框（BuddyWorkspace 统一挂）。密钥只显示「已保存」，保存后输入框清空。
 */

import { ChevronLeft, Copy as CopyIcon, Loader2 } from "lucide-react";
import { useState, type ReactNode } from "react";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { Input, Textarea } from "../../components/ui/input";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import { StatusDot } from "../../components/ui/status-dot";
import { Switch } from "../../components/ui/switch";
import { buddy, type BuddyConfigPatch, type BuddyView } from "../../lib/client";
import { writeClipboard } from "../../lib/clipboard";
import { getLocale, useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";
import { ticketAgent } from "../superagent/agent-model";
import { useNeeds } from "../superagent/needs";
import { taskStatus } from "../superagent/TaskPage";
import { BuddyMark } from "./BuddyMark";
import { ApprovalSelect, draftPatch, emptyDraft, ModelField, SecretRow, type KeyDraft } from "./BuddyConfigDialog";
import { BuddyMoreMenu, BuddyPrimary } from "./BuddyRow";
import { buddyConfigure, buddyDo, buddyLogin } from "./buddy-actions";
import {
  buddyPrimary,
  buddyStatusText,
  buddyTone,
  checkedAgo,
  envText,
  fill,
  installed,
  parseEnv,
  progressBar,
  splitLines,
  type BuddyCopy,
} from "./buddy-model";
import { useBuddy } from "./buddy-store";

const FIELD = "space-y-1.5";
const FIELD_LABEL = "block text-ui-footnote text-muted";
const AREA =
  "min-h-[4.5rem] rounded-lg border border-border bg-background px-2.5 py-2 font-mono text-ui-footnote leading-5 focus-visible:border-accent/45 focus-visible:ring-2 focus-visible:ring-accent/20";

/** Section 是详情里的一段：标题加内容，段与段之间一条细线。 */
function Section({ title, children, testId }: { title: string; children: ReactNode; testId?: string }) {
  return (
    <section className="border-t border-border/60 py-5" data-testid={testId}>
      <h3 className="mb-3 text-ui-body font-semibold text-foreground">{title}</h3>
      {children}
    </section>
  );
}

/** sourceLabel 是来源的说法：托管、系统或自定义命令。 */
function sourceLabel(v: BuddyView, c: BuddyCopy): string {
  if (v.install.kind === "managed") return c.detail.sourceManaged;
  if (v.install.kind === "custom") return c.detail.sourceCustom;
  return c.detail.sourceSystem;
}

/** CopyPath 是路径后面的复制按钮。复制成功后按钮文字短暂变成「已复制」。 */
function CopyPath({ path, c }: { path: string; c: BuddyCopy }) {
  const [done, setDone] = useState(false);
  return (
    <Button
      size="xs"
      variant="ghost"
      aria-label={done ? c.detail.copied : c.detail.copyPath}
      onClick={() => {
        void writeClipboard(path).then((ok) => {
          if (!ok) return;
          setDone(true);
          window.setTimeout(() => setDone(false), 1500);
        });
      }}
    >
      <CopyIcon aria-hidden />
      {done ? c.detail.copied : null}
    </Button>
  );
}

/** Caps 用小标签列出对方声明的能力。一项都没有（还没探测过）时不画。 */
function Caps({ v, c }: { v: BuddyView; c: BuddyCopy }) {
  const caps = v.caps;
  if (!caps) return null;
  const have = (["resume", "systemPrompt", "mcpHttp", "image", "dirs"] as const).filter((k) => caps[k]);
  if (!have.length) return null;
  return (
    <Section title={c.detail.caps}>
      <div className="flex flex-wrap gap-1.5">
        {have.map((k) => (
          <Badge key={k}>{c.detail.cap[k]}</Badge>
        ))}
      </div>
    </Section>
  );
}

/** Tasks 列出这一项手上没结束的工单。点一条就回到对话，把那张工单放到中间。 */
function Tasks({ v }: { v: BuddyView }) {
  const copy = useCopy();
  const tickets = useNeeds((s) => s.view?.tickets);
  const mine = (tickets ?? []).filter((t) => t.stage !== "ended" && ticketAgent(t) === v.id);
  if (!mine.length) return null;
  return (
    <div className="pt-2" data-testid="buddy-tasks">
      <p className="text-ui-footnote text-muted">{copy.buddy.detail.tasks}</p>
      <ul className="mt-1 divide-y divide-border/60">
        {mine.map((t) => (
          <li key={t.id}>
            <button
              type="button"
              className="flex w-full min-w-0 items-center gap-3 py-1.5 text-left text-ui-body text-foreground/85 hover:text-foreground"
              onClick={() => {
                useNeeds.getState().setFocusTask(t.id);
                useUI.getState().showConversation();
              }}
            >
              <span className="min-w-0 flex-1 truncate">{t.title || copy.work.taskEmpty}</span>
              <span className="shrink-0 text-ui-caption text-muted">{taskStatus(t, copy.work)}</span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

/** Login 是登录一段。要登录时每种方式一个按钮，终端型的附一句说明；在线时显示已登录，对方支持时可以退出登录。 */
function Login({ v, c }: { v: BuddyView; c: BuddyCopy }) {
  const pending = useBuddy((s) => s.pending[v.id] || "");
  const put = useBuddy((s) => s.put);
  const run = useBuddy((s) => s.run);
  const methods = v.auth?.methods ?? [];
  if (v.state === "needs_auth") {
    return (
      <Section title={c.detail.login} testId="buddy-login">
        <div className="space-y-3">
          {(methods.length ? methods : [{ id: "", name: c.action.login, type: "terminal" as const, description: "" }]).map((m) => (
            <div key={m.id || "default"} className="space-y-1">
              <Button size="sm" variant="outline" disabled={!!pending} aria-busy={pending === "login" || undefined} onClick={() => void buddyLogin(v, m.id)}>
                {pending === "login" ? <Loader2 className="animate-spin" aria-hidden /> : null}
                {m.name}
              </Button>
              {m.description ? <p className="text-ui-footnote text-muted">{m.description}</p> : null}
              {m.type === "terminal" ? <p className="text-ui-footnote text-muted">{c.detail.terminalNote}</p> : null}
            </div>
          ))}
          {v.detail ? <p className="text-ui-footnote text-muted" role="status">{v.detail}</p> : null}
        </div>
      </Section>
    );
  }
  if (v.state !== "online" || !methods.length) return null;
  return (
    <Section title={c.detail.login}>
      <div className="flex items-center gap-3">
        <span className="flex items-center gap-1.5 text-ui-body text-foreground">
          <StatusDot tone="done" />
          {c.detail.signedIn}
        </span>
        {v.caps?.logout ? (
          <Button
            size="sm"
            variant="ghost"
            disabled={!!pending}
            onClick={() => void run(v.id, "logout", () => buddy.logout(v.id)).then(put)}
          >
            {c.action.logout}
          </Button>
        ) : null}
      </div>
    </Section>
  );
}

/** ConfigForm 是行内配置表单。初值取自第一次画出时的视图，之后轮询带来的新视图不覆盖人正在改的内容；保存成功后用返回的视图重置。 */
function ConfigForm({ v, c }: { v: BuddyView; c: BuddyCopy }) {
  const pending = useBuddy((s) => s.pending[v.id] || "");
  const [base, setBase] = useState(v.config);
  const [enabled, setEnabled] = useState(base.enabled);
  const [model, setModel] = useState(base.model ?? "");
  const [approval, setApproval] = useState(base.approval);
  const [maxParallel, setMaxParallel] = useState(base.maxParallel);
  const [args, setArgs] = useState((base.args ?? []).join("\n"));
  const [env, setEnv] = useState(envText(base.env));
  const [strengths, setStrengths] = useState(base.strengths ?? "");
  const [draft, setDraft] = useState<KeyDraft>(() => emptyDraft(v.config.secrets));
  const [note, setNote] = useState("");
  const secrets = v.config.secrets;

  /** save 只把改过的字段发给后端。成功后清空密钥输入，显示「已保存」。 */
  const save = async () => {
    const parsed = parseEnv(env);
    if (parsed.bad) {
      setNote(fill(c.detail.envBad, { line: parsed.bad }));
      return;
    }
    const patch: BuddyConfigPatch = { ...draftPatch(secrets, draft) };
    if (enabled !== base.enabled) patch.enabled = enabled;
    if (model !== (base.model ?? "")) patch.model = model;
    if (approval !== base.approval) patch.approval = approval;
    if (maxParallel !== base.maxParallel) patch.maxParallel = maxParallel;
    const argList = splitLines(args);
    if (argList.join("\n") !== (base.args ?? []).join("\n")) patch.args = argList;
    if (envText(parsed.env) !== envText(base.env)) patch.env = parsed.env;
    if (strengths.trim() !== (base.strengths ?? "")) patch.strengths = strengths.trim();
    setNote("");
    const next = await buddyConfigure(v, patch, v.state !== "online");
    if (!next) return;
    setBase(next.config);
    setDraft(emptyDraft(next.config.secrets));
    setNote(c.detail.saved);
  };

  return (
    <Section title={c.detail.config} testId="buddy-config">
      <form
        className="space-y-4"
        onSubmit={(e) => {
          e.preventDefault();
          void save();
        }}
        onChange={() => setNote("")}
      >
        <label className="flex items-center justify-between gap-3 text-ui-body text-foreground">
          <span>{c.detail.enabled}</span>
          <Switch checked={enabled} onCheckedChange={(on) => { setEnabled(on); setNote(""); }} aria-label={c.detail.enabled} />
        </label>
        <div className="grid gap-4 sm:grid-cols-3">
          <div className={FIELD}>
            <label htmlFor="buddy-model" className={FIELD_LABEL}>{c.detail.model}</label>
            <ModelField id="buddy-model" v={v} value={model} onChange={(m) => { setModel(m); setNote(""); }} c={c} />
            <p className="text-ui-caption text-muted">{c.detail.modelHint}</p>
          </div>
          <div className={FIELD}>
            <label htmlFor="buddy-approval" className={FIELD_LABEL}>{c.detail.approval}</label>
            <ApprovalSelect id="buddy-approval" value={approval} onChange={(a) => { setApproval(a); setNote(""); }} c={c} />
          </div>
          <div className={FIELD}>
            <label htmlFor="buddy-parallel" className={FIELD_LABEL}>{c.detail.maxParallel}</label>
            <Select value={String(maxParallel)} onValueChange={(n) => { setMaxParallel(Number(n)); setNote(""); }}>
              <SelectTrigger id="buddy-parallel" className="h-8 w-full" aria-label={c.detail.maxParallel}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {[1, 2, 3, 4].map((n) => (
                  <SelectItem key={n} value={String(n)}>
                    {n}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        {secrets.length ? (
          <div className="space-y-3">
            <p className="text-ui-footnote font-medium text-foreground">{c.detail.keys}</p>
            {secrets.map((s) => (
              <SecretRow key={s.name} s={s} draft={draft} onDraft={(d) => { setDraft(d); setNote(""); }} c={c} />
            ))}
          </div>
        ) : null}
        <div className="grid gap-4 sm:grid-cols-2">
          <div className={FIELD}>
            <label htmlFor="buddy-args" className={FIELD_LABEL}>{c.detail.args}</label>
            <Textarea id="buddy-args" className={AREA} value={args} placeholder={c.detail.argsHint} onChange={(e) => setArgs(e.target.value)} spellCheck={false} />
          </div>
          <div className={FIELD}>
            <label htmlFor="buddy-env" className={FIELD_LABEL}>{c.detail.env}</label>
            <Textarea id="buddy-env" className={AREA} value={env} placeholder="NAME=value" aria-describedby="buddy-env-hint" onChange={(e) => setEnv(e.target.value)} spellCheck={false} />
            <p id="buddy-env-hint" className="text-ui-caption text-muted">{c.detail.envHint}</p>
          </div>
        </div>
        <div className={FIELD}>
          <label htmlFor="buddy-strengths" className={FIELD_LABEL}>{c.detail.strengths}</label>
          <Input id="buddy-strengths" value={strengths} maxLength={200} placeholder={c.detail.strengthsHint} onChange={(e) => setStrengths(e.target.value)} />
        </div>
        <div className="flex items-center gap-3">
          <Button type="submit" size="sm" disabled={!!pending} aria-busy={pending === "configure" || undefined}>
            {pending === "configure" ? <Loader2 className="animate-spin" aria-hidden /> : null}
            {c.detail.save}
          </Button>
          {note ? (
            <span role="status" className={cn("text-ui-footnote", note === c.detail.saved ? "text-status-done" : "text-danger")}>
              {note}
            </span>
          ) : null}
        </div>
      </form>
    </Section>
  );
}

/** BuddyDetail 是一个 agent 的详情页。内置的 yoyo 只有头部和一句说明；没装的显示介绍和安装按钮。 */
export function BuddyDetail({ v }: { v: BuddyView }) {
  const copy = useCopy();
  const c = copy.buddy;
  const setFocus = useUI((s) => s.setBuddyFocus);
  const tone = buddyTone(v);
  const status = buddyStatusText(v, c);
  const has = installed(v);
  const bar = progressBar(v);
  const checked = has ? checkedAgo(v.checkedAt, Date.now(), getLocale()) : "";
  return (
    <div className="mx-auto w-full max-w-3xl px-6 py-6" data-testid="buddy-detail">
      <button
        type="button"
        className="mb-4 inline-flex cursor-pointer items-center gap-1 rounded-md text-ui-footnote text-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/40"
        onClick={() => setFocus("")}
      >
        <ChevronLeft className="size-3.5" aria-hidden />
        {c.detail.back}
      </button>
      <header className="flex items-start gap-4 pb-5">
        <BuddyMark v={v} className="size-10 text-ui-title-s text-foreground/85" />
        <div className="min-w-0 flex-1">
          <h2 className="truncate text-ui-title font-semibold tracking-[-0.02em] text-foreground">{v.name}</h2>
          <p className="mt-0.5 flex flex-wrap items-center gap-x-2 text-ui-footnote text-muted">
            {v.vendor ? <span>{v.vendor}</span> : null}
            {v.version ? <span className="tabular-nums">{fill(c.detail.version, { v: v.version })}</span> : null}
          </p>
          {has ? (
            <p className="mt-1.5 flex min-w-0 items-center gap-1.5 text-ui-caption text-muted">
              <span className="shrink-0">{sourceLabel(v, c)}</span>
              {v.install.path ? (
                <>
                  <span aria-hidden>·</span>
                  <code className="min-w-0 truncate font-mono" title={v.install.path}>{v.install.path}</code>
                  <CopyPath path={v.install.path} c={c} />
                </>
              ) : null}
              {v.install.unverified ? <Badge className="text-status-needs">{c.unverified}</Badge> : null}
            </p>
          ) : null}
          {v.description ? <p className="mt-2 max-w-[60ch] text-ui-body text-muted">{v.description}</p> : null}
        </div>
        <div className="flex shrink-0 items-center gap-1">
          {buddyPrimary(v) === "test" ? null : <BuddyPrimary v={v} size="default" />}
          <BuddyMoreMenu v={v} />
        </div>
      </header>
      <Section title={c.detail.status} testId="buddy-status">
        <div className="space-y-2">
          <p className="flex min-w-0 items-center gap-2 text-ui-body text-foreground">
            {v.state === "unsupported" ? null : <StatusDot tone={tone.tone} still={tone.still} />}
            <span className="min-w-0 break-words" data-testid="buddy-detail-status">{status}</span>
          </p>
          {v.state === "installing" ? <ProgressHairline className="max-w-md" value={bar.value} indeterminate={bar.indeterminate} /> : null}
          {v.state === "installing" && v.progress?.line ? <p className="truncate font-mono text-ui-caption text-muted">{v.progress.line}</p> : null}
          {checked ? <p className="text-ui-footnote text-muted">{fill(c.detail.checked, { t: checked })}</p> : null}
          {has && v.state !== "installing" ? (
            <Button size="sm" variant="outline" className="mt-1" onClick={() => void buddyDo(v, "test")}>
              {c.action.test}
            </Button>
          ) : null}
          <Tasks v={v} />
        </div>
      </Section>
      {has ? <Caps v={v} c={c} /> : null}
      {has ? <Login v={v} c={c} /> : null}
      {has ? <ConfigForm key={v.id} v={v} c={c} /> : null}
      {has ? (
        <Section title={c.detail.danger}>
          <p className="mb-3 text-ui-footnote text-muted">{v.custom ? c.confirm.removeBody : c.detail.dangerNote}</p>
          <Button size="sm" variant="danger" onClick={() => void buddyDo(v, v.custom ? "remove" : "uninstall")}>
            {v.custom ? c.action.remove : c.action.uninstall}
          </Button>
        </Section>
      ) : null}
    </div>
  );
}
