import assert from "node:assert/strict";
import test from "node:test";

// @ts-expect-error -- Node 原生 TypeScript 测试运行器需要保留扩展名。
import { createCreationSubmitGate } from "./creation-submit-gate.ts";

test("一次提交的收尾不会放开下一次提交的锁", () => {
  const gate = createCreationSubmitGate();
  const first = gate.tryAcquire();
  assert.ok(first);
  assert.equal(gate.tryAcquire(), 0);

  gate.release(first);
  const second = gate.tryAcquire();
  assert.ok(second);
  assert.notEqual(second, first);
  gate.release(first);
  assert.equal(gate.tryAcquire(), 0);

  gate.release(second);
  assert.ok(gate.tryAcquire());
});
