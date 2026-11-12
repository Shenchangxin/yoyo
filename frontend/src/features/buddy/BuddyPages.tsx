/**
 * 拥有：今天栏里 Buddy 的三张次级页：偏好（下载源、默认 agent、单轮时长、磁盘占用）、添加自定义 agent、一个 agent 的日志。
 * 都是一栏宽的页面，左上角返回，不弹对话框也不滑侧板。
 * 不拥有：取值校验（后端 buddy.prefs 会拒绝不合格的地址和时长）和日志打码（后端写日志时已经打过）。
 */

import { useCallback, useEffect, useState, type ReactNode } from "react";
import { RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import { buddy, type BuddyDisk, type BuddyPrefs, type BuddyView } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { DAY_INPUT, DayArea, DayBack, DayField, DayInput, DayPill, DayRound, DayRow, DaySwitch, DayTitle } from "../superagent/day-ui";
import { BuddyCard, BuddyNote } from "./BuddyParts";
import { canDispatch, fill, formatSize, installed, parseEnv, splitLines, type BuddyCopy } from "./buddy-model";
import { errText, useBuddy } from "./buddy-store";

const NPM_PRESETS = ["auto", "npmjs", "npmmirror"];

/** Head 是次级页的头：返回键、标题和一句说明。 */
function Head(props: { title: string; body?: string; back: string; onBack: () => void; action?: ReactNode }) {
  return (
    <>
      <div className="flex items-center justify-between gap-2">
        <DayBack label={props.back} onClick={props.onBack} testId="buddy-back" />
        {props.action}
      </div>
      <DayTitle className="mt-3 px-1">{props.title}</DayTitle>
      {props.body ? <p className="mt-1.5 px-1 text-ui-footnote leading-5 text-muted">{props.body}</p> : null}
    </>
  );
}

/** Disk 是磁盘占用一张卡：总量、运行时和缓存，以及清理缓存。打开这一页时才去量，因为要把目录走一遍。 */
function Disk({ c }: { c: BuddyCopy }) {
  const [disk, setDisk] = useState<BuddyDisk | null>(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    let alive = true;
    void buddy.disk().then((d) => alive && setDisk(d)).catch(() => {});
    return () => {
      alive = false;
    };
  }, []);
  return (
    <BuddyCard title={c.pref.disk}>
      {disk ? (
        <>
          <p className="text-ui-body-l tabular-nums">{fill(c.pref.diskTotal, { size: formatSize(disk.total) })}</p>
          <dl className="mt-2 grid grid-cols-[1fr_auto] gap-x-4 gap-y-1 text-ui-footnote text-muted">
            <dt>{c.pref.diskRuntime}</dt>
            <dd className="tabular-nums">{formatSize(disk.runtime)}</dd>
            <dt>{c.pref.diskCache}</dt>
            <dd className="tabular-nums">{formatSize(disk.cache)}</dd>
          </dl>
        </>
      ) : (
        <div className="h-12 animate-pulse rounded-lg bg-lift/70" />
      )}
      <div className="mt-4 flex justify-end">
        <DayPill
          tone="soft"
          disabled={busy || !disk?.cache}
          onClick={() => {
            setBusy(true);
            void buddy
              .clearCache()
              .then((res) => {
                toast(fill(c.toast.cleared, { size: formatSize(res.freed) }));
                return buddy.disk().then(setDisk);
              })
              .catch((e) => toast.error(errText(e)))
              .finally(() => setBusy(false));
          }}
        >
          {c.pref.clearCache}
        </DayPill>
      </div>
    </BuddyCard>
  );
}

/** PrefsForm 是偏好表单：下载源一张卡，派任务一张卡。初值取自打开时的偏好，保存后写回仓库。 */
function PrefsForm({ prefs, c }: { prefs: BuddyPrefs; c: BuddyCopy }) {
  const list = useBuddy((s) => s.list);
  const setPrefs = useBuddy((s) => s.setPrefs);
  const custom = !NPM_PRESETS.includes(prefs.npmRegistry);
  const [npm, setNpm] = useState(custom ? "custom" : prefs.npmRegistry || "auto");
  const [npmURL, setNpmURL] = useState(custom ? prefs.npmRegistry : "");
  const [py, setPy] = useState(prefs.pyIndex);
  const [agent, setAgent] = useState(prefs.defaultAgent || "yoyo");
  const [autoPick, setAutoPick] = useState(prefs.autoPick);
  const [minutes, setMinutes] = useState(String(prefs.turnMinutes));
  const [note, setNote] = useState("");
  const [saving, setSaving] = useState(false);
  const agents = list.filter((v) => v.builtin || (installed(v) && v.config.enabled));
  const pick = agents.some((v) => v.id === agent) ? agent : "yoyo";
  return (
    <form
      onChange={() => setNote("")}
      onSubmit={(e) => {
        e.preventDefault();
        const patch: Partial<BuddyPrefs> = {};
        const registry = npm === "custom" ? npmURL.trim() : npm;
        if (registry !== prefs.npmRegistry) patch.npmRegistry = registry;
        if (py.trim() !== prefs.pyIndex) patch.pyIndex = py.trim();
        if (pick !== prefs.defaultAgent) patch.defaultAgent = pick;
        if (autoPick !== prefs.autoPick) patch.autoPick = autoPick;
        const n = Number.parseInt(minutes, 10);
        if (Number.isFinite(n) && n !== prefs.turnMinutes) patch.turnMinutes = n;
        setSaving(true);
        setNote("");
        void buddy
          .prefs(patch)
          .then((next) => {
            setPrefs(next);
            setNote(c.pref.saved);
          })
          .catch((err) => setNote(errText(err)))
          .finally(() => setSaving(false));
      }}
    >
      <BuddyCard title={c.pref.dispatch}>
        <div className="space-y-4">
          <DayField label={c.pref.defaultAgent} htmlFor="buddy-default" hint={c.pref.defaultAgentHint}>
            <Select value={pick} onValueChange={(v) => { setAgent(v); setNote(""); }}>
              <SelectTrigger id="buddy-default" className={DAY_INPUT} aria-label={c.pref.defaultAgent}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {agents.map((v) => (
                  <SelectItem key={v.id} value={v.id}>
                    {v.name}
                    {canDispatch(v) ? "" : ` · ${c.state[v.state]}`}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </DayField>
          <DayRow title={c.pref.autoPick}>
            <DaySwitch on={autoPick} label={c.pref.autoPick} onChange={(on) => { setAutoPick(on); setNote(""); }} />
          </DayRow>
          <DayField label={c.pref.turnMinutes} htmlFor="buddy-minutes">
            <DayInput id="buddy-minutes" type="number" min={5} max={600} step={5} className="w-32 tabular-nums" value={minutes} onChange={(e) => setMinutes(e.target.value)} />
          </DayField>
        </div>
      </BuddyCard>
      <BuddyCard title={c.pref.download}>
        <div className="space-y-4">
          <DayField label={c.pref.npm} htmlFor="buddy-npm">
            <Select value={npm} onValueChange={(v) => { setNpm(v); setNote(""); }}>
              <SelectTrigger id="buddy-npm" className={DAY_INPUT} aria-label={c.pref.npm}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="auto">{c.pref.npmAuto}</SelectItem>
                <SelectItem value="npmjs">{c.pref.npmOfficial}</SelectItem>
                <SelectItem value="npmmirror">{c.pref.npmMirror}</SelectItem>
                <SelectItem value="custom">{c.pref.npmCustom}</SelectItem>
              </SelectContent>
            </Select>
            {npm === "custom" ? (
              <DayInput aria-label={c.pref.npmCustom} value={npmURL} placeholder={c.pref.npmCustomHint} onChange={(e) => setNpmURL(e.target.value)} inputMode="url" />
            ) : null}
          </DayField>
          <DayField label={c.pref.py} htmlFor="buddy-py">
            <DayInput id="buddy-py" value={py} placeholder={c.pref.pyHint} onChange={(e) => setPy(e.target.value)} inputMode="url" />
          </DayField>
        </div>
      </BuddyCard>
      <div className="mt-4 flex items-center justify-end gap-3 px-1">
        <BuddyNote text={note} ok={note === c.pref.saved} />
        <DayPill type="submit" busy={saving}>
          {c.pref.save}
        </DayPill>
      </div>
    </form>
  );
}

/** BuddyPrefsPage 是 Buddy 偏好页。 */
export function BuddyPrefsPage({ onBack }: { onBack: () => void }) {
  const c = useCopy().buddy;
  const prefs = useBuddy((s) => s.prefs);
  return (
    <div className="px-1 pb-8 pt-2" data-testid="buddy-prefs">
      <Head title={c.pref.title} body={c.pref.body} back={c.detail.back} onBack={onBack} />
      {prefs ? <PrefsForm prefs={prefs} c={c} /> : null}
      <Disk c={c} />
    </div>
  );
}

/** BuddyCustomPage 是添加自定义 agent 的页：名字、命令、每行一个的参数和环境变量。加好后翻到它的详情，并等探测结果。 */
export function BuddyCustomPage({ onBack }: { onBack: () => void }) {
  const c = useCopy().buddy;
  const setFocus = useUI((s) => s.setBuddyFocus);
  const [name, setName] = useState("");
  const [command, setCommand] = useState("");
  const [args, setArgs] = useState("");
  const [env, setEnv] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  return (
    <div className="px-1 pb-8 pt-2" data-testid="buddy-custom">
      <Head title={c.custom.title} body={c.custom.body} back={c.detail.back} onBack={onBack} />
      <BuddyCard>
        <form
          className="space-y-4"
          aria-label={c.custom.title}
          onSubmit={(e) => {
            e.preventDefault();
            const parsed = parseEnv(env);
            if (parsed.bad) {
              setNote(fill(c.detail.envBad, { line: parsed.bad }));
              return;
            }
            setBusy(true);
            setNote("");
            void buddy
              .add({ name: name.trim(), command: command.trim(), args: splitLines(args), env: parsed.env })
              .then((v) => {
                const s = useBuddy.getState();
                s.put(v);
                s.watch(v);
                toast(fill(c.toast.added, { name: v.name }));
                onBack();
                setFocus(v.id);
              })
              .catch((err) => setNote(errText(err)))
              .finally(() => setBusy(false));
          }}
        >
          <DayField label={c.custom.name} htmlFor="buddy-custom-name">
            <DayInput id="buddy-custom-name" value={name} onChange={(e) => setName(e.target.value)} required maxLength={60} />
          </DayField>
          <DayField label={c.custom.command} htmlFor="buddy-custom-command">
            <DayInput id="buddy-custom-command" mono value={command} onChange={(e) => setCommand(e.target.value)} required placeholder={c.custom.commandHint} />
          </DayField>
          <DayField label={c.custom.args} htmlFor="buddy-custom-args">
            <DayArea id="buddy-custom-args" value={args} placeholder={c.custom.argsHint} onChange={(e) => setArgs(e.target.value)} />
          </DayField>
          <DayField label={c.custom.env} htmlFor="buddy-custom-env" hint={c.custom.envHint}>
            <DayArea id="buddy-custom-env" value={env} placeholder="NAME=value" onChange={(e) => setEnv(e.target.value)} />
          </DayField>
          <div className="flex items-center justify-end gap-3">
            <BuddyNote text={note} />
            <DayPill type="submit" busy={busy} disabled={!name.trim() || !command.trim()}>
              {c.custom.add}
            </DayPill>
          </div>
        </form>
      </BuddyCard>
    </div>
  );
}

/** BuddyLogPage 是一个 agent 的日志页：读安装和探测日志的末尾，可刷新；正在安装时每两秒自动刷新。 */
export function BuddyLogPage({ v, onBack }: { v: BuddyView; onBack: () => void }) {
  const c = useCopy().buddy;
  const [text, setText] = useState<string | null>(null);
  const installing = v.state === "installing";
  const load = useCallback(() => {
    void buddy
      .logTail(v.id, 400)
      .then((res) => setText(res?.text ?? ""))
      .catch((e) => setText(errText(e)));
  }, [v.id]);
  useEffect(() => {
    load();
    if (!installing) return;
    const timer = window.setInterval(load, 2000);
    return () => window.clearInterval(timer);
  }, [load, installing]);
  return (
    <div className="px-1 pb-8 pt-2" data-testid="buddy-log">
      <Head
        title={fill(c.log.title, { name: v.name })}
        body={c.log.body}
        back={c.detail.back}
        onBack={onBack}
        action={
          <DayRound label={c.log.refresh} onClick={load}>
            <RefreshCw className="size-3.5" aria-hidden />
          </DayRound>
        }
      />
      <div className="day-tile mt-4 px-4 py-3.5">
        {text === null ? (
          <div className="h-24 animate-pulse rounded-lg bg-lift/70" />
        ) : text.trim() ? (
          <pre className="whitespace-pre-wrap break-words font-mono text-ui-caption leading-5 text-foreground/85" data-testid="buddy-log-text">
            {text}
          </pre>
        ) : (
          <p className="text-ui-footnote text-muted">{c.log.empty}</p>
        )}
      </div>
    </div>
  );
}
