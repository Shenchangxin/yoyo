// env.go 定下工单执行变量的名字。这些变量由 yoyo 给出，不是用户配置：Runtime.refreshEnvLocked 按工单填值，
// ticketDir 把它们交给准备命令的进程环境，writePromptContext 把非空的写进工人提示。值里没有钥匙。
//
// 旧名 DSH_SUPERAGENT_* 已废弃。yoyo 改名之前写好的准备脚本可能还在读它们，所以准备命令的进程环境里
// 新旧两套名字都有（见 withLegacyEnv）；工人提示只写新名。下一个大版本删掉旧名时，只改 legacyEnvPrefix 和 withLegacyEnv。

package superagent

import "strings"

const (
	// envDir 是这张工单的工作目录。
	envDir = "YOYO_SUPERAGENT_DIR"
	// envName 是工单短名，没有短名时是工单号。
	envName = "YOYO_SUPERAGENT_NAME"
	// envTicket 是工单号。
	envTicket = "YOYO_SUPERAGENT_TICKET"
	// envGitHubRepo 是来源设置里的 GitHub 仓库。
	envGitHubRepo = "YOYO_SUPERAGENT_GITHUB_REPOSITORY"
	// envGitHubAccount 是来源设置里的 GitHub 账号名。
	envGitHubAccount = "YOYO_SUPERAGENT_GITHUB_ACCOUNT"
	// envFeishuBot 是来源设置里的飞书应用编号。
	envFeishuBot = "YOYO_SUPERAGENT_FEISHU_BOT"
	// envFeishuChat 是来源设置里的飞书群。
	envFeishuChat = "YOYO_SUPERAGENT_FEISHU_CHAT"
)

// envKeys 是写进工人提示的顺序。
var envKeys = []string{envDir, envName, envTicket, envGitHubRepo, envGitHubAccount, envFeishuBot, envFeishuChat}

const (
	// envPrefix 是现在的名字前缀。
	envPrefix = "YOYO_SUPERAGENT_"
	// legacyEnvPrefix 是废弃的旧前缀，只在准备命令的进程环境里继续给出。
	legacyEnvPrefix = "DSH_SUPERAGENT_"
)

// withLegacyEnv 返回 env 的副本：每个 YOYO_SUPERAGENT_X 再以 DSH_SUPERAGENT_X 写一份，值相同。其他名字原样保留。
// env 里已经有同名旧键时不覆盖。env 为 nil 时返回空 map。
func withLegacyEnv(env map[string]string) map[string]string {
	out := make(map[string]string, len(env)*2)
	for k, v := range env {
		out[k] = v
	}
	for k, v := range env {
		if rest, ok := strings.CutPrefix(k, envPrefix); ok {
			legacy := legacyEnvPrefix + rest
			if _, exists := out[legacy]; !exists {
				out[legacy] = v
			}
		}
	}
	return out
}
