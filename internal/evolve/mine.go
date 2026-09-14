package evolve

import (
	"fmt"
	"sort"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

type Signature struct {
	VerifierCause  string `json:"verifier_cause"`
	CausalStatus   string `json:"causal_status"`
	AgentMechanism string `json:"agent_mechanism"`
}

func (s Signature) Key() string {
	return s.VerifierCause + "|" + s.CausalStatus + "|" + s.AgentMechanism
}

type Cluster struct {
	Signature Signature `json:"signature"`
	Size      int       `json:"size"`
	TaskIDs   []string  `json:"task_ids"`
	Symptoms  []string  `json:"symptoms"`
}

type EvidenceBundle struct {
	Clusters []Cluster `json:"clusters"`
}

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

func classify(evs []trace.Event, cause string) Signature {
	retries := 0
	tools := 0
	wrote := false
	for _, ev := range evs {
		switch ev.Type {
		case trace.TypeToolCall:
			tools++
			name, _ := ev.Payload["name"].(string)
			if name == "write_file" || name == "str_replace" {
				wrote = true
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
	case tools > 20:
		mech = "stalled_tool_loop"
		status = "causal"
	}
	if cause == "" {
		cause = "verifier_fail"
	}
	return Signature{VerifierCause: cause, CausalStatus: status, AgentMechanism: mech}
}

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
