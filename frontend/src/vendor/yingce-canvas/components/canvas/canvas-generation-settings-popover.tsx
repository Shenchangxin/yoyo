// @ts-nocheck
import { CanvasAudioSettingsPopover, type CanvasAudioSettingKey } from "./canvas-audio-settings-popover";
import { CanvasImageSettingsPopover } from "./canvas-image-settings-popover";
import { CanvasVideoSettingsPopover } from "./canvas-video-settings-popover";
import type { AiConfig } from "@yingce/stores/use-config-store";

type GenerationSettingsKind = "image" | "video" | "audio";

export function GenerationSettingsPopover({
    kind,
    config,
    onConfigChange,
    buttonClassName,
    placement,
    showCount,
    onOpenChange,
}: {
    kind: GenerationSettingsKind;
    config: AiConfig;
    onConfigChange: (key: string, value: string) => void;
    buttonClassName?: string;
    placement?: "topLeft" | "top" | "topRight" | "bottomLeft" | "bottom" | "bottomRight";
    showCount?: boolean;
    onOpenChange?: (open: boolean) => void;
}) {
    if (kind === "video") {
        return <CanvasVideoSettingsPopover config={config} onConfigChange={onConfigChange} buttonClassName={buttonClassName} placement={placement} />;
    }
    if (kind === "audio") {
        return <CanvasAudioSettingsPopover config={config} onConfigChange={onConfigChange as (key: CanvasAudioSettingKey, value: string) => void} buttonClassName={buttonClassName} placement={placement} />;
    }
    return (
        <CanvasImageSettingsPopover
            config={config}
            onConfigChange={onConfigChange}
            buttonClassName={buttonClassName}
            placement={placement}
            showCount={showCount}
            onOpenChange={onOpenChange}
        />
    );
}

export { CanvasImageSettingsPopover } from "./canvas-image-settings-popover";
export { CanvasVideoSettingsPopover } from "./canvas-video-settings-popover";
export { CanvasAudioSettingsPopover } from "./canvas-audio-settings-popover";
