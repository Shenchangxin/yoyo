// 从轨迹和失败题里归纳机制簇，并给既有通过又有失败的重复配上对照。不调用模型。Cycle 用它生成证据，held 与安全题不进失败表。

package evolve

import (
	"fmt"
	"sort"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// Signature 是一道失败题的验证原因、因果地位和智能体机制。
type Signature struct {
	VerifierCause  string `json:"verifier_cause"`
	CausalStatus   string `json:"causal_status"`
	AgentMechanism string `json:"agent_mechanism"`
}

// Key 把三个字段用竖线接成簇键。
func (s Signature) Key() string {
	return s.VerifierCause + "|" + s.CausalStatus + "|" + s.AgentMechanism
}

// Cluster 是同一签名下的题号和症状。
type Cluster struct {
	Signature Signature `json:"signature"`
	Size      int       `json:"size"`
	TaskIDs   []string  `json:"task_ids"`
	Symptoms  []string  `json:"symptoms"`
}

// ContrastivePair 是同一题既失败又通过时的两侧摘要。
type ContrastivePair struct {
	TaskID string `json:"task_id"`
	Fail   string `json:"fail,omitempty"`
	Pass   string `json:"pass,omitempty"`
}

// EvidenceBundle 是簇、通过摘要、先前试用和对照对。
type EvidenceBundle struct {
	Clusters []Cluster         `json:"clusters"`
	Passing  []PassSummary     `json:"passing,omitempty"`
	Prior    []PriorTrial      `json:"prior,omitempty"`
	Pairs    []ContrastivePair `json:"pairs,omitempty"`
}

// Mine 按失败题归簇。没有 write_file 或 str_replace 时机制是 missing_artifact；工具结果以 ERROR: 开头达到 2 次时是 unproductive_retry；同一工具名连续达到 8 次时是 stalled_tool_loop；否则是 generic_failure，因果地位为 contributing。原因为空时写成 verifier_fail。簇按大小从大到小排。
func Mine(events []trace.Event, failedTasks map[string]string) EvidenceBundle {
	type acc struct {
		sig   Signature
		tasks map[string]bool
		sym   []string
	}
	groups := map[string]*acc{}
	byTask := map[string][]trace.Event{}
	for _, ev := range events {
		if ev.TaskID != "" {
			byTask[ev.TaskID] = append(byTask[ev.TaskID], ev)
		}
	}
	for taskID, cause := range failedTasks {
		sig := classify(byTask[taskID], cause)
		g := groups[sig.Key()]
		if g == nil {
			g = &acc{sig: sig, tasks: map[string]bool{}}
			groups[sig.Key()] = g
		}
		g.tasks[taskID] = true
		g.sym = append(g.sym, summarize(byTask[taskID]))
	}
	var clusters []Cluster
	for _, g := range groups {
		c := Cluster{Signature: g.sig, Size: len(g.tasks), Symptoms: g.sym}
		for id := range g.tasks {
			c.TaskIDs = append(c.TaskIDs, id)
		}
		sort.Strings(c.TaskIDs)
		clusters = append(clusters, c)
	}
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].Size > clusters[j].Size })
	return EvidenceBundle{Clusters: clusters}
}

// AttachPairs 只保留 PassCount 和 FailCount 都大于 0、且不在 held 里的题。事件也跳过 held 题号。对照对按题号排序。
func AttachPairs(bundle EvidenceBundle, events []trace.Event, reportHeld map[string]repeatOutcome, held map[string]bool) EvidenceBundle {
	byTask := map[string][]trace.Event{}
	for _, ev := range events {
		if ev.TaskID != "" && !held[ev.TaskID] {
			byTask[ev.TaskID] = append(byTask[ev.TaskID], ev)
		}
	}
	var pairs []ContrastivePair
	for id, oc := range reportHeld {
		if held[id] {
			continue
		}
		if oc.PassCount == 0 || oc.FailCount == 0 {
			continue
		}
		pairs = append(pairs, ContrastivePair{
			TaskID: id,
			Fail:   summarize(byTask[id]) + " fail_repeats=" + itoa(oc.FailCount),
			Pass:   "pass_repeats=" + itoa(oc.PassCount),
		})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].TaskID < pairs[j].TaskID })
	bundle.Pairs = pairs
	return bundle
}

// repeatOutcome 是一道题的通过次数和失败次数。
type repeatOutcome struct {
	PassCount int
	FailCount int
}

// itoa 把整数格式化成十进制。
func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

// classify 根据是否写过文件、ERROR: 次数和同一工具名的最长连续次数选定机制。因果的三种机制把 CausalStatus 写成 causal。
func classify(evs []trace.Event, cause string) Signature {
	retries := 0
	wrote := false
	streak, best := 0, 0
	last := ""
	for _, ev := range evs {
		switch ev.Type {
		case trace.TypeToolCall:
			name, _ := ev.Payload["name"].(string)
			if name == "write_file" || name == "str_replace" {
				wrote = true
			}
			if name != "" && name == last {
				streak++
			} else {
				streak = 1
				last = name
			}
			if streak > best {
				best = streak
			}
		case trace.TypeToolResult:
			if content, _ := ev.Payload["content"].(string); len(content) >= 6 && content[:6] == "ERROR:" {
				retries++
			}
		}
	}
	mech := "generic_failure"
	status := "contributing"
	switch {
	case !wrote:
		mech = "missing_artifact"
		status = "causal"
	case retries >= 2:
		mech = "unproductive_retry"
		status = "causal"
	case best >= 8:
		mech = "stalled_tool_loop"
		status = "causal"
	}
	if cause == "" {
		cause = "verifier_fail"
	}
	return Signature{VerifierCause: cause, CausalStatus: status, AgentMechanism: mech}
}

// summarize 数工具调用次数，以及内容以 ERROR: 开头的工具结果次数。
func summarize(evs []trace.Event) string {
	nTool := 0
	nErr := 0
	for _, ev := range evs {
		if ev.Type == trace.TypeToolCall {
			nTool++
		}
		if ev.Type == trace.TypeToolResult {
			if c, _ := ev.Payload["content"].(string); len(c) >= 6 && c[:6] == "ERROR:" {
				nErr++
			}
		}
	}
	return fmt.Sprintf("tools=%d errors=%d", nTool, nErr)
}
