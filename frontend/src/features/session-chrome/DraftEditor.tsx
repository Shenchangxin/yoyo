/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 *
 * Internalized DraftEditor surface (scrollport + placeholder + chips).
 * Lexical is replaced with TipTap, which Yoyo already vendors.
 */
import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import type { Editor } from "@tiptap/core";
import { NodeSelection } from "@tiptap/pm/state";
import { MentionChip } from "./mention-chip";
import { activeAtTail, pinFromToken, type MentionPin } from "../../lib/mentions";
import { cn } from "../../lib/utils";

export type DraftSnapshot = { text: string; chips: MentionPin[]; caret: number; before: string };

export type DraftEditorHandle = {
  insertChip: (pin: MentionPin) => void;
  insertText: (text: string) => void;
  setPlain: (text: string) => void;
  focus: () => void;
  snapshot: () => DraftSnapshot;
  replaceMention: (pin: MentionPin) => void;
  replaceTrigger: (text: string) => void;
  replaceSlash: () => void;
};

function liveEditor(editor: Editor | null): editor is Editor {
  return !!editor && !editor.isDestroyed;
}

function snapshotOf(editor: Editor | null): DraftSnapshot {
  if (!liveEditor(editor)) return { text: "", chips: [], caret: 0, before: "" };
  const chips: MentionPin[] = [];
  const parts: string[] = [];
  editor.state.doc.forEach((block, _offset, index) => {
    if (index > 0) parts.push("\n");
    block.forEach((node) => {
      if (node.type.name === "mentionChip") {
        const token = String(node.attrs.token || "");
        const pin = pinFromToken(token);
        const label = String(node.attrs.label || pin.label);
        chips.push({ ...pin, label });
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
  onCaret?: (snap: DraftSnapshot) => void;
  onKeyDown?: (e: KeyboardEvent, snap: DraftSnapshot, before: string) => boolean;
  onPasteFiles?: (files: File[]) => void;
  testId?: string;
  workspaceTrigger?: boolean;
  pickerOpen?: boolean;
}>(function DraftEditor(props, ref) {
  const placeholderRef = useRef(props.placeholder);
  placeholderRef.current = props.placeholder;
  const onSnapshotRef = useRef(props.onSnapshot);
  onSnapshotRef.current = props.onSnapshot;
  const onCaretRef = useRef(props.onCaret);
  onCaretRef.current = props.onCaret;
  const onPasteRef = useRef(props.onPasteFiles);
  onPasteRef.current = props.onPasteFiles;
  const skipUpdate = useRef(false);
  const editorHolder = useRef<Editor | null>(null);
  const pendingPlain = useRef<string | null>(null);
  const draftKeyRef = useRef(props.draftKey);
  const onKeyDownRef = useRef(props.onKeyDown);
  onKeyDownRef.current = props.onKeyDown;
  const [empty, setEmpty] = useState(true);
  const editor = useEditor({
    // React 19 + nested Suspense (Yingce island auth/route) destroys the view
    // and later reconnects passive effects. The default sync first render
    // leaves a stale Editor whose `.commands` getter throws on null view.
    immediatelyRender: false,
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
      Placeholder.configure({
        placeholder: () => placeholderRef.current,
        showOnlyCurrent: false,
      }),
      MentionChip,
    ],
    editable: !props.disabled,
    editorProps: {
      attributes: editorAttrs(props),
      handlePaste: (_view, event) => {
        const files = filesFromClipboard(event);
        if (files.length && onPasteRef.current) {
          onPasteRef.current(files);
          return true;
        }
        return false;
      },
      handleKeyDown: (_view, event) => {
        const ed = editorHolder.current;
        if (!liveEditor(ed)) return false;
        if (onKeyDownRef.current?.(event, snapshotOf(ed), textBeforeCaret(ed))) return true;
        return handleChipSelection(ed, event);
      },
    },
    onCreate: ({ editor: ed }) => {
      editorHolder.current = ed;
      setEmpty(ed.isEmpty);
    },
    onDestroy: () => {
      editorHolder.current = null;
    },
    onUpdate: ({ editor: ed }) => {
      setEmpty(ed.isEmpty);
      if (skipUpdate.current) return;
      onSnapshotRef.current(snapshotOf(ed));
    },
    onSelectionUpdate: ({ editor: ed }) => {
      if (skipUpdate.current) return;
      onCaretRef.current?.(snapshotOf(ed));
    },
  });
  editorHolder.current = liveEditor(editor) ? editor : null;

  useEffect(() => {
    const keyChanged = draftKeyRef.current !== props.draftKey;
    draftKeyRef.current = props.draftKey;
    if (!liveEditor(editor)) {
      if (keyChanged) pendingPlain.current = null;
      return;
    }
    if (keyChanged) {
      pendingPlain.current = null;
      skipUpdate.current = true;
      editor.commands.clearContent();
      skipUpdate.current = false;
      setEmpty(true);
      return;
    }
    if (pendingPlain.current === null) return;
    const text = pendingPlain.current;
    pendingPlain.current = null;
    editor.commands.setContent(plainToContent(text));
    setEmpty(!text.trim());
    if (text.trim()) editor.commands.focus("end");
  }, [props.draftKey, editor]);

  useEffect(() => {
    if (!liveEditor(editor)) return;
    editor.setEditable(!props.disabled);
  }, [editor, props.disabled]);

  useEffect(() => {
    if (!liveEditor(editor)) return;
    editor.setOptions({
      editorProps: {
        ...editor.options.editorProps,
        attributes: editorAttrs(props),
      },
    });
  }, [editor, props.ariaLabel, props.testId, props.workspaceTrigger, props.pickerOpen]);

  useImperativeHandle(ref, () => ({
    insertChip: (pin) => {
      if (!liveEditor(editor)) return;
      editor.chain().focus().insertMentionChip(pin).run();
    },
    insertText: (text) => {
      if (!liveEditor(editor)) return;
      editor.chain().focus().insertContent(text).run();
    },
    setPlain: (text) => {
      if (!liveEditor(editor)) {
        pendingPlain.current = text;
        return;
      }
      pendingPlain.current = null;
      skipUpdate.current = true;
      editor.commands.setContent(plainToContent(text));
      skipUpdate.current = false;
      setEmpty(!text.trim());
    },
    focus: () => {
      if (!liveEditor(editor)) return;
      editor.commands.focus("end");
    },
    snapshot: () => {
      if (liveEditor(editor)) return snapshotOf(editor);
      const text = pendingPlain.current ?? "";
      return { text, chips: [], caret: text.length, before: text };
    },
    replaceMention: (pin) => {
      if (!liveEditor(editor)) return;
      const token = activeAtTail(textBeforeCaret(editor));
      if (!token) {
        editor.chain().focus().insertMentionChip(pin).run();
        return;
      }
      const from = editor.state.selection.from - token.length;
      const to = editor.state.selection.from;
      editor.chain().focus().deleteRange({ from, to }).insertMentionChip(pin).run();
    },
    replaceTrigger: (text) => {
      if (!liveEditor(editor)) return;
      const token = activeAtTail(textBeforeCaret(editor));
      if (!token) {
        editor.chain().focus().insertContent(text).run();
        return;
      }
      const from = editor.state.selection.from - token.length;
      const to = editor.state.selection.from;
      editor.chain().focus().deleteRange({ from, to }).insertContent(text).run();
    },
    replaceSlash: () => {
      if (!liveEditor(editor)) return;
      const before = textBeforeCaret(editor);
      const m = /(?:^|\s)(\/[^\s]*)$/.exec(before);
      if (!m?.[1]) return;
      const token = m[1];
      const from = editor.state.selection.from - token.length;
      const to = editor.state.selection.from;
      editor.chain().focus().deleteRange({ from, to }).run();
    },
  }), [editor]);

  return (
    <div className={cn("sc-scroll")} data-input-scroll>
      <div className="sc-grow">
        {empty ? (
          <div className="sc-placeholder" aria-hidden="true">
            {props.placeholder}
          </div>
        ) : null}
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

const PASSIVE_KEYS = new Set(["Shift", "Control", "Alt", "Meta", "Unidentified", "Process"]);

function handleChipSelection(editor: Editor, event: KeyboardEvent): boolean {
  if (event.key !== "Backspace" && event.key !== "Delete") {
    if (!PASSIVE_KEYS.has(event.key) && event.key.length) clearChipSelected(editor);
    return false;
  }
  const sel = editor.state.selection;
  if (sel instanceof NodeSelection && sel.node.type.name === "mentionChip") return false;
  const backward = event.key === "Backspace";
  const $from = sel.$from;
  if (!sel.empty) return false;
  const node = backward ? $from.nodeBefore : $from.nodeAfter;
  if (!node || node.type.name !== "mentionChip") {
    clearChipSelected(editor);
    return false;
  }
  const pos = backward ? $from.pos - node.nodeSize : $from.pos;
  const dom = editor.view.nodeDOM(pos);
  const el = dom instanceof HTMLElement ? (dom.closest("[data-mention-chip]") || (dom.hasAttribute("data-mention-chip") ? dom : null)) : null;
  if (!(el instanceof HTMLElement)) return false;
  if (el.hasAttribute("data-selected")) {
    clearChipSelected(editor);
    editor.chain().focus().deleteRange({ from: pos, to: pos + node.nodeSize }).run();
    return true;
  }
  clearChipSelected(editor);
  el.setAttribute("data-selected", "");
  editor.view.dom.setAttribute("data-composer-chip-selected", "");
  return true;
}

function clearChipSelected(editor: Editor) {
  editor.view.dom.removeAttribute("data-composer-chip-selected");
  editor.view.dom.querySelectorAll("[data-mention-chip][data-selected]").forEach((n) => n.removeAttribute("data-selected"));
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
