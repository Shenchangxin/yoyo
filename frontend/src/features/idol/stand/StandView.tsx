/**
 * 拥有：立绘目录地址、通话前的服装预热，以及把 StandEngine 投影到画布。
 * 不拥有：片段选择、通话状态和对话模型。预热只拉取服装文件和运行时脚本。绘制循环在 StandEngine。不改服务端。
 */

import { useEffect, useRef, useState } from "react";
import { cn } from "../../../lib/utils";
import type { StandDirection } from "./animations";
import { StandEngine, type Framing, type Outfit } from "./engine";
import { loadSpineRuntime } from "./spineRuntime";

const OUTFIT_ID = /^(stand|stand_costume)\/\d+$/;

/** standURL 把 stand 或 stand_costume 加数字的 id 收成立绘目录。格式不符则空。 */
export function standURL(id: string): string | null {
  if (!OUTFIT_ID.test(id)) return null;
  return `/idol-stand/${id}/`;
}

const warming = new Set<string>();

/** warmStand 通话前拉取服装和 Spine 运行时，让身体不必先等头像。同一目录只预热一次，某个文件失败则允许再试。 */
export function warmStand(id: string) {
  const base = standURL(id);
  if (!base || warming.has(base)) return;
  warming.add(base);
  void loadSpineRuntime().catch(() => {});
  for (const name of ["data.json", "data.atlas", "data.png"]) {
    void fetch(base + name).catch(() => {
      warming.delete(base);
    });
  }
}

/** StandView 把指定立绘画到画布上。加载失败或地址无效时不渲染。阶段通过 onPhase 报出。 */
export function StandView(props: {
  stand: string;
  framing: Framing;
  direction: StandDirection;
  reduced: boolean;
  onPhase?: (phase: "loading" | "shown" | "failed") => void;
}) {
  const url = standURL(props.stand);
  const canvas = useRef<HTMLCanvasElement>(null);
  const engine = useRef<StandEngine | null>(null);
  const [ready, setReady] = useState<StandEngine | null>(null);
  const [phase, setPhase] = useState<"loading" | "shown" | "failed">("loading");
  const report = useRef(props.onPhase);
  report.current = props.onPhase;
  const current = useRef({ framing: props.framing, direction: props.direction, reduced: props.reduced });

  useEffect(() => {
    current.current.framing = props.framing;
    if (engine.current) engine.current.framing = props.framing;
  }, [props.framing]);

  useEffect(() => {
    current.current.direction = props.direction;
    current.current.reduced = props.reduced;
    engine.current?.direct(props.direction, props.reduced);
  }, [props.direction, props.reduced]);

  useEffect(() => {
    const target = canvas.current;
    if (!target || !url) return;
    let cancelled = false;
    let created: StandEngine | null = null;
    loadSpineRuntime()
      .then((spine) => {
        if (cancelled) return;
        created = new StandEngine(target, spine, current.current.framing);
        created.direct(current.current.direction, current.current.reduced);
        engine.current = created;
        setReady(created);
      })
      .catch(() => {
        if (!cancelled) {
          setPhase("failed");
          report.current?.("failed");
        }
      });
    return () => {
      cancelled = true;
      created?.dispose();
      engine.current = null;
    };
  }, [url]);

  useEffect(() => {
    if (!ready || !url) return;
    let cancelled = false;
    ready.load({ id: props.stand, url } satisfies Outfit)
      .then(() => {
        if (!cancelled) {
          setPhase("shown");
          report.current?.("shown");
        }
      })
      .catch(() => {
        if (!cancelled) {
          setPhase("failed");
          report.current?.("failed");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [ready, url, props.stand]);

  if (!url || phase === "failed") return null;
  return (
    <canvas
      ref={canvas}
      aria-hidden
      className={cn("idol-spine", phase === "shown" && "is-shown")}
    />
  );
}
