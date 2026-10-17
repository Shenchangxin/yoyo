/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 *
 * Internalized from harness PermissionSelect. Presets map onto Yoyo AuthMode.
 */
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import type { AuthMode } from "../../lib/protocol";
import { useCopy } from "../../lib/i18n";

export function parseComposerAuth(mode?: string): AuthMode {
  const s = (mode || "").toLowerCase().replace(/[-\s]/g, "_");
  if (s === "ask" || s === "ask_every_time" || s === "strict") return "ask";
  if (s === "auto_edit" || s === "autoedit") return "auto_edit";
  if (s === "full" || s === "full_access") return "full";
  return "default";
}

export function PermissionSelect(props: {
  value?: AuthMode | string;
  disabled?: boolean;
  testId?: string;
  onChange: (mode: AuthMode) => void;
}) {
  const copy = useCopy();
  const current = parseComposerAuth(props.value);
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
      <SelectTrigger className="sc-chip-btn max-w-[7.5rem] border-0 shadow-none" aria-label={copy.composer.authMode} data-testid={props.testId}>
        <SelectValue />
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
