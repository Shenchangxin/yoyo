/**
 * 拥有：「Yoyo 在跟的事」页上给人看的说法：发送审阅还能等到几点、一件事现在怎样了。不露内部种类、哈希和状态枚举。
 * 不拥有：这些事本身（个人引擎）和页面布局（PersonalPanel）。
 */
import { getLocale } from "../../lib/i18n";

function zh(): boolean {
  return String(getLocale()).startsWith("zh");
}

/** proposalWhen 是一条发送审阅的说明：等你点头，到几点为止。时间认不出时只说在等你。 */
export function proposalWhen(at: string): string {
  const d = at ? new Date(at) : null;
  if (!d || Number.isNaN(d.getTime())) return zh() ? "等你点头" : "Waiting for your approval";
  const hm = `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
  return zh() ? `等你点头，${hm} 前有效` : `Waiting for your approval, valid until ${hm}`;
}

const STATUS: Record<string, [string, string]> = {
  queued: ["排队中", "Queued"],
  running: ["正在做", "Working on it"],
  waiting_input: ["在等你回答", "Waiting for your answer"],
  waiting_approval: ["在等你点头", "Waiting for your approval"],
  paused: ["暂停了", "Paused"],
  failed: ["没做成", "Didn't finish"],
  outcome_unknown: ["不确定有没有做成，去看一眼", "Not sure it went through, take a look"],
  cancelled: ["已取消", "Cancelled"],
  succeeded: ["做完了", "Done"],
};

/** taskStatusLabel 是一件事现在怎样了。认不出的状态原样给出。 */
export function taskStatusLabel(status: string): string {
  const row = STATUS[(status || "").trim()];
  if (!row) return status;
  return zh() ? row[0] : row[1];
}
