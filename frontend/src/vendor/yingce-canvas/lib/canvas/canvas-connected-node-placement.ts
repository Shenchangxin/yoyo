// @ts-nocheck
import type { Position } from "@yingce/types/canvas";

type NodeSize = {
    width: number;
    height: number;
};

export function connectedNodeCenterFromEdgeDrop(dropPosition: Position, nodeSize: NodeSize, handleType: "source" | "target"): Position {
    return {
        x: dropPosition.x + (handleType === "source" ? nodeSize.width / 2 : -nodeSize.width / 2),
        y: dropPosition.y,
    };
}
