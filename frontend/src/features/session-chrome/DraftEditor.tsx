/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 *
 * Internalized DraftEditor surface (scrollport + placeholder + chips).
 * Lexical is replaced with TipTap, which Yoyo already vendors.
 */
import { forwardRef, useEffect, useImperativeHandle, useRef, type KeyboardEvent as ReactKeyboardEvent } from "react";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import type { Editor } from "@tiptap/core";
import { MentionChip } from "./mention-chip";
import { pinFromToken, type MentionPin } from "../../lib/mentions";
import { cn } from "../../lib/utils";

export type DraftSnapshot = { text: string; chips: MentionPin[]; caret: number; before: string };

export type DraftEditorHandle = {
  insertChip: (pin: MentionPin) => void;
  insertText: (text: string) => void;
  setPlain: (text: string) => void;
  focus: () => void;
  snapshot: () => DraftSnapshot;
  replaceMention: (pin: MentionPin) => void;
};

function snapshotOf(editor: Editor | null): DraftSnapshot {
  if (!editor) return { text: "", chips: [], caret: 0, before: "" };
  const chips: MentionPin[] = [];
  const parts: string[] = [];
  editor.state.doc.forEach((block, _offset, index) => {
    if (index > 0) parts.push("\n");
    block.forEach((node) => {
      if (node.type.name === "mentionChip") {
        chips.push(pinFromToken(String(node.attrs.token || "")));
        return;
      }
      if (node.isText) parts.push(node.text || "");
      if (node.type.name === "hardBreak") parts.push("\n");
    });
  });
  const caret = editor.state.selection.from;
  return { text: parts.join(""), chips, caret, before: textBeforeCaret(editor) };
}

function plainToContent(text: string) {
  const lines = text.split("\n");
  return {
    type: "doc" as const,
    content: lines.map((line) => ({
      type: "paragraph" as const,
      content: line ? [{ type: "text" as const, text: line }] : [],
    })),
  };
}

function textBeforeCaret(editor: Editor): string {
  const { from } = editor.state.selection;
  return editor.state.doc.textBetween(0, from, "\n", " ");
}

export const DraftEditor = forwardRef<DraftEditorHandle, {
  draftKey: string;
  disabled?: boolean;
  placeholder: string;
  ariaLabel: string;
  onSnapshot: (snap: DraftSnapshot) => void;
  onKeyDown?: (e: ReactKeyboardEvent, snap: DraftSnapshot, before: string) => boolean;
  onPasteFiles?: (files: File[]) => void;
  testId?: string;
  workspaceTrigger?: boolean;
  pickerOpen?: boolean;
}>(function DraftEditor(props, ref) {
  const placeholderRef = useRef(props.placeholder);
  placeholderRef.current = props.placeholder;
  const skipUpdate = useRef(false);
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: false,
        blockquote: false,
        codeBlock: false,
        horizontalRule: false,
        bulletList: false,
        orderedList: false,
        listItem: false,
        bold: false,
        italic: false,
        strike: false,
        code: false,
      }),
      Placeholder.configure({ placeholder: () => placeholderRef.current }),
      MentionChip,
    ],
    editable: !props.disabled,
    editorProps: {
      attributes: editorAttrs(props),
      handlePaste: (_view, event) => {
        const files = filesFromClipboard(event);
        if (files.length && props.onPasteFiles) {
          props.onPasteFiles(files);
          return true;
        }
        return false;
      },
    },
    onUpdate: ({ editor: ed }) => {
      if (skipUpdate.current) return;
      props.onSnapshot(snapshotOf(ed));
    },
  });

  useEffect(() => {
    if (!editor) return;
    skipUpdate.current = true;
    editor.commands.clearContent();
    skipUpdate.current = false;
  }, [props.draftKey, editor]);

  useEffect(() => {
    editor?.setEditable(!props.disabled);
  }, [editor, props.disabled]);

  useEffect(() => {
    editor?.setOptions({
      editorProps: {
        ...editor.options.editorProps,
        attributes: editorAttrs(props),
      },
    });
  }, [editor, props.ariaLabel, props.testId, props.workspaceTrigger, props.pickerOpen]);

  useImperativeHandle(ref, () => ({
    insertChip: (pin) => {
      editor?.chain().focus().insertMentionChip(pin).run();
    },
    insertText: (text) => {
      editor?.chain().focus().insertContent(text).run();
    },
    setPlain: (text) => {
      skipUpdate.current = true;
      editor?.commands.setContent(plainToContent(text));
      skipUpdate.current = false;
    },
    focus: () => {
      editor?.commands.focus("end");
    },
    snapshot: () => snapshotOf(editor),
    replaceMention: (pin) => {
      if (!editor) return;
      const before = textBeforeCaret(editor);
      const m = before.match(/(?:^|\s)(@(?:file:|folder:|skill:)?[^\s]*)$/);
      if (!m) {
        editor.chain().focus().insertMentionChip(pin).run();
        return;
      }
      const token = m[1];
      const from = editor.state.selection.from - token.length;
      const to = editor.state.selection.from;
      editor.chain().focus().deleteRange({ from, to }).insertMentionChip(pin).run();
    },
  }), [editor]);

  return (
    <div
      className={cn("sc-scroll")}
      data-input-scroll
      onKeyDown={(e) => {
        if (!props.onKeyDown || !editor) return;
        const handled = props.onKeyDown(e, snapshotOf(editor), textBeforeCaret(editor));
        if (handled) e.preventDefault();
      }}
    >
      <div className="sc-grow">
        <EditorContent editor={editor} />
      </div>
    </div>
  );
});

function editorAttrs(props: {
  ariaLabel: string;
  testId?: string;
  workspaceTrigger?: boolean;
  pickerOpen?: boolean;
}): Record<string, string> {
  const attrs: Record<string, string> = {
    class: "sc-input",
    "aria-label": props.ariaLabel,
    "data-testid": props.testId || "composer-input",
  };
  if (props.workspaceTrigger) {
    attrs["aria-haspopup"] = "dialog";
    attrs["aria-expanded"] = props.pickerOpen ? "true" : "false";
  }
  return attrs;
}

function filesFromClipboard(e: ClipboardEvent): File[] {
  const files: File[] = [];
  const items = e.clipboardData?.items;
  if (items) {
    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      if (item.kind === "file" || item.type.startsWith("image/")) {
        const f = item.getAsFile();
        if (f) files.push(f);
      }
    }
  }
  return files;
}
