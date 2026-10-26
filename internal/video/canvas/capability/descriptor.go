// descriptor.go 描述一种画布节点能接收什么输入、能改哪些字段。

package capability

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
)

// 补丁字段只有字符串、数字和布尔三种。
const (
	// patchKindString 是字符串补丁 string。
	patchKindString  = "string"
	// patchKindNumber 是数字补丁 number。
	patchKindNumber  = "number"
	// patchKindBoolean 是布尔补丁 boolean。
	patchKindBoolean = "boolean"
)

// ConnectionPolicy 规定节点能否作为连线来源、目标和参考，以及接受或拒绝的输入种类和数量上限。
type ConnectionPolicy struct {
	CanSource          bool
	CanTarget          bool
	CanReference       bool
	AcceptedInputKinds []string
	RejectedInputKinds []string
	MaxInputCount      int
}

// PatchField 是一个可写字段的路径、类型、标签、顺序、数值上限和字数上限。
type PatchField struct {
	Path  string
	Kind  string
	Label string
	Order int
	// Limit > 0 时表示数值字段的绝对值上限（坐标用），避免模型写入离谱的几何值。
	Limit       float64
	Description string
	MaxRunes    int
}

// Descriptor 是服务端保存的画布节点合同。代理工具、创建、持久化和状态投影都读这份登记，调用方不再另备一份节点允许名单。
type Descriptor struct {
	Type            string
	Version         string
	Label           string
	Purpose         string
	GoodFor         []string
	NotIdealFor     []string
	Tradeoffs       []string
	Actions         []string
	DefaultWidth    float64
	DefaultHeight   float64
	InputKind       string
	GenerationMode  string
	Connection      ConnectionPolicy
	CanUpdate       bool
	SummaryFields   []string
	DetailFields    []string
	ProjectionKind  string
	ProjectionField string
	PatchFields     map[string]PatchField
	CreateMetadata  func(content string) map[string]any
}

// Metadata 用 CreateMetadata 生成初始元数据。没有该函数时返回 content 和 status 为 idle。
func (d Descriptor) Metadata(content string) map[string]any {
	if d.CreateMetadata != nil {
		return d.CreateMetadata(content)
	}
	return map[string]any{"content": content, "status": "idle"}
}

// AllowsInput 在节点不能当目标时为假。拒绝列表优先，接受列表为空则其余都允许。
func (d Descriptor) AllowsInput(kind string) bool {
	if !d.Connection.CanTarget {
		return false
	}
	kind = normalizeType(kind)
	for _, rejected := range d.Connection.RejectedInputKinds {
		if rejected == kind {
			return false
		}
	}
	if len(d.Connection.AcceptedInputKinds) == 0 {
		return true
	}
	for _, accepted := range d.Connection.AcceptedInputKinds {
		if accepted == kind {
			return true
		}
	}
	return false
}

// ValidatePatch 要求节点可更新且补丁非空。键必须在补丁字段里，值要符合字符串、数字或布尔。数字不能是 NaN 或无穷，绝对值不能超过 Limit。字符串不能超过 MaxRunes。
func (d Descriptor) ValidatePatch(patch map[string]any) error {
	if !d.CanUpdate {
		return fmt.Errorf("%s 节点不支持更新", d.Label)
	}
	if len(patch) == 0 {
		return fmt.Errorf("%s 节点更新内容不能为空", d.Label)
	}
	for key, value := range patch {
		field, allowed := d.PatchFields[key]
		if !allowed {
			return fmt.Errorf("%s 节点不支持更新字段 %s", d.Label, key)
		}
		switch field.Kind {
		case "string":
			text, ok := value.(string)
			if !ok {
				return fmt.Errorf("%s 字段 %s 必须是字符串", d.Label, key)
			}
			if field.MaxRunes > 0 && len([]rune(text)) > field.MaxRunes {
				return fmt.Errorf("%s 字段 %s 超出长度限制", d.Label, key)
			}
		case "number":
			number, ok := value.(float64)
			if !ok {
				return fmt.Errorf("%s 字段 %s 必须是数字", d.Label, key)
			}
			if math.IsNaN(number) || math.IsInf(number, 0) {
				return fmt.Errorf("%s 字段 %s 不是有效数字", d.Label, key)
			}
			if field.Limit > 0 && math.Abs(number) > field.Limit {
				return fmt.Errorf("%s 字段 %s 超出允许范围（±%g）", d.Label, key, field.Limit)
			}
		case "boolean":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("%s 字段 %s 必须是布尔值", d.Label, key)
			}
		default:
			return fmt.Errorf("%s 字段 %s 的类型契约无效", d.Label, key)
		}
	}
	return nil
}

// ApplyPatch 先校验，再按点号路径写入节点。中间层不是对象时新建。
func (d Descriptor) ApplyPatch(node map[string]any, patch map[string]any) error {
	if err := d.ValidatePatch(patch); err != nil {
		return err
	}
	for key, value := range patch {
		field := d.PatchFields[key]
		parts := strings.Split(field.Path, ".")
		target := node
		for _, part := range parts[:len(parts)-1] {
			child, ok := target[part].(map[string]any)
			if !ok {
				child = map[string]any{}
				target[part] = child
			}
			target = child
		}
		target[parts[len(parts)-1]] = value
	}
	return nil
}

// ValidateConnection 要求节点能接收输入，并且 AllowsInput 接受来源种类。
func (d Descriptor) ValidateConnection(fromKind string) error {
	if !d.Connection.CanTarget {
		return fmt.Errorf("%s 节点不能接收参考输入", d.Label)
	}
	if !d.AllowsInput(fromKind) {
		return fmt.Errorf("%s生成节点不接受%s输入", d.Label, inputKindLabel(fromKind))
	}
	return nil
}

// inputKindLabel 把 image、video、audio 写成图片、视频、音频，其他写成文本。
func inputKindLabel(kind string) string {
	switch kind {
	case "image":
		return "图片"
	case "video":
		return "视频"
	case "audio":
		return "音频"
	default:
		return "文本"
	}
}

// normalizeType 去掉空白并转成小写。
func normalizeType(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

// normalizeInputKind 用「canvas input kind」校验输入种类标识。
func normalizeInputKind(value string) (string, error) {
	return normalizeCapabilityIdentifier(value, "canvas input kind")
}

// normalizeGenerationMode 用「canvas generation mode」校验生成模式标识。
func normalizeGenerationMode(value string) (string, error) {
	return normalizeCapabilityIdentifier(value, "canvas generation mode")
}

// normalizeProjectionKind 用「canvas projection kind」校验投影种类标识。
func normalizeProjectionKind(value string) (string, error) {
	return normalizeCapabilityIdentifier(value, "canvas projection kind")
}

// normalizeCapabilityIdentifier 要求空串直接通过，否则以小写字母开头，后面只能是小写字母、数字、下划线或横线。
func normalizeCapabilityIdentifier(value, label string) (string, error) {
	value = normalizeType(value)
	if value == "" {
		return "", nil
	}
	for index, r := range value {
		if index == 0 {
			if r < 'a' || r > 'z' {
				return "", fmt.Errorf("invalid %s %q", label, value)
			}
			continue
		}
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return "", fmt.Errorf("invalid %s %q", label, value)
		}
	}
	return value, nil
}

// normalizeConnectionKinds 规范化、去重并排序输入种类。空种类失败。
func normalizeConnectionKinds(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		kind, err := normalizeInputKind(value)
		if err != nil {
			return nil, err
		}
		if kind == "" {
			return nil, fmt.Errorf("canvas connection input kind cannot be empty")
		}
		if _, exists := seen[kind]; exists {
			return nil, fmt.Errorf("duplicate canvas connection input kind %q", kind)
		}
		seen[kind] = struct{}{}
		out = append(out, kind)
	}
	sort.Strings(out)
	return out, nil
}

// validatePatchPath 拒绝空段、.、..、__proto__、constructor 和 prototype。每段要以字母或下划线开头。
func validatePatchPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("canvas patch field path cannot be empty")
	}
	for _, segment := range strings.Split(path, ".") {
		if segment == "" || segment == "." || segment == ".." || segment == "__proto__" || segment == "constructor" || segment == "prototype" {
			return fmt.Errorf("invalid canvas patch field path %q", path)
		}
		for index, r := range segment {
			if index == 0 {
				if r != '_' && !unicode.IsLetter(r) {
					return fmt.Errorf("invalid canvas patch field path %q", path)
				}
				continue
			}
			if r != '_' && r != '-' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return fmt.Errorf("invalid canvas patch field path %q", path)
			}
		}
	}
	return nil
}

// normalizePatchFields 检查补丁键、路径、标签、正的显示顺序和种类。MaxRunes 不能为负。
func normalizePatchFields(fields map[string]PatchField) (map[string]PatchField, error) {
	out := make(map[string]PatchField, len(fields))
	for key, field := range fields {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("canvas patch field key cannot be empty")
		}
		if _, exists := out[key]; exists {
			return nil, fmt.Errorf("duplicate canvas patch field %q", key)
		}
		field.Path = strings.TrimSpace(field.Path)
		if err := validatePatchPath(field.Path); err != nil {
			return nil, err
		}
		field.Kind = normalizeType(field.Kind)
		field.Label = strings.TrimSpace(field.Label)
		field.Description = strings.TrimSpace(field.Description)
		if field.Label == "" {
			return nil, fmt.Errorf("canvas patch field %q requires a user-facing label", key)
		}
		if field.Order <= 0 {
			return nil, fmt.Errorf("canvas patch field %q requires a positive display order", key)
		}
		switch field.Kind {
		case patchKindString, patchKindNumber, patchKindBoolean:
		default:
			return nil, fmt.Errorf("unsupported canvas patch field kind %q", field.Kind)
		}
		if field.MaxRunes < 0 {
			return nil, fmt.Errorf("canvas patch field %q has invalid max runes", key)
		}
		out[key] = field
	}
	return out, nil
}
