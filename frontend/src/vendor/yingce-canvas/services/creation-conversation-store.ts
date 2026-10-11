// @ts-nocheck
import { localForageStorageForScope } from "@yingce/lib/localforage-storage";
import { getActiveUserScope } from "@yingce/lib/user-scope";

export const CREATION_CONVERSATIONS_KEY = "creation-conversations-v1";
export const CREATION_CONVERSATIONS_CORRUPT_KEY = "creation-conversations-v1:corrupt";

type PendingCreationMessage = {
    id: string;
    role: "user" | "assistant";
    mode?: string;
    status?: string;
    taskIds?: string[];
};

export type StoredCreationConversation = {
    id: string;
    messages: PendingCreationMessage[];
};

export function updateCreationConversationSnapshot<T extends { id: string }>(conversations: T[], conversationId: string, updater: (conversation: T) => T) {
    return conversations.map((conversation) => (conversation.id === conversationId ? updater(conversation) : conversation));
}

// 对话、生成任务与素材是独立持久状态；删除历史记录不能在这里级联清理任务或资源。
export function removeCreationConversationSnapshot<T extends { id: string }>(conversations: T[], conversationId: string) {
    if (!conversationId) throw new Error("缺少要删除的创作对话 ID");
    const next = conversations.filter((conversation) => conversation.id !== conversationId);
    if (next.length === conversations.length) throw new Error("要删除的创作对话不存在");
    return next;
}

function isRecoverableCreationMessage(message: PendingCreationMessage) {
    if (message.role !== "assistant" || !message.taskIds?.length) return false;
    return message.mode === "text" ? message.status === "streaming" || message.status === "pending" : message.status === "pending";
}

export function pendingCreationTaskKey(conversations: StoredCreationConversation[]) {
    return conversations
        .flatMap((conversation) => conversation.messages.flatMap((message) => (isRecoverableCreationMessage(message) ? [`${conversation.id}:${message.id}:${(message.taskIds || []).join(",")}`] : [])))
        .join("|");
}

export function pendingCreationTaskIds(conversations: StoredCreationConversation[]) {
    const taskIds = conversations.flatMap((conversation) =>
        conversation.messages.flatMap((message) => {
            if (!isRecoverableCreationMessage(message)) return [];
            return message.taskIds || [];
        }),
    );
    return Array.from(new Set(taskIds));
}

export type CreationConversationDocument<T> = {
    activeId: string;
    conversations: T[];
    corrupt: boolean;
};

export function parseCreationConversationValue(value: string): CreationConversationDocument<unknown> {
    let parsed: unknown;
    try {
        parsed = JSON.parse(value);
    } catch {
        return { activeId: "", conversations: [], corrupt: true };
    }
    if (Array.isArray(parsed)) return { activeId: "", conversations: parsed, corrupt: false };
    if (parsed && typeof parsed === "object" && Array.isArray((parsed as { conversations?: unknown }).conversations)) {
        const record = parsed as { activeId?: unknown; conversations: unknown[] };
        return {
            activeId: typeof record.activeId === "string" ? record.activeId : "",
            conversations: record.conversations,
            corrupt: false,
        };
    }
    return { activeId: "", conversations: [], corrupt: true };
}

type MemoryDocument = { scope: string; activeId: string; conversations: unknown[]; corrupt: boolean };

let memory: MemoryDocument | null = null;
let writeTail: Promise<unknown> = Promise.resolve();

async function readCreationDocument<T extends StoredCreationConversation>(): Promise<CreationConversationDocument<T>> {
    const scope = getActiveUserScope();
    const storage = localForageStorageForScope(scope);
    const value = await storage.getItem(CREATION_CONVERSATIONS_KEY);
    if (!value) return { activeId: "", conversations: [], corrupt: false };
    const parsed = parseCreationConversationValue(value);
    if (!parsed.corrupt) return { activeId: parsed.activeId, conversations: parsed.conversations as T[], corrupt: false };
    try {
        await storage.setItem(CREATION_CONVERSATIONS_CORRUPT_KEY, value);
    } catch {
        // 备份失败不能挡住页面；原始键仍在，直到下一次成功写入。
    }
    return { activeId: "", conversations: [], corrupt: true };
}

function enqueueCreationWrite<T>(operation: () => Promise<T>): Promise<T> {
    const pending = writeTail.then(operation, operation);
    writeTail = pending.then(() => undefined, () => undefined);
    return pending;
}

export function readCreationConversationDocument<T extends StoredCreationConversation>() {
    return enqueueCreationWrite(async (): Promise<CreationConversationDocument<T>> => {
        const scope = getActiveUserScope();
        if (!memory || memory.scope !== scope) {
            const loaded = await readCreationDocument<T>();
            memory = { scope, ...loaded };
        }
        return { activeId: memory.activeId, conversations: memory.conversations as T[], corrupt: memory.corrupt };
    });
}

// 所有写入串行，并且基于最新文档做修改。页面卸载后的生成收尾不能拿旧数组覆盖新会话。
export function mutateCreationConversations<T extends StoredCreationConversation>(mutator: (doc: { activeId: string; conversations: T[] }) => { activeId: string; conversations: T[] }) {
    return enqueueCreationWrite(async (): Promise<CreationConversationDocument<T>> => {
        const scope = getActiveUserScope();
        if (!memory || memory.scope !== scope) {
            const loaded = await readCreationDocument<T>();
            memory = { scope, ...loaded };
        }
        const next = mutator({ activeId: memory.activeId, conversations: memory.conversations as T[] });
        const doc: CreationConversationDocument<T> = { activeId: next.activeId, conversations: next.conversations, corrupt: false };
        memory = { scope, ...doc };
        await localForageStorageForScope(scope).setItem(CREATION_CONVERSATIONS_KEY, JSON.stringify({ activeId: doc.activeId, conversations: doc.conversations }));
        return doc;
    });
}

export async function loadCreationConversations<T extends StoredCreationConversation>() {
    const doc = await readCreationConversationDocument<T>();
    return doc.conversations.length ? doc.conversations : null;
}

export async function saveCreationConversations<T extends StoredCreationConversation>(conversations: T[]) {
    await mutateCreationConversations<T>((doc) => ({ activeId: doc.activeId, conversations }));
}
