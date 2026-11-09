import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { en } from "./copy.ts";
import { zhCN } from "./copy-zh.ts";

/**
 * 拥有：中英文案表的键必须一一对应。中文表用 as unknown as Copy 绕过了类型检查，少写一个键编译期发现不了，界面上就会冒出 undefined。
 * 不拥有：文案写得好不好。
 */

/** leafKeys 把嵌套的文案表摊成「a.b.c」形式的叶子键。 */
function leafKeys(table: object, prefix = "", out: string[] = []): string[] {
  for (const [key, value] of Object.entries(table)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (value && typeof value === "object" && !Array.isArray(value)) leafKeys(value, path, out);
    else out.push(path);
  }
  return out;
}

describe("copy catalogs", () => {
  /** 英文有的键中文都有，中文也没有多出来的键。 */
  it("have the same keys in English and Chinese", () => {
    const enKeys = leafKeys(en);
    const zhKeys = new Set(leafKeys(zhCN));
    assert.deepEqual(enKeys.filter((k) => !zhKeys.has(k)), []);
    const enSet = new Set(enKeys);
    assert.deepEqual([...zhKeys].filter((k) => !enSet.has(k)), []);
  });
});
