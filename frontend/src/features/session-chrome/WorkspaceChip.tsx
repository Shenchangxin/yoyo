/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 *
 * Internalized from deepseek-harness EmptyHero WorkspaceChip.
 * Folder + label + chevron; placeholder when no workspace is bound.
 */
import { forwardRef } from "react";
import { ChevronDown, Folder, FolderOpen } from "lucide-react";

export function workspaceLabel(cwd: string): string {
  const parts = cwd.replace(/\\/g, "/").split("/").filter(Boolean);
  const base = parts[parts.length - 1] || "";
  return base || cwd;
}

export const WorkspaceChip = forwardRef<HTMLButtonElement, {
  label?: string;
  menuOpen?: boolean;
  disabled?: boolean;
  chooseLabel: string;
  onClick?: () => void;
}>(function WorkspaceChip(props, ref) {
  return (
    <button
      ref={ref}
      type="button"
      className="sc-ws-chip"
      data-testid="workspace-chip"
      aria-label={props.chooseLabel}
      aria-haspopup="menu"
      aria-expanded={props.menuOpen}
      disabled={props.disabled}
      onClick={props.onClick}
    >
      {props.label
        ? <FolderOpen className="sc-ws-folder" size={16} strokeWidth={1} />
        : <Folder className="sc-ws-folder" size={16} strokeWidth={1} />}
      <span className="sc-ws-label">{props.label ?? props.chooseLabel}</span>
      <ChevronDown className="sc-ws-chevron" size={12} strokeWidth={1} />
    </button>
  );
});
