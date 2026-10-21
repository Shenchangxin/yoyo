import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Switch } from "../../components/ui/switch";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { asArray, str } from "../../lib/normalize";
import { SettingRow, SettingSection } from "./SettingChrome";

type Form = {
  chat_base_url: string;
  chat_model: string;
  judge_model: string;
  speech_base_url: string;
  speech_model: string;
  speech_voice: string;
  asr_base_url: string;
  asr_model: string;
  image_base_url: string;
  image_model: string;
  reach_out: boolean;
  text_only: boolean;
  phone_texture: boolean;
  work_enabled: boolean;
  work_profile_id: string;
  chat_key: string;
  speech_key: string;
  asr_key: string;
  image_key: string;
  address: string;
  gender: string;
  timezone: string;
  birthday: string;
  chat_key_hint: string;
  speech_key_hint: string;
  asr_key_hint: string;
  image_key_hint: string;
};

const empty: Form = {
  chat_base_url: "", chat_model: "", judge_model: "",
  speech_base_url: "", speech_model: "", speech_voice: "",
  asr_base_url: "", asr_model: "",
  image_base_url: "", image_model: "",
  reach_out: true, text_only: false, phone_texture: false, work_enabled: true, work_profile_id: "",
  chat_key: "", speech_key: "", asr_key: "", image_key: "",
  address: "", gender: "", timezone: "", birthday: "",
  chat_key_hint: "", speech_key_hint: "", asr_key_hint: "", image_key_hint: "",
};

function fromRaw(v: any): Form {
  const u = v?.user || {};
  return {
    ...empty,
    chat_base_url: str(v?.chat_base_url),
    chat_model: str(v?.chat_model),
    judge_model: str(v?.judge_model),
    speech_base_url: str(v?.speech_base_url),
    speech_model: str(v?.speech_model),
    speech_voice: str(v?.speech_voice),
    asr_base_url: str(v?.asr_base_url),
    asr_model: str(v?.asr_model),
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
    chat_key_hint: str(v?.chat_key_hint),
    speech_key_hint: str(v?.speech_key_hint),
    asr_key_hint: str(v?.asr_key_hint),
    image_key_hint: str(v?.image_key_hint),
  };
}

export function IdolSettings() {
  const copy = useCopy();
  const c = copy.idol;
  const [form, setForm] = useState<Form>(empty);
  const [profiles, setProfiles] = useState<{ id: string; name: string }[]>([]);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void api.idolCall("settings_get").then((v) => setForm(fromRaw(v))).catch((e) => toast.error(String(e?.message || e)));
    void api.idolCall("profiles").then((v) => {
      setProfiles(asArray(v).map((p: any) => ({ id: str(p.id || p.ID), name: str(p.name || p.Name) })).filter((p) => p.id));
    }).catch(() => {});
  }, []);

  const set = (patch: Partial<Form>) => setForm((f) => ({ ...f, ...patch }));

  async function save() {
    setBusy(true);
    try {
      const next = await api.idolCall("settings_save", {
        chat_base_url: form.chat_base_url,
        chat_model: form.chat_model,
        judge_model: form.judge_model,
        speech_base_url: form.speech_base_url,
        speech_model: form.speech_model,
        speech_voice: form.speech_voice,
        asr_base_url: form.asr_base_url,
        asr_model: form.asr_model,
        image_base_url: form.image_base_url,
        image_model: form.image_model,
        reach_out: form.reach_out,
        text_only: form.text_only,
        phone_texture: form.phone_texture,
        work_enabled: form.work_enabled,
        work_profile_id: form.work_profile_id,
        chat_key: form.chat_key,
        speech_key: form.speech_key,
        asr_key: form.asr_key,
        image_key: form.image_key,
        user: { address: form.address, gender: form.gender, timezone: form.timezone, birthday: form.birthday },
      });
      setForm(fromRaw(next));
      toast.success(copy.app.controlSaved);
    } catch (e: any) {
      toast.error(String(e?.message || e));
    } finally {
      setBusy(false);
    }
  }

  async function ping(kind: string) {
    try {
      await api.idolCall("ping", { kind });
      toast.success("OK");
    } catch (e: any) {
      toast.error(String(e?.message || e));
    }
  }

  return (
    <>
      <SettingSection id="idol-services" title={copy.settings.sections.idolServices} description={copy.settings.tabHints.idol}>
        <Field label="Chat base URL" value={form.chat_base_url} placeholder="empty = Models" onChange={(v) => set({ chat_base_url: v })} />
        <Field label="Chat model" value={form.chat_model} onChange={(v) => set({ chat_model: v })} />
        <Field label="Judge model" value={form.judge_model} placeholder="empty = chat model" onChange={(v) => set({ judge_model: v })} />
        <Field label={`Chat key ${form.chat_key_hint}`} value={form.chat_key} secret placeholder="unchanged, - clears" onChange={(v) => set({ chat_key: v })} />
        <Field label="Speech base URL" value={form.speech_base_url} placeholder="empty = Generation → Speech" onChange={(v) => set({ speech_base_url: v })} />
        <Field label="Speech model" value={form.speech_model} onChange={(v) => set({ speech_model: v })} />
        <Field label="Default voice" value={form.speech_voice} placeholder="alloy" onChange={(v) => set({ speech_voice: v })} />
        <Field label={`Speech key ${form.speech_key_hint}`} value={form.speech_key} secret onChange={(v) => set({ speech_key: v })} />
        <Field label="ASR base URL" value={form.asr_base_url} placeholder="empty = Generation → Transcription" onChange={(v) => set({ asr_base_url: v })} />
        <Field label="ASR model" value={form.asr_model} onChange={(v) => set({ asr_model: v })} />
        <Field label={`ASR key ${form.asr_key_hint}`} value={form.asr_key} secret onChange={(v) => set({ asr_key: v })} />
        <Field label="Image base URL" value={form.image_base_url} placeholder="OpenAI-compatible /images/generations" onChange={(v) => set({ image_base_url: v })} />
        <Field label="Image model" value={form.image_model} onChange={(v) => set({ image_model: v })} />
        <Field label={`Image key ${form.image_key_hint}`} value={form.image_key} secret onChange={(v) => set({ image_key: v })} />
        <div className="flex flex-wrap gap-2 pt-2">
          <Button size="sm" variant="lift" onClick={() => void ping("chat")}>Chat</Button>
          <Button size="sm" variant="lift" onClick={() => void ping("speech")}>Speech</Button>
          <Button size="sm" variant="lift" onClick={() => void ping("image")}>Image</Button>
        </div>
      </SettingSection>
      <SettingSection id="idol-you" title={copy.settings.sections.idolYou}>
        <Field label={c.name} value={form.address} onChange={(v) => set({ address: v })} />
        <Field label="Gender" value={form.gender} onChange={(v) => set({ gender: v })} />
        <Field label="Timezone" value={form.timezone} placeholder="Asia/Shanghai" onChange={(v) => set({ timezone: v })} />
        <Field label="Birthday MM-DD" value={form.birthday} placeholder="03-21" onChange={(v) => set({ birthday: v })} />
      </SettingSection>
      <SettingSection id="idol-work" title={copy.settings.sections.idolWork} description={c.workHint}>
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
        <label className="mt-3 block text-[12px] text-muted">
          Profile
          <select
            className="mt-1 h-8 w-full rounded-md border border-border bg-background px-2 text-[13px]"
            value={form.work_profile_id}
            onChange={(e) => set({ work_profile_id: e.target.value })}
          >
            <option value="">assistant</option>
            {profiles.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
          </select>
        </label>
        <div className="pt-3">
          <Button size="sm" disabled={busy} onClick={() => void save()}>{c.save}</Button>
        </div>
      </SettingSection>
    </>
  );
}

function Field(props: { label: string; value: string; placeholder?: string; secret?: boolean; onChange: (v: string) => void }) {
  return (
    <label className="mb-2 block text-[12px] text-muted">
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
