package video

import (
	"regexp"
	"sort"
	"strings"
)

// AssetRef is a named still used as a multimodal reference slot.
type AssetRef struct {
	Name string
	URL  string // data URL or http(s)
}

var atName = regexp.MustCompile(`@([^\s@]+)`)

// ResolvePromptRefs rewrites @Name to @图片NName (or Wan 图N) so the model
// token lines up with reference_image_urls[N-1]. Longer names win.
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
