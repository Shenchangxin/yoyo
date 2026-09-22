import * as api from "../../lib/client";
import { useDramaSelection } from "./workshop-store";

/** Bind this video conversation to the workshop episode, if one is selected.
 *  Does not create a series — the project (drama/episode) is independent of
 *  the chat, same as LibTV session vs projectUuid. */
export async function bindVideoThread(sessionId: string): Promise<void> {
  const ep = useDramaSelection.getState().episodeId;
  if (!sessionId || !ep) return;
  await api.video.bind(sessionId, ep);
}
