/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 *
 * Internalized from harness PermissionSelect. Presets map onto Yoyo AuthMode.
 */
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import type { AuthMode } from "../../lib/protocol";
import type { Copy } from "../../lib/copy";
import { useCopy } from "../../lib/i18n";

export function parseComposerAuth(mode?: string): AuthMode {
  const s = (mode || "").toLowerCase().replace(/[-\s]/g, "_");
  if (s === "ask" || s === "ask_every_time" || s === "strict") return "ask";
  if (s === "auto_edit" || s === "autoedit") return "auto_edit";
  if (s === "full" || s === "full_access") return "full";
  return "default";
}

function authLabel(copy: Copy["composer"], mode: AuthMode): string {
  if (mode === "auto_edit") return copy.authAutoEdit;
  if (mode === "full") return copy.authFull;
  if (mode === "ask") return copy.authAsk;
  return copy.authDefault;
}

export function PermissionSelect(props: {
  value?: AuthMode | string;
  disabled?: boolean;
  testId?: string;
  onChange: (mode: AuthMode) => void;
}) {
  const copy = useCopy();
  const current = parseComposerAuth(props.value);
  const label = authLabel(copy.composer, current);
  return (
    <Select
      value={current}
      onValueChange={(v) => {
        const next = v as AuthMode;
        if (next === "full" && current !== "full") {
          if (!window.confirm(copy.composer.authFullHint)) return;
        }
        props.onChange(next);
      }}
      disabled={props.disabled}
    >
      <SelectTrigger
        className="sc-chip-btn sc-perm h-7 min-w-0 w-auto max-w-[6.75rem] justify-start gap-1 border-0 px-2 shadow-none"
        aria-label={copy.composer.authMode}
        title={label}
        data-testid={props.testId}
      >
        <span className="sc-perm-label">
          <SelectValue />
        </span>
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="default">{copy.composer.authDefault}</SelectItem>
        <SelectItem value="auto_edit">{copy.composer.authAutoEdit}</SelectItem>
        <SelectItem value="full">{copy.composer.authFull}</SelectItem>
        <SelectItem value="ask">{copy.composer.authAsk}</SelectItem>
      </SelectContent>
    </Select>
  );
}
