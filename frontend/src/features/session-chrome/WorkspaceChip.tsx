/**
 * 拥有：工作区芯片上的文件夹名。没绑定时用占位。
 * 不拥有：目录选择。只投影路径，不调用 api。
 */
import { forwardRef } from "react";
import { ChevronDown, Folder, FolderOpen } from "lucide-react";

/** workspaceLabel 取路径最后一段当标签。cwd 做替换，作为后备值。 */
export function workspaceLabel(cwd: string): string {
  const parts = cwd.replace(/\\/g, "/").split("/").filter(Boolean);
  const base = parts[parts.length - 1] || "";
  return base || cwd;
}

/** WorkspaceChip 画出文件夹、名字和展开箭头。props 读取 chooseLabel、menuOpen、disabled、onClick、label。ref 写到 button 的 ref。 */
export const WorkspaceChip = forwardRef<HTMLButtonElement, {
  label?: string;
  menuOpen?: boolean;
  disabled?: boolean;
  chooseLabel: string;
  onClick?: () => void;
}>(function WorkspaceChip(props, ref) {
  return (
    <button
      ref={ref}
      type="button"
      className="sc-ws-chip"
      data-testid="workspace-chip"
      aria-label={props.chooseLabel}
      aria-haspopup="menu"
      aria-expanded={props.menuOpen}
      disabled={props.disabled}
      onClick={props.onClick}
    >
      {props.label
        ? <FolderOpen className="sc-ws-folder" size={16} strokeWidth={1} />
        : <Folder className="sc-ws-folder" size={16} strokeWidth={1} />}
      <span className="sc-ws-label">{props.label ?? props.chooseLabel}</span>
      <ChevronDown className="sc-ws-chevron" size={12} strokeWidth={1} />
    </button>
  );
});
