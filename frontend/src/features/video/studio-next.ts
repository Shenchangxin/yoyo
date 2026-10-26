/**
 * 本文件拥有短片工作台下一步该去哪一栏的纯判断，以及按钮文案的取词。
 * 不拥有工作台界面。它不请求服务端，也不改服务端状态。
 */
import { projectShots, shotReady, type StudioGate, type StudioProject } from "./studio-model.ts";

/** StudioPane 是雷达、选题、制作、发布四栏。 */
export type StudioPane = "radar" | "topics" | "assemble" | "publish";

/** StudioNextKey 是下一步按钮可能用到的动作名。 */
export type StudioNextKey =
  | "collect"
  | "produce"
  | "setup"
  | "script"
  | "audio"
  | "visuals"
  | "compose"
  | "pack"
  | "open"
  | "watch";

/** StudioNext 是下一步的动作名、要去的栏，以及可选的工序。 */
export type StudioNext = {
  key: StudioNextKey;
  pane: StudioPane;
  gate?: StudioGate;
};

/** studioNext 按当前短片和所在栏决定下一步。没绑短片时雷达去收集，其他栏去选题开拍。current 没有时按栏返回收集或开拍，有则按阶段、成片、剧本、声音和镜头决定下一步。pane 只在没有短片时区分雷达和其他栏。 */
export function studioNext(current: StudioProject | null | undefined, pane: StudioPane): StudioNext {
  if (!current) {
    if (pane === "topics" || pane === "assemble" || pane === "publish") return { key: "produce", pane: "topics" };
    return { key: "collect", pane: "radar" };
  }
  const phase = current.auto_phase || "";
  if (phase === "queued" || phase === "script" || phase === "voice" || phase === "visuals" || phase === "compose" || (phase === "failed" && !current.result_hash)) {
    return { key: "watch", pane: "publish" };
  }
  if (current.result_hash) {
    if (current.delivery?.status !== "publishable") return { key: "pack", pane: "publish" };
    return { key: "open", pane: "publish" };
  }
  const script = String(current.script || "").trim();
  const hasLook = !!(current.identity?.character_id || current.template_id || current.style);
  if ((current.gate === "setup" || current.furthest_gate === "setup") && !hasLook) {
    return { key: "setup", pane: "assemble", gate: "setup" };
  }
  if (current.script_status !== "approved") {
    return { key: "script", pane: "assemble", gate: "script" };
  }
  if (!script) {
    return { key: "script", pane: "assemble", gate: "script" };
  }
  if (!current.skip_audio && !current.narration_hash) {
    return { key: "audio", pane: "assemble", gate: "audio" };
  }
  if (!projectShots(current).some(shotReady)) {
    return { key: "visuals", pane: "assemble", gate: "visuals" };
  }
  return { key: "compose", pane: "assemble", gate: "compose" };
}

type NextCopy = {
  nextCollect: string;
  nextProduce: string;
  nextSetup: string;
  nextScript: string;
  nextAudio: string;
  nextVisuals: string;
  nextCompose: string;
  nextPack: string;
  nextOpen: string;
  nextWatch: string;
  promptCollect: string;
  promptProduce: string;
  promptSetup: string;
  promptScript: string;
  promptAudio: string;
  promptVisuals: string;
  promptCompose: string;
  promptPack: string;
  promptOpen: string;
  promptWatch: string;
};

/** studioNextCopy 按动作名取下一步按钮的短文案。s 按动作名取 nextCollect 到 nextWatch。key 选择其中一项。 */
export function studioNextCopy(s: NextCopy, key: StudioNextKey) {
  const map: Record<StudioNextKey, string> = {
    collect: s.nextCollect,
    produce: s.nextProduce,
    setup: s.nextSetup,
    script: s.nextScript,
    audio: s.nextAudio,
    visuals: s.nextVisuals,
    compose: s.nextCompose,
    pack: s.nextPack,
    open: s.nextOpen,
    watch: s.nextWatch,
  };
  return map[key];
}

/** studioNextPrompt 按动作名取下一步要填进输入框的长文案。s 按动作名取 promptCollect 到 promptWatch。key 选择其中一项。 */
export function studioNextPrompt(s: NextCopy, key: StudioNextKey) {
  const map: Record<StudioNextKey, string> = {
    collect: s.promptCollect,
    produce: s.promptProduce,
    setup: s.promptSetup,
    script: s.promptScript,
    audio: s.promptAudio,
    visuals: s.promptVisuals,
    compose: s.promptCompose,
    pack: s.promptPack,
    open: s.promptOpen,
    watch: s.promptWatch,
  };
  return map[key];
}
