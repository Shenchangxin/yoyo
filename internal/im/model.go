package im

import "strings"

// Phrase 是同一句话的中文和英文。界面按当前语言取一句。
type Phrase struct {
	Zh string `json:"zh"`
	En string `json:"en"`
}

// Field 是一个平台要填写的配置项。Secret 为真时值进保险库，不进 im.json。
type Field struct {
	Key     string `json:"key"`
	Label   Phrase `json:"label"`
	Help    Phrase `json:"help,omitempty"`
	Secret  bool   `json:"secret,omitempty"`
	Default string `json:"default,omitempty"`
}

// Platform 是目录里的一种接入。Transport 是 outbound、listen、webhook 或 either。
type Platform struct {
	ID        string  `json:"id"`
	Name      Phrase  `json:"name"`
	Transport string  `json:"transport"`
	Summary   Phrase  `json:"summary"`
	Proactive bool    `json:"proactive,omitempty"`
	Fields    []Field `json:"fields"`
}

// Settings 是全部账号共用的收消息规则。
type Settings struct {
	WakePrefixes    []string `json:"wake_prefixes"`
	FriendNeedsWake bool     `json:"friend_needs_wake"`
	IgnoreSelf      bool     `json:"ignore_self"`
	IgnoreAtAll     bool     `json:"ignore_at_all"`
	Admins          []string `json:"admins"`
	AllowUsers      []string `json:"allow_users"`
	AllowGroups     []string `json:"allow_groups"`
	RatePerMinute   int      `json:"rate_per_minute"`
	ReplyMention    bool     `json:"reply_mention"`
	Segmented       bool     `json:"segmented"`
	ListenHost      string   `json:"listen_host"`
	ListenPort      int      `json:"listen_port"`
}

// DefaultSettings 是第一次打开时的规则。群聊要 @ 或以前缀开头。私聊直接进入。监听只绑本机。
func DefaultSettings() Settings {
	return Settings{
		WakePrefixes:  []string{"/"},
		IgnoreSelf:    true,
		RatePerMinute: 20,
		ListenHost:    "127.0.0.1",
		ListenPort:    8787,
	}
}

// Account 是一个已保存的机器人账号。HasSecret 只表示保险库里有没有这项，不带原文。
type Account struct {
	ID        string            `json:"id"`
	Platform  string            `json:"platform"`
	Label     string            `json:"label"`
	Enabled   bool              `json:"enabled"`
	Config    map[string]string `json:"config,omitempty"`
	HasSecret map[string]bool   `json:"has_secret,omitempty"`
}

// Status 是一个账号现在的运行情况。
type Status struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

// Inbound 是一条已经从平台译成纯文本的消息。Reply 把机器人的回复送回去。
type Inbound struct {
	AccountID  string
	Platform   string
	Origin     string
	Kind       string
	ChatID     string
	SenderID   string
	SenderName string
	SelfID     string
	Text       string
	Mentioned  bool
	AtAll      bool
	MessageID  string
	Admin      bool
	Reply      func(text string) error
}

// Snapshot 是设置页一次拉取的目录、规则、账号和状态。
type Snapshot struct {
	Catalog  []Platform `json:"catalog"`
	Settings Settings   `json:"settings"`
	Accounts []Account  `json:"accounts"`
	Statuses []Status   `json:"statuses"`
	HookBase string     `json:"hook_base"`
}

func (s Settings) normalized() Settings {
	if s.ListenHost == "" {
		s.ListenHost = "127.0.0.1"
	}
	if s.ListenPort <= 0 || s.ListenPort > 65535 {
		s.ListenPort = 8787
	}
	if s.RatePerMinute < 0 {
		s.RatePerMinute = 0
	}
	s.WakePrefixes = compact(s.WakePrefixes)
	s.Admins = compact(s.Admins)
	s.AllowUsers = compact(s.AllowUsers)
	s.AllowGroups = compact(s.AllowGroups)
	return s
}

func compact(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func cfg(a Account, key string) string {
	if a.Config == nil {
		return ""
	}
	return strings.TrimSpace(a.Config[key])
}
