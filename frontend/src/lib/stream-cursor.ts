/**
 * Cursor rules for one session's live stream.
 *
 * Hub seqs are global across sessions, so the renderer keeps one cursor for the
 * whole stream: a notice from another session still moves it. The cursor only
 * moves past a seq once everything up to it has been applied or deliberately
 * skipped. A notice that jumps past cursor+1 means some earlier seq has not
 * arrived yet (events can also reach the renderer out of order), and the only
 * repair is a pull, because the ring is read back in seq order.
 */

export type LiveStep =
  /** Already covered by the cursor (a seed, a pull, or an earlier notice). */
  | "stale"
  /** Contiguous, belongs to another session: move the cursor and do nothing. */
  | "advance"
  /** Contiguous, ours, carries the event inline: apply it and move the cursor. */
  | "apply"
  /** Cannot be applied in order from the notice alone: pull the ring tail. */
  | "pull";

export function classifyLive(cursor: number, seq: number, own: boolean, inline: boolean): LiveStep {
  if (seq > 0 && seq <= cursor) return "stale";
  if (seq > 0 && seq > cursor + 1) return "pull";
  if (!own) return "advance";
  if (!inline) return "pull";
  return "apply";
}
