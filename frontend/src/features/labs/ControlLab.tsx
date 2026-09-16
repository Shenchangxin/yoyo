import { useEffect, useState, type ReactNode } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { asArray, str } from "../../lib/normalize";
import { copy } from "../../lib/copy";
import { configToForm, controlSchema, formToConfig, type ControlValues } from "../../lib/control-schema";
import type { AppConfig } from "../../lib/protocol";
import { useTheme, type ThemePref } from "../../lib/theme";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import { cn } from "../../lib/utils";
import { LabFrame } from "./LabFrame";

type Section = "appearance" | "provider" | "workspace" | "policy" | "plugins";

const NAV: { id: Section; label: string; hint: string }[] = [
  { id: "appearance", label: copy.control.appearance, hint: copy.control.appearanceHint },
  { id: "provider", label: copy.control.provider, hint: copy.control.providerHint },
  { id: "workspace", label: copy.control.workspace, hint: copy.control.workspaceHint },
  { id: "policy", label: copy.control.policy, hint: copy.control.policyHint },
  { id: "plugins", label: copy.control.plugins, hint: copy.control.pluginsHint },
];

export function ControlLab(props: {
  saved: AppConfig;
  plugins: any;
  onSave: (next: AppConfig, apiKey: string) => Promise<void> | void;
  onUpdate: () => void;
  onUnload: (name: string) => void;
}) {
  const [section, setSection] = useState<Section>(!props.saved.workspace ? "workspace" : "appearance");
  const { pref, setPref, resolved } = useTheme();
  const fibers = asArray(props.plugins?.fibers || props.plugins?.Fibers);
  const current = NAV.find((n) => n.id === section)!;
  const form = useForm<ControlValues>({
    resolver: zodResolver(controlSchema),
    defaultValues: configToForm(props.saved),
    mode: "onChange",
  });

  useEffect(() => {
    form.reset(configToForm(props.saved));
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

  return (
    <LabFrame title={copy.control.title} hint={copy.control.hint}>
      <form
        className="flex min-h-[420px] gap-6"
        onSubmit={form.handleSubmit(async (values) => {
          await props.onSave(formToConfig(props.saved, values), values.apiKey);
          form.reset({ ...values, apiKey: "" });
        })}
      >
        <nav className="w-40 shrink-0 space-y-1" aria-label="Control sections">
          {NAV.map((n) => (
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
              <Field label={copy.control.extraModels}>
                <Input {...form.register("modelsCsv")} />
              </Field>
            </Card>
          ) : null}
          {section === "workspace" ? (
            <Card>
              <Field label={copy.control.path} error={form.formState.errors.workspace?.message}>
                <Input autoComplete="off" {...form.register("workspace")} />
              </Field>
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
          <div className="mt-5 flex items-center gap-2">
            <Button type="submit" disabled={!dirty}>
              {copy.control.save}
            </Button>
            <Button type="button" variant="lift" onClick={props.onUpdate}>
              {copy.control.applyUpdate}
            </Button>
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
