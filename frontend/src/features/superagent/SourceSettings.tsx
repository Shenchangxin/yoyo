/**
 * 拥有：设置里的 GitHub 和飞书来源。人在这里填写仓库、令牌和可以派工的人。
 * 不拥有：来源怎么轮询。那一段在 superagent。
 */

import { useCallback, useEffect, useState } from "react";
import { FormDialog } from "../FormDialog";
import { Button } from "../../components/ui/button";
import { orchestratorAct, orchestratorMode, orchestratorView } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { SettingActionRow, SettingRow, SettingSection, SettingSegmented } from "../settings/SettingChrome";
import type { DeskView, Source } from "./desk-model";

export type WorkSourceName = "github" | "feishu" | "prepare" | "allowed";

/** WorkSources 把工作来源放进设置。凭证在弹框里填。打开时读一次编排台，保存后再读。 */
export function WorkSources(props: { openSource?: WorkSourceName | null; onOpenSource?: (name: WorkSourceName | null) => void }) {
  const copy = useCopy();
  const [view, setView] = useState<DeskView | null>(null);
  const [which, setWhich] = useState<WorkSourceName | null>(null);
  useEffect(() => {
    if (props.openSource) setWhich(props.openSource);
  }, [props.openSource]);
  function closeWhich() {
    setWhich(null);
    props.onOpenSource?.(null);
  }
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
      <ConfigRow title={copy.work.prepare} detail={view.prepare || copy.work.prepareHint} action={copy.settings.configure} onOpen={() => setWhich("prepare")} />
      <ConfigRow title={copy.work.github} detail={github?.repo || copy.work.githubNeed} state={github?.state} action={copy.settings.configure} onOpen={() => setWhich("github")} />
      <ConfigRow title={copy.work.feishu} detail={feishu?.chat || copy.work.feishuNeed} state={feishu?.state} action={copy.settings.configure} onOpen={() => setWhich("feishu")} />
      <ConfigRow title={copy.work.allowed} detail={(view.allowed ?? []).join(", ") || copy.work.allowedHint} action={copy.settings.configure} onOpen={() => setWhich("allowed")} />
      <TextDialog
        open={which === "prepare"}
        title={copy.work.prepare}
        hint={copy.work.prepareHint}
        initial={view.prepare || ""}
        onClose={closeWhich}
        onSave={async (text) => { await orchestratorAct("prepare", "", "", text); await load(); }}
      />
      <SourceDialog
        open={which === "github"}
        title={copy.work.github}
        source={github}
        need={copy.work.githubNeed}
        fields={[
          { key: "repo", label: copy.work.repo, value: github?.repo || "" },
          { key: "account", label: copy.work.account, value: github?.account || "" },
          { key: "token", label: github?.hasSecret ? copy.work.tokenKept : copy.work.token, secret: true },
        ]}
        onClose={closeWhich}
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
      <SourceDialog
        open={which === "feishu"}
        title={copy.work.feishu}
        source={feishu}
        need={copy.work.feishuNeed}
        fields={[
          { key: "app", label: copy.work.appId, value: feishu?.app || "" },
          { key: "chat", label: copy.work.chat, value: feishu?.chat || "" },
          { key: "mention", label: copy.work.mention, value: feishu?.account || "" },
          { key: "secret", label: feishu?.hasSecret ? copy.work.secretKept : copy.work.secret, secret: true },
        ]}
        onClose={closeWhich}
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
      <TextDialog
        open={which === "allowed"}
        title={copy.work.allowed}
        hint={copy.work.allowedHint}
        initial={(view.allowed ?? []).join(", ")}
        onClose={closeWhich}
        onSave={async (text) => { await orchestratorAct("allowed", "", "", text); await load(); }}
      />
    </SettingSection>
  );
}

/** Mode 是新工作到来时怎么办：关闭、观察、自动。和设置里别的行一个版式，左边是说明，右边是分段。 */
function Mode(props: { mode: string; onMode: (mode: string) => Promise<void> }) {
  const copy = useCopy();
  const value = props.mode === "off" || props.mode === "auto" ? props.mode : "observe";
  return (
    <SettingRow title={copy.work.mode} description={copy.work.modeHint}>
      <SettingSegmented
        id="work-source-mode"
        ariaLabel={copy.work.mode}
        value={value}
        options={[
          { value: "off", label: copy.work.modeOff },
          { value: "observe", label: copy.work.modeObserve },
          { value: "auto", label: copy.work.modeAuto },
        ]}
        onChange={(next) => { void props.onMode(next); }}
      />
    </SettingRow>
  );
}

/** ConfigRow 是一条来源：名字、现在填了什么、接没接上，点开在弹框里改。 */
function ConfigRow(props: { title: string; detail: string; state?: string; action: string; onOpen: () => void }) {
  const copy = useCopy();
  const state = props.state || "";
  const stateLabel = state === "ok" ? copy.work.ok : state === "error" ? copy.work.error : state ? copy.work.missing : "";
  return (
    <SettingActionRow
      title={props.title}
      description={props.detail}
      onClick={props.onOpen}
      trailing={
        <span className="flex items-center gap-2">
          {stateLabel ? (
            <span className={cn("inline-flex items-center gap-1.5 text-ui-footnote", state === "error" ? "text-danger" : state === "ok" ? "text-foreground" : "text-warning")}>
              <span className={cn("size-1.5 rounded-full", state === "error" ? "bg-status-failed" : state === "ok" ? "bg-status-done" : "bg-status-needs")} aria-hidden />
              {stateLabel}
            </span>
          ) : null}
          <span>{props.action}</span>
        </span>
      }
    />
  );
}

function SourceDialog(props: {
  open: boolean;
  title: string;
  source?: Source;
  need: string;
  fields: { key: string; label: string; value?: string; secret?: boolean }[];
  onClose: () => void;
  onSave: (values: Record<string, string>, clear: boolean) => Promise<void>;
}) {
  const copy = useCopy();
  const [values, setValues] = useState<Record<string, string>>({});
  const [clear, setClear] = useState(false);
  const [note, setNote] = useState("");
  useEffect(() => {
    if (!props.open) return;
    setValues(Object.fromEntries(props.fields.map((field) => [field.key, field.secret ? "" : field.value || ""])));
    setClear(false);
    setNote("");
    // 只在打开时带入当前值。fields 每次渲染都是新数组，不能放进依赖。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.open]);
  const state = props.source?.state || "unconfigured";
  return (
    <FormDialog open={props.open} title={props.title} description={state === "ok" ? undefined : props.source?.detail || props.need} onClose={props.onClose}>
      <form
        className="flex flex-col gap-3"
        onSubmit={(event) => {
          event.preventDefault();
          void props.onSave(values, clear).then(() => {
            setNote(copy.work.saved);
            setClear(false);
            props.onClose();
          }).catch((e) => setNote(e instanceof Error ? e.message : String(e)));
        }}
      >
        {props.fields.map((field) => (
          <label key={field.key} className="block">
            <span className="text-ui-footnote text-muted">{field.label}</span>
            <input
              className="mt-1 h-8 w-full rounded-xl border border-border bg-background px-2.5 text-ui-body outline-none focus:border-foreground/25"
              value={values[field.key] ?? ""}
              type={field.secret ? "password" : "text"}
              autoComplete="off"
              spellCheck={false}
              onChange={(event) => setValues((cur) => ({ ...cur, [field.key]: event.target.value }))}
            />
          </label>
        ))}
        {props.fields.some((field) => field.secret && props.source?.hasSecret) ? (
          <label className="flex items-center gap-2 text-ui-footnote text-muted">
            <input type="checkbox" checked={clear} onChange={(event) => setClear(event.target.checked)} />
            {copy.work.clearSecret}
          </label>
        ) : null}
        {note ? <p className="text-ui-footnote text-muted">{note}</p> : null}
        <div className="dialog-actions">
          <Button type="button" size="sm" variant="lift" onClick={props.onClose}>{copy.dialog.cancel}</Button>
          <Button type="submit" size="sm">{copy.work.save}</Button>
        </div>
      </form>
    </FormDialog>
  );
}

function TextDialog(props: { open: boolean; title: string; hint: string; initial: string; onClose: () => void; onSave: (text: string) => Promise<void> }) {
  const copy = useCopy();
  const [text, setText] = useState(props.initial);
  const [note, setNote] = useState("");
  useEffect(() => {
    if (!props.open) return;
    setText(props.initial);
    setNote("");
  }, [props.open, props.initial]);
  return (
    <FormDialog open={props.open} title={props.title} description={props.hint} onClose={props.onClose}>
      <form
        className="flex flex-col gap-3"
        onSubmit={(event) => {
          event.preventDefault();
          void props.onSave(text).then(() => {
            setNote(copy.work.saved);
            props.onClose();
          }).catch((e) => setNote(e instanceof Error ? e.message : String(e)));
        }}
      >
        <input className="h-8 w-full rounded-xl border border-border bg-background px-2.5 text-ui-body outline-none" value={text} autoComplete="off" spellCheck={false} onChange={(event) => setText(event.target.value)} />
        {note ? <p className="text-ui-footnote text-muted">{note}</p> : null}
        <div className="dialog-actions">
          <Button type="button" size="sm" variant="lift" onClick={props.onClose}>{copy.dialog.cancel}</Button>
          <Button type="submit" size="sm">{copy.work.save}</Button>
        </div>
      </form>
    </FormDialog>
  );
}
