// 市场列出哪几个站的测试。不发网络请求。

package skillmarket

import "testing"

// TestMarketListsTwoHubs 锁住市场只列 SkillHub 中国和 Anthropic，默认打开 SkillHub 中国，下线的站和认不出的 id 都回到默认站。
func TestMarketListsTwoHubs(t *testing.T) {
	got := HubCatalog()
	if len(got) != 2 || got[0].ID != HubSkillHubCN || got[1].ID != HubAnthropic {
		t.Fatalf("hubs = %+v", got)
	}
	for in, want := range map[string]string{
		"":            DefaultHub,
		"skillhubcn":  HubSkillHubCN,
		"skillhub.cn": HubSkillHubCN,
		"anthropic":   HubAnthropic,
		"ANTHROPIC":   HubAnthropic,
		"workbuddy":   DefaultHub,
		"clawhub":     DefaultHub,
		"skillsmp":    DefaultHub,
		"tessl":       DefaultHub,
		"nope":        DefaultHub,
	} {
		if got := MarketHub(in); got != want {
			t.Errorf("MarketHub(%q) = %s, want %s", in, got, want)
		}
	}
}

// TestRetiredHubsStillResolve 锁住下线的站仍然认得：已装技能的安装 id 还能拆开，修复和更新照常走原来的站。
func TestRetiredHubsStillResolve(t *testing.T) {
	for _, id := range []string{HubWorkBuddy, HubClawHub, HubClawSkillHub, HubSkillsMP, HubOsmani, HubClaudeSkills, HubVercel, HubMDR, HubTessl} {
		if !IsHub(id) {
			t.Errorf("%s should still resolve", id)
		}
	}
	if hub, remote := ParseInstallID("clawhub:awspace/pdf"); hub != HubClawHub || remote != "awspace/pdf" {
		t.Fatalf("clawhub id => %s %q", hub, remote)
	}
	if hub, remote := ParseInstallID("arxiv-watcher"); hub != HubWorkBuddy || remote != "arxiv-watcher" {
		t.Fatalf("bare legacy id => %s %q", hub, remote)
	}
}
