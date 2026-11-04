import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { Decision, Ticket, Trigger } from "./desk-model.ts";
import { applySenses, auditBrief, briefDue, briefPrompt, composeBrief, defaultBriefPrefs, dueItems, filterTickets, latestBrief, manuscriptNames, memoryFolderRows, memoryPortrait, nowTickets, parseStoredBrief, serializeStoredBrief, runningCount, shouldPush, slotDue, visibleSlot, stageKey, taskGroups, todayBadge, unseenSuggestions, visibleIncoming, wouldAccept, type BriefPhrases, type SenseLabels } from "./work-model.ts";

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
    assert.equal(runningCount(rows), 1);
    assert.equal(todayBadge(rows, [], []), 0);
    assert.equal(todayBadge(rows, [{ id: "in", source: "desk", sender: "ada", text: "fix", addressed: true, queued: true }], []), 1);
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
    const older = JSON.parse(kept) as { v: number; brief: unknown };
    older.v = 1;
    assert.equal(parseStoredBrief(JSON.stringify(older), "2026-11-02")?.cards[0]?.title, situation?.title);
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
    assert.equal(visibleSlot(prefs, new Date(2026, 10, 2, 8, 0)), "morning");
    assert.equal(visibleSlot(prefs, new Date(2026, 10, 2, 22, 0)), "evening");
  });

  /** 第一封信只用本机名字和修改时间。有日程时不再说今天无法确认，周末也不替用户选继续还是停下。 */
  it("writes a welcome letter from machine names and leaves the weekend choice open", () => {
    const brief = auditBrief({
      now: new Date(2026, 10, 7, 9, 0),
      welcome: true,
      memory: [{ kind: "profile", text: "caca" }],
      census: {
        user: "scx",
        hours: "08:30–17:40",
        weekday: true,
        lines: [
          { name: "yoyo", state: "moving", evidence: "main.go", when: "2026-11-07" },
          { name: "allo", state: "stalled", evidence: "blank.txt", when: "2026-11-05" },
        ],
        finished: [{ name: "notes.html", kind: "deliverable" }],
        empty: [{ name: "blank.txt", line: "allo", kind: "empty" }],
        place: [{ name: "yoyo-notes.md", line: "yoyo", kind: "match" }],
        archive: [{ name: "setup.exe", kind: "installer" }],
        projects: 2,
        downloads: 1,
        apps: 3,
        traces: 4,
      },
    }, phrases);
    assert.equal(brief.welcome, true);
    assert.equal(brief.cards[0]?.id, "letter");
    assert.equal(brief.cards[0]?.title, "hello caca");
    assert.equal(brief.cards[0]?.primary, "");
    assert.equal(brief.cards.find((card) => card.id === "situation"), undefined);
    const lines = brief.cards.find((card) => card.id === "lines");
    assert.equal(lines?.layout, "rows");
    assert.equal(lines?.primary, "explain yoyo");
    assert.match(lines?.task || "", /Do not invent metrics/);
    assert.equal(lines?.items[1]?.tone, "stalled");
    assert.equal(brief.cards.find((card) => card.id === "done")?.mark, "1 finished, 1 unfinished");
    assert.match(brief.cards.find((card) => card.id === "done")?.task || "", /Do not invent file contents/);
    assert.match(brief.cards.find((card) => card.id === "place")?.task || "", /Do not delete/);
    assert.equal(brief.cards.find((card) => card.id === "tools")?.stats?.map((stat) => stat.value).join(","), "2,3,1,4");
    const pause = brief.cards.find((card) => card.id === "pause");
    assert.equal(pause?.primary, "");
    assert.equal(pause?.secondary, "");
    assert.equal(pause?.note, "no button");
    assert.equal(brief.cards.some((card) => card.id === "quiet" || card.id === "around"), false);
    assert.equal(JSON.stringify(brief).includes("0.667"), false);
    const dated = auditBrief({
      now: new Date(2026, 10, 2, 8, 30),
      queued: [{ id: "today", source: "calendar", sender: "", text: "今天下午的会", addressed: false, queued: true, due: "2026-11-02T15:00:00" }],
      census: { user: "scx", lines: [{ name: "yoyo", state: "moving", evidence: "main.go", when: "2026-11-02" }] },
    }, phrases);
    assert.equal(dated.cards[0]?.id, "letter");
    assert.equal(dated.cards[0]?.choices.length, 0);
    assert.equal(dated.cards[1]?.id, "situation");
    assert.equal(dated.welcome, false);
  });

  /** 画像只用本机事实和当事人说过的话。机器账号不算「你说过」。 */
  it("splits machine facts from what the operator said", () => {
    const rows = [
      { id: "machine:device", kind: "profile", source: "machine", project: "device", text: "scx", created_at: "2026-10-01T00:00:00Z" },
      { id: "machine:host", kind: "profile", source: "machine", project: "device", text: "desk", created_at: "2026-10-01T00:00:00Z" },
      { id: "machine:git", kind: "profile", source: "machine", project: "device", text: "Ada", created_at: "2026-10-01T00:00:00Z" },
      { id: "machine:hours", kind: "profile", source: "machine", project: "work", text: "08:30–17:40", created_at: "2026-10-10T00:00:00Z" },
      { id: "machine:project:yoyo", kind: "project", source: "machine", project: "yoyo", text: "yoyo", state: "moving", evidence: "main.go", when: "2026-10-10", created_at: "2026-10-10T00:00:00Z" },
      { id: "p1", kind: "profile", source: "user", text: "caca", created_at: "2026-10-02T00:00:00Z" },
      { id: "p2", kind: "profile", source: "model", text: "喜欢早上处理代码", created_at: "2026-10-03T00:00:00Z" },
      { id: "s1", kind: "semantic", source: "model", text: "回复要短", created_at: "2026-10-04T00:00:00Z" },
    ];
    const portrait = memoryPortrait(rows);
    assert.equal(portrait.name, "caca");
    assert.equal(portrait.device, "scx");
    assert.equal(portrait.host, "desk");
    assert.equal(portrait.git, "Ada");
    assert.equal(portrait.said.includes("Ada"), false);
    assert.equal(portrait.hours, "08:30–17:40");
    assert.equal(portrait.lines[0]?.evidence, "main.go");
    assert.deepEqual(portrait.said, ["caca"]);
    assert.deepEqual(portrait.noted, ["喜欢早上处理代码"]);
    assert.deepEqual(portrait.preferences, ["回复要短"]);
    assert.equal(portrait.updated, "2026-10-10T00:00:00Z");
    assert.equal(memoryFolderRows(rows, "about").length, 2);
    assert.equal(memoryFolderRows(rows, "device").length, 3);
    assert.equal(JSON.stringify(portrait).includes("0.667"), false);
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
  letterKicker: "first letter",
  letterKickerToday: "today",
  letterHello: "hello {name}",
  letterHelloBare: "hello",
  letterLede: "starts from this machine",
  letterLedeToday: "today from this machine",
  letterKnown: "known",
  letterMissing: "missing",
  letterLines: "{n} lines",
  letterIdentity: "identity",
  letterHost: "computer",
  letterOs: "system",
  letterGit: "git",
  letterApps: "apps",
  letterHours: "hours",
  letterNoMail: "mail",
  letterNoMailDetail: "no signal",
  letterNoPlan: "plan",
  letterNoPlanDetail: "unconfirmed",
  linesKicker: "lines",
  linesTitle: "lines title",
  linesLede: "names and times only",
  lineMoving: "moving",
  lineStalled: "stalled",
  lineQuiet: "quiet",
  linesPrimary: "explain {name}",
  linesTask: "Do not invent metrics\n{name}\n{items}",
  doneKicker: "done",
  doneTitle: "small finishes",
  doneLede: "size or empty",
  doneMark: "{done} finished, {left} unfinished",
  doneFinished: "has size",
  doneEmpty: "empty",
  donePrimary: "why empty",
  doneAlt: "group finished",
  doneTask: "Do not invent file contents\n{items}",
  placeKicker: "place",
  placeTitle: "group leftovers",
  placeLede: "names only",
  placeLeft: "on a line",
  placeRight: "archive",
  placeNone: "none",
  placePrimary: "group files",
  placeTask: "Do not delete\n{place}\n{archive}",
  toolKicker: "tools",
  toolTitle: "in use",
  toolLede: "counts only",
  toolProjects: "projects",
  toolApps: "apps",
  toolDownloads: "downloads",
  toolTraces: "traces",
  toolPrimary: "split used",
  toolTask: "names and recency only\n{items}",
  pauseKicker: "yours",
  pauseTitle: "continue or stop",
  pauseLede: "no calendar",
  pauseGo: "continue",
  pauseGoDetail: "lines remain",
  pauseStop: "stop",
  pauseStopDetail: "nothing due",
  pauseNote: "no button",
};

const senseLabels: SenseLabels = {
  unchecked: "unchecked",
  linked: "linked",
  senseTitle: "seen",
  senseLede: "names only",
  listening: "listening",
  battery: "battery",
  charging: "charging",
  online: "online",
  offline: "offline",
  idle: "away",
  focus: "front",
  power: "power",
  net: "net",
  git: "git",
  mail: "mail",
  calendar: "calendar",
  poi: "names",
  speakKicker: "worth",
  speakTitle: "why",
  speakLede: "quiet would leave it",
  speakPrimary: "look",
  speakTask: "no metrics\n{items}",
  why: { changed: "changed", overdue: "overdue", error: "error" },
};

describe("initiative", () => {
  it("stays quiet unless the gate or the first letter says to speak", () => {
    assert.equal(shouldPush(undefined, false), true);
    assert.equal(shouldPush({ speak: false, signals: [] }, false), false);
    assert.equal(shouldPush({ speak: false, signals: [] }, true), true);
    assert.equal(shouldPush({ speak: true, signals: [] }, false), true);
  });

  it("writes unchecked mail into the letter and a speak card from evidence", () => {
    const brief = auditBrief({ welcome: true, census: { user: "scx", lines: [{ name: "yoyo", state: "moving" }] } }, phrases);
    const next = applySenses(brief, [
      { id: "mail", code: "mail", class: "mail", name: "mail", detail: "", state: "unchecked", why: "", speak: false, held: false },
      { id: "line:yoyo", code: "line", class: "line", name: "yoyo", detail: "blank.txt", state: "stalled", why: "changed", speak: true, held: false },
      { id: "power", code: "power", class: "power", name: "17", detail: "battery", state: "ok", why: "", speak: false, held: false },
    ], senseLabels);
    const letter = next.cards.find((card) => card.id === "letter");
    assert.equal(letter?.choices.some((choice) => choice.detail.includes("unchecked")), true);
    const speak = next.cards.find((card) => card.id === "speak");
    assert.equal(speak?.items[0]?.title, "yoyo");
    assert.equal(speak?.items[0]?.detail.includes("blank.txt"), true);
    assert.equal(next.cards.some((card) => card.id === "sense"), false);
    assert.equal(JSON.stringify(next).includes("sk-"), false);
  });

  /** 画像写进信，事实不再让人标对错。电量、前台和桌面文件名不进简报。断网才单独一行。 */
  it("writes a portrait and leaves routine telemetry off the brief", () => {
    const brief = auditBrief({
      welcome: true,
      census: {
        user: "scx",
        host: "DESK",
        os: "Windows",
        git: "Ada",
        hours: "19:30–21:58",
        appNames: ["Cursor", "Chrome"],
        lines: [{ id: "machine:project:yoyo", name: "yoyo", state: "quiet" }],
      },
    }, phrases);
    const letter = brief.cards.find((card) => card.id === "letter");
    const git = letter?.items.find((item) => item.title === "git");
    assert.equal(git?.detail, "Ada");
    assert.equal(letter?.items.some((item) => item.title === "apps" && item.detail.includes("Cursor")), true);
    const lines = brief.cards.find((card) => card.id === "lines");
    assert.equal(lines?.primary, "explain yoyo");
    assert.equal(letter?.title, "hello Ada");
    const next = applySenses(brief, [
      { id: "power", code: "power", class: "power", name: "80", detail: "battery", state: "ok", why: "", speak: false, held: false },
      { id: "focus", code: "focus", class: "focus", name: "Cursor.exe", detail: "", state: "foreground", why: "", speak: false, held: false },
      { id: "poi:adobe", code: "poi", class: "poi", name: "Adobe", detail: "", state: "changed", why: "", speak: false, held: false },
      { id: "net", code: "net", class: "net", name: "network", detail: "", state: "offline", why: "offline", speak: false, held: false },
    ], senseLabels);
    const sense = next.cards.find((card) => card.id === "sense");
    assert.equal(sense?.items.length, 1);
    assert.equal(sense?.items[0]?.title, "net");
    assert.equal(JSON.stringify(next).includes("Cursor.exe"), false);
    assert.equal(JSON.stringify(next).includes("Adobe"), false);
  });
});
