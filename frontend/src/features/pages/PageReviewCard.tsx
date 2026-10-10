import { useState } from "react";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { Button } from "../../components/ui/button";
import type { Item } from "../../lib/protocol";
import { toolName } from "../../lib/tool-summary";

function parseReview(item?: Item): { id: string; hash: string; title: string } | null {
  const blob = `${item?.text || ""} ${JSON.stringify(item?.payload || {})}`;
  const m = blob.match(/awaiting page review\s*(\{[\s\S]*\})/i) || blob.match(/(\{"review_id"[\s\S]*\})/);
  let raw = m?.[1] || "";
  if (!raw) {
    const p = item?.payload || {};
    const nested = typeof p.content === "string" ? p.content : "";
    const n = nested.match(/(\{[\s\S]*"kind"\s*:\s*"page\.save"[\s\S]*\})/);
    raw = n?.[1] || "";
  }
  if (!raw) return null;
  try {
    const j = JSON.parse(raw);
    return { id: String(j.review_id || j.id || ""), hash: String(j.hash || ""), title: String(j.title || "Page") };
  } catch {
    return null;
  }
}

export function isPageReviewItem(item?: Item): boolean {
  if (!item) return false;
  if (toolName(item) === "review_page") return true;
  return /page\.save|review_page|awaiting page review/i.test(`${item.text || ""} ${JSON.stringify(item.payload || {})}`);
}

export function PageReviewCard(props: { item?: Item; result?: Item; onDone?: () => void }) {
  const copy = useCopy();
  const parsed = parseReview(props.result || props.item);
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState("");
  if (!parsed?.id) return null;
  return (
    <div className="rounded-xl border border-border/70 bg-lift/40 p-3" data-testid="page-review-card">
      <div className="text-[13px] font-medium">{copy.pages.reviewTitle}: {parsed.title}</div>
      <p className="mt-1 text-[12px] text-muted">{copy.pages.reviewHint}</p>
      {done ? (
        <div className="mt-2 text-[12px] text-muted">{done}</div>
      ) : (
        <div className="mt-2 flex gap-2">
          <Button
            size="sm"
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              try {
                await api.pagesDecide(parsed.id, parsed.hash, true);
                setDone(copy.pages.saved);
                props.onDone?.();
              } catch (e: any) {
                setDone(String(e?.message || e));
              } finally {
                setBusy(false);
              }
            }}
          >
            {copy.pages.approveSave}
          </Button>
          <Button
            size="sm"
            variant="ghost"
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              try {
                await api.pagesDecide(parsed.id, parsed.hash, false);
                setDone(copy.pages.declined);
                props.onDone?.();
              } catch (e: any) {
                setDone(String(e?.message || e));
              } finally {
                setBusy(false);
              }
            }}
          >
            {copy.pages.decline}
          </Button>
        </div>
      )}
    </div>
  );
}
