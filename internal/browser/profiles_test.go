package browser

import "testing"

func TestSanitizeProfileName(t *testing.T) {
	got, err := SanitizeProfileName("")
	if err != nil || got != "default" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := SanitizeProfileName("attached"); err == nil {
		t.Fatal("attached is reserved")
	}
	if _, err := SanitizeProfileName("has space"); err == nil {
		t.Fatal("spaces rejected")
	}
	got, err = SanitizeProfileName("github")
	if err != nil || got != "github" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestUseProfileChangesNameWithoutChrome(t *testing.T) {
	h, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	if err := h.UseProfile("work"); err != nil {
		t.Fatal(err)
	}
	if h.ProfileName() != "work" {
		t.Fatalf("name %s", h.ProfileName())
	}
	listed := h.ListProfiles()
	if listed[0] != "default" {
		t.Fatalf("%v", listed)
	}
}
