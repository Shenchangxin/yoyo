package acpworker

import "testing"

func TestPrintProbeMatchesShellEchoes(t *testing.T) {
	yes := []string{
		`node -e "console.log('real120')"`,
		`node -e "console.log('calling orchestrator_report')"`,
		`node.exe -e "console.log('t79')"`,
		`python -c "print('hi')"`,
		`echo still need the real tool call`,
		`Write-Output real`,
		`console.log('real')`,
	}
	no := []string{
		`node verify_tkt52.js plan.html verify_tkt52_results.json`,
		`node -e "require('fs').writeFileSync('a.txt','x')"`,
		`node -e "console.log('x'); console.log('y')"`,
		`echo hi > out.txt`,
		`echo hi && node verify.js`,
		``,
	}
	for _, cmd := range yes {
		if !isPrintProbe(cmd) {
			t.Errorf("want probe: %s", cmd)
		}
	}
	for _, cmd := range no {
		if isPrintProbe(cmd) {
			t.Errorf("want real command: %s", cmd)
		}
	}
}
