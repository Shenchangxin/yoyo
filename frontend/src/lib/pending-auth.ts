import * as api from "./client";
import type { Thread } from "./protocol";
import { useUI } from "./store";

/**
 * 拥有：把会话还不存在时选中的权限模式套到随后创建的第一段对话。
 * 不拥有：权限模式有哪些取值、会话如何创建，以及失败后界面上显示哪一种模式。
 */

/**
 * adoptPendingAuthMode 取出暂存的权限模式并写到这段对话上。
 * 没有暂存，或会话已经是该模式时，原样返回。写入失败时也返回原会话，调用方应显示会话上的真实模式。
 * t 读取 authMode、id，有的分支原样返回。
 */
export async function adoptPendingAuthMode(t: Thread): Promise<Thread> {
  const mode = useUI.getState().pendingAuthMode;
  if (!mode) return t;
  useUI.getState().setPendingAuthMode(null);
  if (t.authMode === mode) return t;
  try {
    return await api.setSessionAuthMode(t.id, mode);
  } catch {
    return t;
  }
}
