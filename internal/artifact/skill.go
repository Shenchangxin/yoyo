// SKILL.md 的解析和隐式调用判断。不把技能写进对象库。运行时、技能包、市场扫描和演化会来解析。

package artifact

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	// DescriptionZH 在 description 为空时顶上。
	DescriptionZH string `yaml:"description_zh"`
	DisplayName   string `yaml:"display_name"`
	Icon          string `yaml:"icon"`
	License       string `yaml:"license"`
	Compatibility string `yaml:"compatibility"`
	Metadata      any    `yaml:"metadata"`
	// AllowedTools 可以是字符串或列表，解析时收成一段。
	AllowedTools           any  `yaml:"allowed-tools"`
	DisableModelInvocation bool `yaml:"disable-model-invocation"`
}

// ParseSkillMD 解析 SKILL.md。会先把 CRLF 收成 LF；必须以 --- 开头的 YAML 前置信息开始，并用换行加 --- 结束。
// 名字和描述都必填，描述为空时改用 description_zh。按字节计，名字超过 64 或描述超过 4096 就拒绝。
// 图标不合法时留空，不报错。嵌套 metadata 会压成点分键。
func ParseSkillMD(raw, source string) (Skill, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(raw, "---\n") {
		return Skill{}, fmt.Errorf("skill: missing YAML frontmatter")
	}
	rest := strings.TrimPrefix(raw, "---\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return Skill{}, fmt.Errorf("skill: unterminated frontmatter")
	}
	fm := rest[:end]
	body := strings.TrimSpace(rest[end+4:])
	var meta skillFrontmatter
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return Skill{}, fmt.Errorf("skill: frontmatter: %w", err)
	}
	if meta.Name == "" {
		return Skill{}, fmt.Errorf("skill: name and description are required")
	}
	desc := strings.TrimSpace(meta.Description)
	if desc == "" {
		desc = strings.TrimSpace(meta.DescriptionZH)
	}
	if desc == "" {
		return Skill{}, fmt.Errorf("skill: name and description are required")
	}
	if len(meta.Name) > 64 {
		return Skill{}, fmt.Errorf("skill: name too long")
	}
	if len(desc) > 4096 {
		return Skill{}, fmt.Errorf("skill: description too long")
	}
	return Skill{
		Name:                   meta.Name,
		Description:            desc,
		DisplayName:            strings.TrimSpace(meta.DisplayName),
		Icon:                   SafeIconURL(meta.Icon),
		License:                meta.License,
		Compatibility:          meta.Compatibility,
		Metadata:               flattenMeta(meta.Metadata),
		AllowedTools:           anyString(meta.AllowedTools),
		DisableModelInvocation: meta.DisableModelInvocation,
		Body:                   body,
		Source:                 source,
		Dir:                    skillDir(source),
	}, nil
}

// AllowsImplicit 判断能否不经显式调用就载入。DisableModelInvocation 为真时返回假。
// Metadata 的键名不区分大小写，空值忽略。以 allow_implicit_invocation 或 allow-implicit-invocation 结尾，且值为 false、0、no、off 时返回假。
// 以 disable-model-invocation 或 disable_model_invocation 结尾，且值为 true、1、yes、on 时也返回假。其余情况返回真。
func (s Skill) AllowsImplicit() bool {
	if s.DisableModelInvocation {
		return false
	}
	for k, v := range s.Metadata {
		kl := strings.ToLower(k)
		vl := strings.ToLower(strings.TrimSpace(v))
		if vl == "" {
			continue
		}
		if strings.HasSuffix(kl, "allow_implicit_invocation") || strings.HasSuffix(kl, "allow-implicit-invocation") {
			if vl == "false" || vl == "0" || vl == "no" || vl == "off" {
				return false
			}
		}
		if strings.HasSuffix(kl, "disable-model-invocation") || strings.HasSuffix(kl, "disable_model_invocation") {
			if vl == "true" || vl == "1" || vl == "yes" || vl == "on" {
				return false
			}
		}
	}
	return true
}

// skillDir 在来源以 skill.md 结尾时返回所在目录，大小写不敏感。空来源、builtin，以及不以 skill.md 结尾的来源，返回空串。
func skillDir(source string) string {
	if source == "" || source == "builtin" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(source), "skill.md") {
		return filepath.Dir(source)
	}
	return ""
}

// LoadSkillFile 读路径上的 SKILL.md，并把该路径当作来源。
func LoadSkillFile(path string) (Skill, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Skill{}, err
	}
	return ParseSkillMD(string(b), path)
}

// CatalogLine 拼成「名字: 描述」。
func (s Skill) CatalogLine() string {
	return s.Name + ": " + s.Description
}

// SafeIconURL 只接受带主机名的 https。http、相对路径、javascript:、file:，以及 localhost、127.0.0.1、::1 和以 .localhost 结尾的主机，都返回空串。
func SafeIconURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".localhost") {
		return ""
	}
	return u.String()
}

// anyString 把字符串、列表或其他值收成一段文本。列表用空格拼接，空项丢掉。
func anyString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			if s := anyString(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

// flattenMeta 把嵌套 metadata 压成点分键。没有可写的叶子时返回 nil，不是空映射。
func flattenMeta(v any) map[string]string {
	out := map[string]string{}
	walkMeta("", v, out)
	if len(out) == 0 {
		return nil
	}
	return out
}

// walkMeta 把标量记在前缀下。两种映射都继续下钻，列表整段收成一个值。前缀为空的标量不写入。
func walkMeta(prefix string, v any, out map[string]string) {
	switch t := v.(type) {
	case nil:
		return
	case string:
		if prefix != "" && strings.TrimSpace(t) != "" {
			out[prefix] = strings.TrimSpace(t)
		}
	case map[string]any:
		for k, child := range t {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			walkMeta(key, child, out)
		}
	case map[any]any:
		for k, child := range t {
			key := fmt.Sprint(k)
			if prefix != "" {
				key = prefix + "." + key
			}
			walkMeta(key, child, out)
		}
	case []any:
		if prefix != "" {
			out[prefix] = anyString(t)
		}
	default:
		if prefix != "" {
			s := strings.TrimSpace(fmt.Sprint(t))
			if s != "" {
				out[prefix] = s
			}
		}
	}
}
