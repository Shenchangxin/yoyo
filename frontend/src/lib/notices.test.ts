import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { en } from "./copy.ts";
import { NOTICE_TTL_MS, noticeExpired, noticeFromItem, noticeSummary } from "./notices.ts";
import type { Item } from "./protocol.ts";
import { useUI } from "./store.ts";

const copy = { app: en.app, transcript: en.transcript };

function item(over: Partial<Item>): Item {
  return {
    key: "s1:turn_end:1",
    type: "turn_end",
    sessionId: "s1",
    source: "runtime",
    ts: "2026-10-14T04:00:00Z",
    text: "",
    name: "",
    delta: false,
    payload: {},
    ...over,
  } as Item;
}

describe("noticeFromItem", () => {
  it("keeps a finished turn with its summary, never the raw type name", () => {
    const n = noticeFromItem(item({ text: "  修好了登录\n\n页面  " }), copy);
    assert.ok(n);
    assert.equal(n.title, en.app.turnFinished);
    assert.equal(n.body, "修好了登录 页面");
    assert.equal(n.sessionId, "s1");
  });

  it("does not show turn_end as the body when the reply is empty", () => {
    const n = noticeFromItem(item({ text: "" }), copy);
    assert.ok(n);
    assert.equal(n.body, "");
  });

  it("never turns approvals into notices; the approvals section owns them", () => {
    const a = item({ type: "approval", text: "", payload: { id: "p1", action: "shell", command: "npm test" } });
    assert.equal(noticeFromItem(a, copy), null);
  });

  it("drops a deliberate Stop, which arrives as a canceled error", () => {
    const e = item({ type: "error", text: "Stopped", payload: { kind: "canceled", error: "Stopped" } });
    assert.equal(noticeFromItem(e, copy), null);
  });

  it("drops a turn that ended with plan steps open; its plan card says so", () => {
    assert.equal(noticeFromItem(item({ payload: { stop: "plan_open" } }), copy), null);
  });

  it("keeps a real error", () => {
    const e = item({ type: "error", text: "boom", payload: { kind: "provider", error: "boom" } });
    const n = noticeFromItem(e, copy);
    assert.ok(n);
    assert.equal(n.title, en.app.errorNotice);
    assert.ok(n.body.length > 0);
  });

  it("skips the session the user is already looking at", () => {
    assert.equal(noticeFromItem(item({ text: "done" }), copy, true), null);
    assert.equal(noticeFromItem(item({ type: "error", text: "boom", payload: { kind: "provider" } }), copy, true), null);
  });

  it("gives the same id to the same item, so the store can dedupe it", () => {
    const a = noticeFromItem(item({ text: "x" }), copy);
    const b = noticeFromItem(item({ text: "x" }), copy);
    assert.equal(a?.id, b?.id);
  });
});

describe("notice store", () => {
  it("keeps one row per id, dismisses one row, and clears a conversation's rows", () => {
    const { clearNotices, pushNotice, dismissNotice, dismissNoticesFor } = useUI.getState();
    clearNotices();
    const row = (id: string, sessionId: string) => ({ id, title: "t", body: "", ts: "", sessionId });
    pushNotice(row("a", "s1"));
    pushNotice(row("a", "s1"));
    pushNotice(row("b", "s2"));
    assert.deepEqual(useUI.getState().notices.map((n) => n.id), ["b", "a"]);
    dismissNotice("b");
    assert.deepEqual(useUI.getState().notices.map((n) => n.id), ["a"]);
    pushNotice(row("c", "s1"));
    dismissNoticesFor("s1");
    assert.equal(useUI.getState().notices.length, 0);
    clearNotices();
  });
});

describe("noticeSummary", () => {
  it("collapses whitespace and clips long replies", () => {
    assert.equal(noticeSummary("a \n b"), "a b");
    const long = "字".repeat(200);
    const s = noticeSummary(long, 10);
    assert.equal(s.length, 10);
    assert.ok(s.endsWith("…"));
  });
});

describe("noticeExpired", () => {
  const at = Date.parse("2026-10-14T04:00:00Z");
  const n = { ts: "2026-10-14T04:00:00Z" };

  it("expires exactly one TTL after the event, counted from the event time", () => {
    assert.equal(noticeExpired(n, at + NOTICE_TTL_MS - 1), false);
    assert.equal(noticeExpired(n, at + NOTICE_TTL_MS), true);
  });

  it("keeps a notice whose time cannot be read, until it is dismissed", () => {
    assert.equal(noticeExpired({ ts: "" }, at + 10 * NOTICE_TTL_MS), false);
  });
});

describe("notice expiry in the store", () => {
  it("drops a replayed notice that is already past its time, and prunes the rest as time passes", () => {
    const { clearNotices, pushNotice, pruneNotices } = useUI.getState();
    clearNotices();
    const at = Date.now();
    const row = (id: string, ts: number) => ({ id, title: "t", body: "", ts: new Date(ts).toISOString(), sessionId: "s1" });
    pushNotice(row("stale", at - NOTICE_TTL_MS - 1));
    assert.equal(useUI.getState().notices.length, 0);
    pushNotice(row("fresh", at));
    assert.deepEqual(useUI.getState().notices.map((n) => n.id), ["fresh"]);
    pruneNotices(at + NOTICE_TTL_MS - 1);
    assert.deepEqual(useUI.getState().notices.map((n) => n.id), ["fresh"]);
    pruneNotices(at + NOTICE_TTL_MS);
    assert.equal(useUI.getState().notices.length, 0);
    clearNotices();
  });
});
