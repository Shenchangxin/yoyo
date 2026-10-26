/**
 * Copyright (c) 2026 DeepSeek
 * SPDX-License-Identifier: MIT
 */

/**
 * 拥有：对只追加的 markdown 做块级增量解析，给出冻结前缀和不稳定尾部。来自 deepseek-harness 的 IncrementalMarkdownParser，并增加 tailStart，便于把 Streamdown 冻在源码前缀上。
 * 不拥有：调用方的语法，以及块的渲染。
 * 整篇重解析随回复变长呈二次。CommonMark 按行解析，追加只会改写前沿，即最后一个顶层块，更早的块已定稿。因此除末尾 UNSTABLE_TAIL_BLOCKS 个块外都冻结，只重解析它们后面的源码尾。最后一块未闭合的顶层围栏不能当块冻结，其已完成的内容行用第二道前沿：只有最后一行完整行和当前半行回到调用方语法。每个源码区域在整段流里被解析的次数有上界，而不是每个块一次。
 * 冻结边界取解析器自己的 position。切口在最后一块已冻结块的结束偏移，不在下一块的起始：下一块的起始会丢掉最多三个不重要的前导空格，丢掉无害，但切在上一块结束还能把块间空行留在尾部，切出的源码保持原文。围栏扫描只认解析器确认过的 code 节点和闭合定界符；含糊的输入退回普通尾部解析。
 * 与任何前缀冻结一样：micromark 在解析时按整篇解析引用式链接和脚注，定义落在冻结边界另一侧的引用会按字面显示，直到定稿的全文解析自己修好。
 */

import type { Code, Root, RootContent } from 'mdast'

/** 尾部保持不稳定的块数。追加最多改写最后一块，倒数第二块留作余量，冻结不必推理解析前沿。 */
const UNSTABLE_TAIL_BLOCKS = 2

/** PositionedBlock 一个顶层 mdast 块，加上跨块稳定的渲染键。 */
export interface PositionedBlock {
  /** 解析出的块。块内位置相对于它那一片源码。 */
  readonly node: RootContent
  /** 块在全文里的起始偏移。从首次出现到冻结都不变，跨过冻结边界时 React 复用而不是重挂。 */
  readonly key: number
}

/** IncrementalBlocks 一次 update 的结果。 */
export interface IncrementalBlocks {
  /** 不会再变的块。同一代里只增不减。 */
  readonly frozen: readonly PositionedBlock[]
  /** 重新解析的不稳定尾部，最多 UNSTABLE_TAIL_BLOCKS 块，再加上这一拍新长出来的。 */
  readonly tail: readonly PositionedBlock[]
  /** 非追加输入丢掉冻结前缀时加一。调用方按它丢掉缓存。 */
  readonly generation: number
  /** 不稳定尾部在全文里的起始偏移。冻结的 markdown 是 text.slice(0, tailStart)。 */
  readonly tailStart: number
}

/** 渲染键是块在全文里的起始偏移。没有位置时用负的列表下标，只在这一次 update 的尾部里唯一。冻结要求切开的那块有位置，所以没有位置的解析会把所有块留在尾部。node 读取 position。base 加上 offset。index 加上 1。 */
function blockKey(node: RootContent, base: number, index: number): number {
  const offset = node.position?.start.offset
  return offset === undefined ? -(index + 1) : base + offset
}

interface OpenFenceState {
  readonly marker: '`' | '~'
  readonly markerLength: number
  readonly syntheticPrefix: string
  readonly codeIndex: number
  readonly frozen: readonly PositionedBlock[]
  readonly tail: readonly PositionedBlock[]
  readonly pendingStart: number
  readonly valuePrefix: string
  readonly end: { readonly line: number; readonly column: number; readonly offset: number }
  readonly endedWithCarriageReturn: boolean
}

/** 返回 start 处或之后第一个行终止符的结束位置，CRLF 算一个。没有则空。text 读取 length，按下标读取。 */
function lineTerminatorEnd(text: string, start: number): number | undefined {
  for (let index = start; index < text.length; index += 1) {
    const char = text[index]
    if (char === '\n') return index + 1
    if (char === '\r') return text[index + 1] === '\n' ? index + 2 : index + 1
  }
  return undefined
}

/** 最后一行完整行之前的源码长度。把那一行留在当前半行旁边，语法才能保住结尾换行的含义。text 读取 length，按下标读取。 */
function committableLinePrefixLength(text: string): number {
  let previousEnd = 0
  let end = 0
  for (let index = 0; index < text.length; index += 1) {
    const char = text[index]
    if (char === '\n') {
      previousEnd = end
      end = index + 1
      continue
    }
    if (char !== '\r' || index + 1 >= text.length) continue
    if (text[index + 1] === '\n') index += 1
    previousEnd = end
    end = index + 1
  }
  return previousEnd
}

/** 非空可提交前缀结尾的那个行终止符。CRLF、CR，否则 LF。text 看结尾。 */
function trailingLineTerminator(text: string): '\n' | '\r' | '\r\n' {
  return text.endsWith('\r\n') ? '\r\n' : text.endsWith('\r') ? '\r' : '\n'
}

/** 某一逻辑行上是否有 CommonMark 闭合围栏：缩进不超过三个空格，标记不少于给定长度，其余只有空白。text 读取 length，截取，按下标读取。marker 与 line[run] 比较是否相等。markerLength 大于等于 run - indent。 */
function containsClosingFence(text: string, marker: '`' | '~', markerLength: number): boolean {
  let start = 0
  while (start <= text.length) {
    let end = start
    while (end < text.length && text[end] !== '\n' && text[end] !== '\r') end += 1
    const line = text.slice(start, end)
    let indent = 0
    while (indent < 3 && line[indent] === ' ') indent += 1
    let run = indent
    while (line[run] === marker) run += 1
    if (run - indent >= markerLength && /^[ \t]*$/.test(line.slice(run))) return true
    if (end === text.length) return false
    start = text[end] === '\r' && text[end + 1] === '\n' ? end + 2 : end + 1
  }
  /* v8 ignore next -- 每次循环要么在文件尾返回，要么越过一个行终止符。 */
  return false
}

/** 把一个 mdast 点按追加的文本往前推。被拆开的 CRLF 仍算一次换行。point 读取 line、column、offset。appended 读取 length，用来遍历。precededByCarriageReturn 为假时走另一支。 */
function advancePoint(
  point: OpenFenceState['end'],
  appended: string,
  precededByCarriageReturn: boolean,
): OpenFenceState['end'] {
  let line = point.line
  let column = point.column
  let afterCarriageReturn = precededByCarriageReturn
  for (const char of appended) {
    if (char === '\n') {
      if (!afterCarriageReturn) line += 1
      column = 1
      afterCarriageReturn = false
      continue
    }
    if (char === '\r') {
      line += 1
      column = 1
      afterCarriageReturn = true
      continue
    }
    column += 1
    afterCarriageReturn = false
  }
  return { line, column, offset: point.offset + appended.length }
}

/** IncrementalMarkdownParser 在调用方给的语法上做只追加的增量解析。一个实例累积一篇流式文档。输入不是追加时重置。 */
export class IncrementalMarkdownParser {
  private prevText = ''
  private tailStart = 0
  private frozen: PositionedBlock[] = []
  private generation = 0
  private cached: IncrementalBlocks | null = null
  private openFence: OpenFenceState | null = null
  private readonly parse: (text: string) => Root

  /** 语法与渲染块的那一方共用，边界才一致。parse 记到 this.parse。 */
  constructor(parse: (text: string) => Root) {
    this.parse = parse
  }

  /** 用调用方语法解析一片未闭合围栏的内容。根上不是恰好一个 code 节点则没有。state 读取 syntheticPrefix。text 拼进字符串。 */
  private fenceValue(state: Pick<OpenFenceState, 'syntheticPrefix'>, text: string): string | undefined {
    const root = this.parse(`${state.syntheticPrefix}${text}`)
    if (root.children.length !== 1) return undefined
    const node = root.children[0] as RootContent
    return node.type === 'code' ? node.value : undefined
  }

  /**
   * 认出尾部最后一块未闭合围栏，并准备它的增量内容前沿。对不上则空。
   * text 读取 length，截取，看结尾。
   * base 加上 end.offset，加上 terminatorEnd，传给 text.slice。
   * tail 读取 length，按下标读取，写入字段 tail。
   * frozen 写入字段 frozen。
   */
  private openFenceState(
    text: string,
    base: number,
    tail: readonly PositionedBlock[],
    frozen: readonly PositionedBlock[],
  ): OpenFenceState | null {
    const codeIndex = tail.length - 1
    const block = tail[codeIndex]
    if (block?.node.type !== 'code') return null
    const node = block.node
    const startOffset = node.position?.start.offset
    const end = node.position?.end
    if (startOffset === undefined || end?.offset === undefined) return null
    /* v8 ignore next -- 调用方的解析片以 text.length 结束，因此最后一个节点也结束在那里。 */
    if (base + end.offset !== text.length) return null
    const source = text.slice(base)
    const previousLf = source.lastIndexOf('\n', startOffset - 1)
    const previousCr = source.lastIndexOf('\r', startOffset - 1)
    const lineStart = Math.max(previousLf, previousCr) + 1
    const terminatorEnd = lineTerminatorEnd(source, startOffset)
    /* v8 ignore next -- 解析器确认过的围栏代码节点一定有开头的行终止符。 */
    if (terminatorEnd === undefined) return null
    if (terminatorEnd === source.length && source.endsWith('\r')) return null
    const openingLine = source.slice(lineStart, terminatorEnd).replace(/[\r\n]+$/, '')
    const opening = /^( {0,3})(`{3,}|~{3,})/.exec(openingLine)
    if (opening === null) return null
    const indent = opening[1] as string
    const run = opening[2] as string
    /* v8 ignore next -- mdast 把围栏代码节点的位置放在缩进之后匹配到的定界符上。 */
    if (lineStart + indent.length !== startOffset) return null
    const marker = run[0] as '`' | '~'
    const contentStart = base + terminatorEnd
    const content = text.slice(contentStart)
    if (containsClosingFence(content, marker, run.length)) return null
    const syntheticPrefix = `${indent}${run}\n`
    const stableLength = committableLinePrefixLength(content)
    const stableValue = stableLength === 0
      ? ''
      : this.fenceValue({ syntheticPrefix }, content.slice(0, stableLength))
    if (stableValue === undefined) return null
    const pendingStart = contentStart + stableLength
    const stableSource = content.slice(0, stableLength)
    const valuePrefix = stableLength === 0 ? '' : `${stableValue}${trailingLineTerminator(stableSource)}`
    const pendingValue = this.fenceValue({ syntheticPrefix }, text.slice(pendingStart))
    if (pendingValue === undefined || `${valuePrefix}${pendingValue}` !== node.value) return null
    return {
      marker,
      markerLength: run.length,
      syntheticPrefix,
      codeIndex,
      frozen,
      tail,
      pendingStart,
      valuePrefix,
      end: { line: end.line, column: end.column, offset: end.offset },
      endedWithCarriageReturn: text.endsWith('\r'),
    }
  }

  /**
   * 延长已认出的未闭合围栏，不再解析已经完成的内容前缀。出现闭合围栏或语法对不上则空。
   * state 读取 pendingStart、marker、markerLength、tail、codeIndex、end 等，传给 fenceValue，展开传递。
   * text 截取，看结尾。
   * previousText 读取 length。
   */
  private updateOpenFence(
    state: OpenFenceState,
    text: string,
    previousText: string,
  ): IncrementalBlocks | undefined {
    const pending = text.slice(state.pendingStart)
    if (containsClosingFence(pending, state.marker, state.markerLength)) return undefined
    const pendingValue = this.fenceValue(state, pending)
    if (pendingValue === undefined) return undefined
    const stableLength = committableLinePrefixLength(pending)
    const stableValue = stableLength === 0
      ? ''
      : this.fenceValue(state, pending.slice(0, stableLength))
    if (stableValue === undefined) return undefined
    // OpenFenceState 是私有的，只从这条保留的 code 装入；更新会换成另一块带位置的 Code。
    const block = state.tail[state.codeIndex] as PositionedBlock
    const previousNode = block.node as Code & { position: NonNullable<Code['position']> }
    const end = advancePoint(
      state.end,
      text.slice(previousText.length),
      state.endedWithCarriageReturn,
    )
    const node: Code = {
      ...previousNode,
      value: `${state.valuePrefix}${pendingValue}`,
      position: { start: previousNode.position.start, end },
    }
    const tail = state.tail.map((entry, index) => index === state.codeIndex ? { ...entry, node } : entry)
    const cached = {
      frozen: state.frozen,
      tail,
      generation: this.generation,
      tailStart: this.tailStart,
    }
    this.openFence = {
      ...state,
      tail,
      pendingStart: state.pendingStart + stableLength,
      valuePrefix: stableLength === 0
        ? state.valuePrefix
        : `${state.valuePrefix}${stableValue}${trailingLineTerminator(pending.slice(0, stableLength))}`,
      end,
      endedWithCarriageReturn: text.endsWith('\r'),
    }
    return cached
  }

  /** update 折当前累积全文，返回冻结和尾部。相同输入直接返回上一次结果，渲染路径重复调用也安全。text 是累积的全部 markdown。 */
  update(text: string): IncrementalBlocks {
    if (this.cached !== null && text === this.prevText) return this.cached
    // 每次更新故意对前缀做线性比较：要确认保留的整段前缀没变，startsWith 比把这些字节再解析一遍快两个数量级，这正是这个类要去掉的成本。改传追加或重置增量，会把追加记账推过会话投影的更新边界，而按实际回复长度这项检查仍在一毫秒以内。
    if (!text.startsWith(this.prevText)) {
      this.prevText = ''
      this.tailStart = 0
      this.frozen = []
      this.openFence = null
      this.generation += 1
    }
    const previousText = this.prevText
    if (previousText !== '' && this.openFence !== null) {
      const incremental = this.updateOpenFence(this.openFence, text, previousText)
      if (incremental !== undefined) {
        this.prevText = text
        this.cached = incremental
        return incremental
      }
      this.openFence = null
    }
    this.prevText = text
    const base = this.tailStart
    const blocks = this.parse(text.slice(base)).children
    let firstUnstable = Math.max(0, blocks.length - UNSTABLE_TAIL_BLOCKS)
    if (firstUnstable > 0) {
      const cutEnd = blocks[firstUnstable - 1]?.position?.end.offset
      if (cutEnd === undefined) {
        // 语法不给位置就没有切口。整段解析留在尾部，不猜边界。
        firstUnstable = 0
      } else {
        for (const node of blocks.slice(0, firstUnstable)) {
          this.frozen.push({ node, key: blockKey(node, base, this.frozen.length) })
        }
        this.tailStart = base + cutEnd
      }
    }
    const tail = blocks.slice(firstUnstable).map((node, index) => ({
      node,
      key: blockKey(node, base, index),
    }))
    this.cached = { frozen: [...this.frozen], tail, generation: this.generation, tailStart: this.tailStart }
    this.openFence = this.openFenceState(text, base, tail, this.cached.frozen)
    return this.cached
  }
}
