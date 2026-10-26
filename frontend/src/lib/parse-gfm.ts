/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 */

/**
 * 拥有：冻结尾部的流式解析和定稿 Streamdown 共用的 GFM 语法。
 * 不拥有：增量切块，以及把 mdast 画成界面。
 */

import { fromMarkdown } from "mdast-util-from-markdown";
import { gfmFromMarkdown } from "mdast-util-gfm";
import { gfm } from "micromark-extension-gfm";
import type { Root } from "mdast";

/** parseGfm 用 micromark 的 GFM 扩展把文本解析成 mdast 根。 */
export function parseGfm(text: string): Root {
  return fromMarkdown(text, {
    extensions: [gfm()],
    mdastExtensions: [gfmFromMarkdown()],
  });
}
