// 本文件保存会话身份、频道和马具跟随策略。运行中的队列在 runstate.go。

package session

import (
	"strings"
	"time"
)

// 频道是会话所在的对话线。agent、video、learn 各用各的身份。
const (
	// ChannelAgent 是智能体聊天频道，值是 agent。NormalizeChannel 的默认也是它。
	ChannelAgent = "agent"
	// ChannelVideo 是视频聊天频道，值是 video。
	ChannelVideo = "video"
	// ChannelLearn 是学习对话频道，值是 learn。它不出现在会话模块的列表里。
	ChannelLearn = "learn"
)

// NormalizeChannel 去掉空白后不区分大小写等于 video 时返回 video，等于 learn 时返回 learn，否则返回 agent。
// s 去掉空白后与 ChannelVideo、ChannelLearn 做不区分大小写比较。
func NormalizeChannel(s string) string {
	s = strings.TrimSpace(s)
	if strings.EqualFold(s, ChannelVideo) {
		return ChannelVideo
	}
	if strings.EqualFold(s, ChannelLearn) {
		return ChannelLearn
	}
	return ChannelAgent
}

// CatalogLane 决定技能目录用哪条车道。学习对话沿用智能体目录，这样工作室和短剧技能不会进学习对话。
// channel 传给 NormalizeChannel。结果为 ChannelLearn 时返回 ChannelAgent。
func CatalogLane(channel string) string {
	if NormalizeChannel(channel) == ChannelLearn {
		return ChannelAgent
	}
	return NormalizeChannel(channel)
}

// 马具策略决定会话执行哪一份快照。
const (
	// FollowActive 表示跟随 refs/active，值是 follow_active。NormalizePolicy 的默认也是它。
	FollowActive = "follow_active"
	// Pin 表示钉住 meta 里的哈希，值是 pin。
	Pin = "pin"
)

// 回合状态机写在 RunState.Status 里。
const (
	// StatusIdle 是空闲，值是 idle。LoadRun 把运行中的状态收成它。
	StatusIdle = "idle"
	// StatusRunning 是正在跑，值是 running。Acquire 写成它。
	StatusRunning = "running"
	// StatusAwaitingApproval 是等待审批，值是 awaiting_approval。
	StatusAwaitingApproval = "awaiting_approval"
	// StatusCompacting 是正在压缩，值是 compacting。
	StatusCompacting = "compacting"
	// StatusCancelled 是已取消，值是 cancelled。
	StatusCancelled = "cancelled"
)

// Meta 是落在磁盘上的会话身份。ID 由调用方保证不改。
type Meta struct {
	// ID 是会话标识。
	ID string `json:"id"`
	// CreatedAt 是创建时间。
	CreatedAt time.Time `json:"created_at"`
	// Workspace 是工作区路径。
	Workspace string `json:"workspace"`
	// Harness 是钉住时使用的快照哈希。
	Harness string `json:"harness"`
	// HarnessPolicy 是 follow_active 或 pin。空字符串按 follow_active 处理。
	HarnessPolicy string `json:"harness_policy,omitempty"`
	// ModelFingerprint 是模型指纹。
	ModelFingerprint string `json:"model_fingerprint"`
	// Title 是标题，可空。
	Title string `json:"title,omitempty"`
	// Archived 表示已归档。
	Archived bool `json:"archived"`
	// Pinned 表示在列表里钉住。这不是马具策略 Pin。
	Pinned bool `json:"pinned"`
	// Model 是模型名，可空。
	Model string `json:"model,omitempty"`
	// LoadedSkills 是已加载的技能 id。
	LoadedSkills []string `json:"loaded_skills,omitempty"`
	// PlanText 是计划正文，可空。
	PlanText string `json:"plan_text,omitempty"`
}

// ResolveHarness 返回这次要执行的快照哈希。policy 是 pin 且 pinned 非空时返回 pinned。
// 否则 active 非空就返回 active。active 也空时返回 pinned。
func ResolveHarness(policy, pinned, active string) string {
	if policy == Pin && pinned != "" {
		return pinned
	}
	if active != "" {
		return active
	}
	return pinned
}

// NormalizePolicy 只保留 pin。其他值，包括空字符串，都返回 follow_active。
func NormalizePolicy(p string) string {
	switch p {
	case Pin:
		return Pin
	default:
		return FollowActive
	}
}
