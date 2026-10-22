import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  achievementLabel,
  groupIdols,
  idolError,
  imageURL,
  isCooling,
  linesOf,
  mmss,
  snapOf,
  type Idol,
} from "./model.ts";

const errors = {
  cooldown: "冷",
  ended: "完",
  needModel: "模型",
  noDraws: "没票",
  noReset: "没卡",
  notHolding: "没有",
  nobodyLow: "不用",
  paused: "停",
  emptyReply: "空",
  missing: "无",
};

describe("idol transcript", () => {
  it("splits a turn into what you said and what she said", () => {
    const lines = linesOf([{
      id: "r1",
      kind: "turn",
      at: "2026-10-22T04:00:00Z",
      user: "在吗",
      segments: [{ subtitle: "在。" }, { spoken: "你今天怎么样" }],
      delta: 2,
      photo_id: "pic",
    }]);
    assert.equal(lines.length, 2);
    assert.equal(lines[0].role, "user");
    assert.equal(lines[1].role, "idol");
    if (lines[1].role === "idol") {
      assert.equal(lines[1].text, "在。\n你今天怎么样");
      assert.equal(lines[1].photo, "pic");
      assert.equal(lines[1].delta, 2);
    }
  });

  it("keeps a gift as an event, not a chat bubble", () => {
    const lines = linesOf([{ id: "g", kind: "event", gift_id: "tea", segments: [{ subtitle: "收到了热茶" }], delta: 4 }]);
    assert.equal(lines.length, 1);
    assert.equal(lines[0].role, "event");
  });
});

describe("idol desk helpers", () => {
  it("groups by circle and leaves outsiders untitled", () => {
    const idols: Idol[] = [
      { ...base("a"), name: "甲", circle_id: "c1", pinned: true },
      { ...base("b"), name: "乙", circle_id: "c1" },
      { ...base("c"), name: "丙" },
      { ...base("d"), name: "丁", hidden: true },
    ];
    const groups = groupIdols(idols, [{ id: "c1", name: "宿舍" }]);
    assert.equal(groups.length, 2);
    assert.deepEqual(groups[0].idols.map((i) => i.id), ["a", "b"]);
    assert.deepEqual(groups[1].idols.map((i) => i.id), ["c"]);
  });

  it("puts the group that is on the desktop first", () => {
    const idols: Idol[] = [
      { ...base("a"), circle_id: "c1" },
      { ...base("b"), pinned: true },
    ];
    const groups = groupIdols(idols, [{ id: "c1", name: "宿舍" }]);
    assert.equal(groups[0].id, "");
    assert.deepEqual(groups[0].idols.map((i) => i.id), ["b"]);
  });

  it("reads cooling, snapshots, and error codes", () => {
    assert.equal(isCooling("2099-01-01T00:00:00Z", Date.parse("2026-01-01T00:00:00Z")), true);
    assert.equal(isCooling("", Date.now()), false);
    assert.equal(mmss(65000), "01:05");
    const snap = snapOf({ idols: [{ id: "a", name: "甲", pinned: true }], chat_ready: true, pinned: { id: "a", name: "甲" }, portrait: "abc" });
    assert.equal(snap.pinnedId, "a");
    assert.equal(snap.chatReady, true);
    assert.equal(idolError(new Error("/api/idol: cooldown"), errors), "冷");
    assert.equal(idolError(new Error("configure a chat model in Settings"), errors), "模型");
    assert.equal(achievementLabel("close:a", { a: "甲" }, {}, "和{name}亲近", "和{name}亲密"), "和甲亲近");
    assert.equal(imageURL("/9j/abc").startsWith("data:image/jpeg"), true);
  });
});

function base(id: string): Idol {
  return {
    id, name: id, intro: "", color: "#c47b8a", spoken_lang: "zh", subtitle_lang: "zh",
    sheet: "", facts: "", readings: "", voice_id: "", portrait: "", circle_id: "",
    work_enabled: true, work_profile_id: "", pinned: false, hidden: false, kind: "", outfit_id: "", liked_gifts: [],
  };
}
