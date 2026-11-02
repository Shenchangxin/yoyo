// policy.go 是确定性的准入。它不调用模型。Mode 为 off、observe 或 auto。

package superagent

import (
	"strings"
	"time"
)

// Policy 是当前策略。Allowed 的键是来源上的发送者标识。
// Admission 的空格子用默认：在名单里但没点名，以及点了名但不在名单里，都是 inferred；两个都不满足是 ignore。
type Policy struct {
	Version     int
	Mode        string
	Allowed     map[string]bool
	Paused      bool
	Admission   Admission
	EvalTimeout time.Duration
	// Rules 是团队写成的主动规则。推断的工作只有在 auto 下命中其中一条启用的规则才开工。
	Rules []Rule
}

// Rule 是一条主动规则。Enabled 为假时不参与匹配，已授权的工单在恢复时也不再认它。
type Rule struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Enabled bool   `json:"enabled"`
}

// evalTimeout 返回一轮评估的期限。EvalTimeout 小于等于 0 时用 120 秒，与原型默认 timeoutSeconds 一致。
func (p Policy) evalTimeout() time.Duration {
	if p.EvalTimeout <= 0 {
		return 120 * time.Second
	}
	return p.EvalTimeout
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
	// 坐在这台机器前的操作者是明确请求，三种模式都有效，不看外部名单。
	if t.Source == "desk" && t.Addressed && t.Sender == "operator" {
		return true, "explicit", ""
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

// Revalidate 检查 m 是否还和当前策略一致。m 的 Version 必须等于当前版本。
// 明确授权要求发送者仍在名单里；坐在机器前的操作者例外。推断授权要求仍是 auto，且当初命中的规则都还启用。
func (p Policy) Revalidate(m Mandate) bool {
	if p.Version != m.Version {
		return false
	}
	if m.Strength == "explicit" && m.Sender != "operator" && !p.Allowed[m.Sender] {
		return false
	}
	if m.Strength == "inferred" {
		if !p.rulesEnabled(m.Rules) {
			return false
		}
		// 人点过接受的工单，依据是 person。模式改回观察之后，后续仍然算数。写成的规则则必须还在自动。
		for _, id := range m.Rules {
			if id != "person" && id != "addressed" && p.Mode != "auto" {
				return false
			}
		}
		return true
	}
	return true
}

// MatchRules 返回 ids 里仍然启用的规则。有空编号、未知编号或停用规则时 ok 为假。ids 为空时 ok 为假。
func (p Policy) MatchRules(ids []string) ([]string, bool) {
	if len(ids) == 0 {
		return nil, false
	}
	var matched []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, false
		}
		found := false
		for _, rule := range p.Rules {
			if rule.ID == id && rule.Enabled && strings.TrimSpace(rule.Text) != "" {
				found = true
				break
			}
		}
		if !found {
			return nil, false
		}
		if !containsRule(matched, id) {
			matched = append(matched, id)
		}
	}
	return matched, true
}

// rulesEnabled 在 ids 每一条都仍然有效时为真。addressed 和 person 是接受时写下的固定依据，不要求出现在规则表里。
func (p Policy) rulesEnabled(ids []string) bool {
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		if id == "addressed" || id == "person" {
			continue
		}
		if _, ok := p.MatchRules([]string{id}); !ok {
			return false
		}
	}
	return true
}

func containsRule(ids []string, id string) bool {
	for _, item := range ids {
		if item == id {
			return true
		}
	}
	return false
}
