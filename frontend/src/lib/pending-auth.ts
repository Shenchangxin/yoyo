import * as api from "./client";
import type { Thread } from "./protocol";
import { useUI } from "./store";

/**
 * A permission picked before any chat existed is held in the store and applied
 * to the first chat created after it, so the selector never silently does nothing.
 * If the apply fails the caller shows the session's real mode, not the pending one.
 */
export async function adoptPendingAuthMode(t: Thread): Promise<Thread> {
  const mode = useUI.getState().pendingAuthMode;
  if (!mode) return t;
  useUI.getState().setPendingAuthMode(null);
  if (t.authMode === mode) return t;
  try {
    return await api.setSessionAuthMode(t.id, mode);
  } catch {
    return t;
  }
}
