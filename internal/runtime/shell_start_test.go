// 会挂住管道的 start 改写成 Start-Process -WindowStyle Hidden。这不是沙箱。聊天熔断阈值在 tools.go。

package runtime

import (
	"strings"
	"testing"
	"time"
)

// TestCmdStartWouldHang 确认 cmd //c start、裸 start /MIN 和 start 带标题算挂起，Start-Process、npm start、cmd /c echo 和 net start 不算。
func TestCmdStartWouldHang(t *testing.T) {
	hang := []string{
		`cd /c/Users/scx/Desktop/codex && cmd //c "start \"permkit\" /MIN permserver.exe -addr 127.0.0.1:8080 -data .\\data\\perm.json -seed" 2>&1 | head -3; sleep 4; powershell -NoProfile -Command "\$p = Get-Process -Name permserver -ErrorAction SilentlyContinue; if (\$p) { Write-Output ('ALIVE pid=' + \$p.Id) } else { Write-Output 'DEAD' }" 2>&1 | head -3`,
		`cmd /c start /MIN permserver.exe -addr 127.0.0.1:8080`,
		`cmd.exe //c start "" /B foo.exe`,
		`start /MIN permserver.exe`,
		`cd dest && start "title" /MIN app.exe`,
	}
	for _, cmd := range hang {
		if !cmdStartWouldHang(cmd) {
			t.Fatalf("want hang: %s", cmd)
		}
	}
	safe := []string{
		`Start-Process -WindowStyle Hidden -FilePath permserver.exe`,
		`npm start`,
		`cd web && npm start`,
		`go test ./...`,
		`cmd /c echo hello`,
		`powershell -NoProfile -Command "Get-Process"`,
		`net start wuauserv`,
	}
	for _, cmd := range safe {
		if cmdStartWouldHang(cmd) {
			t.Fatalf("false hang: %s", cmd)
		}
	}
}

// TestRewriteCmdStartUserCommand 确认改写保留 permserver.exe、地址、数据路径、sleep 4 和 $p，丢掉第一段 head、/MIN 和标题 permkit。
func TestRewriteCmdStartUserCommand(t *testing.T) {
	in := `cd /c/Users/scx/Desktop/codex && cmd //c "start \"permkit\" /MIN permserver.exe -addr 127.0.0.1:8080 -data .\\data\\perm.json -seed" 2>&1 | head -3; sleep 4; powershell -NoProfile -Command "\$p = Get-Process -Name permserver -ErrorAction SilentlyContinue; if (\$p) { Write-Output ('ALIVE pid=' + \$p.Id) } else { Write-Output 'DEAD' }" 2>&1 | head -3`
	got, ok := rewriteCmdStart(in)
	if !ok {
		t.Fatal("rewrite failed")
	}
	if strings.Contains(strings.ToLower(got), "cmd ") && strings.Contains(strings.ToLower(got), "start ") {
		t.Fatalf("still cmd start: %s", got)
	}
	if !strings.Contains(got, "Start-Process") {
		t.Fatalf("missing Start-Process: %s", got)
	}
	if !strings.Contains(got, "permserver.exe") {
		t.Fatalf("missing exe: %s", got)
	}
	if !strings.Contains(got, "127.0.0.1:8080") {
		t.Fatalf("missing addr: %s", got)
	}
	if !strings.Contains(got, `.\data\perm.json`) && !strings.Contains(got, `.\\data\\perm.json`) {
		t.Fatalf("missing data path: %s", got)
	}
	if strings.Contains(got, "| head -3;") || strings.Contains(got, "| head -3 ;") {
		t.Fatalf("first head pipe should be dropped: %s", got)
	}
	if !strings.Contains(got, "sleep 4") {
		t.Fatalf("lost probe: %s", got)
	}
	if !strings.Contains(got, `\$p`) {
		t.Fatalf("must keep bash-escaped $: %s", got)
	}
	if strings.Contains(got, "/MIN") || strings.Contains(got, "permkit") {
		t.Fatalf("cmd start title/flags leaked: %s", got)
	}
}

// TestRewriteBareStartHidden 确认裸 start /b 被改写，重定向不进参数，echo started 保留。
func TestRewriteBareStartHidden(t *testing.T) {
	in := `start /b python -m http.server 8777 --directory "game-adaptations\build\app" >nul 2>&1 & ping -n 3 127.0.0.1 >nul & echo started`
	got, ok := rewriteAnyStart(in)
	if !ok {
		t.Fatal("bare start /b must rewrite")
	}
	if strings.Contains(strings.ToLower(got), "start /b") {
		t.Fatalf("still start /b: %s", got)
	}
	if !strings.Contains(got, "Start-Process") || !strings.Contains(got, "python") {
		t.Fatalf("missing Start-Process python: %s", got)
	}
	if !strings.Contains(got, "http.server") {
		t.Fatalf("lost module: %s", got)
	}
	if strings.Contains(got, `'>nul'`) || strings.Contains(got, `'2>&1'`) {
		t.Fatalf("redirect leaked into args: %s", got)
	}
	if !strings.Contains(got, "echo started") {
		t.Fatalf("lost trailing probe: %s", got)
	}
}

// TestRewriteCmdStartWaitStaysError 确认 /WAIT 不改写，但仍算挂起。
func TestRewriteCmdStartWaitStaysError(t *testing.T) {
	in := `cmd /c start /WAIT permserver.exe`
	if _, ok := rewriteCmdStart(in); ok {
		t.Fatal(" /WAIT must not rewrite")
	}
	if !cmdStartWouldHang(in) {
		t.Fatal(" /WAIT still hangs while stdout is captured")
	}
}

// TestChatConductDoesNotTeachOSTrivia 确认聊天行为片段不含 Start-Process 和 cmd start，中文任务用中文，并含 web_fetch 和 leftover。
func TestChatConductDoesNotTeachOSTrivia(t *testing.T) {
	text := fragmentText(ChatConductFragments(false))
	if strings.Contains(text, "Start-Process") || strings.Contains(text, "cmd start") || strings.Contains(text, "中文任务用中文") {
		t.Fatalf("trivia leaked into conduct: %s", text)
	}
	if !strings.Contains(text, "web_fetch") {
		t.Fatal("public HTTP routing missing")
	}
	if !strings.Contains(text, "leftover") {
		t.Fatal("leftover artifact routing missing")
	}
}

// TestLanguagePinFollowsHan 确认汉字发言返回含 Chinese 的文本，纯英文返回空。
func TestLanguagePinFollowsHan(t *testing.T) {
	pin := LanguagePin("帮我查询一下关于agent自进化的论文")
	if pin.Text == "" || !strings.Contains(pin.Text, "Chinese") {
		t.Fatalf("%q", pin.Text)
	}
	if LanguagePin("find papers on agents").Text != "" {
		t.Fatal("english")
	}
}

// TestChatShellFuse 确认非叠层不熔断；叠层且超时为 5 秒不熔断；叠层且超时一分钟则空闲和阻塞用聊天阈值。
func TestChatShellFuse(t *testing.T) {
	idle, block, fuse := chatShellFuse(false, time.Minute)
	if fuse || idle != 0 || block != 0 {
		t.Fatalf("harbor must wait for timeout: idle=%s block=%s fuse=%v", idle, block, fuse)
	}
	idle, block, fuse = chatShellFuse(true, 5*time.Second)
	if fuse {
		t.Fatal("short timeout_sec must still kill")
	}
	idle, block, fuse = chatShellFuse(true, time.Minute)
	if !fuse || idle != chatShellIdle || block != chatShellBlock {
		t.Fatalf("chat long wait must fuse idle=%s block=%s", idle, block)
	}
}
