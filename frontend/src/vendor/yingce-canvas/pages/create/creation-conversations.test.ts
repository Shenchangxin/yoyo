import assert from "node:assert/strict";
import test from "node:test";

// @ts-expect-error -- Node 原生 TypeScript 测试运行器需要保留扩展名。
import { attachCreationTaskContexts, reconcileCreationTaskMessages, resolveActiveCreationConversationId } from "./creation-conversations.ts";

const runtime = { recoverCreationTextTask() { return null; } };

function conversation() {
  return {
    id: "c1",
    title: "新创作",
    updatedAt: "2026-01-01T00:00:00.000Z",
    messages: [
      { id: "u", role: "user", mode: "image", content: "画两张", createdAt: "2026-01-01T00:00:00.000Z" },
      { id: "a", role: "assistant", mode: "image", content: "", createdAt: "2026-01-01T00:00:00.000Z", status: "pending", taskIds: ["t1", "t2"] },
    ],
  };
}

function task(id: string, status: string, urls: string[] = []) {
  return {
    id,
    status,
    type: "image",
    updatedAt: "2026-01-02T00:00:00.000Z",
    creationResultUrls: urls,
    ...(status === "failed" ? { error: "上游失败" } : {}),
  };
}

test("重新打开时回到上次选中的对话，而不是最新建出来的空对话", () => {
  const conversations = [
    { id: "empty", updatedAt: "2026-04-02T00:00:00.000Z", messages: [] },
    { id: "used", updatedAt: "2026-04-01T00:00:00.000Z", messages: [{ id: "m" }] },
  ];
  assert.equal(resolveActiveCreationConversationId(conversations, ""), "used");
  assert.equal(resolveActiveCreationConversationId(conversations, "empty"), "empty");
});

test("多图恢复要看到整批任务后才结束，部分失败会写进结果", () => {
  const source = conversation();
  const [onlyFirst] = attachCreationTaskContexts([task("t1", "succeeded", ["https://img/1"])], [source]);
  const pending = reconcileCreationTaskMessages(runtime, [source], [onlyFirst]);
  assert.equal(pending[0].messages[1].status, "pending");

  const [first, second] = attachCreationTaskContexts([
    task("t1", "succeeded", ["https://img/1"]),
    task("t2", "failed"),
  ], [source]);
  const done = reconcileCreationTaskMessages(runtime, [source], [first, second]);
  assert.equal(done[0].messages[1].status, "done");
  assert.deepEqual(done[0].messages[1].resultUrls, ["https://img/1"]);
  assert.equal(done[0].messages[1].content, "1 张图片已生成，1 张失败");

  const [failedFirst, failedSecond] = attachCreationTaskContexts([task("t1", "failed"), task("t2", "failed")], [source]);
  const failed = reconcileCreationTaskMessages(runtime, [source], [failedFirst, failedSecond]);
  assert.equal(failed[0].messages[1].status, "error");
});
