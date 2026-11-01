/**
 * 拥有：编排台的路线、过滤和搜索。页面按这些结果画。
 * 不拥有：拉取快照和提交动作。那一段在 OrchestratorDesk。
 */

export type Source = { name: string; state: string; detail: string; repo?: string; chat?: string; app?: string; api?: string; hasSecret?: boolean };
export type Input = { id: string; text: string; human: boolean; delivered: boolean };
export type Follow = { thread: string; own: boolean; closed?: string };
export type Assignment = { id: string; state: string; report?: string; summary?: string };
export type Ticket = {
  id: string; stage: string; action: string; person: boolean; origin: string; outcome?: string;
  strength?: string; sender?: string; attention?: string; deliveryError?: string; originUrl?: string;
  title: string;
  inputs?: Input[]; follows?: Follow[]; assignments?: Assignment[];
};
export type Trigger = {
  id: string; source: string; sender: string; text: string; addressed: boolean;
  thread?: string; at?: string; filtered?: string; queued?: boolean;
};
export type Thread = {
  key: string; title: string; queued: number; subs: number; followers?: string[];
  evaluating: boolean; recent: boolean; at?: string; sender?: string; items?: Trigger[];
};
export type Decision = { id: string; triggerId: string; kind: string; reason: string; thread?: string; ticketId?: string; dismissed?: boolean; at?: string };
export type Eval = { id: string; thread: string; strength: string; deadline?: string; delivered?: boolean; triggers: number };
export type Problem = { kind: string; id: string; title: string; detail?: string; triggerId?: string; decisionId?: string; thread?: string; ticketId?: string };
export type Admission = { listedUnaddressed: string; unlistedAddressed: string; unlistedUnaddressed: string };
export type Flow = { received: number; filtered: number; judged: number; accepted: number; ignored: number };
export type Load = { running: number; queued: number; idle: number; maxWorkers: number };
export type DeskView = {
  mode: string;
  paused: boolean;
  admission: Admission;
  sources: Source[];
  tickets: Ticket[];
  threads: Thread[];
  queued: Trigger[];
  evals: Eval[];
  decisions: Decision[];
  problems?: Problem[];
  allowed?: string[];
  workers?: number;
  health: string;
  at: string;
  evalTimeout?: number;
  flow?: Flow;
  load?: Load;
};

export const tabs = ["overview", "source", "trigger", "eval", "execution", "settings"] as const;
export type Tab = (typeof tabs)[number];
export const threadFilters = ["subscribed", "evaluating", "recent", "all"] as const;
export const ticketFilters = ["open", "person", "working", "idle", "ended"] as const;
export const decisionFilters = ["work", "stop", "ignore", "unavailable", "all"] as const;
export const settingGroups = ["source", "eval", "execution"] as const;
export type ThreadFilter = (typeof threadFilters)[number];
export type TicketFilter = (typeof ticketFilters)[number];
export type DecisionFilter = (typeof decisionFilters)[number];
export type SettingsGroup = (typeof settingGroups)[number];

export type Route =
  | { page: "overview" }
  | { page: "source" }
  | { page: "trigger"; id?: string; filter: ThreadFilter; trigger?: string }
  | { page: "eval"; filter: DecisionFilter }
  | { page: "execution"; id?: string; filter: TicketFilter }
  | { page: "settings"; group: SettingsGroup };

export type Location = { route: Route; stack: Route[] };

const pick = <V extends string>(values: readonly V[], value: unknown, fallback: V): V =>
  values.find((item) => item === value) ?? fallback;

/** parseRoute 把存下来的 value 收成一条路线。认不出的页退回总览。 */
export function parseRoute(value: unknown): Route {
  const raw = (value ?? {}) as { page?: unknown; id?: unknown; filter?: unknown; trigger?: unknown; group?: unknown };
  const id = typeof raw.id === "string" && raw.id ? raw.id : undefined;
  const trigger = typeof raw.trigger === "string" && raw.trigger ? raw.trigger : undefined;
  switch (raw.page) {
    case "source":
      return { page: "source" };
    case "trigger":
      return id
        ? { page: "trigger", id, filter: pick(threadFilters, raw.filter, "subscribed"), ...(trigger ? { trigger } : {}) }
        : { page: "trigger", filter: pick(threadFilters, raw.filter, "subscribed") };
    case "eval":
      return { page: "eval", filter: pick(decisionFilters, raw.filter, "work") };
    case "execution":
      return id
        ? { page: "execution", id, filter: pick(ticketFilters, raw.filter, "open") }
        : { page: "execution", filter: pick(ticketFilters, raw.filter, "open") };
    case "settings":
      return { page: "settings", group: pick(settingGroups, raw.group, "source") };
    default:
      return { page: "overview" };
  }
}

/** tabRoute 是点顶部某一页时的起点。page 决定默认过滤。 */
export function tabRoute(page: Tab): Route {
  switch (page) {
    case "trigger":
      return { page, filter: "subscribed" };
    case "eval":
      return { page, filter: "work" };
    case "execution":
      return { page, filter: "open" };
    case "settings":
      return { page, group: "source" };
    case "source":
      return { page: "source" };
    default:
      return { page: "overview" };
  }
}

/** pushRoute 从 location 走向 route。replace 为真时换掉当前条，不增加返回。 */
export function pushRoute(location: Location, route: Route, replace = false): Location {
  if (replace) return { route, stack: location.stack };
  return { route, stack: [...location.stack, location.route].slice(-20) };
}

/** popRoute 从 location 退回上一条。栈空时回到总览。 */
export function popRoute(location: Location): Location {
  const route = location.stack.length ? location.stack[location.stack.length - 1]! : { page: "overview" as const };
  return { route, stack: location.stack.slice(0, -1) };
}

/** ticketsFor 按 filter 留下工单。open 是还没结束的。 */
export function ticketsFor(tickets: Ticket[], filter: TicketFilter): Ticket[] {
  return tickets.filter((ticket) => {
    if (filter === "ended") return ticket.stage === "ended";
    if (filter === "open") return ticket.stage !== "ended";
    if (filter === "person") return ticket.person;
    if (filter === "working") return ticket.stage === "working";
    return ticket.stage === "idle";
  });
}

/** threadsFor 按 filter 留下线程。subscribed 只要有工单订阅的。 */
export function threadsFor(threads: Thread[], filter: ThreadFilter): Thread[] {
  return threads.filter((thread) => {
    if (filter === "all") return true;
    if (filter === "subscribed") return thread.subs > 0;
    if (filter === "evaluating") return thread.evaluating;
    return thread.recent;
  });
}

/** decisionsFor 按 filter 留下决定。work 只要接单。已驳回的只在 all 里出现。 */
export function decisionsFor(decisions: Decision[], filter: DecisionFilter): Decision[] {
  return decisions.filter((row) => {
    if (row.dismissed && filter !== "all") return false;
    if (filter === "all") return true;
    if (filter === "work") return row.kind === "accept";
    return row.kind === filter;
  });
}

export type Hit = { kind: "ticket" | "thread" | "trigger"; id: string; title: string; detail: string; route: Route };

/** searchDesk 在 view 里找 query。空查询没有结果。工单、线程、触发各按标题匹配，最多十二条。 */
export function searchDesk(view: DeskView, query: string): Hit[] {
  const needle = query.trim().toLowerCase();
  if (!needle) return [];
  const hits: Hit[] = [];
  const has = (...parts: Array<string | undefined>) => parts.join(" ").toLowerCase().includes(needle);
  for (const ticket of view.tickets ?? []) {
    if (!has(ticket.title, ticket.id, ticket.sender, ticket.attention, ticket.origin)) continue;
    hits.push({
      kind: "ticket", id: ticket.id, title: ticket.title || ticket.id, detail: ticket.sender || ticket.stage,
      route: { page: "execution", id: ticket.id, filter: "open" },
    });
  }
  for (const thread of view.threads ?? []) {
    if (!has(thread.title, thread.key, thread.sender)) continue;
    hits.push({
      kind: "thread", id: thread.key, title: thread.title || thread.key, detail: thread.sender || "",
      route: { page: "trigger", id: thread.key, filter: "subscribed" },
    });
  }
  for (const item of view.queued ?? []) {
    if (!has(item.text, item.sender, item.id)) continue;
    hits.push({
      kind: "trigger", id: item.id, title: item.text || item.id, detail: item.sender || item.source,
      route: { page: "trigger", id: item.thread || "", filter: "subscribed", trigger: item.id },
    });
  }
  return hits.filter((hit) => hit.kind !== "trigger" || hit.route.page === "trigger" && hit.route.id).slice(0, 12);
}

/** triggerText 在 view 里找 id 对应的正文。找不到时返回空串。 */
export function triggerText(view: DeskView, id: string): string {
  for (const thread of view.threads ?? []) {
    for (const item of thread.items ?? []) {
      if (item.id === id && item.text) return item.text;
    }
  }
  for (const item of view.queued ?? []) {
    if (item.id === id) return item.text;
  }
  return "";
}
