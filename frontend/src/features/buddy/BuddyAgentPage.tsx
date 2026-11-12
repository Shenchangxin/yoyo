/**
 * 拥有：今天栏里一个 agent 的详情页：概况（名字、状态、主按钮）、它手上的工单、要填的配置、登录、派任务的设置，
 * 以及它是从哪来的、怎么卸载。一栏宽，从上到下一张张卡片，最要紧的在最上面。
 * 不拥有：按钮背后的动作（buddy-actions.ts）。密钥只显示「已保存」，保存后输入框清空。
 */

import { useState } from "react";
import { Copy as CopyIcon, Loader2 } from "lucide-react";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import { StatusDot } from "../../components/ui/status-dot";
import { buddy, type BuddyConfigPatch, type BuddyView } from "../../lib/client";
import { writeClipboard } from "../../lib/clipboard";
import { getLocale, useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { ticketAgent } from "../superagent/agent-model";
import { DAY_INPUT, DayArea, DayBack, DayField, DayInput, DayMore, DayPill, DayRound, DayRow, DaySwitch, DayText } from "../superagent/day-ui";
import { useNeeds } from "../superagent/needs";
import { taskStatus } from "../superagent/TaskPage";
import { AgentAvatar, ApprovalField, BuddyCard, BuddyConfirms, BuddyMore, BuddyNote, BuddyPrimary, KeyField, ModelField } from "./BuddyParts";
import { buddyConfigure, buddyDo, buddyLogin } from "./buddy-actions";
import { draftPatch, emptyDraft, type KeyDraft } from "./buddy-fields";
import { buddyStatusText, buddyTone, checkedAgo, envText, fill, foundOnHost, installed, parseEnv, progressBar, setupSecrets, splitLines, type BuddyCopy } from "./buddy-model";
import { useBuddy } from "./buddy-store";

/** sourceLabel 是来源的说法：托管、系统或自定义命令。 */
function sourceLabel(v: BuddyView, c: BuddyCopy): string {
  if (v.install.kind === "managed") return c.detail.sourceManaged;
  if (v.install.kind === "custom") return c.detail.sourceCustom;
  return c.detail.sourceSystem;
}

/** PathLine 是一行路径：等宽字，放不下时截断，右边可以复制。 */
function PathLine({ path, c }: { path: string; c: BuddyCopy }) {
  const [done, setDone] = useState(false);
  return (
    <div className="flex min-w-0 items-center gap-1">
      <code className="min-w-0 flex-1 truncate font-mono text-ui-footnote text-muted" title={path}>{path}</code>
      <DayRound
        quiet
        className="size-7"
        label={done ? c.detail.copied : c.detail.copyPath}
        onClick={() => {
          void writeClipboard(path).then((ok) => {
            if (!ok) return;
            setDone(true);
            window.setTimeout(() => setDone(false), 1500);
          });
        }}
      >
        <CopyIcon className="size-3.5" aria-hidden />
      </DayRound>
    </div>
  );
}

/** Tasks 列出这一项手上没结束的工单。点一条就把那张工单放到中间的对话里。 */
function Tasks({ v }: { v: BuddyView }) {
  const copy = useCopy();
  const tickets = useNeeds((s) => s.view?.tickets);
  const mine = (tickets ?? []).filter((t) => t.stage !== "ended" && ticketAgent(t) === v.id);
  if (!mine.length) return null;
  return (
    <BuddyCard title={copy.buddy.detail.tasks} testId="buddy-tasks">
      <ul className="-my-1 divide-y divide-border/50">
        {mine.map((t) => (
          <li key={t.id}>
            <button
              type="button"
              className="flex w-full min-w-0 items-center gap-3 py-2 text-left hover:text-foreground"
              onClick={() => useNeeds.getState().setFocusTask(t.id)}
            >
              <span className="min-w-0 flex-1 truncate text-ui-body-l">{t.title || copy.work.taskEmpty}</span>
              <span className="shrink-0 text-ui-footnote text-muted">{taskStatus(t, copy.work)}</span>
            </button>
          </li>
        ))}
      </ul>
    </BuddyCard>
  );
}

/** Login 是登录一张卡。要登录时每种方式一个按钮，终端型的附一句说明；在线时显示已登录，对方支持时可以退出登录。 */
function Login({ v, c }: { v: BuddyView; c: BuddyCopy }) {
  const pending = useBuddy((s) => s.pending[v.id] || "");
  const put = useBuddy((s) => s.put);
  const run = useBuddy((s) => s.run);
  const methods = v.auth?.methods ?? [];
  if (v.state === "needs_auth") {
    return (
      <BuddyCard title={c.detail.login} testId="buddy-login">
        <div className="space-y-3">
          {(methods.length ? methods : [{ id: "", name: c.action.login, type: "terminal" as const, description: "" }]).map((m) => (
            <div key={m.id || "default"}>
              <DayPill busy={pending === "login"} disabled={!!pending} onClick={() => void buddyLogin(v, m.id)}>
                {pending === "login" ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : null}
                {m.name}
              </DayPill>
              {m.description ? <p className="mt-1.5 text-ui-footnote leading-5 text-muted">{m.description}</p> : null}
              {m.type === "terminal" ? <p className="mt-1.5 text-ui-footnote leading-5 text-muted">{c.detail.terminalNote}</p> : null}
            </div>
          ))}
          {v.detail ? <p className="text-ui-footnote leading-5 text-muted" role="status">{v.detail}</p> : null}
        </div>
      </BuddyCard>
    );
  }
  if (v.state !== "online" || !methods.length) return null;
  return (
    <BuddyCard title={c.detail.login}>
      <div className="flex items-center justify-between gap-3">
        <span className="flex items-center gap-1.5 text-ui-body-l">
          <StatusDot tone="done" />
          {c.detail.signedIn}
        </span>
        {v.caps?.logout ? (
          <DayText disabled={!!pending} onClick={() => void run(v.id, "logout", () => buddy.logout(v.id)).then(put)}>
            {c.action.logout}
          </DayText>
        ) : null}
      </div>
    </BuddyCard>
  );
}

/**
 * Setup 是「需要配置」时最上面的那张卡：只列必填的、能从 yoyo 复用的和已经存过的密钥，保存后马上测试连接。
 * 不缺东西时不出现，完整的设置在下面。
 */
function Setup({ v, c }: { v: BuddyView; c: BuddyCopy }) {
  const pending = useBuddy((s) => s.pending[v.id] || "");
  const secrets = setupSecrets(v);
  const [draft, setDraft] = useState<KeyDraft>(() => emptyDraft(secrets));
  return (
    <BuddyCard title={fill(c.setup.title, { name: v.name })} testId="buddy-setup">
      <form
        className="space-y-4"
        aria-label={fill(c.setup.title, { name: v.name })}
        onSubmit={(e) => {
          e.preventDefault();
          void buddyConfigure(v, draftPatch(secrets, draft), true).then((next) => {
            if (next) setDraft(emptyDraft(setupSecrets(next)));
          });
        }}
      >
        <p className="text-ui-body leading-6 text-muted">{secrets.length ? fill(c.setup.body, { name: v.name }) : c.setup.nothing}</p>
        {secrets.map((s) => (
          <KeyField key={s.name} s={s} draft={draft} onDraft={setDraft} c={c} />
        ))}
        <div className="flex justify-end">
          <DayPill type="submit" busy={!!pending}>
            {c.setup.save}
          </DayPill>
        </div>
      </form>
    </BuddyCard>
  );
}

/**
 * Settings 是派任务的设置。初值取自第一次画出时的视图，之后轮询带来的新视图不覆盖人正在改的内容；保存成功后用返回的视图重置。
 * keys 为假时不列密钥（上面的 Setup 卡已经在填了）。参数、环境变量和擅长收在「更多设置」里。
 */
function Settings({ v, c, keys }: { v: BuddyView; c: BuddyCopy; keys: boolean }) {
  const pending = useBuddy((s) => s.pending[v.id] || "");
  const [base, setBase] = useState(v.config);
  const [enabled, setEnabled] = useState(base.enabled);
  const [model, setModel] = useState(base.model ?? "");
  const [approval, setApproval] = useState(base.approval);
  const [maxParallel, setMaxParallel] = useState(base.maxParallel);
  const [hostCli, setHostCli] = useState(base.hostCli?.on ?? true);
  const [args, setArgs] = useState((base.args ?? []).join("\n"));
  const [env, setEnv] = useState(envText(base.env));
  const [strengths, setStrengths] = useState(base.strengths ?? "");
  const [draft, setDraft] = useState<KeyDraft>(() => emptyDraft(v.config.secrets));
  const [note, setNote] = useState("");
  const [more, setMore] = useState(false);
  const secrets = keys ? v.config.secrets : [];
  const host = v.config.hostCli;

  /** save 只把改过的字段发给后端。成功后清空密钥输入，显示「已保存」。 */
  const save = async () => {
    const parsed = parseEnv(env);
    if (parsed.bad) {
      setMore(true);
      setNote(fill(c.detail.envBad, { line: parsed.bad }));
      return;
    }
    const patch: BuddyConfigPatch = { ...draftPatch(secrets, draft) };
    if (enabled !== base.enabled) patch.enabled = enabled;
    if (model !== (base.model ?? "")) patch.model = model;
    if (approval !== base.approval) patch.approval = approval;
    if (maxParallel !== base.maxParallel) patch.maxParallel = maxParallel;
    if (host && hostCli !== host.on) patch.hostCli = hostCli;
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
  const touch = () => setNote("");

  return (
    <BuddyCard title={c.detail.config} testId="buddy-config">
      <form
        className="space-y-4"
        onSubmit={(e) => {
          e.preventDefault();
          void save();
        }}
        onChange={touch}
      >
        <DayRow title={c.detail.enabled}>
          <DaySwitch on={enabled} label={c.detail.enabled} onChange={(on) => { setEnabled(on); touch(); }} />
        </DayRow>
        {host ? (
          <DayRow title={fill(c.detail.hostCli, { name: v.name })} hint={c.detail.hostCliNote}>
            <DaySwitch on={hostCli} label={fill(c.detail.hostCli, { name: v.name })} testId="buddy-host-cli" onChange={(on) => { setHostCli(on); touch(); }} />
          </DayRow>
        ) : null}
        {host ? <PathLine path={host.path} c={c} /> : null}
        <DayField label={c.detail.model} htmlFor="buddy-model" hint={c.detail.modelHint}>
          <ModelField id="buddy-model" v={v} value={model} onChange={(m) => { setModel(m); touch(); }} c={c} />
        </DayField>
        <DayField label={c.detail.approval} htmlFor="buddy-approval">
          <ApprovalField id="buddy-approval" value={approval} onChange={(a) => { setApproval(a); touch(); }} c={c} />
        </DayField>
        <DayField label={c.detail.maxParallel} htmlFor="buddy-parallel">
          <Select value={String(maxParallel)} onValueChange={(n) => { setMaxParallel(Number(n)); touch(); }}>
            <SelectTrigger id="buddy-parallel" className={DAY_INPUT} aria-label={c.detail.maxParallel}>
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
        </DayField>
        {secrets.length ? (
          <div className="space-y-3">
            <p className="px-1 text-ui-footnote font-medium">{c.detail.keys}</p>
            {secrets.map((s) => (
              <KeyField key={s.name} s={s} draft={draft} onDraft={(d) => { setDraft(d); touch(); }} c={c} />
            ))}
          </div>
        ) : null}
        {more ? (
          <>
            <DayField label={c.detail.strengths} htmlFor="buddy-strengths">
              <DayInput id="buddy-strengths" value={strengths} maxLength={200} placeholder={c.detail.strengthsHint} onChange={(e) => setStrengths(e.target.value)} />
            </DayField>
            <DayField label={c.detail.args} htmlFor="buddy-args">
              <DayArea id="buddy-args" value={args} placeholder={c.detail.argsHint} onChange={(e) => setArgs(e.target.value)} />
            </DayField>
            <DayField label={c.detail.env} htmlFor="buddy-env" hint={c.detail.envHint}>
              <DayArea id="buddy-env" value={env} placeholder="NAME=value" onChange={(e) => setEnv(e.target.value)} />
            </DayField>
          </>
        ) : null}
        <DayMore className="mt-0" aria-expanded={more} onClick={() => setMore(!more)}>
          {more ? c.detail.fewerSettings : c.detail.moreSettings}
        </DayMore>
        <div className="flex items-center justify-end gap-3">
          <BuddyNote text={note} ok={note === c.detail.saved} />
          <DayPill type="submit" busy={pending === "configure"} disabled={!!pending}>
            {pending === "configure" ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : null}
            {c.detail.save}
          </DayPill>
        </div>
      </form>
    </BuddyCard>
  );
}

/** About 是「它从哪来」的一张卡：来源和路径、是不是在 PATH 之外找到的、对方声明的能力，以及卸载。 */
function About({ v, c }: { v: BuddyView; c: BuddyCopy }) {
  const caps = v.caps;
  const have = caps ? (["resume", "systemPrompt", "mcpHttp", "image", "dirs"] as const).filter((k) => caps[k]) : [];
  const checked = checkedAgo(v.checkedAt, Date.now(), getLocale());
  return (
    <BuddyCard title={c.detail.about} testId="buddy-about">
      <div className="space-y-3">
        <div>
          <p className="flex flex-wrap items-center gap-x-2 text-ui-body-l">
            <span>{sourceLabel(v, c)}</span>
            {v.install.unverified ? <span className="text-ui-footnote text-status-needs">{c.unverified}</span> : null}
          </p>
          {v.host && v.host.kind === "acp" ? <p className="mt-0.5 text-ui-footnote leading-5 text-muted">{c.detail.hostFound}</p> : null}
          {v.install.path ? <PathLine path={v.install.path} c={c} /> : null}
        </div>
        {have.length ? (
          <div>
            <p className="text-ui-footnote text-muted">{c.detail.caps}</p>
            <div className="mt-1.5 flex flex-wrap gap-1.5">
              {have.map((k) => (
                <span key={k} className="rounded-full bg-lift px-2.5 py-0.5 text-ui-caption text-foreground/85">{c.detail.cap[k]}</span>
              ))}
            </div>
          </div>
        ) : null}
        {checked ? <p className="text-ui-footnote text-muted">{fill(c.detail.checked, { t: checked })}</p> : null}
        <div className="flex items-center justify-between gap-3 border-t border-border/50 pt-3">
          <p className="min-w-0 text-ui-footnote leading-5 text-muted">{v.custom ? c.confirm.removeBody : c.detail.dangerNote}</p>
          <DayPill small tone="danger" onClick={() => void buddyDo(v, v.custom ? "remove" : "uninstall")}>
            {v.custom ? c.action.remove : c.action.uninstall}
          </DayPill>
        </div>
      </div>
    </BuddyCard>
  );
}

/**
 * BuddyAgentPage 是一个 agent 的详情页。内置的 yoyo 只有概况；没装的显示介绍、在这台电脑上找到了什么，和安装或连接的按钮。
 * 确认卡（装到系统、卸载、登录命令）就地出现在概况下面。
 */
export function BuddyAgentPage({ v, onBack }: { v: BuddyView; onBack: () => void }) {
  const c = useCopy().buddy;
  const tone = buddyTone(v);
  const status = buddyStatusText(v, c);
  const has = installed(v);
  const bar = progressBar(v);
  const found = foundOnHost(v);
  const needsSetup = has && v.state === "needs_config";
  return (
    <div className="px-1 pb-8 pt-2" data-testid="buddy-detail">
      <div className="flex items-center justify-between gap-2">
        <DayBack label={c.detail.back} onClick={onBack} testId="buddy-back" />
        <BuddyMore v={v} />
      </div>
      <section className="day-card mt-3 px-5 py-5" aria-label={v.name}>
        <div className="flex items-start gap-3">
          <AgentAvatar v={v} large />
          <div className="min-w-0 flex-1">
            <h1 className="truncate text-ui-title font-semibold tracking-[-0.03em]">{v.name}</h1>
            <p className="mt-0.5 flex flex-wrap items-center gap-x-2 text-ui-footnote text-muted">
              {v.vendor ? <span>{v.vendor}</span> : null}
              {v.version ? <span className="tabular-nums">{fill(c.detail.version, { v: v.version })}</span> : null}
            </p>
          </div>
        </div>
        <p className="mt-4 flex min-w-0 items-start gap-2 text-ui-body-l leading-6" data-testid="buddy-status">
          {v.state === "unsupported" ? null : <StatusDot tone={tone.tone} still={tone.still} className="mt-[9px]" />}
          <span className="min-w-0 break-words" data-testid="buddy-detail-status">{status}</span>
        </p>
        {found && v.host ? (
          <div className="mt-2" data-testid="buddy-host">
            {v.host.kind === "cli" && v.host.app ? <p className="text-ui-footnote leading-5 text-muted">{fill(c.detail.hostFoundApp, { app: v.host.app })}</p> : null}
            <PathLine path={v.host.path} c={c} />
          </div>
        ) : null}
        {v.state === "installing" ? <ProgressHairline className="mt-3" value={bar.value} indeterminate={bar.indeterminate} /> : null}
        {v.state === "installing" && v.progress?.line ? <p className="mt-1.5 truncate font-mono text-ui-caption text-muted">{v.progress.line}</p> : null}
        {v.description && !has ? <p className="mt-3 text-ui-body leading-6 text-muted">{v.description}</p> : null}
        <div className={cn("mt-4 flex flex-wrap items-center gap-2", v.builtin && "hidden")}>
          {/* 要配置时，要填的东西就在下面那张卡里，这里不再放一个「配置」按钮。 */}
          {needsSetup ? null : <BuddyPrimary v={v} hideTest />}
          {has && v.state !== "installing" ? (
            <DayPill tone="soft" onClick={() => void buddyDo(v, "test")}>
              {c.action.test}
            </DayPill>
          ) : null}
        </div>
      </section>
      <BuddyConfirms only={v.id} />
      {needsSetup ? <Setup key={`setup-${v.id}`} v={v} c={c} /> : null}
      {has ? <Login v={v} c={c} /> : null}
      <Tasks v={v} />
      {has ? <Settings key={`settings-${v.id}`} v={v} c={c} keys={!needsSetup} /> : null}
      {has ? <About v={v} c={c} /> : null}
    </div>
  );
}
