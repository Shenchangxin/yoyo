// @ts-nocheck
import { useEffect, useRef, useState } from "react";
import { Camera } from "lucide-react";

import type { CanvasTheme } from "@yingce/lib/canvas-theme";
import type { CameraControlOptions } from "@yingce/lib/canvas/camera-prompt-library";
import { CanvasNodeCameraPanel } from "./canvas-node-camera-dialog";
import { GenerationSettingsPortal } from "./canvas-generation-settings-shell";

type CanvasCameraControlPopoverProps = {
    cameraControl?: CameraControlOptions;
    onCameraControlChange: (options: CameraControlOptions) => void;
    theme: CanvasTheme;
    compact?: boolean;
};

export function CanvasCameraControlPopover({ cameraControl, onCameraControlChange, theme, compact = false }: CanvasCameraControlPopoverProps) {
    const buttonRef = useRef<HTMLButtonElement>(null);
    const panelRef = useRef<HTMLDivElement>(null);
    const [open, setOpen] = useState(false);
    const [buttonRect, setButtonRect] = useState<DOMRect | null>(null);
    const cameraEnabled = cameraControl?.enabled === true;

    useEffect(() => {
        if (!open) return;
        const syncPosition = () => setButtonRect(buttonRef.current?.getBoundingClientRect() || null);
        const closeOnOutsidePointer = (event: PointerEvent) => {
            const target = event.target;
            if (!(target instanceof Node)) return;
            if (buttonRef.current?.contains(target) || panelRef.current?.contains(target)) return;
            setOpen(false);
        };
        syncPosition();
        window.addEventListener("resize", syncPosition);
        window.addEventListener("scroll", syncPosition, true);
        window.addEventListener("pointerdown", closeOnOutsidePointer, true);
        return () => {
            window.removeEventListener("resize", syncPosition);
            window.removeEventListener("scroll", syncPosition, true);
            window.removeEventListener("pointerdown", closeOnOutsidePointer, true);
        };
    }, [open]);

    return (
        <>
            <button
                ref={buttonRef}
                type="button"
                className={`canvas-node-composer-camera-tools-trigger sg-ghost inline-flex shrink-0 items-center gap-1 px-1.5 text-[var(--fs-tiny)] ${compact ? "is-compact h-7 w-auto min-w-7" : "h-8 w-auto min-w-8"}`}
                style={{
                    background: cameraEnabled ? `${theme.node.activeStroke}66` : undefined,
                    color: cameraEnabled ? theme.node.panel : theme.node.text,
                    width: "auto",
                }}
                aria-pressed={cameraEnabled}
                aria-expanded={open}
                aria-label="摄像机控制"
                title={`摄像机控制${cameraEnabled ? " · 已启用" : ""}`}
                onClick={() => setOpen((current) => !current)}
            >
                <Camera className="size-3.5" />
                {!compact ? <span>摄像机</span> : null}
            </button>
            {open && buttonRect ? (
                <GenerationSettingsPortal buttonRect={buttonRect} panelRef={panelRef} placement="topRight" width={Math.min(360, window.innerWidth - 24)} estimatedHeight={480}>
                    <CanvasNodeCameraPanel
                        cameraControl={cameraControl}
                        onClose={() => setOpen(false)}
                        onConfirm={(options) => {
                            onCameraControlChange(options);
                            setOpen(false);
                        }}
                    />
                </GenerationSettingsPortal>
            ) : null}
        </>
    );
}
