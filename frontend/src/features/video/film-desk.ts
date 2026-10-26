/**
 * 本文件拥有成片台当前工程在浏览器里的指针。不拥有台词、镜头和成片文件。
 * 它只改本地状态，不改服务端。
 */
import { create } from "zustand";

/** FilmWord 是一句台词上的字，带可选的行、说话人和起止。 */
export type FilmWord = { id?: string; line_id?: string; text?: string; speaker?: string; start?: number; end?: number };
/** FilmNeed 是成片还缺的一项，带种类、提示、状态和错误。 */
export type FilmNeed = { id?: string; key?: string; kind?: string; prompt?: string; status?: string; cost?: string; error?: string; asset_hash?: string };
/** FilmTask 是挂在某个需求上的任务状态。 */
export type FilmTask = { id?: string; need_id?: string; status?: string; error?: string };
/** FilmArtifact 是成片台记下的一个产物名字、哈希和媒体类型。 */
export type FilmArtifact = { name?: string; hash?: string; media_type?: string };
/** FilmShot 是一个镜头的画面说明和起止。 */
export type FilmShot = { id?: string; visual?: string; start?: number; end?: number; beat?: boolean };
/** FilmProject 是成片台读到的工程形状，含简介、台词、镜头、需求和产物。 */
export type FilmProject = {
  id?: string;
  parent_id?: string;
  title?: string;
  structure?: string;
  note?: string;
  bed?: string;
  brief?: { host?: string; product?: string; language?: string; aspect?: string; hook?: string; price?: string; cta?: string };
  entries?: { id?: string; label?: string }[];
  reference?: { hash?: string; duration?: number };
  words?: FilmWord[];
  shots?: FilmShot[];
  stickers?: { id?: string; text?: string }[];
  needs?: FilmNeed[];
  tasks?: FilmTask[];
  artifacts?: FilmArtifact[];
};

type FilmDeskState = {
  project: FilmProject | null;
  setProject: (project: FilmProject | null) => void;
  clear: () => void;
};

/** useFilmDesk 保存当前成片工程。没有 id 的对象会被记成空。 */
export const useFilmDesk = create<FilmDeskState>((set) => ({
  project: null,
  /** 写入工程；没有 id 时记成空。 */
  setProject: (project) => set({ project: project?.id ? project : null }),
  /** 丢掉本地工程指针。 */
  clear: () => set({ project: null }),
}));
