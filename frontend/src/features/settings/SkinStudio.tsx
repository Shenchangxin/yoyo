import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ImagePlus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { wallpaperVeil } from "../../lib/skin/extract";
import { defaultMaterials, isUserSkinId, type SkinMaterials } from "../../lib/skin/schema";
import { useTheme } from "../../lib/theme";
import { Button } from "../../components/ui/button";
import { Slider } from "../../components/ui/slider";
import { cn } from "../../lib/utils";
import { SettingRow, SettingSection, SettingSegmented } from "./SettingChrome";
import type { SettingsHost } from "./host";

function fileToB64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => {
      const s = String(r.result || "");
      const i = s.indexOf(",");
      resolve(i >= 0 ? s.slice(i + 1) : s);
    };
    r.onerror = () => reject(r.error);
    r.readAsDataURL(file);
  });
}

export function SkinStudio({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const { resolved, skinId, setSkinOverlay } = useTheme();
  const [busy, setBusy] = useState(false);
  const [draftId, setDraftId] = useState("");
  const [wallpaperUrl, setWallpaperUrl] = useState("");
  const [wallpaperFile, setWallpaperFile] = useState("");
  const [materials, setMaterials] = useState<SkinMaterials>(defaultMaterials());
  const wallRef = useRef<HTMLInputElement>(null);
  const saveTimer = useRef<number>(0);

  const activeId = isUserSkinId(host.cfg.skin || skinId) ? host.cfg.skin || skinId : "";

  const paint = useCallback(
    (id: string, mats: SkinMaterials, url: string, file: string) => {
      if (!isUserSkinId(id) || !url) {
        setSkinOverlay(null);
        return;
      }
      setSkinOverlay({
        id,
        tokens: {},
        materials: mats,
        wallpaperUrl: url,
        wallpaperFile: file,
        preserveEvidence: true,
      });
    },
    [setSkinOverlay],
  );

  const hydrate = useCallback(async (id: string) => {
    const got = await api.getSkin(id);
    const mats = got.info.materials || defaultMaterials();
    const url = got.info.wallpaperUrl || "";
    const file = url.split("/").pop() || "";
    setDraftId(id);
    setMaterials(mats);
    setWallpaperUrl(url);
    setWallpaperFile(file);
    paint(id, mats, url, file);
  }, [paint]);

  useEffect(() => {
    if (!activeId) {
      setDraftId("");
      setWallpaperUrl("");
      setWallpaperFile("");
      setSkinOverlay(null);
      return;
    }
    void hydrate(activeId).catch(() => setSkinOverlay(null));
  }, [activeId, hydrate, setSkinOverlay]);

  const persist = useCallback(
    async (id: string, mats: SkinMaterials, extra?: { wallpaper_b64?: string; wallpaper_name?: string }) => {
      const info = await api.saveSkin({
        id,
        name: copy.settings.skinWallpaper,
        preserveEvidence: true,
        dark: {},
        light: {},
        materials: mats,
        wallpaper_b64: extra?.wallpaper_b64,
        wallpaper_name: extra?.wallpaper_name,
      });
      await api.activateSkin(info.id);
      await host.patch({ skin: info.id });
      const url = info.wallpaperUrl || wallpaperUrl;
      const file = (info.wallpaperUrl || "").split("/").pop() || wallpaperFile;
      setDraftId(info.id);
      setWallpaperUrl(url);
      setWallpaperFile(file);
      paint(info.id, mats, url, file);
      return info;
    },
    [copy.settings.skinWallpaper, host, paint, wallpaperFile, wallpaperUrl],
  );

  function queueSave(mats: SkinMaterials) {
    window.clearTimeout(saveTimer.current);
    saveTimer.current = window.setTimeout(() => {
      if (!draftId) return;
      void persist(draftId, mats).catch((e) => toast.error(String(e)));
    }, 400);
  }

  async function onPick(file: File) {
    setBusy(true);
    try {
      const b64 = await fileToB64(file);
      let veil = 0.52;
      try {
        veil = await wallpaperVeil(file, resolved);
      } catch {
        /* keep fallback */
      }
      const mats: SkinMaterials = { ...materials, wallpaperDim: veil, wallpaperFit: materials.wallpaperFit || "cover" };
      setMaterials(mats);
      await persist(draftId || activeId, mats, { wallpaper_b64: b64, wallpaper_name: file.name });
    } catch (e) {
      toast.error(String(e));
    } finally {
      setBusy(false);
    }
  }

  async function clearWallpaper() {
    setBusy(true);
    try {
      if (draftId) await api.deleteSkin(draftId);
      await api.activateSkin("");
      await host.patch({ skin: "" });
      setSkinOverlay(null);
      setDraftId("");
      setWallpaperUrl("");
      setWallpaperFile("");
      setMaterials(defaultMaterials());
    } catch (e) {
      toast.error(String(e));
    } finally {
      setBusy(false);
    }
  }

  const hasWall = !!wallpaperUrl;
  const dimPct = Math.round((materials.wallpaperDim ?? 0) * 100);

  const previewStyle = useMemo(
    () =>
      hasWall
        ? {
            backgroundImage: `url("${wallpaperUrl}")`,
            backgroundSize: materials.wallpaperFit === "contain" ? "contain" : materials.wallpaperFit === "tile" ? "auto" : "cover",
            backgroundRepeat: materials.wallpaperFit === "tile" ? "repeat" : "no-repeat",
            backgroundPosition: "center",
          }
        : undefined,
    [hasWall, wallpaperUrl, materials.wallpaperFit],
  );

  return (
    <SettingSection
      id="appearance-skins"
      title={copy.settings.sections.appearanceSkins}
      description={copy.settings.skinHint}
      footnote={copy.settings.appearanceHint}
    >
      <div
        className="relative mx-4 mt-3 overflow-hidden rounded-xl border border-border"
        style={{ height: 120, ...previewStyle, backgroundColor: "var(--lift)" }}
      >
        {hasWall ? (
          <div
            className="absolute inset-0"
            style={{ background: `color-mix(in srgb, var(--background) ${dimPct}%, transparent)` }}
          />
        ) : (
          <p className="absolute inset-0 grid place-items-center px-4 text-center text-[12px] text-muted">{copy.settings.skinEmpty}</p>
        )}
      </div>
      <SettingRow title={copy.settings.skinWallpaper} description={copy.settings.skinWallpaperDesc} stack>
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="lift" className="h-8" disabled={busy} onClick={() => wallRef.current?.click()}>
            <ImagePlus className="size-3.5" />
            {copy.settings.skinPickImage}
          </Button>
          {hasWall ? (
            <Button type="button" variant="ghost" className="h-8 text-danger" disabled={busy} onClick={() => void clearWallpaper()}>
              <Trash2 className="size-3.5" />
              {copy.settings.skinRemove}
            </Button>
          ) : null}
          <input
            ref={wallRef}
            type="file"
            accept="image/png,image/jpeg,image/webp,image/*"
            className="hidden"
            onChange={(e) => {
              const f = e.target.files?.[0];
              e.target.value = "";
              if (f) void onPick(f);
            }}
          />
        </div>
      </SettingRow>
      <SettingRow title={copy.settings.skinFit}>
        <SettingSegmented
          id="skin-fit"
          ariaLabel={copy.settings.skinFit}
          value={materials.wallpaperFit}
          options={[
            { value: "cover" as const, label: copy.settings.skinFitCover },
            { value: "contain" as const, label: copy.settings.skinFitContain },
            { value: "tile" as const, label: copy.settings.skinFitTile },
          ]}
          onChange={(fit) => {
            const next = { ...materials, wallpaperFit: fit };
            setMaterials(next);
            if (draftId) {
              paint(draftId, next, wallpaperUrl, wallpaperFile);
              queueSave(next);
            }
          }}
        />
      </SettingRow>
      <SettingRow title={copy.settings.skinVeil} description={`${dimPct}% · ${copy.settings.skinVeilDesc}`}>
        <Slider
          min={0}
          max={0.88}
          step={0.01}
          value={[materials.wallpaperDim]}
          onValueChange={(v) => {
            const next = { ...materials, wallpaperDim: v[0] ?? 0 };
            setMaterials(next);
            if (draftId) {
              paint(draftId, next, wallpaperUrl, wallpaperFile);
              queueSave(next);
            }
          }}
        />
      </SettingRow>
      {hasWall ? (
        <div className={cn("px-4 pb-3")}>
          <Button
            type="button"
            variant="ghost"
            className="h-8"
            disabled={busy || !wallpaperUrl}
            onClick={async () => {
              try {
                const blob = await (await fetch(wallpaperUrl)).blob();
                const veil = await wallpaperVeil(blob, resolved);
                const next = { ...materials, wallpaperDim: veil };
                setMaterials(next);
                if (draftId) {
                  paint(draftId, next, wallpaperUrl, wallpaperFile);
                  await persist(draftId, next);
                }
              } catch (e) {
                toast.error(String(e));
              }
            }}
          >
            {copy.settings.skinVeilAuto}
          </Button>
        </div>
      ) : null}
    </SettingSection>
  );
}
