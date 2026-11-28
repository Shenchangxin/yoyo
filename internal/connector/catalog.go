package connector

import "strings"

// Catalog 返回连接目录。本机条目只覆盖已经在读的表层名字。
// 健康、位置、短信和相册留在对应设备上，这里不把它们当成 Windows 上的数据源。
func Catalog() []CatalogEntry {
	out := catalogEntries()
	for i := range out {
		out[i].Ready = readiness(out[i])
	}
	return out
}

// readiness 判断一项现在能不能直接用。界面按它排序和分组，只把 live 的排在最前。
func readiness(e CatalogEntry) string {
	switch e.Access {
	case "local":
		return "live"
	case "oauth":
		if tokenEndpoint(e.Provider) != "" || e.Provider == "slack" {
			return "live"
		}
		return "pack"
	case "source":
		return "elsewhere"
	}
	switch e.Provider {
	case "dingtalk", "wecom", "wechat":
		return "elsewhere"
	}
	// 邮箱（IMAP / SMTP）、CalDAV、飞书日历都是内置直连，填好地址和令牌就能用，不需要 MCP 包。
	if isMailbox(e.Provider) || isCalDAV(e.Provider) || isFeishuCalendar(e.Provider) {
		return "live"
	}
	if e.Platform == "ios" || e.Platform == "android" || strings.HasPrefix(e.Provider, "mac-") || strings.HasPrefix(e.Provider, "linux-operate") {
		return "device"
	}
	if spec, ok := tokenCalls()[e.Provider]; ok {
		if spec.URL == "" {
			return "endpoint"
		}
		return "live"
	}
	return "pack"
}

func catalogEntries() []CatalogEntry {
	return []CatalogEntry{
		local("win-desktop", "Windows Desktop", "windows"),
		local("win-docs", "Windows Documents", "windows"),
		local("win-downloads", "Windows Downloads", "windows"),
		local("win-files", "Windows files", "windows"),
		local("win-shell", "Windows shell", "windows"),
		local("mac-desktop", "Mac Desktop", "mac"),
		local("mac-docs", "Mac Documents", "mac"),
		local("mac-downloads", "Mac Downloads", "mac"),
		local("mac-files", "Mac files", "mac"),
		local("mac-shell", "Mac shell", "mac"),
		local("linux-desktop", "Linux Desktop", "linux"),
		local("linux-docs", "Linux Documents", "linux"),
		local("linux-downloads", "Linux Downloads", "linux"),
		local("linux-files", "Linux files", "linux"),
		local("linux-shell", "Linux shell", "linux"),

		{Provider: "gmail", Kind: KindMail, Label: "Gmail", Access: "oauth", AuthURL: "https://accounts.google.com/o/oauth2/v2/auth"},
		{Provider: "outlook", Kind: KindMail, Label: "Microsoft 365", Access: "oauth", AuthURL: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"},
		{Provider: "gcal", Kind: KindCalendar, Label: "Google Calendar", Access: "oauth", AuthURL: "https://accounts.google.com/o/oauth2/v2/auth"},
		{Provider: "slack", Kind: KindIM, Label: "Slack", Access: "oauth", AuthURL: "https://slack.com/oauth/v2/authorize"},
		{Provider: "qq-mail", Kind: KindMail, Label: "QQ Mail", Access: "mcp", MCPPack: true},
		{Provider: "netease-mail", Kind: KindMail, Label: "NetEase Mail", Access: "mcp", MCPPack: true},
		{Provider: "local", Kind: KindMail, Label: "Local mailbox", Access: "mcp", MCPPack: true},

		{Provider: "github", Kind: KindDrive, Label: "GitHub", Access: "source", MCPPack: true, AuthURL: "https://github.com/login/oauth/authorize"},
		{Provider: "gitlab", Kind: KindDrive, Label: "GitLab", Access: "mcp", MCPPack: true},
		{Provider: "feishu", Kind: KindIM, Label: "Feishu", Access: "source", MCPPack: true, AuthURL: "https://open.feishu.cn/open-apis/authen/v1/index"},
		{Provider: "dingtalk", Kind: KindIM, Label: "DingTalk", Access: "mcp", MCPPack: true, AuthURL: "https://login.dingtalk.com/oauth2/auth"},
		{Provider: "wecom", Kind: KindIM, Label: "WeCom", Access: "mcp", MCPPack: true},
		{Provider: "wechat", Kind: KindIM, Label: "WeChat", Access: "mcp", MCPPack: true},
		{Provider: "notion", Kind: KindDrive, Label: "Notion", Access: "mcp", MCPPack: true, AuthURL: "https://api.notion.com/v1/oauth/authorize"},
		{Provider: "wps", Kind: KindDrive, Label: "WPS", Access: "mcp", MCPPack: true},
		{Provider: "tencent-docs", Kind: KindDrive, Label: "Tencent Docs", Access: "mcp", MCPPack: true},
		{Provider: "weread", Kind: KindDrive, Label: "WeRead", Access: "mcp", MCPPack: true},
		{Provider: "dropbox", Kind: KindDrive, Label: "Dropbox", Access: "mcp", MCPPack: true},
		{Provider: "figma", Kind: KindDrive, Label: "Figma", Access: "mcp", MCPPack: true},
		{Provider: "miro", Kind: KindDrive, Label: "Miro", Access: "mcp", MCPPack: true},
		{Provider: "airtable", Kind: KindDrive, Label: "Airtable", Access: "mcp", MCPPack: true},
		{Provider: "supabase", Kind: KindDrive, Label: "Supabase", Access: "mcp", MCPPack: true},
		// 飞书云文档和飞书日历一样走用户身份的 OAuth：找、读新版文档，新建和追加内容（feishu_docs.go）。
		{Provider: "feishu-docs", Kind: KindDrive, Label: "Feishu Docs", Access: "oauth", AuthURL: feishuAuthorizeURL},

		{Provider: "tencent-meeting", Kind: KindCalendar, Label: "Tencent Meeting", Access: "mcp", MCPPack: true},
		{Provider: "zoom", Kind: KindCalendar, Label: "Zoom", Access: "mcp", MCPPack: true},
		{Provider: "calendly", Kind: KindCalendar, Label: "Calendly", Access: "mcp", MCPPack: true},
		{Provider: "calcom", Kind: KindCalendar, Label: "Cal.com", Access: "mcp", MCPPack: true},
		// CalDAV 填服务器、主体或日历集合地址和「用户名:密码」，iCloud、Fastmail、Nextcloud 都能用，见 caldav.go。
		// 飞书日历走用户身份的 OAuth（带 offline_access，令牌过期自动换票），见 feishu_oauth.go、feishu_calendar.go。
		{Provider: "caldav", Kind: KindCalendar, Label: "CalDAV (iCloud / Fastmail / Nextcloud)", Access: "mcp"},
		{Provider: "feishu-calendar", Kind: KindCalendar, Label: "Feishu Calendar", Access: "oauth", AuthURL: feishuAuthorizeURL},

		{Provider: "linear", Kind: KindTask, Label: "Linear", Access: "mcp", MCPPack: true},
		{Provider: "todoist", Kind: KindTask, Label: "Todoist", Access: "mcp", MCPPack: true},
		{Provider: "ticktick", Kind: KindTask, Label: "TickTick", Access: "mcp", MCPPack: true},
		{Provider: "trello", Kind: KindTask, Label: "Trello", Access: "mcp", MCPPack: true},
		{Provider: "asana", Kind: KindTask, Label: "Asana", Access: "mcp", MCPPack: true},
		{Provider: "clickup", Kind: KindTask, Label: "ClickUp", Access: "mcp", MCPPack: true},
		{Provider: "sentry", Kind: KindTask, Label: "Sentry", Access: "mcp", MCPPack: true},
		{Provider: "hubspot", Kind: KindTask, Label: "HubSpot", Access: "mcp", MCPPack: true},
		{Provider: "attio", Kind: KindTask, Label: "Attio", Access: "mcp", MCPPack: true},
		{Provider: "linkedin", Kind: KindTask, Label: "LinkedIn", Access: "mcp", MCPPack: true},
		{Provider: "quickbooks", Kind: KindTask, Label: "QuickBooks", Access: "mcp", MCPPack: true},
		{Provider: "zoho-invoice", Kind: KindTask, Label: "Zoho Invoice", Access: "mcp", MCPPack: true},
		{Provider: "ynab", Kind: KindTask, Label: "YNAB", Access: "mcp", MCPPack: true},
		// 飞书任务走用户身份的 OAuth：读自己的任务，建、改、完成、删除（feishu_task.go）。
		{Provider: "feishu-task", Kind: KindTask, Label: "Feishu Tasks", Access: "oauth", AuthURL: feishuAuthorizeURL},

		device("ios-calendar", "Apple Calendar", "ios", KindCalendar),
		device("ios-contacts", "Apple Contacts", "ios", KindIM),
		device("ios-mail", "Apple Mail", "ios", KindMail),
		device("ios-notes", "Apple Notes", "ios", KindDrive),
		device("ios-reminders", "Apple Reminders", "ios", KindTask),
		device("ios-health", "Apple Health", "ios", KindLocal),
		device("ios-music", "Apple Music", "ios", KindLocal),
		device("ios-photos", "Photos", "ios", KindDrive),
		device("ios-location", "iPhone Location", "ios", KindLocal),
		device("android-alarm", "Alarm & Timer", "android", KindLocal),
		device("android-screentime", "Android screen time", "android", KindLocal),
		device("android-calendar", "Android Calendar", "android", KindCalendar),
		device("android-contacts", "Contacts", "android", KindIM),
		device("android-files", "Android files", "android", KindDrive),
		device("android-location", "Android Location", "android", KindLocal),
		device("android-notification", "Notification", "android", KindIM),
		device("android-sms", "SMS", "android", KindIM),
		device("mac-calendar", "Apple Calendar", "mac", KindCalendar),
		device("mac-contacts", "Apple Contacts", "mac", KindIM),
		device("mac-mail", "Apple Mail", "mac", KindMail),
		device("mac-notes", "Apple Notes", "mac", KindDrive),
		device("mac-reminders", "Apple Reminders", "mac", KindTask),
		device("mac-operate", "Operate Computer", "mac", KindLocal),
		device("linux-operate", "Operate Computer", "linux", KindLocal),
	}
}

func local(provider, label, platform string) CatalogEntry {
	return CatalogEntry{Provider: provider, Kind: KindLocal, Label: label, Platform: platform, Access: "local"}
}

func device(provider, label, platform string, kind Kind) CatalogEntry {
	return CatalogEntry{Provider: provider, Kind: kind, Label: label, Platform: platform, Access: "mcp", MCPPack: true}
}
