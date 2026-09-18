import type { ReactNode } from "react";
import { cn } from "../../lib/utils";

export function SettingRow({
  title,
  description,
  children,
  border = true,
}: {
  title: string;
  description?: string;
  children: ReactNode;
  border?: boolean;
}) {
  return (
    <div className={cn("flex items-center justify-between gap-6 px-5 py-3.5", border && "border-b border-border/80")}>
      <div className="min-w-0 flex-1 basis-0">
        <div className="truncate text-[13px] font-medium text-foreground" title={title}>
          {title}
        </div>
        {description ? (
          <div className="mt-0.5 line-clamp-2 text-[12px] text-muted" title={description}>
            {description}
          </div>
        ) : null}
      </div>
      <div className="max-w-[min(100%,24rem)] shrink-0">{children}</div>
    </div>
  );
}

export function SettingSection({
  id,
  title,
  description,
  children,
}: {
  id: string;
  title?: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <div className="mb-6 p-1.5" data-setting-section-highlight-target={id}>
      {title ? (
        <h2 id={id} className={cn("text-[15px] font-semibold text-foreground", description ? "mb-1" : "mb-3")}>
          {title}
        </h2>
      ) : null}
      {description ? <p className="mb-3 text-[12px] text-muted">{description}</p> : null}
      <div
        id={title ? undefined : id}
        className="surface-inset overflow-hidden rounded-xl border border-border/80 bg-card"
        data-setting-section-id={id}
      >
        {children}
      </div>
    </div>
  );
}
