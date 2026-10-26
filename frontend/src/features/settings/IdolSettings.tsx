/**
 * 拥有：偶像对话、语音、听写和图像服务的表单。
 * 不拥有：偶像档案。保存和探测会调用 api 改状态。
 */
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Switch } from "../../components/ui/switch";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { composerModelGroups, decodeChatModel, encodeChatModel } from "../../lib/models-dev";
import { asArray, str } from "../../lib/normalize";
import type { ChatProvider } from "../../lib/protocol";
import { customProvider, presetByID, presetsOf, voicesFor, type ServicePreset, type ServiceVoice } from "../idol/service-presets";
import { SettingRow, SettingSection, SettingsGroupLabel } from "./SettingChrome";

type Form = {
  chat_base_url: string;
  chat_model: string;
  judge_model: string;
  speech_provider: string;
  speech_base_url: string;
  speech_model: string;
  speech_voice: string;
  asr_provider: string;
  asr_base_url: string;
  asr_model: string;
  image_provider: string;
  image_base_url: string;
  image_model: string;
  reach_out: boolean;
  text_only: boolean;
  phone_texture: boolean;
  work_enabled: boolean;
  work_profile_id: string;
  speech_key: string;
  asr_key: string;
  image_key: string;
  address: string;
  gender: string;
  timezone: string;
  birthday: string;
  speech_key_hint: string;
  asr_key_hint: string;
  image_key_hint: string;
};

type Fallback = {
  speech_preset: string;
  speech_name: string;
  speech_model: string;
  asr_preset: string;
  asr_name: string;
  asr_model: string;
  image_preset: string;
  image_name: string;
  image_model: string;
};

const emptyFallback: Fallback = {
  speech_preset: "", speech_name: "", speech_model: "",
  asr_preset: "", asr_name: "", asr_model: "",
  image_preset: "", image_name: "", image_model: "",
};

const empty: Form = {
  chat_base_url: "", chat_model: "", judge_model: "",
  speech_provider: "", speech_base_url: "", speech_model: "", speech_voice: "",
  asr_provider: "", asr_base_url: "", asr_model: "",
  image_provider: "", image_base_url: "", image_model: "",
  reach_out: true, text_only: false, phone_texture: false, work_enabled: true, work_profile_id: "",
  speech_key: "", asr_key: "", image_key: "",
  address: "", gender: "", timezone: "", birthday: "",
  speech_key_hint: "", asr_key_hint: "", image_key_hint: "",
};

const fieldClass = "mt-1 h-8 w-full rounded-md border border-border bg-background px-2 text-[13px]";

/** 把已存的设置收成表单。v 读取 user、chat_base_url、chat_model、judge_model、speech_provider、speech_base_url 等，为空时不读。 */
function fromRaw(v: any): Form {
  const u = v?.user || {};
  return {
    ...empty,
    chat_base_url: str(v?.chat_base_url),
    chat_model: str(v?.chat_model),
    judge_model: str(v?.judge_model),
    speech_provider: str(v?.speech_provider),
    speech_base_url: str(v?.speech_base_url),
    speech_model: str(v?.speech_model),
    speech_voice: str(v?.speech_voice),
    asr_provider: str(v?.asr_provider),
    asr_base_url: str(v?.asr_base_url),
    asr_model: str(v?.asr_model),
    image_provider: str(v?.image_provider),
    image_base_url: str(v?.image_base_url),
    image_model: str(v?.image_model),
    reach_out: v?.reach_out !== false,
    text_only: !!v?.text_only,
    phone_texture: !!v?.phone_texture,
    work_enabled: v?.work_enabled !== false,
    work_profile_id: str(v?.work_profile_id),
    address: str(u.address),
    gender: str(u.gender),
    timezone: str(u.timezone),
    birthday: str(u.birthday),
    speech_key_hint: str(v?.speech_key_hint),
    asr_key_hint: str(v?.asr_key_hint),
    image_key_hint: str(v?.image_key_hint),
  };
}

/** 取出没选服务商时用来垫的预设。v 读取 fallback，为空时不读。 */
function fallbackOf(v: any): Fallback {
  const f = v?.fallback || {};
  return {
    speech_preset: str(f.speech_preset),
    speech_name: str(f.speech_name),
    speech_model: str(f.speech_model),
    asr_preset: str(f.asr_preset),
    asr_name: str(f.asr_name),
    asr_model: str(f.asr_model),
    image_preset: str(f.image_preset),
    image_name: str(f.image_name),
    image_model: str(f.image_model),
  };
}

/** 有预设就用预设，否则有地址就当自定义。provider 传给 presetByID。base 去掉首尾空白，为真时才继续。presets 传给 presetByID。kind 传给 presetByID。 */
function choice(provider: string, base: string, presets: ServicePreset[], kind: string): string {
  if (presetByID(presets, provider, kind)) return provider;
  if (base.trim()) return customProvider;
  return "";
}

/**
 * 把模型和声线夹进当前预设里有的那些。
 * form 读取 speech_provider、asr_provider、image_provider、speech_model、speech_voice、asr_model 等。
 * presets 传给 presetByID。
 * kind 与 speech、asr 比较是否相等，传给 presetByID。
 */
function clampKind(form: Form, presets: ServicePreset[], kind: "speech" | "asr" | "image") {
  const id = kind === "speech" ? form.speech_provider : kind === "asr" ? form.asr_provider : form.image_provider;
  const preset = presetByID(presets, id, kind);
  if (!preset?.models.length) return;
  if (kind === "speech") {
    if (!preset.models.includes(form.speech_model)) form.speech_model = preset.models[0];
    const voices = voicesFor(preset, form.speech_model);
    if (voices.length && !voices.some((v) => v.id === form.speech_voice)) form.speech_voice = voices[0].id;
    return;
  }
  if (kind === "asr") {
    if (!preset.models.includes(form.asr_model)) form.asr_model = preset.models[0];
    return;
  }
  if (!preset.models.includes(form.image_model)) form.image_model = preset.models[0];
}

/**
 * 按预设把三类服务对齐，空着的语音可以垫一条声线。
 * form 读取 speech_provider、speech_base_url、asr_provider、asr_base_url、image_provider、image_base_url，展开传递。
 * presets 传给 choice，传给 clampKind，传给 presetByID。
 * fallback 读取 speech_preset、speech_model。
 */
function align(form: Form, presets: ServicePreset[], fallback: Fallback): Form {
  const next: Form = {
    ...form,
    speech_provider: choice(form.speech_provider, form.speech_base_url, presets, "speech"),
    asr_provider: choice(form.asr_provider, form.asr_base_url, presets, "asr"),
    image_provider: choice(form.image_provider, form.image_base_url, presets, "image"),
  };
  clampKind(next, presets, "speech");
  clampKind(next, presets, "asr");
  clampKind(next, presets, "image");
  if (next.speech_provider === "" && !next.speech_voice.trim()) {
    const hint = presetByID(presets, fallback.speech_preset, "speech");
    const voices = voicesFor(hint, fallback.speech_model || hint?.models[0] || "");
    if (voices.length) next.speech_voice = voices[0].id;
  }
  return next;
}

/** 对话模型仍在目录里就留住它带上的服务商。providers 传给 composerModelGroups。raw 去掉首尾空白后，为假时走另一支，传给 decodeChatModel。 */
function keepChat(providers: ChatProvider[], raw: string): string {
  const value = raw.trim();
  if (!value) return "";
  const decoded = decodeChatModel(value);
  const model = decoded.model.trim();
  const groups = composerModelGroups(providers);
  if (decoded.providerId && groups.some((g) => g.providerId === decoded.providerId && g.models.includes(model))) {
    return encodeChatModel(decoded.providerId, model);
  }
  const owners = groups.filter((g) => g.models.includes(model));
  if (owners.length === 1) return encodeChatModel(owners[0].providerId, model);
  return value;
}

/** 自定义和空选择存成空的服务商编号。id 与 customProvider 比较是否相等，为假时走另一支，作为分支结果。 */
function storedProvider(id: string): string {
  return !id || id === customProvider ? "" : id;
}

/** 自定义存地址，预设存它的端点。id 与 customProvider 比较是否相等，为假时走另一支，传给 presetByID。base 去掉首尾空白，作为分支结果，作为后备值。presets 传给 presetByID。kind 传给 presetByID。 */
function storedBase(id: string, base: string, presets: ServicePreset[], kind: string): string {
  if (!id || id === customProvider) return id === customProvider ? base.trim() : "";
  return presetByID(presets, id, kind)?.endpoint || base.trim();
}

/** 自定义或已选预设时存模型，空选择存空。id 与 customProvider 比较是否相等，为假时走另一支。model 去掉首尾空白，作为分支结果。 */
function storedModel(id: string, model: string): string {
  if (!id || id === customProvider) return id === customProvider ? model.trim() : "";
  return model.trim();
}

/** IdolSettings 编辑偶像用的对话、语音、听写和图像。 */
export function IdolSettings() {
  const copy = useCopy();
  const c = copy.idol;
  const [form, setForm] = useState<Form>(empty);
  const [presets, setPresets] = useState<ServicePreset[]>([]);
  const [providers, setProviders] = useState<ChatProvider[]>([]);
  const [fallback, setFallback] = useState<Fallback>(emptyFallback);
  const [profiles, setProfiles] = useState<{ id: string; name: string }[]>([]);
  const [busy, setBusy] = useState(false);
  const groups = composerModelGroups(providers);
  const knownChat = new Set(groups.flatMap((g) => g.models.map((m) => encodeChatModel(g.providerId, m))));

  useEffect(() => {
    let stop = false;
    void (async () => {
      try {
        const [raw, listed] = await Promise.all([
          api.idolCall("settings_get"),
          api.listChatProviders().catch(() => [] as ChatProvider[]),
        ]);
        if (stop) return;
        const nextPresets = presetsOf(raw?.presets);
        const nextFallback = fallbackOf(raw);
        const next = align(fromRaw(raw), nextPresets, nextFallback);
        next.chat_model = keepChat(listed, next.chat_model);
        next.judge_model = next.judge_model ? keepChat(listed, next.judge_model) : "";
        setPresets(nextPresets);
        setProviders(listed);
        setFallback(nextFallback);
        setForm(next);
      } catch (e: any) {
        if (!stop) toast.error(String(e?.message || e));
      }
    })();
    void api.idolCall("profiles").then((v) => {
      if (stop) return;
      setProfiles(asArray(v).map((p: any) => ({ id: str(p.id || p.ID), name: str(p.name || p.Name) })).filter((p) => p.id));
    }).catch(() => {});
    return () => { stop = true; };
  }, []);

  /** 把改动写回表单。patch 展开传递。 */
  const set = (patch: Partial<Form>) => setForm((f) => ({ ...f, ...patch }));

  /** 换某一类的服务商，并把模型和地址夹到预设上。kind 与 speech、asr 比较是否相等。id 与 customProvider 比较是否相等，赋给 next.speech_provider，传给 presetByID。 */
  function setProvider(kind: "speech" | "asr" | "image", id: string) {
    setForm((f) => {
      const next = { ...f };
      if (kind === "speech") {
        next.speech_provider = id;
        if (id === customProvider) return next;
        if (id === "") {
          next.speech_base_url = "";
          next.speech_model = "";
          const hint = presetByID(presets, fallback.speech_preset, "speech");
          const voices = voicesFor(hint, fallback.speech_model || hint?.models[0] || "");
          if (voices.length && !voices.some((v) => v.id === next.speech_voice)) next.speech_voice = voices[0].id;
          return next;
        }
        const p = presetByID(presets, id, "speech");
        if (!p) return f;
        next.speech_base_url = p.endpoint;
        next.speech_model = p.models.includes(next.speech_model) ? next.speech_model : (p.models[0] || "");
        const voices = voicesFor(p, next.speech_model);
        if (voices.length && !voices.some((v) => v.id === next.speech_voice)) next.speech_voice = voices[0].id;
        return next;
      }
      if (kind === "asr") {
        next.asr_provider = id;
        if (id === customProvider) return next;
        if (id === "") {
          next.asr_base_url = "";
          next.asr_model = "";
          return next;
        }
        const p = presetByID(presets, id, "asr");
        if (!p) return f;
        next.asr_base_url = p.endpoint;
        next.asr_model = p.models.includes(next.asr_model) ? next.asr_model : (p.models[0] || "");
        return next;
      }
      next.image_provider = id;
      if (id === customProvider) return next;
      if (id === "") {
        next.image_base_url = "";
        next.image_model = "";
        return next;
      }
      const p = presetByID(presets, id, "image");
      if (!p) return f;
      next.image_base_url = p.endpoint;
      next.image_model = p.models.includes(next.image_model) ? next.image_model : (p.models[0] || "");
      return next;
    });
  }

  /** 换语音模型，声线不在列表里就换一条。model 传给 voicesFor，写入字段 speech_model。 */
  function setSpeechModel(model: string) {
    setForm((f) => {
      const p = presetByID(presets, f.speech_provider, "speech");
      const voices = voicesFor(p, model);
      const voice = voices.length && !voices.some((v) => v.id === f.speech_voice) ? voices[0].id : f.speech_voice;
      return { ...f, speech_model: model, speech_voice: voice };
    });
  }

  /** 把表单存成偶像服务设置。 */
  async function save() {
    setBusy(true);
    try {
      const nextRaw = await api.idolCall("settings_save", {
        chat_base_url: groups.length > 0 || form.chat_model ? "" : form.chat_base_url,
        chat_model: form.chat_model,
        judge_model: form.judge_model,
        speech_provider: storedProvider(form.speech_provider),
        speech_base_url: storedBase(form.speech_provider, form.speech_base_url, presets, "speech"),
        speech_model: storedModel(form.speech_provider, form.speech_model),
        speech_voice: form.speech_voice,
        asr_provider: storedProvider(form.asr_provider),
        asr_base_url: storedBase(form.asr_provider, form.asr_base_url, presets, "asr"),
        asr_model: storedModel(form.asr_provider, form.asr_model),
        image_provider: storedProvider(form.image_provider),
        image_base_url: storedBase(form.image_provider, form.image_base_url, presets, "image"),
        image_model: storedModel(form.image_provider, form.image_model),
        reach_out: form.reach_out,
        text_only: form.text_only,
        phone_texture: form.phone_texture,
        work_enabled: form.work_enabled,
        work_profile_id: form.work_profile_id,
        speech_key: form.speech_key,
        asr_key: form.asr_key,
        image_key: form.image_key,
        user: { address: form.address, gender: form.gender, timezone: form.timezone, birthday: form.birthday },
      });
      const nextPresets = presetsOf(nextRaw?.presets);
      const nextFallback = fallbackOf(nextRaw);
      const next = align(fromRaw(nextRaw), nextPresets, nextFallback);
      next.chat_model = keepChat(providers, next.chat_model);
      next.judge_model = next.judge_model ? keepChat(providers, next.judge_model) : "";
      setPresets(nextPresets);
      setFallback(nextFallback);
      setForm(next);
      toast.success(copy.app.controlSaved);
    } catch (e: any) {
      toast.error(String(e?.message || e));
    } finally {
      setBusy(false);
    }
  }

  /** 探测某一类连接。kind 写入字段 kind。 */
  async function ping(kind: string) {
    try {
      await api.idolCall("ping", { kind });
      toast.success("OK");
    } catch (e: any) {
      toast.error(String(e?.message || e));
    }
  }

  const saveButton = <Button size="sm" disabled={busy} onClick={() => void save()}>{c.save}</Button>;

  return (
    <>
      <SettingSection id="idol-services" title={copy.settings.sections.idolServices} description={copy.settings.tabHints.idol} actions={saveButton}>
        <label className="block text-[12px] text-muted">
          {c.chatModel}
          <select className={fieldClass} value={form.chat_model} onChange={(e) => set({ chat_model: e.target.value })}>
            <option value="">{c.useSession}</option>
            {form.chat_model && !knownChat.has(form.chat_model) ? <option value={form.chat_model}>{decodeChatModel(form.chat_model).model || form.chat_model}</option> : null}
            {groups.map((g) => (
              <optgroup key={g.providerId} label={g.providerName}>
                {g.models.map((m) => <option key={m} value={encodeChatModel(g.providerId, m)}>{m}</option>)}
              </optgroup>
            ))}
          </select>
        </label>
        <label className="block text-[12px] text-muted">
          {c.judgeModel}
          <select className={fieldClass} value={form.judge_model} onChange={(e) => set({ judge_model: e.target.value })}>
            <option value="">{c.sameAsChat}</option>
            {form.judge_model && !knownChat.has(form.judge_model) ? <option value={form.judge_model}>{decodeChatModel(form.judge_model).model || form.judge_model}</option> : null}
            {groups.map((g) => (
              <optgroup key={g.providerId} label={g.providerName}>
                {g.models.map((m) => <option key={m} value={encodeChatModel(g.providerId, m)}>{m}</option>)}
              </optgroup>
            ))}
          </select>
        </label>
        {groups.length === 0 ? <p className="text-[12px] leading-relaxed text-muted">{c.noTextModels}</p> : null}
        <MediaFields
          title={c.speechBlock}
          kind="speech"
          presets={presets}
          provider={form.speech_provider}
          base={form.speech_base_url}
          model={form.speech_model}
          voice={form.speech_voice}
          keyValue={form.speech_key}
          keyHint={form.speech_key_hint}
          fallbackPreset={fallback.speech_preset}
          fallbackName={fallback.speech_name}
          fallbackModel={fallback.speech_model}
          onProvider={(id) => setProvider("speech", id)}
          onBase={(v) => set({ speech_base_url: v, speech_provider: v.trim() ? customProvider : "" })}
          onModel={setSpeechModel}
          onVoice={(v) => set({ speech_voice: v })}
          onKey={(v) => set({ speech_key: v })}
        />
        <MediaFields
          title={c.asrBlock}
          kind="asr"
          presets={presets}
          provider={form.asr_provider}
          base={form.asr_base_url}
          model={form.asr_model}
          keyValue={form.asr_key}
          keyHint={form.asr_key_hint}
          fallbackPreset={fallback.asr_preset}
          fallbackName={fallback.asr_name}
          fallbackModel={fallback.asr_model}
          onProvider={(id) => setProvider("asr", id)}
          onBase={(v) => set({ asr_base_url: v, asr_provider: v.trim() ? customProvider : "" })}
          onModel={(v) => set({ asr_model: v })}
          onKey={(v) => set({ asr_key: v })}
        />
        <MediaFields
          title={c.imageBlock}
          kind="image"
          presets={presets}
          provider={form.image_provider}
          base={form.image_base_url}
          model={form.image_model}
          keyValue={form.image_key}
          keyHint={form.image_key_hint}
          fallbackPreset={fallback.image_preset}
          fallbackName={fallback.image_name}
          fallbackModel={fallback.image_model}
          onProvider={(id) => setProvider("image", id)}
          onBase={(v) => set({ image_base_url: v, image_provider: v.trim() ? customProvider : "" })}
          onModel={(v) => set({ image_model: v })}
          onKey={(v) => set({ image_key: v })}
        />
        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="lift" onClick={() => void ping("chat")}>{c.tryChat}</Button>
          <Button size="sm" variant="lift" onClick={() => void ping("speech")}>{c.trySpeech}</Button>
          <Button size="sm" variant="lift" onClick={() => void ping("image")}>{c.tryImage}</Button>
        </div>
      </SettingSection>
      <SettingSection id="idol-you" title={copy.settings.sections.idolYou}>
        <Field label={c.name} value={form.address} onChange={(v) => set({ address: v })} />
        <Field label="Gender" value={form.gender} onChange={(v) => set({ gender: v })} />
        <Field label="Timezone" value={form.timezone} placeholder="Asia/Shanghai" onChange={(v) => set({ timezone: v })} />
        <Field label="Birthday MM-DD" value={form.birthday} placeholder="03-21" onChange={(v) => set({ birthday: v })} />
      </SettingSection>
      <SettingSection id="idol-work" title={copy.settings.sections.idolWork} description={c.workHint} actions={saveButton}>
        <SettingRow title={c.work}>
          <Switch checked={form.work_enabled} onCheckedChange={(v) => set({ work_enabled: v })} />
        </SettingRow>
        <SettingRow title={c.reach}>
          <Switch checked={form.reach_out} onCheckedChange={(v) => set({ reach_out: v })} />
        </SettingRow>
        <SettingRow title={c.textOnly}>
          <Switch checked={form.text_only} onCheckedChange={(v) => set({ text_only: v })} />
        </SettingRow>
        <SettingRow title={c.phone} border={false}>
          <Switch checked={form.phone_texture} onCheckedChange={(v) => set({ phone_texture: v })} />
        </SettingRow>
        <label className="block text-[12px] text-muted">
          {c.workProfile}
          <select className={fieldClass} value={form.work_profile_id} onChange={(e) => set({ work_profile_id: e.target.value })}>
            <option value="">{c.profileDefault}</option>
            {profiles.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
          </select>
        </label>
      </SettingSection>
    </>
  );
}

/** 画某一类媒体服务的地址、模型和密钥。props 读取 presets、kind、provider、fallbackPreset、model、fallbackModel 等，调用 onProvider，调用 onModel，调用 onVoice。 */
function MediaFields(props: {
  title: string;
  kind: "speech" | "asr" | "image";
  presets: ServicePreset[];
  provider: string;
  base: string;
  model: string;
  voice?: string;
  keyValue: string;
  keyHint: string;
  fallbackPreset: string;
  fallbackName: string;
  fallbackModel: string;
  onProvider: (id: string) => void;
  onBase: (value: string) => void;
  onModel: (model: string) => void;
  onVoice?: (voice: string) => void;
  onKey: (value: string) => void;
}) {
  const c = useCopy().idol;
  const list = props.presets.filter((p) => p.kind === props.kind);
  const selected = presetByID(props.presets, props.provider, props.kind);
  const hint = props.provider === "" ? presetByID(props.presets, props.fallbackPreset, props.kind) : undefined;
  const voiceModel = selected ? props.model : (props.fallbackModel || hint?.models[0] || "");
  const voices = props.kind === "speech" ? voicesFor(selected || hint, voiceModel) : [];
  return (
    <div className="flex flex-col gap-2 border-t border-border/70 pt-3">
      <SettingsGroupLabel>{props.title}</SettingsGroupLabel>
      <label className="block text-[12px] text-muted">
        {c.mediaProvider}
        <select className={fieldClass} value={props.provider} onChange={(e) => props.onProvider(e.target.value)}>
          <option value="">{c.useGeneration}</option>
          {props.provider === customProvider ? <option value={customProvider}>{c.customProvider}</option> : null}
          {list.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
        </select>
      </label>
      {props.provider === "" && (props.fallbackName || props.fallbackModel) ? (
        <p className="text-[11px] text-muted">{c.generationNow}{props.fallbackName ? `：${props.fallbackName}` : ""}{props.fallbackModel ? ` · ${props.fallbackModel}` : ""}</p>
      ) : null}
      {props.provider === customProvider ? (
        <Field label={c.customProvider} value={props.base} onChange={props.onBase} />
      ) : null}
      {selected ? (
        <label className="block text-[12px] text-muted">
          {c.mediaModel}
          <select className={fieldClass} value={props.model} onChange={(e) => props.onModel(e.target.value)}>
            {props.model && !selected.models.includes(props.model) ? <option value={props.model}>{props.model}</option> : null}
            {selected.models.map((m) => <option key={m} value={m}>{m}</option>)}
          </select>
        </label>
      ) : null}
      {props.provider === customProvider ? <Field label={c.mediaModel} value={props.model} onChange={props.onModel} /> : null}
      {props.kind === "speech" && voices.length > 0 ? (
        <VoiceSelect voices={voices} value={props.voice || ""} onChange={(v) => props.onVoice?.(v)} />
      ) : null}
      {props.kind === "speech" && voices.length === 0 ? (
        <Field label={c.mediaVoice} value={props.voice || ""} placeholder="alloy" onChange={(v) => props.onVoice?.(v)} />
      ) : null}
      {props.provider !== "" ? (
        <>
          <Field label={props.keyHint ? `${c.mediaKey} ${props.keyHint}` : c.mediaKey} value={props.keyValue} secret placeholder={c.mediaKeyPh} onChange={props.onKey} />
          <p className="text-[11px] leading-relaxed text-muted">{c.mediaKeyHint}</p>
        </>
      ) : null}
    </div>
  );
}

/** 从可用声线里选一条。props 读取 value、voices，调用 onChange。 */
function VoiceSelect(props: { voices: ServiceVoice[]; value: string; onChange: (voice: string) => void }) {
  const label = useCopy().idol.mediaVoice;
  const extra = props.value && !props.voices.some((v) => v.id === props.value);
  return (
    <label className="block text-[12px] text-muted">
      {label}
      <select className={fieldClass} value={props.value} onChange={(e) => props.onChange(e.target.value)}>
        {extra ? <option value={props.value}>{props.value}</option> : null}
        {props.voices.map((v) => <option key={v.id} value={v.id}>{v.label}</option>)}
      </select>
    </label>
  );
}

/** 画一行输入，密钥可以遮住。props 读取 label、secret、value、placeholder，调用 onChange。 */
function Field(props: { label: string; value: string; placeholder?: string; secret?: boolean; onChange: (v: string) => void }) {
  return (
    <label className="block text-[12px] text-muted">
      {props.label}
      <Input
        className="mt-1 h-8 font-mono text-[12px]"
        type={props.secret ? "password" : "text"}
        value={props.value}
        placeholder={props.placeholder}
        onChange={(e) => props.onChange(e.target.value)}
        autoComplete="off"
      />
    </label>
  );
}
