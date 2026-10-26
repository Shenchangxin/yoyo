// 本文件按模型 id 归族。models.dev 没有给出 family，或 family 与 id 只是大小写不同时，convertModel 调用 FamilyOf。
// dateSuffix 也给 catalog.go 的 lookup 去掉 id 末尾的 8 位日期。

package modelcatalog

import (
	"regexp"
	"strings"
)

var (
	// dateSuffix 匹配 id 末尾的一个连字符加 8 位数字。
	dateSuffix = regexp.MustCompile(`-\d{8}$`)
	// versionTok 匹配一段版本号，可以有前缀 v，末尾可以有一个小写字母。
	versionTok = regexp.MustCompile(`^(?:v)?\d+(?:\.\d+)*[a-z]?$`)
	// oSeries 匹配 o 加数字，末尾可以有一个小写字母。FamilyOf 把这种段收成 o。
	oSeries = regexp.MustCompile(`^o\d+[a-z]?$`)
)

// FamilyOf 把带日期或版本的模型 id 收成族名，算法与随包快照一致，例如 claude-sonnet-4-6 得到 claude-sonnet，gpt-5.2-pro 得到 gpt-pro。
// models.dev 省略 family 时用它。先取小写，若有斜杠则只留最后一段，再去掉末尾 -\d{8}、-latest 和 -preview。
// 以 text-embedding 开头时返回 text-embedding。含 gpt-image，或以 dall-e、chatgpt-image 开头时返回 gpt-image。
// 其余按连字符分段：空段丢掉，o 系列收成 o，版本号段丢掉，reasoner 改成 thinking。一段都不剩时，原串空则返回调用方传入的 id，否则返回去掉后缀后的串。
func FamilyOf(id string) string {
	s := strings.ToLower(strings.TrimSpace(id))
	if i := strings.LastIndex(s, "/"); i >= 0 && i+1 < len(s) {
		s = s[i+1:]
	}
	s = dateSuffix.ReplaceAllString(s, "")
	s = strings.TrimSuffix(s, "-latest")
	s = strings.TrimSuffix(s, "-preview")
	if strings.HasPrefix(s, "text-embedding") {
		return "text-embedding"
	}
	if strings.Contains(s, "gpt-image") || strings.HasPrefix(s, "dall-e") || strings.HasPrefix(s, "chatgpt-image") {
		return "gpt-image"
	}
	var parts []string
	for _, p := range strings.Split(s, "-") {
		if p == "" {
			continue
		}
		if oSeries.MatchString(p) {
			parts = append(parts, "o")
			continue
		}
		if versionTok.MatchString(p) {
			continue
		}
		if p == "reasoner" {
			p = "thinking"
		}
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		if s == "" {
			return id
		}
		return s
	}
	return strings.Join(parts, "-")
}
