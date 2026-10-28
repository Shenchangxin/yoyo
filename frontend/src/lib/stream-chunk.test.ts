import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { BlockAssembler, expandAssistantStream, textFromBlocks } from "./stream-chunk.ts";
import { ConversationFold } from "./conversation-fold.ts";
import { ClientAssistantStream } from "./assistant-stream.ts";
import { eventFromNotice, itemFromEvent, mergeItem, noticeCarriesInline, parseLiveNotice } from "./stream-fold.ts";
import type { Item } from "./protocol.ts";

/**
 * 拥有：块组装、对话折叠、条目合并和客户端助手流在这些用例上的约定。
 * 不拥有：这些类型的实现。
 */

describe("BlockAssembler", () => {
  /** 只有文本增量时拼成一整段正文。 */
  it("assembles delta-only text", () => {
    const a = new BlockAssembler();
    a.push({ type: "text-delta", index: 0, text: "he" });
    a.push({ type: "text-delta", index: 0, text: "llo" });
    assert.equal(textFromBlocks(a.blocks()), "hello");
  });

  /** 结束原因为 max-tokens 时丢掉工具调用，只留正文。 */
  it("drops tools on max-tokens", () => {
    const a = new BlockAssembler();
    a.push({ type: "text-delta", index: 0, text: "cut" });
    a.push({ type: "tool-call-delta", index: 1, id: "c1", name: "write_file", argumentsDelta: "{" });
    a.push({ type: "finish", reason: { kind: "max-tokens" } });
    assert.deepEqual(a.blocks(), [{ type: "text", text: "cut" }]);
  });

  /** 打包的文本块按 dt 展开成带时间的片段。 */
  it("expands packed text chunks", () => {
    const got = expandAssistantStream([
      { type: "text-chunks", time0: 1, index: 0, dt: [1], texts: ["he", "llo"] },
    ]);
    assert.equal(got.length, 2);
    assert.equal(got[1].time, 2);
  });
});

/** 补齐转录条目的缺省字段。 */
function item(partial: Partial<Item> & Pick<Item, "type" | "key">): Item {
  return {
    sessionId: "s",
    source: "model",
    ts: "",
    text: "",
    name: "",
    delta: false,
    payload: {},
    ...partial,
  };
}

describe("ConversationFold", () => {
  /** 流帧先把助手正文长出来，收束后再换成最终正文。 */
  it("grows assistant text from stream frames then settles", () => {
    const f = new ConversationFold();
    f.accept(item({
      key: "st",
      type: "assistant_stream",
      payload: { kind: "start", attempt_id: "a1", round: "s:r1", id: "s:r1", turn: 1, step: 1 },
    }));
    f.accept(item({
      key: "c1",
      type: "assistant_stream",
      payload: {
        kind: "chunk", attempt_id: "a1", round: "s:r1", index: 0,
        chunk: { type: "text-delta", index: 0, text: "hi" },
      },
    }));
    const live = f.items().find((x) => x.type === "assistant");
    assert.equal(live?.text, "hi");
    assert.equal(live?.delta, true);
    f.accept(item({
      key: "s:assistant:s:r1",
      type: "assistant",
      text: "hi there",
      payload: { id: "s:r1", round: "s:r1", attempt_id: "a1", blocks: [{ type: "text", text: "hi there" }] },
    }));
    f.accept(item({ key: "end", type: "step_end", payload: { round: "s:r1" } }));
    const settled = f.items().find((x) => x.type === "assistant");
    assert.equal(settled?.text, "hi there");
    assert.equal(settled?.delta, false);
  });

  /** 各自 from 出来的列表也能把文本增量接上。 */
  it("mergeItem grows text across independent from() calls", () => {
    let list: Item[] = [];
    list = mergeItem(list, item({
      key: "st",
      type: "assistant_stream",
      payload: { kind: "start", attempt_id: "a1", round: "s:r1", id: "s:r1", turn: 1, step: 1 },
    }));
    list = mergeItem(list, item({
      key: "c1",
      type: "assistant_stream",
      payload: {
        kind: "chunk", attempt_id: "a1", round: "s:r1", index: 0, index_from: 0,
        chunk: { type: "text-delta", index: 0, text: "he" },
      },
    }));
    list = mergeItem(list, item({
      key: "c2",
      type: "assistant_stream",
      payload: {
        kind: "chunk", attempt_id: "a1", round: "s:r1", index: 1, index_from: 1,
        chunk: { type: "text-delta", index: 0, text: "llo" },
      },
    }));
    const live = list.find((x) => x.type === "assistant");
    assert.equal(live?.text, "hello");
    assert.equal(live?.payload?.stream_index, 1);
  });

  /** 同一 request_id 的正式用户消息替换乐观气泡。 */
  it("replaces optimistic user by request_id", () => {
    const f = new ConversationFold();
    f.accept(item({
      key: "ui:1",
      type: "user",
      source: "ui",
      text: "hello",
      payload: { request_id: "req-1", text: "hello" },
    }));
    f.accept(item({
      key: "s:user:t",
      type: "user",
      source: "user",
      text: "hello",
      payload: { request_id: "req-1", text: "hello" },
    }));
    const users = f.items().filter((x) => x.type === "user");
    assert.equal(users.length, 1);
    assert.equal(users[0].source, "user");
  });

  /** 助手已经开始之后，作曲回声仍在原位被正式用户消息替换。 */
  it("replaces the composer echo in place after a live assistant has started", () => {
    const f = new ConversationFold();
    f.accept(item({
      key: "ui:1",
      type: "user",
      source: "ui",
      text: "hello",
      payload: { request_id: "req-1", text: "hello" },
    }));
    f.accept(item({
      key: "st",
      type: "assistant_stream",
      payload: { kind: "start", attempt_id: "a1", round: "s:r1", id: "s:r1", turn: 1, step: 1 },
    }));
    f.accept(item({
      key: "s:user:t",
      type: "user",
      source: "user",
      text: "hello",
      payload: { request_id: "req-1", text: "hello" },
    }));
    const users = f.items().filter((x) => x.type === "user");
    assert.equal(users.length, 1);
    assert.equal(users[0].source, "user");
    assert.equal(users[0].key, "s:user:t");
    const types = f.items().map((x) => x.type);
    assert.deepEqual(types, ["user", "assistant"]);
  });

  /** 旧式助手切片一直拼到收束，重复的同一序号不再加一次。 */
  it("concatenates legacy assistant token slices until settlement", () => {
    const f = new ConversationFold();
    f.accept(item({
      key: "s:assistant:s:r1",
      type: "assistant",
      text: "he",
      delta: true,
      payload: { id: "s:r1", round: "s:r1", delta: true, index: 1 },
    }));
    f.accept(item({
      key: "s:assistant:s:r1",
      type: "assistant",
      text: "llo",
      delta: true,
      payload: { id: "s:r1", round: "s:r1", delta: true, index: 2, index_from: 2 },
    }));
    f.accept(item({
      key: "s:assistant:s:r1",
      type: "assistant",
      text: "llo",
      delta: true,
      payload: { id: "s:r1", round: "s:r1", delta: true, index: 2, index_from: 2 },
    }));
    const live = f.items().find((x) => x.type === "assistant");
    assert.equal(live?.text, "hello");
    assert.equal(live?.delta, true);
    f.accept(item({
      key: "s:assistant:s:r1",
      type: "assistant",
      text: "hello",
      payload: { id: "s:r1", round: "s:r1", blocks: [{ type: "text", text: "hello" }] },
    }));
    const settled = f.items().find((x) => x.type === "assistant");
    assert.equal(settled?.text, "hello");
    assert.equal(settled?.delta, false);
  });

  /** 收束后的条目不再带着流磁带。 */
  it("drops the settlement stream tape from the live seat", () => {
    const f = new ConversationFold();
    f.accept(item({
      key: "s:assistant:s:r1",
      type: "assistant",
      text: "hi",
      payload: {
        id: "s:r1",
        round: "s:r1",
        blocks: [{ type: "text", text: "hi" }],
        stream: [{ type: "text-delta", text: "hi" }],
      },
    }));
    const settled = f.items().find((x) => x.type === "assistant");
    assert.equal(settled?.text, "hi");
    assert.equal(settled?.payload?.stream, undefined);
  });
});

describe("frame-only live notices", () => {
  /** 只有帧的助手流仍能长出正文，没有内联事件也不会去回拉。 */
  it("applies a frame-only assistant stream", () => {
    const start = eventFromNotice(parseLiveNotice({
      seq: 1,
      session_id: "s",
      type: "assistant_stream",
      frame: { kind: "start", attempt_id: "a1", id: "s:r1", round: "s:r1", turn: 1, step: 1, index: 0, index_from: 0 },
    }));
    const chunk = eventFromNotice(parseLiveNotice({
      seq: 2,
      session_id: "s",
      type: "assistant_stream",
      frame: {
        kind: "chunk",
        attempt_id: "a1",
        id: "s:r1",
        round: "s:r1",
        index: 0,
        index_from: 0,
        chunk: { type: "text-delta", index: 0, text: "hi" },
      },
    }));
    assert.equal(noticeCarriesInline(parseLiveNotice({ seq: 8, session_id: "s", type: "assistant" })), false);
    let list = mergeItem([], itemFromEvent(start));
    list = mergeItem(list, itemFromEvent(chunk));
    assert.equal(list.find((x) => x.type === "assistant")?.text, "hi");
  });

  /** 旧的助手增量帧只带一块正文，仍按回合接上。 */
  it("applies a frame-only assistant delta", () => {
    const ev = eventFromNotice(parseLiveNotice({
      seq: 4,
      session_id: "s",
      type: "assistant",
      frame: {
        kind: "delta",
        id: "r1",
        round: "r1",
        index: 1,
        index_from: 1,
        block: "text",
        chunk: { type: "text-delta", index: 1, text: "hi" },
      },
    }));
    const it = itemFromEvent(ev);
    assert.equal(it.delta, true);
    assert.equal(it.text, "hi");
    assert.equal(it.payload.id, "r1");
    const list = mergeItem([], it);
    assert.equal(list[0]?.text, "hi");
  });
});

describe("itemFromEvent", () => {
  /** 事件收成条目时，收束用的流磁带在进 React 之前就丢掉。 */
  it("drops the settlement stream tape before React sees it", () => {
    const it = itemFromEvent({
      type: "assistant",
      session_id: "s",
      payload: {
        text: "hi",
        id: "s:r1",
        stream: [{ type: "text-delta", text: "hi" }],
      },
    });
    assert.equal(it.text, "hi");
    assert.equal(it.payload.stream, undefined);
  });
});

describe("ClientAssistantStream", () => {
  /** 合并过的 index_from 到 index 整段收下，下一序号跟着前进。 */
  it("accepts coalesced index_from..index and advances nextIndex", () => {
    const s = new ClientAssistantStream();
    const start = s.acceptFrame({ kind: "start", attemptId: "a1", revision: 1, index: 0, indexFrom: 0, turn: 1, step: 1 });
    assert.equal(start?.type, "start");
    const chunk = s.acceptFrame({
      kind: "chunk",
      attemptId: "a1",
      revision: 4,
      index: 2,
      indexFrom: 0,
      chunk: { type: "text-delta", index: 0, text: "abc" },
    });
    assert.equal(chunk?.type, "chunk");
    assert.equal(s.activeAttempt?.nextIndex, 3);
  });

  /** 重复的开始帧和已经应用过的前缀不再收下。 */
  it("ignores a duplicate start and already-applied prefix", () => {
    const s = new ClientAssistantStream();
    s.acceptFrame({ kind: "start", attemptId: "a1", revision: 1, index: 0, indexFrom: 0, turn: 1, step: 1 });
    s.acceptFrame({ kind: "chunk", attemptId: "a1", revision: 2, index: 0, indexFrom: 0, chunk: { type: "text-delta", index: 0, text: "hi" } });
    assert.equal(s.acceptFrame({ kind: "start", attemptId: "a1", revision: 1, index: 0, indexFrom: 0, turn: 1, step: 1 }), undefined);
    assert.equal(s.acceptFrame({
      kind: "chunk", attemptId: "a1", revision: 2, index: 0, indexFrom: 0,
      chunk: { type: "text-delta", index: 0, text: "hi" },
    }), undefined);
    assert.equal(s.activeAttempt?.nextIndex, 1);
  });

  /** 已提交的尝试留到 retire 才丢掉。 */
  it("keeps a committed attempt until retire", () => {
    const s = new ClientAssistantStream();
    s.acceptFrame({ kind: "start", attemptId: "a1", revision: 1, index: 0, indexFrom: 0, turn: 1, step: 1 });
    s.acceptFrame({ kind: "end", attemptId: "a1", revision: 2, index: 0, indexFrom: 0, outcome: "committed" });
    assert.equal(s.retained?.attemptId, "a1");
    s.retire("a1");
    assert.equal(s.retained, undefined);
  });
});
