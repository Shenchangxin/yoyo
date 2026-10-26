/**
 * 拥有：已装技能、市场和技能包三页。
 * 不拥有：已装列表的初次装载，那由调用方传入。安装、卸载、启用和更新会调用 api。
 */

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

/** 和市场页签同一套取值。 */
type Pane = SkillsTab;

/** 这些源站自己能搜。当前源站的 search 为真时也按远程搜。 */
const REMOTE_SEARCH_HUBS = new Set(["clawhub", "clawskillhub", "skillhubcn", "skillsmp", "mdr", "tessl"]);

/** 挂上时打开全局模块加载，卸下时关掉。 */
function SkillsLoading() {
  useEffect(() => {
    useUI.getState().setModuleLoading(true);
    return () => useUI.getState().setModuleLoading(false);
  }, []);
  return (
    <PresenceModuleLoading />
  );
}

/**
 * SkillsWorkspace 页签跟 store 里的 skillsTab 走。换源站时清掉条目、分类、警告和来源。远程搜索有查询时延迟 400 毫秒再拉，否则立刻拉。新目录没有 hubs 时保留上一份。
 * props 读取 workspace、installed、loaded，调用 onRefreshInstalled。
 */
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
  const [hub, setHub] = useState("workbuddy");
  const [catalog, setCatalog] = useState<api.SkillMarketCatalog>({ source: "", fetchedAt: "", items: [], hubs: [] });
  const [packs, setPacks] = useState<PackStatus[]>([]);
  const [packsLoading, setPacksLoading] = useState(true);

  useEffect(() => { setPane(skillsTab); }, [skillsTab]);

  /** refresh 为真时强制刷新。hubs 为空则沿用上一份。nextHub 传给 api.skillMarket。nextQ 传给 api.skillMarket。 */
  const loadMarket = (refresh = false, nextHub = hub, nextQ = "") => {
    setErr("");
    setLoading(true);
    void api.skillMarket(refresh, nextHub, nextQ)
      .then((c) => setCatalog((prev) => ({ ...c, hubs: c.hubs?.length ? c.hubs : prev.hubs })))
      .catch((e) => setErr(api.errMessage(e)))
      .finally(() => setLoading(false));
  };

  /** 按当前工作区拉技能包。失败写进 err。 */
  const loadPacks = () => {
    setPacksLoading(true);
    void api.listPacks(props.workspace)
      .then(setPacks)
      .catch((e) => setErr(api.errMessage(e)))
      .finally(() => setPacksLoading(false));
  };

  const hubSearches = Boolean((catalog.hubs || []).find((h) => h.id === hub)?.search) || REMOTE_SEARCH_HUBS.has(hub);
  const fetchQ = hubSearches ? q.trim() : "";
  useEffect(() => {
    const delay = hubSearches && fetchQ ? 400 : 0;
    const t = window.setTimeout(() => loadMarket(false, hub, fetchQ), delay);
    return () => window.clearTimeout(t);
  }, [hub, fetchQ]);
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
      if (s.installId) set.add(s.installId);
    }
    return set;
  }, [props.installed]);
  const installedIds = useMemo(() => {
    const set = new Set<string>();
    for (const s of props.installed) {
      if (s.installId) set.add(s.installId);
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
    const remote = hubSearches;
    return catalog.items.filter((it) => {
      if (cat && it.category !== cat) return false;
      if (!needle || remote) return true;
      return (it.slug + it.name + (it.displayName || "") + it.purpose + it.category + (it.owner || "")).toLowerCase().includes(needle);
    });
  }, [catalog.items, q, cat, hubSearches]);

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

  /** 有 warning 时提示不完整，修复成功另提示。失败只记下 err。id 传给 setBusy，传给 api.installMarketSkill。repairing 为真时才继续。 */
  async function install(id: string, repairing = false) {
    setBusy(id);
    setErr("");
    try {
      const out = await api.installMarketSkill(id);
      props.onRefreshInstalled();
      if (out.warning) toast.message(copy.skills.installPartial);
      else if (repairing) toast.success(copy.skills.repaired);
    } catch (e) {
      setErr(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  /** 卸掉市场技能后刷新已装列表。slug 传给 setBusy，传给 api.uninstallMarketSkill。 */
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

  /** 本页和 store 一起换页签。id 传给 setPane，传给 setSkillsTab。 */
  function selectPane(id: Pane) {
    setPane(id);
    setSkillsTab(id);
  }

  /** path 为空走默认安装，非空走本地路径。成功后刷新包和已装技能。id 传给 setBusy，传给 api.installPack。 */
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

  /** 卸掉技能包后刷新包和已装技能。id 传给 setBusy，传给 api.uninstallPack。 */
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

  /** 忙状态的键是 id 加冒号加 scope。inherit 的调用方传入的 enabled 固定为真。 */
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

  /** 忙状态的键是 id 加 :update。 */
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
          <div className="mb-5 flex flex-wrap items-center gap-3">
            <div className="process-tabs flex h-10 items-stretch gap-0.5" role="tablist" aria-label={copy.skills.title}>
              {(["installed", "market", "packs"] as Pane[]).map((id) => (
                <button
                  type="button"
                  key={id}
                  role="tab"
                  aria-selected={pane === id}
                  className={cn(
                    "relative flex h-10 cursor-pointer items-center px-2.5 text-[13px] font-medium transition-colors",
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
                    placeholder={hubSearches ? copy.skills.searchRemote : copy.skills.search}
                    value={q}
                    onChange={(e) => setQ(e.target.value)}
                    aria-label={hubSearches ? copy.skills.searchRemote : copy.skills.search}
                    autoComplete="off"
                    name="skill-catalog-search"
                    spellCheck={false}
                  />
                </div>
                <Button size="sm" variant="outline" onClick={() => loadMarket(true, hub, fetchQ)}>
                  <RefreshCw className="size-3.5" aria-hidden />
                  {copy.skills.refresh}
                </Button>
              </div>
            ) : null}
          </div>
          {err ? <p className="mb-4 text-[13px] text-danger" role="alert">{err}</p> : null}
          {!err && catalog.warning ? <p className="mb-4 text-[13px] text-muted" role="status">{catalog.warning}</p> : null}
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
              hubs={catalog.hubs || []}
              hub={hub}
              onHub={(id) => {
                setHub(id);
                setCat("");
                setCatalog((prev) => ({ ...prev, items: [], searchOnly: false, warning: "", source: "" }));
              }}
              searchOnly={Boolean(catalog.searchOnly) && !q.trim()}
              attribution={catalog.attribution || copy.skills.attribution}
              installed={installedNames}
              installedIds={installedIds}
              incomplete={incomplete}
              busy={busy}
              onInstall={install}
              query={q}
              loading={loading}
              shown={catalog.items.length}
              total={catalog.total || 0}
              truncated={Boolean(catalog.truncated)}
            />
          )}
        </div>
      </div>
    </div>
  );
}

/** 没有技能时显示空状态。修复按钮只给来源是 market 且不完整的技能。卸载也只给 market。检查来回切换同一条。props 读取 skills、loaded、open、busy，调用 onRepair，调用 onOpen，调用 onUninstall。 */
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
                        <Button size="sm" variant="outline" disabled={props.busy === packInstallKey(s) || props.busy === slug} onClick={() => props.onRepair(packInstallKey(s))}>
                          {props.busy === packInstallKey(s) || props.busy === slug ? copy.skills.repairing : copy.skills.repair}
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

/**
 * 加载中且还没有分组时显示加载。searchOnly 优先于空市场。精选只在没有查询、没有分类且有精选时出现。分类芯片要多于 1 个才画。total 大于 shown 时用带总数的文案，truncated 再加省略号。
 * props 读取 grouped、total、shown、hubs、hub、truncated 等，调用 onHub，调用 onCat。
 */
function MarketList(props: {
  featured: api.SkillMarketItem[];
  grouped: [string, api.SkillMarketItem[]][];
  categories: string[];
  cat: string;
  onCat: (v: string) => void;
  hubs: api.SkillMarketHub[];
  hub: string;
  onHub: (id: string) => void;
  searchOnly: boolean;
  attribution: string;
  installed: Set<string>;
  installedIds: Set<string>;
  incomplete: Set<string>;
  busy: string;
  onInstall: (id: string, repairing?: boolean) => void;
  query: string;
  loading: boolean;
  shown: number;
  total: number;
  truncated: boolean;
}) {
  const copy = useCopy();
  const empty = !props.grouped.length;
  const countLabel = props.total > props.shown
    ? copy.skills.countOf.replace("{n}", String(props.shown)).replace("{total}", String(props.total))
    : copy.skills.count.replace("{n}", String(props.shown));
  return (
    <div>
      {props.hubs.length ? (
        <div className="mb-4 flex flex-wrap gap-1.5" role="tablist" aria-label={copy.skills.hubs}>
          {props.hubs.map((h) => (
            <Chip key={h.id} on={props.hub === h.id} title={h.attribution} onClick={() => props.onHub(h.id)}>{h.name}</Chip>
          ))}
        </div>
      ) : null}
      {!empty ? (
        <p className="mb-3 text-[11px] tabular-nums text-muted">{countLabel}{props.truncated ? " · …" : ""}</p>
      ) : null}
      {props.categories.length > 1 ? (
        <div className="mb-5 flex flex-wrap gap-1.5">
          <Chip on={!props.cat} onClick={() => props.onCat("")}>{copy.skills.all}</Chip>
          {props.categories.map((c) => (
            <Chip key={c} on={props.cat === c} onClick={() => props.onCat(c)}>{c}</Chip>
          ))}
        </div>
      ) : null}
      {props.loading && empty ? (
        <SkillsLoading />
      ) : props.searchOnly ? (
        <EmptyState icon={<YoyoMark compact />} title={copy.skills.searchOnly} />
      ) : empty ? (
        <EmptyState icon={<YoyoMark compact />} title={copy.skills.emptyMarket} />
      ) : (
        <>
          {!props.query && !props.cat && props.featured.length ? (
            <section className="mb-6">
              <div className="mb-2 flex items-baseline justify-between px-0.5">
                <h3 className="text-[13px] font-medium">{copy.skills.featured}</h3>
                <span className="text-[11px] tabular-nums text-muted">{copy.skills.count.replace("{n}", String(props.featured.length))}</span>
              </div>
              <div className="overflow-hidden rounded-2xl bg-card shadow-[var(--shadow-card)]">
                {props.featured.map((it, i) => (
                  <MarketRow
                    key={itemInstallID(it)}
                    item={it}
                    installed={itemInstalled(it, props.installed, props.installedIds)}
                    incomplete={itemIncomplete(it, props.incomplete)}
                    busy={props.busy === itemInstallID(it)}
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
                    key={itemInstallID(it)}
                    item={it}
                    installed={itemInstalled(it, props.installed, props.installedIds)}
                    incomplete={itemIncomplete(it, props.incomplete)}
                    busy={props.busy === itemInstallID(it)}
                    onInstall={props.onInstall}
                    className={i > 0 ? "border-t border-border/70" : undefined}
                  />
                ))}
              </div>
            </section>
          ))}
        </>
      )}
      <p className="mt-2 max-w-[68ch] text-[12px] leading-5 text-muted">{props.attribution || copy.skills.attribution}</p>
    </div>
  );
}

/** 不完整时显示修复，已装时只显示徽章，否则显示安装。前提是「无」或空时不画。props 读取 item、className、incomplete、busy、installed，调用 onInstall。 */
function MarketRow(props: {
  item: api.SkillMarketItem;
  installed: boolean;
  incomplete: boolean;
  busy: boolean;
  onInstall: (id: string, repairing?: boolean) => void;
  className?: string;
}) {
  const copy = useCopy();
  const it = props.item;
  const title = it.displayName || it.name || it.slug;
  const id = itemInstallID(it);
  const meta: string[] = [];
  if (it.owner) meta.push("@" + it.owner);
  if (it.downloads) meta.push(copy.skills.downloads.replace("{n}", String(it.downloads)));
  if (it.stars) meta.push(copy.skills.stars.replace("{n}", String(it.stars)));
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
        {meta.length ? (
          <p className="mt-1 text-[11px] leading-4 text-muted">{meta.join(" · ")}</p>
        ) : null}
        {it.prerequisites && it.prerequisites !== "无" ? (
          <p className="mt-1 text-[11px] leading-4 text-muted">{it.prerequisites}</p>
        ) : null}
      </div>
      {props.incomplete ? (
        <Button size="sm" variant="outline" className="mt-0.5 shrink-0 rounded-md" disabled={props.busy} onClick={() => props.onInstall(id, true)}>
          {props.busy ? copy.skills.repairing : copy.skills.repair}
        </Button>
      ) : props.installed ? (
        <span className="mt-0.5 shrink-0 text-[11px] text-muted">{copy.skills.installedBadge}</span>
      ) : (
        <Button size="sm" variant="outline" className="mt-0.5 shrink-0 rounded-md" disabled={props.busy} onClick={() => props.onInstall(id)}>
          {props.busy ? copy.skills.installing : copy.skills.install}
        </Button>
      )}
    </div>
  );
}

/** 图标加载失败就改用首字母。图片不发送 referrer。props 读取 icon、name。 */
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

/** 空名字用 S。一个词取前两个字符。多个词取前两个词的首字母。name 去掉首尾空白，切开。 */
function skillInitials(name: string): string {
  const parts = name.trim().split(/[\s/_-]+/).filter(Boolean);
  if (!parts.length) return "S";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[1][0]).toUpperCase();
}

/** 有 slug 用 slug。否则用目录最后一段，再没有用 name。s 读取 slug、dir、name。 */
function packSlug(s: SkillInfo): string {
  if (s.slug) return s.slug;
  if (s.dir) {
    const parts = s.dir.replace(/\\/g, "/").split("/").filter(Boolean);
    return parts[parts.length - 1] || s.name;
  }
  return s.name;
}

/** 有 installId 用它，否则用 packSlug。s 读取 installId，传给 packSlug。 */
function packInstallKey(s: SkillInfo): string {
  return s.installId || packSlug(s);
}

/** 优先 id。否则 hub、owner 和 slug 都在时拼成 hub:owner/slug。只有 hub 和 slug 时拼成 hub:slug。最后退回 slug。it 读取 id、hub、owner、slug。 */
function itemInstallID(it: api.SkillMarketItem): string {
  if (it.id) return it.id;
  if (it.hub && it.owner && it.slug) return `${it.hub}:${it.owner}/${it.slug}`;
  if (it.hub && it.slug) return `${it.hub}:${it.slug}`;
  return it.slug;
}

/** 安装 id 命中，或 slug、name 命中已装名字，算已装。it 读取 slug、name，传给 itemInstallID。names 调用 has。ids 调用 has。 */
function itemInstalled(it: api.SkillMarketItem, names: Set<string>, ids: Set<string>): boolean {
  const id = itemInstallID(it);
  if (id && ids.has(id)) return true;
  return names.has(it.slug) || names.has(it.name);
}

/** 安装 id、slug 或 name 落在不完整集合里就算不完整。it 读取 slug、name，传给 itemInstallID。incomplete 调用 has。 */
function itemIncomplete(it: api.SkillMarketItem, incomplete: Set<string>): boolean {
  const id = itemInstallID(it);
  if (id && incomplete.has(id)) return true;
  return incomplete.has(it.slug) || incomplete.has(it.name);
}

/** 按下状态跟 on 走。开着时有边框和抬起底。props 读取 title、on、onClick、children。 */
function Chip(props: { on: boolean; onClick: () => void; children: string; title?: string }) {
  return (
    <button
      type="button"
      title={props.title}
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

/** bundled、home、workspace、market、pack 各有文案，其余当 cas。source 与 bundled、home、workspace、market、pack 比较是否相等。copy 读取 skills。 */
function sourceLabel(source: string, copy: Copy): string {
  if (source === "bundled") return copy.skills.bundled;
  if (source === "home") return copy.skills.home;
  if (source === "workspace") return copy.skills.workspace;
  if (source === "market") return copy.skills.marketSource;
  if (source === "pack") return copy.skills.packSource;
  return copy.skills.cas;
}

/**
 * 加载中且没有包时显示加载。本地安装只在选到文件夹后才调用。工作区开关看 enabled，全局开关看 enableGlobal。inherit 按钮只在 enableWorkspace 不是 null 时出现，并传入 enabled 为真。
 * props 读取 loading、packs、busy，调用 onInstall，调用 onEnable，调用 onUpdate，调用 onUninstall。
 */
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
                  {p.methodology ? (
                    <span className="text-[11px] text-muted">{copy.skills.methodology}</span>
                  ) : p.known ? (
                    <span className="text-[11px] text-muted">{copy.skills.domain}</span>
                  ) : null}
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
