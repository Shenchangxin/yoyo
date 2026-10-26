// 本文件锁住 Forget 会删掉情景记忆，以及 profile 写入后 Staging 为假并立刻出现在 ProfilePin 里。
// 不覆盖 Promote。由 go test 调用。

package memory

import (
	"strings"
	"testing"
)

// TestForgetRemoves 要求情景写入后 Staging 为真，Forget 之后按 SECRET 搜不到。
func TestForgetRemoves(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	it := s.Write(Item{Kind: KindEpisodic, Text: "SECRET_TOKEN_XYZ"})
	if !it.Staging {
		t.Fatal("writes must stay staging")
	}
	if !s.Forget(it.ID) {
		t.Fatal("forget")
	}
	if len(s.Search("SECRET", "", 10)) != 0 {
		t.Fatal("still present")
	}
}

// TestProfilePinSkipsStaging 要求 profile 写入后 Staging 为假、Source 保留，并且 ProfilePin 的结果里有 Ada。
func TestProfilePinSkipsStaging(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	it := s.Write(Item{Kind: KindProfile, Text: "call me Ada", Source: "User confirmed in chat"})
	if it.Staging || it.Source != "User confirmed in chat" {
		t.Fatalf("%+v", it)
	}
	if pin := s.ProfilePin(200); !strings.Contains(pin, "Ada") {
		t.Fatalf("profile writes pin immediately: %q", pin)
	}
}
