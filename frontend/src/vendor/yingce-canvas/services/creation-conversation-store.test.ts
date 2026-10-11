import assert from "node:assert/strict";
import test from "node:test";

// @ts-expect-error -- Node 原生 TypeScript 测试运行器需要保留扩展名。
import { parseCreationConversationValue } from "./creation-conversation-store.ts";

test("旧的对话数组和新的当前对话记录都能读出来", () => {
  const legacy = parseCreationConversationValue(JSON.stringify([{ id: "a" }]));
  assert.equal(legacy.corrupt, false);
  assert.equal(legacy.activeId, "");
  assert.equal(legacy.conversations.length, 1);

  const current = parseCreationConversationValue(JSON.stringify({ activeId: "b", conversations: [{ id: "b" }] }));
  assert.equal(current.activeId, "b");
  assert.equal(current.conversations.length, 1);
});

test("损坏的对话记录被标出来，而不是抛给页面卡在加载", () => {
  const parsed = parseCreationConversationValue("{");
  assert.equal(parsed.corrupt, true);
  assert.deepEqual(parsed.conversations, []);
});
