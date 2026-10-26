/** 拥有页面审阅卡片上的通过和退回。不拥有页面树。决定会改服务端。 */
import { useState } from "react";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { Button } from "../../components/ui/button";
import type { Item } from "../../lib/protocol";
import { toolName } from "../../lib/tool-summary";
import { parsePageReviewBlob } from "./review";

/** parsePageReview 从一条工具条目里读出审阅编号。 */
export function parsePageReview(item?: Item): { id: string; hash: string; title: string } | null {
  const nested = typeof item?.payload?.content === "string" ? item.payload.content : "";
  return parsePageReviewBlob(`${item?.text || ""} ${JSON.stringify(item?.payload || {})} ${nested}`);
}

/** isPageReviewItem 判断这条是不是页面审阅。 */
export function isPageReviewItem(item?: Item): boolean {
  if (!item) return false;
  if (toolName(item) === "review_page") return true;
  return /page\.save|review_page|awaiting page review/i.test(`${item.text || ""} ${JSON.stringify(item.payload || {})}`);
}

/** PageReviewCard 让人通过或退回一次页面保存。 */
export function PageReviewCard(props: { item?: Item; result?: Item; onDone?: () => void }) {
  const copy = useCopy();
  const parsed = parsePageReview(props.result || props.item);
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState("");
  const [err, setErr] = useState("");
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
              setErr("");
              try {
                await api.pagesDecide(parsed.id, parsed.hash, true);
                setDone(copy.pages.saved);
                props.onDone?.();
              } catch (e: any) {
                setErr(String(e?.message || e));
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
              setErr("");
              try {
                await api.pagesDecide(parsed.id, parsed.hash, false);
                setDone(copy.pages.declined);
                props.onDone?.();
              } catch (e: any) {
                setErr(String(e?.message || e));
              } finally {
                setBusy(false);
              }
            }}
          >
            {copy.pages.decline}
          </Button>
        </div>
      )}
      {err ? <div className="mt-2 text-[12px] text-danger">{err}</div> : null}
    </div>
  );
}
