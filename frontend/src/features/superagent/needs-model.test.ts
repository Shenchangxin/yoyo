import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { DeskView, Ticket, Trigger } from "./desk-model.ts";
import { laterDue, needItems, normalizeQueue, runningTickets, todayDue } from "./needs-model.ts";

const now = new Date("2026-11-06T20:00:00+08:00");

function view(tickets: Ticket[], queued: Trigger[]): DeskView {
  return {
    mode: "observe",
    paused: false,
    admission: { listedUnaddressed: "", unlistedAddressed: "", unlistedUnaddressed: "" },
    sources: [],
    tickets,
    threads: [],
    queued,
    evals: [],
    decisions: [],
    health: "ok",
    at: now.toISOString(),
  };
}

function ticket(over: Partial<Ticket>): Ticket {
  return { id: "tkt-1", stage: "working", action: "", person: false, origin: "desk", title: "t", ...over };
}

describe("needItems", () => {
  it("lists every kind of decision in one queue, due dates first and then the longest wait", () => {
    const items = needItems(
      view(
        [
          ticket({ id: "tkt-1", stage: "open", action: "answer", person: true, title: "周报", attention: "第 3 节写不写？", updatedAt: "2026-11-06T19:00:00+08:00" }),
          ticket({ id: "tkt-2", stage: "working", action: "running", person: false, title: "在跑" }),
          ticket({ id: "tkt-3", stage: "ended", action: "", person: true, title: "结束了" }),
        ],
        [
          { id: "t1", source: "github", sender: "alice", text: "看下 #163", addressed: true, queued: true, at: "2026-11-06T19:40:00+08:00" },
          { id: "c1", source: "calendar", sender: "", text: "周会", addressed: false, queued: true, due: "2026-11-06" },
          { id: "c2", source: "calendar", sender: "", text: "下周评审", addressed: false, queued: true, due: "2026-11-09" },
        ],
      ),
      normalizeQueue({
        offers: [
          { id: "o1", at: "2026-11-06T18:00:00+08:00", request: { action: "run_command", command: "go test ./...", session_id: "s1" } },
          { id: "o2", request: { action: "ask_user" } },
        ],
        drafts: [{ id: "d1", to: "wang@example.com", subject: "评审材料" }, { id: "d0", to: "li@example.com", subject: "已发出", sent: true }],
        inbox: [{ id: "i1", title: "已读的", unread: false }, { id: "i2", title: "构建完成", body: "全部通过", unread: true }],
      }),
      now,
    );
    assert.deepEqual(items.map((item) => item.id), ["due:c1", "offer:o1", "ticket:tkt-1", "trigger:t1", "draft:d1", "inbox:i2"]);
    assert.equal(items[2].kind, "question");
    assert.equal(items[2].detail, "第 3 节写不写？");
    assert.equal(items[1].detail, "go test ./...");
  });

  it("maps blocked and review tickets to their own kinds", () => {
    const items = needItems(view([
      ticket({ id: "a", stage: "open", action: "review", person: true }),
      ticket({ id: "b", stage: "open", action: "blocked", person: true }),
    ], []), normalizeQueue({}), now);
    assert.deepEqual(items.map((item) => item.kind).sort(), ["failed", "review"]);
  });

  it("is empty when nothing waits for a person", () => {
    assert.deepEqual(needItems(null, normalizeQueue(null), now), []);
  });

  it("brings personal send reviews and waiting personal work into the same queue", () => {
    const items = needItems(null, normalizeQueue({
      proposals: [
        { id: "p1", title: "回复王老师", account: "qq-mail", hash: "abc", created_at: "2026-11-06T17:00:00+08:00" },
        { id: "p2", title: "", hash: "x" },
      ],
      waiting: [{ id: "w1", title: "整理发票", status: "waiting_input", question: "要按月分吗？", updated_at: "2026-11-06T18:30:00+08:00" }],
    }), now);
    assert.deepEqual(items.map((item) => item.id), ["proposal:p1", "personal:w1"]);
    assert.equal(items[0].hash, "abc");
    assert.equal(items[0].detail, "qq-mail");
    assert.equal(items[1].detail, "要按月分吗？");
    assert.equal(items[1].status, "waiting_input");
  });
});

describe("due split", () => {
  it("keeps today and overdue in the queue and later dates for the next card", () => {
    const rows: Trigger[] = [
      { id: "past", source: "calendar", sender: "", text: "昨天", addressed: false, queued: true, due: "2026-11-05" },
      { id: "today", source: "calendar", sender: "", text: "今天", addressed: false, queued: true, due: "2026-11-06" },
      { id: "soon", source: "calendar", sender: "", text: "后天", addressed: false, queued: true, due: "2026-11-08" },
      { id: "next", source: "calendar", sender: "", text: "明天", addressed: false, queued: true, due: "2026-11-07" },
    ];
    assert.deepEqual(todayDue(rows, now).map((row) => row.id), ["past", "today"]);
    assert.deepEqual(laterDue(rows, now).map((row) => row.id), ["next", "soon"]);
  });
});

describe("runningTickets", () => {
  it("counts working tickets that are not queued", () => {
    assert.equal(runningTickets(view([
      ticket({ id: "a", stage: "working", action: "running" }),
      ticket({ id: "b", stage: "working", action: "queued" }),
      ticket({ id: "c", stage: "open", action: "answer", person: true }),
    ], [])), 1);
  });
});
