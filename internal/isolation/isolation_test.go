package isolation

import "testing"

func TestStatusNeverLiesAboutSandbox(t *testing.T) {
	r := Status()
	if r.Sandbox && r.Kind == "none" {
		t.Fatal("sandbox badge with kind none")
	}
	if r.Kind == "job_object" && r.Sandbox {
		t.Fatal("job object is containment, not a sandbox badge")
	}
}
