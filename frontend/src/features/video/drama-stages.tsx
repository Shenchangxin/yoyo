/**
 * 本文件拥有短剧各段的界面：剧本、定妆、分镜检查、成片选择、任务和分集图。
 * 不拥有拉取和保存。改服务端的动作都在回调里，本文件只发「送到剪辑」的本地事件。
 */
import { useEffect, useState } from "react";
import { ImagePlus } from "lucide-react";
import { Button } from "../../components/ui/button";
import { Input, Textarea } from "../../components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { requestSendToCut } from "./open-cut";
import { useUI } from "../../lib/store";
import { DramaMentionField, type MentionAsset } from "./DramaMentionField";
import { DramaStylePicker } from "./DramaStylePicker";
import {
  assetName,
  catalogOf,
  isNarrator,
  jobMoving,
  linkedAssets,
  providerKey,
  selectedClipIds,
  serviceKind,
  shownSeriesTitle,
  type Asset,
  type Bundle,
  type Drama,
  type Episode,
  type EpisodePlan,
  type Job,
  type Shot,
} from "./drama-lib";

type CopyT = ReturnType<typeof useCopy>;

const field =
  "no-drag h-8 rounded-lg border border-border bg-background px-2 text-[13px] text-foreground outline-none focus-visible:border-foreground/25 focus-visible:ring-1 focus-visible:ring-foreground/15";

/** 按镜头关联的角色、场景和道具列出可 @ 的名字。一个都没有就列出全部。 s 用 character_ids、prop_ids 和 scene_id 决定留下哪些。bundle 从 characters、scenes 和 props 里取有名字的项，一个都没对上就改列全部。 */
function shotMentions(s: Shot, bundle: Bundle): MentionAsset[] {
  const out: MentionAsset[] = [];
  const chars = new Set(s.character_ids || []);
  const props = new Set(s.prop_ids || []);
  const scene = s.scene_id || "";
  for (const c of bundle.characters || []) if (c.name && (chars.size === 0 || chars.has(c.id))) out.push({ name: c.name, kind: "character" });
  for (const sc of bundle.scenes || []) if (sc.location && (!scene || sc.id === scene)) out.push({ name: sc.location, kind: "scene" });
  for (const p of bundle.props || []) if (p.name && (props.size === 0 || props.has(p.id))) out.push({ name: p.name, kind: "prop" });
  if (out.length === 0) {
    for (const c of bundle.characters || []) if (c.name) out.push({ name: c.name, kind: "character" });
    for (const sc of bundle.scenes || []) if (sc.location) out.push({ name: sc.location, kind: "scene" });
    for (const p of bundle.props || []) if (p.name) out.push({ name: p.name, kind: "prop" });
  }
  return out.sort(/** a 用名字长度做减数。b 用名字长度做被减数。 */ (a, b) => b.name.length - a.name.length);
}

/** ProviderModelSelect 按种类列出仍可用的提供方和模型。一个都没有就去生成设置。 props 用 kind 决定要哪类提供方，用 providers、providerId、model 和 label 画下拉，没有可选项时用 label 打开设置，改选时把拆开的提供方和模型交给 onChange。 */
export function ProviderModelSelect(props: {
  kind: string;
  label: string;
  providers: any[];
  providerId: string;
  model: string;
  onChange: (providerId: string, model: string) => void;
}) {
  const copy = useCopy();
  const openSettings = useUI(/** s 取出 openSettings。 */ (s) => s.openSettings);
  const want = serviceKind(props.kind);
  const settingSection = want === "tts" ? "generation-speech" : `generation-${want}`;
  const rows = props.providers.filter(/** p 种类对不上就排除；停用、没有密钥、也不是当前选中时也排除。 */ (p) => {
    if (serviceKind(p.service_type ?? p.serviceType) !== want) return false;
    const selected = providerKey(p.id) === providerKey(props.providerId) && providerKey(p.id) !== "";
    return p.is_active !== false || p.has_key || selected;
  });
  if (rows.length === 0) {
    return (
      <button
        type="button"
        className={cn(field, "h-7 max-w-[14rem] text-left text-muted")}
        onClick={() => openSettings("generation", settingSection)}
      >
        {props.label} · {copy.video.openSettings}
      </button>
    );
  }
  const current = rows.find(/** p 用 id 去掉通道前缀后和当前提供方比较。 */ (p) => providerKey(p.id) === providerKey(props.providerId)) || rows.find(/** p 看是不是默认项。 */ (p) => p.is_default) || rows[0];
  const models = catalogOf(current, props.model);
  const model = props.model && models.includes(props.model) ? props.model : (current?.model || models[0] || "");
  const value = current ? `${current.id}::${model}` : "";
  return (
    <Select
      value={value || undefined}
      onValueChange={/** raw 用 :: 切开，前半和后半交给 onChange；没有 :: 就交两个空字符串。 */ (raw) => {
        const i = raw.indexOf("::");
        if (i < 0) {
          props.onChange("", "");
          return;
        }
        props.onChange(raw.slice(0, i), raw.slice(i + 2));
      }}
    >
      <SelectTrigger className={cn(field, "h-7 max-w-[14rem]")} aria-label={props.label}>
        <SelectValue placeholder={props.label} />
      </SelectTrigger>
      <SelectContent>
        {rows.map(/** p 取出模型列表，当前选中的提供方才带上已选模型，并和提供方 id、名字拼成选项。 */ (p) => {
          const models = catalogOf(p, providerKey(p.id) === providerKey(props.providerId) ? props.model : "");
          const ids = models.length ? models : [""];
          return ids.map(/** m 有模型名就接在提供方后面，并和提供方 id 拼成选项值。 */ (m) => (
            <SelectItem key={`${p.id}::${m}`} value={`${p.id}::${m}`}>
              {`${p.name || p.provider}${m ? ` · ${m}` : ""}`}
            </SelectItem>
          ));
        })}
      </SelectContent>
    </Select>
  );
}

/** GenerationSettings 收起风格、图像、视频、语音和分辨率。分辨率选项是 480p、720p、1080p。 props 用 copy 取文案，用 ep 填当前模型，用 dramaStyle、styles 和 onStyle 选风格，用 providers 画提供方，改模型时交给 onEpisode。 */
export function GenerationSettings(props: {
  ep: Episode;
  dramaStyle: string;
  styles: { value: string; name: string }[];
  providers: any[];
  copy: CopyT;
  onStyle: (style: string) => void;
  onEpisode: (patch: Record<string, string>) => void;
}) {
  const c = props.copy.video;
  const ep = props.ep;
  return (
    <details className="drama-gen-fold mt-3">
      <summary className="text-[11px] font-medium text-muted">{c.genSettings}</summary>
      <div className="mt-2 grid gap-1.5">
        <DramaStylePicker
          value={props.dramaStyle || "3d"}
          styles={props.styles}
          names={c.styleNames}
          label={c.style}
          title={c.stylePickerTitle}
          search={c.styleSearch}
          empty={c.styleEmpty}
          groups={c.styleGroups}
          placement="bottomLeft"
          onChange={props.onStyle}
        />
        <ProviderModelSelect kind="image" label={c.imageProvider} providers={props.providers} providerId={ep.image_provider_id || ""} model={ep.image_model || ""} onChange={/** id 写进 image_provider_id。model 写进 image_model。 */ (id, model) => props.onEpisode({ image_provider_id: id, image_model: model })} />
        <ProviderModelSelect kind="video" label={c.videoProvider} providers={props.providers} providerId={ep.video_provider_id || ""} model={ep.video_model || ""} onChange={/** id 写进 video_provider_id。model 写进 video_model。 */ (id, model) => props.onEpisode({ video_provider_id: id, video_model: model })} />
        <ProviderModelSelect kind="tts" label={c.kindSpeech} providers={props.providers} providerId={ep.tts_provider_id || ""} model={ep.tts_model || ""} onChange={/** id 写进 tts_provider_id。model 写进 tts_model。 */ (id, model) => props.onEpisode({ tts_provider_id: id, tts_model: model })} />
        <select className={cn(field, "h-7")} value={ep.resolution || "720p"} aria-label={c.resolution} onChange={/** e 取出 target.value 交给 onEpisode 的 resolution。 */ (e) => props.onEpisode({ resolution: e.target.value })}>
          {["480p", "720p", "1080p"].map(/** r 同时当选项的 key、值和显示文字。 */ (r) => <option key={r} value={r}>{r}</option>)}
        </select>
      </div>
    </details>
  );
}

/** ScriptStage 左右编辑原文和剧本，失焦时一起交给保存。 props 用 ep 填原文和剧本，用 copy 取文案，有 plan 就显示，失焦时把内容交给 onSave。 */
export function ScriptStage(props: { ep: Episode; copy: CopyT; plan?: Bundle["plan"]; onSave: (p: Partial<Episode>) => void }) {
  const [content, setContent] = useState(props.ep.content);
  const [script, setScript] = useState(props.ep.script_content);
  useEffect(() => {
    setContent(props.ep.content);
    setScript(props.ep.script_content);
  }, [props.ep.id, props.ep.content, props.ep.script_content]);
  const c = props.copy.video;
  const plan = props.plan;
  return (
    <div className="drama-script-split">
      {plan ? (
        <p className="drama-script-plan">{plan.chars} {c.chars} · {plan.target_seconds} {c.seconds} · {plan.segment_count} {c.segments}</p>
      ) : null}
      <label className="drama-script-col block">
        <div className="mb-1.5 text-[11px] text-muted">{c.novel}</div>
        <Textarea
          className="min-h-[calc(100%-1.5rem)] rounded-[10px] border border-border bg-card px-3 py-2 text-[13px] leading-6"
          value={content}
          placeholder={c.needSource}
          onChange={/** e 取出 target.value 交给 setContent。 */ (e) => setContent(e.target.value)}
          onBlur={() => props.onSave({ content, script_content: script })}
        />
      </label>
      <label className="drama-script-col block">
        <div className="mb-1.5 text-[11px] text-muted">{c.screenplay}</div>
        <Textarea
          className="min-h-[calc(100%-1.5rem)] rounded-[10px] border border-border bg-card px-3 py-2 font-mono text-[13px] leading-6"
          value={script}
          placeholder={c.screenplay}
          onChange={/** e 取出 target.value 交给 setScript。 */ (e) => setScript(e.target.value)}
          onBlur={() => props.onSave({ content, script_content: script })}
        />
      </label>
    </div>
  );
}

/** CastGrid 分角色、场景、道具三组显示仍关联的定妆。 props 用 copy 取文案，用 bundle 的角色、场景和道具分组，并把 focusId、onFocus、onCreate 传给每一组。 */
export function CastGrid(props: {
  bundle: Bundle;
  copy: CopyT;
  focusId: string;
  onFocus: (kind: string, id: string) => void;
  onCreate: (kind: string, fields: Record<string, string>) => void;
}) {
  const c = props.copy.video;
  const groups: { kind: string; title: string; items: Asset[]; create: string }[] = [
    { kind: "character", title: c.characters, items: linkedAssets(props.bundle.characters), create: c.addCharacter },
    { kind: "scene", title: c.scenes, items: linkedAssets(props.bundle.scenes), create: c.addScene },
    { kind: "prop", title: c.props, items: linkedAssets(props.bundle.props), create: c.addProp },
  ];
  return (
    <div className="grid gap-5 px-1 py-2">
      {groups.map(/** g 按 kind 当 key，其余字段展开传给 LookGroup。 */ (g) => (
        <LookGroup key={g.kind} {...g} copy={props.copy} focusId={props.focusId} onFocus={props.onFocus} onCreate={props.onCreate} />
      ))}
    </div>
  );
}

/** 一组定妆，可以就地加一条。 props 用 copy 取文案，显示 title 和 create，kind 决定新建时写 location 还是 name，列出 items，用 focusId 标出选中，点击交给 onFocus，提交交给 onCreate。 */
function LookGroup(props: {
  kind: string;
  title: string;
  items: Asset[];
  create: string;
  copy: CopyT;
  focusId: string;
  onFocus: (kind: string, id: string) => void;
  onCreate: (kind: string, fields: Record<string, string>) => void;
}) {
  const [adding, setAdding] = useState(false);
  const [name, setName] = useState("");
  const c = props.copy.video;
  return (
    <section>
      <div className="mb-2 flex items-baseline justify-between px-3">
        <span className="text-[11px] font-medium text-muted">{props.title}</span>
        <button type="button" className="text-[11px] text-muted hover:text-foreground" onClick={() => setAdding(true)}>{props.create}</button>
      </div>
      {adding ? (
        <form
          className="mb-2 flex gap-2 px-3"
          onSubmit={/** e 阻止默认。 */ (e) => {
            e.preventDefault();
            const n = name.trim();
            if (!n) return;
            props.onCreate(props.kind, props.kind === "scene" ? { location: n } : { name: n });
            setName("");
            setAdding(false);
          }}
        >
          <Input className="h-7 text-[12px]" autoFocus value={name} placeholder={c.addPlaceholder} onChange={/** e 取出 target.value 交给 setName。 */ (e) => setName(e.target.value)} onBlur={() => { if (!name.trim()) setAdding(false); }} />
          <Button size="sm" type="submit">{props.create}</Button>
        </form>
      ) : null}
      {props.items.length === 0 ? (
        <p className="px-3 py-2 text-[12px] text-muted">{c.noStills}</p>
      ) : (
        <div className="drama-look-grid">
          {props.items.map(/** a 判断是不是旁白，用 id 标出选中，有图就显示，点击把 id 交给 onFocus，并显示名字。 */ (a) => {
            const narrator = props.kind === "character" && isNarrator(a);
            return (
              <button
                key={a.id}
                type="button"
                className={cn("drama-look-card", props.focusId === a.id && "is-on")}
                aria-pressed={props.focusId === a.id}
                onClick={() => props.onFocus(props.kind, a.id)}
              >
                {a.image_url ? <img src={a.image_url} alt={assetName(a, props.kind)} /> : <span className="drama-look-empty grid place-items-center text-[11px]">{narrator ? c.narrator : "—"}</span>}
                <span className="truncate text-[12px] font-medium">{assetName(a, props.kind)}</span>
              </button>
            );
          })}
        </div>
      )}
    </section>
  );
}

/** CastInspector 编辑外观、造型、光线或提示词，并能上传、生成或删除。旁白不生成。 props 用 copy 取文案，kind 和 asset 判断旁白并填表，改字段交给 onSave，读到的图交给 onUpload，生成交给 onGen，删除交给 onDelete。 */
export function CastInspector(props: {
  kind: string;
  asset: Asset;
  copy: CopyT;
  onGen: () => void;
  onUpload: (b64: string) => void;
  onSave: (fields: Record<string, string>) => void;
  onDelete: () => void;
}) {
  const c = props.copy.video;
  const narrator = props.kind === "character" && isNarrator(props.asset);
  const a = props.asset;
  return (
    <div className="grid gap-2">
      <div className="text-[13px] font-medium">{assetName(a, props.kind)}</div>
      {narrator ? <p className="text-[12px] text-muted">{c.narrator}</p> : null}
      {props.kind === "character" ? (
        <>
          <Input className="h-7 text-[12px]" defaultValue={a.appearance || ""} placeholder={c.appearance} onBlur={/** e 取出 target.value，作为 appearance 交给 onSave。 */ (e) => props.onSave({ appearance: e.target.value })} />
          <Input className="h-7 text-[12px]" defaultValue={a.styling || ""} placeholder={c.styling} onBlur={/** e 取出 target.value，作为 styling 交给 onSave。 */ (e) => props.onSave({ styling: e.target.value })} />
        </>
      ) : null}
      {props.kind === "scene" ? (
        <Input className="h-7 text-[12px]" defaultValue={a.lighting || ""} placeholder={c.lighting} onBlur={/** e 取出 target.value，作为 lighting 交给 onSave。 */ (e) => props.onSave({ lighting: e.target.value })} />
      ) : null}
      {props.kind === "prop" ? (
        <Input className="h-7 text-[12px]" defaultValue={a.description || ""} placeholder={c.finalPrompt} onBlur={/** e 取出 target.value，作为 description 交给 onSave。 */ (e) => props.onSave({ description: e.target.value })} />
      ) : null}
      <Textarea className="min-h-[72px] rounded-lg border border-border bg-card px-2 py-1.5 text-[12px]" defaultValue={a.final_prompt || ""} placeholder={c.finalPrompt} onBlur={/** e 取出 target.value，作为 final_prompt 交给 onSave。 */ (e) => props.onSave({ final_prompt: e.target.value })} />
      <div className="flex flex-wrap items-center gap-1.5">
        <label className="grid size-7 cursor-pointer place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground" title={c.upload} aria-label={c.upload}>
          <ImagePlus className="size-3.5" />
          <input type="file" accept="image/*" className="hidden" onChange={/** e 取出第一个文件，读成 data URL 后交给 onUpload。 */ (e) => {
            const f = e.target.files?.[0];
            if (!f) return;
            const r = new FileReader();
            r.onload = () => props.onUpload(String(r.result || ""));
            r.readAsDataURL(f);
          }} />
        </label>
        {narrator ? null : <Button size="sm" variant="lift" onClick={props.onGen}>{c.generate}</Button>}
        <button type="button" className="ml-auto text-[11px] text-danger" onClick={props.onDelete}>{c.delete}</button>
      </div>
    </div>
  );
}

/** BoardInspector 改镜头标题、时长、提示词和绑定，并能再生成一条或设成主条。时长输入限制在 2 到 30。 props 用 copy 取文案，用 shot 和 bundle 里这条镜头已成功的视频任务来画，改字段交给 onPatch，再生成交给 onGen，选用成片交给 onApply。 */
export function BoardInspector(props: {
  bundle: Bundle;
  shot: Shot;
  copy: CopyT;
  onPatch: (s: Record<string, any>) => void;
  onGen: () => void;
  onApply: (id: string) => void;
}) {
  const c = props.copy.video;
  const s = props.shot;
  const takes = (props.bundle.jobs || []).filter(/** j 看是不是这条镜头上已经成功的视频任务。 */ (j) => j.storyboard_id === s.id && j.type === "video" && j.status === "succeeded");
  const hasCast = (s.character_ids || []).length > 0 || !!s.scene_id;
  const mentions = shotMentions(s, props.bundle);
  return (
    <div className="grid gap-2">
      <div className="flex items-center gap-2">
        <span className="font-mono text-[11px] tabular-nums text-muted">{String(s.shot_number).padStart(2, "0")}</span>
        <Input className="h-7 flex-1 text-[13px]" defaultValue={s.title} onBlur={/** e 取出 target.value 作为标题交给 onPatch。 */ (e) => props.onPatch({ id: s.id, title: e.target.value })} />
        <Input className="h-7 w-14 font-mono text-[12px] tabular-nums" type="number" min={2} max={30} defaultValue={s.duration} aria-label={c.durationLabel} onBlur={/** e 取出 target.value 转成时长，转不成就留原来的 duration。 */ (e) => props.onPatch({ id: s.id, duration: Number(e.target.value) || s.duration })} />
        <span className="text-[11px] text-muted">{c.seconds}</span>
      </div>
      <p className="text-[13px] leading-5 text-muted">{s.description}</p>
      {!hasCast ? <p className="text-[12px] text-muted">{c.missingCast}</p> : null}
      <DramaMentionField
        assets={mentions}
        defaultValue={s.video_prompt}
        placeholder={mentions.map(/** a 取出 name，前面加上 @。 */ (a) => `@${a.name}`).join(" ") || c.mention}
        onBlur={/** e 取出 target.value 作为视频提示交给 onPatch。 */ (e) => props.onPatch({ id: s.id, video_prompt: e.target.value })}
      />
      <label className="flex items-center gap-2 text-[12px]">
        <span className="w-16 text-muted">{c.scenes}</span>
        <select className={cn(field, "h-7 flex-1")} value={s.scene_id || ""} onChange={/** e 取出 target.value 作为 scene_id 交给 onPatch。 */ (e) => props.onPatch({ id: s.id, scene_id: e.target.value })}>
          <option value="">{c.unbind}</option>
          {(props.bundle.scenes || []).map(/** sc 用 id 当值和 key，用 location 当显示文字。 */ (sc) => <option key={sc.id} value={sc.id}>{sc.location}</option>)}
        </select>
      </label>
      <div>
        <div className="mb-1 text-[11px] text-muted">{c.characters}</div>
        <div className="flex flex-wrap gap-1.5">
          {(props.bundle.characters || []).filter(/** ch 留下仍关联且不是旁白的角色。 */ (ch) => ch.linked !== false && !isNarrator(ch)).map(/** ch 用 id 判断是否已绑定，点击就加进或移出 character_ids，并显示 name。 */ (ch) => {
            const on = (s.character_ids || []).includes(ch.id);
            return (
              <button
                key={ch.id}
                type="button"
                aria-pressed={on}
                className={cn("rounded-full px-2 py-0.5 text-[11px]", on ? "bg-foreground text-background" : "bg-lift text-muted")}
                onClick={() => {
                  const next = on ? (s.character_ids || []).filter(/** id 和这个角色的 id 不同才留下。 */ (id) => id !== ch.id) : [...(s.character_ids || []), ch.id];
                  props.onPatch({ id: s.id, character_ids: next });
                }}
              >{ch.name}</button>
            );
          })}
        </div>
      </div>
      <div>
        <div className="mb-1 text-[11px] text-muted">{c.props}</div>
        <div className="flex flex-wrap gap-1.5">
          {(props.bundle.props || []).filter(/** p 留下 linked 不是 false 的道具。 */ (p) => p.linked !== false).map(/** p 用 id 判断是否已绑定，点击就加进或移出 prop_ids，并显示 name。 */ (p) => {
            const on = (s.prop_ids || []).includes(p.id);
            return (
              <button
                key={p.id}
                type="button"
                aria-pressed={on}
                className={cn("rounded-full px-2 py-0.5 text-[11px]", on ? "bg-foreground text-background" : "bg-lift text-muted")}
                onClick={() => {
                  const next = on ? (s.prop_ids || []).filter(/** id 和这个道具的 id 不同才留下。 */ (id) => id !== p.id) : [...(s.prop_ids || []), p.id];
                  props.onPatch({ id: s.id, prop_ids: next });
                }}
              >{p.name}</button>
            );
          })}
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        {s.video_url ? <a className="text-[11px] underline" href={s.video_url} download>{c.download}</a> : null}
        <Button size="sm" variant="lift" onClick={props.onGen}>{c.anotherTake}</Button>
        {s.video_hash ? (
          <Button size="sm" variant="ghost" data-testid="drama-shot-send-to-cut" onClick={() => requestSendToCut({ hash: s.video_hash, title: s.title || c.conductor.cutTitle })}>
            {c.conductor.sendToCut}
          </Button>
        ) : null}
      </div>
      <div>
        <div className="mb-1 text-[11px] text-muted">{c.history}</div>
        {takes.length === 0 ? <p className="text-[11px] text-muted">{c.noHistory}</p> : (
          <div className="flex flex-wrap gap-2">
            {takes.map(/** j 用 id 交给 onApply，有 poster_url 就显示成图。 */ (j) => (
              <button key={j.id} type="button" className="w-[72px] text-left" onClick={() => props.onApply(j.id)}>
                {j.poster_url ? <img src={j.poster_url} alt="" className="h-12 w-full rounded object-cover" /> : <div className="h-12 rounded bg-lift" />}
                <span className="text-[10px] text-muted">{c.setMain}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

/** CutInspector 选择要合成的镜头。没有 ffmpeg 时显示安装进度或缺失。 props 用 copy 取文案，用 bundle 的镜头和成片哈希统计选中，用 sel 数已选并用 setSel 全选或清空，没有 ffmpeg 时用 ffmpegInstall 的阶段和进度显示安装，有成片就把哈希和标题交给 onSendToCut。onInstall 函数体没有读取。 */
export function CutInspector(props: {
  bundle: Bundle;
  sel: Record<string, boolean>;
  setSel: (v: Record<string, boolean>) => void;
  copy: CopyT;
  ffmpeg: boolean;
  ffmpegInstall?: { phase?: string; percent?: number; error?: string };
  onInstall: () => void;
  onSendToCut?: (hash: string, title: string) => void;
}) {
  const c = props.copy.video;
  const shots = props.bundle.shots || [];
  const n = selectedClipIds(shots, props.sel).length;
  const phase = String(props.ffmpegInstall?.phase || "");
  const installing = phase === "resolving" || phase === "downloading" || phase === "extracting" || phase === "verifying";
  const epHash = String(props.bundle.episode?.video_hash || "").trim();
  const shotHash = String(shots.find(/** s 留下没有被明确取消且有 video_hash 的镜头。 */ (s) => props.sel[s.id] !== false && s.video_hash)?.video_hash || "").trim();
  const hash = epHash || shotHash;
  const title = props.bundle.episode?.title || props.bundle.drama?.title || c.conductor.cutTitle;
  return (
    <div>
      {!props.ffmpeg ? (
        <p className="mb-3 text-[13px] text-danger">
          {installing ? `${c.ffmpegInstalling} ${Math.max(0, Math.min(100, Math.round(Number(props.ffmpegInstall?.percent) || 0)))}%` : phase === "error" ? c.ffmpegInstallFailed : c.ffmpegMissing}
        </p>
      ) : null}
      {installing ? <ProgressHairline className="mb-3" value={Number(props.ffmpegInstall?.percent) || 0} /> : null}
      <div className="mb-3 flex flex-wrap items-center gap-2 text-[12px] text-muted">
        <span className="tabular-nums">{n} {c.clipCount}</span>
        <button type="button" className="underline" onClick={() => {
          const next: Record<string, boolean> = {};
          for (const s of shots) next[s.id] = !!s.video_url;
          props.setSel(next);
        }}>{c.selectAll}</button>
        <button type="button" className="underline" onClick={() => {
          const next: Record<string, boolean> = {};
          for (const s of shots) next[s.id] = false;
          props.setSel(next);
        }}>{c.selectNone}</button>
      </div>
      {props.onSendToCut ? (
        <Button
          size="sm"
          variant="default"
          className="mt-1"
          data-testid="drama-send-to-cut"
          disabled={!hash}
          title={c.conductor.sendToCutHint}
          onClick={() => { if (hash) props.onSendToCut?.(hash, title); }}
        >{c.conductor.sendToCut}</Button>
      ) : null}
    </div>
  );
}

/** JobDrawer 只列出仍在动或已经失败的任务，并按集和目标分组。 props 从 jobs 里留下仍在动或已失败的，用 copy 取文案，用 bundle 和 episodes 配说明，重试交给 onRetry，取消交给 onCancel。 */
export function JobDrawer(props: { jobs: Job[]; bundle: Bundle | null; episodes?: { id: string; title: string; episode_number?: number }[]; copy: CopyT; onRetry: (id: string) => void; onCancel: (id: string) => void }) {
  const live = props.jobs.filter(/** j 看状态是不是仍在动，或已经失败。 */ (j) => jobMoving(j.status) || j.status === "failed");
  const c = props.copy.video;
  if (!live.length) {
    return <p className="drama-task-empty">{c.jobsIdle}</p>;
  }
  const label: Record<string, string> = { queued: c.queued, running: c.running, polling: c.polling, succeeded: c.succeeded, failed: c.failed };
  const groups: { key: string; caption: string; jobs: Job[] }[] = [];
  const seen = new Map<string, number>();
  for (const j of live) {
        const key = `${j.episode_id || ""}:${j.storyboard_id || j.character_id || j.scene_id || j.prop_id || j.id}`;
    const at = seen.get(key);
    if (at === undefined) {
      seen.set(key, groups.length);
      groups.push({ key, caption: jobCaption(j, props.bundle, props.episodes, c), jobs: [j] });
    } else {
      groups[at].jobs.push(j);
    }
  }
  return (
    <div className="drama-task-list">
      {groups.map(/** g 用 key 当分组 key，显示 caption，并列出 jobs。 */ (g) => (
        <div key={g.key} className="drama-task-group">
          <div className="drama-task-group-head">{g.caption}</div>
          {g.jobs.map(/** j 用 status 判断是否在动或失败并显示文案，有 error 就显示，失败时把 id 交给 onRetry，在动时把 id 交给 onCancel。 */ (j) => {
            const moving = jobMoving(j.status);
            return (
              <div key={j.id} className="relative flex items-center gap-2 rounded-lg bg-lift px-2 py-1.5 pb-2 text-[11px]">
                {moving ? <span className="pulse-dot" /> : null}
                <span className={j.status === "failed" ? "text-danger" : "text-muted"}>{label[j.status] || j.status}</span>
                {j.error ? <span className="min-w-0 truncate text-danger">{j.error}</span> : null}
                {j.status === "failed" ? <button type="button" className="underline" onClick={() => props.onRetry(j.id)}>{c.retry}</button> : null}
                {moving ? <button type="button" className="underline" onClick={() => props.onCancel(j.id)}>{c.cancel}</button> : null}
                {moving ? <ProgressHairline className="absolute inset-x-0 bottom-0 rounded-none" indeterminate /> : null}
              </div>
            );
          })}
        </div>
      ))}
    </div>
  );
}

/** 给任务配上一行说明：镜头序号、定妆名字，或只写种类。 j 用 episode_id、storyboard_id、character_id、scene_id、prop_id 和 type 配说明。bundle 用来对照当前集，并按镜头、角色、场景、道具找对应项。episodes 按任务的 episode_id 找集。c 取未命名、镜头和定妆文案。 */
function jobCaption(j: Job, bundle: Bundle | null, episodes: { id: string; title: string; episode_number?: number }[] | undefined, c: CopyT["video"]) {
  const ep = (episodes || []).find(/** e 用 id 和任务的 episode_id 比对。 */ (e) => e.id === j.episode_id);
  const epLabel = ep && ep.id !== bundle?.episode?.id ? `${ep.title || c.untitledEp} · ` : "";
  const shot = (bundle?.shots || []).find(/** s 用 id 和任务的 storyboard_id 比对。 */ (s) => s.id === j.storyboard_id);
  if (shot) return `${epLabel}${c.jobClip} · ${String(shot.shot_number).padStart(2, "0")}`;
  const ch = (bundle?.characters || []).find(/** a 用 id 和任务的 character_id 比对。 */ (a) => a.id === j.character_id);
  if (ch) return `${epLabel}${c.jobLook} · ${ch.name}`;
  const sc = (bundle?.scenes || []).find(/** a 用 id 和任务的 scene_id 比对。 */ (a) => a.id === j.scene_id);
  if (sc) return `${epLabel}${c.jobLook} · ${sc.location}`;
  const p = (bundle?.props || []).find(/** a 用 id 和任务的 prop_id 比对。 */ (a) => a.id === j.prop_id);
  if (p) return `${epLabel}${c.jobLook} · ${p.name}`;
  if (epLabel) return `${epLabel}${j.type === "video" ? c.jobClip : c.jobLook}`;
  return j.type === "video" ? c.jobClip : c.jobLook;
}

/** EmptyDesk 在没有集时列出最多 8 条已有系列，或提示先起名。 props 用 copy 取文案，dramas 最多列 8 条，点选交给 onPick，起名交给 onNameSeries。 */
export function EmptyDesk(props: {
  copy: CopyT;
  dramas?: Drama[];
  onNameSeries: () => void;
  onPick?: (id: string) => void;
}) {
  const c = props.copy.video;
  const existing = (props.dramas || []).slice(0, 8);
  return (
    <div className="drama-empty-desk" data-testid="drama-empty-desk">
      <p className="drama-empty-kicker">{existing.length ? c.continueSeries : c.empty}</p>
      <p className="drama-empty-body">{existing.length ? c.emptyPickExisting : c.emptyHint}</p>
      {existing.length ? (
        <div className="drama-empty-series">
          {existing.map(/** d 用 id 交给 onPick，并显示标题。 */ (d) => (
            <button
              key={d.id}
              type="button"
              className="drama-empty-series-chip"
              onClick={() => props.onPick?.(d.id)}
            >
              {shownSeriesTitle(d.title, c.untitled)}
            </button>
          ))}
        </div>
      ) : null}
      <button type="button" className="drama-empty-link" onClick={props.onNameSeries}>
        {c.nameFirst}
      </button>
    </div>
  );
}

/** EpisodeMap 列出分集方案，可以确认、拆开、合并或抽设定。 props 用 copy 取文案，用 plan 的集列出方案，busy 时禁用按钮，圣经交给 onBible，确认交给 onConfirm，拆开和合并按集序号交给 onSplit 和 onMerge。 */
export function EpisodeMap(props: {
  plan: EpisodePlan;
  copy: CopyT;
  busy?: boolean;
  onConfirm: () => void;
  onSplit: (n: number) => void;
  onMerge: (n: number) => void;
  onBible: () => void;
}) {
  const c = props.copy.video;
  const rows = props.plan.episodes || [];
  return (
    <div className="drama-map" data-testid="drama-episode-map">
      <div className="drama-map-head">
        <div>
          <h2>{c.mapTitle}</h2>
          <p>{c.mapHint}</p>
        </div>
        <div className="drama-map-actions">
          <Button type="button" variant="ghost" size="sm" disabled={!!props.busy} onClick={props.onBible}>{c.mapBible}</Button>
          <Button type="button" size="sm" data-testid="drama-map-confirm" disabled={!!props.busy || rows.length === 0} onClick={props.onConfirm}>{c.mapConfirm}</Button>
        </div>
      </div>
      {rows.length === 0 ? (
        <p className="drama-map-empty">{c.mapEmpty}</p>
      ) : (
        <ol className="drama-map-list">
          {rows.map(/** row 显示序号、标题、logline、hook、目标秒数和起止字数，并把 n 交给拆开或合并。i 还没到最后一条才显示合并。 */ (row, i) => (
            <li key={`${row.n}-${row.start}`} className="drama-map-row">
              <div className="drama-map-n">{String(row.n).padStart(2, "0")}</div>
              <div className="drama-map-copy">
                <strong>{row.title || c.untitledEp}</strong>
                {row.logline ? <span>{row.logline}</span> : null}
                {row.hook ? <em>{c.mapHook} · {row.hook}</em> : null}
              </div>
              <div className="drama-map-meta">
                <span>{row.target_seconds || 0}{c.seconds}</span>
                <span>{Math.max(0, (row.end || 0) - (row.start || 0))}{c.chars}</span>
              </div>
              <div className="drama-map-row-actions">
                <button type="button" disabled={!!props.busy} onClick={() => props.onSplit(row.n)}>{c.mapSplit}</button>
                {i < rows.length - 1 ? <button type="button" disabled={!!props.busy} onClick={() => props.onMerge(row.n)}>{c.mapMerge}</button> : null}
              </div>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}
