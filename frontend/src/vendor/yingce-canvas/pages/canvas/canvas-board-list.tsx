// @ts-nocheck
import { useMemo, useState } from "react";
import { useCopy } from "@/lib/i18n";
import { canvasNodeTypeMark } from "@yingce/lib/canvas/canvas-node-mark";
import { searchCanvasNodes } from "@yingce/lib/canvas/canvas-node-search";
import { useEffectiveConfig } from "@yingce/stores/use-config-store";
import type { CanvasNodeData } from "@yingce/types/canvas";

export function CanvasBoardList({
    nodes,
    selectedNodeIds,
    onFocus,
}: {
    nodes: CanvasNodeData[];
    selectedNodeIds: Set<string>;
    onFocus: (id: string) => void;
}) {
    const board = useCopy().video.canvasBoard;
    const config = useEffectiveConfig();
    const [query, setQuery] = useState("");
    const results = useMemo(() => searchCanvasNodes(nodes, query, nodes.length, config), [config, nodes, query]);
    return (
        <div className="canvas-board-list" data-testid="canvas-board-list">
            <label className="canvas-board-list-search">
                <span className="sr-only">{board.search}</span>
                <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={board.search} />
            </label>
            <ul>
                {results.map((node) => {
                    const selected = selectedNodeIds.has(node.id);
                    return (
                        <li key={node.id}>
                            <button
                                type="button"
                                className={selected ? "is-selected" : undefined}
                                aria-current={selected ? "true" : undefined}
                                onClick={() => onFocus(node.id)}
                            >
                                <span className="canvas-node-type-mark" aria-hidden>{canvasNodeTypeMark(node.type)}</span>
                                <span className="canvas-board-list-title">{node.title || board.untitled}</span>
                                {node.metadata?.shotRole ? <span className="canvas-board-list-role">{shotRoleLabel(node.metadata.shotRole, board)}</span> : null}
                            </button>
                        </li>
                    );
                })}
            </ul>
        </div>
    );
}

function shotRoleLabel(role: "keyframe" | "final" | "ref", board: ReturnType<typeof useCopy>["video"]["canvasBoard"]) {
    if (role === "keyframe") return board.roleKeyframe;
    if (role === "final") return board.roleFinal;
    return board.roleRef;
}
