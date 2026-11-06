// Copyright 2026 Hypit.AI
// Translated from @hypit/workspace/src/node/workspace.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package workspace

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/remake/protocol"
	"github.com/Shenchangxin/yoyo/remake/source"
)

// NodeFilesystemExternalSource is one host-resolved source and its read boundary.
type NodeFilesystemExternalSource struct {
	Root   string
	Source string
	// Frontend is an explicit override for sources that are not self-described text.
	// A nil Frontend means the property is absent.
	Frontend *string
}

// NodeFilesystemExternalSourceResolver resolves an explicit non-relative Source locator.
type NodeFilesystemExternalSourceResolver func(importer source.SourceUnit, request source.SourceImportRequest) (NodeFilesystemExternalSource, error)

// NodeFilesystemExternalAsset is one host-resolved asset and its read boundary.
type NodeFilesystemExternalAsset struct {
	Root  string
	Asset string
}

// NodeFilesystemExternalAssetResolver resolves an explicit non-relative asset locator.
type NodeFilesystemExternalAssetResolver func(importer source.SourceUnit, request source.SourceAssetRequest) (NodeFilesystemExternalAsset, error)

// NodeFilesystemSourceAdapter selects a Frontend for each admitted Source.
type NodeFilesystemSourceAdapter func(unit source.SourceUnit) source.ResolvedSource

// NodeFilesystemWorkspaceOptions configures one filesystem Workspace.
type NodeFilesystemWorkspaceOptions struct {
	// Root is the fixed containment root.
	// A nil Root uses the entry SourceUnit directory for each session.
	Root *string
	// AssetRoots are additional read-only roots for asset bytes.
	// They never permit Source imports.
	AssetRoots []string
	// ExternalSourceResolver is the host-owned resolver for explicit non-relative Source locators.
	ExternalSourceResolver NodeFilesystemExternalSourceResolver
	// ExternalAssetResolver is the host-owned resolver for explicit non-relative asset locators.
	ExternalAssetResolver NodeFilesystemExternalAssetResolver
	// SourceAdapter selects a Frontend for each admitted Source.
	// A nil SourceAdapter uses the self-described text adapter.
	SourceAdapter NodeFilesystemSourceAdapter
}

// NodeFilesystemWorkspace is the host filesystem implementation of one compilation Workspace.
type NodeFilesystemWorkspace struct {
	root                   *string
	assetRoots             []string
	externalSourceResolver NodeFilesystemExternalSourceResolver
	externalAssetResolver  NodeFilesystemExternalAssetResolver
	sourceAdapter          NodeFilesystemSourceAdapter
}

// NodeFilesystemWorkspaceSession is one open filesystem compilation.
type NodeFilesystemWorkspaceSession struct {
	root                   string
	entry                  source.ResolvedSource
	assetRoots             []string
	externalSourceResolver NodeFilesystemExternalSourceResolver
	externalAssetResolver  NodeFilesystemExternalAssetResolver
	sourceAdapter          NodeFilesystemSourceAdapter
	sourceCache            map[string]source.SourceUnit
	sourceRoots            map[string]string
	assetIdentity          map[string]assetIdentity
	attachments            map[string]BlobAttachment
}

type assetIdentity struct {
	resource protocol.ResourceId
	size     int64
}

// NewNodeFilesystemWorkspace builds a filesystem Workspace.
// A zero options value reads each entry from its own directory through the self-described text adapter.
func NewNodeFilesystemWorkspace(options NodeFilesystemWorkspaceOptions) *NodeFilesystemWorkspace {
	adapter := options.SourceAdapter
	if adapter == nil {
		adapter = selfDescribedTextSource
	}
	roots := options.AssetRoots
	if roots == nil {
		roots = []string{}
	}
	return &NodeFilesystemWorkspace{
		root:                   options.Root,
		assetRoots:             roots,
		externalSourceResolver: options.ExternalSourceResolver,
		externalAssetResolver:  options.ExternalAssetResolver,
		sourceAdapter:          adapter,
	}
}

func selfDescribedTextSource(unit source.SourceUnit) source.ResolvedSource {
	return source.ResolveSelfDescribedTextSource(source.SelfDescribedTextSource{
		ID:   unit.ID,
		Name: unit.Name,
		Text: source.DecodeSourceText(unit),
	})
}

// Open admits one entry Source and returns its session.
func (w *NodeFilesystemWorkspace) Open(entryLocator string) (WorkspaceSession, error) {
	entry, err := filepath.Abs(entryLocator)
	if err != nil {
		return nil, err
	}
	rootLocator := filepath.Dir(entry)
	if w.root != nil {
		rootLocator = *w.root
	}
	return openFilesystemSession(rootLocator, entry, w.assetRoots, w.externalSourceResolver, w.externalAssetResolver, w.sourceAdapter)
}

func openFilesystemSession(
	rootLocator, entryLocator string,
	assetRootLocators []string,
	externalSourceResolver NodeFilesystemExternalSourceResolver,
	externalAssetResolver NodeFilesystemExternalAssetResolver,
	sourceAdapter NodeFilesystemSourceAdapter,
) (*NodeFilesystemWorkspaceSession, error) {
	root, err := realPath(rootLocator)
	if err != nil {
		return nil, err
	}
	assetRoots := make([]string, len(assetRootLocators))
	for i, path := range assetRootLocators {
		assetRoots[i], err = realPath(path)
		if err != nil {
			return nil, err
		}
	}
	canonicalEntry, err := realPath(entryLocator)
	if err != nil {
		return nil, err
	}
	if !isWithin(root, canonicalEntry) {
		panic(workspaceError("SOURCE_OUTSIDE_ROOT", fmt.Sprintf("Source %s is outside workspace root %s", canonicalEntry, root), canonicalEntry))
	}
	bytes, err := os.ReadFile(canonicalEntry)
	if err != nil {
		return nil, err
	}
	entryUnit := source.SourceUnit{
		ID:    canonicalEntry,
		Name:  sourceName(root, canonicalEntry),
		Bytes: bytes,
	}
	session := &NodeFilesystemWorkspaceSession{
		root:                   root,
		entry:                  sourceAdapter(entryUnit),
		assetRoots:             append([]string{root}, assetRoots...),
		externalSourceResolver: externalSourceResolver,
		externalAssetResolver:  externalAssetResolver,
		sourceAdapter:          sourceAdapter,
		sourceCache:            map[string]source.SourceUnit{},
		sourceRoots:            map[string]string{},
		assetIdentity:          map[string]assetIdentity{},
		attachments:            map[string]BlobAttachment{},
	}
	session.sourceCache[entryUnit.ID] = entryUnit
	session.sourceRoots[entryUnit.ID] = root
	return session, nil
}

// Entry returns the admitted entry Source.
func (s *NodeFilesystemWorkspaceSession) Entry() source.ResolvedSource {
	return s.entry
}

// Root returns the canonical containment root.
func (s *NodeFilesystemWorkspaceSession) Root() string {
	return s.root
}

// ResolveSource loads a relative import, or asks the host resolver for any other locator.
func (s *NodeFilesystemWorkspaceSession) ResolveSource(importer source.SourceUnit, request source.SourceImportRequest) (source.ResolvedSource, error) {
	importerRoot, ok := s.sourceRoots[importer.ID]
	if !ok {
		panic(workspaceError("UNKNOWN_SOURCE_IMPORTER", importer.ID+" is outside this Workspace", importer.ID))
	}
	if strings.HasPrefix(request.From, "./") || strings.HasPrefix(request.From, "../") {
		unit, err := s.loadSource(filepath.Join(filepath.Dir(importer.ID), request.From), importerRoot)
		if err != nil {
			return source.ResolvedSource{}, err
		}
		return s.sourceAdapter(unit), nil
	}
	if s.externalSourceResolver == nil {
		panic(workspaceError("UNSUPPORTED_SOURCE_IMPORT", "Source import "+request.From+" has no resolver in this Workspace", request.From))
	}
	external, err := s.externalSourceResolver(importer, request)
	if err != nil {
		return source.ResolvedSource{}, err
	}
	unit, err := s.loadSource(external.Source, external.Root)
	if err != nil {
		return source.ResolvedSource{}, err
	}
	if external.Frontend == nil {
		return s.sourceAdapter(unit), nil
	}
	return source.ResolvedSource{Unit: unit, Frontend: *external.Frontend}, nil
}

// ResolveAsset admits inline bytes or a file inside an allowed root.
func (s *NodeFilesystemWorkspaceSession) ResolveAsset(importer source.SourceUnit, request source.SourceAssetRequest) (source.ResolvedSourceAsset, error) {
	importerRoot, ok := s.sourceRoots[importer.ID]
	if !ok {
		panic(workspaceError("UNKNOWN_SOURCE_IMPORTER", importer.ID+" is outside this Workspace", importer.ID))
	}
	if request.Bytes != nil {
		copied := append([]byte(nil), request.Bytes...)
		artifact := protocol.BlobRef{
			Kind:      "blob",
			Resource:  newResourceID(),
			Size:      int64(len(copied)),
			MediaType: request.MediaType,
		}
		s.attachments[attachmentKey(artifact)] = BlobAttachment{
			Artifact: cloneBlob(artifact),
			Open: func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(append([]byte(nil), copied...))), nil
			},
		}
		return source.ResolvedSourceAsset{Artifact: artifact}, nil
	}
	relativeAsset := strings.HasPrefix(request.From, "./") || strings.HasPrefix(request.From, "../")
	var canonical string
	var allowed []string
	if relativeAsset {
		var err error
		canonical, err = realPath(filepath.Join(filepath.Dir(importer.ID), request.From))
		if err != nil {
			return source.ResolvedSourceAsset{}, err
		}
		if importerRoot == s.root {
			allowed = s.assetRoots
		} else {
			allowed = []string{importerRoot}
		}
	} else {
		if s.externalAssetResolver == nil {
			panic(workspaceError("UNSUPPORTED_SOURCE_ASSET", "Source asset "+request.From+" has no resolver in this Workspace", request.From))
		}
		external, err := s.externalAssetResolver(importer, request)
		if err != nil {
			return source.ResolvedSourceAsset{}, err
		}
		externalRoot, err := realPath(external.Root)
		if err != nil {
			return source.ResolvedSourceAsset{}, err
		}
		canonical, err = realPath(external.Asset)
		if err != nil {
			return source.ResolvedSourceAsset{}, err
		}
		allowed = []string{externalRoot}
	}
	inside := false
	for _, root := range allowed {
		if isWithin(root, canonical) {
			inside = true
			break
		}
	}
	if !inside {
		panic(workspaceError("SOURCE_ASSET_OUTSIDE_ROOT", "Source asset "+canonical+" is outside the workspace and every allowed asset root", canonical))
	}
	identity, ok := s.assetIdentity[canonical]
	if !ok {
		info, err := os.Stat(canonical)
		if err != nil {
			return source.ResolvedSourceAsset{}, err
		}
		identity = assetIdentity{resource: newResourceID(), size: info.Size()}
		s.assetIdentity[canonical] = identity
	}
	artifact := protocol.BlobRef{
		Kind:      "blob",
		Resource:  identity.resource,
		Size:      identity.size,
		MediaType: request.MediaType,
	}
	key := attachmentKey(artifact)
	if _, exists := s.attachments[key]; !exists {
		location := pathToFileURL(canonical)
		path := canonical
		s.attachments[key] = BlobAttachment{
			Artifact: cloneBlob(artifact),
			Location: &location,
			Open: func() (io.ReadCloser, error) {
				return os.Open(path)
			},
		}
	}
	return source.ResolvedSourceAsset{Artifact: cloneBlob(artifact)}, nil
}

// Attachments lists admitted blobs, ordered by resource and then media type.
func (s *NodeFilesystemWorkspaceSession) Attachments() []BlobAttachment {
	items := make([]BlobAttachment, 0, len(s.attachments))
	for _, item := range s.attachments {
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		byResource := protocol.LocaleCompare(string(items[i].Artifact.Resource), string(items[j].Artifact.Resource))
		if byResource == 0 {
			return protocol.LocaleCompare(items[i].Artifact.MediaType, items[j].Artifact.MediaType) < 0
		}
		return byResource < 0
	})
	out := make([]BlobAttachment, len(items))
	for i, item := range items {
		out[i] = item
		out[i].Artifact = cloneBlob(item.Artifact)
	}
	return out
}

func (s *NodeFilesystemWorkspaceSession) loadSource(path, sourceRootLocator string) (source.SourceUnit, error) {
	sourceRoot, err := realPath(sourceRootLocator)
	if err != nil {
		return source.SourceUnit{}, err
	}
	canonical, err := realPath(path)
	if err != nil {
		return source.SourceUnit{}, err
	}
	if !isWithin(sourceRoot, canonical) {
		panic(workspaceError("SOURCE_OUTSIDE_ROOT", fmt.Sprintf("Source %s is outside its source root %s", canonical, sourceRoot), canonical))
	}
	if cached, ok := s.sourceCache[canonical]; ok {
		previous := s.sourceRoots[canonical]
		if previous != sourceRoot {
			label := previous
			if label == "" {
				label = "an unknown root"
			}
			panic(workspaceError("SOURCE_ROOT_CONFLICT", fmt.Sprintf("Source %s was resolved through both %s and %s", canonical, label, sourceRoot), canonical))
		}
		return cached, nil
	}
	payload, err := os.ReadFile(canonical)
	if err != nil {
		return source.SourceUnit{}, err
	}
	unit := source.SourceUnit{ID: canonical, Name: sourceName(sourceRoot, canonical), Bytes: payload}
	s.sourceCache[canonical] = unit
	s.sourceRoots[canonical] = sourceRoot
	return unit, nil
}

func sourceName(root, canonical string) string {
	relation, err := filepath.Rel(root, canonical)
	if err != nil || relation == "." {
		relation = ""
	}
	if relation != "" {
		return relation
	}
	base := filepath.Base(canonical)
	if base != "" && base != "." {
		return base
	}
	return canonical
}

func isWithin(root, path string) bool {
	relation, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if relation == "." {
		relation = ""
	}
	if relation == "" {
		return true
	}
	return relation != ".." && !strings.HasPrefix(relation, ".."+string(filepath.Separator)) && !filepath.IsAbs(relation)
}

func attachmentKey(artifact protocol.BlobRef) string {
	return string(artifact.Resource) + "\x00" + artifact.MediaType
}

func cloneBlob(artifact protocol.BlobRef) protocol.BlobRef {
	return artifact
}

func newResourceID() protocol.ResourceId {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return protocol.ResourceId(fmt.Sprintf("res_%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16]))
}
