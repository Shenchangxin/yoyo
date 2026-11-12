/**
 * 拥有：文库和阅读的类型和调用。字段名和 Go 侧的 JSON 一致。
 * 不拥有：索引、检索和批注定位的规则，那些在引擎里。也不持有任何状态。
 */

import { learnCall } from "./client";

/** IndexProgress 是正在建索引的进度。 */
export type IndexProgress = { phase: "parsing" | "embedding" | "publishing"; done: number; total: number; file: string };

/** Library 是一个可检索的库。progress 只在建索引时有。 */
export type Library = {
  id: string;
  name: string;
  kind: "indexed" | "linked" | "notes";
  description: string;
  linked_path: string;
  status: "ready" | "indexing" | "needs_reindex" | "error";
  documents: number;
  embed_model: string;
  published_gen: number;
  last_error: string;
  progress?: IndexProgress;
  updated_at: string;
};

/** Document 是库里的一份原文。unit 是 page 或 section，page_count 是单元数。 */
export type LibDocument = {
  id: string;
  library_id: string;
  rel_path: string;
  title: string;
  kind: "pdf" | "markdown" | "text" | "html" | "docx" | "note" | "";
  bytes: number;
  page_count: number;
  unit: "page" | "section";
  status: "pending" | "parsed" | "failed" | "needs_ocr";
  error: string;
  source_url: string;
  char_count: number;
  updated_at: string;
};

/** Hit 是一次检索命中。page 是定位号，heading 是它所在的标题路径。 */
export type Hit = {
  library_id: string;
  document_id: string;
  title: string;
  rel_path: string;
  page: number;
  heading: string;
  chunk_id: string;
  source_hash: string;
  excerpt: string;
  text: string;
  score: number;
};

/** UnitView 是读出的一个文本单元。status 为 changed 时原文在命中之后变了，没有正文。 */
export type UnitView = { text: string; status: "ok" | "needs_ocr" | "changed"; page: number; pages: number; unit: "page" | "section"; kind: string; source_hash: string };

/** Material 是阅读台上的一份材料。 */
export type Material = {
  id: string;
  library_id: string;
  document_id: string;
  title: string;
  kind: string;
  unit: "page" | "section";
  page_count: number;
  last_opened_at: string;
  locator: number;
  percent: number;
  annotations: number;
};

/** AnnotColor 是批注的五种颜色。 */
export type AnnotColor = "yellow" | "green" | "blue" | "pink" | "purple";

/** ANNOT_COLORS 是颜色按钮从左到右的顺序。 */
export const ANNOT_COLORS: AnnotColor[] = ["yellow", "green", "blue", "pink", "purple"];

/** Annotation 是材料上的一条批注。start 和 end 是引文在单元文字里的位置，找不到时是 -1。 */
export type Annotation = {
  id: string;
  material_id: string;
  kind: "highlight" | "underline" | "note";
  color: AnnotColor;
  locator: number;
  quote: string;
  note: string;
  start: number;
  end: number;
  author: "user" | "assistant";
  resolution: "resolved" | "unresolved";
  created_at: string;
  updated_at: string;
};

/** Bookmark 是材料里的一个书签。 */
export type Bookmark = { id: string; material_id: string; locator: number; label: string; created_at: string };

/** OutlineEntry 是提纲里的一条。 */
export type OutlineEntry = { locator: number; title: string; level: number; offset: number };

/** MaterialView 是打开一份材料时界面要的全部东西。 */
export type MaterialView = {
  material: Material;
  document: LibDocument;
  outline: OutlineEntry[];
  position: { locator: number; percent: number };
  bookmarks: Bookmark[];
  annotations: Annotation[];
};

/** ImportOutcome 是一次导入的结果。 */
export type ImportOutcome = { imported: number; skipped: { path: string; reason: string }[] };

/** list 把可能是 null 的数组收成数组。 */
function list<T>(v: T[] | null | undefined): T[] {
  return Array.isArray(v) ? v : [];
}

/** libraries 列出全部库。 */
export async function libraries(): Promise<Library[]> {
  return list(await learnCall("learn.libraries"));
}

/** createLibrary 新建一个库。kind 是 indexed 或 linked，linked 要给文件夹路径。 */
export async function createLibrary(name: string, kind: "indexed" | "linked" = "indexed", path = ""): Promise<Library> {
  return learnCall("learn.library.create", { name, kind, path });
}

/** renameLibrary 改库的名字。 */
export async function renameLibrary(id: string, name: string): Promise<void> {
  await learnCall("learn.library.update", { id, name });
}

/** deleteLibrary 删除库，连同它的索引、阅读材料和批注。 */
export async function deleteLibrary(id: string): Promise<void> {
  await learnCall("learn.library.delete", { id });
}

/** setEmbedModel 设置库的嵌入模型并重新建索引。空串表示只做关键词检索。 */
export async function setEmbedModel(id: string, model: string): Promise<void> {
  await learnCall("learn.library.embed", { id, model });
}

/** documents 列出一个库里的文档。 */
export async function documents(libraryId: string): Promise<LibDocument[]> {
  return list(await learnCall("learn.documents", { library_id: libraryId }));
}

/** importPaths 把一组文件或文件夹导入库。索引在后台自己建。 */
export async function importPaths(libraryId: string, paths: string[]): Promise<ImportOutcome> {
  const res = await learnCall("learn.import", { library_id: libraryId, paths });
  return { imported: Number(res?.imported || 0), skipped: list(res?.skipped) };
}

/** importURL 把一个网页的正文存进库。 */
export async function importURL(libraryId: string, url: string): Promise<LibDocument> {
  return learnCall("learn.import.url", { library_id: libraryId, url });
}

/** rebuild 让一个库重新建索引。 */
export async function rebuild(libraryId: string): Promise<void> {
  await learnCall("learn.rebuild", { library_id: libraryId });
}

/** removeDocument 把一份文档从库里拿掉。 */
export async function removeDocument(libraryId: string, documentId: string): Promise<void> {
  await learnCall("learn.document.remove", { library_id: libraryId, document_id: documentId });
}

/** renameDocument 改一份文档显示的标题。 */
export async function renameDocument(libraryId: string, documentId: string, title: string): Promise<void> {
  await learnCall("learn.document.rename", { library_id: libraryId, document_id: documentId, title });
}

/** search 在一个库里检索。 */
export async function search(libraryId: string, query: string, limit = 0): Promise<{ hits: Hit[]; message: string }> {
  const res = await learnCall("learn.search", { library_id: libraryId, query, limit });
  return { hits: list(res?.hits), message: String(res?.message || "") };
}

/** readUnit 读一份文档的一个单元。 */
export async function readUnit(libraryId: string, documentId: string, page: number, sourceHash = ""): Promise<UnitView> {
  return learnCall("learn.read", { library_id: libraryId, document_id: documentId, page, source_hash: sourceHash });
}

/** normalizeView 给材料视图里可能缺的数组补上空数组。 */
function normalizeView(v: MaterialView): MaterialView {
  return { ...v, outline: list(v?.outline), bookmarks: list(v?.bookmarks), annotations: list(v?.annotations) };
}

/** materials 列出阅读台上的材料，最近打开的在前。 */
export async function materials(): Promise<Material[]> {
  return list(await learnCall("learn.materials"));
}

/** openMaterial 把一份文档放到阅读台，返回提纲、位置、书签和批注。 */
export async function openMaterial(libraryId: string, documentId: string): Promise<MaterialView> {
  return normalizeView(await learnCall("learn.material.open", { library_id: libraryId, document_id: documentId }));
}

/** savePosition 记下读到的位置。 */
export async function savePosition(materialId: string, locator: number, percent: number): Promise<void> {
  await learnCall("learn.material.position", { material_id: materialId, locator, percent });
}

/** annotations 列出一份材料上的批注。 */
export async function annotations(materialId: string): Promise<Annotation[]> {
  return list(await learnCall("learn.annotations", { material_id: materialId }));
}

/** annotate 记一条批注。quote 是选中的原文，prefix 和 suffix 是它前后的一小段，引擎用它们在原文里定位。 */
export async function annotate(req: {
  materialId: string;
  locator: number;
  quote: string;
  note?: string;
  color?: AnnotColor;
  kind?: Annotation["kind"];
  prefix?: string;
  suffix?: string;
}): Promise<Annotation> {
  return learnCall("learn.annotate", {
    material_id: req.materialId, locator: req.locator, quote: req.quote, note: req.note || "", color: req.color || "yellow",
    kind: req.kind || "", prefix: req.prefix || "", suffix: req.suffix || "",
  });
}

/** updateAnnotation 改一条批注的笔记或颜色。只传要改的项。 */
export async function updateAnnotation(id: string, patch: { note?: string; color?: AnnotColor }): Promise<Annotation> {
  return learnCall("learn.annotation.update", { id, ...patch });
}

/** deleteAnnotation 删除一条批注。 */
export async function deleteAnnotation(id: string): Promise<void> {
  await learnCall("learn.annotation.delete", { id });
}

/** toggleBookmark 在一个定位号上放书签，已经有时拿掉。返回现在的全部书签。 */
export async function toggleBookmark(materialId: string, locator: number): Promise<Bookmark[]> {
  const res = await learnCall("learn.bookmark.toggle", { material_id: materialId, locator });
  return list(res?.bookmarks);
}

/** locate 在一份材料自己的文字里查找。 */
export async function locate(materialId: string, query: string): Promise<{ hits: Hit[]; message: string }> {
  const res = await learnCall("learn.locate", { material_id: materialId, query });
  return { hits: list(res?.hits), message: String(res?.message || "") };
}

/** exportMaterial 把一份材料上的书签和批注导成 Markdown。 */
export async function exportMaterial(materialId: string): Promise<{ filename: string; markdown: string }> {
  return learnCall("learn.material.export", { material_id: materialId });
}

/** setViewport 告诉引擎学习者此刻看的位置和选中的文字，导师的回合简报用它。失败不要紧。 */
export async function setViewport(session: string, materialId: string, locator: number, selection = ""): Promise<void> {
  if (!session) return;
  await learnCall("learn.viewport", { session, material_id: materialId, locator, selection }).catch(() => undefined);
}
