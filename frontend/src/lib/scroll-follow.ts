export type ViewportMetrics = {
  top: number;
  floor: number;
  height: number;
};

export function scrollMetrics(element: HTMLElement): ViewportMetrics {
  const height = element.clientHeight;
  return { top: element.scrollTop, height, floor: Math.max(0, element.scrollHeight - height) };
}

/** One scrollport's follow intent; native animation progress does not count as reader movement. */
export class ScrollFollow {
  private static readonly owners = new WeakMap<HTMLElement, ScrollFollow>();
  private target: number | null = null;
  private sampledTop: number | undefined;
  private following: boolean;
  private readonly threshold: number;

  constructor(following: boolean, threshold: number) {
    this.following = following;
    this.threshold = threshold;
  }

  static forElement(element: HTMLElement): ScrollFollow | undefined {
    return this.owners.get(element);
  }

  bind(element: HTMLElement): () => void {
    ScrollFollow.owners.set(element, this);
    return () => {
      if (ScrollFollow.owners.get(element) === this) ScrollFollow.owners.delete(element);
    };
  }

  get active(): boolean {
    return this.following;
  }

  get animating(): boolean {
    return this.target !== null;
  }

  nearBottom(metrics: ViewportMetrics): boolean {
    return Math.abs(metrics.floor - metrics.top) <= this.threshold;
  }

  setFollowing(active: boolean): void {
    this.following = active;
    if (!active) this.target = null;
  }

  reset(): void {
    this.setFollowing(false);
    this.sampledTop = undefined;
  }

  sample(metrics: ViewportMetrics, movedByReader = this.sampledTop === undefined
    || Math.abs(metrics.top - this.sampledTop) > 0.5): boolean {
    this.sampledTop = metrics.top;
    if (!this.animating && movedByReader) this.following = this.nearBottom(metrics);
    return this.active;
  }

  settle(metrics: ViewportMetrics): boolean {
    const target = this.target;
    this.target = null;
    return this.sample(metrics, target === null ? undefined
      : Math.abs(metrics.top - Math.min(target, metrics.floor)) > this.threshold);
  }

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

  interrupt(element: HTMLElement, metrics: ViewportMetrics): void {
    if (!this.animating) return;
    this.target = null;
    this.sampledTop = metrics.top;
    element.scrollTo({ top: metrics.top, behavior: "instant" });
  }
}
