// 锁住对象库和指针：信封能按类型读回；文件裸字节入库后再次写入返回同一哈希；active 可以改指向；SKILL.md 保留名字、嵌套 metadata 和 https 图标，并拒绝非 https；关掉模型自行调用后不再隐式载入；相同正文的剧本条目合并计数；快照能按哈希读回。

package artifact

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCASRoundTrip 确认 Put 写入的提示词信封能按 PromptFragment 读回。
func TestCASRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "cas"))
	h, err := s.Put(KindPromptFragment, "boot", PromptFragment{ID: "boot", Slot: "bootstrap", Text: "start small"})
	if err != nil {
		t.Fatal(err)
	}
	got, env, err := Decode[PromptFragment](s, h)
	if err != nil {
		t.Fatal(err)
	}
	if env.Kind != KindPromptFragment || got.Text != "start small" {
		t.Fatalf("%+v %+v", env, got)
	}
}

// TestPutFileCopiesBytes 确认拷进库的是源文件裸字节，并且第二次 PutFile 返回同一哈希。
func TestPutFileCopiesBytes(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "clip.bin")
	if err := os.WriteFile(src, []byte("frame-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(filepath.Join(dir, "cas"))
	h, err := s.PutFile(src)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := s.GetRaw(h)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "frame-bytes" {
		t.Fatalf("payload %q", raw)
	}
	again, err := s.PutFile(src)
	if err != nil || again != h {
		t.Fatalf("second put %s %v", again, err)
	}
}

// TestRefsCheckout 确认 active 被改写后读到新哈希，并能对 12 位哈希做 Archive。
func TestRefsCheckout(t *testing.T) {
	dir := t.TempDir()
	r := NewRefs(filepath.Join(dir, "refs"))
	if err := r.Set(RefActive, "abc"); err != nil {
		t.Fatal(err)
	}
	if err := r.Set(RefCanary, "def"); err != nil {
		t.Fatal(err)
	}
	if err := r.Set(RefActive, "def"); err != nil {
		t.Fatal(err)
	}
	got, err := r.Get(RefActive)
	if err != nil || got != "def" {
		t.Fatalf("got %q %v", got, err)
	}
	if err := r.Archive("defdefdefdef"); err != nil {
		t.Fatal(err)
	}
}

// TestSkillParse 确认前置信息里的名字能被解析出来。
func TestSkillParse(t *testing.T) {
	raw := "---\nname: verify-artifact\ndescription: Always check required output files exist.\n---\n\nRead the task, then confirm the artifact path.\n"
	sk, err := ParseSkillMD(raw, "mem")
	if err != nil {
		t.Fatal(err)
	}
	if sk.Name != "verify-artifact" {
		t.Fatal(sk.Name)
	}
}

// TestSkillParseNestedMetadataAndIcon 确认中文显示名、https 图标和嵌套 metadata 能留下，非 https 图标被丢掉。
func TestSkillParseNestedMetadataAndIcon(t *testing.T) {
	raw := "---\nname: maishou\ndescription: Compare prices across shops.\ndisplay_name: 买手\nicon: https://cdn.example.com/skill.png\nmetadata:\n  openclaw:\n    emoji: shop\nallowed-tools: Read, Bash\n---\n\nRun `uv run scripts/main.py search`.\n"
	sk, err := ParseSkillMD(raw, filepath.Join("skills", "maishou", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if sk.Icon != "https://cdn.example.com/skill.png" || sk.DisplayName != "买手" {
		t.Fatalf("%+v", sk)
	}
	if sk.Dir == "" || sk.Metadata["openclaw.emoji"] != "shop" {
		t.Fatalf("dir/meta %+v", sk)
	}
	if SafeIconURL("javascript:alert(1)") != "" || SafeIconURL("http://cdn.example.com/x.png") != "" {
		t.Fatal("unsafe icon")
	}
}

// TestSkillAllowsImplicitFromFrontmatter 确认 disable-model-invocation 关掉隐式调用，缺省技能仍允许。
func TestSkillAllowsImplicitFromFrontmatter(t *testing.T) {
	raw := "---\nname: verify\ndescription: Run verify only when asked.\ndisable-model-invocation: true\nmetadata:\n  allow_implicit_invocation: false\n---\n\nManual only.\n"
	sk, err := ParseSkillMD(raw, "skills/verify/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if !sk.DisableModelInvocation || sk.AllowsImplicit() {
		t.Fatalf("%+v", sk)
	}
	ok, err := ParseSkillMD("---\nname: pdf\ndescription: Handle PDFs.\n---\n\nGo.\n", "skills/pdf/SKILL.md")
	if err != nil || !ok.AllowsImplicit() {
		t.Fatalf("%+v %v", ok, err)
	}
}

// TestPlaybookDeltaNoCollapse 确认不同正文会留下，相同正文则把 Helpful 加在一起。
func TestPlaybookDeltaNoCollapse(t *testing.T) {
	p := Playbook{ID: "main", Bullets: []PlaybookBullet{{ID: "1", Text: "create output early", Helpful: 2}}}
	p = p.ApplyDelta([]PlaybookBullet{{ID: "2", Text: "do not retry the same failing command"}}, nil)
	if len(p.Bullets) != 2 {
		t.Fatalf("len=%d", len(p.Bullets))
	}
	p = p.ApplyDelta([]PlaybookBullet{{ID: "3", Text: "create output early", Helpful: 1}}, nil)
	if len(p.Bullets) != 2 {
		t.Fatalf("dedup failed %d", len(p.Bullets))
	}
	if p.Bullets[0].Helpful != 3 {
		t.Fatalf("helpful=%d", p.Bullets[0].Helpful)
	}
}

// TestSnapshotGet 确认 PutSnapshot 写入的快照能按哈希读回模型指纹。
func TestSnapshotGet(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "cas"))
	h, err := s.PutSnapshot(HarnessSnapshot{ID: "h", ModelFingerprint: "m1", LoopPreset: "x"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetSnapshot(h)
	if err != nil {
		t.Fatal(err)
	}
	if got.ModelFingerprint != "m1" {
		t.Fatalf("%+v", got)
	}
	_ = os.MkdirAll(dir, 0o755)
}
