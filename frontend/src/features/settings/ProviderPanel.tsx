import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { useCopy } from "../../lib/i18n";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { cn } from "../../lib/utils";
import { applyProviderPreset, PROVIDER_PRESETS } from "../../lib/providers";
import { formatTokens, formatUsd, lookupCatalogModel, mergeModelIds, modelsForProvider } from "../../lib/models-dev";
import { MODELS_DEV_SNAPSHOT } from "../../lib/models-dev.snapshot";
import { SettingRow, SettingSection } from "./SettingChrome";
import { ProviderMark } from "./ProviderMark";
import type { SettingsHost } from "./host";

export function ProviderSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const [provider, setProvider] = useState(host.cfg.provider || "openai");
  const [model, setModel] = useState(host.cfg.model);
  const [baseUrl, setBaseUrl] = useState(host.cfg.baseUrl);
  const [apiKey, setApiKey] = useState("");
  const [modelsCsv, setModelsCsv] = useState(host.cfg.models.join(","));
  const vaultSource = String(host.vault?.source || host.doctor?.vault?.source || "");
  const catalog = useMemo(() => modelsForProvider(MODELS_DEV_SNAPSHOT, provider), [provider]);
  const selected = lookupCatalogModel(MODELS_DEV_SNAPSHOT, provider, model)?.model;

  useEffect(() => {
    const id = host.cfg.provider === "anthropic" ? "claude" : (host.cfg.provider || "openai");
    setProvider(PROVIDER_PRESETS.some((p) => p.id === id) ? id : "custom");
    setModel(host.cfg.model);
    setBaseUrl(host.cfg.baseUrl);
    setModelsCsv(host.cfg.models.join(","));
  }, [host.cfg]);

  function pickProvider(id: string) {
    setProvider(id);
    const next = applyProviderPreset(id);
    if (!next) {
      setModelsCsv(mergeModelIds(model).join(", "));
      return;
    }
    setBaseUrl(next.baseUrl);
    const cats = modelsForProvider(MODELS_DEV_SNAPSHOT, id);
    const pick = cats.find((m) => m.id === next.model) || cats.find((m) => next.models.includes(m.id)) || cats[0];
    setModel(pick?.id || next.model);
    setModelsCsv(mergeModelIds(pick?.id, next.models).join(", "));
  }

  function pickModel(id: string) {
    setModel(id);
    setModelsCsv(mergeModelIds(id, modelsCsv.split(",")).join(", "));
  }

  const hint =
    provider === "claude" ? copy.settings.presetHintClaude
    : provider === "azure" ? copy.settings.presetHintAzure
    : provider === "custom" ? copy.settings.presetHintCustom
    : undefined;

  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.provider}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.control.providerHint}</p>
      <div role="radiogroup" aria-label={copy.settings.provider} className="mb-6 grid grid-cols-2 gap-2 sm:grid-cols-3">
        {PROVIDER_PRESETS.map((p) => {
          const count = modelsForProvider(MODELS_DEV_SNAPSHOT, p.id).length;
          const active = provider === p.id;
          return (
            <button
              type="button"
              role="radio"
              aria-checked={active}
              aria-label={p.label}
              key={p.id}
              onClick={() => pickProvider(p.id)}
              className={cn(
                "flex items-center gap-3 rounded-xl border px-3 py-2.5 text-left transition-colors",
                active ? "border-foreground/30 bg-lift" : "border-border/80 bg-card hover:bg-lift/60",
              )}
            >
              <ProviderMark id={p.id} />
              <span className="min-w-0">
                <span className="block truncate text-[13px] font-medium text-foreground">{p.label}</span>
                <span className="block truncate text-[11px] text-muted">
                  {count ? `${count} ${copy.settings.modelsCount}` : p.model || "—"}
                </span>
              </span>
            </button>
          );
        })}
      </div>
      <SettingSection id="provider-account" title={copy.settings.sections.providerAccount} description={copy.settings.providerDesc}>
        <SettingRow title={copy.settings.baseUrl} description={hint}>
          <Input aria-label={copy.settings.baseUrl} className="h-8 w-64 text-[12px]" value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} />
        </SettingRow>
        <SettingRow title={copy.settings.apiKey} description={`${copy.settings.vaultSource}: ${vaultSource || "empty"}`} border={false}>
          <Input className="h-8 w-64 text-[12px]" type="password" autoComplete="off" placeholder={copy.settings.apiKeyPh} value={apiKey} onChange={(e) => setApiKey(e.target.value)} />
        </SettingRow>
      </SettingSection>
      <SettingSection id="provider-models" title={copy.settings.sections.providerModels} description={copy.settings.catalogHint}>
        <SettingRow title={copy.settings.selectedModel}>
          <Input aria-label={copy.settings.model} className="h-8 w-56 text-[12px]" value={model} onChange={(e) => setModel(e.target.value)} />
        </SettingRow>
        {selected ? (
          <div className="flex flex-wrap gap-1.5 border-b border-border/80 px-5 py-3">
            <MetaChip label={copy.settings.ctxWindow} value={formatTokens(selected.contextWindow)} />
            <MetaChip label={copy.settings.outputLimit} value={formatTokens(selected.maxTokens)} />
            {selected.input?.includes("image") ? <MetaChip label={copy.settings.vision} value={copy.settings.yes} /> : null}
            {selected.reasoning ? <MetaChip label={copy.settings.reasoning} value={copy.settings.yes} /> : null}
            {selected.cost ? (
              <MetaChip
                label={copy.settings.perMillion}
                value={`${formatUsd(selected.cost.input)} → ${formatUsd(selected.cost.output)}`}
              />
            ) : null}
          </div>
        ) : null}
        {catalog.length ? (
          <>
            <ul className="max-h-80 overflow-auto border-b border-border/80">
              {catalog.map((m, i) => {
                const active = m.id === model;
                return (
                  <li key={m.id} className={cn(i < catalog.length - 1 && "border-b border-border/70")}>
                    <button
                      type="button"
                      className={cn("flex w-full items-start justify-between gap-3 px-5 py-3 text-left hover:bg-lift/70", active && "bg-lift")}
                      onClick={() => pickModel(m.id)}
                    >
                      <span className="min-w-0">
                        <span className="block truncate text-[13px] font-medium text-foreground">{m.name || m.id}</span>
                        <span className="mt-0.5 block truncate font-mono text-[11px] text-muted">{m.id}</span>
                      </span>
                      <span className="shrink-0 text-right text-[11px] tabular-nums text-muted">
                        <span className="block">{formatTokens(m.contextWindow)}</span>
                        {m.cost ? <span className="block">{formatUsd(m.cost.input)}{copy.settings.perMillion}</span> : null}
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
            <SettingRow title={copy.settings.extraModels} border={false}>
              <Input className="h-8 w-64 text-[12px]" value={modelsCsv} onChange={(e) => setModelsCsv(e.target.value)} />
            </SettingRow>
          </>
        ) : (
          <SettingRow title={copy.settings.extraModels} border={false}>
            <Input className="h-8 w-64 text-[12px]" value={modelsCsv} onChange={(e) => setModelsCsv(e.target.value)} />
          </SettingRow>
        )}
      </SettingSection>
      <div className="mt-2 flex gap-2 px-1.5">
        <Button
          onClick={async () => {
            await host.saveProvider(
              {
                provider,
                model,
                baseUrl,
                models: modelsCsv.split(",").map((s) => s.trim()).filter(Boolean),
              },
              apiKey,
            );
            setApiKey("");
            toast.success(copy.app.controlSaved);
          }}
        >
          {copy.settings.saveProvider}
        </Button>
        <Button
          variant="lift"
          onClick={async () => {
            try {
              const r = await host.onTestProvider();
              if (r?.ok === false || r?.error) toast.error(String(r.error || copy.settings.testFail));
              else toast.success(copy.settings.testOk);
            } catch (e: any) {
              toast.error(e?.message || copy.settings.testFail);
            }
          }}
        >
          {copy.settings.testConnection}
        </Button>
      </div>
    </>
  );
}

function MetaChip({ label, value }: { label: string; value: string }) {
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-lift px-2 py-0.5 text-[11px] text-muted">
      <span>{label}</span>
      <span className="font-medium text-foreground">{value}</span>
    </span>
  );
}
