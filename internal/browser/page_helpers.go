package browser

// pageHelpers is the single definition of interactive, visible and text that
// the action list, click, type, fill and select all share. Only function
// declarations go here: each evaluation wraps them in its own IIFE, so
// nothing leaks into the page's global scope.
//
// Refs are assigned by a deterministic walk (same-origin iframes, then
// shadow roots). Resolving ref=btn_0 re-walks the live document — the handle
// names the current node, not a CSS guess. Coordinates add iframe offsets
// so a real pointer event in the top viewport lands on the inner node.
const pageHelpers = `
function __yoyoVisible(el) {
  const r = el.getBoundingClientRect();
  if (r.width < 2 || r.height < 2) return false;
  const cs = getComputedStyle(el);
  return cs.visibility !== "hidden" && cs.display !== "none" && parseFloat(cs.opacity) > 0;
}
function __yoyoText(el) {
  const t = el.innerText || el.value || el.getAttribute("aria-label") || el.getAttribute("placeholder") || el.title || "";
  return String(t).replace(/\s+/g, " ").trim().slice(0, 90);
}
function __yoyoPrefix(tag) {
  if (tag === "input") return "inp";
  if (tag === "button") return "btn";
  if (tag === "a") return "lnk";
  if (tag === "select") return "sel";
  if (tag === "textarea") return "txt";
  return String(tag || "el").slice(0, 3);
}
function __yoyoQueryRoot(el) {
  const root = el.getRootNode ? el.getRootNode() : el.ownerDocument;
  if (root && root.querySelector) return root;
  return el.ownerDocument || document;
}
function __yoyoSel(el) {
  const ident = /^[A-Za-z][\\w-]*$/;
  const root = __yoyoQueryRoot(el);
  const ok = (s) => { try { return root.querySelector(s) === el; } catch (e) { return false; } };
  const testid = el.getAttribute("data-testid") || el.getAttribute("data-test") || el.getAttribute("data-cy");
  if (testid) {
    const s = '[data-testid="' + String(testid).replace(/"/g, '\\"') + '"]';
    if (ok(s)) return s;
  }
  if (el.id && ident.test(el.id) && ok("#" + el.id)) return "#" + el.id;
  const parts = [];
  const stop = root.body || root;
  for (let n = el; n && n.nodeType === 1 && n !== stop && n !== document.documentElement; n = n.parentElement || (n.getRootNode && n.getRootNode().host)) {
    if (n.id && ident.test(n.id)) { parts.unshift("#" + n.id); break; }
    let p = n.tagName.toLowerCase();
    const parent = n.parentElement;
    if (parent) {
      const same = Array.from(parent.children).filter((c) => c.tagName === n.tagName);
      if (same.length > 1) p += ":nth-of-type(" + (same.indexOf(n) + 1) + ")";
    }
    parts.unshift(p);
    if (parts.length > 8) break;
  }
  const s = parts.join(" > ");
  return ok(s) ? s : "";
}
function __yoyoFrameOffset(win) {
  let ox = 0, oy = 0;
  let w = win;
  while (w && w !== w.top) {
    try {
      const f = w.frameElement;
      if (!f) break;
      const r = f.getBoundingClientRect();
      ox += r.left;
      oy += r.top;
      w = w.parent;
    } catch (e) { break; }
  }
  return {ox: ox, oy: oy};
}
function __yoyoSeed(root) {
  return Array.from(root.querySelectorAll('button,a[href],input:not([type=hidden]),select,textarea,summary,[role=button],[role=option],[role=link],[role=menuitem],[role=tab],[role=radio],[role=checkbox],[role=textbox],[role=combobox],[role=switch],[role=searchbox],[onclick],[data-opt],[data-action],[data-testid],[tabindex]:not([tabindex="-1"])'));
}
function __yoyoPointerExtras(root, seen) {
  const scope = root.body || root;
  if (!scope || !scope.querySelectorAll) return;
  const all = scope.querySelectorAll("*");
  for (let i = 0; i < all.length && i < 3000; i++) {
    const el = all[i];
    if (seen.has(el)) continue;
    try {
      if (getComputedStyle(el).cursor !== "pointer") continue;
      const pe = el.parentElement;
      if (pe && getComputedStyle(pe).cursor === "pointer") continue;
    } catch (e) { continue; }
    seen.add(el);
  }
}
function __yoyoWalkDocs(visit) {
  const seenDoc = new Set();
  function walk(root, win, framePath, shadowPath) {
    if (!root || seenDoc.has(root)) return;
    seenDoc.add(root);
    visit(root, win, framePath, shadowPath);
    const scope = root.body || root;
    if (!scope || !scope.querySelectorAll) return;
    const all = scope.querySelectorAll("*");
    for (let i = 0; i < all.length && i < 4000; i++) {
      const el = all[i];
      if (el.shadowRoot) {
        const hostSel = el.id ? (el.tagName.toLowerCase() + "#" + el.id) : el.tagName.toLowerCase();
        walk(el.shadowRoot, win, framePath, shadowPath.concat([hostSel]));
      }
      const tag = el.tagName;
      if (tag === "IFRAME" || tag === "FRAME") {
        try {
          const child = el.contentDocument;
          const cw = el.contentWindow;
          if (child && cw) {
            const sel = el.id ? (tag.toLowerCase() + "#" + el.id) : tag.toLowerCase();
            walk(child, cw, framePath.concat([sel]), shadowPath);
          }
        } catch (e) {}
      }
    }
  }
  walk(document, window, [], []);
}
function __yoyoCandidates() {
  const out = [];
  const counters = {};
  __yoyoWalkDocs(function(root, win, framePath, shadowPath) {
    const seen = new Set();
    for (const el of __yoyoSeed(root)) seen.add(el);
    __yoyoPointerExtras(root, seen);
    const off = __yoyoFrameOffset(win);
    for (const el of seen) {
      if (!__yoyoVisible(el)) continue;
      const tag = el.tagName.toLowerCase();
      if (tag === "a" && !el.getAttribute("href")) continue;
      const prefix = __yoyoPrefix(tag);
      const idx = counters[prefix] || 0;
      counters[prefix] = idx + 1;
      out.push({
        el: el,
        ref: prefix + "_" + idx,
        tag: tag,
        ox: off.ox,
        oy: off.oy,
        framePath: framePath,
        shadowPath: shadowPath
      });
    }
  });
  return out;
}
function __yoyoParseRef(sel) {
  sel = String(sel || "").trim();
  if (sel.charAt(0) === "[" && sel.slice(-1) === "]") sel = sel.slice(1, -1);
  const low = sel.toLowerCase();
  if (low.indexOf("ref=") === 0) return sel.slice(4).trim();
  if (low.indexOf("ref:") === 0) return sel.slice(4).trim();
  if (/^[a-z]{2,3}_\\d+$/.test(sel)) return sel;
  return "";
}
function __yoyoResolveRec(sel) {
  const wantRef = __yoyoParseRef(sel);
  if (wantRef) {
    for (const c of __yoyoCandidates()) {
      if (c.ref === wantRef) return c;
    }
    return null;
  }
  if (String(sel).indexOf("text=") === 0) {
    const want = String(sel).slice(5).trim();
    let best = null;
    let bestLen = Infinity;
    for (const c of __yoyoCandidates()) {
      const t = __yoyoText(c.el);
      if (!t || t.indexOf(want) < 0) continue;
      if (t.length < bestLen) { best = c; bestLen = t.length; }
    }
    return best;
  }
  try {
    let found = null;
    __yoyoWalkDocs(function(root, win, framePath, shadowPath) {
      if (found) return;
      try {
        const el = root.querySelector(sel);
        if (!el) return;
        const off = __yoyoFrameOffset(win);
        found = {el: el, ref: "", tag: el.tagName.toLowerCase(), ox: off.ox, oy: off.oy, framePath: framePath, shadowPath: shadowPath};
      } catch (e) {}
    });
    return found;
  } catch (e) {
    return null;
  }
}
function __yoyoResolve(sel) {
  const rec = __yoyoResolveRec(sel);
  return rec ? rec.el : null;
}
function __yoyoDescribe(el) {
  if (!el) return "nothing";
  const id = el.id ? "#" + el.id : "";
  const cls = el.classList && el.classList.length ? "." + Array.from(el.classList).slice(0, 2).join(".") : "";
  const t = __yoyoText(el).slice(0, 30);
  return el.tagName.toLowerCase() + id + cls + (t ? ' "' + t + '"' : "");
}
function __yoyoHitOK(rec) {
  const el = rec.el;
  const r = el.getBoundingClientRect();
  const lx = r.left + r.width / 2;
  const ly = r.top + r.height / 2;
  const x = lx + rec.ox;
  const y = ly + rec.oy;
  const doc = el.ownerDocument || document;
  const top = doc.elementFromPoint(lx, ly);
  if (!top || (top !== el && !el.contains(top))) return {ok: false, x: x, y: y, by: __yoyoDescribe(top)};
  return {ok: true, x: x, y: y, by: ""};
}
function __yoyoActionOf(c) {
  const el = c.el;
  const t = __yoyoText(el);
  let s = __yoyoSel(el);
  if (!s && t) s = "text=" + t.slice(0, 40);
  if (!s && c.ref) s = "ref=" + c.ref;
  const testId = el.getAttribute("data-testid") || el.getAttribute("data-test") || el.getAttribute("data-cy") || "";
  return {
    ref: c.ref,
    selector: s,
    tag: c.tag,
    text: t,
    disabled: !!el.disabled,
    role: (el.getAttribute("role") || c.tag),
    testId: testId,
    name: el.getAttribute("aria-label") || "",
    placeholder: el.getAttribute("placeholder") || "",
    id: el.id || "",
    nameAttr: el.getAttribute("name") || "",
    href: el.getAttribute("href") || "",
    type: (el.getAttribute("type") || ""),
    framePath: c.framePath || [],
    shadowPath: c.shadowPath || []
  };
}
`
