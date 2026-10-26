// documentation.go 用内嵌的 Markdown 填上随包协议的说明。

package protocol

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed docs/*.md
var builtinDocumentationFS embed.FS

// attachDocumentation 在说明还空着时读取 docs/<id>.md。文件不在就保持原样。
// info 为 nil，或 Documentation 去掉空白后非空时直接返回；否则按 ID 读 docs 下的 Markdown，再交给 renderBuiltinDocumentation。
func attachDocumentation(info *Metadata) {
	if info == nil || strings.TrimSpace(info.Documentation) != "" {
		return
	}
	document, err := builtinDocumentationFS.ReadFile("docs/" + info.ID + ".md")
	if err != nil {
		return
	}
	info.Documentation = renderBuiltinDocumentation(*info, string(document))
}

// AttachDocumentation 给随包协议填上参考说明。
// info 传给 attachDocumentation。
func AttachDocumentation(info *Metadata) {
	attachDocumentation(info)
}

// renderBuiltinDocumentation 替换 {{NAME}}、{{OPERATIONS}}、{{PARAMETERS}} 和 {{CONTRACT}}，并在末尾补一个换行。
// info 的 Name 替换 {{NAME}}，整份元数据传给 renderOperations 和 renderRuntimeContract，Parameters 传给 renderParameters。
// document 按这四个标记替换后去掉首尾空白，再补一个换行。
func renderBuiltinDocumentation(info Metadata, document string) string {
	replacements := map[string]string{
		"{{NAME}}":       info.Name,
		"{{OPERATIONS}}": renderOperations(info),
		"{{PARAMETERS}}": renderParameters(info.Parameters),
		"{{CONTRACT}}":   renderRuntimeContract(info),
	}
	for marker, replacement := range replacements {
		document = strings.ReplaceAll(document, marker, replacement)
	}
	return strings.TrimSpace(document) + "\n"
}

// renderOperations 把创建，以及非空的轮询和取消，写成一张表。
// info 的 Create 总是写入表格；Poll、Cancel 去掉空白后非空才写入，ContentType 传给 firstNonEmpty。
func renderOperations(info Metadata) string {
	var b strings.Builder
	b.WriteString("| 阶段 | 方法与相对路径 | 请求类型 | 宿主行为 |\n| --- | --- | --- | --- |\n")
	writeOperation(&b, "创建", info.Create, firstNonEmpty(info.ContentType, "application/json"), "提交请求；同步协议直接解析结果，异步协议读取任务 ID")
	if strings.TrimSpace(info.Poll) != "" {
		writeOperation(&b, "轮询", info.Poll, "application/json", "查询状态，成功后提取媒体地址")
	}
	if strings.TrimSpace(info.Cancel) != "" {
		writeOperation(&b, "取消", info.Cancel, "application/json", "取消仍在运行的任务")
	}
	return strings.TrimSpace(b.String())
}

// renderParameters 把参数写成表。没有参数时说明宿主还没有稳定映射，不表示上游没有参数。
// parameters 长度为 0 时返回一句说明；否则逐条写 Name、Type、Required、Mapping、Description 和 Values。
func renderParameters(parameters []Parameter) string {
	if len(parameters) == 0 {
		return "> 当前适配器没有声明可配置参数；这不表示上游没有参数，只表示宿主尚未建立稳定映射。"
	}
	var b strings.Builder
	b.WriteString("| 宿主字段 | 类型 | 必填 | 实际上游映射 | 说明 |\n| --- | --- | --- | --- | --- |\n")
	for _, parameter := range parameters {
		description := strings.TrimSpace(parameter.Description)
		if len(parameter.Values) > 0 {
			description = strings.TrimSpace(description + " 可选值：" + strings.Join(parameter.Values, "、") + "。")
		}
		fmt.Fprintf(&b, "| `%s` | `%s` | %s | `%s` | %s |\n", markdownCell(parameter.Name), markdownCell(parameter.Type), yesNo(parameter.Required), markdownCell(firstNonEmpty(parameter.Mapping, "未映射")), markdownCell(firstNonEmpty(description, "由当前协议适配器按请求类型转换。")))
	}
	return strings.TrimSpace(b.String())
}

// renderRuntimeContract 写出宿主运行时合同：基址、密钥、模型名、错误、异步状态和参考素材地址。有轮询路径时才写状态和轮询那两行。
// info 的 Poll 去掉空白后非空时，多写异步状态和轮询两行。
func renderRuntimeContract(info Metadata) string {
	var b strings.Builder
	b.WriteString("## 宿主运行时合同\n\n")
	b.WriteString("- Base URL 只填写协议服务根地址，表中的相对路径由适配器拼接；不要把同一路径重复写进 Base URL。\n")
	b.WriteString("- 密钥由后端渠道中转读取，插件详情和浏览器请求不保存密钥。Bearer、供应商签名或专用版本头以本文鉴权章节为准。\n")
	b.WriteString("- 模型名来自渠道模型配置并原样发送。此插件不暗改模型名，也不根据名称猜测价格、额度或能力。\n")
	b.WriteString("- HTTP 非 2xx、非 JSON 响应、缺少任务 ID、成功状态却没有可识别结果都会作为真实错误向上返回。\n")
	if strings.TrimSpace(info.Poll) != "" {
		b.WriteString("- 异步状态统一为：`queued/pending/created/submitted` -> 等待，`running/processing/in_progress` -> 处理中，`succeeded/completed/done` -> 成功，`failed/error/expired/cancelled` -> 失败。\n")
		b.WriteString("- 轮询频率应遵守上游速率限制；媒体 URL 通常有有效期，成功后应立即进入宿主资源保存流程。\n")
	}
	b.WriteString("- 参考素材只接受上游可访问的 HTTP(S) URL 或带 MIME 前缀的 data URL；裸 Base64、登录后才能访问的链接和 Cookie 保护地址不可用。\n")
	b.WriteString("- 插件只保证下方“当前实现”明确列出的字段。上游新增能力不会自动获得宿主支持，必须补适配、解析和测试。\n")
	return strings.TrimSpace(b.String())
}

// writeOperation 把「方法 路径」拆开写进表格的一行。没有空格时整段当路径。
// b 接收这一行表格。
// phase 写在第一列。
// value 去掉空白后按第一个空格拆成方法和路径；没有空格时整段当路径。
// contentType 经 markdownCell 写进请求类型列。
// purpose 写在最后一列。
func writeOperation(b *strings.Builder, phase, value, contentType, purpose string) {
	parts := strings.SplitN(strings.TrimSpace(value), " ", 2)
	method, path := "", strings.TrimSpace(value)
	if len(parts) == 2 {
		method, path = parts[0], parts[1]
	}
	fmt.Fprintf(b, "| %s | `%s %s` | `%s` | %s |\n", phase, markdownCell(method), markdownCell(path), markdownCell(contentType), purpose)
}

// markdownCell 把竖线和换行换成表格里能放的文本。
// value 里的竖线换成反斜杠加竖线，换行换成空格。
func markdownCell(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "|", "\\|"), "\n", " ")
}

// yesNo 把真写成「是」，假写成「否」。
// value 为真时返回「是」，否则返回「否」。
func yesNo(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

// firstNonEmpty 返回第一段去掉空白后仍非空的文本。
// values 逐段 TrimSpace，返回第一段非空的原文，都空则返回空串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
