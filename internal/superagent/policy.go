// policy.go 是确定性的准入。它不调用模型。Mode 为 off、observe 或 auto。

package superagent

// Policy 是当前策略。Allowed 的键是来源上的发送者标识。
// Admission 的空格子用默认：在名单里但没点名，以及点了名但不在名单里，都是 inferred；两个都不满足是 ignore。
type Policy struct {
	Version   int
	Mode      string
	Allowed   map[string]bool
	Paused    bool
	Admission Admission
}

// Admission 是三格准入。取值是 explicit、inferred 或 ignore。空字符串表示用默认。
type Admission struct {
	ListedUnaddressed   string
	UnlistedAddressed   string
	UnlistedUnaddressed string
}

// Resolved 把空格子填成默认。ListedUnaddressed 和 UnlistedAddressed 默认 inferred，UnlistedUnaddressed 默认 ignore。
func (a Admission) Resolved() Admission {
	if a.ListedUnaddressed == "" {
		a.ListedUnaddressed = "inferred"
	}
	if a.UnlistedAddressed == "" {
		a.UnlistedAddressed = "inferred"
	}
	if a.UnlistedUnaddressed == "" {
		a.UnlistedUnaddressed = "ignore"
	}
	return a
}

// Gate 判断 t 能不能成为候选。admitted 为假时 reason 是拒绝短码；为真时 strength 为 explicit 或 inferred。
// 名单里且被点名的是 explicit。Mode 为 off 时其余都是 mode_off。剩下的格子由 Admission 决定。
func (p Policy) Gate(t Trigger) (admitted bool, strength, reason string) {
	if !t.Human {
		return false, "", "not_human"
	}
	if p.Allowed[t.Sender] && t.Addressed {
		return true, "explicit", ""
	}
	if p.Mode == "off" {
		return false, "", "mode_off"
	}
	switch p.admit(p.Allowed[t.Sender], t.Addressed) {
	case "explicit":
		return true, "explicit", ""
	case "inferred":
		return true, "inferred", ""
	default:
		return false, "", "not_admitted"
	}
}

// admit 读取准入格。allowed 为真且 addressed 为假时用 ListedUnaddressed。
// allowed 为假且 addressed 为真时用 UnlistedAddressed。其余用 UnlistedUnaddressed。空字符串用 Resolved 的默认。
func (p Policy) admit(allowed, addressed bool) string {
	a := p.Admission.Resolved()
	switch {
	case allowed && !addressed:
		return a.ListedUnaddressed
	case !allowed && addressed:
		return a.UnlistedAddressed
	default:
		return a.UnlistedUnaddressed
	}
}

// Revalidate 检查 m 是否还和当前策略一致。m 的 Version 必须等于当前版本；Strength 为 explicit 时 Sender 仍须在 Allowed 里。
func (p Policy) Revalidate(m Mandate) bool {
	if p.Version != m.Version {
		return false
	}
	if m.Strength == "explicit" && !p.Allowed[m.Sender] {
		return false
	}
	return true
}
