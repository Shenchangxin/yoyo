/**
 * Official spine-ts 3.6 WebGL runtime: `public/vendor/spine-3.6/spine-webgl.js`.
 * It is loaded with a script tag and used through `window.spine`. The types below
 * cover only the calls the stand makes.
 */

export const SPINE_RUNTIME_URL = "/vendor/spine-3.6/spine-webgl.js";

let loading: Promise<SpineRuntime> | null = null;

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

export type ManagedWebGLRenderingContext = { gl: WebGLRenderingContext | null };

export type SceneRenderer = {
  camera: { position: { x: number; y: number }; setViewport(width: number, height: number): void };
  begin(): void;
  drawSkeleton(skeleton: Skeleton, premultipliedAlpha: boolean): void;
  end(): void;
  dispose(): void;
};

export type AssetManager = {
  assets: Record<string, string | { dispose?: () => void } | null>;
  loadText(path: string, success: () => void, error: (path: string, message: string) => void): void;
  loadTextureAtlas(path: string, success: () => void, error: (path: string, message: string) => void): void;
  get(path: string): unknown;
  dispose(): void;
};

export type TextureAtlas = { readonly __brand: "TextureAtlas" };
type AttachmentLoader = { readonly __brand: "AttachmentLoader" };

export type SkeletonJson = { readSkeletonData(json: string): SkeletonData };

export type SkeletonData = {
  animations: Animation[];
  findSkin(name: string): object | null;
  findAnimation(name: string): Animation | null;
};

export type Animation = { name: string; duration: number; timelines: object[] };

export type EventTimeline = { events: SpineEvent[] };

export type SpineEvent = { time: number; stringValue: string | null; data: { name: string; stringValue: string | null } };

export type Attachment = { name: string };

export type ClippingAttachment = Attachment & { endSlot: SlotData | null };

export type SlotData = { index: number };

export type Slot = {
  data: SlotData;
  getAttachment(): Attachment | null;
  setAttachment(attachment: Attachment | null): void;
};

export type Skeleton = {
  slots: Slot[];
  drawOrder: Slot[];
  setSkinByName(name: string): void;
  setToSetupPose(): void;
  updateWorldTransform(): void;
  getBounds(offset: Vector2, size: Vector2, temp: number[]): void;
};

export type Vector2 = { x: number; y: number };

export type AnimationStateData = { defaultMix: number };

export type AnimationState = {
  setAnimation(track: number, name: string, loop: boolean): TrackEntry;
  addAnimation(track: number, name: string, loop: boolean, delay: number): TrackEntry;
  setEmptyAnimation(track: number, mixDuration: number): TrackEntry;
  addEmptyAnimation(track: number, mixDuration: number, delay: number): TrackEntry;
  getCurrent(track: number): TrackEntry | null;
  update(delta: number): void;
  apply(skeleton: Skeleton): boolean;
};

export type TrackEntry = {
  animation: Animation;
  loop: boolean;
  trackTime: number;
  animationStart: number;
  animationEnd: number;
  mixDuration: number;
  listener: { complete?: (entry: TrackEntry) => void } | null;
};
