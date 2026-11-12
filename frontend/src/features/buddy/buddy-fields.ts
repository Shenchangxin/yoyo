/**
 * 拥有：Buddy 表单里和画法无关的那部分：还没保存的密钥草稿，以及把草稿变成 buddy.configure 的参数。
 * 不拥有：任何组件，也不保存密钥值。输入框里的字保存成功后由表单清空，这里只在内存里过一下。
 */

import type { BuddyConfigPatch, BuddySecret } from "../../lib/client.ts";

/** DEFAULT_MODEL 是「用它自己的默认模型」在下拉里的值。Radix 的选项不能用空串。 */
export const DEFAULT_MODEL = "__default";

/** KeyDraft 是还没保存的密钥输入：values 是新填的值，cleared 是要删掉的，reuse 是复用开关。 */
export type KeyDraft = { values: Record<string, string>; cleared: Record<string, boolean>; reuse: Record<string, boolean> };

/** emptyDraft 按当前视图起一份草稿：没有新值、没有要删的，复用开关照现状。 */
export function emptyDraft(secrets: BuddySecret[]): KeyDraft {
  return { values: {}, cleared: {}, reuse: Object.fromEntries(secrets.map((s) => [s.name, !!s.reuse])) };
}

/** draftPatch 把草稿变成 buddy.configure 的 secrets 和 reuse 两项。没有改动的项不放进去。 */
export function draftPatch(secrets: BuddySecret[], draft: KeyDraft): Pick<BuddyConfigPatch, "secrets" | "reuse"> {
  const out: Pick<BuddyConfigPatch, "secrets" | "reuse"> = {};
  const keys: Record<string, string> = {};
  for (const s of secrets) {
    const value = (draft.values[s.name] ?? "").trim();
    if (value) keys[s.name] = value;
    else if (draft.cleared[s.name] && s.hasKey) keys[s.name] = "";
  }
  if (Object.keys(keys).length) out.secrets = keys;
  if (secrets.some((s) => s.reusable && !!draft.reuse[s.name] !== !!s.reuse)) {
    out.reuse = secrets.filter((s) => s.reusable && draft.reuse[s.name]).map((s) => s.name);
  }
  return out;
}
