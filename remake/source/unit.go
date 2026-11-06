// Copyright 2026 Hypit.AI
// Translated from @hypit/source/src/unit.ts (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

// Package source is the byte-oriented Source contract shared by Author and Run compilation.
package source

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

type SourceUnit struct {
	// ID is the Workspace-canonical identity used only for recursion and diagnostics.
	ID   string
	Name string
	// Bytes is the opaque source payload. Its selected Frontend alone decides how to decode it.
	Bytes []byte
}

// ResolvedSource is a Workspace resolution.
// The Frontend choice is explicit and never guessed by the compiler.
type ResolvedSource struct {
	Unit     SourceUnit
	Frontend string
}

type SourceImportRequest struct {
	From  string
	Alias string
	Range *protocol.SourceRange
}

type SourceAssetRequest struct {
	From      string
	MediaType string
	// Bytes are the package-owned bytes handed to the Workspace for identity and attachment ownership.
	Bytes []byte
	Range *protocol.SourceRange
}

type ResolvedSourceAsset struct {
	Artifact protocol.BlobRef
}

type SourceResolver func(importer SourceUnit, request SourceImportRequest) ResolvedSource

type SourceAssetResolver func(importer SourceUnit, request SourceAssetRequest) ResolvedSourceAsset

// DecodeSourceText is the explicit adapter for textual Frontends; Source itself remains byte-oriented.
func DecodeSourceText(unit SourceUnit) string {
	if !utf8.Valid(unit.Bytes) {
		panic(errors.New("The encoded data was not valid for encoding utf-8"))
	}
	text := string(unit.Bytes)
	return strings.TrimPrefix(text, "\uFEFF")
}

// SourceFrontendPackageABI is the logical package address for Frontend facets selected by a ResolvedSource authority.
const SourceFrontendPackageABI = "hypit.source-frontend@1"
