// buddy_diag_test.go 锁住 Buddy 和外部 agent 的诊断日志：状态只在变化时写、没装的项第一次不写；操作日志只列改了哪些字段和密钥名；
// 协议帧只在调试分类 acp 打开时写，先整帧打码再截断；Doctor 的 Buddy 一节不替人开始检测，整份 Doctor 里也没有密钥值。

package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/buddy"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/mcpserve"
)

// diagSecret 不像任何通用的密钥格式，只有按值打码才抹得掉。
const diagSecret = "hunter2-very-secret-value"

// diagCapture 把进程默认日志换成临时目录里的一份，测试结束换回去。
func diagCapture(t *testing.T) *diaglog.Logger {
	t.Helper()
	l, err := diaglog.Open(diaglog.Options{Dir: t.TempDir(), Process: diaglog.ProcessCLI, Version: "test", Level: slog.LevelInfo, Console: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	prev := diaglog.Default()
	diaglog.SetDefault(l)
	t.Cleanup(func() {
		diaglog.SetDefault(prev)
		_ = l.Close()
	})
	return l
}

// diagLines 是日志里 msg 为 msg 的记录。
func diagLines(t *testing.T, l *diaglog.Logger, msg string) []diaglog.Record {
	t.Helper()
	recs, err := l.Tail(500, diaglog.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	var out []diaglog.Record
	for _, r := range recs {
		if r.Msg == msg {
			out = append(out, r)
		}
	}
	return out
}

// isWarn 判断记录是不是警告。
func isWarn(r diaglog.Record) bool { return strings.HasPrefix(strings.ToUpper(r.Level), "WARN") }

// TestNoteBuddyLogsChanges 锁住状态日志只在变化时写：没装的项第一次出现不写，同状态的刷新不重复，出错是警告且详情打过码，
// 第一次见到的已装项不带 from。
func TestNoteBuddyLogsChanges(t *testing.T) {
	l := diagCapture(t)
	a := &App{}
	for _, v := range []buddy.View{
		{ID: "codex", State: buddy.StateNotInstalled},
		{ID: "codex", State: buddy.StateInstalling},
		{ID: "codex", State: buddy.StateInstalling},
		{ID: "codex", State: buddy.StateError, Detail: "exit 1: OPENAI_API_KEY=" + diagSecret},
		{ID: "claude", State: buddy.StateOnline},
	} {
		a.noteBuddy(v)
	}
	got := diagLines(t, l, "buddy state")
	if len(got) != 3 {
		t.Fatalf("lines %+v", got)
	}
	if got[0].Component != "buddy" || !strings.Contains(got[0].Raw, `"state":"installing"`) || !strings.Contains(got[0].Raw, `"from":"not_installed"`) {
		t.Fatalf("install line %s", got[0].Raw)
	}
	if !isWarn(got[1]) || strings.Contains(got[1].Raw, "hunter2") || !strings.Contains(got[1].Raw, `"from":"installing"`) {
		t.Fatalf("error line %s", got[1].Raw)
	}
	if isWarn(got[2]) || !strings.Contains(got[2].Raw, `"agent":"claude"`) || strings.Contains(got[2].Raw, `"from"`) {
		t.Fatalf("first sighting %s", got[2].Raw)
	}
}

// TestBuddyActionLogsFieldNames 锁住配置操作只记改了哪些字段和密钥名，不记值；失败是警告，错误文本打过码；
// 不针对某一项的操作（清缓存）不写 agent。
func TestBuddyActionLogsFieldNames(t *testing.T) {
	l := diagCapture(t)
	on, model := true, "model-from-test"
	p := buddy.ConfigPatch{Enabled: &on, Model: &model, Secrets: map[string]string{"Z_TOKEN": diagSecret, "A_API_KEY": diagSecret}}
	if got, want := patchFields(p), []string{"enabled", "model", "secret:A_API_KEY", "secret:Z_TOKEN"}; !slices.Equal(got, want) {
		t.Fatalf("fields %v", got)
	}
	logBuddyAction("configure", "fake", nil, "fields", patchFields(p))
	logBuddyAction("login", "fake", fmt.Errorf("auth failed: Bearer %s", diagSecret), "method", "api-key")
	ok := diagLines(t, l, "buddy action")
	if len(ok) != 1 || ok[0].Component != "buddy" || !strings.Contains(ok[0].Raw, "secret:A_API_KEY") ||
		strings.Contains(ok[0].Raw, "hunter2") || strings.Contains(ok[0].Raw, "model-from-test") {
		t.Fatalf("configure %+v", ok)
	}
	failed := diagLines(t, l, "buddy action failed")
	if len(failed) != 1 || !isWarn(failed[0]) || !strings.Contains(failed[0].Raw, `"action":"login"`) || strings.Contains(failed[0].Raw, "hunter2") {
		t.Fatalf("login %+v", failed)
	}
	logBuddyAction("cache.clear", "", nil, "freed", int64(1<<20))
	var cleared []diaglog.Record
	for _, r := range diagLines(t, l, "buddy action") {
		if strings.Contains(r.Raw, `"action":"cache.clear"`) {
			cleared = append(cleared, r)
		}
	}
	if len(cleared) != 1 || strings.Contains(cleared[0].Raw, `"agent"`) || !strings.Contains(cleared[0].Raw, `"freed":1048576`) {
		t.Fatalf("cache.clear %+v", cleared)
	}
}

// TestACPFramesNeedDebugCategory 锁住协议帧只在调试分类 acp 打开时写：写出的帧带 agent 和方向，启动环境里的密钥值被抹掉，
// session/new 里交给桥的控制面 token（环境变量和请求头两种）也被抹掉；连接的坏行记成警告，同样打码；调用方自己给的 Frame 不被换掉。
func TestACPFramesNeedDebugCategory(t *testing.T) {
	l := diagCapture(t)
	opts := withACPDiag(acp.Options{Name: "fake", Env: []string{"PATH=/usr/bin", "FAKE_API_KEY=" + diagSecret}})
	frame := []byte(`{"jsonrpc":"2.0","method":"session/prompt","params":{"text":"` + diagSecret + `"}}`)
	opts.Frame("send", frame)
	if got := diagLines(t, l, "acp frame"); len(got) != 0 {
		t.Fatalf("logged with the category off: %+v", got)
	}
	l.Apply(diaglog.Options{DebugCats: []string{acpDebugCat}})
	opts.Frame("recv", frame)
	got := diagLines(t, l, "acp frame")
	if len(got) != 1 || got[0].Component != "acp" || got[0].Cat != acpDebugCat ||
		!strings.Contains(got[0].Raw, `"dir":"recv"`) || !strings.Contains(got[0].Raw, `"agent":"fake"`) {
		t.Fatalf("frames %+v", got)
	}
	if strings.Contains(got[0].Raw, "hunter2") || !strings.Contains(got[0].Raw, "session/prompt") {
		t.Fatalf("frame %s", got[0].Raw)
	}
	token := mcpserve.NewToken()
	params, _ := json.Marshal(acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: []acp.McpServer{
		{Name: "yoyo", Command: "yoyo", Args: []string{}, Env: []acp.EnvVariable{{Name: "YOYO_MCP_BRIDGE", Value: "1"}, {Name: "YOYO_MCP_TOKEN", Value: token}}},
		{Type: "http", Name: "yoyo", URL: "http://127.0.0.1:4321/mcp", Headers: []acp.HTTPHeader{{Name: "Authorization", Value: "Bearer " + token}}},
	}})
	opts.Frame("send", []byte(`{"jsonrpc":"2.0","id":2,"method":"session/new","params":`+string(params)+`}`))
	if got = diagLines(t, l, "acp frame"); len(got) != 2 || strings.Contains(got[1].Raw, token) || !strings.Contains(got[1].Raw, "YOYO_MCP_BRIDGE") {
		t.Fatalf("session/new frame %+v", got)
	}
	opts.Log("acp: bad line " + diagSecret)
	if warn := diagLines(t, l, "acp connection"); len(warn) != 1 || !isWarn(warn[0]) || strings.Contains(warn[0].Raw, "hunter2") {
		t.Fatalf("connection %+v", warn)
	}
	called := false
	kept := withACPDiag(acp.Options{Frame: func(string, []byte) { called = true }})
	kept.Frame("send", frame)
	if !called {
		t.Fatal("the caller's Frame was replaced")
	}
}

// TestFrameTextRedactsBeforeCut 锁住帧先整帧打码再截断：跨在 2 KB 边上的密钥不会剩下半截；截断不切开字符，并注明原来的字节数。
func TestFrameTextRedactsBeforeCut(t *testing.T) {
	line := []byte(`{"x":"` + strings.Repeat("a", acpFrameMax-10) + diagSecret + `"}`)
	got := frameText(line, []string{diagSecret})
	if strings.Contains(got, "hunt") || !strings.HasSuffix(got, fmt.Sprintf("…(%d bytes)", len(line))) {
		t.Fatalf("tail %q", got[len(got)-60:])
	}
	if short := frameText([]byte(`{"k":"`+diagSecret+`"}`), []string{diagSecret}); strings.Contains(short, "hunter2") || strings.Contains(short, "…") {
		t.Fatalf("short %q", short)
	}
	wide := []byte(strings.Repeat("界", acpFrameMax))
	body, ok := strings.CutSuffix(frameText(wide, nil), fmt.Sprintf("…(%d bytes)", len(wide)))
	if !ok || len(body) > acpFrameMax || len(body) < acpFrameMax-3 || !utf8.ValidString(body) {
		t.Fatalf("cut to %d bytes, valid %v", len(body), utf8.ValidString(body))
	}
}

// TestEnvSecretsOnlySecretLike 锁住打码用的值只取名字像密钥、值不为空的变量。
func TestEnvSecretsOnlySecretLike(t *testing.T) {
	got := envSecrets([]string{"PATH=/usr/bin", "OPENAI_API_KEY=sk-1", "GH_TOKEN=t-2", "EMPTY_SECRET=", "HOME=/home/me", "NOEQUALS", "=x"})
	if !slices.Equal(got, []string{"sk-1", "t-2"}) {
		t.Fatalf("%v", got)
	}
}

// TestDoctorBuddyBlock 锁住 Doctor 的 Buddy 一节：没人打开过 Buddy 时只说没开始，也不会因此开始；还没登记工单时控制面不在听；
// 存过密钥、agent 在线以后，摘要列出它，整份 Doctor 里也没有密钥值。
func TestDoctorBuddyBlock(t *testing.T) {
	a := openBuddyApp(t, t.TempDir(), &fakeAgents{})
	defer a.Close()
	block, _ := a.Doctor()["buddy"].(map[string]any)
	if block["enabled"] != true {
		t.Fatalf("block %v", block)
	}
	if _, ok := block["frames"].(bool); !ok {
		t.Fatalf("frames %v", block["frames"])
	}
	if s, _ := block["summary"].(buddy.Summary); s.Started || a.Buddies().Summary().Started {
		t.Fatal("Doctor started Buddy")
	}
	if ctl, _ := block["control"].(map[string]any); ctl["listening"] != false || ctl["tickets"] != 0 {
		t.Fatalf("control %v", ctl)
	}
	if _, err := a.BuddyAdd(buddy.CustomAgent{ID: "fake", Name: "Fake", Command: fakeCommand(t)}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.BuddyConfigure("fake", buddy.ConfigPatch{Secrets: map[string]string{"FAKE_API_KEY": diagSecret}}); err != nil {
		t.Fatal(err)
	}
	probeOnline(t, a.Buddies(), "fake")
	raw, err := json.Marshal(a.Doctor())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "hunter2") {
		t.Fatal("a secret value reached Doctor")
	}
	var back struct {
		Buddy struct {
			Summary buddy.Summary `json:"summary"`
		} `json:"buddy"`
	}
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	listed := slices.ContainsFunc(back.Buddy.Summary.Agents, func(it buddy.SummaryItem) bool {
		return it.ID == "fake" && it.State == buddy.StateOnline && it.Install == "custom"
	})
	if !back.Buddy.Summary.Started || !listed {
		t.Fatalf("summary %+v", back.Buddy.Summary)
	}
}
