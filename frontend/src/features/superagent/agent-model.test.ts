/**
 * 拥有：工单上 agent 判断的用例：谁在做、要不要先登录、用量细字和换人名单。
 * 不拥有：任务页怎么画，也不碰网络。
 */

import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { BuddyView } from "../../lib/client.ts";
import { approvalKind, compactTokens, external, money, reassignTargets, ticketAgent, ticketNeed, usageText } from "./agent-model.ts";
import type { Ticket } from "./desk-model.ts";

/** buddy 造一项装好、启用的 agent，默认在线。 */
function buddy(id: string, patch: Partial<BuddyView> = {}): BuddyView {
  return {
    id,
    name: id,
    install: { kind: "managed" },
    state: "online",
    config: { enabled: true, mode: "managed", approval: "yoyo", maxParallel: 1, secrets: [] },
    ...patch,
  };
}

const ticket: Ticket = { id: "tkt-1", stage: "working", action: "", person: false, origin: "desk:s1:t1", title: "修登录页" };

describe("ticket agent", () => {
  /** 没写 agent 的工单是 yoyo 做的；最近一次派工最后一条报告带 login 时要先去登录，换了新派工就不算了。 */
  it("reads who works on it and what it needs", () => {
    assert.equal(ticketAgent(ticket), "yoyo");
    assert.equal(external(ticket), false);
    assert.equal(external({ agent: "codex" }), true);
    const blocked = { ...ticket, agent: "codex", assignments: [{ id: "asg-1", state: "done", lines: [{ state: "blocked", summary: "Codex 需要登录", need: "login" }] }] };
    assert.equal(ticketNeed(blocked), "login");
    assert.equal(ticketNeed({ assignments: [...blocked.assignments, { id: "asg-2", state: "pending" }] }), "");
  });

  /** 用量写成「上下文 53k/200k · $0.05」；只有花费或只有上下文时只写那一样；没报过时为空。 */
  it("writes usage short", () => {
    const tpl = "上下文 {used}/{size}";
    assert.equal(usageText({ used: 53_210, size: 200_000, cost: 0.05, currency: "USD" }, tpl), "上下文 53k/200k · $0.05");
    assert.equal(usageText({ size: 1_000_000 }, tpl), "上下文 0/1M");
    assert.equal(usageText({ cost: 1.5, currency: "eur" }, tpl), "1.50 EUR");
    assert.equal(usageText(undefined, tpl), "");
    assert.equal(compactTokens(999), "999");
    assert.equal(compactTokens(1_250_000), "1.3M");
    assert.equal(money(0.004), "$0.0040");
  });

  /** 审批的动作名去掉 buddy. 前缀后查文案，不认识的和原型链上的名字都落到 other。 */
  it("names approval actions", () => {
    const kinds = { execute: "运行命令", other: "做一个操作" };
    assert.equal(approvalKind("buddy.execute", kinds), "运行命令");
    assert.equal(approvalKind("buddy.think", kinds), "做一个操作");
    assert.equal(approvalKind("toString", kinds), "做一个操作");
  });

  /** 换人只列现在能派的，yoyo 在前，不列正在做的这个。 */
  it("offers routable agents and yoyo", () => {
    const list = [
      buddy("claude", { name: "Claude Code", order: 1, state: "needs_auth" }),
      buddy("codex", { name: "Codex", order: 2 }),
      buddy("gemini", { name: "Gemini CLI", order: 3, config: { enabled: false, mode: "managed", approval: "yoyo", maxParallel: 1, secrets: [] } }),
      buddy("yoyo", { builtin: true, install: { kind: "none" } }),
    ];
    assert.deepEqual(reassignTargets(list, "codex").map((v) => v.id), ["yoyo"]);
    assert.deepEqual(reassignTargets(list, "yoyo").map((v) => v.id), ["codex"]);
  });
});
