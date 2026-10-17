import { gameStepOf, hasPlayable, qaStatus, type GameRow, type GameStep } from "./game-model";

export const GAME_PAGES: GameStep[] = ["intent", "build", "play", "check"];

export type GameDeskCopy = {
  stepIntent: string;
  stepBuild: string;
  stepPlay: string;
  stepCheck: string;
  stepIntentLead: string;
  stepBuildLead: string;
  stepPlayLead: string;
  stepCheckLead: string;
  askIntent: string;
  askBuild: string;
  askPlay: string;
  askCheck: string;
  askRepair: string;
  askWithGoal: string;
  copilotIntent: string;
  copilotBuild: string;
  copilotPlay: string;
  copilotCheck: string;
  statusDraft: string;
  statusScene: string;
  statusRules: string;
  statusPlaced: string;
  statusAssets: string;
  statusCompiled: string;
};

export function deskPage(row: GameRow | null, focus?: GameStep | ""): GameStep {
  if (focus === "intent" || focus === "build" || focus === "play" || focus === "check") return focus;
  return gameStepOf(row?.stage, row) || "intent";
}

export function pageComplete(step: GameStep, row: GameRow | null): boolean {
  if (!row) return false;
  if (step === "intent") return !!(row.intent || (row.stage && row.stage !== "draft" && row.stage !== "failed"));
  if (step === "build" || step === "play") return hasPlayable(row);
  return qaStatus(row) === "pass";
}

export function pageLabel(step: GameStep, g: Pick<GameDeskCopy, "stepIntent" | "stepBuild" | "stepPlay" | "stepCheck">): string {
  if (step === "intent") return g.stepIntent;
  if (step === "build") return g.stepBuild;
  if (step === "play") return g.stepPlay;
  return g.stepCheck;
}

export function pageLead(step: GameStep, g: GameDeskCopy): string {
  if (step === "intent") return g.stepIntentLead;
  if (step === "build") return g.stepBuildLead;
  if (step === "play") return g.stepPlayLead;
  return g.stepCheckLead;
}

export function copilotHint(step: GameStep, g: GameDeskCopy): string {
  if (step === "intent") return g.copilotIntent;
  if (step === "build") return g.copilotBuild;
  if (step === "play") return g.copilotPlay;
  return g.copilotCheck;
}

export function askAgentText(step: GameStep, row: GameRow | null, g: GameDeskCopy): string {
  const goal = (row?.intent || "").trim();
  if (step === "intent" || step === "build") {
    return goal ? g.askWithGoal.replace("{goal}", goal) : (step === "intent" ? g.askIntent : g.askBuild);
  }
  if (step === "play") return g.askPlay;
  if (qaStatus(row) === "fail") return g.askRepair;
  return g.askCheck;
}

export function askAgentLabel(step: GameStep, row: GameRow | null, g: GameDeskCopy): string {
  if (step === "intent") return g.askIntent;
  if (step === "build") return g.askBuild;
  if (step === "play") return g.askPlay;
  if (qaStatus(row) === "fail") return g.askRepair;
  return g.askCheck;
}

export function buildMarks(stage: string | undefined, g: GameDeskCopy): { key: string; label: string; done: boolean }[] {
  const rank = ["draft", "scene", "rules", "candidates", "placed", "assets", "compiled", "verified"].indexOf(stage || "draft");
  const n = rank < 0 ? 0 : rank;
  return [
    { key: "scene", label: g.statusScene, done: n >= 1 },
    { key: "rules", label: g.statusRules, done: n >= 2 },
    { key: "place", label: g.statusPlaced, done: n >= 4 },
    { key: "look", label: g.statusAssets, done: n >= 5 },
    { key: "play", label: g.statusCompiled, done: n >= 6 },
  ];
}

export function deskStarters(step: GameStep, row: GameRow | null, g: GameDeskCopy): { label: string; text: string }[] {
  return [
    { label: g.askBuild, text: askAgentText("build", row, g) },
    { label: g.askPlay, text: g.askPlay },
    { label: qaStatus(row) === "fail" ? g.askRepair : g.askCheck, text: askAgentText("check", row, g) },
  ].filter((item, i, all) => all.findIndex((x) => x.label === item.label) === i);
}
