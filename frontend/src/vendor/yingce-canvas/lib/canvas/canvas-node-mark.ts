import { CanvasNodeType, type CanvasNodeTypeId } from "@yingce/types/canvas";

/** One-letter type mark. Color lives on the selection ring, not on the glyph. */
export function canvasNodeTypeMark(type: CanvasNodeTypeId): "I" | "V" | "A" | "T" | "S" | "D" | "N" {
    if (type === CanvasNodeType.Image) return "I";
    if (type === CanvasNodeType.Video) return "V";
    if (type === CanvasNodeType.Audio) return "A";
    if (type === CanvasNodeType.Text || type === CanvasNodeType.Markdown) return "T";
    if (type === CanvasNodeType.Script) return "S";
    if (type === CanvasNodeType.Drawing) return "D";
    return "N";
}
