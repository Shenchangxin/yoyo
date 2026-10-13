// @ts-nocheck
import type { CSSProperties, ReactNode } from "react";

import { canvasThemes } from "@yingce/lib/canvas-theme";
import { cn } from "@yingce/lib/utils";
import { useActiveTheme } from "@yingce/stores/canvas/use-canvas-theme-store";

export type CanvasCreateCommand = {
    id: string;
    label: string;
    icon: ReactNode;
    badge?: string;
    section: "node" | "workflow" | "project" | "resource";
    onClick: () => void;
};

const SECTION_LABEL = {
    node: "节点",
    workflow: "工作流",
    resource: "资源",
    project: "项目",
} as const;

export function CanvasCreateMenu({ commands }: { commands: CanvasCreateCommand[] }) {
    const theme = canvasThemes[useActiveTheme()];
    const groups: Array<CanvasCreateCommand["section"]> = ["node", "workflow", "resource", "project"];
    const uniqueCommands = uniqueCreateCommands(commands);

    return (
        <div className="canvas-create-menu min-w-[224px]">
            {groups.map((section) => {
                const items = uniqueCommands.filter((command) => command.section === section);
                if (!items.length) return null;
                return (
                    <section key={section}>
                        <h3 className="sg-eyebrow">{SECTION_LABEL[section]}</h3>
                        {items.map((command) => (
                            <button
                                key={command.id}
                                type="button"
                                className={cn("sg-menu-item canvas-create-command")}
                                style={{ color: theme.node.text } as CSSProperties}
                                title={command.label}
                                onMouseDown={(event) => event.stopPropagation()}
                                onClick={command.onClick}
                            >
                                <span className="grid size-4 shrink-0 place-items-center opacity-70 [&_svg]:size-4">{command.icon}</span>
                                <span className="min-w-0 truncate">{command.label}</span>
                                {command.badge ? <span className="sg-kbd">{command.badge}</span> : null}
                            </button>
                        ))}
                    </section>
                );
            })}
        </div>
    );
}

function uniqueCreateCommands(commands: CanvasCreateCommand[]): CanvasCreateCommand[] {
    const seen = new Map<string, CanvasCreateCommand>();
    for (const command of commands) {
        if (!seen.has(command.id)) seen.set(command.id, command);
    }
    return [...seen.values()];
}
