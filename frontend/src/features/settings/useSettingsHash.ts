import { useEffect, useLayoutEffect, useRef } from "react";
import { resolveSettingsTarget } from "./registry";
import { useUI } from "../../lib/store";

/** Accepts links minted before the eight-tab regroup; they resolve onto the new layout. */
function parseSettingsHash() {
  const h = location.hash.slice(1);
  const [path, query] = h.split("?");
  const m = path.match(/^\/settings\/([a-z-]+)$/);
  if (!m) return null;
  const params = new URLSearchParams(query || "");
  return resolveSettingsTarget(m[1], params.get("section") || "");
}

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
    const apply = () => {
      const parsed = parseSettingsHash();
      if (!parsed) return;
      useUI.getState().openSettings(parsed.tab, parsed.section);
    };
    window.addEventListener("hashchange", apply);
    return () => window.removeEventListener("hashchange", apply);
  }, []);
}
