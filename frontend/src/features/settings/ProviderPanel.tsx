import { useEffect, useMemo, useRef, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { toast } from "sonner";
import { Check, ChevronDown, Plus, RefreshCw, X } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Switch } from "../../components/ui/switch";
import { cn } from "../../lib/utils";
import { applyProviderPreset, PROVIDER_PRESETS, providerPreset } from "../../lib/providers";
import { formatTokens, mergeModelIds, modelsForProvider, type ModelsDevCatalog } from "../../lib/models-dev";
import { useModelCatalog } from "../../lib/model-catalog";
import type { ChatProvider } from "../../lib/protocol";
import { ConfirmDialog } from "../ConfirmDialog";
import { DURATION, motionTransition, useMotionReduced } from "../../lib/motion";
import {
  CONTROL_LG,
  SettingRow,
  SettingSection,
  SettingsPageHeader,
  SettingsSaveBar,
} from "./SettingChrome";
import { ProviderMark } from "./ProviderMark";
import type { SettingsHost } from "./host";

type Draft = {
  id?: string;
  name: string;
  vendor: string;
  endpoint: string;
  model: string;
  models: string[];
  apiKey: string;
  makeDefault: boolean;
  hasKey?: boolean;
};

function hostOf(url: string): string {
  const raw = (url || "").trim();
  if (!raw) return "";
  try {
    return new URL(raw.includes("://") ? raw : `https://${raw}`).host;
  } catch {
    return raw.replace(/^https?:\/\//, "").split("/")[0] || raw;
  }
}

function fromRow(p: ChatProvider): Draft {
  return {
    id: p.id,
    name: p.name,
    vendor: p.vendor || "custom",
    endpoint: p.endpoint,
    model: p.model,
    models: mergeModelIds(p.model, p.models),
    apiKey: "",
    makeDefault: !!p.isDefault,
    hasKey: p.hasKey,
  };
}

function fromPreset(id: string, catalog: ModelsDevCatalog | null, makeDefault: boolean): Draft {
  const preset = providerPreset(id);
  const next = applyProviderPreset(id);
  const vendor = preset?.id || "custom";
  const model = next?.model || preset?.model || "";
  const models = mergeModelIds(model, next?.models || preset?.models);
  const cats = modelsForProvider(catalog, vendor);
  const pick = cats.find((m) => m.id === model) || cats.find((m) => models.includes(m.id)) || cats[0];
  return {
    name: preset?.label || vendor,
    vendor,
    endpoint: next?.baseUrl || preset?.baseUrl || "",
    model: pick?.id || model,
    models: mergeModelIds(pick?.id, models),
    apiKey: "",
    makeDefault,
    hasKey: false,
  };
}

function sameDraft(a: Draft, b: Draft): boolean {
  return (
    a.name === b.name &&
    a.vendor === b.vendor &&
    a.endpoint === b.endpoint &&
    a.model === b.model &&
    a.makeDefault === b.makeDefault &&
    a.apiKey === b.apiKey &&
    a.models.join(",") === b.models.join(",")
  );
}

export function ProviderSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const reduced = useMotionReduced();
  const fade = motionTransition(reduced, DURATION);
  const [rows, setRows] = useState<ChatProvider[]>(host.health.chatProviders || []);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [baseline, setBaseline] = useState<Draft | null>(null);
  const [picking, setPicking] = useState(false);
  const [busy, setBusy] = useState("");
  const [pending, setPending] = useState<ChatProvider | null>(null);
  const catalog = useModelCatalog((s) => s.catalog);
  const selected = draft?.id ? rows.find((p) => p.id === draft.id) : undefined;
  const dirty = !!draft && (!baseline || !sameDraft(draft, baseline));
  const empty = rows.length === 0;

  async function refresh() {
    try {
      setRows(await api.listChatProviders());
    } catch (e) {
      toast.error(api.errMessage(e));
    }
    await host.refresh?.();
  }

  useEffect(() => {
    void refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- boot once
  }, []);

  function openDraft(next: Draft) {
    setPicking(false);
    setDraft(next);
    setBaseline({ ...next, models: [...next.models] });
  }

  function closeEditor() {
    setDraft(null);
    setBaseline(null);
    setPicking(false);
  }

  function startAdd() {
    setDraft(null);
    setBaseline(null);
    setPicking(true);
  }

  async function save(d: Draft, keepOpen = false): Promise<ChatProvider | null> {
    setBusy("save");
    try {
      const models = mergeModelIds(d.model, d.models);
      const out = await api.upsertChatProvider(
        {
          id: d.id,
          name: (d.name.trim() || providerPreset(d.vendor)?.label || d.vendor).trim(),
          vendor: d.vendor || "custom",
          endpoint: d.endpoint.trim().replace(/\/+$/, ""),
          model: d.model.trim() || models[0] || "",
          models,
          make_default: d.makeDefault || rows.length === 0,
          active: true,
        },
        d.apiKey,
      );
      toast.success(copy.app.controlSaved);
      await refresh();
      if (keepOpen) {
        const next = fromRow(out);
        setDraft(next);
        setBaseline(next);
      } else {
        closeEditor();
      }
      return out;
    } catch (e) {
      toast.error(api.errMessage(e));
      return null;
    } finally {
      setBusy("");
    }
  }

  async function testCurrent() {
    if (!draft) return;
    let id = draft.id;
    if (dirty || !id) {
      const saved = await save(draft, true);
      if (!saved) return;
      id = saved.id;
    }
    setBusy("test");
    try {
      const r = await api.testProvider(id);
      if (r?.ok) toast.success(copy.settings.testOk);
      else toast.error(String(r?.error || copy.settings.testFail));
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  async function setDefault(p: ChatProvider) {
    try {
      await api.setDefaultChatProvider(p.id);
      await refresh();
      if (draft?.id === p.id) {
        const next = { ...draft, makeDefault: true };
        setDraft(next);
        if (baseline) setBaseline({ ...baseline, makeDefault: true });
      }
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  const showPicker = picking || empty;

  return (
    <>
      <SettingsPageHeader
        title={copy.settings.tabs.provider}
        description={copy.settings.tabHints.provider}
        actions={
          empty ? null : (
            <Button size="sm" variant="lift" onClick={startAdd}>
              <Plus aria-hidden />
              {copy.settings.addProvider}
            </Button>
          )
        }
      />

      {empty ? null : (
        <SettingSection id="provider-account" title={copy.settings.sections.providerAccount}>
          {rows.map((p) => {
            const open = draft?.id === p.id;
            return (
              <div key={p.id} className={cn("-mx-1 rounded-[10px] px-1", open && "bg-lift/55")}>
                <div className="flex items-center gap-3 py-2">
                  <button
                    type="button"
                    className="flex min-w-0 flex-1 items-center gap-3 text-left"
                    aria-expanded={open}
                    onClick={() => {
                      if (open) closeEditor();
                      else openDraft(fromRow(p));
                    }}
                  >
                    <ProviderMark id={p.vendor} className="size-8" />
                    <span className="min-w-0 flex-1">
                      <span className="flex items-center gap-2">
                        <span className="truncate text-[13px] font-medium leading-[1.4] text-foreground">{p.name || p.vendor}</span>
                        {p.isDefault ? (
                          <span className="shrink-0 rounded-full bg-background px-1.5 py-px text-[10px] font-medium text-muted">
                            {copy.settings.defaultBadge}
                          </span>
                        ) : null}
                      </span>
                      <span className="mt-0.5 block truncate text-[12px] leading-[1.4] text-muted">
                        {[p.model, hostOf(p.endpoint)].filter(Boolean).join(" · ") || hostOf(p.endpoint) || "—"}
                      </span>
                    </span>
                    <ChevronDown
                      className={cn("size-4 shrink-0 text-muted/70 transition-transform duration-200", open ? "rotate-0" : "-rotate-90")}
                      aria-hidden
                    />
                  </button>
                  <div className="flex shrink-0 items-center gap-1">
                    {p.hasKey ? null : (
                      <span className="rounded-full bg-danger/10 px-2 py-0.5 text-[11px] font-medium text-danger">
                        {copy.settings.noKey}
                      </span>
                    )}
                    {p.isDefault ? null : (
                      <button
                        type="button"
                        className="rounded-full px-2 py-0.5 text-[11px] font-medium text-muted hover:bg-background hover:text-foreground"
                        onClick={() => void setDefault(p)}
                      >
                        {copy.settings.setDefault}
                      </button>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </SettingSection>
      )}

      <AnimatePresence mode="wait">
        {showPicker ? (
          <motion.div key="picker" initial={reduced ? false : { opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }} exit={reduced ? undefined : { opacity: 0, y: 6 }} transition={fade}>
            <SettingSection
              id="provider-pick"
              title={copy.settings.pickVendor}
              description={copy.settings.providerTemplateHint}
            >
              <div className="grid grid-cols-2 gap-1.5 py-2 sm:grid-cols-3">
                {PROVIDER_PRESETS.map((p) => (
                  <button
                    type="button"
                    key={p.id}
                    onClick={() => openDraft(fromPreset(p.id, catalog, empty))}
                    className="flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-left transition-colors hover:bg-lift/70"
                  >
                    <ProviderMark id={p.id} className="size-8" />
                    <span className="min-w-0">
                      <span className="block truncate text-[13px] font-medium text-foreground">{p.label}</span>
                      <span className="block truncate text-[11px] text-muted">{p.id === "custom" ? copy.settings.customProvider : p.model || hostOf(p.baseUrl)}</span>
                    </span>
                  </button>
                ))}
              </div>
              {empty ? null : (
                <div className="flex justify-end pb-2">
                  <Button size="sm" variant="ghost" onClick={() => setPicking(false)}>
                    {copy.settings.backToList}
                  </Button>
                </div>
              )}
            </SettingSection>
          </motion.div>
        ) : draft ? (
          <motion.div key={draft.id || "new"} initial={reduced ? false : { opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }} exit={reduced ? undefined : { opacity: 0, y: 6 }} transition={fade}>
            <ProviderEditor
              draft={draft}
              setDraft={setDraft}
              selected={selected}
              busy={busy}
              canDelete={!!draft.id && rows.length > 1}
              onTest={() => void testCurrent()}
              onDelete={() => {
                const row = rows.find((p) => p.id === draft.id);
                if (row) setPending(row);
              }}
              onClose={closeEditor}
            />
          </motion.div>
        ) : null}
      </AnimatePresence>

      <SettingsSaveBar
        open={dirty && !!draft && !picking}
        saveLabel={copy.settings.saveProvider}
        onDiscard={() => {
          if (baseline) setDraft({ ...baseline, models: [...baseline.models] });
          else closeEditor();
        }}
        onSave={() => {
          if (!draft) return;
          if (!draft.endpoint.trim()) {
            toast.error(copy.settings.needEndpoint);
            return;
          }
          if (!draft.id && !draft.apiKey.trim()) {
            toast.error(copy.settings.needKey);
            return;
          }
          void save(draft);
        }}
      />

      <ConfirmDialog
        open={!!pending}
        title={copy.video.delete}
        body={copy.settings.deleteProvider}
        danger
        confirmLabel={copy.video.delete}
        onCancel={() => setPending(null)}
        onConfirm={async () => {
          if (!pending) return;
          try {
            await api.deleteChatProvider(pending.id);
            if (draft?.id === pending.id) closeEditor();
            setPending(null);
            await refresh();
          } catch (e) {
            toast.error(api.errMessage(e));
          }
        }}
      />
    </>
  );
}

function ProviderEditor({
  draft,
  setDraft,
  selected,
  busy,
  canDelete,
  onTest,
  onDelete,
  onClose,
}: {
  draft: Draft;
  setDraft: (d: Draft) => void;
  selected?: ChatProvider;
  busy: string;
  canDelete: boolean;
  onTest: () => void;
  onDelete: () => void;
  onClose: () => void;
}) {
  const copy = useCopy();
  const keyRef = useRef<HTMLInputElement>(null);
  const catalog = useModelCatalog((s) => s.catalog);
  const catalogLoading = useModelCatalog((s) => s.loading);
  const catalogError = useModelCatalog((s) => s.error);
  const refreshCatalog = useModelCatalog((s) => s.load);
  const patch = (p: Partial<Draft>) => setDraft({ ...draft, ...p });
  const hint =
    draft.vendor === "claude" ? copy.settings.presetHintClaude
    : draft.vendor === "azure" ? copy.settings.presetHintAzure
    : draft.vendor === "custom" ? copy.settings.presetHintCustom
    : undefined;

  useEffect(() => {
    if (!draft.id) keyRef.current?.focus();
  }, [draft.id, draft.vendor]);

  return (
    <SettingSection
      id="provider-edit"
      title={draft.id ? (draft.name || selected?.name || copy.settings.sections.providerAccount) : copy.settings.editorNew}
    >
      {hint ? <p className="-mt-1 text-[12px] leading-[1.5] text-muted">{hint}</p> : null}

      <Field label={copy.settings.providerName}>
        <Input className={CONTROL_LG} value={draft.name} onChange={(e) => patch({ name: e.target.value })} />
      </Field>
      <Field label={copy.settings.baseUrl}>
        <Input
          className={cn(CONTROL_LG, "font-mono")}
          value={draft.endpoint}
          onChange={(e) => patch({ endpoint: e.target.value })}
          onBlur={() => patch({ endpoint: draft.endpoint.trim().replace(/\/+$/, "") })}
        />
      </Field>
      <Field label={copy.settings.apiKey} hint={draft.hasKey && !draft.apiKey ? copy.settings.keepKey : copy.settings.apiKeyPh}>
        <Input
          ref={keyRef}
          className={CONTROL_LG}
          type="password"
          placeholder={draft.hasKey ? "••••••••" : copy.settings.apiKeyPh}
          value={draft.apiKey}
          onChange={(e) => patch({ apiKey: e.target.value })}
        />
      </Field>

      <SettingRow title={copy.settings.useForNewChats} list>
        <Switch checked={draft.makeDefault} onCheckedChange={(on) => patch({ makeDefault: on })} />
      </SettingRow>

      <div className="flex flex-col gap-[var(--space-item)]">
        <div>
          <div className="text-[13px] font-medium leading-[1.4] text-foreground">{copy.settings.modelsCsv}</div>
          <p className="mt-1 text-[12px] leading-[1.45] text-muted">{copy.settings.composerModelsHint}</p>
        </div>
        <ModelTokens
          models={draft.models}
          current={draft.model}
          onCurrent={(model) => patch({ model, models: mergeModelIds(model, draft.models) })}
          onRemove={(id) => {
            const models = draft.models.filter((m) => m !== id);
            const model = draft.model === id ? (models[0] || "") : draft.model;
            patch({ model, models });
          }}
        />
        <AddModelRow
          onAdd={(id) => patch({ model: draft.model || id, models: mergeModelIds(draft.models, id) })}
        />
        <CatalogBrowse
          vendor={draft.vendor}
          catalog={catalog}
          current={draft.model}
          selected={draft.models}
          loading={catalogLoading}
          error={catalogError}
          onPick={(id) => patch({ model: id, models: mergeModelIds(draft.models, id) })}
          onRefresh={refreshCatalog}
        />
      </div>

      <div className="flex flex-wrap items-center gap-2 pb-2">
        <Button size="sm" variant="ghost" onClick={onClose}>{copy.settings.backToList}</Button>
        <span className="flex-1" />
        {canDelete ? (
          <Button size="sm" variant="ghost" className="text-danger hover:text-danger" onClick={onDelete}>
            {copy.video.delete}
          </Button>
        ) : null}
        <Button size="sm" variant="lift" disabled={!!busy || !draft.endpoint.trim()} onClick={onTest}>
          {copy.settings.testConnection}
        </Button>
      </div>
    </SettingSection>
  );
}

function ModelTokens({
  models,
  current,
  onCurrent,
  onRemove,
}: {
  models: string[];
  current: string;
  onCurrent: (id: string) => void;
  onRemove: (id: string) => void;
}) {
  if (!models.length) return null;
  return (
    <div className="flex flex-wrap gap-1.5" role="list">
      {models.map((id) => {
        const on = id === current;
        return (
          <span
            key={id}
            role="listitem"
            className={cn(
              "inline-flex max-w-full items-center gap-0.5 rounded-full pl-2.5 pr-1 text-[12px] font-mono",
              on ? "bg-foreground text-background" : "bg-lift text-foreground",
            )}
          >
            <button
              type="button"
              className="max-w-[14rem] truncate py-1 text-left"
              aria-pressed={on}
              onClick={() => onCurrent(id)}
            >
              {id}
            </button>
            <button
              type="button"
              className={cn("grid size-6 place-items-center rounded-full", on ? "hover:bg-background/15" : "hover:bg-background/70")}
              aria-label={id}
              onClick={() => onRemove(id)}
            >
              <X className="size-3" aria-hidden />
            </button>
          </span>
        );
      })}
    </div>
  );
}

function AddModelRow({ onAdd }: { onAdd: (id: string) => void }) {
  const copy = useCopy();
  const [value, setValue] = useState("");
  function commit() {
    const id = value.trim();
    if (!id) return;
    onAdd(id);
    setValue("");
  }
  return (
    <div className="flex items-center gap-2">
      <Input
        className={cn(CONTROL_LG, "font-mono")}
        placeholder={copy.settings.addModelPh}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            commit();
          }
        }}
      />
      <Button size="sm" variant="lift" disabled={!value.trim()} onClick={commit}>
        {copy.settings.addModel}
      </Button>
    </div>
  );
}

function CatalogBrowse({
  vendor,
  catalog,
  current,
  selected,
  loading,
  error,
  onPick,
  onRefresh,
}: {
  vendor: string;
  catalog: ModelsDevCatalog;
  current: string;
  selected: string[];
  loading: boolean;
  error: string;
  onPick: (id: string) => void;
  onRefresh: (force?: boolean) => Promise<void>;
}) {
  const copy = useCopy();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const listed = useMemo(() => {
    const all = modelsForProvider(catalog, vendor);
    const q = query.trim().toLowerCase();
    if (!q) return all.slice(0, 24);
    return all.filter((m) => m.id.toLowerCase().includes(q) || (m.name || "").toLowerCase().includes(q)).slice(0, 24);
  }, [catalog, vendor, query]);
  if (vendor === "custom") return null;
  return (
    <div>
      <button
        type="button"
        className="flex w-full items-center justify-between rounded-lg py-1 text-left text-[12px] font-medium text-muted hover:text-foreground"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
      >
        <span>{copy.settings.browseCatalog}</span>
        <ChevronDown className={cn("size-3.5 transition-transform", open ? "rotate-0" : "-rotate-90")} aria-hidden />
      </button>
      {open ? (
        <div className="mt-2 flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <Input className={CONTROL_LG} placeholder={copy.settings.filterModels} value={query} onChange={(e) => setQuery(e.target.value)} />
            <Button
              size="icon"
              variant="ghost"
              disabled={loading}
              aria-label={copy.settings.catalogRefresh}
              onClick={async () => {
                await onRefresh(true);
                const st = useModelCatalog.getState();
                if (st.error || st.catalog.stale) toast.error(copy.settings.catalogOffline);
                else toast.success(copy.settings.catalogUpdated);
              }}
            >
              <RefreshCw className={cn("size-3.5", loading && "animate-spin")} aria-hidden />
            </Button>
          </div>
          {listed.length ? (
            <ul className="max-h-48 overflow-auto">
              {listed.map((m) => {
                const on = m.id === current;
                const inList = selected.includes(m.id);
                return (
                  <li key={m.id}>
                    <button
                      type="button"
                      className={cn(
                        "flex w-full items-center justify-between gap-3 rounded-md px-2 py-1.5 text-left hover:bg-lift/70",
                        on && "bg-lift",
                      )}
                      onClick={() => onPick(m.id)}
                    >
                      <span className="min-w-0">
                        <span className="block truncate text-[13px] text-foreground">{m.name || m.id}</span>
                        <span className="block truncate font-mono text-[11px] text-muted">{m.id}</span>
                      </span>
                      <span className="flex shrink-0 items-center gap-2 text-[11px] tabular-nums text-muted">
                        {formatTokens(m.contextWindow)}
                        {on || inList ? <Check className="size-3.5 text-accent" aria-hidden /> : null}
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
          ) : (
            <p className="px-1 text-[12px] text-muted">{error || "—"}</p>
          )}
        </div>
      ) : null}
    </div>
  );
}

function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
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
