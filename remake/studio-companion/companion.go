// Copyright 2026 Hypit.AI
// Translated from @hypit/studio-companion (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package studiocompanion

import (
	"errors"
	"strings"

	"github.com/Shenchangxin/yoyo/remake/facet"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// StudioCompanionFacet is a Facet whose ABI is StudioCompanionFacetABI.
// Implementation is a StudioCompanionContribution.
type StudioCompanionFacet = facet.Facet

// CreateStudioTrackCompanionFacet builds a facet that contributes only track companions.
func CreateStudioTrackCompanionFacet(tracks []StudioTrackCompanion) StudioCompanionFacet {
	return StudioCompanionFacet{
		ABI: StudioCompanionFacetABI,
		Implementation: StudioCompanionContribution{
			Format: studioCompanionsFormat,
			Tracks: tracks,
		},
	}
}

// CreateStudioCompanionFacet builds a facet from the contribution arms the caller supplied.
// Omitted arms are left absent. Omitted tracks become an empty array.
func CreateStudioCompanionFacet(input StudioCompanionFacetInput) StudioCompanionFacet {
	tracks := input.Tracks
	if tracks == nil {
		tracks = []StudioTrackCompanion{}
	}
	implementation := StudioCompanionContribution{
		Format: studioCompanionsFormat,
		Tracks: tracks,
	}
	if input.Films != nil {
		implementation.Films = input.Films
	}
	if input.TemporalDomains != nil {
		implementation.TemporalDomains = input.TemporalDomains
	}
	if input.TemporalDeclarations != nil {
		implementation.TemporalDeclarations = input.TemporalDeclarations
	}
	if input.TemporalRelations != nil {
		implementation.TemporalRelations = input.TemporalRelations
	}
	if input.Parameters != nil {
		implementation.Parameters = input.Parameters
	}
	return StudioCompanionFacet{
		ABI:            StudioCompanionFacetABI,
		Implementation: implementation,
	}
}

func qualify(owner, local, subject string) string {
	if len(local) == 0 || strings.Contains(local, "#") {
		panic(errors.New(subject + " ids are package-local names without '#': " + owner + "#" + local))
	}
	return owner + "#" + local
}

func invalidContribution(owner string) error {
	return errors.New("Studio companion facet from " + owner + " has an invalid contribution")
}

func contributionFrom(impl any, owner string) StudioCompanionContribution {
	if impl == nil {
		panic(errors.New("Cannot read properties of undefined (reading 'format')"))
	}
	switch value := impl.(type) {
	case StudioCompanionContribution:
		if value.Format != studioCompanionsFormat || value.Tracks == nil {
			panic(invalidContribution(owner))
		}
		return value
	case *StudioCompanionContribution:
		if value == nil {
			panic(errors.New("Cannot read properties of null (reading 'format')"))
		}
		if value.Format != studioCompanionsFormat || value.Tracks == nil {
			panic(invalidContribution(owner))
		}
		return *value
	default:
		panic(invalidContribution(owner))
	}
}

// StudioContributionFromPackage qualifies package-local Companion names with the physical package identity.
// The executable facet cannot choose or impersonate its physical owner.
func StudioContributionFromPackage(owner string, facets []facet.Facet) StudioPackageContribution {
	if len(owner) == 0 || strings.Contains(owner, "#") {
		panic(errors.New("Invalid Studio companion package identity: " + owner))
	}
	tracks := []StudioTrackCompanion{}
	films := []StudioFilmCompanion{}
	temporalDomains := []StudioTemporalDomainCompanion{}
	temporalDeclarations := []StudioTemporalDeclarationCompanion{}
	temporalRelations := []StudioTemporalRelationCompanion{}
	parameters := []StudioParameterCompanion{}
	for _, item := range facets {
		if item.ABI != StudioCompanionFacetABI {
			continue
		}
		contribution := contributionFrom(item.Implementation, owner)
		for _, track := range contribution.Tracks {
			track.ID = qualify(owner, track.ID, "Studio Track companion")
			tracks = append(tracks, track)
		}
		filmList := contribution.Films
		if filmList == nil {
			filmList = []StudioFilmCompanion{}
		}
		for _, film := range filmList {
			film.ID = qualify(owner, film.ID, "Studio Film companion")
			films = append(films, film)
		}
		parameterList := contribution.Parameters
		if parameterList == nil {
			parameterList = []StudioParameterCompanion{}
		}
		for _, parameter := range parameterList {
			parameter.ID = qualify(owner, parameter.ID, "Studio Parameter companion")
			parameters = append(parameters, parameter)
		}
		domainList := contribution.TemporalDomains
		if domainList == nil {
			domainList = []StudioTemporalDomainCompanion{}
		}
		for _, domain := range domainList {
			domain.ID = qualify(owner, domain.ID, "Studio Temporal Domain companion")
			temporalDomains = append(temporalDomains, domain)
		}
		declarationList := contribution.TemporalDeclarations
		if declarationList == nil {
			declarationList = []StudioTemporalDeclarationCompanion{}
		}
		for _, declaration := range declarationList {
			declaration.ID = qualify(owner, declaration.ID, "Studio Temporal Declaration companion")
			temporalDeclarations = append(temporalDeclarations, declaration)
		}
		relationList := contribution.TemporalRelations
		if relationList == nil {
			relationList = []StudioTemporalRelationCompanion{}
		}
		for _, relation := range relationList {
			relation.ID = qualify(owner, relation.ID, "Studio Temporal Relation companion")
			temporalRelations = append(temporalRelations, relation)
		}
	}
	return StudioPackageContribution{
		Tracks:               tracks,
		Films:                films,
		TemporalDomains:      temporalDomains,
		TemporalDeclarations: temporalDeclarations,
		TemporalRelations:    temporalRelations,
		Parameters:           parameters,
	}
}

// StudioTrackCompanionsFromPackage qualifies track companions from the selected physical package.
func StudioTrackCompanionsFromPackage(owner string, facets []facet.Facet) []StudioTrackCompanion {
	return StudioContributionFromPackage(owner, facets).Tracks
}

func isUndefined(value any) bool {
	_, ok := value.(protocol.Undefined)
	return ok
}

// SameSurfaceValue returns the value of one same-Surface output port.
// protocol.Undefined means the port or the value is JavaScript undefined.
// A nil result means null.
func SameSurfaceValue(context StudioTrackCompanionContext, port string) any {
	found := false
	ref := ""
	for _, candidate := range context.Track.Trace.OutputPorts {
		if candidate.Name == port {
			ref = candidate.Ref
			found = true
			break
		}
	}
	if !found {
		return protocol.Undefined{}
	}
	value, ok := context.Values[ref]
	if !ok || isUndefined(value) {
		return protocol.Undefined{}
	}
	return value
}

// RequiredSurfaceValue returns one required same-Surface value or throws.
func RequiredSurfaceValue(context StudioTrackCompanionContext, port string) any {
	value := SameSurfaceValue(context, port)
	if isUndefined(value) {
		panic(errors.New("Studio Companion " + context.Track.Type + " requires same-Surface value port " + port))
	}
	return value
}

// RequiredReferencedValue resolves one direct authored reference by input name and exact public TypeRef.
func RequiredReferencedValue(context StudioTrackCompanionContext, input string, typ protocol.TypeRef) any {
	found := []StudioTraceReference{}
	for _, reference := range context.Track.Trace.References {
		if reference.Input != nil && *reference.Input == input && protocol.SameType(reference.TypeRef, typ) {
			found = append(found, reference)
		}
	}
	if len(found) != 1 {
		panic(errors.New("Studio Companion " + context.Track.Type + " requires one " + input + " reference of type " + typ.Module.Name + "@" + typ.Module.Version + "#" + typ.Name))
	}
	value, ok := context.Values[found[0].Ref]
	if !ok || isUndefined(value) {
		panic(errors.New("Studio Companion " + context.Track.Type + " cannot resolve referenced value " + found[0].Ref))
	}
	return value
}

// ArtifactPreview builds an artifact-backed material preview.
func ArtifactPreview(kind, resource string) StudioMaterialPreview {
	return StudioMaterialPreview{Kind: kind, Source: StudioPreviewSource{Kind: "artifact", Resource: resource}}
}

// PreviewLayer builds a preview display layer.
// Omitting role uses "content". A supplied role is used exactly, including the empty string.
func PreviewLayer(preview StudioMaterialPreview, layout string, role ...string) StudioDisplayLayer {
	resolved := "content"
	if len(role) > 0 {
		resolved = role[0]
	}
	return StudioDisplayLayer{Kind: "preview", Role: resolved, Preview: preview, Layout: layout}
}

// TextLayer builds a text display layer.
func TextLayer(text string) StudioDisplayLayer {
	return StudioDisplayLayer{Kind: "text", Role: "content", Text: text}
}

// TemporalBindingsFor finds executed projections that name this domain item as a domain consumer.
// Omitting input matches every domain consumer. A supplied input must equal that consumer's input.
func TemporalBindingsFor(context StudioTrackCompanionContext, subjectID string, input ...string) []StudioTemporalBinding {
	found := []StudioTemporalBinding{}
	for _, binding := range context.TemporalBindings {
		if binding.SubjectID != subjectID {
			continue
		}
		matched := false
		for _, consumer := range binding.Consumers {
			if consumer.Role != "domain" {
				continue
			}
			if len(input) == 0 || consumer.Input == input[0] {
				matched = true
				break
			}
		}
		if matched {
			found = append(found, binding)
		}
	}
	return found
}

func childIDIs(child StudioPlacementChild, authoredID string) bool {
	return child.ID != nil && *child.ID == authoredID
}

func objectIDIs(value any, authoredID string) bool {
	if value == nil {
		return false
	}
	object, ok := value.(map[string]any)
	if !ok || object == nil {
		return false
	}
	id, ok := object["id"].(string)
	return ok && id == authoredID
}

func childMatchesTyped(child StudioPlacementChild, authoredID string, types []protocol.TypeRef) bool {
	for _, observed := range child.Values {
		typeOK := false
		for _, typ := range types {
			if protocol.SameType(observed.Type, typ) {
				typeOK = true
				break
			}
		}
		if typeOK && objectIDIs(observed.Value, authoredID) {
			return true
		}
	}
	return false
}

// AuthoredChildFor finds the exact authored child whose typed public value owns a projected identity.
// It returns nil when no child matches. The result points at the child stored on the placement.
func AuthoredChildFor(context StudioTrackCompanionContext, authoredID string, types []protocol.TypeRef) *StudioPlacementChild {
	var children []StudioPlacementChild
	if context.Placement != nil {
		children = context.Placement.Children
	}
	explicit := []int{}
	typed := []int{}
	for index := range children {
		if childIDIs(children[index], authoredID) {
			explicit = append(explicit, index)
		}
	}
	for index := range children {
		if childMatchesTyped(children[index], authoredID, types) {
			typed = append(typed, index)
		}
	}
	seen := map[int]struct{}{}
	found := []int{}
	combined := make([]int, 0, len(explicit)+len(typed))
	combined = append(combined, explicit...)
	combined = append(combined, typed...)
	for _, index := range combined {
		if _, ok := seen[index]; ok {
			continue
		}
		seen[index] = struct{}{}
		found = append(found, index)
	}
	if len(found) > 1 {
		panic(errors.New("Studio authored child identity " + authoredID + " is ambiguous."))
	}
	if len(found) == 0 {
		return nil
	}
	return &children[found[0]]
}

// TemporalLineageFor returns the single executed lineage for a domain item.
// It returns nil when nothing matches. Several matches throw.
func TemporalLineageFor(context StudioTrackCompanionContext, subjectID string, input ...string) *StudioTemporalLineage {
	found := TemporalBindingsFor(context, subjectID, input...)
	if len(found) > 1 {
		suffix := ""
		if len(input) > 0 {
			suffix = " at " + input[0]
		}
		panic(errors.New("Studio Temporal lineage is ambiguous for " + subjectID + suffix + "."))
	}
	if len(found) == 0 {
		return nil
	}
	binding := found[0]
	return &StudioTemporalLineage{Record: binding.Record, Projection: binding.Projection}
}

func sameDomain(left, right *StudioTemporalSourceDomain) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return left.Companion == right.Companion && left.ID == right.ID
}

// TemporalDomainSource returns the unique domain-owned author identity carried by one projection.
// It returns nil when the lineage is missing or the domain endpoints disagree.
func TemporalDomainSource(lineage *StudioTemporalLineage) *StudioTemporalSource {
	if lineage == nil {
		return nil
	}
	domain := []*StudioTemporalSource{}
	if lineage.Projection.Kind == "instant" {
		if lineage.Projection.Authority.Kind == "domain" {
			domain = append(domain, &lineage.Projection.Authority.Source)
		}
	} else {
		if lineage.Projection.Start.Authority.Kind == "domain" {
			domain = append(domain, &lineage.Projection.Start.Authority.Source)
		}
		if lineage.Projection.End.Authority.Kind == "domain" {
			domain = append(domain, &lineage.Projection.End.Authority.Source)
		}
	}
	if len(domain) == 0 {
		return nil
	}
	first := domain[0]
	for _, candidate := range domain {
		if candidate.Kind != first.Kind || candidate.ID != first.ID || candidate.TimelineID != first.TimelineID || !sameDomain(candidate.Domain, first.Domain) {
			return nil
		}
	}
	return first
}

// AuthoredItemTitle chooses a source reference that can name an otherwise unnamed Item.
// A package chooses which source references can name it.
func AuthoredItemTitle(context StudioTrackCompanionContext, authoredID string, sourceTypes []protocol.TypeRef, sourceAttributes []string) string {
	child := AuthoredChildFor(context, authoredID, sourceTypes)
	if child != nil {
		if id := child.Attributes["id"]; id != "" {
			return id
		}
		for _, attribute := range sourceAttributes {
			if reference := child.ReferenceAttributes[attribute]; reference != "" {
				return reference
			}
		}
	}
	return authoredID
}

// ChildItems projects domain items into timeline drafts from public identity and exact Spec Types.
func ChildItems(context StudioTrackCompanionContext, items []StudioChildItem, kind, chrome string) []StudioItemDraft {
	drafts := make([]StudioItemDraft, 0, len(items))
	for _, item := range items {
		authoredID := item.ID
		if item.SubjectID != nil {
			authoredID = *item.SubjectID
		}
		sourceTypes := item.SourceTypes
		if sourceTypes == nil {
			sourceTypes = []protocol.TypeRef{}
		}
		child := AuthoredChildFor(context, authoredID, sourceTypes)
		// Both correspondences are exact public facts: the Program item identity
		// and the renderer's declared subject. No renderer naming convention is
		// interpreted here.
		renders := []StudioSpan{}
		for _, span := range context.Spans {
			if span.ID == item.ID || (span.SubjectID != nil && *span.SubjectID == authoredID) {
				renders = append(renders, span)
			}
		}
		draft := StudioItemDraft{
			ID:                context.Track.OutputRef + ":item:" + item.ID,
			AuthoredID:        authoredID,
			Display:           StudioItemDisplay{Title: authoredID, Layers: []StudioDisplayLayer{}},
			StartFrame:        item.StartFrame,
			EndFrameExclusive: item.EndFrameExclusive,
			StackOrder:        item.StackOrder,
			Presentation:      &StudioTimelinePresentation{Kind: kind, Chrome: chrome},
		}
		if child != nil {
			elementRange := child.Range
			draft.ElementRange = &elementRange
		}
		if len(renders) > 0 {
			presentID := renders[0].ID
			draft.PresentID = &presentID
			renderIDs := make([]string, len(renders))
			for index, span := range renders {
				renderIDs[index] = span.ID
			}
			draft.RenderIDs = renderIDs
		}
		drafts = append(drafts, draft)
	}
	return drafts
}
