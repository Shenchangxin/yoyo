// Package preview 把文件转成一段文本预览。
// xlsx 列出单元格，求值成功时再附上数值。docx、pptx 和 pdf 交给 office.Query。
// 其余扩展名按原文读，长度超过 12000 字节时只留前面 12000 字节。
package preview

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/office"
)

// Render 按扩展名生成预览文本。扩展名比较前转成小写。
// xlsx 先写单元格，标题是 spreadsheet；EvaluateXlsx 成功时再写 evaluated。求值失败则只有单元格。
// docx 和 pptx 返回 office.Query 的文本。pdf 同样走 office.Query。
// path 取出扩展名并转成小写。.xlsx 时传给 office.ReadXlsx 和 office.EvaluateXlsx，.docx、.pptx、.pdf 时传给 office.Query，其余传给 os.ReadFile。
func Render(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".xlsx":
		cells, err := office.ReadXlsx(path)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		b.WriteString("# spreadsheet\n\n")
		for k, v := range cells {
			fmt.Fprintf(&b, "- %s: %s\n", k, v)
		}
		if vals, err := office.EvaluateXlsx(path); err == nil {
			b.WriteString("\n# evaluated\n")
			for k, v := range vals {
				fmt.Fprintf(&b, "- %s = %g\n", k, v)
			}
		}
		return b.String(), nil
	case ".docx", ".pptx":
		text, err := office.Query(path)
		if err != nil {
			return "", err
		}
		return text, nil
	case ".pdf":
		return office.Query(path)
	default:
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		if len(raw) > 12_000 {
			raw = raw[:12_000]
		}
		return string(raw), nil
	}
}
