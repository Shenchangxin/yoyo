/**
 * 拥有：一个滚动容器是否该跟着底部，以及程序滚动和读者滚动的区分。
 * 不拥有：转录何时请求滚到底，以及滚动条的样式。
 */

/** ViewportMetrics 视口的滚动位置、内容底部和可视高度。 */
export type ViewportMetrics = {
  top: number;
  floor: number;
  height: number;
};

/** scrollMetrics 从元素读出视口度量。底部是滚动高度减去可视高度，不小于零。element 读取 clientHeight、scrollTop、scrollHeight。 */
export function scrollMetrics(element: HTMLElement): ViewportMetrics {
  const height = element.clientHeight;
  return { top: element.scrollTop, height, floor: Math.max(0, element.scrollHeight - height) };
}

/**
 * ScrollFollow 一个滚动容器的跟随意图。
 * 原生平滑滚动的中途位移不算读者自己在滚。
 */
export class ScrollFollow {
  private static readonly owners = new WeakMap<HTMLElement, ScrollFollow>();
  private target: number | null = null;
  private sampledTop: number | undefined;
  private following: boolean;
  private readonly threshold: number;

  /** 记录初始跟随意图，以及离底部多远仍算贴底。following 记到 this.following。threshold 记到 this.threshold。 */
  constructor(following: boolean, threshold: number) {
    this.following = following;
    this.threshold = threshold;
  }

  /** forElement 取这个元素上绑定的跟随器。element 传给 owners.get。 */
  static forElement(element: HTMLElement): ScrollFollow | undefined {
    return this.owners.get(element);
  }

  /** bind 把跟随器登记到元素上，返回解除登记的函数。element 交给 ScrollFollow.owners.set，传给 ScrollFollow.owners.get，传给 ScrollFollow.owners.delete。 */
  bind(element: HTMLElement): () => void {
    ScrollFollow.owners.set(element, this);
    return () => {
      if (ScrollFollow.owners.get(element) === this) ScrollFollow.owners.delete(element);
    };
  }

  /** active 当前是否跟着底部。 */
  get active(): boolean {
    return this.following;
  }

  /** animating 是否有一段尚未落地的程序滚动。 */
  get animating(): boolean {
    return this.target !== null;
  }

  /** nearBottom 当前位置离底部不超过阈值。metrics 读取 floor、top。 */
  nearBottom(metrics: ViewportMetrics): boolean {
    return Math.abs(metrics.floor - metrics.top) <= this.threshold;
  }

  /** setFollowing 设置跟随。关掉时放弃未完成的滚动目标。active 为假时走另一支，记到 this.following。 */
  setFollowing(active: boolean): void {
    this.following = active;
    if (!active) this.target = null;
  }

  /** reset 停止跟随，并忘掉上次采样位置。 */
  reset(): void {
    this.setFollowing(false);
    this.sampledTop = undefined;
  }

  /** sample 记一次位置。不在程序滚动中、且位移来自读者时，按是否靠近底部更新跟随。返回当前是否跟随。metrics 读取 top，传给 nearBottom。movedByReader 为真时才继续。 */
  sample(metrics: ViewportMetrics, movedByReader = this.sampledTop === undefined
    || Math.abs(metrics.top - this.sampledTop) > 0.5): boolean {
    this.sampledTop = metrics.top;
    if (!this.animating && movedByReader) this.following = this.nearBottom(metrics);
    return this.active;
  }

  /** settle 程序滚动结束后再采样。离预定目标超过阈值才当成读者移动。metrics 读取 top、floor，传给 sample。 */
  settle(metrics: ViewportMetrics): boolean {
    const target = this.target;
    this.target = null;
    return this.sample(metrics, target === null ? undefined
      : Math.abs(metrics.top - Math.min(target, metrics.floor)) > this.threshold);
  }

  /** jump 立刻跳到指定位置，并按落地位置决定是否继续跟随。已经在动画中时用即时滚动打断。element 读取 scrollTop，调用 scrollTo。metrics 读取 floor、top，展开传递。top 传给 Math.min。 */
  jump(element: HTMLElement, metrics: ViewportMetrics, top: number): ViewportMetrics {
    const animated = this.animating;
    this.target = null;
    const target = Math.max(0, Math.min(metrics.floor, top));
    if (animated) element.scrollTo({ top: target, behavior: "instant" });
    else if (target !== metrics.top) element.scrollTop = target;
    const landed = { ...metrics, top: element.scrollTop };
    this.sampledTop = landed.top;
    this.following = this.nearBottom(landed);
    return landed;
  }

  /** toBottom 跟随到底部。已在底部、要求立刻、靠近底部或用户减少动效时直接跳；否则只发起一次平滑滚动。element 调用 scrollTo，传给 jump。metrics 读取 top、floor，传给 jump，传给 nearBottom。behavior 与 instant 比较是否相等。 */
  toBottom(element: HTMLElement, metrics: ViewportMetrics, behavior: "instant" | "smooth"): ViewportMetrics {
    this.following = true;
    if (metrics.top === metrics.floor) return this.jump(element, metrics, metrics.floor);
    if (behavior === "instant" && this.animating) {
      if (this.target !== metrics.floor) {
        this.target = metrics.floor;
        element.scrollTo({ top: metrics.floor, behavior: "smooth" });
      }
      return metrics;
    }
    if (behavior === "instant" || (!this.animating && this.nearBottom(metrics))) {
      return this.jump(element, metrics, metrics.floor);
    }
    if (typeof matchMedia === "function" && matchMedia("(prefers-reduced-motion: reduce)").matches) {
      return this.jump(element, metrics, metrics.floor);
    }
    if (this.target === null) {
      this.target = metrics.floor;
      element.scrollTo({ top: metrics.floor, behavior: "smooth" });
    }
    return metrics;
  }

  /** interrupt 打断进行中的平滑滚动，停在当前视口位置。element 调用 scrollTo。metrics 读取 top。 */
  interrupt(element: HTMLElement, metrics: ViewportMetrics): void {
    if (!this.animating) return;
    this.target = null;
    this.sampledTop = metrics.top;
    element.scrollTo({ top: metrics.top, behavior: "instant" });
  }
}
