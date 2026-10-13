// @ts-nocheck
import { ImageSizePicker } from "@yingce/components/image-size-picker";
import { imageResolutionUsesQuality } from "@yingce/lib/image-size-presets";
import { createPortal } from "react-dom";
import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState, type CSSProperties, type PointerEvent, type ReactNode, type RefObject } from "react";
import { App, Button, Dropdown, Popover } from "antd";
import { AppDrawer } from "@yingce/components/ui/product/app-drawer";
import { AppModal } from "@yingce/components/ui/product/app-modal";
import { useWorkspaceTopBarMount } from "@yingce/components/layout/workspace-top-bar-extension";
import { Tooltip } from "@yingce/components/ui/base/tooltip";
import { Reorder, LayoutGroup, motion, useReducedMotion } from "motion/react";
import { ArrowDown, ArrowUp, BookOpen, Brain, ChevronDown, ChevronLeft, ChevronRight, Clapperboard, Clock3, Copy, Download, FileText, Film, History, Image as ImageIcon, LoaderCircle, Maximize2, MessageSquareText, Minimize2, MoreHorizontal, Music2, Pencil, Plus, RefreshCw, Search, SlidersHorizontal, Trash2, UserRound, Waves, X } from "lucide-react";

import { AIMessageMarkdown } from "@yingce/components/ai/ai-message-markdown";
import { GenerationToolCard, type GenerationToolStatus } from "@yingce/components/ai/generation-tool-card";
import { WorkingDots } from "@yingce/components/ai/working-indicator";
import { MessageReasoning } from "@yingce/components/ai/message-reasoning";
import { creationResultAssetIds } from "@yingce/lib/canvas/canvas-asset-handoff";
import { generationErrorMessage } from "@yingce/lib/generation-error";
import { formatVideoResolutionLabel as videoResolutionLabel } from "@yingce/lib/video-generation-options";
import { useAssetStore } from "@yingce/stores/use-asset-store";
import { CachedResourceImage } from "@yingce/components/cached-resource-image";
import { CanvasImagePreview } from "@yingce/components/canvas/canvas-image-preview";
import { CanvasResourceMentionTextarea } from "@yingce/components/canvas/canvas-resource-mention-textarea";
import { VoiceRecordingButton } from "@yingce/components/conversation/voice-recording-button";
import { ModelPicker } from "@yingce/components/model-picker";
import { aceternityMotion } from "@yingce/lib/aceternity-motion";
import { CreditSymbol, requestCreditCost } from "@yingce/constant/credits";
import { ASSET_CATEGORY_LABELS } from "@yingce/lib/asset-category";
import { formatShotOrdinal } from "@yingce/lib/shot-label";
import { useCopyText } from "@yingce/hooks/use-copy-text";
import { buildImageResolutionOptions, formatImageResolutionSize, supportsImageResolutionPresets } from "@yingce/lib/image-resolution-tiers";
import { modelCapabilityConfigFor, normalizeVideoValue, videoDurationOptions, type ImageCapabilityConfig, type VideoCapabilityConfig } from "@yingce/lib/model-capabilities";
import { mergedImageCapabilityConfig, type ModelRequirements } from "@yingce/lib/model-selection";
import { modelQuoteDescription, modelQuoteRequest } from "@yingce/lib/model-pricing";
import type { Skill } from "@yingce/services/api/skills";
import { quoteModel, type LogicalModelQuote } from "@yingce/services/api/logical-models";
import { resolveResourceUrl } from "@yingce/services/api/resources";
import { modelOptionName, resolveModelChannel, type AiConfig } from "@yingce/stores/use-config-store";
import { useAppearanceStore } from "@yingce/stores/use-appearance-store";
import { useUserStore } from "@yingce/stores/use-user-store";
import type { PromptOptimizerProvider } from "@yingce/lib/plugins/plugin-types";
import { displayCreationPrompt, type CreationReference } from "./creation-references";
import { creationAttachmentKind, creationMediaAspectRatio, removeCreationAttachment, type CreationAttachment, type CreationMode } from "./creation-assets";
import { conversationTimestamp, isImageAttachment, isVideoAttachment } from "./creation-conversations";
import { conversationTimeFormatter, countOptions, historyDayFormatter, messageTimeFormatter, modeLabels, qualityOptions, ratioOptions, resolutionOptions, shotScriptLabels, type CreationConversation, type CreationMessage, type CreationShotRailEntry, type CreationStatus } from "./creation-types";
import "./creation-product.css";
import "./creation-scrollbars.css";
import { creationFeaturedWorks, inspirationSource } from "./creation-inspirations";
import { styleCoverSrc } from "@yingce/lib/style-cover";
import { useCopy } from "@/lib/i18n";
import { isYoyoCanvasHost } from "@/features/video/canvas-host/design-system";

const CanvasPromptOptimizerDrawer = lazy(() => import("@yingce/components/canvas/canvas-prompt-optimizer-drawer").then((module) => ({ default: module.CanvasPromptOptimizerDrawer })));

export const creationAssetCategoryLabels: Record<string, string> = { all: "全部素材", ...ASSET_CATEGORY_LABELS };


function videoFrameRoleLabel(attachments: CreationAttachment[], id: string, create: ReturnType<typeof useCopy>["video"]["create"]) {
    const imageIds = attachments.filter((attachment) => creationAttachmentKind(attachment) === "image").map((attachment) => attachment.id);
    const index = imageIds.indexOf(id);
    if (index === 0) return create.frameIn;
    if (index === 1) return create.frameOut;
    return create.frameRef;
}

type CreationCopy = ReturnType<typeof useCopy>["video"]["create"];

function creationModeLabel(mode: CreationMode, create: CreationCopy) {
    if (mode === "video") return create.modeVideo;
    if (mode === "image") return create.modeImage;
    if (mode === "text") return create.modeText;
    return create.work;
}

function CreationShotStructure({ prompt, setPrompt, disabled, create }: { prompt: string; setPrompt: (value: string) => void; disabled: boolean; create: CreationCopy }) {
    const [open, setOpen] = useState(false);
    const [fields, setFields] = useState({ subject: "", scene: "", action: "", camera: "", light: "" });
    const rows = [
        { key: "subject", label: create.shotSubject },
        { key: "scene", label: create.shotScene },
        { key: "action", label: create.shotAction },
        { key: "camera", label: create.shotCamera },
        { key: "light", label: create.shotLight },
    ] as const;
    const insert = () => {
        const block = rows.map((row) => {
            const value = fields[row.key].trim();
            return value ? `${row.label}: ${value}` : "";
        }).filter(Boolean).join("\n");
        if (!block) return;
        setPrompt(prompt.trim() ? `${prompt.trim()}\n${block}` : block);
    };
    return (
        <div className="creation-shot-structure" data-testid="creation-shot-structure">
            <button type="button" className="creation-shot-structure-toggle" aria-expanded={open} disabled={disabled} onClick={() => setOpen((value) => !value)}>
                {create.shotControls}
            </button>
            {open ? (
                <div className="creation-shot-structure-grid">
                    {rows.map((row) => (
                        <label key={row.key}>
                            <span>{row.label}</span>
                            <input
                                value={fields[row.key]}
                                disabled={disabled}
                                onChange={(event) => setFields((current) => ({ ...current, [row.key]: event.target.value }))}
                            />
                        </label>
                    ))}
                    <button type="button" className="creation-shot-structure-insert" disabled={disabled} onClick={insert}>{create.shotInsert}</button>
                </div>
            ) : null}
        </div>
    );
}

function creationConversationBucket(updatedAt: string): "today" | "yesterday" | "week" | "earlier" {
    const at = new Date(updatedAt).getTime();
    if (!Number.isFinite(at)) return "earlier";
    const nowStart = new Date();
    nowStart.setHours(0, 0, 0, 0);
    const targetStart = new Date(at);
    targetStart.setHours(0, 0, 0, 0);
    const days = Math.round((nowStart.getTime() - targetStart.getTime()) / 86400000);
    if (days <= 0) return "today";
    if (days === 1) return "yesterday";
    if (days < 7) return "week";
    return "earlier";
}

export function CreationHistoryDrawer({ open, conversations, activeId, onNew, onClose, onSelect, onDelete, onRename }: { open: boolean; conversations: CreationConversation[]; activeId: string; onNew: () => void; onClose: () => void; onSelect: (conversation: CreationConversation) => void; onDelete: (conversation: CreationConversation) => void; onRename: (conversation: CreationConversation, title: string) => void }) {
    const create = useCopy().video.create;
    const [keyword, setKeyword] = useState("");
    const [renamingId, setRenamingId] = useState<string | null>(null);
    const [menuOpenId, setMenuOpenId] = useState<string | null>(null);
    const renameInputRef = useRef<HTMLInputElement>(null);
    const skipRenameCommitRef = useRef(false);

    const assistantName = useAppearanceStore((state) => state.appearance.brandName);
    const exportUser = useUserStore((state) => state.user);

    const { message: drawerToast } = App.useApp();
    const untitled = create.untitledThread;
    const bucketLabel = { today: create.today, yesterday: create.yesterday, week: create.week, earlier: create.earlier };

    useEffect(() => {
        if (!open) return;
        setKeyword("");
        setRenamingId(null);
        setMenuOpenId(null);
    }, [open]);

    const commitRename = (conversation: CreationConversation) => {
        const shouldSkip = skipRenameCommitRef.current;
        skipRenameCommitRef.current = false;
        const value = renameInputRef.current?.value.trim() || "";
        setRenamingId(null);
        if (shouldSkip || !value) return;
        const original = conversation.title.trim() || untitled;
        if (value === original) return;
        onRename(conversation, value);
    };

    const cancelRename = () => {
        skipRenameCommitRef.current = true;
        setRenamingId(null);
    };

    const beginRename = (conversation: CreationConversation) => {
        skipRenameCommitRef.current = false;
        setMenuOpenId(null);
        setRenamingId(conversation.id);
    };

    const visibleConversations = useMemo(() => {
        const query = keyword.trim().toLowerCase();
        if (!query) return conversations;
        return conversations.filter((conversation) => {
            const latest = conversationPreviewMessage(conversation);
            const searchable = [
                conversation.title,
                ...conversation.messages.flatMap((message) => [message.content, displayCreationPrompt(message.content, message.references || [])]),
                latest?.mode ? creationModeLabel(latest.mode, create) : create.work,
                formatConversationTime(conversation.updatedAt, create),
            ].filter(Boolean).join(" ").toLowerCase();
            return searchable.includes(query);
        });
    }, [conversations, create, keyword]);

    return <AppDrawer flush open={open} onClose={onClose} placement="right" size="min(440px, 100vw)" closeIcon={<X className="size-4" />} className="creation-history-drawer" rootClassName="creation-history-drawer-root" title={<div className="creation-history-title"><span>{create.history}</span><small>{create.threadsCount.replace("{n}", String(conversations.length))}</small></div>}>
        <div className="creation-history-content">
            <label className="creation-history-search">
                <Search aria-hidden="true" />
                <input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder={create.searchThreads} aria-label={create.searchThreads} />
            </label>

            <button type="button" className="creation-history-new" onClick={onNew}><span className="creation-history-new-icon"><Plus /></span><span className="creation-history-new-copy"><strong>{create.newThread}</strong><small>{create.newThreadHint}</small></span></button>
            {visibleConversations.length ? <ul className="creation-history-list" aria-label={create.history}>
                {visibleConversations.flatMap((conversation, index) => {
                    const showGroupHead = !keyword.trim() && (index === 0 || creationConversationBucket(conversation.updatedAt) !== creationConversationBucket(visibleConversations[index - 1].updatedAt));
                    const latest = conversationPreviewMessage(conversation);
                    const active = conversation.id === activeId;
                    const HistoryTypeIcon = latest?.mode === "video" ? Clapperboard : latest?.mode === "image" ? ImageIcon : latest?.mode === "text" ? MessageSquareText : BookOpen;
                    return [
                        showGroupHead ? <li key={`${conversation.id}-group`} className="creation-history-group-head"><h4>{bucketLabel[creationConversationBucket(conversation.updatedAt)]}</h4></li> : null,
                        <li key={conversation.id} className={active ? "is-active" : undefined}>
                            {renamingId === conversation.id ? (
                                <div className="creation-history-rename">
                                    <input ref={renameInputRef} className="creation-history-rename-input" defaultValue={conversation.title.trim() || untitled} aria-label={create.renameThread} autoFocus onFocus={(event) => event.currentTarget.select()} onKeyDown={(event) => { if (event.key === "Enter") { event.preventDefault(); event.currentTarget.blur(); } else if (event.key === "Escape") { cancelRename(); } }} onBlur={() => commitRename(conversation)} />
                                </div>
                            ) : (
                                <div className={menuOpenId === conversation.id ? "creation-history-row is-menu-open" : "creation-history-row"}>
                                    <button type="button" className="creation-history-item-main" aria-current={active ? "page" : undefined} onClick={() => { setMenuOpenId(null); onSelect(conversation); }}>
                                        <span className="creation-history-item-icon" aria-hidden="true"><HistoryTypeIcon /></span>
                                        <span className="creation-history-item-text">
                                            <strong className="creation-history-item-heading">{conversation.title.trim() || untitled}</strong>
                                        <span className="creation-history-snippet">{latest ? <><em>{latest.mode ? creationModeLabel(latest.mode, create) : create.work}</em><span>{displayCreationPrompt(latest.content, latest.references || []).trim() || create.notStarted}</span></> : <><em>{create.work}</em><span>{create.notStarted}</span></>}</span>
                                        </span>
                                    </button>
                                    <span className="creation-history-time-slot" aria-hidden={menuOpenId === conversation.id}><time dateTime={conversation.updatedAt}>{formatHistoryRelativeTime(conversation.updatedAt, create)}</time></span>
                                    <Dropdown trigger={["click"]} placement="bottomRight" open={menuOpenId === conversation.id} onOpenChange={(open) => setMenuOpenId(open ? conversation.id : null)} overlayClassName="creation-history-menu-overlay" menu={{ items: [{ key: "rename", label: create.renameThread, icon: <Pencil /> }, { key: "export", label: create.exportThread, icon: <Download /> }, { key: "delete", label: create.deleteThreadOk, danger: true, icon: <Trash2 /> }], onClick: ({ key }) => { setMenuOpenId(null); if (key === "rename") { beginRename(conversation); } else if (key === "export") { downloadCreationConversation(conversation, assistantName, exportUser?.displayName || create.you, untitled); drawerToast.success(create.exportedMd); } else { onDelete(conversation); } } }}>
                                        <button type="button" className={menuOpenId === conversation.id ? "creation-history-more is-open" : "creation-history-more"} aria-label={create.moreOnThread.replace("{title}", conversation.title.trim() || untitled)} onClick={(event) => event.preventDefault()}><MoreHorizontal /></button>
                                    </Dropdown>
                                </div>
                            )}
                        </li>,
                    ];
                })}
            </ul> : <div className="creation-history-empty">{keyword.trim() ? create.noMatch : create.emptyRecent}</div>}
        </div>
    </AppDrawer>;
}

function shotStatusLabel(status: CreationShotRailEntry["result"] extends infer T ? T extends { status: infer S } ? S : never : never, create: ReturnType<typeof useCopy>["video"]["create"]) {
    if (status === "done") return create.shotDone;
    if (status === "error") return create.shotError;
    if (status === "pending") return create.shotPending;
    if (status === "cancelled") return create.shotStopped;
    return create.shotIdle;
}

export function CreationWorkspaceToolbar({ onNewConversation, onOpenHistory, onContinueCanvas, openingCanvas }: { shots?: CreationShotRailEntry[]; onJumpToShot?: (shot: CreationShotRailEntry) => void; onNewConversation: () => void; onOpenHistory: () => void; onContinueCanvas: () => void; openingCanvas: boolean }) {
    const create = useCopy().video.create;
    const mount = useWorkspaceTopBarMount();
    const hosted = isYoyoCanvasHost();
    const toolbar = <header className="creation-thread-toolbar">
        <div className="creation-toolbar-actions">
            <Button size="small" loading={openingCanvas} onClick={onContinueCanvas} data-testid="creation-open-on-board">{openingCanvas ? create.openingBoard : create.openOnBoard}</Button>
            <Tooltip title={create.newThread}><button type="button" aria-label={create.newThread} className="creation-toolbar-action" onClick={onNewConversation}><Plus /></button></Tooltip>
            <Tooltip title={create.history}><button type="button" aria-label={create.historyAria} className="creation-toolbar-action" onClick={onOpenHistory}><History /></button></Tooltip>
        </div>
    </header>;
    if (mount) return createPortal(toolbar, mount);
    if (hosted) return toolbar;
    if (mount === null) return null;
    return toolbar;
}

export function CreationShotFilmstrip({ shots, onJumpToShot }: { shots: CreationShotRailEntry[]; onJumpToShot: (shot: CreationShotRailEntry) => void }) {
    const create = useCopy().video.create;
    return (
        <nav className="creation-shot-filmstrip" aria-label={create.shotStrip} data-testid="creation-shot-filmstrip">
            {shots.length ? shots.map((shot) => {
                const resultStatus = shot.result?.status;
                return (
                    <button
                        key={shot.key}
                        type="button"
                        className={`creation-shot-chip is-${resultStatus || "idle"}`}
                        onClick={() => onJumpToShot(shot)}
                    >
                        <span className="creation-shot-chip-index">{shot.ordinal}</span>
                        <span className="creation-shot-chip-state">{shotStatusLabel(resultStatus, create)}</span>
                    </button>
                );
            }) : <p className="creation-shot-filmstrip-empty">{create.shotEmpty}</p>}
        </nav>
    );
}

export function CreationRecentThreads({ conversations, onSelect, onNew }: { conversations: CreationConversation[]; onSelect: (conversation: CreationConversation) => void; onNew: () => void }) {
    const create = useCopy().video.create;
    const [query, setQuery] = useState("");
    const needle = query.trim().toLowerCase();
    const recent = conversations
        .filter((conversation) => conversation.messages.length > 0)
        .filter((conversation) => {
            if (!needle) return true;
            const haystack = [
                conversation.title,
                ...conversation.messages.map((message) => displayCreationPrompt(message.content, message.references || [])),
            ].join(" ").toLowerCase();
            return haystack.includes(needle);
        })
        .slice(0, 8);
    return (
        <section className="creation-recent-threads" aria-label={create.recent} data-testid="creation-recent-threads">
            <header className="creation-recent-head">
                <strong>{create.recent}</strong>
                <button type="button" onClick={onNew}>{create.newThread}</button>
            </header>
            <label className="creation-recent-search">
                <Search aria-hidden="true" />
                <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={create.searchThreads} aria-label={create.searchThreads} />
            </label>
            {recent.length ? (
                <ul>
                    {recent.map((conversation) => (
                        <li key={conversation.id}>
                            <button type="button" className="creation-recent-row" onClick={() => onSelect(conversation)}>
                                <span className="creation-recent-title">{conversation.title.trim() || create.untitledThread}</span>
                            </button>
                        </li>
                    ))}
                </ul>
            ) : <p className="creation-recent-empty">{needle ? create.noMatch : create.emptyRecent}</p>}
        </section>
    );
}

export function CreationMessageView({ item, shotNumber, onRetryFailure, onCreateVariant, onEditUserMessage, onContinueCanvas, openingCanvas, onStop }: { item: CreationMessage; shotNumber: number; onRetryFailure: () => void; onCreateVariant: () => void; onEditUserMessage: (text: string) => void; onContinueCanvas: (ids?: string[]) => void; openingCanvas: boolean; onStop?: () => void }) {
    const create = useCopy().video.create;
    const hosted = isYoyoCanvasHost();
    const brandName = useAppearanceStore((state) => state.appearance.brandName);
    if (item.role === "user") return <CreationUserMessage item={item} shotNumber={shotNumber} onEditUserMessage={onEditUserMessage} />;
    const mode = item.mode || "text";
    const stateLabel = item.status === "pending" ? create.statusGenerating : item.status === "cancelled" ? create.statusStopped : item.status === "error" ? create.statusFailed : "";
    const stopButton = (item.status === "pending" || item.status === "streaming") && onStop ? <button type="button" className="creation-message-variant-action" onClick={onStop}>{create.stop}</button> : null;
    const shot = shotNumber > 0 ? <span className="creation-shot-badge">{create.shotN.replace("{n}", String(shotNumber))}</span> : null;
    const heading =
        mode !== "text" ? (
            <>{shot}<span className="creation-message-mark"><Clapperboard /></span><strong>{mode === "image" ? create.imageGen : create.videoGen}</strong>{item.status === "pending" ? <span className="creation-message-progress-copy">{mode === "video" ? create.generatingVideo : create.generatingImage}</span> : item.status === "done" ? <span className="creation-message-progress-copy">{mode === "video" ? create.videoReady : create.imageReady}</span> : null}{item.status === "done" ? <button type="button" className="creation-message-variant-action" onClick={onCreateVariant}><RefreshCw />{create.sameAgain}</button> : null}{stopButton}{item.createdAt ? <time dateTime={item.createdAt}>{formatMessageTime(item.createdAt)}</time> : null}{stateLabel ? <span className={`creation-message-state is-${item.status}`}>{stateLabel}</span> : null}</>
        ) : (
            <>{shot}<span className="creation-message-mark"><Clapperboard /></span><strong>{brandName}</strong>{stopButton}{item.createdAt ? <time dateTime={item.createdAt}>{formatMessageTime(item.createdAt)}</time> : null}{stateLabel ? <span className={`creation-message-state is-${item.status}`}>{stateLabel}</span> : null}</>
        );
    const media = <MediaResult item={item} onRetryFailure={onRetryFailure} onCreateVariant={onCreateVariant} onContinueCanvas={onContinueCanvas} openingCanvas={openingCanvas} />;
    const toolStatus: GenerationToolStatus = item.status === "pending" ? "running" : item.status === "error" ? "error" : item.status === "cancelled" ? "cancelled" : "completed";
    return <article className={`creation-assistant-message is-${mode}`}>
        {mode === "text" ? <><div className="creation-message-heading">{heading}</div>{item.reasoning ? <div className="creation-message-reasoning-wrap"><MessageReasoning reasoning={item.reasoning} isStreaming={item.status === "streaming"} /></div> : null}<div className="creation-message-content">{item.content ? <AIMessageMarkdown isStreaming={item.status === "streaming"}>{item.content}</AIMessageMarkdown> : <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}><WorkingDots dotSize={5} gap={2} /><span>{create.generatingText}</span></span>}</div></> : hosted ? <div className="creation-media-brick"><div className="creation-message-heading">{heading}</div>{media}</div> : <GenerationToolCard status={toolStatus} heading={heading}>{media}</GenerationToolCard>}
        {item.error && mode === "text" ? <div className="creation-message-error"><span>{generationErrorMessage(item.error)}</span><button type="button" onClick={onRetryFailure}><RefreshCw />{create.retryGenerate}</button></div> : null}
    </article>;
}

function CreationUserMessage({ item, shotNumber, onEditUserMessage }: { item: CreationMessage; shotNumber: number; onEditUserMessage: (text: string) => void }) {
    const create = useCopy().video.create;
    const [previewUrl, setPreviewUrl] = useState("");
    const [previewType, setPreviewType] = useState<"image" | "video">("image");
    const copyText = useCopyText();
    const visiblePrompt = displayCreationPrompt(item.content, item.references || []);
    const user = useUserStore((state) => state.user);
    const userAvatarUrl = user?.avatarUrl?.trim();
    return <article className="creation-user-message">
        <div className="creation-user-message-meta">{shotNumber > 0 ? <span className="creation-shot-badge">{create.shotN.replace("{n}", String(shotNumber))}</span> : null}{item.createdAt ? <time dateTime={item.createdAt}>{formatMessageTime(item.createdAt)}</time> : null}<strong>{user?.displayName || create.you}</strong><span className="creation-user-avatar">{userAvatarUrl ? <img src={userAvatarUrl} alt="" referrerPolicy="no-referrer" loading="lazy" decoding="async" /> : <UserRound />}</span></div>
        <div className="creation-user-message-copy-wrap"><p>{visiblePrompt}</p></div>
        {item.references?.length ? <CreationMessageReferences references={item.references} /> : null}
        {item.attachments?.length ? <div className="creation-user-message-attachments">{item.attachments.map((attachment) => {
            const kind = creationAttachmentKind(attachment);
            const previewable = kind === "image" || kind === "video";
            const url = attachment.previewUrl || ("dataUrl" in attachment ? attachment.dataUrl : attachment.url) || "";
            const imageUrl = kind === "image" ? resolveResourceUrl(attachment.storageKey, url) : "";
            const previewUrl = kind === "image" ? imageUrl : url;
            return <button key={attachment.id} type="button" className={!previewable ? "is-file" : undefined} onClick={() => { if (!previewable) return; setPreviewType(kind === "video" ? "video" : "image"); setPreviewUrl(kind === "video" ? attachment.url || "" : previewUrl); }} aria-label={previewable ? create.previewNamed.replace("{name}", attachment.name || create.addRef) : attachment.name || create.addRef} disabled={previewable && !previewUrl}>{kind === "video" ? <video src={attachment.url || ""} poster={url !== attachment.url ? url : undefined} muted playsInline preload="metadata" /> : kind === "image" ? <CachedResourceImage storageKey={attachment.storageKey} src={imageUrl} alt={attachment.name || create.addRef} width={44} height={44} loading="lazy" decoding="async" /> : kind === "audio" ? <Music2 /> : <FileText />}{previewable ? <span aria-hidden="true"><Maximize2 /></span> : null}</button>;
        })}</div> : null}
        <div className="creation-user-message-actions"><Tooltip title={create.copyPrompt}><button type="button" className="creation-user-message-copy" aria-label={create.copyPrompt} onClick={() => copyText(visiblePrompt, create.copyPromptDone)}><Copy /></button></Tooltip><Tooltip title={create.editResend}><button type="button" className="creation-user-message-edit" aria-label={create.editResend} onClick={() => onEditUserMessage(visiblePrompt)}><Pencil /></button></Tooltip></div>
        <CreationMediaPreviewModal url={previewUrl} type={previewType} onClose={() => setPreviewUrl("")} />
    </article>;
}

function MediaResult({ item, onRetryFailure, onCreateVariant, onContinueCanvas, openingCanvas }: { item: CreationMessage; onRetryFailure: () => void; onCreateVariant: () => void; onContinueCanvas: (ids?: string[]) => void; openingCanvas: boolean }) {
    const create = useCopy().video.create;
    const [previewUrl, setPreviewUrl] = useState("");
    const [previewType, setPreviewType] = useState<"image" | "video">("image");
    const assets = useAssetStore((state) => state.assets);
    const resultUrls = item.resultUrls || [];
    const resultAssetIds = resultUrls.length ? creationResultAssetIds(assets, { messageId: item.id, taskIds: item.taskIds || [], resultUrls }) : [];
    const canContinueWithResults = resultUrls.length > 0 && resultAssetIds.length === resultUrls.length;
    if (item.status === "pending") return <CreationMediaPending mode={item.mode || "image"} ratio={item.settings?.ratio} />;
    if ((item.status === "error" || item.status === "cancelled") && !resultUrls.length) return <div className="creation-media-error"><span>{item.status === "cancelled" ? item.content || create.statusStopped : generationErrorMessage(item.error || create.statusFailed)}</span><button type="button" onClick={onRetryFailure}><RefreshCw />{create.retryGenerate}</button></div>;
    if (!resultUrls.length) return <div className="creation-media-empty">{create.noPreview} <button type="button" onClick={onRetryFailure}>{create.retryGenerate}</button></div>;
    const isVideo = item.mode === "video";
    return <div className="creation-media-result">
        {isVideo ? <button type="button" className="creation-video-result" onClick={() => { setPreviewType("video"); setPreviewUrl(resultUrls[0]); }} aria-label={create.previewVideo}><video muted preload="metadata" src={resultUrls[0]} /><span><Maximize2 />{create.previewVideo}</span></button> : <div className="creation-image-result-grid">{resultUrls.map((url) => <button key={url} type="button" className="creation-image-result" onClick={() => { setPreviewType("image"); setPreviewUrl(url); }} aria-label={create.previewImage}><img src={url} alt={create.previewImage} /><span><Maximize2 /></span></button>)}</div>}
        <div className="creation-media-actions"><span>{isVideo ? create.videoResult : create.imageCount.replace("{n}", String(resultUrls.length))}</span><Button type="link" size="small" loading={openingCanvas} disabled={!canContinueWithResults} data-testid="creation-open-on-board-result" title={canContinueWithResults ? undefined : create.waitAssets} onClick={() => onContinueCanvas(resultAssetIds)}>{create.openOnBoard}</Button>{resultUrls.map((url, index) => <a key={`${url}-download`} href={url} download>{resultUrls.length > 1 ? create.downloadN.replace("{n}", String(index + 1)) : <><Download />{create.download}</>}</a>)}</div>
        <CreationMediaPreviewModal url={previewUrl} type={previewType} onClose={() => setPreviewUrl("")} />
    </div>;
}

function CreationMediaPending({ mode, ratio }: { mode: CreationMode; ratio?: string }) {
    const create = useCopy().video.create;
    return <div className={`creation-media-pending is-${mode}`} style={{ aspectRatio: creationMediaAspectRatio(ratio, mode) }} aria-live="polite"><span className="creation-media-pending-icon"><WorkingDots dotSize={7} gap={3} minOpacity={0.3} /></span><span className="sr-only">{mode === "video" ? create.generatingVideo : create.generatingImage}</span></div>;
}

function CreationMessageReferences({ references }: { references: CreationReference[] }) {
    return <div className="creation-user-message-references" aria-label="本次引用">{references.map((reference) => {
        const Icon = reference.kind === "skill" ? BookOpen : reference.kind === "image" ? ImageIcon : reference.kind === "video" ? Film : reference.kind === "audio" ? Music2 : FileText;
        const imageUrl = reference.kind === "image" ? resolveResourceUrl(reference.storageKey, reference.previewUrl) : reference.previewUrl;
        return <span key={reference.id} className="creation-user-message-reference">{imageUrl && reference.kind === "video" ? <video src={imageUrl} muted playsInline preload="metadata" aria-label={reference.label} /> : imageUrl && reference.kind === "image" ? <CachedResourceImage storageKey={reference.storageKey} src={imageUrl} alt="" loading="lazy" decoding="async" /> : <Icon />}<span>{reference.label}</span></span>;
    })}</div>;
}

function CreationMediaPreviewModal({ url, type, onClose }: { url: string; type: "image" | "video"; onClose: () => void }) {
    const create = useCopy().video.create;
    if (type === "image") return <CanvasImagePreview src={url} alt={create.previewMedia} onClose={onClose} />;

    return <AppModal flush open={Boolean(url)} title={null} footer={null} centered destroyOnHidden width="min(1160px, calc(100vw - 32px))" onCancel={onClose} className="creation-media-preview-modal">
        {url ? <video controls autoPlay className="creation-media-preview-video" src={url} /> : null}
    </AppModal>;
}

function CreationAttachmentThumbnail({ item, onPreview, onRemove, frameRole }: {
    item: CreationAttachment;
    onPreview: (type: "image" | "video", url: string) => void;
    onRemove: (id: string) => void;
    frameRole?: string;
}) {
    const create = useCopy().video.create;
    const kind = creationAttachmentKind(item);
    const previewable = kind === "image" || kind === "video";
    const url = (kind === "video" ? item.url : item.previewUrl) || "";
    const imageUrl = kind === "image" ? resolveResourceUrl(item.storageKey, item.previewUrl) : "";
    const previewUrl = kind === "image" ? imageUrl : url;
    const content = kind === "video" ? <video src={item.url} poster={item.previewUrl !== item.url ? item.previewUrl : undefined} muted playsInline preload="metadata" aria-label={item.name} /> : kind === "image" ? <CachedResourceImage storageKey={item.storageKey} src={imageUrl} alt={item.name} loading="lazy" decoding="async" fallback={<span className="creation-chat-file-icon"><ImageIcon /></span>} /> : <span className="creation-chat-file-icon">{kind === "audio" ? <Music2 /> : <FileText />}<em>{item.name}</em></span>;
    return <div className="creation-reference-card-content" style={frameRole ? { position: "relative" } : undefined}>
        {previewable ? <button type="button" className="creation-reference-card-preview" onClick={() => onPreview(kind === "video" ? "video" : "image", previewUrl)} aria-label={create.previewNamed.replace("{name}", item.name)} disabled={!previewUrl}>{content}<span aria-hidden="true"><Maximize2 /></span></button> : <div className="creation-reference-card-preview is-file" aria-label={item.name}>{content}</div>}
        {frameRole ? <span className="creation-reference-frame-role">{frameRole}</span> : null}
        <button type="button" className="creation-reference-card-remove" onPointerDownCapture={(event) => event.stopPropagation()} onMouseDownCapture={(event) => event.stopPropagation()} onClick={(event) => { event.stopPropagation(); onRemove(item.id); }} aria-label={create.removeNamed.replace("{name}", item.name)}><X /></button>
    </div>;
}

type ComposerProps = {
    variant: "empty" | "thread";
    mode: CreationMode;
    prompt: string;
    setPrompt: (value: string) => void;
    busy: boolean;
    generationActive: boolean;
    referenceReplacementBusy: boolean;
    attachments: CreationAttachment[];
    referenceImageSize?: { width: number; height: number };
    maxReferences: number;
    references: CreationReference[];
    onRemoveAttachment: (id: string) => void;
    onClearAttachments: () => void;
    onClearComposer: () => void;
    onReorderAttachments: (attachments: CreationAttachment[]) => void;
    onReplaceAttachment: (targetAttachmentId: string, replacement: CreationAttachment) => void;
    onReplaceReferenceFiles: (targetAttachmentId: string, files: File[]) => void;
    onOpenLibrary: () => void;
    onModeChange: (mode: CreationMode) => void;
    model: string;
    modelRequirements: ModelRequirements;
    videoProfile: VideoCapabilityConfig;
    imageProfile: ImageCapabilityConfig;
    config: AiConfig;
    onModelChange: (value: string) => void;
    ratio: string;
    setRatio: (value: string) => void;
    seconds: string;
    setSeconds: (value: string) => void;
    quality: string;
    setQuality: (value: string) => void;
    videoQuality: string;
    setVideoQuality: (value: string) => void;
    count: string;
    setCount: (value: string) => void;
    textStreaming: boolean;
    setTextStreaming: (value: boolean) => void;
    textThinking: boolean;
    setTextThinking: (value: boolean) => void;
    promptOptimizerProvider: PromptOptimizerProvider | null;
    composerFocusRef: RefObject<HTMLTextAreaElement | null>;
    onPromptFocus: () => void;
    placeholderOverride?: string;
    onSubmit: () => void;
    onStop?: () => void;
};

type CreationReferenceFilter = "all" | "image" | "video" | "audio" | "file";

export function CreationComposer(props: ComposerProps) {
    const [previewUrl, setPreviewUrl] = useState("");
    const [previewType, setPreviewType] = useState<"image" | "video">("image");
    const [promptOptimizerOpen, setPromptOptimizerOpen] = useState(false);
    const [referenceFilter, setReferenceFilter] = useState<CreationReferenceFilter>("all");
    const [canDragReferences, setCanDragReferences] = useState(false);
    const [dropTargetReferenceId, setDropTargetReferenceId] = useState<string | null>(null);
    const attachmentTrackRef = useRef<HTMLUListElement>(null);
    const cardDragRef = useRef<{ startX: number; startY: number; moved: boolean } | null>(null);
    const suppressAttachmentClickRef = useRef(false);
    const [trackState, setTrackState] = useState({ canScrollLeft: false, canScrollRight: false, isExpanded: true, isDragging: false });
    const previousAttachmentCountRef = useRef(0);
    const interactionBusy = props.busy || props.referenceReplacementBusy;
    const canSubmit = Boolean(props.prompt.trim()) && !interactionBusy;
    const showStop = Boolean(props.onStop) && (props.busy || props.generationActive) && !canSubmit;
    const creditsEnabled = useUserStore((state) => state.features.creditsEnabled);
    const priceChannel = resolveModelChannel(props.config, props.model);
    const quoteRequest = useMemo(() => modelQuoteRequest(props.config, props.model, props.mode, props.modelRequirements), [props.config, props.mode, props.model, props.modelRequirements]);
    const [routeQuote, setRouteQuote] = useState<LogicalModelQuote | null>(null);
    const canOptimizePrompt = Boolean(props.promptOptimizerProvider) && (props.mode === "image" || props.mode === "video");
    const optimizerReferences = props.references.filter((reference) => reference.active && reference.kind !== "skill");
    const credits = requestCreditCost({
        channelMode: priceChannel.scope === "system" ? "remote" : "local",
        modelCosts: priceChannel.modelCosts,
        model: modelOptionName(props.model),
        count: props.mode === "image" ? props.count : 1,
        seconds: props.mode === "video" ? props.seconds : 1,
        capability: props.mode,
        config: props.config,
        requirements: props.modelRequirements,
    });
    useEffect(() => {
        if (!creditsEnabled || !quoteRequest) {
            setRouteQuote(null);
            return;
        }
        const controller = new AbortController();
        setRouteQuote(null);
        quoteModel(quoteRequest, controller.signal)
            .then(({ quote }) => setRouteQuote(quote))
            .catch(() => {
                if (!controller.signal.aborted) setRouteQuote(null);
            });
        return () => controller.abort();
    }, [creditsEnabled, quoteRequest]);
    const generationCredits = routeQuote ? routeQuote.amountMicrocredits / 1_000_000 : credits;
    const showCost = !isYoyoCanvasHost() && creditsEnabled && generationCredits !== null && generationCredits !== undefined;
    const formattedCredits = generationCredits?.toLocaleString("zh-CN", { maximumFractionDigits: 6 });
    const create = useCopy().video.create;
    const actionLabel = props.referenceReplacementBusy ? create.generating : showStop ? create.stop : interactionBusy || (props.generationActive && !canSubmit) ? create.generating : create.send;
    // 生成还没结束、输入框也是空的时候，按钮是停止。写了下一句之后才变回发送。
    const showWorkingSpinner = !showStop && (interactionBusy || (props.generationActive && !canSubmit));
    const placeholder = props.mode === "text" ? create.placeholderText : props.mode === "image" ? create.placeholderImage : create.placeholderVideo;
    const emptyPlaceholder = create.hint;
    const imageReferencesSupported = props.imageProfile.references.maxImages > 0;
    const referencesSupported = props.mode === "image" ? imageReferencesSupported : props.mode !== "video" || props.maxReferences > 0;
    const canAddMoreReferences = referencesSupported && props.attachments.length < props.maxReferences;
    const addReferenceLabel = interactionBusy ? (props.referenceReplacementBusy ? create.replacingRef : create.addRefsBusy) : canAddMoreReferences ? create.addRef : create.refLimit.replace("{n}", String(props.maxReferences));
    const referenceCounts = useMemo(() => props.attachments.reduce((counts, attachment) => {
        const kind = creationAttachmentKind(attachment);
        counts[kind] += 1;
        return counts;
    }, { image: 0, video: 0, audio: 0, file: 0 }), [props.attachments]);
    const visibleAttachments = useMemo(() => referenceFilter === "all"
        ? props.attachments
        : props.attachments.filter((attachment) => creationAttachmentKind(attachment) === referenceFilter), [props.attachments, referenceFilter]);
    const imageSettingsSupported = props.imageProfile.size.parameter !== "none" || props.imageProfile.quality.supported || props.imageProfile.maxOutputs > 1;
    const updateTrackScrollState = useCallback(() => {
        const track = attachmentTrackRef.current;
        if (!track) return;
        setTrackState((current) => ({
            ...current,
            canScrollLeft: track.scrollLeft > 1,
            canScrollRight: track.scrollLeft + track.clientWidth < track.scrollWidth - 1,
        }));
    }, []);
    const setReferencePanelExpanded = useCallback((isExpanded: boolean) => {
        setTrackState((current) => ({ ...current, isExpanded }));
        if (!isExpanded) setReferenceFilter("all");
    }, []);
    useEffect(() => {
        const hadAttachments = previousAttachmentCountRef.current > 0;
        if (!props.attachments.length) setReferencePanelExpanded(false);
        else if (!hadAttachments) setReferencePanelExpanded(true);
        previousAttachmentCountRef.current = props.attachments.length;
        updateTrackScrollState();
    }, [props.attachments.length, setReferencePanelExpanded, updateTrackScrollState]);
    useEffect(() => {
        const query = window.matchMedia("(hover: hover) and (pointer: fine)");
        const update = () => setCanDragReferences(query.matches);
        update();
        query.addEventListener("change", update);
        return () => query.removeEventListener("change", update);
    }, []);
    useEffect(() => {
        const frame = window.requestAnimationFrame(updateTrackScrollState);
        return () => window.cancelAnimationFrame(frame);
    }, [referenceFilter, trackState.isExpanded, updateTrackScrollState, visibleAttachments.length]);
    const beginCardDrag = (event: PointerEvent<HTMLElement>) => {
        if (event.button !== 0 || interactionBusy || !trackState.isExpanded) return;
        if ((event.target as HTMLElement).closest(".creation-reference-card-remove")) return;
        cardDragRef.current = { startX: event.clientX, startY: event.clientY, moved: false };
    };
    const endCardDrag = (event: PointerEvent<HTMLElement>) => {
        const drag = cardDragRef.current;
        if (!drag) return;
        cardDragRef.current = null;
        if (drag.moved) {
            suppressAttachmentClickRef.current = true;
            window.setTimeout(() => { suppressAttachmentClickRef.current = false; }, 0);
        }
        if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
        setTrackState((current) => ({ ...current, isDragging: false }));
    };
    const moveCardDrag = (event: PointerEvent<HTMLElement>) => {
        const drag = cardDragRef.current;
        if (!drag || drag.moved) return;
        if (Math.hypot(event.clientX - drag.startX, event.clientY - drag.startY) <= 4) return;
        drag.moved = true;
        setTrackState((current) => ({ ...current, isDragging: true, isExpanded: true }));
    };
    const previewAttachment = (type: "image" | "video", url: string) => {
        if (suppressAttachmentClickRef.current || cardDragRef.current?.moved) return;
        setPreviewType(type);
        setPreviewUrl(url);
    };
    const reorderVisibleAttachments = useCallback((next: CreationAttachment[]) => {
        if (referenceFilter === "all") {
            props.onReorderAttachments(next);
            return;
        }
        const visibleIds = new Set(visibleAttachments.map((attachment) => attachment.id));
        const reordered = [...next];
        props.onReorderAttachments(props.attachments.map((attachment) => visibleIds.has(attachment.id) ? reordered.shift() || attachment : attachment));
    }, [props.attachments, props.onReorderAttachments, referenceFilter, visibleAttachments]);
    useEffect(() => {
        if (!canOptimizePrompt) setPromptOptimizerOpen(false);
    }, [canOptimizePrompt]);

    const scrollAttachmentTrack = (direction: -1 | 1) => {
        const track = attachmentTrackRef.current;
        if (!track) return;
        track.scrollBy({ left: direction * Math.max(track.clientWidth * 0.72, 120), behavior: "smooth" });
        window.setTimeout(updateTrackScrollState, 180);
    };
    const imageReferenceAtPoint = (x: number, y: number) => {
        for (const element of document.elementsFromPoint(x, y)) {
            const chip = element.closest<HTMLElement>("[data-mention-reference-id]");
            const referenceId = chip?.dataset.mentionReferenceId;
            const reference = referenceId ? props.references.find((item) => item.id === referenceId) : undefined;
            if (reference?.kind === "image" && reference.attachmentId) return reference;
        }
        return undefined;
    };
    const composer = <div className="creation-composer-shell">
        <div className={`creation-chat-composer is-${props.variant}`}>
        <div className="creation-chat-writing-surface">
            <div className="creation-chat-editor">
                <CanvasResourceMentionTextarea ref={props.composerFocusRef} value={props.prompt} references={props.references} mentionMenuWidth={400} sendOnEnter onFocus={props.onPromptFocus} onChange={props.setPrompt} onSubmit={props.onSubmit} containerClassName="creation-chat-mention-container" className="creation-chat-mention-editor creation-scrollbar" style={{ color: "var(--creation-text)" }} placeholder={props.placeholderOverride || (props.variant === "empty" ? emptyPlaceholder : placeholder)} aria-label={create.promptAria} spellCheck disabled={interactionBusy} activeDropReferenceId={dropTargetReferenceId} onReferenceFilesDrop={(reference, files) => { const target = props.references.find((item) => item.id === reference.id); if (target?.attachmentId) props.onReplaceReferenceFiles(target.attachmentId, files); }} />
                {props.mode === "video" ? <CreationShotStructure prompt={props.prompt} setPrompt={props.setPrompt} disabled={interactionBusy} create={create} /> : null}
                {props.attachments.length || referencesSupported ? <div className={`creation-reference-panel${trackState.isExpanded ? " is-expanded" : ""}`} aria-busy={interactionBusy}>
                    {trackState.isExpanded ? <div className="creation-reference-panel-header">
                        <div className="creation-reference-filter-tabs" role="group" aria-label={create.filterRefs}>
                            {([
                                { id: "all", label: create.filterAll, count: props.attachments.length },
                                { id: "image", label: create.filterImage, count: referenceCounts.image },
                                { id: "video", label: create.filterVideo, count: referenceCounts.video },
                                { id: "audio", label: create.filterAudio, count: referenceCounts.audio },
                                { id: "file", label: create.filterFile, count: referenceCounts.file },
                            ] as const).map((filter) => <button key={filter.id} type="button" aria-pressed={referenceFilter === filter.id} className={referenceFilter === filter.id ? "is-active" : undefined} onClick={() => setReferenceFilter(filter.id)}>{filter.label}{filter.count ? ` (${filter.count})` : ""}</button>)}
                        </div>
                        <div className="creation-reference-panel-actions">
                            {props.attachments.length ? <button type="button" onClick={props.onClearAttachments} disabled={interactionBusy}>{create.clearRefs}</button> : null}
                            <Tooltip title={create.collapseRefs}><button type="button" className="creation-reference-panel-collapse" onClick={() => setReferencePanelExpanded(false)} aria-label={create.collapseRefs}><Minimize2 aria-hidden="true" /></button></Tooltip>
                        </div>
                    </div> : null}
                    <div className="creation-reference-track-wrapper">
                        <div className="creation-reference-stack-shell">
                                {trackState.canScrollLeft ? <button type="button" className="creation-reference-track-button is-left" onClick={() => scrollAttachmentTrack(-1)} aria-label={create.scrollRefsLeft} title={create.scrollRefsLeft}><ChevronLeft aria-hidden="true" /></button> : null}
                            <Reorder.Group<CreationAttachment[]>
                                as="ul"
                                ref={attachmentTrackRef}
                                className={`creation-reference-track${trackState.isExpanded ? " is-expanded" : ""}${trackState.isDragging ? " is-dragging" : ""}${visibleAttachments.length ? "" : " is-empty"}`}
                                axis="x"
                                values={visibleAttachments}
                                onReorder={reorderVisibleAttachments}
                                layoutScroll
                                role="list"
                                aria-label={create.refsTrack}
                                onScroll={updateTrackScrollState}
                            >
                                {visibleAttachments.map((item) => <Reorder.Item<CreationAttachment>
                                    key={item.id}
                                    value={item}
                                    layout="position"
                                    drag={trackState.isExpanded && canDragReferences && !interactionBusy}
                                    className="creation-reference-stack-card"
                                    onPointerDown={beginCardDrag}
                                    onPointerMove={moveCardDrag}
                                    onPointerUp={endCardDrag}
                                    onPointerCancel={endCardDrag}
                                    onDragStart={() => { setDropTargetReferenceId(null); setTrackState((current) => ({ ...current, isDragging: true, isExpanded: true })); }}
                                    onDrag={(_, info) => {
                                        if (creationAttachmentKind(item) !== "image") return;
                                        const target = imageReferenceAtPoint(info.point.x, info.point.y);
                                        setDropTargetReferenceId(target?.attachmentId !== item.id ? target?.id || null : null);
                                    }}
                                    onDragEnd={(_, info) => {
                                        const target = creationAttachmentKind(item) === "image" ? imageReferenceAtPoint(info.point.x, info.point.y) : undefined;
                                        setDropTargetReferenceId(null);
                                        setTrackState((current) => ({ ...current, isDragging: false, isExpanded: true }));
                                        if (target?.attachmentId && target.attachmentId !== item.id) props.onReplaceAttachment(target.attachmentId, item);
                                    }}
                                >
                                    <CreationAttachmentThumbnail item={item} onPreview={previewAttachment} onRemove={props.onRemoveAttachment} frameRole={props.mode === "video" && creationAttachmentKind(item) === "image" ? videoFrameRoleLabel(props.attachments, item.id, create) : undefined} />
                                </Reorder.Item>)}
                                {!visibleAttachments.length && props.attachments.length ? <li className="creation-reference-filter-empty">{create.noRefsOfType}</li> : null}
                                {referencesSupported ? <li className="creation-reference-add-slot"><Tooltip title={addReferenceLabel}><button type="button" className="creation-reference-add-button" onClick={props.onOpenLibrary} disabled={interactionBusy || !canAddMoreReferences} aria-label={addReferenceLabel}><Plus aria-hidden="true" /><span>{create.addRef}</span></button></Tooltip></li> : null}
                            </Reorder.Group>
                            {trackState.canScrollRight ? <button type="button" className="creation-reference-track-button is-right" onClick={() => scrollAttachmentTrack(1)} aria-label={create.scrollRefsRight} title={create.scrollRefsRight}><ChevronRight aria-hidden="true" /></button> : null}
                            {!trackState.isExpanded && props.attachments.length ? <Tooltip title={create.expandRefs}><button type="button" className="creation-reference-panel-expand" onClick={() => setReferencePanelExpanded(true)} aria-label={create.expandRefs} aria-expanded="false"><Maximize2 aria-hidden="true" /></button></Tooltip> : null}
                        </div>
                    </div>
                </div> : null}
            </div>
        </div>
        <footer className="creation-chat-dock">
            <div className="creation-chat-controls">
                {props.variant === "thread" ? <ModePicker mode={props.mode} onModeChange={props.onModeChange} /> : null}
                <VoiceRecordingButton
                    className="creation-voice-trigger"
                    disabled={interactionBusy}
                    onTranscribed={(text) => props.setPrompt(props.prompt.trim() ? `${props.prompt} ${text}` : text)}
                />
                {canOptimizePrompt ? <Tooltip title={create.optimizePrompt}>
                    <button
                        type="button"
                        className="creation-chat-control"
                        onClick={() => setPromptOptimizerOpen(true)}
                        aria-label={create.optimizePrompt}
                        aria-expanded={promptOptimizerOpen}
                        aria-haspopup="dialog"
                    >
                        <SlidersHorizontal />
                        <span>{create.optimize}</span>
                    </button>
                </Tooltip> : null}
				<ModelPicker config={props.config} value={props.model} onChange={props.onModelChange} capability={props.mode} requirements={props.modelRequirements} className="creation-model-picker" placeholder={`选择${modeLabels[props.mode]}模型`} showSelectedPrice={false} showOptionPrices variant="creation" />
                {props.mode === "video" || (props.mode === "image" && imageSettingsSupported) ? <GenerationSettingsMenu {...props} /> : null}
                {props.mode === "video" ? <DurationMenu profile={props.videoProfile} seconds={props.seconds} onChange={props.setSeconds} /> : null}
                {props.mode === "text" ? <>
                    <Tooltip title={interactionBusy ? create.switchNextSend : (props.textStreaming ? create.streamingOn : create.streamingOff)}><button type="button" className="creation-chat-control" aria-pressed={props.textStreaming} disabled={interactionBusy} onClick={() => props.setTextStreaming(!props.textStreaming)}><Waves /><span>{create.streaming}</span></button></Tooltip>
                    <Tooltip title={interactionBusy ? create.switchNextSend : (props.textThinking ? create.thinkingOn : create.thinkingOff)}><button type="button" className="creation-chat-control" aria-pressed={props.textThinking} disabled={interactionBusy} onClick={() => props.setTextThinking(!props.textThinking)}><Brain /><span>{create.thinking}</span></button></Tooltip>
                </> : null}
                {props.prompt.trim() || props.attachments.length || props.references.some((reference) => reference.active) ? <Tooltip title={create.clearComposer}><button type="button" className="creation-chat-control is-clear" onClick={props.onClearComposer} disabled={interactionBusy} aria-label={create.clearComposer}><Trash2 /><span>{create.clear}</span></button></Tooltip> : null}
            </div>
            <Button
                type="text"
                className={`creation-submit ${showCost ? "has-cost" : ""}`}
                disabled={props.referenceReplacementBusy || (!canSubmit && !showStop)}
                style={{
                    position: "relative",
                    color: "var(--user-ink)",
                } as CSSProperties}
                onClick={showStop ? props.onStop : interactionBusy ? undefined : props.onSubmit}
                aria-label={actionLabel}
                title={!canSubmit && !showStop && !interactionBusy ? create.submitNeedPrompt : actionLabel}
            >
                {showCost && !showStop ? <span className="creation-submit-cost" title={routeQuote ? modelQuoteDescription(routeQuote) : undefined}><CreditSymbol /><span>{routeQuote?.estimated ? `预估:${formattedCredits}` : formattedCredits}</span></span> : null}
                <span className="creation-submit-action" aria-hidden>{showStop ? <span>{create.stop}</span> : showWorkingSpinner ? <><LoaderCircle className="size-4 animate-spin" /><span>{create.generating}</span></> : <><ArrowUp className="size-4" /><span>{create.send}</span></>}</span>
            </Button>
        </footer>
        <CreationMediaPreviewModal url={previewUrl} type={previewType} onClose={() => setPreviewUrl("")} />
        </div>
    </div>;

    if (!promptOptimizerOpen) return composer;

    return (
        <Suspense fallback={composer}><CanvasPromptOptimizerDrawer
            open={promptOptimizerOpen}
            prompt={props.prompt}
            generationMode={props.mode === "video" ? "video" : "image"}
            targetModel={modelOptionName(props.model) || props.model}
            targetProtocol={priceChannel.modelCosts?.find((item) => item.model === modelOptionName(props.model))?.protocol || priceChannel.interfaceType}
            config={props.config}
            optimizerModel={props.config.textModel}
            references={optimizerReferences}
            provider={props.promptOptimizerProvider}
            onClose={() => setPromptOptimizerOpen(false)}
            onApply={props.setPrompt}
        >
            {composer}
        </CanvasPromptOptimizerDrawer></Suspense>
    );
}

export function CreationModeTabs({ mode, onModeChange, agentActive = false, onAgentSelect, orientation = "horizontal" }: { mode: CreationMode; onModeChange: (mode: CreationMode) => void; agentActive?: boolean; onAgentSelect?: () => void; orientation?: "horizontal" | "vertical" }) {
    const create = useCopy().video.create;
    const reducedMotion = useReducedMotion();
    const items: { mode: CreationMode; icon: ReactNode; label: string }[] = [
        { mode: "video", icon: <Film />, label: create.modeVideo },
        { mode: "image", icon: <ImageIcon />, label: create.modeImage },
        { mode: "text", icon: <MessageSquareText />, label: create.modeText },
    ];
    const indicator = (pressed: boolean) => pressed ? (
        <motion.span
            layoutId={`creation-mode-indicator-${orientation}`}
            className="creation-mode-indicator"
            aria-hidden
            transition={reducedMotion ? { duration: 0 } : aceternityMotion.spring.dock}
        />
    ) : null;
    return <LayoutGroup id={`creation-mode-tabs-${orientation}`}>
        <div className="creation-mode-tabs" role="group" aria-label={create.modesAria} data-active-mode={agentActive ? "agent" : mode} data-orientation={orientation} style={{ gridTemplateColumns: orientation === "vertical" ? "minmax(0, 1fr)" : `repeat(${onAgentSelect ? 4 : 3}, minmax(0, 1fr))` }}>
        {items.map((item) => (
            <button key={item.mode} type="button" className="creation-mode-button" data-mode={item.mode} aria-pressed={!agentActive && item.mode === mode} aria-label={item.label} onClick={() => onModeChange(item.mode)}>
                {indicator(!agentActive && item.mode === mode)}
                {item.icon}
                <span>{item.label}</span>
            </button>
        ))}
        {onAgentSelect ? <button type="button" className="creation-mode-button" data-mode="agent" aria-pressed={agentActive} onClick={onAgentSelect}>{indicator(agentActive)}<Brain /><span>{create.modeAgent}</span></button> : null}
        </div>
    </LayoutGroup>;
}

function ModePicker({ mode, onModeChange }: { mode: CreationMode; onModeChange: (mode: CreationMode) => void }) {
    return <CreationModeTabs mode={mode} onModeChange={onModeChange} />;
}

function GenerationSettingsMenu(props: ComposerProps) {
    const [open, setOpen] = useState(false);
    const activeQualityOptions = props.imageProfile.quality.values.map((value) => qualityOptions.find((item) => item.value === value) || { value, label: value.toUpperCase(), description: "模型支持的质量/分辨率" });
    const qualityLabel = activeQualityOptions.find((item) => item.value === props.quality)?.label || qualityOptions.find((item) => item.value === props.quality)?.label || props.quality || "自动";
    // 尺寸/比例/分辨率选项取同显示名分组内全部模型的并集，路由模型只决定发送参数。
    const mergedProfile = mergedImageCapabilityConfig(props.config, props.model || props.config.imageModel);
    const usesImageResolutionPicker = props.mode === "image" && supportsImageResolutionPresets(mergedProfile.size);
    const imageResolutionOptions = usesImageResolutionPicker ? buildImageResolutionOptions(mergedProfile.size.values) : [];
    const ratios = props.videoProfile.ratios;
    const referenceImageSize = props.mode === "image" && mergedProfile.size.allowCustom ? props.referenceImageSize : undefined;
    const referenceImageSizeValue = referenceImageSize ? String(referenceImageSize.width) + "x" + String(referenceImageSize.height) : "";
    const referenceImageSizeLabel = referenceImageSize ? String(referenceImageSize.width) + " × " + String(referenceImageSize.height) : "";
    const referenceImageSizeSelected = Boolean(referenceImageSizeValue && props.ratio === referenceImageSizeValue);
    const resolutions = props.mode === "video" ? props.videoProfile.resolutions.map((value) => ({ value: value.replace(/p$/i, ""), label: videoResolutionLabel(value) })) : resolutionOptions;
    const selectReferenceImageSize = () => {
        if (!referenceImageSizeValue) return;
        props.setRatio(referenceImageSizeValue);
    };
    const videoResolutionSupported = props.mode === "video" && resolutions.length > 0;
    const imageSummary = [
        ...(mergedProfile.size.parameter !== "none" ? [referenceImageSizeSelected ? referenceImageSizeLabel : usesImageResolutionPicker ? formatImageResolutionSize(props.ratio, imageResolutionOptions) : props.ratio] : []),
        ...(props.imageProfile.quality.supported ? [qualityLabel] : []),
        ...(props.imageProfile.maxOutputs > 1 ? [props.count] : []),
    ].join(" · ");
    const videoRatioSupported = props.mode === "video" && ratios.length > 0;
    const summary = props.mode === "video" ? [...(videoRatioSupported ? [props.ratio] : []), ...(videoResolutionSupported ? [videoResolutionLabel(props.videoQuality)] : [])].join(" · ") : imageSummary;
    const panel = <div className="creation-parameter-menu">
        {props.mode === "image" ? <ImageSizePicker profile={mergedProfile} size={props.ratio} quality={props.quality} onChange={(size, quality) => { props.setRatio(size); if (quality) props.setQuality(quality); }} /> : videoRatioSupported ? <SettingSection title="画幅" value={props.ratio}><div className="creation-choice-grid is-ratio">{ratios.map((value) => <button key={value} type="button" aria-pressed={value === props.ratio} className={value === props.ratio ? "is-selected" : ""} onClick={() => props.setRatio(value)}><span className="creation-ratio-preview"><span style={ratioPreviewStyle(value)} /></span><span>{value}</span></button>)}</div></SettingSection> : null}
        {props.mode === "image" && referenceImageSizeValue ? <button type="button" className="creation-custom-trigger" onClick={selectReferenceImageSize}>使用参考图尺寸 · {referenceImageSizeLabel}</button> : null}
        {props.mode === "video" ? (videoResolutionSupported ? <SettingSection title="清晰度" value={videoResolutionLabel(props.videoQuality)}><div className="creation-choice-grid is-resolution">{resolutions.map((option) => <button key={option.value} type="button" aria-pressed={option.value === props.videoQuality} className={option.value === props.videoQuality ? "is-selected" : ""} onClick={() => props.setVideoQuality(option.value)}>{option.label}</button>)}</div></SettingSection> : null) : <>

            {props.imageProfile.quality.supported && !imageResolutionUsesQuality(mergedProfile) ? <SettingSection title={activeQualityOptions.some((item) => item.value === "1k" || item.value === "2k") ? "分辨率" : "图片质量"} value={qualityLabel}><div className="creation-choice-grid is-quality">{activeQualityOptions.map((option) => <button key={option.value} type="button" aria-pressed={option.value === props.quality} className={option.value === props.quality ? "is-selected" : ""} onClick={() => props.setQuality(option.value)}><span>{option.label}</span><small>{option.description}</small></button>)}</div></SettingSection> : null}
            {props.imageProfile.maxOutputs > 1 ? <SettingSection title="生成数量" value={`${props.count} 张`}><div className="creation-parameter-content"><div className="creation-choice-grid is-count">{countOptions.filter((option) => Number(option) <= props.imageProfile.maxOutputs).map((option) => <button key={option} type="button" aria-pressed={option === props.count} className={option === props.count ? "is-selected" : ""} onClick={() => props.setCount(option)}>{option}</button>)}</div><label className="creation-custom-value"><span>自定义</span><input inputMode="numeric" pattern="[0-9]*" value={props.count} onChange={(event) => props.setCount(String(Math.max(1, Math.min(props.imageProfile.maxOutputs, Number(event.target.value) || 1))))} aria-label={`生成数量，范围 1 到 ${props.imageProfile.maxOutputs}`} /><em>张</em></label></div></SettingSection> : null}
        </>}
    </div>;
    return <Popover open={open} onOpenChange={setOpen} trigger="click" placement="bottom" arrow={false} classNames={{ root: "creation-control-popover", container: "creation-control-popover-surface", content: "creation-control-popover-content" }} content={panel}>
        <button type="button" className="creation-chat-control" aria-label={`生成设置：${summary}`}><SlidersHorizontal /><span>{summary}</span><ChevronDown className={open ? "is-open" : ""} /></button>
    </Popover>;
}

function SettingSection({ title, value, children }: { title: string; value?: string; children: ReactNode }) {
    return <section className="creation-parameter-section"><header><h3>{title}</h3>{value ? <span>{value}</span> : null}</header>{children}</section>;
}

function DurationMenu({ profile, seconds, onChange }: { profile: VideoCapabilityConfig; seconds: string; onChange: (value: string) => void }) {
    const [open, setOpen] = useState(false);
    const value = Number(normalizeVideoValue(profile, { seconds }).seconds);
    const presets = profile.duration.selection === "enum" ? videoDurationOptions(profile) : [];
    const fallbackPreset = presets.length ? presets : [profile.duration.default];
    const min = profile.duration.selection === "range" ? profile.duration.min || 1 : Math.min(...fallbackPreset);
    const max = profile.duration.selection === "range" ? Math.max(min, profile.duration.max || min) : Math.max(...fallbackPreset);
    const step = Math.max(1, profile.duration.step || 1);
    const durationControl = profile.duration.selection === "range" ? <>
        <input className="h-8 w-full" style={{ accentColor: "var(--creation-text)" }} type="range" min={min} max={max} step={step} value={value} aria-label="视频时长（秒）" onChange={(event) => onChange(event.target.value)} />
        <div className="flex justify-between px-0.5 text-[var(--fs-tiny)] text-[var(--creation-muted)]"><span>{min}s</span><span>{max}s</span></div>
        <label className="creation-custom-value is-duration"><span>自定义时长</span><span className="creation-duration-custom-field"><input type="number" min={min} max={max} step={step} inputMode="numeric" value={seconds} onFocus={(event) => event.currentTarget.select()} onBlur={() => onChange(String(value))} onChange={(event) => onChange(event.target.value)} aria-label="自定义视频时长，单位秒" /><em>秒</em></span></label>
    </> : <div className="creation-duration-choices">{presets.map((item) => <button key={item} type="button" className={item === value ? "is-selected" : ""} onClick={() => onChange(String(item))}>{item}s</button>)}</div>;
    return <Popover open={open} onOpenChange={setOpen} trigger="click" placement="bottom" arrow={false} classNames={{ root: "creation-control-popover", container: "creation-control-popover-surface", content: "creation-control-popover-content" }} content={<div className="creation-duration-menu"><div className="creation-duration-heading"><span>时长</span><strong>{value} 秒</strong></div>{durationControl}</div>}>
        <button type="button" className="creation-chat-control is-duration" aria-label={`视频时长：${value}秒`}><Clock3 /><span>{value}s</span><ChevronDown className={open ? "is-open" : ""} /></button>
    </Popover>;
}

const creationEmptyBannerFrames = [
    { src: "/short-drama-styles/cyberpunk-neon.jpg", caption: "镜头01 · 雨港记忆诊所" },
    { src: "/short-drama-styles/suspense-noir.jpg", caption: "镜头02 · 巷口证人" },
    { src: "/short-drama-styles/retro-hong-kong.jpg", caption: "镜头03 · 天台晾衣绳" },
];

export function CreationEmptyBanner() {
    const brandName = useAppearanceStore((state) => state.appearance.brandName);
    return <div className="creation-empty-art" aria-hidden="true">
        {creationEmptyBannerFrames.map((frame, index) => <figure key={frame.caption} className={`creation-empty-art-frame ${index === 1 ? "is-main" : index === 0 ? "is-back" : "is-front"}`}>
            <img src={styleCoverSrc(frame.src)} alt="" />
            <span>{frame.caption}</span>
        </figure>)}
        <span className="creation-empty-art-caption"><span>{brandName}</span>把每一帧，交给镜头导演</span>
    </div>;
}

export function CreationEmptySuggest({ onStartPrompt, onOpenLibrary }: { onStartPrompt: (mode: CreationMode, prompt: string) => void; onOpenLibrary: () => void }) {
    const create = useCopy().video.create;
    const reducedMotion = useReducedMotion();
    const [hovered, setHovered] = useState<string | null>(null);
    const suggestions = [
        { mode: "video" as const, icon: Clapperboard, title: create.suggestShot, hint: create.suggestShotHint, prompt: "雨夜十字路口，镜头从湿沥青上的霓虹倒影缓缓推到撑伞年轻人的侧脸，电影感布光，无字幕" },
        { mode: "image" as const, icon: ImageIcon, title: create.suggestRef, hint: create.suggestRefHint, prompt: "", openLibrary: true },
        { mode: "text" as const, icon: FileText, title: create.suggestScript, hint: create.suggestScriptHint, prompt: "帮我续写一个短剧故事，先聊聊剧情走向：" },
    ];
    return <LayoutGroup id="creation-empty-suggest">
        <div className="creation-empty-suggest" aria-label={create.hint}>
        {suggestions.map((item) => {
            const Icon = item.icon;
            const start = () => {
                if ("openLibrary" in item && item.openLibrary) onOpenLibrary();
                else onStartPrompt(item.mode, item.prompt);
            };
            return <motion.button
                key={item.title}
                type="button"
                className="suggest-card"
                onClick={start}
                onHoverStart={() => setHovered(item.title)}
                onHoverEnd={() => setHovered(null)}
                whileHover={reducedMotion ? undefined : { y: -2 }}
                transition={aceternityMotion.spring.surface}
            >
                {hovered === item.title ? (
                    <motion.span
                        layoutId="creation-suggest-hover"
                        className="suggest-card-hover"
                        aria-hidden
                        transition={reducedMotion ? { duration: 0 } : aceternityMotion.spring.surface}
                    />
                ) : null}
                <span className="library-icon-tile suggest-icon"><Icon size={18} strokeWidth={2} /></span>
                <span className="suggest-copy">
                    <strong>{item.title}</strong>
                    <span>{item.hint}</span>
                </span>
            </motion.button>;
        })}
        </div>
    </LayoutGroup>;
}


export function CreationFeaturedWorks({ onStartPrompt }: { onStartPrompt: (mode: CreationMode, prompt: string) => void }) {
    const [filter, setFilter] = useState<"all" | CreationMode>("all");
    const [limit, setLimit] = useState(12);
    const filtered = creationFeaturedWorks.filter((item) => filter === "all" || item.mode === filter);
    return <section className="creation-featured-works" aria-labelledby="creation-featured-title">
        <div className="creation-featured-heading">
            <div><h2 id="creation-featured-title">精选灵感</h2></div>
            <p>{creationFeaturedWorks.length} 个创意起点 · 点击填入提示词，不自动生成</p>
        </div>
        <div className="creation-inspiration-filters" role="group" aria-label="灵感类型">
            {(["all", "video", "image", "text"] as const).map((value) => <button key={value} type="button" aria-pressed={filter === value} onClick={() => { setFilter(value); setLimit(12); }}>{value === "all" ? "全部灵感" : modeLabels[value]}<span>{creationFeaturedWorks.filter((item) => value === "all" || item.mode === value).length}</span></button>)}
        </div>
        <div className="creation-featured-layout">
                {filtered.slice(0, limit).map((item, index) => <button key={item.title} type="button" className={`product-collection-card creation-featured-card ${index === 0 ? "is-featured-hero" : ""}`} onClick={() => onStartPrompt(item.mode, item.prompt)}>
                    <span className="creation-featured-media"><img src={styleCoverSrc(item.image)} alt="" loading="lazy" /><span className="creation-inspiration-overlay"><ArrowUp />使用这个创意</span></span>
                    <span className="creation-featured-copy"><strong>{item.title}</strong><span>{item.description}</span><em>Yoyo 工作室案例 · {modeLabels[item.mode]}</em></span>
                </button>)}
        </div>
        <footer className="creation-inspiration-footer">{limit < filtered.length ? <Button onClick={() => setLimit((count) => count + 12)}>展开更多灵感<ChevronDown /></Button> : <span>已展示全部 {filtered.length} 个创意</span>}<details><summary>封面与案例说明</summary><p>{inspirationSource.notice}</p></details></footer>
    </section>;
}

type CreationThinking = { title: string; hint: string; steps: string[]; activity: string };

function thinkingFor(mode: CreationMode, brandName: string): CreationThinking {
    if (mode === "image") return { title: "正在为你画这一镜", hint: `${brandName}正在理解你的构图意图，并把画面交给模型出图。`, steps: ["理解构图", "定调画风", "生成画面"], activity: "正在理解构图并把画面交给模型出图" };
    if (mode === "text") return { title: "正在为你写这段", hint: `${brandName}正在梳理你的创作脉络，组织语言与结构。`, steps: ["梳理脉络", "组织语言", "输出段落"], activity: "正在梳理脉络并组织语言" };
    return { title: "正在为你拍这一镜", hint: `${brandName}正在拆解你的镜头脚本，设计运镜与光线，并交给模型渲染成片。`, steps: ["拆解镜头", "设计运镜", "定调布光", "渲染成片"], activity: "正在按导演思路拆解镜头并渲染" };
}
function formatMessageTime(value: string) {
    const timestamp = new Date(value).getTime();
    return Number.isFinite(timestamp) ? conversationTimeFormatter.format(timestamp) : "";
}

function buildConversationExportMarkdown(conversation: CreationConversation, assistantName: string, userName: string, untitled: string) {
    const lines: string[] = [`# ${conversation.title.trim() || untitled}`, ""];
    for (const message of conversation.messages) {
        const stamp = formatMessageTime(message.createdAt);
        const modeTag = message.mode && message.mode !== "text" ? (message.mode === "image" ? "[图像生成] " : "[视频生成] ") : "";
        const speaker = message.role === "user" ? (userName || "我") : assistantName;
        if (message.role === "user") {
            const prompt = displayCreationPrompt(message.content, message.references || []).trim();
            if (!prompt) continue;
            lines.push(`## ${speaker} · ${stamp}`, "", prompt, "");
        } else {
            const body = (message.content || "").trim();
            if (body) lines.push(`## ${speaker} · ${stamp}`, "", `${modeTag}${body}`, "");
            else if (message.resultUrls?.length) lines.push(`## ${speaker} · ${stamp}`, "", `${modeTag}已生成，素材保留在项目中。`, "");
            else continue;
            if (message.reasoning?.trim()) lines.push("> 思考过程：", message.reasoning.trim(), "");
        }
    }
    return lines.join("\n").trim() + "\n";
}
function downloadCreationConversation(conversation: CreationConversation, assistantName: string, userName: string, untitled: string) {
    const safeTitle = (conversation.title.trim() || untitled).replace(/[\\/:*?"<>|]/g, "_").slice(0, 60) || untitled;
    const blob = new Blob([buildConversationExportMarkdown(conversation, assistantName, userName, untitled)], { type: "text/markdown;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `${safeTitle}.md`;
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    URL.revokeObjectURL(url);
}
function conversationPreviewMessage(conversation: CreationConversation) {
    let fallback: CreationMessage | undefined;
    for (let index = conversation.messages.length - 1; index >= 0; index -= 1) {
        const message = conversation.messages[index];
        if (!message.content.trim()) continue;
        fallback ||= message;
        if (message.role === "user") return message;
    }
    return fallback;
}

function formatHistoryRelativeTime(value: string, create: CreationCopy) {
    const timestamp = conversationTimestamp(value);
    if (!timestamp) return "";
    const seconds = Math.max(0, Math.round((Date.now() - timestamp) / 1000));
    if (seconds < 60) return create.justNow;
    const minutes = Math.round(seconds / 60);
    if (minutes < 60) return create.minutesAgo.replace("{n}", String(minutes));
    const hours = Math.round(minutes / 60);
    if (hours < 24) return create.hoursAgo.replace("{n}", String(hours));
    const days = Math.round(hours / 24);
    if (days < 7) return create.daysAgo.replace("{n}", String(days));
    return historyDayFormatter.format(timestamp);
}

function formatConversationTime(value: string, create: CreationCopy) {
    const timestamp = conversationTimestamp(value);
    if (!timestamp) return create.timeUnknown;
    return conversationTimeFormatter.format(timestamp);
}

function ratioPreviewStyle(value: string) {
    const [width, height] = value.replace("x", ":").split(":").map(Number);
    if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) return { width: 10, height: 10 };
    // 画幅容器的可用空间是 14×10；同时计算宽高，避免 CSS 的 max-width/max-height 把宽银幕比例压扁。
    const scale = Math.min(14 / width, 10 / height);
    return { width: Math.max(4, Math.round(width * scale)), height: Math.max(4, Math.round(height * scale)) };
}
