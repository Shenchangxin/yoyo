import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { decisionsFor, parseRoute, pushRoute, searchDesk, tabRoute, threadsFor, ticketsFor, type DeskView, type Location } from "./desk-model.ts";

/**
 * 拥有：编排台路线、过滤和搜索在这些用例上的约定。
 * 不拥有：页面怎么画。
 */

const view = {
  mode: "observe",
  paused: false,
  admission: { listedUnaddressed: "inferred", unlistedAddressed: "inferred", unlistedUnaddressed: "ignore" },
  sources: [],
  tickets: [
    { id: "k1", title: "修复登录", stage: "person", action: "answer", person: true, origin: "t1", sender: "ada" },
    { id: "k2", title: "跑测试", stage: "working", action: "queued", person: false, origin: "t2" },
    { id: "k3", title: "已合并", stage: "ended", action: "", person: false, origin: "t3", outcome: "merged" },
  ],
  threads: [
    { key: "room", title: "修复登录", queued: 1, subs: 1, evaluating: false, recent: true, sender: "ada" },
    { key: "noise", title: "闲聊", queued: 0, subs: 0, evaluating: true, recent: false },
  ],
  queued: [{ id: "q1", source: "github", sender: "ada", text: "再看一眼", addressed: true, thread: "room" }],
  evals: [],
  decisions: [
    { id: "d1", triggerId: "q1", kind: "accept", reason: "点名了", thread: "room" },
    { id: "d2", triggerId: "q2", kind: "ignore", reason: "观察", dismissed: true },
  ],
  health: "ready",
  at: "",
} satisfies DeskView;

describe("desk routes", () => {
  /** 认不出的页回到总览，触发页补上默认过滤。 */
  it("parses a stored route", () => {
    assert.deepEqual(parseRoute({ page: "nope" }), { page: "overview" });
    assert.deepEqual(parseRoute({ page: "trigger", id: "room", trigger: "q1" }), {
      page: "trigger", id: "room", filter: "subscribed", trigger: "q1",
    });
    assert.deepEqual(tabRoute("execution"), { page: "execution", filter: "open" });
  });

  /** 打开工单会留下返回，换过滤不会。 */
  it("pushes a ticket and replaces a filter", () => {
    const here: Location = { route: tabRoute("execution"), stack: [] };
    const opened = pushRoute(here, { page: "execution", id: "k1", filter: "open" });
    assert.equal(opened.stack.length, 1);
    const filtered = pushRoute(opened, { page: "execution", filter: "person" }, true);
    assert.equal(filtered.stack.length, 1);
    assert.equal(filtered.route.page === "execution" && filtered.route.filter, "person");
  });
});

describe("desk lists", () => {
  /** 进行中不含已结束，待处理只要等人的。 */
  it("filters tickets by who they wait on", () => {
    assert.deepEqual(ticketsFor(view.tickets, "open").map((row) => row.id), ["k1", "k2"]);
    assert.deepEqual(ticketsFor(view.tickets, "person").map((row) => row.id), ["k1"]);
    assert.deepEqual(ticketsFor(view.tickets, "ended").map((row) => row.id), ["k3"]);
  });

  /** 默认线程是有工单订阅的，评估记录默认不显示已驳回。 */
  it("filters threads and decisions", () => {
    assert.deepEqual(threadsFor(view.threads, "subscribed").map((row) => row.key), ["room"]);
    assert.deepEqual(threadsFor(view.threads, "evaluating").map((row) => row.key), ["noise"]);
    assert.deepEqual(decisionsFor(view.decisions, "work").map((row) => row.id), ["d1"]);
    assert.deepEqual(decisionsFor(view.decisions, "all").map((row) => row.id), ["d1", "d2"]);
  });

  /** 搜索按标题跳到工单或线程，没有线程的触发不给一条空路线。 */
  it("searches by the words a person reads", () => {
    const hits = searchDesk(view, "登录");
    assert.equal(hits[0]?.route.page, "execution");
    assert.equal(hits.some((hit) => hit.kind === "thread"), true);
    const queued = searchDesk(view, "再看一眼");
    assert.equal(queued[0]?.route.page, "trigger");
    assert.equal(queued[0]?.route.page === "trigger" && queued[0].route.id, "room");
  });
});
