/**
 * 拥有：简报里记忆整理建议卡的「对」「不对」：直接交给 memoryTidy，不放进输入框；办成了告诉人一声，没办成说清原因。
 * 不拥有：建议从哪来、点了「对」怎么并（Go 的 memory/tidy.go、app/brief_tidy.go）；卡怎么画（WorkPane）。
 */

import { toast } from "sonner";
import { memoryTidy } from "../../lib/client";
import type { BriefCard } from "./work-model.ts";

/** isTidyCard 在这张卡是记忆整理建议卡时为真。 */
export function isTidyCard(card: BriefCard): boolean {
  return !!(card.tidy || "").trim();
}

/** tidyChinese 看卡上的按钮字：卡是按写简报时的语言写的，提示跟着卡走。 */
function tidyChinese(card: BriefCard): boolean {
  return /[\u4e00-\u9fff]/.test(`${card.primary}${card.title}`);
}

/** settleTidy 处理一张整理卡：primary 是「对」，secondary 是「不对」。办成了返回真。 */
export async function settleTidy(card: BriefCard, which: "primary" | "secondary"): Promise<boolean> {
  const id = (card.tidy || "").trim();
  if (!id) return false;
  const accept = which === "primary";
  const zh = tidyChinese(card);
  try {
    const got = await memoryTidy(id, accept);
    if (!got.ok) {
      const why = got.error.replace(/^memory:\s*/, "").trim();
      toast(why || (zh ? "这条建议没办成。" : "That suggestion could not be applied."));
      return false;
    }
    if (accept) toast(zh ? "已经并成一条。记忆页里可以撤销。" : "Merged. You can undo it on the Memory page.");
    else toast(zh ? "好的，以后不再问这两条。" : "Got it. These two will not be asked again.");
    return true;
  } catch (e) {
    toast(e instanceof Error ? e.message : String(e || ""));
    return false;
  }
}
