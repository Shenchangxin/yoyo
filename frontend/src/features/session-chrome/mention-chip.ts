/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 *
 * Inline mention/command chips, internalized from harness decorator chips.
 * Implemented as a TipTap atom so caret geometry stays native.
 */
import { Node, mergeAttributes } from "@tiptap/core";
import type { MentionKind, MentionPin } from "../../lib/mentions";

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    mentionChip: {
      insertMentionChip: (pin: MentionPin) => ReturnType;
    };
  }
}

export const MentionChip = Node.create({
  name: "mentionChip",
  group: "inline",
  inline: true,
  atom: true,
  selectable: true,
  addAttributes() {
    return {
      token: { default: "" },
      label: { default: "" },
      kind: { default: "file" as MentionKind },
    };
  },
  parseHTML() {
    return [{ tag: "span[data-mention-chip]" }];
  },
  renderHTML({ HTMLAttributes }) {
    const token = String(HTMLAttributes.token || "");
    return [
      "span",
      mergeAttributes({
        "data-mention-chip": "",
        "data-testid": "mention-chip",
        "data-kind": HTMLAttributes.kind,
        class: "sc-mention",
        "data-token": token,
        "aria-label": token,
      }),
      String(HTMLAttributes.label || token),
    ];
  },
  addCommands() {
    return {
      insertMentionChip: (pin: MentionPin) => ({ chain }) =>
        chain()
          .insertContent({
            type: this.name,
            attrs: { token: pin.token, label: pin.label, kind: pin.kind },
          })
          .insertContent(" ")
          .run(),
    };
  },
});
