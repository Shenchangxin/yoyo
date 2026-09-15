import { useEffect, useState, type ReactNode } from "react";
import { asArray, str } from "../../lib/normalize";
import type { AppConfig } from "../../lib/protocol";
import { useTheme, type ThemePref } from "../../lib/theme";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { cn } from "../../lib/utils";
import { LabFrame } from "./LabFrame";

type Section = "appearance" | "provider" | "workspace" | "policy" | "plugins";

const NAV: { id: Section; label: string; hint: string }[] = [
  { id: "appearance", label: "Appearance", hint: "Theme follows the window, not the agent." },
  { id: "provider", label: "Provider", hint: "Keys stay in the vault. The agent cannot read them." },
  { id: "workspace", label: "Workspace", hint: "Turns run against this directory." },
  { id: "policy", label: "Policy", hint: "Deny-first unless you opt in. Budget is a hard stop." },
  { id: "plugins", label: "Plugins", hint: "WASM stays HighRisk + ForceAsk. No WASI filesystem." },
];

export function ControlLab(props: {
  cfg: AppConfig;
  saved: AppConfig;
  plugins: any;
  onCfg: (c: AppConfig) => void;
  onSave: (apiKey: string) => void;
  onUpdate: () => void;
  onUnload: (name: string) => void;
}) {
  const [section, setSection] = useState<Section>(!props.saved.workspace ? "workspace" : "appearance");
  const [apiKey, setApiKey] = useState("");
  const { pref, setPref, resolved } = useTheme();
  const fibers = asArray(props.plugins?.fibers || props.plugins?.Fibers);
  const current = NAV.find((n) => n.id === section)!;
  const dirty = apiKey.trim().length > 0 || JSON.stringify(props.cfg) !== JSON.stringify(props.saved);

  useEffect(() => {
    if (!props.saved.workspace) setSection("workspace");
  }, [props.saved.workspace]);

  return (
    <LabFrame title="Control" hint="Vault, policy, and updater. The agent cannot change these surfaces.">
      <div className="flex min-h-[420px] gap-6">
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
              <h3 className="mb-3 text-sm font-medium">Color mode</h3>
              <div className="flex flex-wrap gap-2">
                {(["system", "dark", "light"] as ThemePref[]).map((p) => (
                  <Button key={p} variant={pref === p ? "default" : "lift"} size="sm" onClick={() => setPref(p)}>
                    {p[0].toUpperCase() + p.slice(1)}
                  </Button>
                ))}
              </div>
              <p className="mt-3 text-xs text-muted">Resolved {resolved}. Caption buttons and window background follow this.</p>
            </Card>
          ) : null}
          {section === "provider" ? (
            <Card>
              <Field label="Model">
                <Input autoComplete="off" value={props.cfg.model} onChange={(e) => props.onCfg({ ...props.cfg, model: e.target.value })} />
              </Field>
              <Field label="Base URL">
                <Input autoComplete="off" value={props.cfg.baseUrl} onChange={(e) => props.onCfg({ ...props.cfg, baseUrl: e.target.value })} />
              </Field>
              <Field label="API key">
                <Input
                  type="password"
                  autoComplete="off"
                  placeholder="paste then Save — stored in vault"
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                />
              </Field>
              <Field label="Extra models (BoN)">
                <Input value={props.cfg.models.join(",")} onChange={(e) => props.onCfg({ ...props.cfg, models: e.target.value.split(",").map((s) => s.trim()).filter(Boolean) })} />
              </Field>
            </Card>
          ) : null}
          {section === "workspace" ? (
            <Card>
              <Field label="Path">
                <Input autoComplete="off" value={props.cfg.workspace} onChange={(e) => props.onCfg({ ...props.cfg, workspace: e.target.value })} />
              </Field>
            </Card>
          ) : null}
          {section === "policy" ? (
            <Card>
              <label className="flex items-center gap-2 text-sm text-muted">
                <input type="checkbox" checked={props.cfg.autoAllow} onChange={(e) => props.onCfg({ ...props.cfg, autoAllow: e.target.checked })} />
                Auto-allow shell (deny-first stays off unless you opt in)
              </label>
              <Field label="Max budget USD">
                <Input type="number" value={props.cfg.maxBudgetUsd || ""} onChange={(e) => props.onCfg({ ...props.cfg, maxBudgetUsd: Number(e.target.value) })} />
              </Field>
              <Field label="USD per MTok">
                <Input type="number" value={props.cfg.usdPerMtok || ""} onChange={(e) => props.onCfg({ ...props.cfg, usdPerMtok: Number(e.target.value) })} />
              </Field>
            </Card>
          ) : null}
          {section === "plugins" ? (
            <Card>
              {fibers.length === 0 ? <p className="text-sm text-muted">No extra fibers loaded.</p> : fibers.map((name: any) => (
                <div className="mb-2 flex items-center justify-between rounded-xl border border-border bg-background px-3 py-2" key={str(name)}>
                  <span className="font-mono text-xs">{str(name)}</span>
                  <Button size="sm" variant="ghost" onClick={() => props.onUnload(str(name))}>Unload</Button>
                </div>
              ))}
            </Card>
          ) : null}
          <div className="mt-5 flex items-center gap-2">
            <Button
              disabled={!dirty}
              onClick={() => {
                const key = apiKey;
                setApiKey("");
                props.onSave(key);
              }}
            >
              Save
            </Button>
            <Button variant="lift" onClick={props.onUpdate}>Apply staged update</Button>
            {dirty ? <span className="text-xs text-muted">Unsaved changes</span> : null}
          </div>
        </div>
      </div>
    </LabFrame>
  );
}

function Card({ children }: { children: ReactNode }) {
  return <section className="space-y-3 rounded-2xl border border-border bg-panel p-5">{children}</section>;
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block text-xs text-muted">
      {label}
      <div className="mt-1">{children}</div>
    </label>
  );
}
