/**
 * 拥有：官方 spine-ts 3.6 WebGL 运行时的加载，以及立绘实际调用到的类型。
 * 不拥有：运行时源码和立绘播放。脚本在 public/vendor/spine-3.6/spine-webgl.js，用 script 标签加载，经 window.spine 使用。下面的类型只覆盖立绘调用到的部分。不改服务端。
 */

/** SPINE_RUNTIME_URL 运行时脚本的地址。 */
export const SPINE_RUNTIME_URL = "/vendor/spine-3.6/spine-webgl.js";

let loading: Promise<SpineRuntime> | null = null;

/** loadSpineRuntime 加载脚本并解析 window.spine。已经在窗口上则直接返回。失败时清掉进行中的加载，以便下次再试。 */
export function loadSpineRuntime(): Promise<SpineRuntime> {
  const loaded = (window as SpineWindow).spine;
  if (loaded) return Promise.resolve(loaded);
  loading ??= new Promise<SpineRuntime>((resolve, reject) => {
    const script = document.createElement("script");
    script.src = SPINE_RUNTIME_URL;
    script.async = true;
    script.onload = () => {
      const spine = (window as SpineWindow).spine;
      if (spine) resolve(spine);
      else reject(new Error(`${SPINE_RUNTIME_URL} did not set window.spine`));
    };
    script.onerror = () => {
      script.remove();
      reject(new Error(`could not load ${SPINE_RUNTIME_URL}`));
    };
    document.head.appendChild(script);
  }).catch((error: unknown) => {
    loading = null;
    throw error;
  });
  return loading;
}

type SpineWindow = Window & { spine?: SpineRuntime };

type Class<T, Args extends unknown[] = never[]> = new (...args: Args) => T;

/** SpineRuntime 立绘用到的 spine-ts 3.6 入口：WebGL、骨骼、动画状态和附件类。 */
export type SpineRuntime = {
  webgl: {
    ManagedWebGLRenderingContext: Class<ManagedWebGLRenderingContext, [HTMLCanvasElement, WebGLContextAttributes]>;
    SceneRenderer: Class<SceneRenderer, [HTMLCanvasElement, ManagedWebGLRenderingContext, boolean]>;
    AssetManager: Class<AssetManager, [ManagedWebGLRenderingContext]>;
  };
  AtlasAttachmentLoader: Class<AttachmentLoader, [TextureAtlas]>;
  SkeletonJson: Class<SkeletonJson, [AttachmentLoader]>;
  Skeleton: Class<Skeleton, [SkeletonData]>;
  AnimationStateData: Class<AnimationStateData, [SkeletonData]>;
  AnimationState: Class<AnimationState, [AnimationStateData]>;
  Vector2: Class<Vector2, []>;
  ClippingAttachment: Class<ClippingAttachment>;
  RegionAttachment: Class<Attachment>;
  MeshAttachment: Class<Attachment>;
  EventTimeline: Class<EventTimeline>;
};

/** ManagedWebGLRenderingContext 画布上的 WebGL 上下文包装。gl 可以空。 */
export type ManagedWebGLRenderingContext = { gl: WebGLRenderingContext | null };

/** SceneRenderer 是相机、开始绘制、画骨架和释放。 */
export type SceneRenderer = {
  camera: { position: { x: number; y: number }; setViewport(width: number, height: number): void };
  begin(): void;
  drawSkeleton(skeleton: Skeleton, premultipliedAlpha: boolean): void;
  end(): void;
  dispose(): void;
};

/** AssetManager 按路径加载文本和图集，并按路径取出或释放。 */
export type AssetManager = {
  assets: Record<string, string | { dispose?: () => void } | null>;
  loadText(path: string, success: () => void, error: (path: string, message: string) => void): void;
  loadTextureAtlas(path: string, success: () => void, error: (path: string, message: string) => void): void;
  get(path: string): unknown;
  dispose(): void;
};

/** TextureAtlas 图集标记。立绘只把它交给附件加载器。 */
export type TextureAtlas = { readonly __brand: "TextureAtlas" };
type AttachmentLoader = { readonly __brand: "AttachmentLoader" };

/** SkeletonJson 从 JSON 文本读出骨骼数据。 */
export type SkeletonJson = { readSkeletonData(json: string): SkeletonData };

/** SkeletonData 骨骼数据：动画表、按名找皮肤和按名找动画。 */
export type SkeletonData = {
  animations: Animation[];
  findSkin(name: string): object | null;
  findAnimation(name: string): Animation | null;
};

/** Animation 是一条动画的名字、时长和时间线。 */
export type Animation = { name: string; duration: number; timelines: object[] };

/** EventTimeline 一条动画事件时间线。 */
export type EventTimeline = { events: SpineEvent[] };

/** SpineEvent 时间线上的一个事件，带时刻和字符串值。 */
export type SpineEvent = { time: number; stringValue: string | null; data: { name: string; stringValue: string | null } };

/** Attachment 是槽位上的一个附件。 */
export type Attachment = { name: string };

/** ClippingAttachment 裁剪附件。endSlot 是裁剪结束的槽。 */
export type ClippingAttachment = Attachment & { endSlot: SlotData | null };

/** SlotData 是槽位的静态数据。 */
export type SlotData = { index: number };

/** Slot 运行时槽，可读可写当前附件。 */
export type Slot = {
  data: SlotData;
  getAttachment(): Attachment | null;
  setAttachment(attachment: Attachment | null): void;
};

/** Skeleton 骨骼实例：皮肤、设置姿势、世界变换和包围盒。 */
export type Skeleton = {
  slots: Slot[];
  drawOrder: Slot[];
  setSkinByName(name: string): void;
  setToSetupPose(): void;
  updateWorldTransform(): void;
  getBounds(offset: Vector2, size: Vector2, temp: number[]): void;
};

/** Vector2 是一个二维点。 */
export type Vector2 = { x: number; y: number };

/** AnimationStateData 动画混合的默认时长。 */
export type AnimationStateData = { defaultMix: number };

/** AnimationState 在各轨道上设动画、推进并套到骨架上。 */
export type AnimationState = {
  setAnimation(track: number, name: string, loop: boolean): TrackEntry;
  addAnimation(track: number, name: string, loop: boolean, delay: number): TrackEntry;
  setEmptyAnimation(track: number, mixDuration: number): TrackEntry;
  addEmptyAnimation(track: number, mixDuration: number, delay: number): TrackEntry;
  getCurrent(track: number): TrackEntry | null;
  update(delta: number): void;
  apply(skeleton: Skeleton): boolean;
};

/** TrackEntry 某一轨上正在播的条目，含循环、时间和结束监听。 */
export type TrackEntry = {
  animation: Animation;
  loop: boolean;
  trackTime: number;
  animationStart: number;
  animationEnd: number;
  mixDuration: number;
  listener: { complete?: (entry: TrackEntry) => void } | null;
};
