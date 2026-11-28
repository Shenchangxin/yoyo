/**
 * 拥有：设置里的「主动与通知」页：简报几点写、主动程度和安静时段、怎么告诉你、来信要不要自己接成任务。
 * 不拥有：开口门和编排本身，也不拥有它自己在跑的定时和目标（那在「自动化」页）。改动经 host.patch、开口门反馈和编排模式落到 Go。
 */
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { deskState, imHome, imPair, imTest, imUnbind, orchestratorMode, type DeskLead, type DeskMute, type DeskTally, type IMHomeState } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { applyProactiveLevel, briefDeskOf, briefTimesOf, cacheBriefTimes, parseProactiveLevel, type BriefTimes, type ProactiveLevel } from "../../lib/proactive";
import type { DeskConfig } from "../../lib/protocol";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Switch } from "../../components/ui/switch";
import { DESK_EVENT, deskMutes, refreshDesk, sendFeedback } from "../superagent/desk-feed";
import { refreshNeeds, useNeeds } from "../superagent/needs";
import { SettingEmpty, SettingRow, SettingSection, SettingSegmented } from "./SettingChrome";
import type { SettingsHost } from "./host";

/** ProactiveSettings 是「主动与通知」整页。 */
export function ProactiveSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const s = copy.settings;
  const p = s.proactive;
  const desk: DeskConfig = host.cfg.desk || {};
  const times = briefTimesOf(desk);
  const level = parseProactiveLevel(desk.level);
  const notifications = host.cfg.notificationsEnabled !== false;
  function saveDesk(next: Partial<DeskConfig>) {
    void host.patch({ desk: { ...desk, ...next } });
  }
  function saveTimes(next: BriefTimes) {
    cacheBriefTimes(next);
    saveDesk(briefDeskOf(next));
  }
  return (
    <>
      <SettingSection id="proactive-brief" title={s.sections.proactiveBrief} description={p.briefDesc}>
        <BriefRow
          label={p.morning}
          on={times.morningOn}
          at={times.morningAt}
          onChange={(on, at) => saveTimes({ ...times, morningOn: on, morningAt: at })}
        />
        <BriefRow
          label={p.evening}
          on={times.eveningOn}
          at={times.eveningAt}
          border={false}
          onChange={(on, at) => saveTimes({ ...times, eveningOn: on, eveningAt: at })}
        />
      </SettingSection>

      <SettingSection id="proactive-level" title={s.sections.proactiveLevel} description={p.levelDesc}>
        <SettingRow title={p.level}>
          <SettingSegmented<ProactiveLevel>
            id="proactive-level"
            ariaLabel={p.level}
            value={level}
            options={[
              { value: "less", label: p.levelLess },
              { value: "normal", label: p.levelNormal },
              { value: "more", label: p.levelMore },
            ]}
            onChange={(next) => {
              void applyProactiveLevel(next).then((ok) => { if (!ok) toast.error(p.levelFailed); });
              saveDesk({ level: next });
            }}
          />
        </SettingRow>
        <SettingRow title={p.quiet} description={p.quietDesc}>
          <QuietHours start={desk.quietStart || ""} end={desk.quietEnd || ""} onChange={(quietStart, quietEnd) => saveDesk({ quietStart, quietEnd })} />
        </SettingRow>
        <SettingRow title={p.weekend} description={p.weekendDesc}>
          <Switch checked={!!desk.quietWeekend} aria-label={p.weekend} onCheckedChange={(on) => saveDesk({ quietWeekend: on })} />
        </SettingRow>
        <HeldClasses />
        <LeadRow />
        <TallyRow />
      </SettingSection>

      <SettingSection id="proactive-notify" title={s.sections.proactiveNotify} description={p.notifyDesc}>
        <SettingRow title={s.notifications} description={s.notificationsDesc} border={false}>
          <Switch checked={notifications} aria-label={s.notifications} onCheckedChange={(on) => void host.patch({ notificationsEnabled: on })} />
        </SettingRow>
      </SettingSection>

      <HomeSection />

      <IntakeSection />
    </>
  );
}

function fillCopy(text: string, vars: Record<string, string | number>): string {
  return text.replace(/\{(\w+)\}/g, (_, key: string) => String(vars[key] ?? ""));
}

/** LeadRow 是这台电脑上谁在跑主动链路：桌面开着时是桌面，关掉之后是 yoyo daemon。不选主（只开着一个旧式进程）时不显示。 */
function LeadRow() {
  const p = useCopy().settings.proactive;
  const [lead, setLead] = useState<DeskLead | null>(null);
  useEffect(() => {
    let alive = true;
    const pull = () => { void deskState().then((state) => { if (alive) setLead(state.lead); }).catch(() => {}); };
    pull();
    const timer = window.setInterval(pull, 10000);
    return () => { alive = false; window.clearInterval(timer); };
  }, []);
  if (!lead || !lead.role) return null;
  // 收尾中（领头的已经决定让出、等手上的回合跑完）优先显示，免得人以为桌面没接上。
  const text = lead.draining
    ? (lead.role === "lead" ? p.leadDrainSelf : p.leadDrainOther)
    : lead.role === "lead" ? p.leadSelf : lead.holder === "desktop" ? p.leadDesktop : lead.holder ? p.leadDaemon : p.leadNone;
  return (
    <SettingRow title={p.lead} description={p.leadDesc}>
      <span className="text-ui-footnote text-muted" data-testid="proactive-lead">{text}</span>
    </SettingRow>
  );
}

/** TallyRow 是最近七天主动开口了几次、用上了几次。还没说过话时写一句。 */
function TallyRow() {
  const p = useCopy().settings.proactive;
  const [days, setDays] = useState<DeskTally[]>([]);
  const [rate, setRate] = useState(-1);
  useEffect(() => {
    let alive = true;
    const pull = () => {
      void deskState().then((state) => {
        if (!alive) return;
        setDays(state.tally);
        setRate(state.accepted);
      }).catch(() => {});
    };
    pull();
    window.addEventListener(DESK_EVENT, pull);
    return () => { alive = false; window.removeEventListener(DESK_EVENT, pull); };
  }, []);
  const spoken = days.reduce((n, d) => n + d.spoken, 0);
  const used = days.reduce((n, d) => n + d.used, 0);
  return (
    <SettingRow title={p.tally} description={p.tallyDesc} border={false}>
      {spoken > 0 ? (
        <div className="text-right text-ui-footnote tabular-nums" data-testid="proactive-tally">
          <div>{fillCopy(p.tallyLine, { spoken, used })}</div>
          {rate >= 0 ? <div className="text-muted">{fillCopy(p.tallyRate, { rate: Math.round(rate * 100) })}</div> : null}
        </div>
      ) : (
        <span className="text-ui-footnote text-muted">{p.tallyEmpty}</span>
      )}
    </SettingRow>
  );
}

/** HomeSection 是 IM 主通道：生成配对码、看绑在哪、发测试、解除。配对码有效时每三秒看一次绑上没有。 */
function HomeSection() {
  const copy = useCopy();
  const p = copy.settings.proactive;
  const [state, setState] = useState<IMHomeState | null>(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    let alive = true;
    const pull = () => { void imHome().then((next) => { if (alive) setState(next); }).catch(() => {}); };
    pull();
    const timer = window.setInterval(pull, state?.code && !state.bound ? 3000 : 30000);
    return () => { alive = false; window.clearInterval(timer); };
  }, [state?.code, state?.bound]);
  async function run(fn: () => Promise<unknown>, ok?: string) {
    if (busy) return;
    setBusy(true);
    try {
      await fn();
      if (ok) toast.success(ok);
      setState(await imHome());
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <SettingSection id="proactive-home" title={copy.settings.sections.proactiveHome} description={p.homeDesc}>
      {state?.bound ? (
        <>
          <SettingRow
            title={fillCopy(p.homeBound, { label: state.label || state.platform })}
            description={[state.senderName ? fillCopy(p.homeBoundBy, { name: state.senderName }) : "", state.held ? fillCopy(p.homeHeld, { n: state.held }) : "", state.lastErr ? fillCopy(p.homeLastErr, { err: state.lastErr }) : ""].filter(Boolean).join(" · ") || undefined}
            border={false}
          >
            <div className="flex items-center gap-1.5">
              <Button size="sm" variant="outline" disabled={busy} onClick={() => { void run(imTest, p.homeTestOk); }}>{p.homeTest}</Button>
              <Button size="sm" variant="ghost" disabled={busy} onClick={() => { void run(imUnbind); }}>{p.homeUnbind}</Button>
            </div>
          </SettingRow>
        </>
      ) : (
        <SettingRow title={state?.code ? fillCopy(p.homeCode, { code: state.code }) : p.homeNone} border={false}>
          <div className="flex items-center gap-2">
            {state?.code ? <code className="rounded-md border border-border px-2 py-1 text-ui-body tabular-nums" data-testid="im-pair-code">{state.code}</code> : null}
            <Button size="sm" variant="outline" disabled={busy} onClick={() => { void run(imPair); }}>{p.homePair}</Button>
          </div>
        </SettingRow>
      )}
    </SettingSection>
  );
}

/** BriefRow 是一档简报：开关和几点写。 */
function BriefRow(props: { label: string; on: boolean; at: string; border?: boolean; onChange: (on: boolean, at: string) => void }) {
  const copy = useCopy();
  const [at, setAt] = useState(props.at);
  useEffect(() => setAt(props.at), [props.at]);
  return (
    <SettingRow title={props.label} border={props.border}>
      <div className="flex items-center gap-2">
        <Input
          type="time"
          aria-label={`${props.label} ${copy.settings.proactive.at}`}
          className="h-8 w-[7.5rem] text-ui-body tabular-nums"
          value={at}
          disabled={!props.on}
          onChange={(event) => setAt(event.target.value)}
          onBlur={() => { if (at && at !== props.at) props.onChange(props.on, at); }}
        />
        <Switch checked={props.on} aria-label={props.label} onCheckedChange={(on) => props.onChange(on, at || props.at)} />
      </div>
    </SettingRow>
  );
}

/** QuietHours 是两段时间。两段都填时生效，清空任一段即不设。 */
function QuietHours(props: { start: string; end: string; onChange: (start: string, end: string) => void }) {
  const p = useCopy().settings.proactive;
  const [start, setStart] = useState(props.start);
  const [end, setEnd] = useState(props.end);
  useEffect(() => { setStart(props.start); setEnd(props.end); }, [props.start, props.end]);
  function commit(nextStart: string, nextEnd: string) {
    if (nextStart === props.start && nextEnd === props.end) return;
    props.onChange(nextStart, nextEnd);
  }
  return (
    <div className="flex items-center gap-1.5 text-ui-footnote text-muted">
      <Input type="time" aria-label={p.quietFrom} className="h-8 w-[7.5rem] text-ui-body tabular-nums" value={start} onChange={(e) => setStart(e.target.value)} onBlur={() => commit(start, end)} />
      <span>–</span>
      <Input type="time" aria-label={p.quietTo} className="h-8 w-[7.5rem] text-ui-body tabular-nums" value={end} onChange={(e) => setEnd(e.target.value)} onBlur={() => commit(start, end)} />
      {props.start || props.end ? (
        <Button size="sm" variant="ghost" onClick={() => { setStart(""); setEnd(""); props.onChange("", ""); }}>{p.quietClear}</Button>
      ) : null}
    </div>
  );
}

/** HeldClasses 列出现在按住、不主动开口的类别，可以逐个放开。公开渠道单独一个开关。 */
function HeldClasses() {
  const copy = useCopy();
  const w = copy.work;
  const p = copy.settings.proactive;
  const [mutes, setMutes] = useState<DeskMute[]>(() => deskMutes());
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    const pull = () => setMutes(deskMutes());
    window.addEventListener(DESK_EVENT, pull);
    void refreshDesk();
    return () => window.removeEventListener(DESK_EVENT, pull);
  }, []);
  const labels: Record<string, string> = {
    calendar: w.classCalendar,
    source: w.classSource,
    line: w.classLine,
    git: w.classGit,
    power: w.classPower,
    net: w.classNet,
    reach: w.classReach,
    poi: w.classPoi,
  };
  const reachOff = mutes.some((row) => row.class === "reach" && row.until === "9999-12-31");
  const held = mutes.filter((row) => row.class !== "reach" || !reachOff);
  async function send(className: string, kind: "mute" | "unmute") {
    if (busy) return;
    setBusy(true);
    try {
      await sendFeedback(className, kind);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <SettingRow title={w.reachSwitch} description={w.reachSwitchHint}>
        <Switch checked={!reachOff} disabled={busy} aria-label={w.reachSwitch} onCheckedChange={() => { void send("reach", reachOff ? "unmute" : "mute"); }} />
      </SettingRow>
      <SettingRow title={p.held} description={p.heldDesc} stack border={false}>
        {held.length ? (
          <div className="flex w-full flex-wrap gap-1.5" data-testid="held-classes">
            {held.map((row) => (
              <span key={row.class} className="inline-flex h-7 items-center gap-1 rounded-md border border-border bg-background pl-2 pr-0.5 text-ui-footnote">
                <span>{labels[row.class] || row.class}</span>
                <span className="text-muted">· {row.until === "9999-12-31" ? w.quietOff : w.quietToday}</span>
                <Button size="xs" variant="ghost" disabled={busy} onClick={() => { void send(row.class, "unmute"); }}>{w.quietOn}</Button>
              </span>
            ))}
          </div>
        ) : (
          <SettingEmpty compact>{w.quietEmpty}</SettingEmpty>
        )}
      </SettingRow>
    </>
  );
}

/** IntakeSection 决定来信和请求要不要自己接成任务：不接、只观察、自动接。 */
function IntakeSection() {
  const copy = useCopy();
  const p = copy.settings.proactive;
  const mode = useNeeds((s) => s.view?.mode || "");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    if (!useNeeds.getState().view) void refreshNeeds();
  }, []);
  const value = mode === "off" || mode === "auto" ? mode : "observe";
  const hints: Record<string, string> = { off: p.intakeOffHint, observe: p.intakeObserveHint, auto: p.intakeAutoHint };
  return (
    <SettingSection id="proactive-intake" title={copy.settings.sections.proactiveIntake} description={p.intakeDesc}>
      <SettingRow title={p.intake} description={hints[value]} border={false}>
        <SettingSegmented
          id="proactive-intake"
          ariaLabel={p.intake}
          value={value}
          options={[
            { value: "off", label: p.intakeOff },
            { value: "observe", label: p.intakeObserve },
            { value: "auto", label: p.intakeAuto },
          ]}
          onChange={(next) => {
            if (busy || next === value) return;
            setBusy(true);
            void orchestratorMode(next)
              .then(() => refreshNeeds())
              .catch((e) => toast.error(e instanceof Error ? e.message : String(e)))
              .finally(() => setBusy(false));
          }}
        />
      </SettingRow>
    </SettingSection>
  );
}
