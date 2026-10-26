// registry.go 登记画布节点描述，并给出稳定的能力集哈希。

package capability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SetVersion 是能力集版本 canvas-capabilities/v4。
const SetVersion = "canvas-capabilities/v4"

// Registry 按节点类型保存描述。
type Registry struct{ descriptors map[string]Descriptor }

// NewRegistry 规范化并检查每条描述。类型、版本、标签和默认尺寸必须有效。生成模式不能重复。投影种类和字段必须同时有或同时无。接受和拒绝的输入不能冲突。能作为来源时必须有输入种类。可更新和补丁字段必须同时声明。类型不能重复。
// descriptors 逐条做上述检查，不通过或 Type 重复就返回错误，通过的经 cloneDescriptor 按 Type 放进登记表。
func NewRegistry(descriptors []Descriptor) (*Registry, error) {
	items := make(map[string]Descriptor, len(descriptors))
	generationModes := make(map[string]string, len(descriptors))
	for _, descriptor := range descriptors {
		descriptor.Type = normalizeType(descriptor.Type)
		descriptor.Version = strings.TrimSpace(descriptor.Version)
		descriptor.Label = strings.TrimSpace(descriptor.Label)
		descriptor.Purpose = strings.TrimSpace(descriptor.Purpose)
		descriptor.GoodFor = normalizeFields(descriptor.GoodFor)
		descriptor.NotIdealFor = normalizeFields(descriptor.NotIdealFor)
		descriptor.Tradeoffs = normalizeFields(descriptor.Tradeoffs)
		descriptor.Actions = normalizeFields(descriptor.Actions)
		if descriptor.Type == "" || descriptor.Version == "" || descriptor.Label == "" || descriptor.DefaultWidth <= 0 || descriptor.DefaultHeight <= 0 {
			return nil, fmt.Errorf("invalid canvas capability descriptor %q", descriptor.Type)
		}
		inputKind, err := normalizeInputKind(descriptor.InputKind)
		if err != nil {
			return nil, fmt.Errorf("capability %q: %w", descriptor.Type, err)
		}
		descriptor.InputKind = inputKind
		generationMode, err := normalizeGenerationMode(descriptor.GenerationMode)
		if err != nil {
			return nil, fmt.Errorf("capability %q: %w", descriptor.Type, err)
		}
		descriptor.GenerationMode = generationMode
		if generationMode != "" {
			if existing, exists := generationModes[generationMode]; exists {
				return nil, fmt.Errorf("generation mode %q is registered by both %q and %q", generationMode, existing, descriptor.Type)
			}
			generationModes[generationMode] = descriptor.Type
		}
		projectionKind, err := normalizeProjectionKind(descriptor.ProjectionKind)
		if err != nil {
			return nil, fmt.Errorf("capability %q: %w", descriptor.Type, err)
		}
		descriptor.ProjectionKind = projectionKind
		descriptor.ProjectionField = strings.TrimSpace(descriptor.ProjectionField)
		if (descriptor.ProjectionKind == "") != (descriptor.ProjectionField == "") {
			return nil, fmt.Errorf("capability %q must declare projection kind and field together", descriptor.Type)
		}
		if descriptor.ProjectionField != "" {
			if err := validatePatchPath(descriptor.ProjectionField); err != nil {
				return nil, fmt.Errorf("capability %q has invalid projection field: %w", descriptor.Type, err)
			}
		}
		if descriptor.Connection.MaxInputCount < 0 {
			return nil, fmt.Errorf("capability %q has invalid max input count", descriptor.Type)
		}
		accepted, err := normalizeConnectionKinds(descriptor.Connection.AcceptedInputKinds)
		if err != nil {
			return nil, fmt.Errorf("capability %q: %w", descriptor.Type, err)
		}
		rejected, err := normalizeConnectionKinds(descriptor.Connection.RejectedInputKinds)
		if err != nil {
			return nil, fmt.Errorf("capability %q: %w", descriptor.Type, err)
		}
		rejectedSet := make(map[string]struct{}, len(rejected))
		for _, kind := range rejected {
			rejectedSet[kind] = struct{}{}
		}
		for _, kind := range accepted {
			if _, conflict := rejectedSet[kind]; conflict {
				return nil, fmt.Errorf("capability %q accepts and rejects input kind %q", descriptor.Type, kind)
			}
		}
		if descriptor.Connection.CanSource && descriptor.InputKind == "" {
			return nil, fmt.Errorf("capability %q can source without an input kind", descriptor.Type)
		}
		descriptor.Connection.AcceptedInputKinds = accepted
		descriptor.Connection.RejectedInputKinds = rejected
		descriptor.PatchFields, err = normalizePatchFields(descriptor.PatchFields)
		if err != nil {
			return nil, fmt.Errorf("capability %q: %w", descriptor.Type, err)
		}
		if descriptor.CanUpdate != (len(descriptor.PatchFields) > 0) {
			return nil, fmt.Errorf("capability %q must declare both canUpdate and patch fields, or neither", descriptor.Type)
		}
		descriptor.SummaryFields = normalizeFields(descriptor.SummaryFields)
		descriptor.DetailFields = normalizeFields(descriptor.DetailFields)
		if _, exists := items[descriptor.Type]; exists {
			return nil, fmt.Errorf("duplicate canvas capability %q", descriptor.Type)
		}
		items[descriptor.Type] = cloneDescriptor(descriptor)
	}
	return &Registry{descriptors: items}, nil
}

// Resolve 按小写类型返回一份拷贝。没有或登记表为空时返回假。
// nodeType 经 normalizeType 后当作 descriptors 的键，命中再经 cloneDescriptor 返回。
func (r *Registry) Resolve(nodeType string) (Descriptor, bool) {
	if r == nil {
		return Descriptor{}, false
	}
	d, ok := r.descriptors[normalizeType(nodeType)]
	if !ok {
		return Descriptor{}, false
	}
	d = cloneDescriptor(d)
	return d, ok
}

// List 按类型排序返回每条描述的拷贝。登记表为空时返回 nil。
func (r *Registry) List() []Descriptor {
	if r == nil {
		return nil
	}
	out := make([]Descriptor, 0, len(r.descriptors))
	for _, d := range r.descriptors {
		out = append(out, cloneDescriptor(d))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}

// Types 返回已排序的节点类型。
func (r *Registry) Types() []string {
	items := r.List()
	out := make([]string, 0, len(items))
	for _, d := range items {
		out = append(out, d.Type)
	}
	return out
}

// GenerationModeNames 返回去重并排序的生成模式。
func (r *Registry) GenerationModeNames() []string {
	seen := map[string]struct{}{}
	for _, d := range r.List() {
		if d.GenerationMode != "" {
			seen[d.GenerationMode] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for mode := range seen {
		out = append(out, mode)
	}
	sort.Strings(out)
	return out
}

// SupportsGenerationMode 在该模式能解析到一条描述时为真。
// mode 传给 ResolveGenerationMode，只采用它的布尔结果。
func (r *Registry) SupportsGenerationMode(mode string) bool {
	_, ok := r.ResolveGenerationMode(mode)
	return ok
}

// ResolveGenerationMode 返回第一个生成模式相同的描述拷贝。模式为空时返回假。
// mode 经 normalizeType，空串时返回假，否则与每条描述的 GenerationMode 比较，命中则 cloneDescriptor 后返回。
func (r *Registry) ResolveGenerationMode(mode string) (Descriptor, bool) {
	mode = normalizeType(mode)
	if mode == "" || r == nil {
		return Descriptor{}, false
	}
	for _, descriptor := range r.descriptors {
		if descriptor.GenerationMode == mode {
			return cloneDescriptor(descriptor), true
		}
	}
	return Descriptor{}, false
}

// Hash 对排序后的描述做 JSON，再取 SHA256 的十六进制。
func (r *Registry) Hash() string {
	data, _ := json.Marshal(registryHashItems(r.List()))
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// normalizeFields 去掉空白和重复，保留第一次出现的顺序。
// fields 逐项 TrimSpace，空串和已出现过的丢掉，其余按首次出现的顺序返回。
func normalizeFields(fields []string) []string {
	seen := make(map[string]struct{}, len(fields))
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if _, exists := seen[field]; exists {
			continue
		}
		seen[field] = struct{}{}
		out = append(out, field)
	}
	return out
}

// cloneDescriptor 拷贝切片和补丁字段，避免调用方改到登记表里的原件。
// descriptor 的 AcceptedInputKinds、RejectedInputKinds、GoodFor、NotIdealFor、Tradeoffs、Actions、SummaryFields、DetailFields 换成新切片，PatchFields 换成新 map。
func cloneDescriptor(descriptor Descriptor) Descriptor {
	descriptor.Connection.AcceptedInputKinds = append([]string(nil), descriptor.Connection.AcceptedInputKinds...)
	descriptor.Connection.RejectedInputKinds = append([]string(nil), descriptor.Connection.RejectedInputKinds...)
	descriptor.GoodFor = append([]string(nil), descriptor.GoodFor...)
	descriptor.NotIdealFor = append([]string(nil), descriptor.NotIdealFor...)
	descriptor.Tradeoffs = append([]string(nil), descriptor.Tradeoffs...)
	descriptor.Actions = append([]string(nil), descriptor.Actions...)
	descriptor.SummaryFields = append([]string(nil), descriptor.SummaryFields...)
	descriptor.DetailFields = append([]string(nil), descriptor.DetailFields...)
	patchFields := make(map[string]PatchField, len(descriptor.PatchFields))
	for key, field := range descriptor.PatchFields {
		patchFields[key] = field
	}
	descriptor.PatchFields = patchFields
	return descriptor
}

// hashDescriptor 是参与哈希的描述字段，补丁字段排成切片。
type hashDescriptor struct {
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
	Connection      hashConnectionPolicy
	CanUpdate       bool
	SummaryFields   []string
	DetailFields    []string
	ProjectionKind  string
	ProjectionField string
	PatchFields     []hashPatchField
}

// hashConnectionPolicy 是参与哈希的连线策略。
type hashConnectionPolicy struct {
	CanSource          bool
	CanTarget          bool
	CanReference       bool
	AcceptedInputKinds []string
	RejectedInputKinds []string
	MaxInputCount      int
}

// hashPatchField 是参与哈希的补丁字段，不含数值上限。
type hashPatchField struct {
	Key         string
	Path        string
	Kind        string
	Label       string
	Order       int
	Description string
	MaxRunes    int
}

// registryHashItems 把补丁字段按键排序后收成哈希条目。
// descriptors 逐条把各字段抄进 hashDescriptor，PatchFields 按键排序后收成 hashPatchField。
func registryHashItems(descriptors []Descriptor) []hashDescriptor {
	items := make([]hashDescriptor, 0, len(descriptors))
	for _, descriptor := range descriptors {
		keys := make([]string, 0, len(descriptor.PatchFields))
		for key := range descriptor.PatchFields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		patchFields := make([]hashPatchField, 0, len(keys))
		for _, key := range keys {
			field := descriptor.PatchFields[key]
			patchFields = append(patchFields, hashPatchField{Key: key, Path: field.Path, Kind: field.Kind, Label: field.Label, Order: field.Order, Description: field.Description, MaxRunes: field.MaxRunes})
		}
		items = append(items, hashDescriptor{
			Type: descriptor.Type, Version: descriptor.Version, Label: descriptor.Label,
			Purpose: descriptor.Purpose, GoodFor: descriptor.GoodFor, NotIdealFor: descriptor.NotIdealFor,
			Tradeoffs: descriptor.Tradeoffs, Actions: descriptor.Actions,
			DefaultWidth: descriptor.DefaultWidth, DefaultHeight: descriptor.DefaultHeight,
			InputKind: descriptor.InputKind, GenerationMode: descriptor.GenerationMode,
			ProjectionKind: descriptor.ProjectionKind, ProjectionField: descriptor.ProjectionField,
			Connection: hashConnectionPolicy{CanSource: descriptor.Connection.CanSource, CanTarget: descriptor.Connection.CanTarget, CanReference: descriptor.Connection.CanReference, AcceptedInputKinds: descriptor.Connection.AcceptedInputKinds, RejectedInputKinds: descriptor.Connection.RejectedInputKinds, MaxInputCount: descriptor.Connection.MaxInputCount},
			CanUpdate:  descriptor.CanUpdate, SummaryFields: descriptor.SummaryFields, DetailFields: descriptor.DetailFields, PatchFields: patchFields,
		})
	}
	return items
}
