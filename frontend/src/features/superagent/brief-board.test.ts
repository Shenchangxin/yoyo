import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { BriefCard, MorningBrief } from "./work-model.ts";
import { actionLabel, briefLayout, briefProgress, cardActionable, cardHandled, checkedCount, itemKey, shortKicker } from "./brief-board.ts";

function card(over: Partial<BriefCard>): BriefCard {
  return { id: "c", kicker: "", title: "t", lede: "", items: [], choices: [], primary: "", secondary: "", ...over };
}

const items = [{ title: "a", detail: "" }, { title: "b", detail: "" }];

describe("brief board", () => {
  it("infers the layout when the model did not pick one", () => {
    assert.equal(briefLayout(card({ choices: [{ id: "x", title: "x", detail: "" }, { id: "y", title: "y", detail: "" }] })), "compare");
    assert.equal(briefLayout(card({ items })), "checks");
    assert.equal(briefLayout(card({})), "plain");
    assert.equal(briefLayout(card({ layout: "rows", items })), "rows");
  });

  it("counts a card as handled after a button or once every item is ticked", () => {
    const list = card({ id: "s", layout: "checks", items });
    assert.equal(cardHandled(list, {}), false);
    assert.equal(checkedCount(list, { [itemKey("s", 0)]: "done" }), 1);
    assert.equal(cardHandled(list, { [itemKey("s", 0)]: "done", [itemKey("s", 1)]: "done" }), true);
    assert.equal(cardHandled(list, { [itemKey("s", 0)]: "done", [itemKey("s", 1)]: "" }), false);
    assert.equal(cardHandled(card({ id: "b", primary: "Go" }), { b: "primary" }), true);
    assert.equal(cardHandled(card({ id: "b", primary: "Go" }), { b: "" }), false);
  });

  it("only counts cards with something to do toward progress", () => {
    const brief: MorningBrief = {
      cards: [
        card({ id: "a", primary: "Do it" }),
        card({ id: "b", layout: "checks", items }),
        card({ id: "c", layout: "stats", stats: [{ label: "apps", value: "3" }] }),
        card({ id: "d", task: "Look into this\nwith detail" }),
      ],
      picked: { a: "primary", [itemKey("b", 0)]: "done" },
    };
    assert.equal(cardActionable(brief.cards[2]), false);
    assert.equal(cardActionable(brief.cards[3]), true);
    assert.deepEqual(briefProgress(brief), { done: 1, total: 3 });
  });

  it("labels buttons and index chips briefly", () => {
    assert.equal(actionLabel("", "看看 hypit 停在哪里，给我一个继续的下一步，并说明理由"), "看看 hypit 停在哪里，给我一个");
    assert.equal(actionLabel("先回周报", "ignored"), "先回周报");
    assert.equal(shortKicker(card({ kicker: "今天先确认 · 2026-11-06" })), "今天先确认");
    assert.equal(shortKicker(card({ title: "这 2 项一确认就能继续" })), "这 2 项一确认…");
  });
});
