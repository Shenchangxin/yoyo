import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { CanvasNodeType, type CanvasNodeData } from "@yingce/types/canvas";
import { applyCanvasStoryboardLayout, layoutCanvasStoryboard } from "./canvas-layout.ts";

function node(partial: Partial<CanvasNodeData> & Pick<CanvasNodeData, "id" | "type">): CanvasNodeData {
    return {
        title: partial.id,
        position: { x: 0, y: 0 },
        width: 160,
        height: 90,
        ...partial,
    } as CanvasNodeData;
}

describe("layoutCanvasStoryboard", () => {
    it("puts characters on the left and shots on the right", () => {
        const nodes = [
            node({ id: "hero", type: CanvasNodeType.Image, metadata: { workflowKind: "character" } }),
            node({ id: "shot-1", type: CanvasNodeType.Video }),
            node({ id: "shot-2", type: CanvasNodeType.Image }),
        ];
        const positions = layoutCanvasStoryboard(nodes, { x: 80, y: 80 });
        assert.equal(positions.get("hero")?.x, 80);
        assert.ok((positions.get("shot-1")?.x || 0) > 80);
        assert.equal(positions.get("shot-1")?.x, positions.get("shot-2")?.x);
        assert.ok((positions.get("shot-2")?.y || 0) > (positions.get("shot-1")?.y || 0));
    });

    it("leaves a single node alone", () => {
        const positions = layoutCanvasStoryboard([node({ id: "only", type: CanvasNodeType.Video })]);
        assert.equal(positions.size, 0);
    });

    it("applies positions without rewriting locked nodes", () => {
        const nodes = [
            node({ id: "hero", type: CanvasNodeType.Image, metadata: { workflowKind: "character" } }),
            node({ id: "shot-1", type: CanvasNodeType.Video, position: { x: 12, y: 12 } }),
            node({ id: "locked", type: CanvasNodeType.Image, metadata: { locked: true }, position: { x: 9, y: 9 } }),
        ];
        const next = applyCanvasStoryboardLayout(nodes);
        assert.equal(next.find((item) => item.id === "hero")?.position.x, 80);
        assert.equal(next.find((item) => item.id === "locked")?.position.x, 9);
        assert.ok((next.find((item) => item.id === "shot-1")?.position.x || 0) > 80);
    });
});
