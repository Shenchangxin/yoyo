// buddy_config.go 拥有 config.yaml 里 buddies 段的类型名。字段和校验都在 internal/buddy，这里只取别名，避免 buddy 反过来依赖 app。
// 读写在 buddy.go 的 buddySettings 和 saveBuddySettings。

package app

import "github.com/Shenchangxin/yoyo/internal/buddy"

// BuddiesConfig 是 config.yaml 的 buddies 段：npm 源、Python 源、注册表地址、默认 agent、自动挑选、单轮时长、每项设置和自定义 agent。
type BuddiesConfig = buddy.Settings

// BuddyConfig 是一项 agent 的设置。Env 只放非密钥，Secrets 只存名字，值在 vault 的 buddy.<id>.<NAME>。
type BuddyConfig = buddy.ItemConfig

// CustomBuddy 是人手动添加的 ACP agent。
type CustomBuddy = buddy.CustomAgent
