// Copyright 2026 Hypit.AI
// Translated from @hypit/author (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package author

import (
	"github.com/Shenchangxin/yoyo/remake/facet"
	"github.com/Shenchangxin/yoyo/remake/source"
)

// AuthorFrontendFacet is a Frontend facet whose kind is "author".
type AuthorFrontendFacet = facet.Facet

// CreateAuthorFrontendFacet builds the inert author Frontend declaration.
func CreateAuthorFrontendFacet(frontend AuthorFrontend) AuthorFrontendFacet {
	return facet.CreateFrontendFacet(source.SourceFrontendPackageABI, "author", frontend)
}

// AuthorFrontendsFromFacets returns author Frontends owned by the source Frontend ABI.
func AuthorFrontendsFromFacets(facets []facet.Facet) []AuthorFrontend {
	raw := facet.FrontendsFromFacets(source.SourceFrontendPackageABI, "author", facets)
	out := make([]AuthorFrontend, 0, len(raw))
	for _, item := range raw {
		out = append(out, item.(AuthorFrontend))
	}
	return out
}

// InstallAuthorFrontendFacets registers every author Frontend found in facets.
func InstallAuthorFrontendFacets(facets []facet.Facet, registry interface {
	Register(AuthorFrontend)
}) {
	for _, frontend := range AuthorFrontendsFromFacets(facets) {
		registry.Register(frontend)
	}
}
