/**
 * 拥有：皮肤模块的公共入口，把同目录的定义、调色、校验、对比度、上屏和取色转出。
 * 不拥有：这些能力的实现。
 */

/** 转出令牌规格、皮肤类型和本地存储键。 */
export * from "./schema.ts";
/** 转出内置调色板。 */
export * from "./builtin.ts";
/** 转出令牌名和值的校验。 */
export * from "./validate.ts";
/** 转出对比度和纱厚度。 */
export * from "./contrast.ts";
/** 转出把皮肤写到文档和窗口。 */
export * from "./apply.ts";
/** 转出从壁纸取色和取亮度。 */
export * from "./extract.ts";
