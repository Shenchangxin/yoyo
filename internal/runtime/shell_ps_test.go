package runtime

import (
	"strings"
	"testing"
)

func TestParsePowerShellInvocation(t *testing.T) {
	file, extra, body, encoded := parsePowerShellInvocation(`Get-Process | ForEach-Object { $_.Id }`)
	if file != "" || encoded != nil || !strings.Contains(body, "Get-Process") {
		t.Fatalf("raw body file=%q extra=%v encoded=%v body=%q", file, extra, encoded, body)
	}

	file, extra, body, encoded = parsePowerShellInvocation(`powershell -NoProfile -Command Start-Process x`)
	if file != "" || encoded != nil || body != "Start-Process x" {
		t.Fatalf("unwrap command %q %v %q %v", file, extra, body, encoded)
	}

	file, extra, body, encoded = parsePowerShellInvocation(`pwsh -NoLogo -File C:\tmp\run.ps1 -Verbose`)
	if file != `C:\tmp\run.ps1` || body != "" || encoded != nil || len(extra) != 1 || extra[0] != "-Verbose" {
		t.Fatalf("file %q extra=%v body=%q encoded=%v", file, extra, body, encoded)
	}

	file, extra, body, encoded = parsePowerShellInvocation(`pwsh C:\packs\scripts\foo.ps1 arg1`)
	if file != `C:\packs\scripts\foo.ps1` || extra[0] != "arg1" || body != "" {
		t.Fatalf("implied file %q extra=%v body=%q", file, extra, body)
	}

	_, _, _, encoded = parsePowerShellInvocation(`powershell -EncodedCommand QQ==`)
	if len(encoded) < 2 || strings.ToLower(encoded[0]) != "-encodedcommand" {
		t.Fatalf("encoded %v", encoded)
	}
}

func TestMaterializePowerShellWritesFile(t *testing.T) {
	ws := t.TempDir()
	exe, args, cleanup, err := materializePowerShell("pwsh", `Get-ChildItem | Select-Object -First 3`, ws)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	if exe != "pwsh" {
		t.Fatalf("exe %s", exe)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-File") || !strings.Contains(joined, "yoyo-inline-") {
		t.Fatalf("args %v", args)
	}
}

func TestPowershellCommandArgStillUnwraps(t *testing.T) {
	got := powershellCommandArg(`powershell -NoProfile -Command Start-Process x`)
	if got != "Start-Process x" {
		t.Fatalf("unwrap %q", got)
	}
}
