import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { BlockAssembler, expandAssistantStream, textFromBlocks } from "./stream-chunk.ts";
import { ConversationFold } from "./conversation-fold.ts";
import { ClientAssistantStream } from "./assistant-stream.ts";
import { itemFromEvent, mergeItem } from "./stream-fold.ts";
import type { Item } from "./protocol.ts";

describe("BlockAssembler", () => {
  it("assembles delta-only text", () => {
    const a = new BlockAssembler();
    a.push({ type: "text-delta", index: 0, text: "he" });
    a.push({ type: "text-delta", index: 0, text: "llo" });
    assert.equal(textFromBlocks(a.blocks()), "hello");
  });

  it("drops tools on max-tokens", () => {
    const a = new BlockAssembler();
    a.push({ type: "text-delta", index: 0, text: "cut" });
    a.push({ type: "tool-call-delta", index: 1, id: "c1", name: "write_file", argumentsDelta: "{" });
    a.push({ type: "finish", reason: { kind: "max-tokens" } });
    assert.deepEqual(a.blocks(), [{ type: "text", text: "cut" }]);
  });

  it("expands packed text chunks", () => {
    const got = expandAssistantStream([
      { type: "text-chunks", time0: 1, index: 0, dt: [1], texts: ["he", "llo"] },
    ]);
    assert.equal(got.length, 2);
    assert.equal(got[1].time, 2);
  });
});

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

describe("itemFromEvent", () => {
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

  it("keeps a committed attempt until retire", () => {
    const s = new ClientAssistantStream();
    s.acceptFrame({ kind: "start", attemptId: "a1", revision: 1, index: 0, indexFrom: 0, turn: 1, step: 1 });
    s.acceptFrame({ kind: "end", attemptId: "a1", revision: 2, index: 0, indexFrom: 0, outcome: "committed" });
    assert.equal(s.retained?.attemptId, "a1");
    s.retire("a1");
    assert.equal(s.retained, undefined);
  });
});
