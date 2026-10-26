/**
 * 拥有：对话列的宽度和左右留白类名。
 * 不拥有：空首页和启动英雄区的排版。`--thread-measure` 只给那两处封顶。
 * 留白跟这一列的容器查询走，打开两侧栏时字宽跟着变宽。滚动容器保持铺满窗格，滚动条贴着窗格边缘。
 */

/** THREAD_MAX 带 `--thread-measure` 封顶的类，只用于空首页和启动英雄区。 */
export const THREAD_MAX = "max-w-[min(100%,var(--thread-measure))]";
/** THREAD_COL 进行中的对话列，不套那道封顶。 */
export const THREAD_COL = "w-full min-w-0";
/** THREAD_GUTTER 随容器变宽加大的左右留白。 */
export const THREAD_GUTTER = "px-4 @[28rem]:px-6 @[42rem]:px-8 @[60rem]:px-10";
/** THREAD_GUTTER_COMPACT 紧凑布局的左右留白。 */
export const THREAD_GUTTER_COMPACT = "px-3";
