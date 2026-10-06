import { useEffect, useMemo, useState } from "react";
import { RefreshCw, Search } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { EmptyState } from "../../components/ui/empty-state";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import type { PackStatus, SkillInfo } from "../../lib/protocol";
import type { Copy } from "../../lib/copy";
import { YoyoMark } from "../shell/YoyoMark";
import { PresenceModuleLoading } from "../presence";
import { useUI, type SkillsTab } from "../../lib/store";

type Pane = SkillsTab;

function SkillsLoading() {
  useEffect(() => {
    useUI.getState().setModuleLoading(true);
    return () => useUI.getState().setModuleLoading(false);
  }, []);
  return (
    <PresenceModuleLoading />
  );
}

export function SkillsWorkspace(props: {
  installed: SkillInfo[];
  loaded: string[];
  workspace?: string;
  onRefreshInstalled: () => void;
}) {
  const copy = useCopy();
  const skillsTab = useUI((s) => s.skillsTab);
  const setSkillsTab = useUI((s) => s.setSkillsTab);
  const [pane, setPane] = useState<Pane>(skillsTab);
  const [q, setQ] = useState("");
  const [cat, setCat] = useState("");
  const [open, setOpen] = useState("");
  const [busy, setBusy] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(true);
  const [catalog, setCatalog] = useState<api.SkillMarketCatalog>({ source: "", fetchedAt: "", items: [] });
  const [packs, setPacks] = useState<PackStatus[]>([]);
  const [packsLoading, setPacksLoading] = useState(true);

  useEffect(() => { setPane(skillsTab); }, [skillsTab]);

  const loadMarket = (refresh = false) => {
    setErr("");
    setLoading(true);
    void api.skillMarket(refresh)
      .then(setCatalog)
      .catch((e) => setErr(api.errMessage(e)))
      .finally(() => setLoading(false));
  };

  const loadPacks = () => {
    setPacksLoading(true);
    void api.listPacks(props.workspace)
      .then(setPacks)
      .catch((e) => setErr(api.errMessage(e)))
      .finally(() => setPacksLoading(false));
  };

  useEffect(() => { loadMarket(false); }, []);
  useEffect(() => { loadPacks(); }, [props.workspace]);

  const installedNames = useMemo(() => {
    const set = new Set<string>();
    for (const s of props.installed) {
      set.add(s.name);
      if (s.slug) set.add(s.slug);
      if (s.displayName) set.add(s.displayName);
    }
    return set;
  }, [props.installed]);
  const incomplete = useMemo(() => {
    const set = new Set<string>();
    for (const s of props.installed) {
      if (!s.incomplete) continue;
      set.add(s.name);
      if (s.slug) set.add(s.slug);
    }
    return set;
  }, [props.installed]);
  const loaded = new Set(props.loaded);
  const categories = useMemo(() => {
    const set = new Set<string>();
    for (const it of catalog.items) if (it.category) set.add(it.category);
    return [...set];
  }, [catalog.items]);

  const filtered = useMemo(() => {
    const needle = q.trim().toLowerCase();
    return catalog.items.filter((it) => {
      if (cat && it.category !== cat) return false;
      if (!needle) return true;
      return (it.slug + it.name + (it.displayName || "") + it.purpose + it.category).toLowerCase().includes(needle);
    });
  }, [catalog.items, q, cat]);

  const featured = filtered.filter((it) => it.featured);
  const grouped = useMemo(() => {
    const map = new Map<string, api.SkillMarketItem[]>();
    for (const it of filtered) {
      const key = it.category || copy.skills.market;
      const list = map.get(key) || [];
      list.push(it);
      map.set(key, list);
    }
    return [...map.entries()];
  }, [filtered, copy.skills.market]);

  async function install(slug: string, repairing = false) {
    setBusy(slug);
    setErr("");
    try {
      const out = await api.installMarketSkill(slug);
      props.onRefreshInstalled();
      if (out.warning) toast.message(copy.skills.installPartial);
      else if (repairing) toast.success(copy.skills.repaired);
    } catch (e) {
      setErr(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  async function uninstall(slug: string) {
    setBusy(slug);
    setErr("");
    try {
      await api.uninstallMarketSkill(slug);
      props.onRefreshInstalled();
    } catch (e) {
      setErr(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  function selectPane(id: Pane) {
    setPane(id);
    setSkillsTab(id);
  }

  async function installKnownPack(id: string, path = "") {
    setBusy(id);
    setErr("");
    try {
      await api.installPack(id, path);
      loadPacks();
      props.onRefreshInstalled();
      toast.success(copy.skills.installedBadge);
    } catch (e) {
      setErr(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  async function uninstallKnownPack(id: string) {
    setBusy(id);
    setErr("");
    try {
      await api.uninstallPack(id);
      loadPacks();
      props.onRefreshInstalled();
    } catch (e) {
      setErr(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  async function setPackEnabled(id: string, scope: "workspace" | "global" | "inherit", enabled: boolean) {
    setBusy(id + ":" + scope);
    setErr("");
    try {
      await api.enablePack(id, scope, enabled);
      loadPacks();
      props.onRefreshInstalled();
    } catch (e) {
      setErr(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  async function refreshPack(id: string) {
    setBusy(id + ":update");
    setErr("");
    try {
      await api.updatePack(id);
      loadPacks();
      props.onRefreshInstalled();
    } catch (e) {
      setErr(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="skills-workspace">
      <div className="min-h-0 flex-1 overflow-auto">
        <div className="mx-auto w-full max-w-5xl px-6 py-6">
          <p className="mb-4 max-w-[46ch] text-[13px] leading-[1.55] text-pretty text-muted">{copy.skills.hint}</p>
          <div className="mb-5 flex h-10 items-stretch gap-3">
            <div className="process-tabs flex h-10 items-stretch gap-0.5" role="tablist" aria-label={copy.skills.title}>
              {(["installed", "market", "packs"] as Pane[]).map((id) => (
                <button
                  type="button"
                  key={id}
                  role="tab"
                  aria-selected={pane === id}
                  className={cn(
                    "relative flex h-full cursor-pointer items-center px-2.5 text-[13px] font-medium transition-colors",
                    pane === id ? "text-foreground" : "text-muted hover:text-foreground",
                  )}
                  onClick={() => selectPane(id)}
                >
                  {id === "installed" ? copy.skills.installed : id === "market" ? copy.skills.market : copy.skills.packs}
                  <span
                    className={cn(
                      "absolute inset-x-2 -bottom-px h-[2px] rounded-full bg-accent transition-opacity duration-200 ease-[var(--ease-out)]",
                      pane === id ? "opacity-100" : "opacity-0",
                    )}
                    aria-hidden
                  />
                </button>
              ))}
            </div>
            {pane === "market" ? (
              <div className="flex min-w-0 flex-1 items-center justify-end gap-2">
                <div className="relative min-w-[10rem] max-w-[18rem] flex-1">
                  <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted" aria-hidden />
                  <Input
                    className="h-8 rounded-lg border-transparent bg-lift/50 pl-8 text-[13px]"
                    placeholder={copy.skills.search}
                    value={q}
                    onChange={(e) => setQ(e.target.value)}
                    aria-label={copy.skills.search}
                    autoComplete="off"
                    name="skill-catalog-search"
                    spellCheck={false}
                  />
                </div>
                <Button size="sm" variant="outline" onClick={() => loadMarket(true)}>
                  <RefreshCw className="size-3.5" aria-hidden />
                  {copy.skills.refresh}
                </Button>
              </div>
            ) : null}
          </div>
          {err ? <p className="mb-4 text-[13px] text-danger" role="alert">{err}</p> : null}
          {pane === "installed" ? (
            <InstalledList
              skills={props.installed}
              loaded={loaded}
              open={open}
              onOpen={setOpen}
              busy={busy}
              onUninstall={uninstall}
              onRepair={(slug) => install(slug, true)}
            />
          ) : pane === "packs" ? (
            <PacksList
              packs={packs}
              loading={packsLoading}
              busy={busy}
              onInstall={installKnownPack}
              onUninstall={uninstallKnownPack}
              onEnable={setPackEnabled}
              onUpdate={refreshPack}
            />
          ) : (
            <MarketList
              featured={featured}
              grouped={grouped}
              categories={categories}
              cat={cat}
              onCat={setCat}
              installed={installedNames}
              incomplete={incomplete}
              busy={busy}
              onInstall={install}
              query={q}
              loading={loading}
            />
          )}
        </div>
      </div>
    </div>
  );
}

function InstalledList(props: {
  skills: SkillInfo[];
  loaded: Set<string>;
  open: string;
  onOpen: (name: string) => void;
  busy: string;
  onUninstall: (slug: string) => void;
  onRepair: (slug: string) => void;
}) {
  const copy = useCopy();
  if (!props.skills.length) {
    return <EmptyState icon={<YoyoMark compact />} title={copy.skills.emptyInstalled} />;
  }
  const groups = new Map<string, SkillInfo[]>();
  for (const s of props.skills) {
    const key = s.source || "cas";
    const list = groups.get(key) || [];
    list.push(s);
    groups.set(key, list);
  }
  return (
    <div className="space-y-6">
      {[...groups.entries()].map(([source, items]) => (
        <section key={source}>
          <div className="mb-2 flex items-baseline justify-between px-0.5">
            <h3 className="text-[13px] font-medium text-foreground">{sourceLabel(source, copy)}</h3>
            <span className="text-[11px] tabular-nums text-muted">{copy.skills.count.replace("{n}", String(items.length))}</span>
          </div>
          <div className="flex flex-col gap-1">
            {items.map((s) => {
              const isLoaded = props.loaded.has(s.name);
              const expanded = props.open === s.name;
              const slug = packSlug(s);
              const fileCount = s.files ? s.files.split(",").filter(Boolean).length : 0;
              return (
                <div key={s.name} className="u-row-hover rounded-xl px-3 py-2.5">
                  <div className="flex min-h-10 items-center gap-3">
                    <SkillAvatar name={s.displayName || s.name} icon={s.icon} />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-[13px] font-medium text-foreground">{s.displayName || s.name}</span>
                        {isLoaded ? <span className="text-[11px] text-muted">{copy.skills.loaded}</span> : null}
                        {s.incomplete ? <span className="text-[11px] text-warning">{copy.skills.incomplete}</span> : null}
                      </div>
                      <p className="mt-0.5 truncate text-[12px] leading-5 text-muted">{s.description || copy.skills.none}</p>
                    </div>
                    <div className="flex shrink-0 flex-wrap justify-end gap-1.5">
                      {s.incomplete && s.source === "market" ? (
                        <Button size="sm" variant="outline" disabled={props.busy === slug} onClick={() => props.onRepair(slug)}>
                          {props.busy === slug ? copy.skills.repairing : copy.skills.repair}
                        </Button>
                      ) : null}
                      <Button size="sm" variant="ghost" onClick={() => props.onOpen(expanded ? "" : s.name)}>{copy.skills.inspect}</Button>
                      {s.source === "market" ? (
                        <Button size="sm" variant="ghost" disabled={props.busy === slug} onClick={() => props.onUninstall(slug)}>
                          {copy.skills.uninstall}
                        </Button>
                      ) : null}
                    </div>
                  </div>
                  {expanded ? (
                    <div className="mt-2 space-y-2">
                      {fileCount ? (
                        <p className="text-[11px] tabular-nums text-muted">{copy.skills.files.replace("{n}", String(fileCount))}</p>
                      ) : null}
                      {s.body ? (
                        <pre className="max-h-56 overflow-auto whitespace-pre-wrap rounded-md border border-border/60 bg-lift/30 p-2.5 font-mono text-[11px] leading-4 text-muted">{s.body}</pre>
                      ) : null}
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        </section>
      ))}
    </div>
  );
}

function MarketList(props: {
  featured: api.SkillMarketItem[];
  grouped: [string, api.SkillMarketItem[]][];
  categories: string[];
  cat: string;
  onCat: (v: string) => void;
  installed: Set<string>;
  incomplete: Set<string>;
  busy: string;
  onInstall: (slug: string, repairing?: boolean) => void;
  query: string;
  loading: boolean;
}) {
  const copy = useCopy();
  if (props.loading && !props.grouped.length) {
    return <SkillsLoading />;
  }
  if (!props.grouped.length) {
    return <EmptyState icon={<YoyoMark compact />} title={copy.skills.emptyMarket} />;
  }
  return (
    <div>
      {props.categories.length > 1 ? (
        <div className="mb-5 flex flex-wrap gap-1.5">
          <Chip on={!props.cat} onClick={() => props.onCat("")}>{copy.skills.all}</Chip>
          {props.categories.map((c) => (
            <Chip key={c} on={props.cat === c} onClick={() => props.onCat(c)}>{c}</Chip>
          ))}
        </div>
      ) : null}
      {!props.query && !props.cat && props.featured.length ? (
        <section className="mb-6">
          <div className="mb-2 flex items-baseline justify-between px-0.5">
            <h3 className="text-[13px] font-medium">{copy.skills.featured}</h3>
            <span className="text-[11px] tabular-nums text-muted">{copy.skills.count.replace("{n}", String(props.featured.length))}</span>
          </div>
          <div className="overflow-hidden rounded-2xl bg-card shadow-[var(--shadow-card)]">
            {props.featured.map((it, i) => (
              <MarketRow
                key={it.slug}
                item={it}
                installed={props.installed.has(it.slug) || props.installed.has(it.name)}
                incomplete={props.incomplete.has(it.slug) || props.incomplete.has(it.name)}
                busy={props.busy === it.slug}
                onInstall={props.onInstall}
                className={i > 0 ? "border-t border-border/70" : undefined}
              />
            ))}
          </div>
        </section>
      ) : null}
      {props.grouped.map(([category, items]) => (
        <section key={category} className="mb-6">
          <div className="mb-2 flex items-baseline justify-between px-0.5">
            <h3 className="text-[13px] font-medium">{category}</h3>
            <span className="text-[11px] tabular-nums text-muted">{copy.skills.count.replace("{n}", String(items.length))}</span>
          </div>
          <div className="overflow-hidden rounded-2xl bg-card shadow-[var(--shadow-card)]">
            {items.map((it, i) => (
              <MarketRow
                key={it.slug}
                item={it}
                installed={props.installed.has(it.slug) || props.installed.has(it.name)}
                incomplete={props.incomplete.has(it.slug) || props.incomplete.has(it.name)}
                busy={props.busy === it.slug}
                onInstall={props.onInstall}
                className={i > 0 ? "border-t border-border/70" : undefined}
              />
            ))}
          </div>
        </section>
      ))}
      <p className="mt-2 max-w-[68ch] text-[12px] leading-5 text-muted">{copy.skills.attribution}</p>
    </div>
  );
}

function MarketRow(props: {
  item: api.SkillMarketItem;
  installed: boolean;
  incomplete: boolean;
  busy: boolean;
  onInstall: (slug: string, repairing?: boolean) => void;
  className?: string;
}) {
  const copy = useCopy();
  const it = props.item;
  const title = it.displayName || it.name || it.slug;
  return (
    <div className={cn("flex items-start gap-3 px-3.5 py-3 transition-colors duration-[var(--duration-fast)] ease-[var(--ease-out)] hover:bg-lift/40", props.className)}>
      <SkillAvatar name={title} icon={it.icon} />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <div className="text-[13px] font-medium text-foreground">{title}</div>
          {it.hasScripts ? <span className="text-[11px] text-muted">{copy.skills.hasScripts}</span> : null}
          {props.incomplete ? <span className="text-[11px] text-warning">{copy.skills.incomplete}</span> : null}
        </div>
        <p className="mt-0.5 line-clamp-2 text-[13px] leading-5 text-muted">{it.purpose || it.slug}</p>
        {it.prerequisites && it.prerequisites !== "无" ? (
          <p className="mt-1 text-[11px] leading-4 text-muted">{it.prerequisites}</p>
        ) : null}
      </div>
      {props.incomplete ? (
        <Button size="sm" variant="outline" className="mt-0.5 shrink-0 rounded-md" disabled={props.busy} onClick={() => props.onInstall(it.slug, true)}>
          {props.busy ? copy.skills.repairing : copy.skills.repair}
        </Button>
      ) : props.installed ? (
        <span className="mt-0.5 shrink-0 text-[11px] text-muted">{copy.skills.installedBadge}</span>
      ) : (
        <Button size="sm" variant="outline" className="mt-0.5 shrink-0 rounded-md" disabled={props.busy} onClick={() => props.onInstall(it.slug)}>
          {props.busy ? copy.skills.installing : copy.skills.install}
        </Button>
      )}
    </div>
  );
}

function SkillAvatar(props: { name: string; icon?: string }) {
  const [failed, setFailed] = useState(false);
  const showImg = Boolean(props.icon) && !failed;
  const initials = skillInitials(props.name);
  return (
    <span
      className="relative mt-0.5 flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-lift text-[11px] font-medium text-foreground/85"
      aria-hidden
    >
      {showImg ? (
        <img
          src={props.icon}
          alt=""
          referrerPolicy="no-referrer"
          decoding="async"
          className="size-full object-cover"
          onError={() => setFailed(true)}
        />
      ) : initials}
    </span>
  );
}

function skillInitials(name: string): string {
  const parts = name.trim().split(/[\s/_-]+/).filter(Boolean);
  if (!parts.length) return "S";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[1][0]).toUpperCase();
}

function packSlug(s: SkillInfo): string {
  if (s.slug) return s.slug;
  if (s.dir) {
    const parts = s.dir.replace(/\\/g, "/").split("/").filter(Boolean);
    return parts[parts.length - 1] || s.name;
  }
  return s.name;
}

function Chip(props: { on: boolean; onClick: () => void; children: string }) {
  return (
    <button
      type="button"
      className={cn(
        "rounded-md border px-2 py-1 text-[12px] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        props.on ? "border-border bg-lift text-foreground" : "border-transparent text-muted hover:bg-lift/50 hover:text-foreground",
      )}
      aria-pressed={props.on}
      onClick={props.onClick}
    >
      {props.children}
    </button>
  );
}

function sourceLabel(source: string, copy: Copy): string {
  if (source === "bundled") return copy.skills.bundled;
  if (source === "home") return copy.skills.home;
  if (source === "workspace") return copy.skills.workspace;
  if (source === "market") return copy.skills.marketSource;
  if (source === "pack") return copy.skills.packSource;
  return copy.skills.cas;
}

function PacksList(props: {
  packs: PackStatus[];
  loading: boolean;
  busy: string;
  onInstall: (id: string, path?: string) => void;
  onUninstall: (id: string) => void;
  onEnable: (id: string, scope: "workspace" | "global" | "inherit", enabled: boolean) => void;
  onUpdate: (id: string) => void;
}) {
  const copy = useCopy();
  if (props.loading && !props.packs.length) {
    return <SkillsLoading />;
  }
  if (!props.packs.length) {
    return <EmptyState icon={<YoyoMark compact />} title={copy.skills.emptyPacks} />;
  }
  return (
    <div className="space-y-4">
      <p className="max-w-[68ch] text-[12px] leading-5 text-muted">{copy.skills.packHint}</p>
      {props.packs.map((p) => {
        const installing = props.busy === p.id;
        const updating = props.busy === p.id + ":update";
        const togglingWs = props.busy === p.id + ":workspace";
        const togglingGlobal = props.busy === p.id + ":global";
        const togglingInherit = props.busy === p.id + ":inherit";
        return (
          <section key={p.id} className="rounded-2xl bg-card px-4 py-3.5 shadow-[var(--shadow-card)]" data-testid={`pack-${p.id}`}>
            <div className="flex items-start gap-3">
              <SkillAvatar name={p.name} />
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <h3 className="text-[14px] font-medium text-foreground">{p.name}</h3>
                  {p.methodology ? <span className="text-[11px] text-muted">{copy.skills.methodology}</span> : null}
                  {p.enabled ? (
                    <span className="text-[11px] text-muted">{copy.skills.packEnabled}</span>
                  ) : p.installed ? (
                    <span className="text-[11px] text-muted">{copy.skills.defaultOff}</span>
                  ) : (
                    <span className="text-[11px] text-muted">{copy.skills.packDisabled}</span>
                  )}
                </div>
                <p className="mt-0.5 text-[13px] leading-5 text-muted">{p.description}</p>
                <p className="mt-1 text-[11px] tabular-nums text-muted">
                  {copy.skills.skillCount.replace("{n}", String(p.skillCount || p.skills?.length || 0))}
                  {p.license ? ` · ${copy.skills.packLicense} ${p.license}` : ""}
                  {p.origin?.repo ? ` · ${p.origin.repo}` : ""}
                </p>
                {p.skills?.length ? (
                  <p className="mt-1 truncate text-[11px] text-muted">{p.skills.join(" · ")}</p>
                ) : null}
              </div>
            </div>
            <div className="mt-3 flex flex-wrap justify-end gap-1.5">
              {!p.installed ? (
                <>
                  <Button size="sm" variant="outline" disabled={installing} onClick={() => props.onInstall(p.id)}>
                    {installing ? copy.skills.installing : copy.skills.fromGitHub}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled={installing}
                    onClick={() => {
                      void api.pickFolder().then((path) => {
                        if (path) props.onInstall(p.id, path);
                      });
                    }}
                  >
                    {copy.skills.fromLocal}
                  </Button>
                </>
              ) : (
                <>
                  <Button
                    size="sm"
                    variant={p.enabled ? "outline" : "ghost"}
                    disabled={togglingWs}
                    onClick={() => props.onEnable(p.id, "workspace", !p.enabled)}
                  >
                    {p.enabled ? copy.skills.disableWorkspace : copy.skills.enableWorkspace}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled={togglingGlobal}
                    onClick={() => props.onEnable(p.id, "global", !p.enableGlobal)}
                  >
                    {p.enableGlobal ? copy.skills.disableGlobal : copy.skills.enableGlobal}
                  </Button>
                  {p.enableWorkspace != null ? (
                    <Button size="sm" variant="ghost" disabled={togglingInherit} onClick={() => props.onEnable(p.id, "inherit", true)}>
                      {copy.skills.inheritGlobal}
                    </Button>
                  ) : null}
                  <Button size="sm" variant="ghost" disabled={updating} onClick={() => props.onUpdate(p.id)}>
                    {updating ? copy.skills.updating : copy.skills.update}
                  </Button>
                  <Button size="sm" variant="ghost" disabled={installing} onClick={() => props.onUninstall(p.id)}>
                    {copy.skills.uninstall}
                  </Button>
                </>
              )}
            </div>
          </section>
        );
      })}
    </div>
  );
}
