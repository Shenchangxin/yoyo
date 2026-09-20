import { Transcript } from "./Transcript";
import { Composer } from "./Composer";
import type { Approval, Attachment, AuthMode, ContextUsage, FileHit, Item, SkillInfo } from "../lib/protocol";
import type { TaskPlan } from "../lib/plan";
import { useCopy } from "../lib/i18n";
import { lastUserTurns } from "../lib/stream";

export function ChatDock(props: {
  items: Item[];
  taskPlan?: TaskPlan | null;
  approvals: Approval[];
  running: boolean;
  draftKey: string;
  disabled: boolean;
  disabledReason?: string;
  model?: string;
  models?: string[];
  provider?: string;
  ctx?: ContextUsage;
  queued?: number;
  files?: FileHit[];
  skills?: SkillInfo[];
  authMode?: AuthMode | string;
  onSearchFiles?: (q: string) => void;
  onPickFiles?: () => Promise<Attachment[]>;
  onAuthMode?: (mode: AuthMode) => void;
  workspace?: string;
  workspaces?: string[];
  isolate?: boolean;
  onWorkspace?: (path: string) => void;
  onBrowseWorkspace?: () => void;
  onIsolate?: (isolate: boolean) => void;
  onSend: (opts?: { steer?: boolean; attachments?: Attachment[]; text?: string }) => void;
  onStop: () => void;
  onClipboard?: () => Promise<string>;
  onScreenshot?: () => Promise<string>;
  onResolve: (id: string, decision: string) => void;
  onOpenAgent: () => void;
  onRetry?: () => void;
  onSlash?: (cmd: string, rest: string) => void;
  onModel?: (model: string) => void;
}) {
  const copy = useCopy();
  return (
    <aside className="flex h-full min-h-0 flex-col bg-transparent">
      <div className="flex h-10 shrink-0 items-center border-b border-border/70 px-3">
        <span className="text-[12.5px] font-medium tracking-[-0.005em]">{copy.dock.chat}</span>
        <button type="button" className="ml-auto h-6 cursor-pointer rounded-md px-2 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground" onClick={props.onOpenAgent}>
          {copy.dock.expand}
        </button>
      </div>
      <Transcript items={lastUserTurns(props.items, 3)} approvals={props.approvals} running={props.running} compact workspace={props.workspace} onResolve={props.onResolve} onRetry={props.onRetry} />
      <Composer
        draftKey={props.draftKey}
        running={props.running}
        disabled={props.disabled}
        disabledReason={props.disabledReason || copy.composer.disabled}
        model={props.model}
        models={props.models}
        provider={props.provider}
        ctx={props.ctx}
        queued={props.queued}
        files={props.files}
        skills={props.skills}
        authMode={props.authMode}
        onSearchFiles={props.onSearchFiles}
        onPickFiles={props.onPickFiles}
        onAuthMode={props.onAuthMode}
        workspace={props.workspace}
        workspaces={props.workspaces}
        isolate={props.isolate}
        onWorkspace={props.onWorkspace}
        onBrowseWorkspace={props.onBrowseWorkspace}
        onIsolate={props.onIsolate}
        onSend={props.onSend}
        onStop={props.onStop}
        onSlash={props.onSlash}
        onModel={props.onModel}
        onClipboard={props.onClipboard}
        onScreenshot={props.onScreenshot}
        taskPlan={props.taskPlan}
        compact
      />
    </aside>
  );
}
