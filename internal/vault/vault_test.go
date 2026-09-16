package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Set("default", "sk-test")
	got, err := s.Get("default")
	if err != nil || got != "sk-test" {
		t.Fatalf("%q %v", got, err)
	}
	s2 := New(dir)
	got, err = s2.Get("default")
	if err != nil || got != "sk-test" {
		t.Fatalf("reload %q %v", got, err)
	}
	st, err := os.Stat(filepath.Join(dir, "vault.json"))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 && st.Mode().Perm() != 0o666 {
		// Windows may not honor 0600; Unix must.
		if filepath.Separator == '/' && st.Mode().Perm() != 0o600 {
			t.Fatalf("perm %v", st.Mode().Perm())
		}
	}
}
