/** Drop duplicate paste into editable fields (Wails Edit menu + WebView2). */

export const DUPLICATE_PASTE_MS = 48;
export const DUPLICATE_INSERT_MS = 120;

export type PasteStamp = {
  at: number;
  target: EventTarget | null;
  signature: string;
};

export function pasteSignature(text: string, fileCount: number): string {
  return `${fileCount}:${text}`;
}

export function isDuplicateStamp(prev: PasteStamp | null, next: PasteStamp, windowMs = DUPLICATE_PASTE_MS): boolean {
  if (!prev) return false;
  const dt = next.at - prev.at;
  return prev.target === next.target && prev.signature === next.signature && dt >= 0 && dt < windowMs;
}

export function shouldSkipProgrammaticInsert(opts: {
  inserted: string;
  fieldValue: string;
  caret: number;
  lastPasteText: string;
  elapsedMs: number;
}): boolean {
  const inserted = opts.inserted;
  if (!inserted || opts.elapsedMs < 0 || opts.elapsedMs >= DUPLICATE_INSERT_MS) return false;
  if (inserted !== opts.lastPasteText) return false;
  return opts.caret >= inserted.length && opts.fieldValue.slice(opts.caret - inserted.length, opts.caret) === inserted;
}

const SKIP_INPUT_TYPES = new Set([
  "button",
  "checkbox",
  "radio",
  "file",
  "submit",
  "reset",
  "range",
  "color",
  "hidden",
  "image",
]);

export function isEditableTarget(target: EventTarget | null): target is HTMLElement {
  if (!(target instanceof HTMLElement)) return false;
  if (target instanceof HTMLInputElement) {
    if (SKIP_INPUT_TYPES.has(target.type)) return false;
    return !target.readOnly && !target.disabled;
  }
  if (target instanceof HTMLTextAreaElement) return !target.readOnly && !target.disabled;
  return target.isContentEditable;
}

type TextField = HTMLInputElement | HTMLTextAreaElement;

function isTextField(el: Element | null): el is TextField {
  return el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement;
}

let installed = false;

export function installInputGuards(): void {
  if (installed || typeof document === "undefined") return;
  installed = true;

  let lastPaste: PasteStamp | null = null;
  let lastBeforePaste: PasteStamp | null = null;
  let lastCommitted = { at: 0, text: "" };

  const remember = (e: Event, text: string, files: number, slot: "paste" | "before") => {
    const stamp: PasteStamp = {
      at: performance.now(),
      target: e.target,
      signature: pasteSignature(text, files),
    };
    const prev = slot === "paste" ? lastPaste : lastBeforePaste;
    if (isDuplicateStamp(prev, stamp)) {
      e.preventDefault();
      e.stopImmediatePropagation();
      return true;
    }
    if (slot === "paste") lastPaste = stamp;
    else lastBeforePaste = stamp;
    lastCommitted = { at: stamp.at, text };
    return false;
  };

  document.addEventListener(
    "paste",
    (e) => {
      if (!isEditableTarget(e.target)) return;
      const text = e.clipboardData?.getData("text/plain") ?? "";
      remember(e, text, e.clipboardData?.files?.length ?? 0, "paste");
    },
    true,
  );

  document.addEventListener(
    "beforeinput",
    (e) => {
      if (e.inputType !== "insertFromPaste") return;
      if (!isEditableTarget(e.target)) return;
      remember(e, e.data ?? "", 0, "before");
    },
    true,
  );

  const nativeExec = document.execCommand.bind(document);
  document.execCommand = ((commandId: string, showUI?: boolean, value?: string) => {
    if ((commandId === "insertText" || commandId === "paste") && isTextField(document.activeElement)) {
      const el = document.activeElement;
      const inserted = commandId === "paste" ? lastCommitted.text : String(value ?? "");
      const caret = el.selectionEnd ?? el.value.length;
      if (
        shouldSkipProgrammaticInsert({
          inserted,
          fieldValue: el.value,
          caret,
          lastPasteText: lastCommitted.text,
          elapsedMs: performance.now() - lastCommitted.at,
        })
      ) {
        return false;
      }
    }
    return nativeExec(commandId, showUI, value);
  }) as typeof document.execCommand;
}
