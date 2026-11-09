/**
 * 拥有：端到端测试里 Buddy 的假后端：buddy.* 的有状态应答。安装和探测随列表轮询往前走，配置只记 hasKey，从不回显密钥。
 * 不拥有：真实的检测、安装和探测（在 internal/buddy）。mock.ts 把 buddy.* 转到这里，用例用 buddyFixture 造数据。
 */

/** INSTALL_STEPS 是假安装依次经过的进度。 */
const INSTALL_STEPS = [{ phase: "queued" }, { phase: "node", percent: 40 }, { phase: "deps" }, { phase: "verify", percent: 95 }];

/** STEP_MS 是两步之间至少隔的时间。挨得很近的两次列表只走一步，每一步都能被界面看到。 */
const STEP_MS = 700;

/** YOYO_BUDDY 是内置的 yoyo。 */
export const YOYO_BUDDY = {
  id: "yoyo",
  name: "yoyo",
  builtin: true,
  order: 0,
  install: { kind: "none" },
  state: "online",
  config: { enabled: true, mode: "managed", approval: "yoyo", maxParallel: 4, secrets: [] },
};

/** buddyFixture 造一个外部 agent 的视图。默认是能托管安装、还没装；over 覆盖任意字段，config 按字段合并。 */
export function buddyFixture(id: string, over: Record<string, any> = {}): any {
  const config = { enabled: true, mode: "managed", approval: "yoyo", maxParallel: 1, secrets: [] };
  return {
    id,
    name: id,
    featured: true,
    order: 10,
    install: { kind: "none" },
    canManage: true,
    state: "not_installed",
    systemInstall: { available: true, command: `npm install -g @example/${id}` },
    ...over,
    config: { ...config, ...(over.config || {}) },
  };
}

/** createBuddyMock 返回按方法名应答的假后端。seed 为空时只有内置 yoyo。 */
export function createBuddyMock(seed?: any[]) {
  let list: any[] = (seed ?? [YOYO_BUDDY]).map((v) => structuredClone(v));
  const step: Record<string, number> = {};
  const signedIn = new Set<string>();
  let prefs = { npmRegistry: "auto", pyIndex: "", defaultAgent: "yoyo", autoPick: false, turnMinutes: 30 };
  let cache = 23_456_789;
  let last = 0;
  const find = (id: string) => list.find((v) => v.id === id);
  const now = () => new Date().toISOString();

  /** settled 是探完以后的状态：缺必填钥匙要配置，有登录方式还没登录要登录，否则在线。 */
  const settled = (v: any) => {
    if (v.config.secrets.some((s: any) => !s.optional && !s.hasKey && !s.reuse)) return "needs_config";
    if (v.auth?.methods?.length && !signedIn.has(v.id)) return "needs_auth";
    return "online";
  };

  /** advance 让在装和在探的项往前走一步。 */
  const advance = () => {
    if (Date.now() - last < STEP_MS) return;
    last = Date.now();
    for (const v of list) {
      if (v.state === "installing") {
        const i = (step[v.id] ?? 0) + 1;
        if (i < INSTALL_STEPS.length) {
          step[v.id] = i;
          v.progress = INSTALL_STEPS[i];
          continue;
        }
        delete step[v.id];
        delete v.progress;
        v.install = { kind: "managed", source: "npm", path: `C:/Users/me/AppData/Local/yoyo/agents/${v.id}`, sizeBytes: 52_000_000 };
        v.version = v.latest || "1.0.0";
        v.state = settled(v);
        v.checkedAt = now();
      } else if (v.state === "probing") {
        v.state = settled(v);
        v.checkedAt = now();
      }
    }
  };

  /** handle 应答一个 buddy.* 方法。 */
  const handle = (method: string, params: any): any => {
    const v = params?.id ? find(String(params.id)) : undefined;
    switch (method) {
      case "buddy.list":
      case "buddy.refresh":
        advance();
        return { buddies: list, prefs, detecting: false };
      case "buddy.get":
        return v ?? {};
      case "buddy.install":
        if (!v) return {};
        v.state = "installing";
        v.progress = INSTALL_STEPS[0];
        step[v.id] = 0;
        last = Date.now();
        delete v.detail;
        return v;
      case "buddy.cancel":
        if (v) {
          v.state = "not_installed";
          v.detail = "Install cancelled";
          delete v.progress;
          delete step[v.id];
        }
        return { ok: true };
      case "buddy.uninstall":
        if (!v) return {};
        if (v.custom) {
          list = list.filter((x) => x.id !== v.id);
          return { id: v.id, state: "not_installed" };
        }
        v.install = { kind: "none" };
        v.state = "not_installed";
        delete v.version;
        return v;
      case "buddy.configure": {
        if (!v) return {};
        const { id: _id, secrets, reuse, ...rest } = params;
        for (const s of v.config.secrets) {
          if (secrets && s.name in secrets) s.hasKey = secrets[s.name] !== "";
          if (Array.isArray(reuse)) s.reuse = reuse.includes(s.name);
        }
        Object.assign(v.config, rest);
        if (v.install.kind !== "none" && ["needs_config", "needs_auth", "error", "offline"].includes(v.state)) v.state = "probing";
        return v;
      }
      case "buddy.login": {
        if (!v) return {};
        const how = v.auth?.methods?.find((m: any) => m.id === params.method) ?? v.auth?.methods?.[0];
        signedIn.add(v.id);
        v.state = "probing";
        return { started: true, terminal: how?.type === "terminal", buddy: v };
      }
      case "buddy.logout":
        if (!v) return {};
        signedIn.delete(v.id);
        v.state = "needs_auth";
        return v;
      case "buddy.probe":
        if (!v) return {};
        v.state = settled(v);
        v.checkedAt = now();
        return v;
      case "buddy.prefs":
        prefs = { ...prefs, ...params };
        return prefs;
      case "buddy.add": {
        const added = buddyFixture(`custom-${String(params.name).toLowerCase().replace(/[^a-z0-9]+/g, "-")}`, {
          name: params.name,
          custom: true,
          featured: false,
          canManage: false,
          systemInstall: undefined,
          install: { kind: "custom", path: params.command },
          state: "probing",
          config: { args: params.args, env: params.env },
        });
        list = [...list, added];
        return added;
      }
      case "buddy.log.tail":
        return { text: v ? `[install] ${v.name}: queued\n[install] done\n` : "" };
      case "buddy.disk.get":
        return { total: 150_000_000 + cache, runtime: 150_000_000, cache, items: {} };
      case "buddy.cache.clear": {
        const freed = cache;
        cache = 0;
        return { freed };
      }
      default:
        return {};
    }
  };

  return { handle };
}
