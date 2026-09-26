package runtime

import (
	"strings"
	"testing"
	"time"
)

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

func TestRewriteCmdStartWaitStaysError(t *testing.T) {
	in := `cmd /c start /WAIT permserver.exe`
	if _, ok := rewriteCmdStart(in); ok {
		t.Fatal(" /WAIT must not rewrite")
	}
	if !cmdStartWouldHang(in) {
		t.Fatal(" /WAIT still hangs while stdout is captured")
	}
}

func TestChatConductDoesNotTeachOSTrivia(t *testing.T) {
	text := ChatConductFragments(false)[0].Text
	if strings.Contains(text, "Start-Process") || strings.Contains(text, "cmd start") || strings.Contains(text, "中文任务用中文") {
		t.Fatalf("trivia leaked into conduct: %s", text)
	}
	if !strings.Contains(text, "web_fetch") {
		t.Fatal("public HTTP routing missing")
	}
}

func TestLanguagePinFollowsHan(t *testing.T) {
	pin := LanguagePin("帮我查询一下关于agent自进化的论文")
	if pin.Text == "" || !strings.Contains(pin.Text, "Chinese") {
		t.Fatalf("%q", pin.Text)
	}
	if LanguagePin("find papers on agents").Text != "" {
		t.Fatal("english")
	}
}

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
