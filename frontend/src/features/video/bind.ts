import * as api from "../../lib/client";
import { looksLikeSeries } from "./drama-lib";
import { defaultDramaStyle, useDramaSelection } from "./workshop-store";

/** Bind this video conversation to the workshop episode, if one is selected.
 *  Does not create a series — the project (drama/episode) is independent of
 *  the chat, same as LibTV session vs projectUuid. */
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

export function guessSeriesTitle(text: string) {
  const line = (text.split(/\n/)[0] || "").trim();
  if (!line || Array.from(line).length > 40) return "";
  if (/^第.+[章节回]|Chapter\s+\d+/i.test(line)) return "";
  return line;
}

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

export { looksLikeSeries };
