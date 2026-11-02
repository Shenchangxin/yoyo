/**
 * 拥有：设置里的 GitHub 和飞书来源。人在这里填写仓库、令牌和可以派工的人。
 * 不拥有：来源怎么轮询。那一段在 superagent。
 */

import { useCallback, useEffect, useState } from "react";
import { orchestratorAct, orchestratorMode, orchestratorView } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { useUI } from "../../lib/store";
import { SettingSection } from "../settings/SettingChrome";
import type { DeskView, Source } from "./desk-model";

/** WorkSources 把工作来源放进设置。打开时读一次编排台，保存后再读。 */
export function WorkSources() {
  const copy = useCopy();
  const openOrchestrator = useUI((s) => s.openOrchestrator);
  const [view, setView] = useState<DeskView | null>(null);
  const load = useCallback(async () => {
    try {
      setView(await orchestratorView());
    } catch {
      setView(null);
    }
  }, []);
  useEffect(() => { void load(); }, [load]);
  if (!view) return null;
  const github = view.sources?.find((src) => src.name === "github");
  const feishu = view.sources?.find((src) => src.name === "feishu");
  return (
    <SettingSection id="extensions-work" title={copy.settings.sections.extensionsWorkSources} description={copy.work.linksHint}>
      <Mode mode={view.mode} onMode={async (mode) => { await orchestratorMode(mode); await load(); }} />
      <Prepare initial={view.prepare || ""} onSave={async (text) => { await orchestratorAct("prepare", "", "", text); await load(); }} />
      <SourceCard
        title={copy.work.github}
        source={github}
        need={copy.work.githubNeed}
        fields={[
          { key: "repo", label: copy.work.repo, value: github?.repo || "" },
          { key: "account", label: copy.work.account, value: github?.account || "" },
          { key: "token", label: github?.hasSecret ? copy.work.tokenKept : copy.work.token, secret: true },
        ]}
        onSave={async (values, clear) => {
          await orchestratorAct("sources", "", "", JSON.stringify({
            githubRepo: values.repo || "",
            githubAccount: values.account || "",
            githubToken: values.token || "",
            clearGithubToken: clear,
            feishuAppId: feishu?.app || "",
            feishuChat: feishu?.chat || "",
            feishuMention: feishu?.account || "",
          }));
          await load();
        }}
      />
      <SourceCard
        title={copy.work.feishu}
        source={feishu}
        need={copy.work.feishuNeed}
        fields={[
          { key: "app", label: copy.work.appId, value: feishu?.app || "" },
          { key: "chat", label: copy.work.chat, value: feishu?.chat || "" },
          { key: "mention", label: copy.work.mention, value: feishu?.account || "" },
          { key: "secret", label: feishu?.hasSecret ? copy.work.secretKept : copy.work.secret, secret: true },
        ]}
        onSave={async (values, clear) => {
          await orchestratorAct("sources", "", "", JSON.stringify({
            githubRepo: github?.repo || "",
            githubAccount: github?.account || "",
            feishuAppId: values.app || "",
            feishuChat: values.chat || "",
            feishuMention: values.mention || "",
            feishuSecret: values.secret || "",
            clearFeishuSecret: clear,
          }));
          await load();
        }}
      />
      <Allowed initial={(view.allowed ?? []).join(", ")} onSave={async (text) => { await orchestratorAct("allowed", "", "", text); await load(); }} />
      <button type="button" className="mt-4 text-[12px] text-muted underline-offset-2 hover:text-foreground hover:underline" onClick={() => openOrchestrator()}>
        {copy.work.openDesk}
      </button>
    </SettingSection>
  );
}

function Mode(props: { mode: string; onMode: (mode: string) => Promise<void> }) {
  const copy = useCopy();
  const modes = [
    { id: "off", label: copy.work.modeOff },
    { id: "observe", label: copy.work.modeObserve },
    { id: "auto", label: copy.work.modeAuto },
  ];
  return (
    <div className="mt-4">
      <p className="text-[11px] font-medium text-muted">{copy.work.mode}</p>
      <div className="mt-1.5 flex max-w-md gap-1 rounded-xl bg-lift/50 p-1">
        {modes.map((mode) => (
          <button
            key={mode.id}
            type="button"
            className={cn("h-7 flex-1 rounded-lg text-[12px] font-medium", props.mode === mode.id ? "bg-background text-foreground shadow-sm" : "text-muted")}
            onClick={() => { void props.onMode(mode.id); }}
          >
            {mode.label}
          </button>
        ))}
      </div>
      <p className="mt-1.5 text-[12px] leading-5 text-muted">{copy.work.modeHint}</p>
    </div>
  );
}

function SourceCard(props: {
  title: string;
  source?: Source;
  need: string;
  fields: { key: string; label: string; value?: string; secret?: boolean }[];
  onSave: (values: Record<string, string>, clear: boolean) => Promise<void>;
}) {
  const copy = useCopy();
  const [values, setValues] = useState<Record<string, string>>(() => Object.fromEntries(props.fields.map((field) => [field.key, field.secret ? "" : field.value || ""])));
  const [clear, setClear] = useState(false);
  const [note, setNote] = useState("");
  const state = props.source?.state || "unconfigured";
  const stateLabel = state === "ok" ? copy.work.ok : state === "error" ? copy.work.error : copy.work.missing;
  return (
    <form
      className="mt-4 max-w-md border-t border-border/50 pt-3"
      onSubmit={(event) => {
        event.preventDefault();
        void props.onSave(values, clear).then(() => {
          setNote(copy.work.saved);
          setClear(false);
        }).catch((e) => setNote(e instanceof Error ? e.message : String(e)));
      }}
    >
      <div className="flex items-baseline justify-between gap-2">
        <h3 className="text-[13px] font-medium">{props.title}</h3>
        <span className={cn("text-[11px]", state === "error" ? "text-danger" : state === "ok" ? "text-foreground" : "text-warning")}>{stateLabel}</span>
      </div>
      {state !== "ok" ? <p className="mt-1 text-[12px] leading-5 text-muted">{props.source?.detail || props.need}</p> : null}
      {props.fields.map((field) => (
        <label key={field.key} className="mt-2.5 block">
          <span className="text-[11px] text-muted">{field.label}</span>
          <input
            className="mt-1 h-8 w-full rounded-xl border border-transparent bg-lift/80 px-2.5 text-[13px] outline-none focus:border-accent/40"
            value={values[field.key] ?? ""}
            type={field.secret ? "password" : "text"}
            autoComplete="off"
            spellCheck={false}
            onChange={(event) => setValues((cur) => ({ ...cur, [field.key]: event.target.value }))}
          />
        </label>
      ))}
      {props.fields.some((field) => field.secret && props.source?.hasSecret) ? (
        <label className="mt-2 flex items-center gap-2 text-[12px] text-muted">
          <input type="checkbox" checked={clear} onChange={(event) => setClear(event.target.checked)} />
          {copy.work.clearSecret}
        </label>
      ) : null}
      <div className="mt-2.5 flex items-center gap-3">
        <button type="submit" className="h-8 rounded-lg bg-foreground px-3 text-[12px] font-medium text-background">{copy.work.save}</button>
        {note ? <span className="text-[12px] text-muted">{note}</span> : null}
      </div>
    </form>
  );
}

function Prepare(props: { initial: string; onSave: (text: string) => Promise<void> }) {
  const copy = useCopy();
  const [text, setText] = useState(props.initial);
  const [note, setNote] = useState("");
  return (
    <form className="mt-4 max-w-md" onSubmit={(event) => {
      event.preventDefault();
      void props.onSave(text).then(() => setNote(copy.work.saved)).catch((e) => setNote(e instanceof Error ? e.message : String(e)));
    }}>
      <label className="block">
        <span className="text-[11px] text-muted">{copy.work.prepare}</span>
        <input className="mt-1 h-8 w-full rounded-xl bg-lift/80 px-2.5 text-[13px] outline-none" value={text} autoComplete="off" spellCheck={false} onChange={(event) => setText(event.target.value)} />
      </label>
      <p className="mt-1.5 text-[12px] leading-5 text-muted">{copy.work.prepareHint}</p>
      <div className="mt-2.5 flex items-center gap-3">
        <button type="submit" className="h-8 rounded-lg bg-foreground px-3 text-[12px] font-medium text-background">{copy.work.save}</button>
        {note ? <span className="text-[12px] text-muted">{note}</span> : null}
      </div>
    </form>
  );
}

function Allowed(props: { initial: string; onSave: (text: string) => Promise<void> }) {
  const copy = useCopy();
  const [text, setText] = useState(props.initial);
  const [note, setNote] = useState("");
  return (
    <form className="mt-4 max-w-md" onSubmit={(event) => {
      event.preventDefault();
      void props.onSave(text).then(() => setNote(copy.work.saved)).catch((e) => setNote(e instanceof Error ? e.message : String(e)));
    }}>
      <label className="block">
        <span className="text-[11px] text-muted">{copy.work.allowed}</span>
        <input className="mt-1 h-8 w-full rounded-xl bg-lift/80 px-2.5 text-[13px] outline-none" value={text} autoComplete="off" spellCheck={false} onChange={(event) => setText(event.target.value)} />
      </label>
      <p className="mt-1.5 text-[12px] leading-5 text-muted">{copy.work.allowedHint}</p>
      <div className="mt-2.5 flex items-center gap-3">
        <button type="submit" className="h-8 rounded-lg bg-foreground px-3 text-[12px] font-medium text-background">{copy.work.save}</button>
        {note ? <span className="text-[12px] text-muted">{note}</span> : null}
      </div>
    </form>
  );
}
