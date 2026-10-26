/**
 * 拥有：把一套 Spine 服装画到画布上，并按 pickAnimations 换片段。
 * 不拥有：服装预热、通话状态和对话模型。这里的循环只推进骨骼并绘制，不调用模型。不改服务端。
 */

import { CALM, pickAnimations, type Emotion, type StandDirection } from "./animations";
import { FrameGate, MAX_PIXEL_RATIO, pixelRatio } from "./frames";
import type {
  AnimationState,
  AssetManager,
  ClippingAttachment,
  SceneRenderer,
  Skeleton,
  SkeletonData,
  Slot,
  SlotData,
  SpineRuntime,
  TextureAtlas,
  TrackEntry,
} from "./spineRuntime";

/** Framing 是取景：膝盖、腰或全身。 */
export type Framing = "knees" | "waist" | "full";
/** Outfit 一套服装的 id 和资源目录。 */
export type Outfit = { id: string; url: string };

const FRAMING_HEIGHT: Record<Exclude<Framing, "full">, number> = { knees: 0.85, waist: 0.66 };
const MIN_WIDTH = 0.7;
const HEADROOM = 0.03;
const TRACK = { body: 0, face: 1, nod: 2, eye: 3, mouth: 5 } as const;
const MIX_SECONDS = 0.3;
const MOUTH_MIX_SECONDS = 0.1;
const MAX_STEP_SECONDS = 0.1;
const UNPACK_PREMULTIPLY_ALPHA_WEBGL = 0x9241;

/** StandUnavailable 画布没有可用的 WebGL 时抛出。 */
export class StandUnavailable extends Error {}

type Pose = {
  url: string;
  skeleton: Skeleton;
  data: SkeletonData;
  state: AnimationState;
  animations: ReadonlyMap<string, number>;
  bounds: { x: number; y: number; width: number; height: number };
  clips: Map<ClippingAttachment, Clip>;
  shown: Shown;
};

type Shown = {
  mood: Emotion | null;
  face: string | null;
  mouth: string | null;
  eye: string | null;
  nod: number | null;
};

type Clip = { slot: Slot; end: SlotData; detached: boolean };

/** StandEngine 把一套 Spine 服装画到画布上。片段名来自 pickAnimations。 */
export class StandEngine {
  private readonly gl: WebGLRenderingContext;
  private readonly renderer: SceneRenderer;
  private readonly assets: AssetManager;
  private readonly listeners = new AbortController();
  private readonly gate = new FrameGate();
  private pose: Pose | null = null;
  private raf = 0;
  private last = 0;
  private paused = false;
  private still = false;
  private lost = false;
  private disposed = false;
  private direction: StandDirection = CALM;
  private reducedMotion = false;
  private loads = 0;
  private wanted: string | null = null;
  private readonly fetching = new Map<string, Promise<void>>();

  /**
   * constructor 绑定上下文丢失与恢复，并创建场景渲染器和资源管理器。没有 WebGL 就抛 StandUnavailable。
   * canvas 调用 addEventListener，传给 spine.webgl.ManagedWebGLRenderingContext，传给 spine.webgl.SceneRenderer。
   * spine 读取 webgl。
   * framing 记到 this.framing。
   * maxPixelRatio 记到 this.maxPixelRatio。
   */
  constructor(
    private readonly canvas: HTMLCanvasElement,
    private readonly spine: SpineRuntime,
    public framing: Framing,
    private readonly maxPixelRatio: number = MAX_PIXEL_RATIO,
  ) {
    const signal = this.listeners.signal;
    canvas.addEventListener("webglcontextlost", () => {
      this.lost = true;
      this.stop();
    }, { signal });
    canvas.addEventListener("webglcontextrestored", () => {
      this.gl.pixelStorei(UNPACK_PREMULTIPLY_ALPHA_WEBGL, true);
      this.lost = false;
      this.start();
    }, { signal });
    const context = new spine.webgl.ManagedWebGLRenderingContext(canvas, { alpha: true, premultipliedAlpha: true });
    if (!context.gl) {
      this.listeners.abort();
      throw new StandUnavailable("WebGL is unavailable");
    }
    this.gl = context.gl;
    this.lost = this.gl.isContextLost();
    this.gl.pixelStorei(UNPACK_PREMULTIPLY_ALPHA_WEBGL, true);
    this.renderer = new spine.webgl.SceneRenderer(canvas, context, true);
    this.assets = new spine.webgl.AssetManager(context);
  }

  /** direct 记下方向和是否减弱动效，已有姿势就跟着换。direction 记到 this.direction。reducedMotion 记到 this.reducedMotion。 */
  direct(direction: StandDirection, reducedMotion: boolean): void {
    this.direction = direction;
    this.reducedMotion = reducedMotion;
    if (this.pose) this.follow(this.pose);
  }

  /** load 拉取并读入服装。同一地址已在场上则返回。票过期或引擎已释放则放弃，并按需要丢掉资源。outfit 解开 url。 */
  async load(outfit: Outfit): Promise<void> {
    const { url } = outfit;
    const ticket = ++this.loads;
    this.wanted = url;
    if (this.pose?.url === url) return;
    try {
      await this.fetchOutfit(url);
    } catch (error) {
      if (this.disposed) this.assets.dispose();
      else this.release(url);
      throw error;
    }
    if (this.disposed) {
      this.assets.dispose();
      return;
    }
    if (ticket !== this.loads) {
      if (this.wanted !== url) this.release(url);
      return;
    }
    let pose: Pose;
    try {
      pose = this.read(url);
    } catch (error) {
      this.release(url);
      throw error;
    }
    const previous = this.pose;
    this.pose = pose;
    if (previous) this.release(previous.url);
    this.follow(this.pose);
    this.start();
  }

  /** 用 data.json 和 data.atlas 建骨骼、动画状态和包围盒。有 normal 皮肤就穿上。url 拼进字符串，写入字段 url。 */
  private read(url: string): Pose {
    const json = `${url}data.json`;
    const atlas = `${url}data.atlas`;
    const spine = this.spine;
    const loader = new spine.AtlasAttachmentLoader(this.assets.get(atlas) as TextureAtlas);
    const data = new spine.SkeletonJson(loader).readSkeletonData(this.assets.get(json) as string);
    const skeleton = new spine.Skeleton(data);
    if (data.findSkin("normal")) skeleton.setSkinByName("normal");
    skeleton.setToSetupPose();
    skeleton.updateWorldTransform();
    const offset = new spine.Vector2();
    const size = new spine.Vector2();
    skeleton.getBounds(offset, size, []);
    const stateData = new spine.AnimationStateData(data);
    stateData.defaultMix = MIX_SECONDS;
    return {
      url,
      skeleton,
      data,
      state: new spine.AnimationState(stateData),
      animations: new Map(data.animations.map((a) => [a.name, a.duration])),
      bounds: { x: offset.x, y: offset.y, width: size.x, height: size.y },
      clips: new Map(),
      shown: { mood: null, face: null, mouth: null, eye: null, nod: this.direction.nod?.id ?? null },
    };
  }

  /** 同一地址只拉一次 JSON 和图集。任一失败则整次失败，结束后从进行中的表里去掉。url 拼进字符串，传给 fetching.get，传给 fetching.delete，交给 fetching.set。 */
  private fetchOutfit(url: string): Promise<void> {
    let fetching = this.fetching.get(url);
    if (!fetching) {
      fetching = Promise.allSettled([
        new Promise<void>((resolve, reject) =>
          this.assets.loadText(`${url}data.json`, () => resolve(), (_, message) => reject(new Error(message))),
        ),
        new Promise<void>((resolve, reject) =>
          this.assets.loadTextureAtlas(`${url}data.atlas`, () => resolve(), (_, message) => reject(new Error(message))),
        ),
      ])
        .then((results) => {
          for (const result of results) if (result.status === "rejected") throw result.reason;
        })
        .finally(() => this.fetching.delete(url));
      this.fetching.set(url, fetching);
    }
    return fetching;
  }

  /** 丢掉该地址前缀下已加载的资源。对象资源先 dispose。url 传给 path.startsWith。 */
  private release(url: string): void {
    const assets = this.assets.assets;
    for (const path of Object.keys(assets)) {
      if (!path.startsWith(url)) continue;
      const asset = assets[path];
      if (typeof asset === "object" && asset) asset.dispose?.();
      delete assets[path];
    }
  }

  /** setStill 记下画完这一帧之后是否停住。still 记到 this.still。 */
  setStill(still: boolean): void {
    this.still = still;
  }

  /** setPaused 暂停时停帧，恢复时再开。paused 为真时才继续，记到 this.paused。 */
  setPaused(paused: boolean): void {
    this.paused = paused;
    if (paused) this.stop();
    else this.start();
  }

  /** dispose 停帧、卸监听、丢掉姿势并释放资源和渲染器。重复调用直接返回。 */
  dispose(): void {
    if (this.disposed) return;
    this.disposed = true;
    this.stop();
    this.listeners.abort();
    this.pose = null;
    this.assets.dispose();
    this.renderer.dispose();
  }

  /** follow 按挑好的片段换身体、脸、嘴、眼睛和点头。pose 读取 animations，解开 state、shown，传给 playBody。 */
  private follow(pose: Pose): void {
    const { state, shown } = pose;
    const want = pickAnimations(this.direction, pose.animations, this.reducedMotion);
    if (shown.mood !== this.direction.mood) {
      shown.mood = this.direction.mood;
      this.playBody(pose, want.body);
    }
    if (shown.face !== want.face) {
      shown.face = want.face;
      if (pose.animations.has(want.face)) state.setAnimation(TRACK.face, want.face, true);
      else state.setEmptyAnimation(TRACK.face, MIX_SECONDS);
    }
    const mouth = want.mouth && `${want.mouth.name} ${want.mouth.open ? "open" : "shut"}`;
    if (want.mouth && shown.mouth !== mouth) {
      shown.mouth = mouth;
      const entry = state.setAnimation(TRACK.mouth, want.mouth.name, want.mouth.open);
      entry.mixDuration = MOUTH_MIX_SECONDS;
      if (!want.mouth.open) entry.animationEnd = 0;
    }
    if (shown.eye !== want.eye) {
      shown.eye = want.eye;
      if (want.eye) state.setAnimation(TRACK.eye, want.eye, true);
      else state.setEmptyAnimation(TRACK.eye, MIX_SECONDS);
    }
    if (want.nod && shown.nod !== want.nod.id) {
      shown.nod = want.nod.id;
      const nod = state.setAnimation(TRACK.nod, want.nod.name, false);
      state.addEmptyAnimation(TRACK.nod, MIX_SECONDS, nod.animation.duration);
    }
  }

  /**
   * 播放身体片段。当前动画带 relay 时先播接力再接到目标。目标有 loop_start 时，第一次播完后从该时刻循环。没有该片段则不动。
   * state 调用 getCurrent，调用 setAnimation，调用 addAnimation。
   * data 传给 event。
   * animations 调用 has。
   * name 与 current?.animation.name 比较是否相等，传给 animations.has，传给 state.addAnimation，传给 state.setAnimation，传给 event。
   */
  private playBody({ state, data, animations }: Pose, name: string): void {
    if (!animations.has(name)) return;
    const current = state.getCurrent(TRACK.body);
    if (current?.animation.name === name) return;
    const relay = current && this.event(data, current.animation.name, "relay");
    let entry: TrackEntry;
    if (relay && animations.has(relay.value)) {
      state.setAnimation(TRACK.body, relay.value, false);
      entry = state.addAnimation(TRACK.body, name, false, 0);
    } else {
      entry = state.setAnimation(TRACK.body, name, false);
    }
    const loopStart = this.event(data, name, "loop_start");
    if (!loopStart) {
      entry.loop = true;
      return;
    }
    entry.listener = {
      /** complete 第一次播完后从循环起点接着播。played 读取 loop、animationStart、trackTime。 */
      complete: (played) => {
        if (played.loop) return;
        played.loop = true;
        played.animationStart = loopStart.time;
        played.trackTime = 0;
      },
    };
  }

  /** 在指定动画的事件时间线里找同名事件，返回时刻和字符串值。没有则空。data 调用 findAnimation。name 传给 data.findAnimation。eventName 与 event.data.name 比较是否相等。 */
  private event(data: SkeletonData, name: string, eventName: string): { time: number; value: string } | null {
    for (const timeline of data.findAnimation(name)?.timelines ?? []) {
      if (!(timeline instanceof this.spine.EventTimeline)) continue;
      for (const event of timeline.events) {
        if (event.data.name === eventName) {
          return { time: event.time, value: event.stringValue || event.data.stringValue || "" };
        }
      }
    }
    return null;
  }

  /** 有姿势、未暂停、未丢上下文、未释放时才请求下一帧。已有帧编号则不重复申请。 */
  private start(): void {
    if (this.raf || !this.pose || this.paused || this.lost || this.disposed) return;
    this.last = 0;
    this.raf = requestAnimationFrame(this.frame);
  }

  /** stop 取消当前的帧请求。 */
  private stop(): void {
    cancelAnimationFrame(this.raf);
    this.raf = 0;
  }

  /** frame 到点就画一帧，静止时画完即停。now 减去 this.last，记到 this.last，传给 gate.shouldDraw。 */
  private readonly frame = (now: number): void => {
    this.raf = requestAnimationFrame(this.frame);
    if (!this.gate.shouldDraw(now)) return;
    const step = this.last ? Math.min((now - this.last) / 1000, MAX_STEP_SECONDS) : 0;
    this.last = now;
    if (this.pose) this.draw(this.pose, step);
    if (this.still) this.stop();
  };

  /** 按像素比调整画布，推进动画，补上裁剪结尾，对好镜头并画出骨骼。pose 读取 state、skeleton，传给 keepClipsEnding，传给 aim。step 传给 pose.state.update。 */
  private draw(pose: Pose, step: number): void {
    const { canvas, gl, renderer } = this;
    const ratio = pixelRatio(window.devicePixelRatio, this.maxPixelRatio);
    const width = Math.max(1, Math.round(canvas.clientWidth * ratio));
    const height = Math.max(1, Math.round(canvas.clientHeight * ratio));
    if (canvas.width !== width || canvas.height !== height) {
      canvas.width = width;
      canvas.height = height;
    }
    gl.viewport(0, 0, width, height);
    gl.clearColor(0, 0, 0, 0);
    gl.clear(gl.COLOR_BUFFER_BIT);
    pose.state.update(step);
    pose.state.apply(pose.skeleton);
    pose.skeleton.updateWorldTransform();
    this.keepClipsEnding(pose);
    this.aim(pose, width / height);
    renderer.begin();
    renderer.drawSkeleton(pose.skeleton, true);
    renderer.end();
  }

  /** 全身取景把角色放进画面并留顶空。膝上或腰上取景从顶部往下裁，并保证一定宽度。bounds 读取 height、width、x、y。aspect 除以 bounds.width，乘以 viewHeight。 */
  private aim({ bounds }: Pose, aspect: number): void {
    const camera = this.renderer.camera;
    if (this.framing === "full") {
      const viewHeight = Math.max(bounds.height, bounds.width / aspect) * (1 + 2 * HEADROOM);
      camera.position.x = bounds.x + bounds.width / 2;
      camera.position.y = bounds.y + bounds.height / 2;
      camera.setViewport(viewHeight * aspect, viewHeight);
      return;
    }
    const viewHeight = Math.max(bounds.height * FRAMING_HEIGHT[this.framing], (bounds.width * MIN_WIDTH) / aspect);
    const top = bounds.y + bounds.height * (1 + HEADROOM);
    camera.position.x = bounds.x + bounds.width / 2;
    camera.position.y = top - viewHeight / 2;
    camera.setViewport(viewHeight * aspect, viewHeight);
  }

  /** Spine 3.6 在结束槽没有附件时会跳过它，裁剪就收不住。把结束槽改到最近一个有区域或网格附件的槽；这一段没有可画附件时卸掉裁剪。skeleton 读取 slots、drawOrder。clips 调用 has，调用 set，用来遍历。 */
  private keepClipsEnding({ skeleton, clips }: Pose): void {
    const { ClippingAttachment, RegionAttachment, MeshAttachment } = this.spine;
    for (const slot of skeleton.slots) {
      const attachment = slot.getAttachment();
      if (attachment instanceof ClippingAttachment && attachment.endSlot && !clips.has(attachment)) {
        clips.set(attachment, { slot, end: attachment.endSlot, detached: false });
      }
    }
    const order = skeleton.drawOrder;
    for (const [clip, c] of clips) {
      const current = c.slot.getAttachment();
      if (current !== clip && !(c.detached && current === null)) {
        c.detached = false;
        continue;
      }
      const from = order.indexOf(c.slot);
      const to = order.findIndex((slot) => slot.data === c.end);
      if (from < 0 || to < 0) continue;
      let last: SlotData | null = null;
      for (let i = to; i > from; i--) {
        const drawn = order[i].getAttachment();
        if (drawn instanceof RegionAttachment || drawn instanceof MeshAttachment) {
          last = order[i].data;
          break;
        }
      }
      if (last) {
        clip.endSlot = last;
        if (c.detached) c.slot.setAttachment(clip);
        c.detached = false;
      } else {
        c.slot.setAttachment(null);
        c.detached = true;
      }
    }
  }
}
