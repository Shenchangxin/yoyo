// Copyright 2026 Hypit.AI
// Translated from @hypit/result/src/node/file-reference.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package result

import (
	"runtime"
	"testing"
)

func TestFileURLMessagesMatchNode(t *testing.T) {
	if message := recoverMessage(func() { fileURLToPath("http://example.com/a") }); message != "The URL must be of scheme file" {
		t.Fatal(message)
	}
	const encoded = "File URL path must not include encoded \\ or / characters"
	for _, raw := range []string{"file:///C:/temp/a%2Fb", "file:///C:/temp/a%5Cb"} {
		if message := recoverMessage(func() { fileURLToPath(raw) }); message != encoded {
			t.Fatalf("%s: %s", raw, message)
		}
	}
	if runtime.GOOS != "windows" {
		return
	}
	if got := fileURLToPath("file:///C:/temp/a"); got != `C:\temp\a` {
		t.Fatal(got)
	}
	if got := fileURLToPath("file://host/tmp/a"); got != `\\host\tmp\a` {
		t.Fatal(got)
	}
}

func recoverMessage(fn func()) (message string) {
	defer func() {
		rec := recover()
		err, ok := rec.(error)
		if !ok {
			panic(rec)
		}
		message = err.Error()
	}()
	fn()
	return ""
}
