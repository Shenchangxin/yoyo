// @ts-nocheck
import { AnimatePresence, motion } from "motion/react";
import { FileImage, Film, Music2 } from "lucide-react";

import type { CanvasTheme } from "@yingce/lib/canvas-theme";
import { aceternityMotion } from "@yingce/lib/aceternity-motion";

export function CanvasFileDropOverlay({ active, theme }: { active: boolean; theme: CanvasTheme }) {
    return (
        <AnimatePresence>
            {active ? (
                <motion.div
                    data-canvas-no-zoom
                    aria-live="polite"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    transition={{ duration: aceternityMotion.duration.state }}
                    className="pointer-events-none absolute inset-3 z-[var(--z-modal-overlay)] grid place-items-center overflow-hidden rounded-xl canvas-file-drop"
                    style={{ background: theme.spatial.dropzone, borderColor: theme.spatial.glowStrong, color: theme.node.text, boxShadow: `inset 0 0 0 1px ${theme.spatial.glow}, 0 30px 100px rgba(0,0,0,.22)` }}
                >
                    <motion.div initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: 6 }} transition={{ duration: 0.18 }} className="relative flex w-[min(420px,calc(100vw-48px))] flex-col items-center px-6 py-5 text-center">
                        <h2 className="text-[15px] font-semibold tracking-[-0.02em]">放到画布</h2>
                        <p className="mt-1 text-[13px] leading-5" style={{ color: theme.node.muted }}>图片、视频和音频会在这里变成节点</p>
                        <div className="mt-4 flex items-center gap-2 text-[11px] font-medium" style={{ color: theme.node.muted }}>
                            <span className="inline-flex items-center gap-1.5 rounded-md px-2 py-1" style={{ background: theme.node.fill }}><FileImage className="size-3.5" />图片</span>
                            <span className="inline-flex items-center gap-1.5 rounded-md px-2 py-1" style={{ background: theme.node.fill }}><Film className="size-3.5" />视频</span>
                            <span className="inline-flex items-center gap-1.5 rounded-md px-2 py-1" style={{ background: theme.node.fill }}><Music2 className="size-3.5" />音频</span>
                        </div>
                    </motion.div>
                </motion.div>
            ) : null}
        </AnimatePresence>
    );
}
