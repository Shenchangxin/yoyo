/**
 * 拥有：已装和市场两页。技能合集（后端仍按包安装）不单列一页：市场顶部可一键装，已装页按合集分组，组头能开关、更新、卸载。
 * 不拥有：已装列表的初次装载，那由调用方传入。安装、卸载、启用和更新会调用 api。
 */

import { useEffect, useMemo, useState } from "react";
import { Info, PackageCheck, RefreshCw, Search, Store, X } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { Tooltip } from "../../components/ui/tooltip";
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
const REMOTE_SEARCH_HUBS = new Set(["skillhubcn"]);

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
  /** 给了就在工具栏右端画关闭按钮。对话框里传，独立页面不传。 */
  onClose?: () => void;
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
  // 和后端的 DefaultHub 一致。市场只列 SkillHub 中国和 Anthropic 两个站。
  const [hub, setHub] = useState("skillhubcn");
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
  const installedPacks = useMemo(() => packs.filter((p) => p.installed), [packs]);
  /** 市场顶部的合集跟着搜索词过滤；选了分类时不出现，分类只属于源站条目。 */
  const collections = useMemo(() => {
    if (cat) return [];
    const needle = q.trim().toLowerCase();
    return packs.filter((p) => !needle || (p.id + p.name + p.description).toLowerCase().includes(needle));
  }, [packs, q, cat]);
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

  /** 合集只有一个开关：有工作区写工作区，没有写全局。忙状态的键是 id 加 :toggle。 */
  async function togglePack(id: string, enabled: boolean) {
    setBusy(id + ":toggle");
    setErr("");
    try {
      await api.enablePack(id, props.workspace ? "workspace" : "global", enabled);
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

  const paneLabel = (id: Pane) => (id === "installed" ? copy.skills.installed : copy.skills.market);
  const paneIcon = (id: Pane) => (id === "installed" ? <PackageCheck className="size-4 shrink-0" aria-hidden /> : <Store className="size-4 shrink-0" aria-hidden />);

  return (
    <div className="skills-shell flex h-full min-h-0" data-testid="skills-workspace">
      <aside className="skills-aside flex w-52 shrink-0 flex-col gap-1 p-3">
        <div className="skills-aside-head px-2 pb-3 pt-1" aria-hidden>
          <div className="text-ui-title font-medium text-foreground">{copy.skills.title}</div>
          <p className="mt-1 text-ui-footnote leading-5 text-muted">{copy.skills.sheetHint}</p>
        </div>
        <div className="flex flex-col gap-0.5" role="tablist" aria-orientation="vertical" aria-label={copy.skills.title}>
          {(["installed", "market"] as Pane[]).map((id) => (
            <button
              type="button"
              key={id}
              role="tab"
              aria-selected={pane === id}
              className="skills-nav-item cursor-pointer text-ui-body font-medium"
              onClick={() => selectPane(id)}
            >
              {paneIcon(id)}
              <span className="min-w-0 flex-1 truncate">{paneLabel(id)}</span>
              {id === "installed" && props.installed.length ? (
                <span className="text-ui-caption tabular-nums text-muted">{props.installed.length}</span>
              ) : null}
            </button>
          ))}
        </div>
        <p className="skills-aside-foot mt-auto flex items-start gap-1 px-2 text-ui-footnote leading-5 text-muted">
          <span>{copy.skills.hint}</span>
          <Tooltip content={<span className="block max-w-[44ch]">{copy.skills.hintMore}</span>}>
            <button type="button" className="grid size-5 shrink-0 place-items-center rounded-full text-muted hover:text-foreground" aria-label={copy.skills.hintMore}>
              <Info className="size-3.5" />
            </button>
          </Tooltip>
        </p>
      </aside>
      <section className="flex min-h-0 min-w-0 flex-1 flex-col">
        <header className="flex h-14 shrink-0 items-center gap-3 border-b border-border pl-5 pr-3">
          <h3 className="shrink-0 text-ui-body font-medium text-foreground">{paneLabel(pane)}</h3>
          <div className="flex min-w-0 flex-1 items-center justify-end gap-2">
            {pane === "market" ? (
              <>
                <div className="relative w-full max-w-[16rem]">
                  <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted" aria-hidden />
                  <Input
                    className="h-8 rounded-lg border-transparent bg-lift/50 pl-8 text-ui-body"
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
              </>
            ) : null}
          </div>
          {props.onClose ? (
            <button
              type="button"
              className="grid size-8 shrink-0 place-items-center rounded-lg text-muted transition-colors hover:bg-lift hover:text-foreground"
              aria-label={copy.skills.close}
              data-testid="skills-sheet-close"
              onClick={props.onClose}
            >
              <X className="size-4" />
            </button>
          ) : null}
        </header>
        <div className="min-h-0 flex-1 overflow-auto overscroll-contain">
          <div className="w-full px-5 py-5">
          {err ? <p className="mb-4 text-ui-body text-danger" role="alert">{err}</p> : null}
          {!err && catalog.warning ? <p className="mb-4 text-ui-body text-muted" role="status">{catalog.warning}</p> : null}
          {pane === "installed" ? (
            <InstalledList
              skills={props.installed}
              loaded={loaded}
              open={open}
              onOpen={setOpen}
              busy={busy}
              onUninstall={uninstall}
              onRepair={(slug) => install(slug, true)}
              packs={installedPacks}
              onPackToggle={togglePack}
              onPackUpdate={refreshPack}
              onPackUninstall={uninstallKnownPack}
            />
          ) : (
            <>
            <CollectionList packs={collections} loading={packsLoading} busy={busy} onInstall={installKnownPack} />
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
            </>
          )}
          </div>
        </div>
      </section>
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
  /** 已安装的合集。每个合集自成一组，排在普通来源前面。 */
  packs: PackStatus[];
  onPackToggle: (id: string, enabled: boolean) => void;
  onPackUpdate: (id: string) => void;
  onPackUninstall: (id: string) => void;
}) {
  const copy = useCopy();
  if (!props.skills.length && !props.packs.length) {
    return <EmptyState icon={<YoyoMark compact />} title={copy.skills.emptyInstalled} />;
  }
  const packIds = new Set(props.packs.map((p) => p.id));
  const groups = new Map<string, SkillInfo[]>();
  for (const s of props.skills) {
    // 属于已装合集的技能归到合集组里；对不上合集的仍按来源分组，不会丢。
    if (s.pack && packIds.has(s.pack)) continue;
    const key = s.source || "cas";
    const list = groups.get(key) || [];
    list.push(s);
    groups.set(key, list);
  }
  const sections: { key: string; pack?: PackStatus; items: SkillInfo[] }[] = [
    ...props.packs.map((p) => ({ key: "pack:" + p.id, pack: p, items: p.enabled ? props.skills.filter((s) => s.pack === p.id) : [] })),
    ...[...groups.entries()].map(([key, items]) => ({ key, items })),
  ];
  return (
    <div className="space-y-6">
      {sections.map(({ key, pack, items }) => (
        <section key={key} data-testid={pack ? `collection-${pack.id}` : undefined}>
          {pack ? (
            <CollectionHead pack={pack} busy={props.busy} onToggle={props.onPackToggle} onUpdate={props.onPackUpdate} onUninstall={props.onPackUninstall} />
          ) : (
            <div className="mb-2 flex items-baseline justify-between px-0.5">
              <h3 className="text-ui-body font-medium text-foreground">{sourceLabel(key, copy)}</h3>
              <span className="text-ui-caption tabular-nums text-muted">{copy.skills.count.replace("{n}", String(items.length))}</span>
            </div>
          )}
          {pack && !items.length ? (
            <p className="rounded-xl border border-dashed border-border px-3.5 py-2.5 text-ui-footnote leading-5 text-muted">
              {pack.enabled ? (pack.skills || []).join(" · ") || copy.skills.none : copy.skills.collectionOffHint}
            </p>
          ) : null}
            <div className="flex flex-col gap-1.5">
            {items.map((s) => {
              const isLoaded = props.loaded.has(s.name);
              const expanded = props.open === s.name;
              const slug = packSlug(s);
              const fileCount = s.files ? s.files.split(",").filter(Boolean).length : 0;
              return (
                <div key={s.name} className="u-card-hover rounded-xl border border-border bg-card px-3.5 py-2.5 shadow-[var(--shadow-card)]">
                  <div className="flex min-h-10 items-center gap-3">
                    <SkillAvatar name={s.displayName || s.name} icon={s.icon} />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-ui-body font-medium text-foreground">{s.displayName || s.name}</span>
                        {isLoaded ? <span className="text-ui-caption text-muted">{copy.skills.loaded}</span> : null}
                        {s.incomplete ? <span className="text-ui-caption text-warning">{copy.skills.incomplete}</span> : null}
                      </div>
                      <p className="mt-0.5 truncate text-ui-footnote leading-5 text-muted">{s.description || copy.skills.none}</p>
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
                        <p className="text-ui-caption tabular-nums text-muted">{copy.skills.files.replace("{n}", String(fileCount))}</p>
                      ) : null}
                      {s.body ? (
                        <pre className="max-h-56 overflow-auto whitespace-pre-wrap rounded-md border border-border/60 bg-lift/30 p-2.5 font-mono text-ui-caption leading-4 text-muted">{s.body}</pre>
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
        <p className="mb-3 text-ui-caption tabular-nums text-muted">{countLabel}{props.truncated ? " · …" : ""}</p>
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
                <h3 className="text-ui-body font-medium">{copy.skills.featured}</h3>
                <span className="text-ui-caption tabular-nums text-muted">{copy.skills.count.replace("{n}", String(props.featured.length))}</span>
              </div>
              <div className="overflow-hidden rounded-xl border border-border bg-card shadow-[var(--shadow-card)]">
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
                <h3 className="text-ui-body font-medium">{category}</h3>
                <span className="text-ui-caption tabular-nums text-muted">{copy.skills.count.replace("{n}", String(items.length))}</span>
              </div>
              <div className="overflow-hidden rounded-xl border border-border bg-card shadow-[var(--shadow-card)]">
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
      <p className="mt-2 max-w-[68ch] text-ui-footnote leading-5 text-muted">{props.attribution || copy.skills.attribution}</p>
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
          <div className="text-ui-body font-medium text-foreground">{title}</div>
          {it.hasScripts ? <span className="text-ui-caption text-muted">{copy.skills.hasScripts}</span> : null}
          {props.incomplete ? <span className="text-ui-caption text-warning">{copy.skills.incomplete}</span> : null}
        </div>
        <p className="mt-0.5 line-clamp-2 text-ui-body leading-5 text-muted">{it.purpose || it.slug}</p>
        {meta.length ? (
          <p className="mt-1 text-ui-caption leading-4 text-muted">{meta.join(" · ")}</p>
        ) : null}
        {it.prerequisites && it.prerequisites !== "无" ? (
          <p className="mt-1 text-ui-caption leading-4 text-muted">{it.prerequisites}</p>
        ) : null}
      </div>
      {props.incomplete ? (
        <Button size="sm" variant="outline" className="mt-0.5 shrink-0 rounded-md" disabled={props.busy} onClick={() => props.onInstall(id, true)}>
          {props.busy ? copy.skills.repairing : copy.skills.repair}
        </Button>
      ) : props.installed ? (
        <span className="mt-0.5 shrink-0 text-ui-caption text-muted">{copy.skills.installedBadge}</span>
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
      className="relative mt-0.5 flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-lift text-ui-caption font-medium text-foreground/85"
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
        "rounded-md border px-2 py-1 text-ui-footnote transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
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
  if (source === "pack") return copy.skills.collection;
  return copy.skills.cas;
}

/** 已装页里一个合集的组头：名字、技能数、关闭时的提示，右侧开关、更新、卸载。开关看当前生效的 enabled。 */
function CollectionHead(props: {
  pack: PackStatus;
  busy: string;
  onToggle: (id: string, enabled: boolean) => void;
  onUpdate: (id: string) => void;
  onUninstall: (id: string) => void;
}) {
  const copy = useCopy();
  const p = props.pack;
  const count = p.skillCount || p.skills?.length || 0;
  const updating = props.busy === p.id + ":update";
  return (
    <div className="mb-2 flex flex-wrap items-center gap-x-2 gap-y-1 px-0.5">
      <h3 className="text-ui-body font-medium text-foreground">{p.name}</h3>
      <span className="text-ui-caption tabular-nums text-muted">
        {copy.skills.collection}
        {count ? ` · ${copy.skills.skillCount.replace("{n}", String(count))}` : ""}
        {p.enabled ? "" : ` · ${copy.skills.collectionOff}`}
      </span>
      <div className="ml-auto flex shrink-0 gap-1.5">
        <Button size="sm" variant={p.enabled ? "ghost" : "outline"} disabled={props.busy === p.id + ":toggle"} onClick={() => props.onToggle(p.id, !p.enabled)}>
          {p.enabled ? copy.skills.turnOff : copy.skills.turnOn}
        </Button>
        <Button size="sm" variant="ghost" disabled={updating} onClick={() => props.onUpdate(p.id)}>
          {updating ? copy.skills.updating : copy.skills.update}
        </Button>
        <Button size="sm" variant="ghost" disabled={props.busy === p.id} onClick={() => props.onUninstall(p.id)}>
          {copy.skills.uninstall}
        </Button>
      </div>
    </div>
  );
}

/** 市场顶部的技能合集，和源站条目同一种行样式。已装的只给徽章，管理在已装页。加载中或为空时不画。 */
function CollectionList(props: { packs: PackStatus[]; loading: boolean; busy: string; onInstall: (id: string, path?: string) => void }) {
  const copy = useCopy();
  if (props.loading || !props.packs.length) return null;
  return (
    <section className="mb-6" data-testid="skill-collections">
      <div className="mb-2 flex items-baseline justify-between px-0.5">
        <h3 className="text-ui-body font-medium">{copy.skills.collections}</h3>
        <span className="text-ui-caption tabular-nums text-muted">{copy.skills.count.replace("{n}", String(props.packs.length))}</span>
      </div>
      <div className="overflow-hidden rounded-xl border border-border bg-card shadow-[var(--shadow-card)]">
        {props.packs.map((p, i) => {
          const busy = props.busy === p.id;
          const count = p.skillCount || p.skills?.length || 0;
          const meta: string[] = [];
          if (count) meta.push(copy.skills.skillCount.replace("{n}", String(count)));
          if (p.origin?.repo) meta.push(p.origin.repo);
          if (p.license) meta.push(p.license);
          return (
            <div
              key={p.id}
              data-testid={`collection-row-${p.id}`}
              className={cn("flex items-start gap-3 px-3.5 py-3 transition-colors duration-[var(--duration-fast)] ease-[var(--ease-out)] hover:bg-lift/40", i > 0 && "border-t border-border/70")}
            >
              <SkillAvatar name={p.name} />
              <div className="min-w-0 flex-1">
                <div className="text-ui-body font-medium text-foreground">{p.name}</div>
                <p className="mt-0.5 line-clamp-2 text-ui-body leading-5 text-muted">{p.description}</p>
                {meta.length ? <p className="mt-1 text-ui-caption leading-4 text-muted">{meta.join(" · ")}</p> : null}
              </div>
              {p.installed ? (
                <span className="mt-0.5 shrink-0 text-ui-caption text-muted">{copy.skills.installedBadge}</span>
              ) : (
                <div className="mt-0.5 flex shrink-0 gap-1.5">
                  <Button
                    size="sm"
                    variant="ghost"
                    className="rounded-md"
                    disabled={busy}
                    onClick={() => {
                      void api.pickFolder().then((path) => {
                        if (path) props.onInstall(p.id, path);
                      });
                    }}
                  >
                    {copy.skills.fromLocal}
                  </Button>
                  <Button size="sm" variant="outline" className="rounded-md" disabled={busy} onClick={() => props.onInstall(p.id)}>
                    {busy ? copy.skills.installing : copy.skills.install}
                  </Button>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </section>
  );
}
