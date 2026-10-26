/**
 * 拥有：把纯文本写入剪贴板。Wails 网页视图里 navigator.clipboard 经常失败，这时改用 execCommand。
 * 不拥有：从剪贴板读取，以及决定复制哪段文字。
 */

/** writeClipboard 写入纯文本。空文本返回假。剪贴板接口失败时，用隐藏文本域执行 copy。 */
export async function writeClipboard(text: string): Promise<boolean> {
  const value = text ?? "";
  if (!value) return false;
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(value);
      return true;
    }
  } catch {
    /* 改用 execCommand */
  }
  try {
    const el = document.createElement("textarea");
    el.value = value;
    el.setAttribute("readonly", "");
    el.style.cssText = "position:fixed;left:-9999px;top:0;opacity:0";
    document.body.appendChild(el);
    el.focus();
    el.select();
    el.setSelectionRange(0, value.length);
    const ok = document.execCommand("copy");
    document.body.removeChild(el);
    return ok;
  } catch {
    return false;
  }
}
