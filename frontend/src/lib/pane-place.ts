/**
 * 拥有：这一次打开里，今天、任务、记忆、Buddy 停在哪一页、各自滚到多深。
 * 不拥有：滚动条怎么画。应用重启后这些都丢，四个板块回到顶部。
 * 位置从内容顶部量起。没有记录就是 0。
 */

export type PaneTab = "now" | "tasks" | "memory" | "buddy";

const tabs: PaneTab[] = ["now", "tasks", "memory", "buddy"];

type PaneRoute = { tab: PaneTab; groupId: string };

let route: PaneRoute = { tab: "now", groupId: "" };
const tops = new Map<string, number>();

/** readPaneRoute 读这次打开里留下的页签和任务组。 */
export function readPaneRoute(): PaneRoute {
  return { tab: route.tab, groupId: route.groupId };
}

/** writePaneRoute 记下页签。不认识的页签不写。任务组只在任务页有意义，别的页签也留着，回来还能进同一个组。 */
export function writePaneRoute(tab: string, groupId: string): void {
  if (!tabs.includes(tab as PaneTab)) return;
  route = { tab: tab as PaneTab, groupId: groupId || "" };
}

/** readPaneTop 读这一屏上次的 scrollTop。没有就是顶部。 */
export function readPaneTop(key: string): number {
  if (!key) return 0;
  return tops.get(key) ?? 0;
}

/** writePaneTop 记下从顶部量起的位置。0 或无效就忘掉，下次自然在顶部。 */
export function writePaneTop(key: string, top: number): void {
  if (!key) return;
  const n = Number(top);
  if (!Number.isFinite(n) || n <= 0) {
    tops.delete(key);
    return;
  }
  tops.set(key, n);
}

/** resetPanePlace 清掉页签和位置。测试之间互不影响。 */
export function resetPanePlace(): void {
  route = { tab: "now", groupId: "" };
  tops.clear();
}

/** buddyScrollKey 是 Buddy 里当前那一屏。名单、详情、偏好、自定义和日志各记各的，都从顶部量。 */
export function buddyScrollKey(dialog: { kind: string; id?: string } | null, focus: string): string {
  if (dialog?.kind === "prefs") return "buddy:prefs";
  if (dialog?.kind === "custom") return "buddy:custom";
  if (dialog?.kind === "log" && dialog.id) return `buddy:log:${dialog.id}`;
  if (focus) return `buddy:agent:${focus}`;
  return "buddy";
}
