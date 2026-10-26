// 技能市场的目录、安装和来源判断。扫描不通过则不落盘。打开路径和链接也从这里转到 hostopen。

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/hostopen"
	"github.com/Shenchangxin/yoyo/internal/skillmarket"
)

// SkillMarket 从家目录 skill-market 加载指定 hub 和查询的目录。refresh 为真时允许重新拉取。
// query 写入 LoadOpts.Query。
func (a *App) SkillMarket(refresh bool, hub, query string) (skillmarket.Catalog, error) {
	dir := filepath.Join(a.Home.Root, "skill-market")
	return skillmarket.LoadHub(dir, skillmarket.LoadOpts{Hub: hub, Query: query, Refresh: refresh})
}

// InstallMarketSkill 按 slug 下载技能包。缺少 SKILL.md 或扫描不通过则不安装。正文提到 scripts/ 但包里没有时写入警告。元数据里的 mcp 或 mcp_url 会 StartMCPHTTP。
func (a *App) InstallMarketSkill(slug string) (map[string]any, error) {
	hub, remote := skillmarket.ParseInstallID(slug)
	local := skillmarket.InstallSlug(remote)
	if local == "" {
		return nil, fmt.Errorf("invalid skill slug")
	}
	pack, err := skillmarket.FetchFrom(hub, remote)
	if err != nil {
		return nil, err
	}
	files, warn := pack.Files, pack.Warn
	raw := packSkillMD(files)
	if len(raw) == 0 {
		return nil, fmt.Errorf("pack missing SKILL.md")
	}
	skParsed, scan := skillmarket.ScanSkillMD(raw, local)
	if !scan.OK {
		return map[string]any{"scan": scan, "ok": false}, fmt.Errorf("scan blocked: %s", strings.Join(scan.Reasons, "; "))
	}
	scan.Files = len(files)
	if skillmarket.MentionsScripts(skParsed.Body) && !packHasScripts(files) {
		if warn == "" {
			warn = "skill references scripts/ but the downloaded pack has none"
		}
	}
	info, _ := skillmarket.LookupHub(hub)
	rec := skillmarket.Record{
		Source:  hub,
		Catalog: info.Name,
		URL:     pack.URL,
		Remote:  remote,
	}
	if pack.Catalog != "" {
		rec.Catalog = pack.Catalog
	}
	dir, err := skillmarket.InstallRecord(a.Home.Skills(), local, scan, files, rec)
	if err != nil {
		return nil, err
	}
	mcpURL := ""
	if skParsed.Metadata != nil {
		mcpURL = strings.TrimSpace(skParsed.Metadata["mcp"])
		if mcpURL == "" {
			mcpURL = strings.TrimSpace(skParsed.Metadata["mcp_url"])
		}
	}
	mcpNote := ""
	if mcpURL != "" {
		if err := a.StartMCPHTTP(local, mcpURL); err != nil {
			mcpNote = err.Error()
		} else {
			mcpNote = "mcp started"
		}
	}
	sk := a.GetSkill("", scan.Name)
	if sk == nil {
		sk = a.GetSkill("", local)
	}
	return map[string]any{
		"ok":      true,
		"dir":     dir,
		"scan":    scan,
		"skill":   sk,
		"files":   skillmarketFiles(files),
		"warning": warn,
		"mcp":     mcpNote,
		"bundle":  skillBundleNote(skParsed.Metadata),
	}, nil
}

// UninstallMarketSkill 从家目录技能夹卸下该 slug。
func (a *App) UninstallMarketSkill(slug string) error {
	return skillmarket.Uninstall(a.Home.Skills(), slug)
}

// skillBundleNote 返回元数据里的 connectors 文本。元数据为空返回空串。
// meta 为 nil 时返回空串，否则取 connectors 并去掉首尾空白。
func skillBundleNote(meta map[string]string) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta["connectors"])
}

// packSkillMD 返回包内大小写不敏感的 SKILL.md 正文，没有则 nil。
// files 里按 Rel 找大小写不敏感的 SKILL.md，没有则返回 nil。
func packSkillMD(files []skillmarket.PackFile) []byte {
	for _, f := range files {
		if strings.EqualFold(f.Rel, "SKILL.md") {
			return f.Data
		}
	}
	return nil
}

// packHasScripts 在任一相对路径以 scripts/ 开头时返回真。
// files 的 Rel 换成正斜杠并转小写后，看是否以 scripts/ 开头。
func packHasScripts(files []skillmarket.PackFile) bool {
	for _, f := range files {
		rel := strings.ToLower(strings.ReplaceAll(f.Rel, "\\", "/"))
		if strings.HasPrefix(rel, "scripts/") {
			return true
		}
	}
	return false
}

// skillmarketFiles 只收集包文件的相对路径。
// files 只收集每项的 Rel。
func skillmarketFiles(files []skillmarket.PackFile) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Rel)
	}
	return out
}

// skillInstallID 从目录里的市场记录拼安装 id。有 Remote 和 Source 时用 ItemID。没有记录返回空串。
// dir 传给 skillmarket.ReadRecord。
func skillInstallID(dir string) string {
	rec, ok := skillmarket.ReadRecord(dir)
	if !ok {
		return ""
	}
	if rec.Remote != "" && rec.Source != "" {
		return skillmarket.ItemID(rec.Source, rec.Remote)
	}
	if rec.Slug != "" && rec.Source != "" && rec.Source != skillmarket.HubWorkBuddy {
		return skillmarket.ItemID(rec.Source, rec.Slug)
	}
	return rec.Slug
}

// skillOrigin 判断技能目录来自 bundled、pack、market、home、workspace 还是 cas。市场目录要有 .market.json。
// dir 先 Clean，再与 bundled、home 下的 packs 或 skills 比较。
func skillOrigin(dir, home, bundled string) string {
	dir = filepath.Clean(dir)
	if bundled != "" && pathUnder(bundled, dir) {
		return "bundled"
	}
	if home != "" && pathUnder(filepath.Join(home, "packs"), dir) {
		return "pack"
	}
	if home != "" && pathUnder(filepath.Join(home, "skills"), dir) {
		if _, err := os.Stat(filepath.Join(dir, ".market.json")); err == nil {
			return "market"
		}
		return "home"
	}
	if dir != "" && dir != "." {
		return "workspace"
	}
	return "cas"
}

// skillPackID 从家目录 packs 下的相对路径取出第一段包 id。不在该根下返回空串。
// dir 为空时返回空串，否则和 home 拼成 packs 根再取相对路径的第一段。
// home 为空时返回空串，否则和 dir 拼成 packs 根。
func skillPackID(dir, home string) string {
	if home == "" || dir == "" {
		return ""
	}
	root := filepath.Join(home, "packs")
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(dir))
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || strings.HasPrefix(rel, "..") {
		return ""
	}
	id, _, _ := strings.Cut(rel, "/")
	return id
}

// pathUnder 判断 path 是否等于 parent 或位于其下。相对路径以 .. 开头则否。
func pathUnder(parent, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel == "." || (rel != "" && !strings.HasPrefix(rel, ".."))
}

// OpenPath 用 hostopen 打开本地路径。
// path 传给 hostopen.Path。
func (a *App) OpenPath(path string) error {
	return hostopen.Path(path)
}

// OpenURL 用 hostopen 打开链接。
// raw 传给 hostopen.URL。
func (a *App) OpenURL(raw string) error {
	return hostopen.URL(raw)
}

// OpenInEditor 用 hostopen 在编辑器里打开路径。
// path 传给 hostopen.Editor。
func (a *App) OpenInEditor(path string) error {
	return hostopen.Editor(path)
}

// OpenWorkspaceTerminal 用 hostopen 在该目录打开终端。
// dir 传给 hostopen.Terminal。
func (a *App) OpenWorkspaceTerminal(dir string) error {
	return hostopen.Terminal(dir)
}
