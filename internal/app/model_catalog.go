package app

import (
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/modelcatalog"
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

// maxOutputTokensFor returns the reply cap for a desktop turn, or 0 to keep the
// provider default when the model's limit is not known.
func maxOutputTokensFor(model string) int {
	n := modelcatalog.OutputFor(model)
	if n > chatMaxOutputTokens {
		n = chatMaxOutputTokens
	}
	return n
}

func (a *App) warmModelCatalog() {
	if testing.Testing() {
		return
	}
	_, _ = a.ModelCatalog(false)
}
