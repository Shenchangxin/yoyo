/**
 * 拥有：一键配置对话框（需要配置时点「配置」打开，只列必填和能复用的密钥、模型和审批，保存后自动全探），以及详情页也用的密钥行和密钥草稿。
 * 不拥有：保存接口和结果提示（buddy-actions.ts），也不保存任何密钥值：输入框里的字保存成功后立刻清空。
 */

import { useState } from "react";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import { Switch } from "../../components/ui/switch";
import type { BuddyConfigPatch, BuddySecret, BuddyView } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { FormDialog } from "../FormDialog";
import { buddyConfigure } from "./buddy-actions";
import { fill, setupSecrets, type BuddyCopy } from "./buddy-model";
import { useBuddy } from "./buddy-store";

/** DEFAULT_MODEL 是「用它自己的默认模型」在下拉里的值。Radix 的选项不能用空串。 */
export const DEFAULT_MODEL = "__default";

/** KeyDraft 是还没保存的密钥输入：values 是新填的值，cleared 是要删掉的，reuse 是复用开关。 */
export type KeyDraft = { values: Record<string, string>; cleared: Record<string, boolean>; reuse: Record<string, boolean> };

/** emptyDraft 按当前视图起一份草稿：没有新值、没有要删的，复用开关照现状。 */
export function emptyDraft(secrets: BuddySecret[]): KeyDraft {
  return { values: {}, cleared: {}, reuse: Object.fromEntries(secrets.map((s) => [s.name, !!s.reuse])) };
}

/** draftPatch 把草稿变成 buddy.configure 的 secrets 和 reuse 两项。没有改动的项不放进去。 */
export function draftPatch(secrets: BuddySecret[], draft: KeyDraft): Pick<BuddyConfigPatch, "secrets" | "reuse"> {
  const out: Pick<BuddyConfigPatch, "secrets" | "reuse"> = {};
  const keys: Record<string, string> = {};
  for (const s of secrets) {
    const value = (draft.values[s.name] ?? "").trim();
    if (value) keys[s.name] = value;
    else if (draft.cleared[s.name] && s.hasKey) keys[s.name] = "";
  }
  if (Object.keys(keys).length) out.secrets = keys;
  if (secrets.some((s) => s.reusable && !!draft.reuse[s.name] !== !!s.reuse)) {
    out.reuse = secrets.filter((s) => s.reusable && draft.reuse[s.name]).map((s) => s.name);
  }
  return out;
}

/** SecretRow 是一把密钥：名字、已保存的标记和删除、密码输入框；yoyo 有同类钥匙时多一个复用开关，并写明会把钥匙交给它。 */
export function SecretRow(props: { s: BuddySecret; draft: KeyDraft; onDraft: (next: KeyDraft) => void; c: BuddyCopy }) {
  const { s, draft, c } = props;
  const id = `buddy-key-${s.name}`;
  const saved = s.hasKey && !draft.cleared[s.name];
  const reuse = !!draft.reuse[s.name];
  const label = s.label || s.name;
  return (
    <div className="space-y-1.5">
      <div className="flex min-w-0 items-center gap-2">
        <label htmlFor={id} className="min-w-0 flex-1 truncate text-ui-footnote text-muted">
          <span className="text-foreground">{label}</span>
          {s.label ? <span className="ml-1.5 font-mono text-ui-caption">{s.name}</span> : null}
          {s.optional ? <span className="ml-1.5">{c.detail.optional}</span> : null}
        </label>
        {saved ? <Badge className="text-status-done">{c.detail.keySaved}</Badge> : null}
        {saved ? (
          <Button size="xs" variant="ghost" onClick={() => props.onDraft({ ...draft, cleared: { ...draft.cleared, [s.name]: true } })}>
            {c.detail.keyClear}
          </Button>
        ) : null}
      </div>
      <Input
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
        <label className="flex items-start gap-2 text-ui-footnote text-muted">
          <Switch className="mt-0.5" checked={reuse} onCheckedChange={(on) => props.onDraft({ ...draft, reuse: { ...draft.reuse, [s.name]: on } })} aria-label={c.detail.reuse} />
          <span>
            <span className="block text-foreground">{c.detail.reuse}</span>
            {fill(c.detail.reuseNote, { name: label })}
          </span>
        </label>
      ) : null}
    </div>
  );
}

/** DIALOG_LAYER 让下拉浮在对话框（z-200）上面。 */
const DIALOG_LAYER = "z-[210]";

/** ModelField 是模型：探测到列表、而且当前值就在列表里时用下拉；否则用输入框，好填 yoyo 的模型或别的名字。 */
export function ModelField(props: { v: BuddyView; value: string; onChange: (model: string) => void; c: BuddyCopy; id?: string; layer?: string }) {
  const models = props.v.config.models ?? [];
  const known = !props.value || models.some((m) => m.id === props.value);
  if (models.length && known) return <ModelSelect {...props} />;
  const listId = props.id ? `${props.id}-models` : undefined;
  return (
    <>
      <Input
        id={props.id}
        list={listId}
        className="h-8"
        value={props.value}
        placeholder={props.c.detail.modelPlaceholder}
        spellCheck={false}
        onChange={(e) => props.onChange(e.target.value)}
      />
      {listId && models.length ? (
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

/** ModelSelect 是模型下拉：第一项是用 agent 自己的默认，其余是探测时拿到的模型。layer 是下拉层的类名，放在对话框里时要抬高。 */
export function ModelSelect(props: { v: BuddyView; value: string; onChange: (model: string) => void; c: BuddyCopy; id?: string; layer?: string }) {
  const models = props.v.config.models ?? [];
  return (
    <Select value={props.value || DEFAULT_MODEL} onValueChange={(m) => props.onChange(m === DEFAULT_MODEL ? "" : m)}>
      <SelectTrigger id={props.id} className="h-8 w-full" aria-label={props.c.detail.model}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent className={props.layer}>
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

/** ApprovalSelect 是审批下拉：yoyo 决定（默认）或每次问我。没有会降低监督的选项。layer 同 ModelSelect。 */
export function ApprovalSelect(props: { value: "yoyo" | "ask"; onChange: (v: "yoyo" | "ask") => void; c: BuddyCopy; id?: string; layer?: string }) {
  return (
    <Select value={props.value} onValueChange={(v) => props.onChange(v === "ask" ? "ask" : "yoyo")}>
      <SelectTrigger id={props.id} className="h-8 w-full" aria-label={props.c.detail.approval}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent className={props.layer}>
        <SelectItem value="yoyo">{props.c.detail.approvalYoyo}</SelectItem>
        <SelectItem value="ask">{props.c.detail.approvalAsk}</SelectItem>
      </SelectContent>
    </Select>
  );
}

/** SetupForm 是一键配置的表单。初值取自打开时的视图；保存成功后关掉对话框，等全探结果再提示。 */
function SetupForm({ v, onClose }: { v: BuddyView; onClose: () => void }) {
  const c = useCopy().buddy;
  const pending = useBuddy((s) => s.pending[v.id] || "");
  const secrets = setupSecrets(v);
  const [draft, setDraft] = useState<KeyDraft>(() => emptyDraft(secrets));
  const [model, setModel] = useState(v.config.model ?? "");
  const [approval, setApproval] = useState<"yoyo" | "ask">(v.config.approval);
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        const patch: BuddyConfigPatch = { ...draftPatch(secrets, draft) };
        if (model !== (v.config.model ?? "")) patch.model = model;
        if (approval !== v.config.approval) patch.approval = approval;
        void buddyConfigure(v, patch, true).then((next) => {
          if (!next) return;
          setDraft(emptyDraft(next.config.secrets));
          onClose();
        });
      }}
    >
      {secrets.length ? (
        secrets.map((s) => <SecretRow key={s.name} s={s} draft={draft} onDraft={setDraft} c={c} />)
      ) : (
        <p className="text-ui-body text-muted">{c.setup.nothing}</p>
      )}
      <div className="space-y-1.5">
        <label htmlFor="buddy-setup-model" className="text-ui-footnote text-muted">{c.detail.model}</label>
        <ModelField id="buddy-setup-model" v={v} value={model} onChange={setModel} c={c} layer={DIALOG_LAYER} />
        <p className="text-ui-caption text-muted">{c.detail.modelHint}</p>
      </div>
      <div className="space-y-1.5">
        <label htmlFor="buddy-setup-approval" className="text-ui-footnote text-muted">{c.detail.approval}</label>
        <ApprovalSelect id="buddy-setup-approval" value={approval} onChange={setApproval} c={c} layer={DIALOG_LAYER} />
      </div>
      <div className="dialog-actions">
        <Button type="button" size="sm" variant="lift" onClick={onClose}>
          {c.action.cancel}
        </Button>
        <Button type="submit" size="sm" disabled={!!pending} aria-busy={!!pending || undefined}>
          {c.setup.save}
        </Button>
      </div>
    </form>
  );
}

/** BuddyConfigDialog 是一键配置对话框。v 为空时关着。 */
export function BuddyConfigDialog({ v, onClose }: { v: BuddyView | undefined; onClose: () => void }) {
  const c = useCopy().buddy;
  return (
    <FormDialog open={!!v} title={v ? fill(c.setup.title, { name: v.name }) : ""} description={v ? fill(c.setup.body, { name: v.name }) : undefined} onClose={onClose}>
      {v ? <SetupForm key={v.id} v={v} onClose={onClose} /> : null}
    </FormDialog>
  );
}
