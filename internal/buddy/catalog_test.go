// catalog_test.go 锁住目录合并顺序、精选项从注册表补全、平台键和二进制回落规则，以及内嵌快照本身合格。

package buddy

import (
	"testing"
)

// TestEmbeddedSnapshotsParse 锁住内嵌的精选表和注册表快照能解析，精选项都能在快照里找到。
func TestEmbeddedSnapshotsParse(t *testing.T) {
	curated, err := parseCurated(curatedJSON)
	if err != nil || len(curated) < 14 {
		t.Fatalf("curated %d %v", len(curated), err)
	}
	reg, err := parseRegistry(registryJSON)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, a := range reg.Agents {
		ids[a.ID] = true
	}
	for _, c := range curated {
		if !ids[c.Registry] {
			t.Errorf("curated %s points at missing registry id %s", c.ID, c.Registry)
		}
	}
}

// TestCatalogOrder 锁住 yoyo 第一、精选按 order、自定义随后、其余按名字进「更多」，被精选引用的注册表项不再重复出现。
func TestCatalogOrder(t *testing.T) {
	curated := []Curated{
		{ID: "b", Registry: "reg-b", Name: "Bee", Order: 2},
		{ID: "a", Registry: "reg-a", Name: "Ay", Order: 1, Vendor: "Acme"},
	}
	reg := Registry{Version: "1.0.0", Agents: []RegistryAgent{
		{ID: "zeta", Name: "Zeta"},
		{ID: "reg-a", Name: "Registry A", Version: "1.2.3", Authors: []string{"Someone"}, Website: "https://a.example", Distribution: Distribution{NPX: &PackageTarget{Package: "a@1.2.3"}}},
		{ID: "alpha", Name: "alpha", Distribution: Distribution{Binary: map[string]BinaryTarget{"linux-x86_64": {Archive: "https://x/a.tar.gz", Cmd: "./bin/alpha-acp"}}}},
		{ID: "reg-b", Name: "Registry B"},
	}}
	got := buildCatalog(curated, reg, []CustomAgent{{ID: "mine", Name: "Mine", Command: "mine"}})
	var order []string
	for _, e := range got {
		order = append(order, e.ID)
	}
	want := []string{YoyoID, "a", "b", "mine", "alpha", "zeta"}
	if len(order) != len(want) {
		t.Fatalf("order %v", order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order %v", order)
		}
	}
	a := got[1]
	if a.Name != "Ay" || a.Vendor != "Acme" || a.Version != "1.2.3" || a.Docs != "https://a.example" || a.Dist.NPX == nil || !a.Featured {
		t.Fatalf("curated entry %+v", a)
	}
	if alpha := got[4]; len(alpha.ACPBins) != 1 || alpha.ACPBins[0] != "alpha-acp" || alpha.Featured {
		t.Fatalf("registry entry %+v", alpha)
	}
}

// TestRegistryValidation 锁住主版本不是 1、没有 agents 或缺编号的注册表被拒。
func TestRegistryValidation(t *testing.T) {
	for _, raw := range []string{
		`{"version":"2.0.0","agents":[{"id":"x","name":"X"}]}`,
		`{"version":"1.0.0","agents":[]}`,
		`{"version":"1.0.0","agents":[{"name":"X"}]}`,
		`not json`,
	} {
		if _, err := parseRegistry([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

// TestPlatformKeys 锁住平台键的写法、musl 后缀和 windows-aarch64 回落到 x86_64。
func TestPlatformKeys(t *testing.T) {
	cases := map[string]Platform{
		"windows-x86_64":     {GOOS: "windows", GOARCH: "amd64"},
		"darwin-aarch64":     {GOOS: "darwin", GOARCH: "arm64"},
		"linux-x86_64-musl":  {GOOS: "linux", GOARCH: "amd64", Musl: true},
		"linux-aarch64":      {GOOS: "linux", GOARCH: "arm64"},
		"windows-aarch64":    {GOOS: "windows", GOARCH: "arm64"},
		"darwin-x86_64":      {GOOS: "darwin", GOARCH: "amd64"},
		"linux-aarch64-musl": {GOOS: "linux", GOARCH: "arm64", Musl: true},
	}
	for want, p := range cases {
		if got := p.Key(); got != want {
			t.Errorf("%+v => %s", p, got)
		}
	}
	bin := map[string]BinaryTarget{"windows-x86_64": {Archive: "https://x/a.zip", Cmd: "a.exe"}}
	if _, key, ok := pickTarget(bin, Platform{GOOS: "windows", GOARCH: "arm64"}); !ok || key != "windows-x86_64" {
		t.Fatalf("arm64 fallback %s %v", key, ok)
	}
	if _, _, ok := pickTarget(bin, Platform{GOOS: "linux", GOARCH: "amd64"}); ok {
		t.Fatal("linux should not match a windows binary")
	}
	kiro := Entry{Unsupported: []string{"windows"}, Dist: Distribution{NPX: &PackageTarget{Package: "k"}}}
	if installable(kiro, Platform{GOOS: "windows", GOARCH: "amd64"}) {
		t.Fatal("declared unsupported platform is installable")
	}
}

// TestSplitPackage 锁住作用域包和两种版本写法的拆分。
func TestSplitPackage(t *testing.T) {
	cases := map[string][2]string{
		"@google/gemini-cli@0.67.0": {"@google/gemini-cli", "0.67.0"},
		"cline@3.0.74":              {"cline", "3.0.74"},
		"fast-agent-acp==0.10.1":    {"fast-agent-acp", "0.10.1"},
		"@scope/pkg":                {"@scope/pkg", ""},
	}
	for spec, want := range cases {
		name, version := splitPackage(spec)
		if name != want[0] || version != want[1] {
			t.Errorf("%s => %s %s", spec, name, version)
		}
	}
}
