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

func (a *App) warmModelCatalog() {
	if testing.Testing() {
		return
	}
	_, _ = a.ModelCatalog(false)
}
