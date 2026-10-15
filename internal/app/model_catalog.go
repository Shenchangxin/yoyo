package app

import (
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/modelcatalog"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

func (a *App) catalogDir() string {
	return filepath.Join(a.Home.Root, "model-catalog")
}

// ModelCatalog serves the cached models.dev snapshot, fetching when stale.

func (a *App) ModelCatalog(refresh bool) (modelcatalog.Catalog, error) {
	return modelcatalog.Load(a.catalogDir(), refresh)
}

// chatMaxOutputTokens caps the output limit a desktop turn asks for. The cap is
// high enough for one reply to carry a long file part, and low enough that a
// provider's own ceiling is not the first thing the request hits.
const chatMaxOutputTokens = 32_000

// chatLargeWindowOutput is the floor for a large-window custom model whose
// catalog still lists 8k output. 32 KB write_file parts need headroom; 16k is
// conservative if a provider rejects a higher cap.
const chatLargeWindowOutput = 16_384

// maxOutputTokensFor returns the reply cap for a desktop turn, or 0 to keep the
// provider default when the model's limit is not known.
func maxOutputTokensFor(model string) int {
	return capOutputTokens(modelcatalog.OutputFor(model), runtime.ModelContextWindow(model))
}

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

func (a *App) warmModelCatalog() {
	if testing.Testing() {
		return
	}
	_, _ = a.ModelCatalog(false)
}
