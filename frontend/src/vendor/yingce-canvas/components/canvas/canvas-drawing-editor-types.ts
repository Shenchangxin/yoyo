// @ts-nocheck
import type { CanvasDrawingRenderDraft } from "@yingce/lib/canvas/canvas-drawing-storage";

export type CanvasDrawingEditorHandle = {
    createSave: () => Promise<{
        snapshot: unknown;
        preview: Blob | null;
        render: CanvasDrawingRenderDraft | null;
    }>;
};

export type CanvasDrawingEditorProps = {
    snapshot: unknown;
    colorScheme: "light" | "dark";
    onReady: () => void;
};
