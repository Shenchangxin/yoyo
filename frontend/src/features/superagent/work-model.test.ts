import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { Decision, Ticket, Trigger } from "./desk-model.ts";
import { auditBrief, briefDue, briefPrompt, composeBrief, defaultBriefPrefs, dueItems, filterTickets, latestBrief, manuscriptNames, nowTickets, parseStoredBrief, serializeStoredBrief, slotDue, stageKey, taskGroups, unseenSuggestions, visibleIncoming, wouldAccept, type BriefPhrases } from "./work-model.ts";

function ticket(partial: Partial<Ticket> & Pick<Ticket, "id" | "stage">): Ticket {
  return {
    action: "",
    person: false,
    origin: "",
    title: partial.id,
    ...partial,
  };
}

describe("work pane grouping", () => {
  /** 等人的和正在做的进现在。已结束的只进任务。 */
  it("keeps waiting and running work in now", () => {
    const rows = [
      ticket({ id: "a", stage: "person", action: "answer", person: true }),
      ticket({ id: "b", stage: "working", action: "queued" }),
      ticket({ id: "c", stage: "ended" }),
      ticket({ id: "d", stage: "idle" }),
    ];
    assert.deepEqual(nowTickets(rows).map((row) => row.id), ["a", "b"]);
    const groups = taskGroups(rows);
    assert.deepEqual(groups.open.map((row) => row.id), ["a", "b", "d"]);
    assert.deepEqual(groups.ended.map((row) => row.id), ["c"]);
    assert.equal(stageKey(rows[0]), "answer");
    assert.equal(stageKey(rows[1]), "queued");
    assert.equal(stageKey(rows[2]), "ended");
    assert.equal(stageKey(rows[3]), "idle");
  });

  /** 搜索只留标题对上的。定时这一档不列工单。文件名从正文里收，不收路径。 */
  it("filters tasks and lifts manuscript names", () => {
    const rows = [
      ticket({ id: "a", stage: "working", title: "整理素材" }),
      ticket({ id: "b", stage: "ended", title: "旧方案" }),
    ];
    assert.deepEqual(filterTickets(rows, "open").map((row) => row.id), ["a"]);
    assert.deepEqual(filterTickets(rows, "all", "方案").map((row) => row.id), ["b"]);
    assert.deepEqual(filterTickets(rows, "scheduled"), []);
    assert.deepEqual(manuscriptNames("见 notes.csv 和 C:\\desk\\表.xlsx"), ["notes.csv", "表.xlsx"]);
  });

  /** 没点名的邮件不进现在。观察记下的本来会接单独列出来。 */
  it("hides quiet mail and keeps observed decisions", () => {
    const queued: Trigger[] = [
      { id: "d", source: "desk", sender: "operator", text: "fix", addressed: true, queued: true },
      { id: "m", source: "mail", sender: "ada", text: "fyi", addressed: false, queued: true },
      { id: "g", source: "github", sender: "ada", text: "bug", addressed: true, queued: true },
    ];
    assert.deepEqual(visibleIncoming(queued).map((row) => row.id), ["d", "g"]);
    const decisions: Decision[] = [
      { id: "1", triggerId: "m", kind: "ignore", reason: "observed" },
      { id: "2", triggerId: "x", kind: "ignore", reason: "chat", dismissed: true },
    ];
    assert.deepEqual(wouldAccept(decisions).map((row) => row.id), ["1"]);
  });

  /** 带时间的日历出现在到期里，普通邮件不会。 */
  it("lists dated calendar items", () => {
    const rows: Trigger[] = [
      { id: "c", source: "calendar", sender: "ada", text: "stand up", addressed: false, queued: true, due: "2026-11-03T01:00:00Z" },
      { id: "m", source: "mail", sender: "ada", text: "fyi", addressed: false, queued: true },
    ];
    assert.deepEqual(dueItems(rows).map((row) => row.id), ["c"]);
  });

  /** 最近一份标题像简报的工单留在今天。已经写过的定时规则不再推荐。 */
  it("keeps the latest brief and hides rules already written", () => {
    const rows = [
      ticket({ id: "old", stage: "ended", title: "晨间简报" }),
      ticket({ id: "other", stage: "ended", title: "修构建" }),
      ticket({ id: "new", stage: "working", title: "Morning brief" }),
    ];
    assert.equal(latestBrief(rows)?.id, "new");
    assert.deepEqual(unseenSuggestions(["daily-brief"]), [
      "waiting-reply",
      "meeting-prep",
      "weekly-status",
      "deadline-watch",
      "mentioned-mail",
    ]);
  });

  /** 对话里的「标题：说明」收成简报。json 代码块优先。 */
  it("builds a brief from labeled lines and from json", () => {
    const prose = [
      "经历和技术栈已经够用。",
      "简历：文件路径发我，或直接贴内容。",
      "目标方向：AI Agent 工程，还是全栈通吃。",
      "城市与工作方式：目标城市，是否接受远程。",
      "薪资：现在多少，或区间。",
      "其他硬条件：公司规模、行业、出差。",
      "方向越窄，能投的岗位越少。",
    ].join("\n");
    const brief = composeBrief(prose);
    assert.equal(brief?.items.length, 5);
    assert.equal(brief?.items[0].title, "简历");
    assert.match(brief?.insight || "", /方向越窄/);
    const json = composeBrief("```json\n{\"title\":\"先确认\",\"lede\":\"只补关键条件\",\"items\":[{\"title\":\"简历\",\"detail\":\"贴过来\"},{\"title\":\"方向\",\"detail\":\"全栈或专注\"}],\"insightTitle\":\"开筛前\",\"insight\":\"先定方向\"}\n```");
    assert.equal(json?.title, "先确认");
    assert.equal(json?.items.length, 2);
  });

  /** 简报只审计资产。过期先于今天，两周外的日历和已结束的工单不进来。点选不进对话。 */
  it("audits dated assets ahead of work that is waiting", () => {
    const now = new Date(2026, 10, 2, 8, 30);
    const brief = auditBrief({
      now,
      queued: [
        { id: "old", source: "calendar", sender: "", text: "过期的会", addressed: false, queued: true, due: "2026-11-01T09:00:00" },
        { id: "today", source: "calendar", sender: "", text: "今天下午的会", addressed: true, queued: true, due: "2026-11-02T15:00:00" },
        { id: "later", source: "calendar", sender: "", text: "下周的会", addressed: false, queued: true, due: "2026-12-20T09:00:00" },
      ],
      tickets: [
        ticket({ id: "wait", stage: "idle", person: true, title: "等你确认范围" }),
        ticket({ id: "done", stage: "ended", person: true, title: "已经结束" }),
      ],
      rules: [{ id: "daily-brief", text: "每天早上汇总", enabled: true }],
    }, phrases);
    const situation = brief.cards[0];
    assert.deepEqual(situation?.items.map((item) => item.title), ["过期的会", "今天下午的会", "等你确认范围"]);
    assert.equal(situation?.title, "GATE 3");
    assert.equal(situation?.layout, "checks");
    assert.match(situation?.task || "", /过期的会/);
    assert.match(situation?.alt || "", /过期的会/);
    assert.equal(brief.cards.some((card) => card.id === "hold"), false);
    assert.equal(JSON.stringify(brief).includes("简历"), false);
    assert.deepEqual(brief.picked, {});
    const kept = serializeStoredBrief("2026-11-02", brief);
    assert.equal(parseStoredBrief(kept, "2026-11-02")?.cards[0]?.title, situation?.title);
    assert.equal(parseStoredBrief(kept, "2026-11-03"), null);
    assert.equal(parseStoredBrief(JSON.stringify({ day: "2026-11-02", brief: { title: "old", items: [] } }), "2026-11-02"), null);
  });

  /** 两周内的日期要进来。本地什么都没有时，仍然给出一叠可点的卡。 */
  it("keeps the next two weeks and still briefs an empty desk", () => {
    const now = new Date(2026, 10, 2, 8, 30);
    const soon = auditBrief({
      now,
      queued: [{ id: "s", source: "calendar", sender: "", text: "两周内的会", addressed: false, queued: true, due: "2026-11-12T09:00:00" }],
    }, phrases);
    assert.equal(soon.cards[0]?.items[0]?.title, "两周内的会");
    const empty = auditBrief({ now }, phrases);
    assert.equal(empty.cards.length, 1);
    assert.equal(empty.cards[0]?.id, "quiet");
    assert.equal(empty.cards[0]?.task, phrases.quietTask);
  });

  /** 没有时效问题时，档案单独成卡，不编进当天到期。 */
  it("keeps a quiet brief when nothing is due", () => {
    const brief = auditBrief({
      now: new Date(2026, 10, 2, 8, 30),
      memory: [{ kind: "profile", text: "想做 Agent 工程" }],
    }, phrases);
    assert.equal(brief.cards[0]?.id, "kept");
    assert.equal(brief.cards.find((card) => card.id === "kept")?.items[0]?.title, phrases.profile);
    assert.equal(brief.cards.find((card) => card.id === "around"), undefined);
    assert.match(brief.cards[0]?.task || "", /想做 Agent 工程/);
  });

  /** 没有记忆时，应用和桌面留下的名字仍然单独成卡。 */
  it("briefs installed apps when memory is empty", () => {
    const brief = auditBrief({
      now: new Date(2026, 10, 2, 8, 30),
      points: [
        { name: "Cursor", kind: "app" },
        { name: "yoyo", kind: "trace" },
      ],
    }, phrases);
    const around = brief.cards.find((card) => card.id === "around");
    assert.deepEqual(around?.items.map((item) => item.title), ["Cursor", "yoyo"]);
    assert.equal(around?.layout, "chips");
    assert.match(around?.task || "", /Cursor/);
    assert.equal(brief.cards.some((card) => card.id === "kept"), false);
    const bare = { id: "old", kicker: "", title: "旧卡", lede: "", items: [{ title: "一件事", detail: "还开着" }], choices: [], primary: "去做", secondary: "先不" };
    assert.match(briefPrompt(bare, "primary"), /一件事/);
  });

  /** 到了设定时间才静默生成。早于时间、或关掉早晨，都不算到点。 */
  it("treats the morning brief as due only after its time", () => {
    const prefs = defaultBriefPrefs();
    assert.equal(briefDue(prefs, new Date(2026, 10, 2, 8, 30)), true);
    assert.equal(briefDue(prefs, new Date(2026, 10, 2, 8, 29)), false);
    assert.equal(briefDue({ ...prefs, morningOn: false }, new Date(2026, 10, 2, 9, 0)), false);
    assert.equal(slotDue(prefs, "evening", new Date(2026, 10, 2, 21, 30)), true);
    assert.equal(slotDue(prefs, "evening", new Date(2026, 10, 2, 21, 29)), false);
  });
});

const phrases: BriefPhrases = {
  dueTitle: "DUE {n}",
  personTitle: "WAIT {n}",
  stuckTitle: "STUCK {n}",
  quietTitle: "QUIET",
  lede: "dated {due} waiting {wait} stuck {stuck}",
  quietLede: "quiet",
  insightTitle: "time",
  insight: "order",
  rules: "rules {n}",
  overdue: "overdue",
  dueToday: "today",
  dueSoon: "soon",
  waiting: "waiting",
  blocked: "blocked",
  failed: "failed",
  observed: "observed",
  source: "source",
  profile: "profile",
  project: "project",
  episodic: "episodic",
  semantic: "semantic",
  poiTitle: "poi",
  poiLede: "names",
  aroundTitle: "around",
  aroundLede: "from the desk",
  aroundEmpty: "no names",
  aroundEmptyDetail: "nothing named",
  appSeen: "app",
  traceSeen: "trace",
  statApps: "apps",
  statTraces: "traces",
  statOpen: "open",
  statDated: "dated",
  open: "open",
  nothing: "none",
  nothingDetail: "empty",
  keep: "keep",
  hold: "hold",
  directionTitle: "direction",
  directionLede: "pick one",
  useChoice: "use",
  keepAll: "all",
  keptTitle: "kept",
  keptLede: "on record",
  useMemory: "use memory",
  skipMemory: "skip",
  afterTitle: "after",
  afterLede: "one thing",
  lookFirst: "look",
  leaveAll: "leave",
  watchTitle: "watch",
  watchLede: "stays here",
  watchDue: "due",
  watchWait: "wait",
  watchOpen: "open",
  watchMemory: "memory",
  saveWatch: "save",
  leaveWatch: "leave watch",
  situationKicker: "morning",
  emptyKicker: "empty",
  gateKicker: "gate",
  gateTitle: "GATE {n}",
  gateLede: "gate",
  gatePrimary: "handle {n}",
  gateSecondary: "one",
  gateTask: "TASK\n{items}",
  gateOne: "ONE {title} {detail}",
  splitKicker: "split",
  splitTitle: "split title",
  splitLede: "split lede",
  splitPrimary: "do {title}",
  splitTask: "DO {title} {detail}",
  memoryKicker: "memory",
  memoryTitle: "memory title",
  memoryLede: "memory lede",
  memoryPrimary: "use memory",
  memorySecondary: "check one",
  memoryTask: "MEM\n{items}",
  memoryOne: "MEM1 {title} {detail}",
  aroundKicker: "around kicker",
  aroundPrimary: "make one",
  aroundSecondary: "one name",
  aroundTask: "AROUND\n{items}",
  aroundOne: "NAME {title} {detail}",
  nextTitle: "still open",
  nextPrimary: "continue {title}",
  nextTask: "NEXT {title} {detail}",
  quietPrimary: "choose one",
  quietTask: "QUIET_TASK",
  countLabel: "{n} items",
};
