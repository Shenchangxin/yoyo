/**
 * 拥有：设置页哈希和当前页签的同步。旧分组时期的链接会落到现行布局。
 * 不拥有：表单。改的是地址栏和界面状态，不调用 api。
 */
import { useEffect, useLayoutEffect, useRef } from "react";
import { resolveSettingsTarget } from "./registry";
import { useUI } from "../../lib/store";

/** 解析地址栏里的设置链接；旧分组时期的链接会落到现行布局。 */
function parseSettingsHash() {
  const h = location.hash.slice(1);
  const [path, query] = h.split("?");
  const m = path.match(/^\/settings\/([a-z-]+)$/);
  if (!m) return null;
  const params = new URLSearchParams(query || "");
  return resolveSettingsTarget(m[1], params.get("section") || "");
}

/** useSettingsHash 让地址栏和当前设置页签保持一致。 */
export function useSettingsHash() {
  const surface = useUI((s) => s.surface);
  const tab = useUI((s) => s.settingsTab);
  const section = useUI((s) => s.settingsSection);
  const ready = useRef(false);

  useLayoutEffect(() => {
    const parsed = parseSettingsHash();
    if (parsed) useUI.getState().openSettings(parsed.tab, parsed.section);
    ready.current = true;
  }, []);

  useEffect(() => {
    if (!ready.current) return;
    if (surface === "settings") {
      const q = new URLSearchParams();
      if (section) q.set("section", section);
      const qs = q.toString();
      const next = `#/settings/${tab}${qs ? `?${qs}` : ""}`;
      if (location.hash !== next) history.replaceState(null, "", next);
      return;
    }
    if (location.hash.startsWith("#/settings")) history.replaceState(null, "", "#/");
  }, [surface, tab, section]);

  useEffect(() => {
    /** 地址变了就按哈希打开对应页签。 */
    const apply = () => {
      const parsed = parseSettingsHash();
      if (!parsed) return;
      useUI.getState().openSettings(parsed.tab, parsed.section);
    };
    window.addEventListener("hashchange", apply);
    return () => window.removeEventListener("hashchange", apply);
  }, []);
}
