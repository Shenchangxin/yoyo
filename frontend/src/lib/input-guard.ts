/**
 * 拥有：丢掉可编辑区域里重复的粘贴。Wails 编辑菜单和 WebView2 会各送一次。
 * 不拥有：粘贴内容如何插入文档。
 */

/** DUPLICATE_PASTE_MS 同一目标、同一签名的粘贴在此毫秒内视为重复。 */
export const DUPLICATE_PASTE_MS = 48;
/** DUPLICATE_INSERT_MS 程序化插入若在此毫秒内、且光标前已经是刚粘贴的文本，则跳过。 */
export const DUPLICATE_INSERT_MS = 120;

/** PasteStamp 一次粘贴的时刻、目标和签名。 */
export type PasteStamp = {
  at: number;
  target: EventTarget | null;
  signature: string;
};

/** pasteSignature 文件个数和文本拼成签名。 */
export function pasteSignature(text: string, fileCount: number): string {
  return `${fileCount}:${text}`;
}

/** isDuplicateStamp 同一目标、同一签名，且时间差落在窗口内。没有上一次，或时间倒退，则不是重复。 */
export function isDuplicateStamp(prev: PasteStamp | null, next: PasteStamp, windowMs = DUPLICATE_PASTE_MS): boolean {
  if (!prev) return false;
  const dt = next.at - prev.at;
  return prev.target === next.target && prev.signature === next.signature && dt >= 0 && dt < windowMs;
}

/** shouldSkipProgrammaticInsert 插入文本与上次粘贴相同，时间差在 DUPLICATE_INSERT_MS 内，且光标前已经是这段文本。 */
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

/** isEditableTarget 可写的输入框、文本域，或 contentEditable。按钮、选择、文件、范围、颜色、隐藏和图片类型排除。只读和禁用排除。 */
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

/** 是文本输入或文本域。 */
function isTextField(el: Element | null): el is TextField {
  return el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement;
}

let installed = false;

/** installInputGuards 只安装一次。捕获阶段拦住重复的 paste 和 insertFromPaste。文本框上的 execCommand insertText 与 paste，会跳过刚粘贴过的插入。 */
export function installInputGuards(): void {
  if (installed || typeof document === "undefined") return;
  installed = true;

  let lastPaste: PasteStamp | null = null;
  let lastBeforePaste: PasteStamp | null = null;
  let lastCommitted = { at: 0, text: "" };

  /** 同一槽位、同一目标、同一签名且落在窗口内则拦下。否则记下这次签名和正文。 */
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
  /** execCommand 在原生粘贴刚写过同一段文字时，丢掉紧接着的 insertText 或 paste。 */
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
