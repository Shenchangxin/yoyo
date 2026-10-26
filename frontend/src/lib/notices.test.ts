/**
 * 拥有：铃铛留哪些条目、一行摘要、从事件时间起算的过期，以及存储里的去重和裁剪。
 * 不拥有：通知列表的绘制。
 */

import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { en } from "./copy.ts";
import { NOTICE_TTL_MS, noticeExpired, noticeFromItem, noticeSummary } from "./notices.ts";
import type { Item } from "./protocol.ts";
import { useUI } from "./store.ts";

const copy = { app: en.app, transcript: en.transcript };

/** 补齐一条转录，缺省是已结束的回合。 */
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
  /** 已完成回合留下摘要。换行收成空格。标题用完成文案，不用类型名。 */
  it("keeps a finished turn with its summary, never the raw type name", () => {
    const n = noticeFromItem(item({ text: "  修好了登录\n\n页面  " }), copy);
    assert.ok(n);
    assert.equal(n.title, en.app.turnFinished);
    assert.equal(n.body, "修好了登录 页面");
    assert.equal(n.sessionId, "s1");
  });

  /** 回复为空时正文为空，不拿 turn_end 当正文。 */
  it("does not show turn_end as the body when the reply is empty", () => {
    const n = noticeFromItem(item({ text: "" }), copy);
    assert.ok(n);
    assert.equal(n.body, "");
  });

  /** 批准条目不成通知。 */
  it("never turns approvals into notices; the approvals section owns them", () => {
    const a = item({ type: "approval", text: "", payload: { id: "p1", action: "shell", command: "npm test" } });
    assert.equal(noticeFromItem(a, copy), null);
  });

  /** kind 为 canceled 的错误不成通知。 */
  it("drops a deliberate Stop, which arrives as a canceled error", () => {
    const e = item({ type: "error", text: "Stopped", payload: { kind: "canceled", error: "Stopped" } });
    assert.equal(noticeFromItem(e, copy), null);
  });

  /** stop 为 plan_open 的回合结束不成通知。 */
  it("drops a turn that ended with plan steps open; its plan card says so", () => {
    assert.equal(noticeFromItem(item({ payload: { stop: "plan_open" } }), copy), null);
  });

  /** 供应商错误留下，标题用错误文案，正文非空。 */
  it("keeps a real error", () => {
    const e = item({ type: "error", text: "boom", payload: { kind: "provider", error: "boom" } });
    const n = noticeFromItem(e, copy);
    assert.ok(n);
    assert.equal(n.title, en.app.errorNotice);
    assert.ok(n.body.length > 0);
  });

  /** 正在看的会话，完成和错误都不进铃铛。 */
  it("skips the session the user is already looking at", () => {
    assert.equal(noticeFromItem(item({ text: "done" }), copy, true), null);
    assert.equal(noticeFromItem(item({ type: "error", text: "boom", payload: { kind: "provider" } }), copy, true), null);
  });

  /** 同一条目两次收成同一个 id。 */
  it("gives the same id to the same item, so the store can dedupe it", () => {
    const a = noticeFromItem(item({ text: "x" }), copy);
    const b = noticeFromItem(item({ text: "x" }), copy);
    assert.equal(a?.id, b?.id);
  });
});

describe("notice store", () => {
  /** 同一 id 只留一行。关掉一行只影响它。按会话清掉时该会话的行都走。 */
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
  /** 换行收成空格。超过上限的正文截断并以省略号结尾，长度等于上限。 */
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

  /** 距事件刚好一个 TTL 时过期，少一毫秒则没有。 */
  it("expires exactly one TTL after the event, counted from the event time", () => {
    assert.equal(noticeExpired(n, at + NOTICE_TTL_MS - 1), false);
    assert.equal(noticeExpired(n, at + NOTICE_TTL_MS), true);
  });

  /** 读不出时间的通知，即使过了很多个 TTL 也不过期。 */
  it("keeps a notice whose time cannot be read, until it is dismissed", () => {
    assert.equal(noticeExpired({ ts: "" }, at + 10 * NOTICE_TTL_MS), false);
  });
});

describe("notice expiry in the store", () => {
  /** 到达时已经过期的重放直接丢掉。未过期的留到整好一个 TTL 再被裁掉。 */
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
