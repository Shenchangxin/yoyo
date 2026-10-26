// 本文件锁住 Status 不会谎报沙箱：Kind 为 none 时 Sandbox 不能为 true，job_object 也不能把 Sandbox 报成 true。
// 不实现平台后端。由 go test 调用。

package isolation

import "testing"

// TestStatusNeverLiesAboutSandbox 锁住 Kind 为 none 时 Sandbox 不能为 true，且 job_object 不能把 Sandbox 报成 true。
func TestStatusNeverLiesAboutSandbox(t *testing.T) {
	r := Status()
	if r.Sandbox && r.Kind == "none" {
		t.Fatal("sandbox badge with kind none")
	}
	if r.Kind == "job_object" && r.Sandbox {
		t.Fatal("job object is containment, not a sandbox badge")
	}
}
