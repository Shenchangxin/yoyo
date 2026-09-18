import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { Check } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { cn } from "../../lib/utils";
import { applyProviderPreset, PROVIDER_PRESETS } from "../../lib/providers";
import { formatTokens, formatUsd, lookupCatalogModel, mergeModelIds, modelsForProvider } from "../../lib/models-dev";
import { MODELS_DEV_SNAPSHOT } from "../../lib/models-dev.snapshot";
import {
  CONTROL_LG,
  CONTROL_MD,
  SettingRow,
  SettingSection,
  SettingsPageHeader,
  SettingsSaveBar,
} from "./SettingChrome";
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

  function hydrate() {
    const id = host.cfg.provider === "anthropic" ? "claude" : host.cfg.provider || "openai";
    setProvider(PROVIDER_PRESETS.some((p) => p.id === id) ? id : "custom");
    setModel(host.cfg.model);
    setBaseUrl(host.cfg.baseUrl);
    setModelsCsv(host.cfg.models.join(","));
    setApiKey("");
  }

  useEffect(hydrate, [host.cfg]);

  const savedProvider = host.cfg.provider === "anthropic" ? "claude" : host.cfg.provider || "openai";
  const dirty =
    provider !== savedProvider ||
    model !== host.cfg.model ||
    baseUrl !== host.cfg.baseUrl ||
    modelsCsv !== host.cfg.models.join(",") ||
    apiKey.length > 0;

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
      <SettingsPageHeader
        title={copy.settings.tabs.provider}
        description={copy.settings.tabHints.provider}
        actions={
          <Button
            size="sm"
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
        }
      />

      <section className="mb-7">
        <h2 className="mb-2 px-1 text-[12.5px] font-semibold text-foreground/75">{copy.settings.provider}</h2>
        <div role="radiogroup" aria-label={copy.settings.provider} className="grid grid-cols-2 gap-2 sm:grid-cols-3">
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
                  "surface-inset relative flex items-center gap-2.5 rounded-[11px] border px-3 py-2.5 text-left transition-colors",
                  active ? "border-accent/70 bg-lift ring-1 ring-accent/25" : "border-border bg-card hover:bg-lift/55",
                )}
              >
                <ProviderMark id={p.id} />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-[12.5px] font-medium text-foreground">{p.label}</span>
                  <span className="block truncate text-[11px] text-muted">
                    {count ? `${count} ${copy.settings.modelsCount}` : p.model || "—"}
                  </span>
                </span>
                {active ? <Check className="size-3.5 shrink-0 text-accent" aria-hidden /> : null}
              </button>
            );
          })}
        </div>
      </section>

      <SettingSection
        id="provider-account"
        title={copy.settings.sections.providerAccount}
        footnote={hint || copy.settings.providerDesc}
      >
        <SettingRow title={copy.settings.baseUrl}>
          <Input
            aria-label={copy.settings.baseUrl}
            className={cn(CONTROL_LG, "font-mono")}
            value={baseUrl}
            onChange={(e) => setBaseUrl(e.target.value)}
          />
        </SettingRow>
        <SettingRow
          title={copy.settings.apiKey}
          description={`${copy.settings.vaultSource}: ${vaultSource || "empty"}`}
          border={false}
        >
          <Input
            aria-label={copy.settings.apiKey}
            className={CONTROL_LG}
            type="password"
            autoComplete="off"
            placeholder={copy.settings.apiKeyPh}
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
          />
        </SettingRow>
      </SettingSection>

      <SettingSection
        id="provider-models"
        title={copy.settings.sections.providerModels}
        footnote={copy.settings.catalogHint}
      >
        <SettingRow title={copy.settings.selectedModel} border={!!selected || catalog.length > 0}>
          <Input
            aria-label={copy.settings.model}
            className={cn(CONTROL_MD, "font-mono")}
            value={model}
            onChange={(e) => setModel(e.target.value)}
          />
        </SettingRow>
        {selected ? (
          <div className="relative flex flex-wrap gap-1.5 px-4 py-3">
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
            <span aria-hidden className="pointer-events-none absolute bottom-0 left-4 right-0 h-px bg-border" />
          </div>
        ) : null}
        {catalog.length ? (
          <ul className="max-h-80 overflow-auto">
            {catalog.map((m) => {
              const active = m.id === model;
              return (
                <li key={m.id}>
                  <button
                    type="button"
                    className={cn(
                      "relative flex w-full items-center justify-between gap-3 px-4 py-2.5 text-left transition-colors hover:bg-lift/55",
                      active && "bg-lift",
                    )}
                    onClick={() => pickModel(m.id)}
                  >
                    <span className="min-w-0">
                      <span className="block truncate text-[12.5px] font-medium text-foreground">{m.name || m.id}</span>
                      <span className="mt-0.5 block truncate font-mono text-[11px] text-muted">{m.id}</span>
                    </span>
                    <span className="flex shrink-0 items-center gap-2 text-right text-[11px] tabular-nums text-muted">
                      <span>
                        <span className="block">{formatTokens(m.contextWindow)}</span>
                        {m.cost ? (
                          <span className="block">
                            {formatUsd(m.cost.input)}
                            {copy.settings.perMillion}
                          </span>
                        ) : null}
                      </span>
                      {active ? <Check className="size-3.5 text-accent" aria-hidden /> : null}
                    </span>
                    <span aria-hidden className="pointer-events-none absolute bottom-0 left-4 right-0 h-px bg-border" />
                  </button>
                </li>
              );
            })}
          </ul>
        ) : null}
        <SettingRow title={copy.settings.extraModels} border={false}>
          <Input
            aria-label={copy.settings.extraModels}
            className={cn(CONTROL_LG, "font-mono")}
            value={modelsCsv}
            onChange={(e) => setModelsCsv(e.target.value)}
          />
        </SettingRow>
      </SettingSection>

      <SettingsSaveBar
        open={dirty}
        saveLabel={copy.settings.saveProvider}
        onDiscard={hydrate}
        onSave={async () => {
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
      />
    </>
  );
}

function MetaChip({ label, value }: { label: string; value: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-lift px-2 py-0.5 text-[11px] text-muted">
      <span>{label}</span>
      <span className="font-medium tabular-nums text-foreground">{value}</span>
    </span>
  );
}
