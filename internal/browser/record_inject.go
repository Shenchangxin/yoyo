// 本文件的 recorderJS 是注入页面的录制脚本，模板字符串是数据，不在这里改写。
// 脚本监听点击、输入、按键和滚动，经绑定 yoyoRecord 送回宿主，并在页面里打码敏感地址。

package browser

import (
	"encoding/json"
	"fmt"
)

// recorderJS 用 nonce 和录制 id 填好注入脚本。同一 REC 已经装过就直接返回。
// recID 经 json.Marshal 写进脚本里的 REC。
func recorderJS(nonce, recID string) string {
	n, _ := json.Marshal(nonce)
	id, _ := json.Marshal(recID)
	return fmt.Sprintf(`(() => {
  const NONCE = %s;
  const REC = %s;
  const BINDING = "yoyoRecord";
  if (window.__yoyoRecInstalled === REC) return;
  window.__yoyoRecInstalled = REC;
  let lastScroll = 0;
  const inputTimers = new WeakMap();
  const SENSITIVE = /password|token|secret|key|code|otp|auth|session|jwt/i;
  function text(v, max) {
    if (v == null) return "";
    return String(v).replace(/\s+/g, " ").trim().slice(0, max || 300);
  }
  function redactUrl(raw) {
    try {
      const u = new URL(raw, location.href);
      ["token","access_token","refresh_token","code","state","session","key","secret","password","auth","jwt"].forEach((k) => {
        if (u.searchParams.has(k)) u.searchParams.set(k, "***");
      });
      return u.toString();
    } catch (e) { return String(raw || "").slice(0, 2000); }
  }
  function emit(kind, data) {
    try {
      const fn = window[BINDING];
      if (typeof fn !== "function") return;
      const ev = Object.assign({
        kind: kind,
        ts: new Date().toISOString(),
        url: redactUrl(location.href),
        title: text(document.title, 200)
      }, data || {});
      fn(JSON.stringify({nonce: NONCE, event: ev}));
    } catch (e) {}
  }
  function selectors(el) {
    const out = [];
    const test = el.getAttribute("data-testid") || el.getAttribute("data-test") || el.getAttribute("data-cy");
    if (test) out.push('[data-testid="' + test + '"]');
    if (el.id) out.push("#" + el.id);
    if (el.getAttribute("name")) out.push(el.tagName.toLowerCase() + '[name="' + el.getAttribute("name") + '"]');
    if (el.getAttribute("aria-label")) out.push(el.tagName.toLowerCase() + '[aria-label="' + el.getAttribute("aria-label") + '"]');
    return out;
  }
  function describe(el) {
    if (!el || !el.tagName) return {};
    const hay = [el.type, el.name, el.id, el.getAttribute("placeholder"), el.getAttribute("aria-label"), el.innerText].join(" ");
    let value = "";
    if ("value" in el) value = String(el.value || "");
    if (SENSITIVE.test(hay) || /sk-|api_key|-----begin/i.test(value)) value = "{{sensitive_value}}";
    return {
      target: {
        tag: el.tagName.toLowerCase(),
        role: el.getAttribute("role") || "",
        type: el.getAttribute("type") || "",
        name: text(el.getAttribute("aria-label") || el.innerText, 80),
        label: text(el.getAttribute("aria-label"), 80),
        placeholder: el.getAttribute("placeholder") || "",
        id: el.id || "",
        nameAttr: el.getAttribute("name") || "",
        text: text(el.innerText || el.value, 80),
        testId: el.getAttribute("data-testid") || el.getAttribute("data-test") || "",
        href: el.getAttribute("href") || "",
        selectorCandidates: selectors(el)
      },
      value: value
    };
  }
  document.addEventListener("click", (e) => {
    const d = describe(e.target);
    emit("click", d);
  }, true);
  document.addEventListener("input", (e) => {
    const el = e.target;
    const prev = inputTimers.get(el);
    if (prev) clearTimeout(prev);
    inputTimers.set(el, setTimeout(() => {
      emit("input", describe(el));
    }, 300));
  }, true);
  document.addEventListener("change", (e) => {
    emit("change", describe(e.target));
  }, true);
  document.addEventListener("keydown", (e) => {
    if (["Enter","Tab","Escape","ArrowDown","ArrowUp"].indexOf(e.key) < 0) return;
    emit("keydown", {key: e.key, target: describe(e.target).target});
  }, true);
  document.addEventListener("scroll", () => {
    const now = Date.now();
    if (now - lastScroll < 500) return;
    lastScroll = now;
    emit("scroll", {deltaY: window.scrollY});
  }, true);
  emit("ready", {});
})();`, n, id)
}
