import { projectShots, shotReady, type StudioGate, type StudioProject } from "./studio-model.ts";

export type StudioPane = "radar" | "topics" | "assemble" | "publish";

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

export type StudioNext = {
  key: StudioNextKey;
  pane: StudioPane;
  gate?: StudioGate;
};

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
