import { useEffect } from "react";
import { useUI } from "../../lib/store";
import { useGameSelection } from "../video/workshop-store";

/** Bind a workshop when the host publishes video_route (agent entry). */
export function useVideoRoute() {
  useEffect(() => {
    let off: (() => void) | undefined;
    void import("@wailsio/runtime").then((mod: any) => {
      const Events = mod.Events;
      if (!Events?.On) return;
      off = Events.On("yoyo:item", (ev: any) => {
        const kind = String(ev?.item_kind || ev?.ItemKind || "");
        const payload = ev?.payload || ev?.Payload || {};
        if (kind !== "video_route" && kind !== "game_sync") return;
        const pane = String(payload.pane || payload.dest || "");
        const id = String(payload.id || "").trim();
        if (pane !== "game" || !id) return;
        const cur = useGameSelection.getState().gameId;
        if (kind === "video_route" && (!cur || cur === id)) {
          useGameSelection.getState().setGameId(id);
          useUI.getState().openVideoPane("game");
        }
        useGameSelection.getState().touch(id);
      });
    }).catch(() => {});
    return () => { off?.(); };
  }, []);
}
