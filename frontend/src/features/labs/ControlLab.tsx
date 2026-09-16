import { useEffect, useState, type ReactNode } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { asArray, str } from "../../lib/normalize";
import { applyLocale, getLocale, useCopy, type Locale } from "../../lib/i18n";
import { DEFAULT_KEYMAP, mergeKeymap, type KeymapId } from "../../lib/keymap";
import { configToForm, controlSchema, formToConfig, type ControlValues } from "../../lib/control-schema";
import type { AppConfig } from "../../lib/protocol";
import { useTheme, type ThemePref } from "../../lib/theme";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import { cn } from "../../lib/utils";
import { LabFrame } from "./LabFrame";

type Section = "appearance" | "provider" | "workspace" | "policy" | "plugins" | "mcp" | "logs" | "shortcuts";

export function ControlLab(props: {
  saved: AppConfig;
  plugins: any;
  logs?: any;
  doctor?: any;
  vault?: any;
  onSave: (next: AppConfig, apiKey: string) => Promise<void> | void;
  onUpdate: () => void;
  onCheckUpdate?: () => void;
  onUnload: (name: string) => void;
  onBrowse?: () => Promise<string>;
  onStartMcp?: (name: string, command: string, args: string[]) => Promise<void> | void;
  onStopMcp?: (name: string) => Promise<void> | void;
}) {
  const copy = useCopy();
  const [section, setSection] = useState<Section>(!props.saved.workspace ? "workspace" : "appearance");
  const { pref, setPref, resolved } = useTheme();
  const fibers = asArray(props.plugins?.fibers || props.plugins?.Fibers);
  const mcp = asArray(props.plugins?.mcp || props.plugins?.servers);
  const [mcpName, setMcpName] = useState("");
  const [mcpCmd, setMcpCmd] = useState("");
  const [mcpArgs, setMcpArgs] = useState("");
  const [keymap, setKeymap] = useState(() => mergeKeymap(props.saved.keymap));
  const [locale, setLoc] = useState<Locale>(() => (props.saved.locale === "zh-CN" ? "zh-CN" : getLocale()));
  const nav: { id: Section; label: string; hint: string }[] = [
    { id: "appearance", label: copy.control.appearance, hint: copy.control.appearanceHint },
    { id: "provider", label: copy.control.provider, hint: copy.control.providerHint },
    { id: "workspace", label: copy.control.workspace, hint: copy.control.workspaceHint },
    { id: "policy", label: copy.control.policy, hint: copy.control.policyHint },
    { id: "mcp", label: copy.control.mcp, hint: copy.control.mcpHint },
    { id: "plugins", label: copy.control.plugins, hint: copy.control.pluginsHint },
    { id: "shortcuts", label: copy.control.shortcuts, hint: copy.control.shortcutsHint },
    { id: "logs", label: copy.control.logs, hint: copy.control.logsHint },
  ];
  const current = nav.find((n) => n.id === section)!;
  const form = useForm<ControlValues>({
    resolver: zodResolver(controlSchema),
    defaultValues: configToForm(props.saved),
    mode: "onChange",
  });

  useEffect(() => {
    form.reset(configToForm(props.saved));
    setKeymap(mergeKeymap(props.saved.keymap));
    if (props.saved.locale === "zh-CN" || props.saved.locale === "en") setLoc(props.saved.locale);
  }, [props.saved, form]);

  useEffect(() => {
    if (!props.saved.workspace) setSection("workspace");
  }, [props.saved.workspace]);

  const dirty = form.formState.isDirty;
  const THEME: { id: ThemePref; label: string }[] = [
    { id: "system", label: copy.control.system },
    { id: "dark", label: copy.control.dark },
    { id: "light", label: copy.control.light },
  ];
  const vaultSource = str(props.vault?.source || props.doctor?.vault?.source);
  const vaultSaved = !!props.vault?.saved || !!props.doctor?.vault?.saved;

  return (
    <LabFrame title={copy.control.title} hint={copy.control.hint}>
      <form
        className="flex min-h-[420px] gap-6"
        onSubmit={form.handleSubmit(async (values) => {
          const next = { ...formToConfig(props.saved, values), keymap, locale };
          await props.onSave(next, values.apiKey);
          form.reset({ ...values, apiKey: "" });
        })}
      >
        <nav className="w-40 shrink-0 space-y-1" aria-label="Control sections">
          {nav.map((n) => (
            <button
              type="button"
              key={n.id}
              aria-current={section === n.id ? "page" : undefined}
              className={cn(
                "w-full rounded-xl px-3 py-2 text-left text-[13px]",
                section === n.id ? "bg-lift text-foreground" : "text-muted hover:bg-lift/60 hover:text-foreground",
              )}
              onClick={() => setSection(n.id)}
            >
              {n.label}
            </button>
          ))}
        </nav>
        <div className="min-w-0 flex-1">
          <p className="mb-4 text-sm text-muted">{current.hint}</p>
          {section === "appearance" ? (
            <Card>
              <h3 className="mb-3 text-sm font-medium">{copy.control.colorMode}</h3>
              <div className="flex flex-wrap gap-2">
                {THEME.map((p) => (
                  <Button key={p.id} type="button" variant={pref === p.id ? "default" : "lift"} size="sm" onClick={() => setPref(p.id)}>
                    {p.label}
                  </Button>
                ))}
              </div>
              <p className="mt-3 text-xs text-muted">
                {copy.control.resolved} {resolved}. {copy.control.captionFollow}
              </p>
              <h3 className="mb-3 mt-5 text-sm font-medium">{copy.control.language}</h3>
              <div className="flex flex-wrap gap-2">
                {(["en", "zh-CN"] as Locale[]).map((id) => (
                  <Button
                    key={id}
                    type="button"
                    variant={locale === id ? "default" : "lift"}
                    size="sm"
                    onClick={() => {
                      setLoc(id);
                      applyLocale(id);
                    }}
                  >
                    {id === "zh-CN" ? "中文" : "English"}
                  </Button>
                ))}
              </div>
            </Card>
          ) : null}
          {section === "provider" ? (
            <Card>
              <Field label={copy.control.model} error={form.formState.errors.model?.message}>
                <Input autoComplete="off" {...form.register("model")} />
              </Field>
              <Field label={copy.control.baseUrl} error={form.formState.errors.baseUrl?.message}>
                <Input autoComplete="off" {...form.register("baseUrl")} />
              </Field>
              <Field label={copy.control.apiKey}>
                <Input type="password" autoComplete="off" placeholder={copy.control.apiKeyPh} {...form.register("apiKey")} />
              </Field>
              <p className="text-xs text-muted">
                {copy.control.vaultSource}: {vaultSource || (vaultSaved ? "file" : "empty")}
              </p>
              <Field label={copy.control.extraModels}>
                <Input {...form.register("modelsCsv")} />
              </Field>
            </Card>
          ) : null}
          {section === "workspace" ? (
            <Card>
              <Field label={copy.control.path} error={form.formState.errors.workspace?.message}>
                <div className="flex gap-2">
                  <Input autoComplete="off" {...form.register("workspace")} />
                  {props.onBrowse ? (
                    <Button
                      type="button"
                      variant="lift"
                      onClick={async () => {
                        const p = await props.onBrowse?.();
                        if (p) form.setValue("workspace", p, { shouldDirty: true, shouldValidate: true });
                      }}
                    >
                      {copy.control.browse}
                    </Button>
                  ) : null}
                </div>
              </Field>
              <label className="flex items-center gap-2 text-sm text-muted">
                <input type="checkbox" {...form.register("closeToTray")} />
                {copy.control.closeToTray}
              </label>
            </Card>
          ) : null}
          {section === "policy" ? (
            <Card>
              <label className="flex items-center gap-2 text-sm text-muted">
                <input type="checkbox" {...form.register("autoAllow")} />
                {copy.control.autoAllow}
              </label>
              <Field label={copy.control.maxBudget} error={form.formState.errors.maxBudgetUsd?.message}>
                <Input type="number" {...form.register("maxBudgetUsd", { valueAsNumber: true })} />
              </Field>
              <Field label={copy.control.usdPerMtok} error={form.formState.errors.usdPerMtok?.message}>
                <Input type="number" {...form.register("usdPerMtok", { valueAsNumber: true })} />
              </Field>
            </Card>
          ) : null}
          {section === "plugins" ? (
            <Card>
              {fibers.length === 0 ? (
                <p className="text-sm text-muted">{copy.control.noFibers}</p>
              ) : (
                fibers.map((name: any) => (
                  <div className="mb-2 flex items-center justify-between rounded-xl border border-border bg-background px-3 py-2" key={str(name)}>
                    <span className="font-mono text-xs">{str(name)}</span>
                    <Button type="button" size="sm" variant="ghost" onClick={() => props.onUnload(str(name))}>
                      {copy.control.unload}
                    </Button>
                  </div>
                ))
              )}
            </Card>
          ) : null}
          {section === "mcp" ? (
            <Card>
              {mcp.length === 0 ? <p className="text-sm text-muted">{copy.control.noMcp}</p> : null}
              {mcp.map((row: any) => {
                const name = str(pickName(row));
                return (
                  <div className="mb-2 flex items-center justify-between rounded-xl border border-border bg-background px-3 py-2" key={name}>
                    <span className="font-mono text-xs">{name}</span>
                    <Button type="button" size="sm" variant="ghost" onClick={() => props.onStopMcp?.(name)}>
                      {copy.control.mcpStop}
                    </Button>
                  </div>
                );
              })}
              <Field label={copy.control.mcpName}>
                <Input value={mcpName} onChange={(e) => setMcpName(e.target.value)} />
              </Field>
              <Field label={copy.control.mcpCommand}>
                <Input value={mcpCmd} onChange={(e) => setMcpCmd(e.target.value)} />
              </Field>
              <Field label={copy.control.mcpArgs}>
                <Input value={mcpArgs} onChange={(e) => setMcpArgs(e.target.value)} />
              </Field>
              <Button
                type="button"
                size="sm"
                disabled={!mcpName.trim() || !mcpCmd.trim()}
                onClick={() => props.onStartMcp?.(mcpName.trim(), mcpCmd.trim(), mcpArgs.split(/\s+/).filter(Boolean))}
              >
                {copy.control.mcpStart}
              </Button>
            </Card>
          ) : null}
          {section === "shortcuts" ? (
            <Card>
              {(Object.keys(DEFAULT_KEYMAP) as KeymapId[]).map((id) => (
                <Field key={id} label={id}>
                  <Input
                    value={keymap[id]}
                    aria-label={id}
                    onChange={(e) => setKeymap((prev) => ({ ...prev, [id]: e.target.value }))}
                    spellCheck={false}
                  />
                </Field>
              ))}
            </Card>
          ) : null}
          {section === "logs" ? (
            <Card>
              <pre className="max-h-72 overflow-auto whitespace-pre-wrap font-mono text-[11px] text-muted">
                {JSON.stringify(props.doctor || props.logs || {}, null, 2).slice(0, 8000)}
              </pre>
            </Card>
          ) : null}
          <div className="mt-5 flex items-center gap-2">
            <Button type="submit">
              {copy.control.save}
            </Button>
            <Button type="button" variant="lift" onClick={props.onUpdate}>
              {copy.control.applyUpdate}
            </Button>
            {props.onCheckUpdate ? (
              <Button type="button" variant="lift" onClick={props.onCheckUpdate}>
                {copy.control.checkUpdate}
              </Button>
            ) : null}
            {dirty ? <span className="text-xs text-muted">{copy.control.unsaved}</span> : null}
          </div>
        </div>
      </form>
    </LabFrame>
  );
}

function Card({ children }: { children: ReactNode }) {
  return <section className="space-y-3 rounded-2xl border border-border bg-panel p-5">{children}</section>;
}

function Field({ label, error, children }: { label: string; error?: string; children: ReactNode }) {
  return (
    <div>
      <Label>{label}</Label>
      <div className="mt-1">{children}</div>
      {error ? <p className="mt-1 text-xs text-danger">{error}</p> : null}
    </div>
  );
}

function pickName(row: any): string {
  if (typeof row === "string") return row;
  return row?.name || row?.Name || "";
}
