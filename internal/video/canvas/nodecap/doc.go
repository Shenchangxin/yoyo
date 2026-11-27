// Package nodecap 描述画布节点的能力：每种节点能接收什么输入、能改哪些字段。
//
// Descriptor 是一种节点的描述，ConnectionPolicy 和 PatchField 分别约束连线和可改字段。Registry 登记这些描述，
// 并给出稳定的能力集哈希，画布 Agent 和工具菜单用它判断能力集有没有变。BuiltinRegistry 登记内置的文本、Markdown、
// 图片、视频、音频、背板、批量创作表和分镜脚本节点。这个包只描述能力，不读写画布文档。
package nodecap
