/**
 * 本文件拥有把视频会话绑到已选剧集，以及在需要时新建系列、写入原文和提出分集。
 * 不拥有短剧看板。绑定、新建、写入和提出分集都会改服务端状态。
 */
import * as api from "../../lib/client";
import { looksLikeSeries } from "./drama-lib";
import { defaultDramaStyle, useDramaSelection } from "./workshop-store";

/** bindVideoThread 把这次视频会话绑到已选剧集；没选集时只绑系列。这里不新建系列。项目和聊天分开，和 LibTV 的 session 与 projectUuid 一样。 */
export async function bindVideoThread(sessionId: string): Promise<void> {
  const { episodeId, dramaId } = useDramaSelection.getState();
  if (!sessionId) return;
  if (episodeId) {
    await api.video.bind(sessionId, episodeId, dramaId);
    return;
  }
  if (dramaId) await api.video.bind(sessionId, "", dramaId);
}

let dramaEnsure: Promise<string> | null = null;
let episodeEnsure: Promise<string> | null = null;

/** ensureDrama 返回已选系列；没有就建一条，比例缺省是 9:16。并发调用共用同一次创建。 */
export async function ensureDrama(opts: { title?: string; ratio?: string } = {}) {
  const sel = useDramaSelection.getState();
  if (sel.dramaId) return sel.dramaId;
  if (dramaEnsure) return dramaEnsure;
  dramaEnsure = (async () => {
    const latest = useDramaSelection.getState();
    if (latest.dramaId) return latest.dramaId;
    const d = await api.video.createDrama({ title: opts.title || "", style: defaultDramaStyle(), aspect_ratio: opts.ratio || "9:16" });
    latest.setDramaId(d.id);
    latest.setEpisodeId("");
    return d.id as string;
  })();
  try {
    return await dramaEnsure;
  } finally {
    dramaEnsure = null;
  }
}

/** ensureDramaEpisode 返回已选集；没有正文的已选集会补上正文。没有系列时先建系列再建集。 */
export async function ensureDramaEpisode(opts: { title: string; episodeTitle: string; content?: string; ratio?: string }) {
  const sel = useDramaSelection.getState();
  if (sel.episodeId) {
    if (opts.content) {
      const ep = await api.video.getEpisode(sel.episodeId).catch(() => null) as { content?: string } | null;
      if (ep && !(ep.content || "").trim()) {
        await api.video.updateEpisode({ id: sel.episodeId, content: opts.content }).catch(() => {});
      }
    }
    return sel.episodeId;
  }
  if (episodeEnsure) return episodeEnsure;
  episodeEnsure = (async () => {
    const latest = useDramaSelection.getState();
    if (latest.episodeId) return latest.episodeId;
    if (latest.dramaId) {
      const ep = await api.video.createEpisode(latest.dramaId, opts.episodeTitle, opts.content || "");
      latest.setEpisodeId(ep.id);
      return ep.id as string;
    }
    const d = await api.video.createDrama({ title: opts.title, style: defaultDramaStyle(), aspect_ratio: opts.ratio || "9:16" });
    latest.setDramaId(d.id);
    const ep = await api.video.createEpisode(d.id, opts.episodeTitle, opts.content || "");
    latest.setEpisodeId(ep.id);
    return ep.id as string;
  })();
  try {
    return await episodeEnsure;
  } finally {
    episodeEnsure = null;
  }
}

/** guessSeriesTitle 取第一行当系列名。空行、超过 40 个字符，或像章节标题的行，返回空。 */
export function guessSeriesTitle(text: string) {
  const line = (text.split(/\n/)[0] || "").trim();
  if (!line || Array.from(line).length > 40) return "";
  if (/^第.+[章节回]|Chapter\s+\d+/i.test(line)) return "";
  return line;
}

/** ingestSeriesText 在当前系列已提交或已有非草稿集时另建系列，然后写入原文并提出分集。 */
export async function ingestSeriesText(text: string, title = "") {
  const sel = useDramaSelection.getState();
  const plan = sel.dramaId ? await api.video.getPlan(sel.dramaId).catch(() => null) as { status?: string } | null : null;
  const eps = sel.dramaId ? await api.video.episodes(sel.dramaId).catch(() => []) as unknown[] : [];
  let dramaId = sel.dramaId;
  const occupied = Array.isArray(eps) && eps.length > 0 && plan?.status !== "draft";
  if (!dramaId || plan?.status === "committed" || occupied) {
    const d = await api.video.createDrama({ title: title || guessSeriesTitle(text), style: defaultDramaStyle(), aspect_ratio: "9:16" });
    sel.setDramaId(d.id);
    sel.setEpisodeId("");
    dramaId = d.id;
  }
  await api.video.ingestSource(dramaId, text, title || guessSeriesTitle(text));
  await api.video.proposeEpisodes(dramaId);
  return dramaId as string;
}

/** ingestSeriesFile 在当前系列已提交或已有非草稿集时另建系列，然后从文件写入并提出分集。 */
export async function ingestSeriesFile(path: string) {
  const sel = useDramaSelection.getState();
  const plan = sel.dramaId ? await api.video.getPlan(sel.dramaId).catch(() => null) as { status?: string } | null : null;
  const eps = sel.dramaId ? await api.video.episodes(sel.dramaId).catch(() => []) as unknown[] : [];
  let dramaId = sel.dramaId;
  const occupied = Array.isArray(eps) && eps.length > 0 && plan?.status !== "draft";
  if (!dramaId || plan?.status === "committed" || occupied) {
    const d = await api.video.createDrama({ title: "", style: defaultDramaStyle(), aspect_ratio: "9:16" });
    sel.setDramaId(d.id);
    sel.setEpisodeId("");
    dramaId = d.id;
  }
  await api.video.ingestSourceFile(dramaId, path);
  await api.video.proposeEpisodes(dramaId);
  return dramaId as string;
}

/** looksLikeSeries 原样转出。 */
export { looksLikeSeries };
