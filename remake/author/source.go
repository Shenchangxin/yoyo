// Copyright 2026 Hypit.AI
// Translated from @hypit/author (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package author

import (
	"fmt"

	"github.com/Shenchangxin/yoyo/remake/protocol"
	"github.com/Shenchangxin/yoyo/remake/source"
)

// AuthorSourceUnit is one authored source payload.
type AuthorSourceUnit = source.SourceUnit

// AuthorFrontendSourceUnit is the source unit a Frontend discovers and decodes.
type AuthorFrontendSourceUnit = AuthorSourceUnit

// AuthorSourceImport is one requested import edge.
type AuthorSourceImport = source.SourceImportRequest

// AuthorSourceDiscovery is the import set a Frontend reports before decoding.
type AuthorSourceDiscovery struct {
	Modules []string
	Sources []AuthorSourceImport
}

// AuthorSourceAssetRequest is one asset the Host must resolve.
type AuthorSourceAssetRequest = source.SourceAssetRequest

// ResolvedAuthorSourceAsset is the identity returned for one asset request.
type ResolvedAuthorSourceAsset = source.ResolvedSourceAsset

// AuthorSourceResolver resolves one import to a source unit.
type AuthorSourceResolver = source.SourceResolver

// AuthorSourceAssetResolver resolves one asset request.
type AuthorSourceAssetResolver = source.SourceAssetResolver

// AuthorSourceExport is one public value a decoded source publishes.
type AuthorSourceExport struct {
	Name string
	Ref  AuthorValueRef
	Type protocol.TypeRef
}

// AuthorSourceIdentity is one qualified identity in a source namespace.
type AuthorSourceIdentity struct {
	Namespace protocol.TypeRef
	ID        string
}

// AuthorInputProvenanceDraft is one input endpoint before identities are qualified.
// A nil Ref means the property is absent.
type AuthorInputProvenanceDraft struct {
	Name  string
	Range protocol.SourceRange
	Kind  string
	Ref   *AuthorValueRef
}

// AuthorElementProvenanceDraft is one author element before identities are qualified.
type AuthorElementProvenanceDraft struct {
	Range      protocol.SourceRange
	Records    []string
	Components []string
	Outputs    []struct {
		Component string
		Name      string
		ID        string
	}
	Inputs []AuthorInputProvenanceDraft
}

// ResolvedAuthorSourceImport is one import after the Host has resolved it.
type ResolvedAuthorSourceImport struct {
	Request AuthorSourceImport
	Source  string
	Exports []AuthorSourceExport
	Records []protocol.TypedRecord
}

// AuthorSourceDecodeContext is the closed world presented to Frontend decode.
type AuthorSourceDecodeContext struct {
	Closure      protocol.ResolvedModuleClosure
	Imports      []ResolvedAuthorSourceImport
	ResolveAsset func(AuthorSourceAssetRequest) ResolvedAuthorSourceAsset
}

// DecodedAuthorSource is the author module data a Frontend returns.
// A nil Identities or Provenance slice means the property is absent.
type DecodedAuthorSource struct {
	Records    []protocol.TypedRecord
	Components []AuthorComponent
	Fragments  []GraphFragment
	Exports    []AuthorSourceExport
	Identities []AuthorSourceIdentity
	Provenance []AuthorElementProvenanceDraft
}

// AuthorFrontend discovers and decodes one source language into author module data.
type AuthorFrontend struct {
	ID       string
	Discover func(AuthorFrontendSourceUnit) AuthorSourceDiscovery
	Decode   func(AuthorFrontendSourceUnit, AuthorSourceDecodeContext) DecodedAuthorSource
}

// AuthorFrontendRegistryLike resolves a Frontend by id.
// The boolean is false when the id is not registered.
type AuthorFrontendRegistryLike interface {
	Resolve(id string) (AuthorFrontend, bool)
}

// AuthorFrontendError is the coded failure thrown by Frontend registration and source exports.
type AuthorFrontendError struct {
	Code    string
	Message string
	Subject *string
	Name    string
}

func (e *AuthorFrontendError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// AuthorFrontendRegistry stores Frontends by id.
type AuthorFrontendRegistry struct {
	frontends map[string]AuthorFrontend
}

// Register adds one Frontend. An empty or repeated id throws AuthorFrontendError.
func (r *AuthorFrontendRegistry) Register(frontend AuthorFrontend) {
	if frontend.ID == "" {
		panic(&AuthorFrontendError{Code: "EMPTY_FRONTEND_ID", Message: "Frontend id is empty", Name: "AuthorFrontendError"})
	}
	if r.frontends == nil {
		r.frontends = map[string]AuthorFrontend{}
	}
	if _, exists := r.frontends[frontend.ID]; exists {
		id := frontend.ID
		panic(&AuthorFrontendError{
			Code:    "DUPLICATE_FRONTEND",
			Message: "Frontend " + id + " is already registered",
			Subject: &id,
			Name:    "AuthorFrontendError",
		})
	}
	r.frontends[frontend.ID] = frontend
}

// Resolve returns the Frontend registered under id.
func (r *AuthorFrontendRegistry) Resolve(id string) (AuthorFrontend, bool) {
	if r == nil || r.frontends == nil {
		return AuthorFrontend{}, false
	}
	frontend, ok := r.frontends[id]
	return frontend, ok
}

// AuthorRecordAdmitter validates one authored record without rewriting it.
type AuthorRecordAdmitter func(protocol.ResolvedModuleClosure, protocol.TypedRecord)

// SourceClosureUnit is one unit inside a source closure.
type SourceClosureUnit struct {
	ID       string
	Name     string
	Frontend string
	Imports  []struct {
		Alias  string
		Source string
	}
}

// AuthorInputProvenance is one qualified input endpoint.
// A nil Ref means the property is absent.
type AuthorInputProvenance struct {
	ID    string
	Name  string
	Range protocol.SourceRange
	Kind  string
	Ref   *AuthorValueRef
}

// AuthorElementProvenance is one qualified author element.
type AuthorElementProvenance struct {
	ID         string
	Source     string
	SourceName string
	Frontend   string
	Range      protocol.SourceRange
	Records    []struct {
		Local string
		ID    string
	}
	Components []struct {
		Local string
		ID    string
	}
	Outputs []struct {
		Component string
		Name      string
		Local     string
		ID        string
	}
	Inputs []AuthorInputProvenance
}

// AuthorProvenance is the ephemeral hypit.author-provenance@1 sidecar.
type AuthorProvenance struct {
	Format   string
	Elements []AuthorElementProvenance
}

// SourceClosure is the source units selected for one compilation.
type SourceClosure struct {
	Entry string
	Units []SourceClosureUnit
}

// CompiledSourceExport is one public export after author linking.
type CompiledSourceExport struct {
	Name string
	Type protocol.TypeRef
	Ref  protocol.GraphValueRef
}

// CompiledSourceClosure is the domain result of authoring.
// The Compiler owns how a Source closure is assembled.
type CompiledSourceClosure struct {
	Closure    SourceClosure
	Program    protocol.LinkedProgram
	Graph      protocol.CompiledGraph
	Exports    []CompiledSourceExport
	Provenance AuthorProvenance
}

// ResolveCompiledSourceExport returns the named export.
// A nil expected type skips the type check.
func ResolveCompiledSourceExport(compiled CompiledSourceClosure, name string, expected *protocol.TypeRef) CompiledSourceExport {
	found := false
	var item CompiledSourceExport
	for _, candidate := range compiled.Exports {
		if candidate.Name == name {
			item = candidate
			found = true
			break
		}
	}
	if !found {
		panic(&AuthorFrontendError{
			Code:    "UNKNOWN_SOURCE_EXPORT",
			Message: "unknown source export " + name,
			Subject: &name,
			Name:    "AuthorFrontendError",
		})
	}
	if expected != nil && !sameType(item.Type, *expected) {
		panic(&AuthorFrontendError{
			Code:    "SOURCE_EXPORT_TYPE_MISMATCH",
			Message: fmt.Sprintf("%s is %s, expected %s", name, typeName(item.Type), typeName(*expected)),
			Subject: &name,
			Name:    "AuthorFrontendError",
		})
	}
	return item
}
