// 桌面回合用的模型目录缓存，以及回复 token 上限。目录快照的拉取在 internal/modelcatalog。

package app

import (
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/modelcatalog"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// catalogDir 是 Home.Root 下的 model-catalog 目录。
func (a *App) catalogDir() string {
	return filepath.Join(a.Home.Root, "model-catalog")
}

// ModelCatalog 加载缓存的 models.dev 快照。refresh 为真时允许拉取；过期与否由 modelcatalog.Load 决定。
func (a *App) ModelCatalog(refresh bool) (modelcatalog.Catalog, error) {
	return modelcatalog.Load(a.catalogDir(), refresh)
}

// chatMaxOutputTokens 是桌面回合向模型要求的输出上限，值为 32000。
const chatMaxOutputTokens = 32_000

// chatLargeWindowOutput 是大窗口模型的输出下限，值为 16384。窗口大于等于 200000 且目录输出更低时用它。
const chatLargeWindowOutput = 16_384

// maxOutputTokensFor 用目录里的输出上限和模型上下文窗口计算桌面回合的回复上限。目录未知时结果为 0，调用方保留供应商默认。
// model 传给 modelcatalog.OutputFor 和 runtime.ModelContextWindow。
func maxOutputTokensFor(model string) int {
	return capOutputTokens(modelcatalog.OutputFor(model), runtime.ModelContextWindow(model))
}

// capOutputTokens 在目录输出小于等于 0 时返回 0。大于 32000 时截到 chatMaxOutputTokens。窗口大于等于 200000 且当前值小于 16384 时抬到 chatLargeWindowOutput。
// catalogOut 小于等于 0 时返回 0，大于 chatMaxOutputTokens 时截到该常量。
// window 大于等于 200000 且当前值小于 chatLargeWindowOutput 时把结果抬到该常量。
func capOutputTokens(catalogOut, window int) int {
	n := catalogOut
	if n <= 0 {
		return 0
	}
	if n > chatMaxOutputTokens {
		n = chatMaxOutputTokens
	}
	if window >= 200_000 && n < chatLargeWindowOutput {
		n = chatLargeWindowOutput
	}
	return n
}

// warmModelCatalog 在 testing.Testing 为真时不拉取。否则以 refresh 为 false 调用 ModelCatalog。
func (a *App) warmModelCatalog() {
	if testing.Testing() {
		return
	}
	_, _ = a.ModelCatalog(false)
}
