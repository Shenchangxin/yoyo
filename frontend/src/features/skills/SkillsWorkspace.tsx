import { useEffect, useMemo, useState } from "react";
import { Pin, RefreshCw, Search } from "lucide-react";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { EmptyState } from "../../components/ui/empty-state";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import type { SkillInfo } from "../../lib/protocol";
import type { Copy } from "../../lib/copy";

type Pane = "installed" | "market";

export function SkillsWorkspace(props: {
  installed: SkillInfo[];
  pinned: string[];
  loaded: string[];
  threadId?: string;
  onPinSkills?: (names: string[]) => void;
  onRefreshInstalled: () => void;
}) {
  const copy = useCopy();
  const [pane, setPane] = useState<Pane>("market");
  const [q, setQ] = useState("");
  const [cat, setCat] = useState("");
  const [open, setOpen] = useState("");
  const [busy, setBusy] = useState("");
  const [err, setErr] = useState("");
  const [catalog, setCatalog] = useState<api.SkillMarketCatalog>({ source: "", fetchedAt: "", items: [] });

  const loadMarket = (refresh = false) => {
    setErr("");
    void api.skillMarket(refresh).then(setCatalog).catch((e) => setErr(api.errMessage(e)));
  };

  useEffect(() => { loadMarket(false); }, []);

  const installedNames = useMemo(() => new Set(props.installed.map((s) => s.name)), [props.installed]);
  const pinned = new Set(props.pinned);
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
      return (it.slug + it.name + it.purpose + it.category).toLowerCase().includes(needle);
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

  async function install(slug: string) {
    setBusy(slug);
    setErr("");
    try {
      await api.installMarketSkill(slug);
      props.onRefreshInstalled();
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

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="skills-workspace">
      <div className="min-h-0 flex-1 overflow-auto">
        <div className="mx-auto w-full max-w-5xl px-6 py-5">
          <p className="mb-4 max-w-[62ch] text-[13px] leading-5 text-muted">{copy.skills.hint}</p>
          <div className="mb-5 flex flex-wrap items-center gap-3">
            <div className="flex rounded-lg border border-border/80 bg-lift/30 p-0.5" role="tablist" aria-label={copy.skills.title}>
              {(["installed", "market"] as Pane[]).map((id) => (
                <button
                  type="button"
                  key={id}
                  role="tab"
                  aria-selected={pane === id}
                  className={cn(
                    "rounded-md px-3 py-1.5 text-[12.5px] font-medium transition-colors",
                    pane === id ? "bg-panel text-foreground" : "text-muted hover:text-foreground",
                  )}
                  onClick={() => setPane(id)}
                >
                  {id === "installed" ? copy.skills.installed : copy.skills.market}
                </button>
              ))}
            </div>
            {pane === "market" ? (
              <>
                <div className="relative min-w-[12rem] flex-1">
                  <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted" aria-hidden />
                  <Input
                    className="h-8 rounded-lg border-border/80 bg-transparent pl-8 text-[13px]"
                    placeholder={copy.skills.search}
                    value={q}
                    onChange={(e) => setQ(e.target.value)}
                    aria-label={copy.skills.search}
                  />
                </div>
                <Button size="sm" variant="outline" onClick={() => loadMarket(true)}>
                  <RefreshCw className="size-3.5" aria-hidden />
                  {copy.skills.refresh}
                </Button>
              </>
            ) : null}
          </div>
          {err ? <p className="mb-4 text-[12.5px] text-danger">{err}</p> : null}
          {pane === "installed" ? (
            <InstalledList
              skills={props.installed}
              pinned={pinned}
              loaded={loaded}
              open={open}
              onOpen={setOpen}
              threadId={props.threadId}
              onPin={props.onPinSkills}
              pinNames={props.pinned}
              busy={busy}
              onUninstall={uninstall}
            />
          ) : (
            <MarketList
              featured={featured}
              grouped={grouped}
              categories={categories}
              cat={cat}
              onCat={setCat}
              installed={installedNames}
              busy={busy}
              onInstall={install}
              query={q}
            />
          )}
        </div>
      </div>
    </div>
  );
}

function InstalledList(props: {
  skills: SkillInfo[];
  pinned: Set<string>;
  loaded: Set<string>;
  open: string;
  onOpen: (name: string) => void;
  threadId?: string;
  onPin?: (names: string[]) => void;
  pinNames: string[];
  busy: string;
  onUninstall: (slug: string) => void;
}) {
  const copy = useCopy();
  if (!props.skills.length) {
    return <EmptyState title={copy.skills.emptyInstalled} />;
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
          <div className="overflow-hidden rounded-[10px] border border-border/80 bg-card">
            {items.map((s, i) => {
              const isPinned = props.pinned.has(s.name);
              const isLoaded = props.loaded.has(s.name);
              const expanded = props.open === s.name;
              return (
                <div key={s.name} className={cn("px-3.5 py-3", i > 0 && "border-t border-border/70")}>
                  <div className="flex items-start gap-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-[13px] font-medium text-foreground">{s.name}</span>
                        {isLoaded ? <span className="text-[11px] text-muted">{copy.skills.loaded}</span> : null}
                        {isPinned ? <span className="text-[11px] text-muted">{copy.review.pinned}</span> : null}
                      </div>
                      <p className="mt-0.5 line-clamp-2 text-[12.5px] leading-5 text-muted">{s.description || copy.skills.none}</p>
                    </div>
                    <div className="flex shrink-0 flex-wrap justify-end gap-1.5">
                      <Button
                        size="sm"
                        variant={isPinned ? "lift" : "ghost"}
                        disabled={!props.threadId}
                        title={!props.threadId ? copy.skills.noThread : undefined}
                        onClick={() => {
                          const next = isPinned ? props.pinNames.filter((n) => n !== s.name) : [...props.pinNames, s.name];
                          props.onPin?.(next);
                        }}
                      >
                        <Pin className="size-3" aria-hidden />
                        {isPinned ? copy.skills.unpinThread : copy.skills.pinThread}
                      </Button>
                      <Button size="sm" variant="ghost" onClick={() => props.onOpen(expanded ? "" : s.name)}>{copy.skills.inspect}</Button>
                      {s.source === "market" ? (
                        <Button size="sm" variant="ghost" disabled={props.busy === s.name} onClick={() => props.onUninstall(s.name)}>
                          {copy.skills.uninstall}
                        </Button>
                      ) : null}
                    </div>
                  </div>
                  {expanded && s.body ? (
                    <pre className="mt-2 max-h-56 overflow-auto whitespace-pre-wrap rounded-md border border-border/60 bg-lift/30 p-2.5 font-mono text-[11px] leading-4 text-muted">{s.body}</pre>
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
  busy: string;
  onInstall: (slug: string) => void;
  query: string;
}) {
  const copy = useCopy();
  if (!props.grouped.length) {
    return <EmptyState title={copy.skills.emptyMarket} />;
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
          <div className="grid overflow-hidden rounded-[10px] border border-border/80 bg-card sm:grid-cols-2">
            {props.featured.map((it, i) => (
              <MarketRow
                key={it.slug}
                item={it}
                installed={props.installed.has(it.slug) || props.installed.has(it.name)}
                busy={props.busy === it.slug}
                onInstall={props.onInstall}
                className={cn(i > 0 && "border-t border-border/70 sm:border-t-0", i % 2 === 1 && "sm:border-l sm:border-border/70", i > 1 && "sm:border-t sm:border-border/70")}
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
          <div className="overflow-hidden rounded-[10px] border border-border/80 bg-card">
            {items.map((it, i) => (
              <MarketRow
                key={it.slug}
                item={it}
                installed={props.installed.has(it.slug) || props.installed.has(it.name)}
                busy={props.busy === it.slug}
                onInstall={props.onInstall}
                className={i > 0 ? "border-t border-border/70" : undefined}
              />
            ))}
          </div>
        </section>
      ))}
      <p className="mt-2 max-w-[68ch] text-[11.5px] leading-5 text-muted">{copy.skills.attribution}</p>
    </div>
  );
}

function MarketRow(props: {
  item: api.SkillMarketItem;
  installed: boolean;
  busy: boolean;
  onInstall: (slug: string) => void;
  className?: string;
}) {
  const copy = useCopy();
  const it = props.item;
  return (
    <div className={cn("flex items-start gap-3 px-3.5 py-3", props.className)}>
      <div className="min-w-0 flex-1">
        <div className="text-[13px] font-medium text-foreground">{it.name || it.slug}</div>
        <p className="mt-0.5 line-clamp-2 text-[12.5px] leading-5 text-muted">{it.purpose || it.slug}</p>
        {it.prerequisites && it.prerequisites !== "无" ? (
          <p className="mt-1 text-[11px] leading-4 text-muted">{it.prerequisites}</p>
        ) : null}
      </div>
      {props.installed ? (
        <span className="mt-0.5 shrink-0 text-[11px] text-muted">{copy.skills.installedBadge}</span>
      ) : (
        <Button size="sm" variant="outline" className="mt-0.5 shrink-0 rounded-md" disabled={props.busy} onClick={() => props.onInstall(it.slug)}>
          {props.busy ? copy.skills.installing : copy.skills.install}
        </Button>
      )}
    </div>
  );
}

function Chip(props: { on: boolean; onClick: () => void; children: string }) {
  return (
    <button
      type="button"
      className={cn(
        "rounded-md border px-2 py-1 text-[11.5px] transition-colors",
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
  return copy.skills.cas;
}
