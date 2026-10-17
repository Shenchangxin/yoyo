import * as api from "../../lib/client";
import { useUI } from "../../lib/store";
import { useGameSelection } from "../video/workshop-store";
import type { GameRow } from "./game-model";

export function titleFromIntent(text: string, fallback: string): string {
  const line = (text || "").replace(/\s+/g, " ").trim();
  if (!line) return fallback;
  return line.length > 36 ? `${line.slice(0, 36).trim()}…` : line;
}

export async function createGameWorld(input: {
  title?: string;
  intent?: string;
  camera?: string;
} = {}): Promise<GameRow> {
  const created = await api.video.createGame({
    title: input.title || "",
    intent: input.intent || "",
    camera: input.camera || "",
  }) as GameRow;
  if (!created?.id) {
    throw new Error("create returned no id");
  }
  useGameSelection.getState().setGameId(created.id, created.title || input.title || "");
  useUI.getState().openVideoPane("game");
  return created;
}
