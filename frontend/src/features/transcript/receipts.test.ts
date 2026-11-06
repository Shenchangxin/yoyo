import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { AgentPart } from "../../lib/transcript-layout.ts";
import type { Item } from "../../lib/protocol.ts";
import { memoryReceipts } from "./receipts.ts";

function call(id: string, name: string, args: Record<string, unknown>): Item {
  return { key: `c-${id}`, type: "tool_call", ts: "", text: "", payload: { id, name, arguments: JSON.stringify(args) } } as Item;
}

function result(id: string, content: string, extra: Record<string, unknown> = {}): Item {
  return { key: `r-${id}`, type: "tool_result", ts: "", text: "", payload: { id, content, ...extra } } as Item;
}

describe("memoryReceipts", () => {
  it("lists what was written to memory in this turn, including nested work parts", () => {
    const parts: AgentPart[] = [
      { key: "p", kind: "process", live: false, items: [call("1", "memory_write", { kind: "semantic", text: "报告先写结论" }), result("1", "remembered")] },
      {
        key: "w",
        kind: "work",
        live: false,
        folded: true,
        parts: [{ key: "p2", kind: "process", live: false, items: [call("2", "remember_fact", { text: "周三下午不开会" }), result("2", "ok")] }],
      },
    ];
    assert.deepEqual(memoryReceipts(parts), ["报告先写结论", "周三下午不开会"]);
  });

  it("skips refused, failed, unanswered and unrelated calls", () => {
    const parts: AgentPart[] = [
      {
        key: "p",
        kind: "process",
        live: false,
        items: [
          call("1", "memory_write", { text: "password: hunter22" }),
          result("1", "没有记：看起来像密钥"),
          call("2", "memory_write", { text: "失败的一条" }),
          result("2", "", { ok: false }),
          call("3", "memory_write", { text: "还没结果" }),
          call("4", "read_file", { path: "a.md" }),
          result("4", "body"),
        ],
      },
    ];
    assert.deepEqual(memoryReceipts(parts), []);
  });
});
