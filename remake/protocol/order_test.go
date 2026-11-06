// Copyright 2026 Hypit.AI
// Translated from @hypit/protocol (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package protocol

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestCanonicalStringifyMatchesJSONStringifyOnStrings(t *testing.T) {
	got := CanonicalStringify(map[string]any{"html": "<b>&</b>", "sep": "\u2028\u2029"})
	if !strings.Contains(got, `"<b>&</b>"`) {
		t.Fatalf("html escaped: %s", got)
	}
	if !strings.Contains(got, `\u2028`) || !strings.Contains(got, `\u2029`) {
		t.Fatalf("line separators: %s", got)
	}
}

func TestLocaleCompareMatchesNodeDefaultLocale(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is the localeCompare oracle and is not on PATH")
	}
	keys := []string{"b", "a", "A", "10", "2", "é", "0", " ", "_", "-", "."}
	got := append([]string(nil), keys...)
	sortKeys(got)
	cmd := exec.Command(node, "-e", `
const keys = process.argv[1].split("\n");
keys.sort((left, right) => left.localeCompare(right));
process.stdout.write(JSON.stringify(keys));
`, strings.Join(keys, "\n"))
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("localeCompare\n go   %q\n node %q", got, want)
	}
}
