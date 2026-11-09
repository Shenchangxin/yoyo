// catalog.go 拥有 agent 目录：内嵌的精选表和注册表快照，以及两者和自定义 agent 的合并。
// 注册表的下载与缓存在 registry.go；这里只解析和合并，不碰磁盘和网络。

package buddy

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
)

//go:embed curated.json
var curatedJSON []byte

//go:embed registry.json
var registryJSON []byte

// Localized 是一句中英文。
type Localized struct {
	Zh string `json:"zh,omitempty"`
	En string `json:"en,omitempty"`
}

// Pick 按界面语言取一句：zh 开头取中文，其余取英文；缺哪个就用另一个。
func (l Localized) Pick(locale string) string {
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		if l.Zh != "" {
			return l.Zh
		}
		return l.En
	}
	if l.En != "" {
		return l.En
	}
	return l.Zh
}

// SecretSpec 是这个 agent 认的一把密钥。Provider 非空时可以从 yoyo 同类提供商复用。
type SecretSpec struct {
	Name     string `json:"name"`
	Label    string `json:"label,omitempty"`
	Provider string `json:"provider,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// LoginSpec 是精选表里的兜底登录命令，对方没声明登录方式时在终端里跑它。
type LoginSpec struct {
	Label string   `json:"label"`
	Cmd   []string `json:"cmd"`
}

// SystemSpec 是装到系统的办法：npm 包名，或各平台的官方安装命令。
type SystemSpec struct {
	NPM     string `json:"npm,omitempty"`
	Windows string `json:"windows,omitempty"`
	Darwin  string `json:"darwin,omitempty"`
	Linux   string `json:"linux,omitempty"`
}

// Command 返回 goos 上的官方安装命令。
func (s SystemSpec) Command(goos string) string {
	switch goos {
	case "windows":
		return s.Windows
	case "darwin":
		return s.Darwin
	case "linux":
		return s.Linux
	}
	return ""
}

// Curated 是精选表的一项，覆盖在注册表的同名项上。
type Curated struct {
	ID          string       `json:"id"`
	Registry    string       `json:"registry"`
	Name        string       `json:"name"`
	Vendor      string       `json:"vendor"`
	Order       int          `json:"order"`
	Aliases     []string     `json:"aliases,omitempty"`
	ACPBins     []string     `json:"acpBins,omitempty"`
	CLIBins     []string     `json:"cliBins,omitempty"`
	Strengths   Localized    `json:"strengths"`
	Secrets     []SecretSpec `json:"secrets,omitempty"`
	Login       []LoginSpec  `json:"login,omitempty"`
	System      SystemSpec   `json:"system"`
	Unsupported []string     `json:"unsupported,omitempty"`
	Docs        string       `json:"docs,omitempty"`
}

// Registry 是 ACP 注册表。
type Registry struct {
	Version string          `json:"version"`
	Agents  []RegistryAgent `json:"agents"`
}

// RegistryAgent 是注册表里的一个 agent。
type RegistryAgent struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Description  string       `json:"description,omitempty"`
	Repository   string       `json:"repository,omitempty"`
	Website      string       `json:"website,omitempty"`
	Icon         string       `json:"icon,omitempty"`
	License      string       `json:"license,omitempty"`
	Authors      []string     `json:"authors,omitempty"`
	Distribution Distribution `json:"distribution"`
}

// Distribution 是注册表给的分发方式，可以同时有几种。
type Distribution struct {
	Binary map[string]BinaryTarget `json:"binary,omitempty"`
	NPX    *PackageTarget          `json:"npx,omitempty"`
	UVX    *PackageTarget          `json:"uvx,omitempty"`
}

// BinaryTarget 是某个平台的二进制归档。SHA256 为空表示对方没给校验和。
type BinaryTarget struct {
	Archive string            `json:"archive"`
	Cmd     string            `json:"cmd"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	SHA256  string            `json:"sha256,omitempty"`
}

// PackageTarget 是 npx 或 uvx 的包，Package 里带版本。
type PackageTarget struct {
	Package string            `json:"package"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// Entry 是合并后的一项。Command、Args、Env 只对自定义 agent 有意义。
type Entry struct {
	ID          string
	RegistryID  string
	Name        string
	Vendor      string
	Description string
	Version     string
	Docs        string
	Icon        string
	Order       int
	Featured    bool
	Builtin     bool
	Custom      bool
	Aliases     []string
	ACPBins     []string
	CLIBins     []string
	Strengths   Localized
	Secrets     []SecretSpec
	Login       []LoginSpec
	System      SystemSpec
	Unsupported []string
	Dist        Distribution
	Command     string
	Args        []string
	Env         map[string]string
}

// errRegistry 是注册表内容不合格。
var errRegistry = errors.New("buddy: bad registry")

// parseCurated 解析精选表。
func parseCurated(raw []byte) ([]Curated, error) {
	var out []Curated
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// parseRegistry 解析并校验注册表：主版本必须是 1，agents 不能为空，每项要有编号和名字。
func parseRegistry(raw []byte) (Registry, error) {
	var reg Registry
	if err := json.Unmarshal(raw, &reg); err != nil {
		return Registry{}, fmt.Errorf("%w: %v", errRegistry, err)
	}
	if !strings.HasPrefix(reg.Version, "1.") && reg.Version != "1" {
		return Registry{}, fmt.Errorf("%w: version %q", errRegistry, reg.Version)
	}
	if len(reg.Agents) == 0 {
		return Registry{}, fmt.Errorf("%w: no agents", errRegistry)
	}
	for _, a := range reg.Agents {
		if strings.TrimSpace(a.ID) == "" || strings.TrimSpace(a.Name) == "" {
			return Registry{}, fmt.Errorf("%w: agent without id or name", errRegistry)
		}
	}
	return reg, nil
}

// builtinEntry 是内置的 yoyo。
func builtinEntry() Entry {
	return Entry{
		ID: YoyoID, Name: "yoyo", Vendor: "yoyo", Builtin: true, Featured: true,
		Strengths: Localized{Zh: "本机工具最全，读写工作区、浏览器和电脑操作", En: "the full local toolset: workspace, browser and computer use"},
	}
}

// buildCatalog 合并出完整目录：yoyo 第一，精选项按 order，自定义项按配置顺序，注册表里其余项按名字放进「更多」。
// 精选项在注册表里找不到时仍然列出，只是没有分发方式。
func buildCatalog(curated []Curated, reg Registry, custom []CustomAgent) []Entry {
	byID := map[string]RegistryAgent{}
	for _, a := range reg.Agents {
		byID[a.ID] = a
	}
	used := map[string]bool{}
	out := []Entry{builtinEntry()}
	sorted := append([]Curated(nil), curated...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Order < sorted[j].Order })
	taken := map[string]bool{YoyoID: true}
	for _, c := range sorted {
		if c.ID == "" || taken[c.ID] {
			continue
		}
		taken[c.ID] = true
		e := Entry{
			ID: c.ID, RegistryID: c.Registry, Name: c.Name, Vendor: c.Vendor, Order: c.Order, Featured: true,
			Aliases: c.Aliases, ACPBins: c.ACPBins, CLIBins: c.CLIBins, Strengths: c.Strengths, Secrets: c.Secrets,
			Login: c.Login, System: c.System, Unsupported: c.Unsupported, Docs: c.Docs,
		}
		if a, ok := byID[c.Registry]; ok {
			used[c.Registry] = true
			fillFromRegistry(&e, a)
		}
		out = append(out, e)
	}
	for i, c := range custom {
		id := strings.TrimSpace(c.ID)
		if id == "" || taken[id] {
			continue
		}
		taken[id] = true
		out = append(out, Entry{
			ID: id, Name: firstNonEmpty(c.Name, id), Custom: true, Order: 1000 + i,
			Command: c.Command, Args: c.Args, Env: c.Env,
		})
	}
	var rest []Entry
	for _, a := range reg.Agents {
		if used[a.ID] || taken[a.ID] {
			continue
		}
		taken[a.ID] = true
		e := Entry{ID: a.ID, RegistryID: a.ID, Name: a.Name}
		fillFromRegistry(&e, a)
		e.ACPBins = binaryNames(a.Distribution)
		rest = append(rest, e)
	}
	sort.SliceStable(rest, func(i, j int) bool { return strings.ToLower(rest[i].Name) < strings.ToLower(rest[j].Name) })
	return append(out, rest...)
}

// fillFromRegistry 把注册表里的描述、版本、图标、文档和分发方式填进 e。精选表已有的名字和厂商不覆盖。
func fillFromRegistry(e *Entry, a RegistryAgent) {
	e.Description = a.Description
	e.Version = a.Version
	e.Icon = a.Icon
	e.Dist = a.Distribution
	if e.Name == "" {
		e.Name = a.Name
	}
	if e.Vendor == "" && len(a.Authors) > 0 {
		e.Vendor = a.Authors[0]
	}
	if e.Docs == "" {
		e.Docs = firstNonEmpty(a.Website, a.Repository)
	}
}

// binaryNames 从二进制分发的 cmd 里取出可执行文件名，系统检测时按它在 PATH 上找。
func binaryNames(d Distribution) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range d.Binary {
		name := strings.ReplaceAll(t.Cmd, `\`, "/")
		name = path.Base(name)
		for _, ext := range []string{".exe", ".cmd", ".bat"} {
			name = strings.TrimSuffix(name, ext)
		}
		if name == "" || name == "." || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// splitPackage 把 pkg@1.2.3 或 pkg==1.2.3 拆成包名和版本；作用域包开头的 @ 不算。
func splitPackage(spec string) (string, string) {
	if i := strings.Index(spec, "=="); i > 0 {
		return spec[:i], spec[i+2:]
	}
	if i := strings.LastIndex(spec, "@"); i > 0 {
		return spec[:i], spec[i+1:]
	}
	return spec, ""
}

// firstNonEmpty 返回第一个非空串。
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
