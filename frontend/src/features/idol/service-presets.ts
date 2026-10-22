import { asArray, str } from "../../lib/normalize";

export type ServiceVoice = { id: string; label: string; models?: string[] };

export type ServicePreset = {
  id: string;
  name: string;
  kind: string;
  vendor: string;
  endpoint: string;
  models: string[];
  voices?: ServiceVoice[];
};

/** UI-only value. It is never stored; a custom address is saved with an empty provider id. */
export const customProvider = "__custom__";

export function presetsOf(raw: unknown): ServicePreset[] {
  return asArray(raw).map((v: any) => ({
    id: str(v?.id),
    name: str(v?.name),
    kind: str(v?.kind),
    vendor: str(v?.vendor),
    endpoint: str(v?.endpoint),
    models: asArray(v?.models).map(String).filter(Boolean),
    voices: asArray(v?.voices).map((voice: any) => ({
      id: str(voice?.id),
      label: str(voice?.label) || str(voice?.id),
      models: asArray(voice?.models).map(String).filter(Boolean),
    })).filter((voice: ServiceVoice) => voice.id),
  })).filter((p) => p.id && p.kind);
}

export function voicesFor(preset: ServicePreset | undefined, model: string): ServiceVoice[] {
  if (!preset?.voices?.length) return [];
  const want = model.trim();
  return preset.voices.filter((v) => !v.models?.length || !want || v.models.includes(want));
}

export function presetByID(presets: ServicePreset[], id: string, kind: string): ServicePreset | undefined {
  const want = id.trim();
  if (!want) return undefined;
  return presets.find((p) => p.id === want && p.kind === kind);
}
