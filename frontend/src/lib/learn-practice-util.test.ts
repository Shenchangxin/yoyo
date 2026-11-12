/** 锁住练习界面的纯函数：间隔的读法、多选作答的拼法、标签的拆法和柱高的换算。 */

import test from "node:test";
import assert from "node:assert/strict";
import { answerLabels, barHeights, intervalParts, newRequestId, objective, splitTags, toggleLabel } from "./learn-practice-util.ts";

test("间隔按大小换单位：十分钟、几小时、几天、几个月、几年", () => {
  assert.deepEqual(intervalParts(10 / 1440), { n: 10, unit: "minute" });
  assert.deepEqual(intervalParts(0), { n: 1, unit: "minute" });
  assert.deepEqual(intervalParts(0.5), { n: 12, unit: "hour" });
  assert.deepEqual(intervalParts(1), { n: 1, unit: "day" });
  assert.deepEqual(intervalParts(3.4), { n: 3, unit: "day" });
  assert.deepEqual(intervalParts(45), { n: 1.5, unit: "month" });
  assert.deepEqual(intervalParts(730), { n: 2, unit: "year" });
});

test("多选的作答按选项顺序拼，再点一次去掉", () => {
  const options = [{ label: "A" }, { label: "B" }, { label: "C" }];
  assert.equal(toggleLabel(options, "", "C"), "C");
  assert.equal(toggleLabel(options, "C", "A"), "A,C");
  assert.equal(toggleLabel(options, "A,C", "A"), "C");
  assert.deepEqual([...answerLabels(" A , C ")], ["A", "C"]);
  assert.equal(answerLabels("").size, 0);
});

test("标签按逗号、分号和空白拆，去掉井号和重复项", () => {
  assert.deepEqual(splitTags("#微积分, 导数；导数 Limits limits"), ["微积分", "导数", "Limits"]);
  assert.deepEqual(splitTags("  "), []);
});

test("四种客观题在本地批改，简答和开放题不是", () => {
  for (const t of ["choice", "multi", "true_false", "fill"]) assert.equal(objective(t), true);
  for (const t of ["short", "open", ""]) assert.equal(objective(t), false);
});

test("柱高按最大值换算，非零的值看得见，全零时不画", () => {
  assert.deepEqual(barHeights([0, 0]), [0, 0]);
  assert.deepEqual(barHeights([0, 1, 100]), [0, 4, 100]);
  assert.deepEqual(barHeights([5, 10]), [50, 100]);
});

test("幂等键每次都不一样", () => {
  assert.notEqual(newRequestId(), newRequestId());
  assert.match(newRequestId(), /^rev-/);
});
