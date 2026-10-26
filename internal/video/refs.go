// refs.go 把提示词里的 @名字 换成与参考图序号对齐的记号。

package video

import (
	"regexp"
	"sort"
	"strings"
)

// AssetRef 是一张带名字的静帧，用作多模态参考槽。
type AssetRef struct {
	Name string
	URL  string // 数据 URL 或 http(s)
}

// atName 匹配 @ 后面一段不含空白和 @ 的名字。
var atName = regexp.MustCompile(`@([^\s@]+)`)

// ResolvePromptRefs 把 @名字 改写成 @图片N 加原名，wan 为真时改成图N。序号从 1 起，与返回的 URL 切片下标 N-1 对齐。较长的名字先匹配。没有可用参考时原样返回提示词，URL 为 nil。
func ResolvePromptRefs(prompt string, refs []AssetRef, wan bool) (rewritten string, urls []string) {
	ordered := uniqueNamed(refs)
	if len(ordered) == 0 {
		if wan {
			return prompt, nil
		}
		return prompt, nil
	}
	index := map[string]int{}
	for i, a := range ordered {
		index[a.Name] = i + 1
		urls = append(urls, a.URL)
	}
	names := make([]string, 0, len(index))
	for n := range index {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	rewritten = atName.ReplaceAllStringFunc(prompt, func(m string) string {
		raw := strings.TrimPrefix(m, "@")
		for _, name := range names {
			if strings.HasPrefix(raw, name) {
				n := index[name]
				if wan {
					return "图" + itoa(n) + raw[len(name):]
				}
				return "@图片" + itoa(n) + name + raw[len(name):]
			}
		}
		return m
	})
	if wan {
		rewritten = strings.ReplaceAll(rewritten, "@图片", "图")
	}
	return rewritten, urls
}

// uniqueNamed 丢掉空名字、空 URL 和重复名字，保留第一次出现的顺序。
func uniqueNamed(refs []AssetRef) []AssetRef {
	seen := map[string]bool{}
	var out []AssetRef
	for _, a := range refs {
		name := strings.TrimSpace(a.Name)
		if name == "" || a.URL == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, AssetRef{Name: name, URL: a.URL})
	}
	return out
}

// itoa 把非负整数转成十进制文本。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
