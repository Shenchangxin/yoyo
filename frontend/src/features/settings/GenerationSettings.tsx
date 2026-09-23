import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Switch } from "../../components/ui/switch";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import { ConfirmDialog } from "../ConfirmDialog";
import { cn } from "../../lib/utils";
import {
  CONTROL_LG,
  SettingActionRow,
  SettingEmpty,
  SettingRow,
  SettingSection,
  SettingSegmented,
  SettingsPageHeader,
} from "./SettingChrome";
import { ProviderMark } from "./ProviderMark";

type Kind = "image" | "video" | "tts";

type MediaProvider = {
  id: string;
  service_type: Kind | string;
  provider: string;
  name: string;
  base_url: string;
  model: string;
  models: string;
  is_default?: boolean;
  is_active?: boolean;
  has_key?: boolean;
  priority?: number;
};

type StyleRow = { id: string; name: string; value: string; prompt: string; is_active?: boolean };

type Draft = {
  id?: string;
  service_type: Kind;
  provider: string;
  name: string;
  base_url: string;
  model: string;
  models: string;
  api_key: string;
  is_default: boolean;
  is_active: boolean;
  has_key?: boolean;
};

const VENDOR_LABEL: Record<string, string> = {
  volcengine: "Volcengine",
  openai: "OpenAI",
  gemini: "Gemini",
  minimax: "MiniMax",
  aliyun: "Alibaba Cloud",
  custom: "Custom",
};

function parseModels(raw: string): string[] {
  const s = (raw || "").trim();
  if (!s) return [];
  if (s.startsWith("[")) {
    try {
      const v = JSON.parse(s);
      if (Array.isArray(v)) return v.map(String).filter(Boolean);
    } catch {
      /* fall through */
    }
  }
  return s.split(",").map((x) => x.trim()).filter(Boolean);
}

function hostOf(url: string): string {
  const raw = (url || "").trim();
  if (!raw) return "";
  try {
    return new URL(raw.includes("://") ? raw : `https://${raw}`).host;
  } catch {
    return raw.replace(/^https?:\/\//, "").split("/")[0] || raw;
  }
}

function vendorLabel(id: string): string {
  return VENDOR_LABEL[id] || id || "Custom";
}

function emptyDraft(kind: Kind): Draft {
  return {
    service_type: kind,
    provider: "custom",
    name: "",
    base_url: "",
    model: "",
    models: "",
    api_key: "",
    is_default: false,
    is_active: true,
  };
}

function fromRow(p: MediaProvider): Draft {
  return {
    id: p.id,
    service_type: (p.service_type as Kind) || "image",
    provider: p.provider,
    name: p.name,
    base_url: p.base_url,
    model: p.model,
    models: parseModels(p.models).join(", "),
    api_key: "",
    is_default: !!p.is_default,
    is_active: p.is_active !== false,
    has_key: !!p.has_key,
  };
}

function fromTemplate(kind: Kind, t: MediaProvider): Draft {
  return {
    ...emptyDraft(kind),
    provider: t.provider,
    name: t.name,
    base_url: t.base_url,
    model: t.model,
    models: parseModels(t.models).join(", "),
  };
}

function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <label className="flex w-full flex-col gap-[var(--space-item)] text-[13px] font-medium text-foreground">
      <span>
        {label}
        {hint ? <span className="mt-0.5 block text-[12px] font-normal leading-[1.45] text-muted">{hint}</span> : null}
      </span>
      {children}
    </label>
  );
}

function StatusPill({ ok, label }: { ok: boolean; label: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11.5px] font-medium",
        ok ? "bg-success/15 text-success" : "bg-danger/10 text-danger",
      )}
    >
      <span className={cn("size-1.5 rounded-full", ok ? "bg-success" : "bg-danger")} aria-hidden />
      {label}
    </span>
  );
}

export function GenerationSettings() {
  const copy = useCopy();
  const v = copy.video;
  const [templates, setTemplates] = useState<MediaProvider[]>([]);
  const [providers, setProviders] = useState<MediaProvider[]>([]);
  const [styles, setStyles] = useState<StyleRow[]>([]);
  const [lang, setLang] = useState("zh");
  const [ffmpeg, setFfmpeg] = useState(true);
  const [tick, setTick] = useState(0);

  function refresh() {
    setTick((n) => n + 1);
  }

  useEffect(() => {
    void api.video.templates().then((t) => setTemplates(Array.isArray(t) ? t : [])).catch(() => {});
    void api.video.providers().then((p) => setProviders(Array.isArray(p) ? p : [])).catch(() => {});
    void api.video.allStyles().then((s) => setStyles(Array.isArray(s) ? s : [])).catch(() => {
      void api.video.styles().then((s) => setStyles(Array.isArray(s) ? s : [])).catch(() => {});
    });
    void api.video.settings().then((s) => setLang(s?.content_language || "zh")).catch(() => {});
    void api.video.status().then((s) => setFfmpeg(!!s?.ffmpeg)).catch(() => {});
  }, [tick]);

  const langChoice = lang === "zh" || lang === "zh-CN" ? "zh" : lang === "en" || lang === "en-US" ? "en" : "other";

  async function setLanguage(next: string) {
    setLang(next);
    try {
      await api.video.setSetting("content_language", next);
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  return (
    <>
      <SettingsPageHeader title={copy.settings.tabs.generation} description={copy.settings.tabHints.generation} />

      <SettingSection
        id="generation-studio"
        title={copy.settings.sections.generationStudio}
        footnote={v.dramaHint}
      >
        <SettingRow title={v.language}>
          <SettingSegmented
            id="content-language"
            ariaLabel={v.language}
            value={langChoice}
            options={[
              { value: "zh", label: v.langZh },
              { value: "en", label: v.langEn },
              { value: "other", label: v.langOther },
            ]}
            onChange={(next) => {
              if (next === "other") {
                if (langChoice !== "other") setLang("");
                return;
              }
              void setLanguage(next);
            }}
          />
        </SettingRow>
        {langChoice === "other" ? (
          <SettingRow title={v.langOther}>
            <Input
              className={CONTROL_LG}
              value={lang}
              aria-label={v.language}
              onChange={(e) => setLang(e.target.value)}
              onBlur={() => {
                const next = lang.trim();
                if (next) void setLanguage(next);
              }}
            />
          </SettingRow>
        ) : null}
        <SettingRow title={v.ffmpeg} border={false} list>
          <StatusPill ok={ffmpeg} label={ffmpeg ? v.ffmpegReady : v.ffmpegMissing} />
        </SettingRow>
      </SettingSection>

      <AdapterSection
        id="generation-image"
        title={copy.settings.sections.generationImage}
        kind="image"
        rows={providers}
        templates={templates}
        copy={copy}
        onChanged={refresh}
      />
      <AdapterSection
        id="generation-video"
        title={copy.settings.sections.generationVideo}
        kind="video"
        rows={providers}
        templates={templates}
        copy={copy}
        onChanged={refresh}
      />
      <AdapterSection
        id="generation-speech"
        title={copy.settings.sections.generationSpeech}
        kind="tts"
        rows={providers}
        templates={templates}
        copy={copy}
        onChanged={refresh}
        footnote={v.speechHint}
      />

      <StylesSection styles={styles} copy={copy} onChanged={refresh} />
    </>
  );
}

function AdapterSection(props: {
  id: string;
  title: string;
  kind: Kind;
  rows: MediaProvider[];
  templates: MediaProvider[];
  copy: ReturnType<typeof useCopy>;
  onChanged: () => void;
  footnote?: string;
}) {
  const v = props.copy.video;
  const rows = useMemo(
    () => props.rows.filter((p) => p.service_type === props.kind),
    [props.rows, props.kind],
  );
  const kindTemplates = useMemo(
    () => props.templates.filter((t) => t.service_type === props.kind),
    [props.templates, props.kind],
  );
  const [draft, setDraft] = useState<Draft | null>(null);
  const [picking, setPicking] = useState(false);
  const [busy, setBusy] = useState("");
  const [pending, setPending] = useState<MediaProvider | null>(null);

  async function save(d: Draft) {
    setBusy("save");
    try {
      const models = parseModels(d.models);
      await api.video.upsertProvider(
        {
          id: d.id,
          service_type: d.service_type,
          provider: d.provider || "custom",
          name: d.name || d.provider,
          base_url: d.base_url,
          model: d.model,
          models: JSON.stringify(models.length ? models : (d.model ? [d.model] : [])),
          is_default: d.is_default,
          is_active: d.is_active,
        },
        d.api_key,
      );
      setDraft(null);
      setPicking(false);
      props.onChanged();
      toast.success(props.copy.app.controlSaved);
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  async function patchRow(p: MediaProvider, extra: { is_active?: boolean; is_default?: boolean; model?: string }) {
    try {
      const model = extra.model ?? p.model;
      const models = parseModels(p.models);
      if (model && !models.includes(model)) models.unshift(model);
      await api.video.upsertProvider(
        {
          id: p.id,
          service_type: p.service_type,
          provider: p.provider,
          name: p.name,
          base_url: p.base_url,
          model,
          models: JSON.stringify(models.length ? models : (model ? [model] : [])),
          is_default: extra.is_default ?? !!p.is_default,
          is_active: extra.is_active ?? p.is_active !== false,
          priority: p.priority,
        },
        "",
      );
      props.onChanged();
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  async function test(id: string) {
    setBusy("test:" + id);
    try {
      const r = await api.video.testProvider(id);
      if (r?.ok) toast.success(props.copy.settings.testOk);
      else toast.error(String(r?.error || props.copy.settings.testFail));
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  const showPicker = picking || (rows.length === 0 && !draft);

  return (
    <SettingSection id={props.id} title={props.title} footnote={props.footnote || v.multiAdapterHint}>
      {rows.length === 0 && !draft ? <SettingEmpty>{v.noAdapter}</SettingEmpty> : null}

      {rows.map((p) => {
        const open = draft?.id === p.id;
        const models = parseModels(p.models);
        if (p.model && !models.includes(p.model)) models.unshift(p.model);
        return (
          <div key={p.id} className={cn("rounded-xl px-1", open && "bg-background/40")}>
            <div className="flex items-center gap-3 py-1">
              <ProviderMark id={p.provider} className="size-8" />
              <button
                type="button"
                className="min-w-0 flex-1 text-left"
                onClick={() => {
                  setPicking(false);
                  setDraft(open ? null : fromRow(p));
                }}
              >
                <div className="truncate text-[13px] font-medium leading-[1.4] text-foreground">{p.name}</div>
                <div className="mt-0.5 truncate font-mono text-[11px] leading-[1.4] text-muted">
                  {[vendorLabel(p.provider), hostOf(p.base_url)].filter(Boolean).join(" · ")}
                </div>
              </button>
              <div className="flex shrink-0 items-center gap-1.5">
                {p.is_default ? (
                  <span className="rounded-full bg-lift px-2 py-0.5 text-[11px] font-medium text-foreground">{v.makeDefault}</span>
                ) : (
                  <button
                    type="button"
                    className="rounded-full px-2 py-0.5 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground"
                    onClick={() => void patchRow(p, { is_default: true })}
                  >
                    {v.setDefault}
                  </button>
                )}
                <span
                  className={cn(
                    "rounded-full px-2 py-0.5 text-[11px] font-medium",
                    p.has_key ? "bg-success/15 text-success" : "bg-danger/10 text-danger",
                  )}
                >
                  {p.has_key ? v.keySaved : v.noKey}
                </span>
                <div
                  onClick={(e) => e.stopPropagation()}
                  onPointerDown={(e) => e.stopPropagation()}
                >
                  <Switch
                    checked={p.is_active !== false}
                    aria-label={v.enabled}
                    onCheckedChange={(on) => void patchRow(p, { is_active: on })}
                  />
                </div>
              </div>
            </div>
            {models.length ? (
                <div className="flex flex-wrap gap-1 pb-1.5 pl-11" role="group" aria-label={v.defaultModel}>
                  {models.map((m) => {
                    const on = m === p.model;
                    return (
                      <button
                        key={m}
                        type="button"
                        title={v.defaultModel}
                        aria-pressed={on}
                        className={cn(
                          "max-w-[14rem] truncate rounded-full px-2 py-0.5 font-mono text-[10.5px]",
                          on ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70 hover:text-foreground",
                        )}
                        onClick={() => void patchRow(p, { model: m })}
                      >
                        {m}
                      </button>
                    );
                  })}
                </div>
            ) : null}
            {open && draft ? (
              <AdapterForm
                draft={draft}
                setDraft={setDraft}
                copy={props.copy}
                busy={busy}
                onSave={() => void save(draft)}
                onTest={() => void test(p.id)}
                onDelete={() => setPending(p)}
              />
            ) : null}
          </div>
        );
      })}

      {draft && !draft.id ? (
        <AdapterForm
          draft={draft}
          setDraft={(d) => {
            setDraft(d);
            if (!d) setPicking(false);
          }}
          copy={props.copy}
          busy={busy}
          onSave={() => void save(draft)}
        />
      ) : null}

      {showPicker && !draft ? (
        <div className="flex flex-col gap-0.5">
          <p className="px-1 pb-1 text-[12px] text-muted">{v.templateHint}</p>
          {kindTemplates.map((t) => (
            <button
              key={t.provider + t.service_type + t.model}
              type="button"
              className="flex min-h-10 w-full items-center gap-3 rounded-md px-1 text-left transition-colors hover:bg-lift/70"
              onClick={() => {
                setDraft(fromTemplate(props.kind, t));
                setPicking(false);
              }}
            >
              <ProviderMark id={t.provider} className="size-8" />
              <span className="min-w-0 flex-1">
                <span className="block text-[13px] font-medium leading-[1.4] text-foreground">{t.name}</span>
                <span className="mt-0.5 block truncate font-mono text-[11.5px] leading-[1.5] text-muted">
                  {vendorLabel(t.provider)} · {t.model}
                </span>
              </span>
            </button>
          ))}
          <SettingActionRow
            border={false}
            chevron={false}
            title={v.customProvider}
            trailing={<Plus className="size-4" aria-hidden />}
            onClick={() => {
              setDraft(emptyDraft(props.kind));
              setPicking(false);
            }}
          />
        </div>
      ) : rows.length > 0 && !draft ? (
        <SettingActionRow
          border={false}
          chevron={false}
          title={v.addAdapter}
          trailing={<Plus className="size-4" aria-hidden />}
          onClick={() => setPicking(true)}
        />
      ) : null}

      <ConfirmDialog
        open={!!pending}
        title={v.delete}
        body={v.deleteAdapter}
        danger
        confirmLabel={v.delete}
        onCancel={() => setPending(null)}
        onConfirm={async () => {
          if (!pending) return;
          await api.video.deleteProvider(pending.id);
          if (draft?.id === pending.id) setDraft(null);
          setPending(null);
          props.onChanged();
        }}
      />
    </SettingSection>
  );
}

function AdapterForm({
  draft,
  setDraft,
  copy,
  busy,
  onSave,
  onTest,
  onDelete,
}: {
  draft: Draft;
  setDraft: (d: Draft | null) => void;
  copy: ReturnType<typeof useCopy>;
  busy: string;
  onSave: () => void;
  onTest?: () => void;
  onDelete?: () => void;
}) {
  const v = copy.video;
  const models = parseModels(draft.models);
  const patch = (p: Partial<Draft>) => setDraft({ ...draft, ...p });
  return (
    <div className="flex flex-col gap-[var(--space-group)] border-t border-border/60 px-1 py-3">
      <Field label={v.adapterName}>
        <Input className={CONTROL_LG} value={draft.name} onChange={(e) => patch({ name: e.target.value })} />
      </Field>
      <Field label={v.vendor}>
        <Input className={cn(CONTROL_LG, "font-mono")} value={draft.provider} onChange={(e) => patch({ provider: e.target.value })} />
      </Field>
      <Field label={copy.settings.baseUrl}>
        <Input className={cn(CONTROL_LG, "font-mono")} value={draft.base_url} onChange={(e) => patch({ base_url: e.target.value })} />
      </Field>
      <Field label={copy.settings.model}>
        {models.length ? (
          <Select value={draft.model || models[0]} onValueChange={(value) => patch({ model: value })}>
            <SelectTrigger className="h-8 w-full font-mono">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {models.map((m) => (
                <SelectItem key={m} value={m}>{m}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <Input className={cn(CONTROL_LG, "font-mono")} value={draft.model} onChange={(e) => patch({ model: e.target.value })} />
        )}
      </Field>
      <Field label={v.modelsCsv}>
        <Input className={cn(CONTROL_LG, "font-mono")} value={draft.models} onChange={(e) => patch({ models: e.target.value })} />
      </Field>
      <Field label={copy.settings.apiKey} hint={draft.has_key ? v.keySaved : copy.settings.apiKeyPh}>
        <Input
          className={cn(CONTROL_LG, "font-mono")}
          type="password"
          autoComplete="off"
          placeholder={draft.id && draft.has_key ? "••••••••" : copy.settings.apiKeyPh}
          value={draft.api_key}
          onChange={(e) => patch({ api_key: e.target.value })}
        />
      </Field>
      <SettingRow title={v.enabled} list>
        <Switch checked={draft.is_active} onCheckedChange={(on) => patch({ is_active: on })} />
      </SettingRow>
      <SettingRow title={v.makeDefault} list>
        <Switch checked={draft.is_default} onCheckedChange={(on) => patch({ is_default: on })} />
      </SettingRow>
      <div className="flex flex-wrap items-center justify-end gap-2">
        {onDelete ? (
          <Button size="sm" variant="ghost" onClick={onDelete}>{v.delete}</Button>
        ) : (
          <Button size="sm" variant="ghost" onClick={() => setDraft(null)}>{copy.settings.discard}</Button>
        )}
        {onTest && draft.id ? (
          <Button size="sm" variant="lift" disabled={!!busy} onClick={onTest}>{copy.settings.testConnection}</Button>
        ) : null}
        <Button size="sm" disabled={!!busy || !draft.name.trim() || !draft.base_url.trim()} onClick={onSave}>
          {copy.settings.saveChanges}
        </Button>
      </div>
    </div>
  );
}

function StylesSection({
  styles,
  copy,
  onChanged,
}: {
  styles: StyleRow[];
  copy: ReturnType<typeof useCopy>;
  onChanged: () => void;
}) {
  const v = copy.video;
  const [draft, setDraft] = useState({ name: "", value: "", prompt: "" });
  const [adding, setAdding] = useState(false);
  const [pending, setPending] = useState<StyleRow | null>(null);

  return (
    <SettingSection id="generation-styles" title={copy.settings.sections.generationStyles} footnote={v.styleHint}>
      {styles.length === 0 ? <SettingEmpty>{v.noAdapter}</SettingEmpty> : null}
      {styles.map((s) => (
        <SettingRow
          key={s.id}
          list
          title={s.name}
          description={s.is_active === false ? `${s.value} · ${v.styleHidden}` : s.value}
        >
          {s.is_active === false ? (
            <Button
              size="sm"
              variant="ghost"
              onClick={async () => {
                await api.video.upsertStyle({ ...s, is_active: true });
                onChanged();
              }}
            >
              {v.restoreStyle}
            </Button>
          ) : (
            <Button size="sm" variant="ghost" onClick={() => setPending(s)}>
              {v.hideStyle}
            </Button>
          )}
        </SettingRow>
      ))}
      {adding ? (
        <div className="flex flex-col gap-[var(--space-group)]">
          <Field label={v.styleName}>
            <Input className={CONTROL_LG} value={draft.name} onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))} />
          </Field>
          <Field label={v.styleValue}>
            <Input className={cn(CONTROL_LG, "font-mono")} value={draft.value} onChange={(e) => setDraft((d) => ({ ...d, value: e.target.value }))} />
          </Field>
          <Field label={v.finalPrompt}>
            <Input className={CONTROL_LG} value={draft.prompt} onChange={(e) => setDraft((d) => ({ ...d, prompt: e.target.value }))} />
          </Field>
          <div className="flex justify-end gap-2">
            <Button size="sm" variant="ghost" onClick={() => { setAdding(false); setDraft({ name: "", value: "", prompt: "" }); }}>
              {copy.settings.discard}
            </Button>
            <Button
              size="sm"
              disabled={!draft.name.trim() || !draft.value.trim()}
              onClick={async () => {
                await api.video.upsertStyle({ name: draft.name, value: draft.value, prompt: draft.prompt, is_active: true });
                setDraft({ name: "", value: "", prompt: "" });
                setAdding(false);
                onChanged();
                toast.success(copy.app.controlSaved);
              }}
            >
              {v.addStyle}
            </Button>
          </div>
        </div>
      ) : (
        <SettingActionRow
          border={false}
          chevron={false}
          title={v.addStyle}
          trailing={<Plus className="size-4" aria-hidden />}
          onClick={() => setAdding(true)}
        />
      )}
      <ConfirmDialog
        open={!!pending}
        title={v.hideStyle}
        body={v.styleHint}
        confirmLabel={v.hideStyle}
        onCancel={() => setPending(null)}
        onConfirm={async () => {
          if (!pending) return;
          await api.video.deleteStyle(pending.id);
          setPending(null);
          onChanged();
        }}
      />
    </SettingSection>
  );
}
