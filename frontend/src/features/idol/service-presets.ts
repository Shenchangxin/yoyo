/** 拥有偶像服务预设和声线的形状、解析与查找。不拥有设置页。这是预设载荷的投影，不改服务端状态。 */

import { asArray, str } from "../../lib/normalize";

/** ServiceVoice 一条服务商声音。models 有值时只适用于这些模型。 */
export type ServiceVoice = { id: string; label: string; models?: string[] };

/** ServicePreset 一条服务预设：种类、厂商、地址、模型和可选声音。 */
export type ServicePreset = {
  id: string;
  name: string;
  kind: string;
  vendor: string;
  endpoint: string;
  models: string[];
  voices?: ServiceVoice[];
};

/** customProvider 只在界面里表示自定义地址，不会入库。自定义地址保存时服务商 id 为空。 */
export const customProvider = "__custom__";

/** presetsOf 从接口原文整理预设。没有 id 或种类的丢掉，没有 id 的声音也丢掉。 */
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

/** voicesFor 按模型筛声音。声音未限定模型、未指定模型，或列表含该模型时留下。没有声音表则空。 */
export function voicesFor(preset: ServicePreset | undefined, model: string): ServiceVoice[] {
  if (!preset?.voices?.length) return [];
  const want = model.trim();
  return preset.voices.filter((v) => !v.models?.length || !want || v.models.includes(want));
}

/** presetByID 按 id 和种类找预设。id 为空白则没有。 */
export function presetByID(presets: ServicePreset[], id: string, kind: string): ServicePreset | undefined {
  const want = id.trim();
  if (!want) return undefined;
  return presets.find((p) => p.id === want && p.kind === kind);
}
