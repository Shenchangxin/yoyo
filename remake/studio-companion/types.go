// Copyright 2026 Hypit.AI
// Translated from @hypit/studio-companion (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

// Package studiocompanion is the stable companion-package ABI understood by Hypit Studio.
package studiocompanion

import "github.com/Shenchangxin/yoyo/remake/protocol"

// StudioCompanionFacetABI is the facet ABI understood by Studio.
const StudioCompanionFacetABI = "hypit.studio-companion@1"

const studioCompanionsFormat = "hypit.studio-companions@1"

// Range is UTF-16 offsets into the exact author source.
type Range struct {
	Start float64
	End   float64
}

// StudioTrackFamily is a package-owned track family name.
type StudioTrackFamily = string

// StudioIcon is brand, captions, component, layers, ranking, text, timeline, video, or waveform.
type StudioIcon = string

// StudioTimelineTone is Studio's finite palette.
// Domain families never become CSS selectors.
// The tones are blue, blue-muted, green, green-muted, teal, violet, magenta,
// magenta-muted, orange, orange-muted, and neutral.
type StudioTimelineTone = string

// StudioTimelinePresentation is package-owned item presentation.
type StudioTimelinePresentation struct {
	// Kind is the package-owned item kind used only for presentation and diagnostics.
	Kind string
	// Chrome is standard, group, point, or compact.
	// Compact presents the primary label in one line, with time in details.
	Chrome string
}

// StudioTemporalSourceDomain is present only when a package Companion recognizes the domain-owned value.
type StudioTemporalSourceDomain struct {
	Companion string
	ID        string
}

// StudioTemporalSource is the domain-owned origin of one temporal endpoint.
type StudioTemporalSource struct {
	TimelineID string
	// Type is the domain-owned Type. Studio does not enumerate future time domains.
	Type protocol.TypeRef
	Kind string
	ID   string
	// Domain is present only when a package Companion recognizes the domain-owned value.
	Domain *StudioTemporalSourceDomain
}

// StudioTemporalAuthority is domain, parameter, or fixed.
// Inactive arms stay at the zero value. Owner nil means the parameter owner is absent.
// Relation is direct, after-start, or before-end.
type StudioTemporalAuthority struct {
	Kind     string
	Source   StudioTemporalSource
	Boundary string
	Binding  string
	Owner    *string
	Relation string
}

// StudioTemporalInstantProjection is one absolute instant.
type StudioTemporalInstantProjection struct {
	Kind       string
	Expression string
	// Reference is domain-owned reference spelling, or one of Timeline's common expressions.
	Reference string
	Frame     float64
	Source    StudioTemporalSource
	Authority StudioTemporalAuthority
}

// StudioTemporalWindowProjection is a half-open window of two instants.
type StudioTemporalWindowProjection struct {
	Kind              string
	Start             StudioTemporalInstantProjection
	End               StudioTemporalInstantProjection
	StartFrame        float64
	EndFrameExclusive float64
}

// StudioTemporalProjection is an instant or a window.
// Kind "instant" uses the instant fields. Kind "window" uses Start, End, and the frame fields.
type StudioTemporalProjection struct {
	Kind              string
	Expression        string
	Reference         string
	Frame             float64
	Source            StudioTemporalSource
	Authority         StudioTemporalAuthority
	Start             StudioTemporalInstantProjection
	End               StudioTemporalInstantProjection
	StartFrame        float64
	EndFrameExclusive float64
}

// StudioTemporalLineage is one executed Temporal value and its projection.
type StudioTemporalLineage struct {
	// Record is the exact executed Temporal value. Studio keeps this identity for inverse traversal.
	Record     string
	Projection StudioTemporalProjection
}

// StudioTemporalConsumerInput is one input recorded beside a temporal consumer.
type StudioTemporalConsumerInput struct {
	Name   string
	Record string
	Type   protocol.TypeRef
	// HasValue is false when value was omitted. A true HasValue with a nil Value is null.
	HasValue bool
	Value    any
}

// StudioTemporalConsumer is one graph edge that consumed an executed temporal record.
type StudioTemporalConsumer struct {
	Step     string
	Producer protocol.ProducerRef
	Input    string
	// Role is projection or domain.
	// Projection plumbing is classified by Studio's graph reader, never by a Companion name guess.
	Role   string
	Inputs []StudioTemporalConsumerInput
}

// StudioTemporalBinding is one executed Instant or Window and the graph edges that consumed that exact record.
type StudioTemporalBinding struct {
	Record     string
	SubjectID  string
	ID         string
	Projection StudioTemporalProjection
	Consumers  []StudioTemporalConsumer
}

// StudioParameterControls is the finite Studio parameter control list.
var StudioParameterControls = []string{
	"text", "number", "boolean", "select", "color", "list", "record",
}

// StudioParameterControl is one of StudioParameterControls.
type StudioParameterControl = string

// StudioParameterLanguage is svml, svs, or svrun.
type StudioParameterLanguage = string

// StudioParameterOptionPreview is a color or font preview hint.
type StudioParameterOptionPreview struct {
	// Kind is color or font.
	Kind string
	// Color is set when Kind is color.
	Color string
	// Family is set when Kind is font.
	Family string
	Sample *string
}

// StudioParameterOption is a bare string or a structured choice.
// Text non-nil means the bare string, and the other fields are unused.
// Data-only choices: packages supply meaning; Studio owns all option rendering.
type StudioParameterOption struct {
	Text        *string
	Value       any
	Label       string
	Description *string
	Preview     *StudioParameterOptionPreview
}

// StudioNumberPresentation scales an authored number for display.
// Displayed number = authored number × scale. Suffixes are retained, never converted.
// A nil slice or pointer means that property was omitted.
type StudioNumberPresentation struct {
	Scale    *float64
	Suffixes []string
	Minimum  *float64
	Maximum  *float64
	Step     *float64
}

// StudioAttributeGroupEdit is an attribute range relative to the owning element's source preimage.
// A nil Ranges value means that attribute's range is null.
type StudioAttributeGroupEdit struct {
	InsertionOffset float64
	Ranges          map[string]*Range
}

// StudioAuthorSource is one real author endpoint inside a binding.
type StudioAuthorSource struct {
	// Endpoint is the exact compilation-local author endpoint when the Frontend supplied one.
	Endpoint *string
	Path     string
	Range    Range
	Preimage string
	// Prefix and Suffix are generic syntax framing used only when an absent author property is first materialized.
	Prefix *string
	Suffix *string
}

// StudioSourceBinding is a real author endpoint. Its existence never implies Inspector visibility.
type StudioSourceBinding struct {
	ID      string
	Binding string
	// Name is the author vocabulary name at the terminal source element.
	Name  string
	Value any
	// Schema is the public author value structure, absent only for legacy scalar bindings.
	Schema         *protocol.ValueSchema
	Language       string
	Writable       bool
	Source         StudioAuthorSource
	Attributes     *StudioAttributeGroupEdit
	DisabledReason *string
}

// StudioInspectorDomain is where, when, or how.
type StudioInspectorDomain = string

// StudioInspectorPageDeclaration is one Inspector page.
type StudioInspectorPageDeclaration struct {
	ID    string
	Label string
}

// StudioInspectorSectionDeclaration is one Inspector section.
type StudioInspectorSectionDeclaration struct {
	ID    string
	Label string
}

// StudioInspectorFieldDeclaration is one visible field selected from a Companion's source bindings.
type StudioInspectorFieldDeclaration struct {
	Binding string
	Label   string
	Domain  string
	// Page is omitted for a domain with one unlabelled page.
	Page    *StudioInspectorPageDeclaration
	Section StudioInspectorSectionDeclaration
	Summary *string
	// Control is omitted when the binding's public schema selects the finite Studio control.
	Control   *string
	Options   []StudioParameterOption
	Unit      *string
	Number    *StudioNumberPresentation
	Multiline *bool
	// Swatches are optional suggested colors, independent of the accepted color value.
	Swatches []string
}

// StudioInspectorValue is a selected Item's package-owned fact, with no author write endpoint.
type StudioInspectorValue struct {
	Label   string
	Domain  string
	Page    *StudioInspectorPageDeclaration
	Section StudioInspectorSectionDeclaration
	Summary *string
	Unit    *string
	ID      string
	Value   any
}

// StudioInspectorEdit is the author endpoint attached to a resolved Inspector field.
type StudioInspectorEdit struct {
	Language   string
	Source     StudioAuthorSource
	Attributes *StudioAttributeGroupEdit
}

// StudioInspectorField is a resolved Inspector field.
// Display and edit authority are independent of the Where / When / How grouping.
type StudioInspectorField struct {
	Label     string
	Domain    string
	Page      *StudioInspectorPageDeclaration
	Section   StudioInspectorSectionDeclaration
	Summary   *string
	Options   []StudioParameterOption
	Unit      *string
	Number    *StudioNumberPresentation
	Multiline *bool
	Swatches  []string
	Binding   *string
	ID        string
	Control   string
	Value     any
	Schema    *protocol.ValueSchema
	Edit      *StudioInspectorEdit
}

// StudioFallbackFunc computes a typed fallback from authored properties.
// Returning protocol.Undefined means the function returned undefined.
// Returning nil means null.
type StudioFallbackFunc func(authored map[string]any) any

// StudioFallback is a canonical value or a function of authored properties.
// Func non-nil is the function form. Otherwise Value is the canonical value, and a nil Value is null.
type StudioFallback struct {
	Func  StudioFallbackFunc
	Value any
}

// StudioSourceBindingDeclaration is a Companion-owned source allowlist. It contains no editor presentation.
type StudioSourceBindingDeclaration struct {
	Name     string
	Writable *bool
	Schema   *protocol.ValueSchema
	// Fallback is the typed value used when this editable attribute is omitted.
	// Nil means the property was omitted.
	Fallback *StudioFallback
	// Companion composes the referenced object's package-owned parameter Companion.
	Companion *bool
	// Attributes lists scalar attributes edited together as one schema-described record.
	Attributes []string
	// Referenced is an optional declaration for the authored element named by a reference.
	Referenced []StudioSourceBindingDeclaration
	// Recipe is the Companion-owned source allowlist for the SVS Recipe reached through this reference.
	// Inspector presentation is declared separately.
	Recipe *StudioRecipeReferenceBindingDeclaration
}

// StudioRecipeReferenceBindingDeclaration is an explicit reference path from one author input to one SVS Recipe.
type StudioRecipeReferenceBindingDeclaration struct {
	// Through lists reference-valued attributes followed on local authored elements, in order.
	Through []string
	// Bindings is the package-owned source allowlist for properties of the reached Recipe.
	Bindings []StudioRecipeBindingDeclaration
}

// StudioRecipeBindingDeclaration is one property of a reached SVS Recipe.
type StudioRecipeBindingDeclaration struct {
	Name     string
	Writable *bool
	// Schema is the domain-owned public Recipe value structure, never editor presentation.
	Schema *protocol.ValueSchema
	// Fallback is the typed public fallback. Studio writes the property only after the author changes it.
	// Nil means the property was omitted.
	Fallback *StudioFallback
}

// StudioTimelineGesture is move, trim-start, or trim-end.
type StudioTimelineGesture = string

// StudioEditCoordinate is program-frame, domain-anchor, source-frame, canvas-pixel, or normalized-progress.
type StudioEditCoordinate = string

// StudioSnapTarget is frame, domain-anchor, or item-edge.
type StudioSnapTarget = string

// StudioEditSourceRole is start, end, duration, frame, x, or y.
type StudioEditSourceRole = string

// StudioEditSource is one author endpoint consumed by a timeline gesture.
type StudioEditSource struct {
	Role   string
	Source StudioAuthorSource
}

// StudioTemporalDomainEditTarget is the package-owned temporal item adjusted by a rectangle.
// Kind is span or point.
type StudioTemporalDomainEditTarget struct {
	Kind          string
	Companion     string
	DomainID      string
	ItemID        string
	StartAnchorID string
	EndAnchorID   string
	AnchorID      string
}

// StudioEditHandle is a timeline affordance. It is present only when its source write is explicit.
type StudioEditHandle struct {
	ID        string
	Operation string
	Gesture   string
	Enabled   bool
	// Coordinate is the space in which the central gesture resolver measures intent.
	Coordinate *string
	// MoveEffect is how moving a domain Point changes the visible Item before recompilation.
	// It is translate-window or move-start.
	MoveEffect *string
	// SnapTo is data, not a timeline-wide guess.
	SnapTo  []string
	Sources []StudioEditSource
	// Domain is the package-owned temporal-domain identity adjusted by this rectangle.
	Domain *StudioTemporalDomainEditTarget
	// Temporal is the exact endpoint authority used to validate and execute this inverse.
	Temporal *StudioTemporalProjection
	// TemporalRecord is the exact executed Temporal value traversed by the server-side inverse planner.
	TemporalRecord *string
	DisabledReason *string
}

// StudioPreviewSource is a source descriptor. Companion packages never depend on Studio's HTTP routes.
// Kind is artifact or surface-preview.
type StudioPreviewSource struct {
	Kind     string
	Resource string
	Module   string
	Version  string
	Surface  string
}

// StudioMaterialPreview is an image, video, or audio preview.
type StudioMaterialPreview struct {
	Kind   string
	Source StudioPreviewSource
}

// StudioDisplayLayer is one finite, composable timeline body layer owned by Studio.
// Kind is text or preview. Text uses Role "content". Preview Role is decoration or content.
// Layout is repeat-x, cover, contain, storyboard, or waveform.
type StudioDisplayLayer struct {
	Kind    string
	Role    string
	Text    string
	Preview StudioMaterialPreview
	Layout  string
}

// StudioItemDisplay is the title and ordered body of one timeline item.
type StudioItemDisplay struct {
	// Title is a header in standard chrome, and the complete single-line content in compact chrome.
	Title string
	// Layers are ordered back-to-front body layers.
	Layers []StudioDisplayLayer
}

// StudioLaneDescription is the height and grouping of one timeline lane.
type StudioLaneDescription struct {
	// HeightPx is the height of this single timeline lane; overlapping Items share it.
	HeightPx   float64
	GroupID    *string
	AttachedTo *string
	Order      *float64
}

// StudioCandidateProvenance is how a candidate value was obtained.
// Origin is run, source, or none. Status is resolved or unresolved.
type StudioCandidateProvenance struct {
	Output      string
	OutputRef   *string
	CandidateID *string
	Origin      string
	Status      string
	Errors      []string
}

// StudioTemporalDomainAnchor is one domain anchor on the shared timeline.
type StudioTemporalDomainAnchor struct {
	ID    string
	Frame float64
	// Kind is the domain-owned classification used only for display and stable coincident-point ordering.
	Kind   string
	Label  *string
	Detail *string
	Range  *Range
}

// StudioTemporalDomainLane is one lane inside a domain view.
type StudioTemporalDomainLane struct {
	ID       string
	Label    *string
	HeightPx float64
}

// StudioTemporalDomainItemSource matches the domain-owned value carried by Temporal lineage.
type StudioTemporalDomainItemSource struct {
	Type protocol.TypeRef
	Kind string
	ID   string
}

// StudioTemporalDomainItem is a span or a point on a domain lane.
// Kind "span" uses Appearance block or compact, the anchor ids, and the frame window.
// Kind "point" uses Appearance marker, AnchorID, and Frame.
type StudioTemporalDomainItem struct {
	ID     string
	LaneID string
	Label  string
	Range  *Range
	Source *StudioTemporalDomainItemSource
	// Editable allows the package Companion to receive an inverse edit for this item.
	Editable *bool
	// FollowPlayhead lets the code pane follow this item's source range while the playhead crosses it.
	FollowPlayhead    *bool
	Kind              string
	Appearance        string
	StartAnchorID     string
	EndAnchorID       string
	StartFrame        float64
	EndFrameExclusive float64
	AnchorID          string
	Frame             float64
}

// StudioTemporalDomainViewSource is the exact package-owned author source used only for inverse dispatch.
type StudioTemporalDomainViewSource struct {
	Path    string
	Content Range
}

// StudioTemporalDomainPresentation is the shell of one domain view.
type StudioTemporalDomainPresentation struct {
	Family string
	Tone   string
	Label  *string
	Icon   string
}

// StudioTemporalDomainView is one package projection onto the shared Timeline.
type StudioTemporalDomainView struct {
	// ID is the exact projected-view identity supplied by its package, such as one Projection id.
	ID string
	// Companion is the qualified Companion identity; together with ID this is globally unambiguous.
	Companion    string
	TimelineID   string
	Presentation StudioTemporalDomainPresentation
	Lanes        []StudioTemporalDomainLane
	Anchors      []StudioTemporalDomainAnchor
	// Items are visible on the shared Timeline.
	Items []StudioTemporalDomainItem
	// EditItems are domain-owned inverse targets retained for Track and Item editing but not drawn as Timeline rows.
	EditItems  []StudioTemporalDomainItem
	Provenance StudioCandidateProvenance
	Source     StudioTemporalDomainViewSource
}

// StudioObservedValue is one typed value observed on an authored element.
type StudioObservedValue struct {
	ID    string
	Type  protocol.TypeRef
	Value any
}

// StudioPlacementOutputPort is one public output port of a placement.
type StudioPlacementOutputPort struct {
	Name string
	Ref  string
}

// StudioPlacementChild is one authored child element observed for Studio.
type StudioPlacementChild struct {
	// AuthorElement is the exact compilation-local author element identity from AuthorProvenance.
	AuthorElement        *string
	AuthorEndpoints      map[string]string
	SourcePath           string
	Tag                  string
	ID                   *string
	Range                Range
	Attributes           map[string]string
	AttributeValueRanges map[string]Range
	References           []string
	ReferenceAttributes  map[string]string
	// ResolvedReferenceAttributes are resolved graph refs keyed by author input; raw written paths remain above.
	ResolvedReferenceAttributes map[string]string
	ReferenceTypes              map[string]string
	Values                      []StudioObservedValue
	Records                     []string
	Outputs                     []string
}

// StudioPlacement is one authored Surface element observed for Studio.
type StudioPlacement struct {
	// AuthorElement is the exact compilation-local author element identity from AuthorProvenance.
	AuthorElement        *string
	AuthorEndpoints      map[string]string
	SourcePath           string
	Tag                  string
	Module               protocol.ModuleRef
	Surface              string
	ID                   *string
	Range                Range
	Records              []string
	Values               []StudioObservedValue
	Outputs              []string
	OutputPorts          []StudioPlacementOutputPort
	Children             []StudioPlacementChild
	Attributes           map[string]string
	AttributeValueRanges map[string]Range
	ReferenceAttributes  map[string]string
	// ResolvedReferenceAttributes are resolved graph refs keyed by author input; raw written paths remain above.
	ResolvedReferenceAttributes map[string]string
	ReferenceTypes              map[string]string
	References                  []string
}

// StudioTraceOutputPort is one output port retained on a resolved track trace.
type StudioTraceOutputPort struct {
	Name    string
	Ref     string
	Type    *string
	TypeRef *protocol.TypeRef
}

// StudioTraceReference is one authored reference retained on a resolved track trace.
type StudioTraceReference struct {
	// Input is the exact authored input attribute when this is a direct Surface reference.
	Input   *string
	Name    string
	Ref     string
	Type    string
	TypeRef protocol.TypeRef
}

// StudioTrackTrace is the authoring trace of one resolved track.
type StudioTrackTrace struct {
	Placement   *string
	Surface     *string
	Module      *protocol.ModuleRef
	AuthoredID  *string
	OutputPorts []StudioTraceOutputPort
	References  []StudioTraceReference
}

// StudioResolvedTrack is the stable data view supplied to Companions.
// No Studio implementation object crosses the ABI.
type StudioResolvedTrack struct {
	Name string
	// Type is the short display name retained for diagnostics and UI only.
	Type string
	// TypeRef is the exact protocol identity used for every Companion decision.
	TypeRef         protocol.TypeRef
	OutputRef       string
	CandidateID     *string
	CandidateOrigin string
	Role            string
	Trace           StudioTrackTrace
	SurfacePreview  *StudioMaterialPreview
	Value           any
}

// StudioViewRole is timeline, supporting-value, or track.
type StudioViewRole = string

// StudioSpan is one rendered interval addressed by a public id.
type StudioSpan struct {
	ID string
	// SubjectID is the exact public terminal provenance, never inferred from id syntax.
	SubjectID         *string
	StartFrame        float64
	EndFrameExclusive float64
	StackOrder        float64
}

// StudioItemDraft is one timeline item projected by a Companion.
type StudioItemDraft struct {
	ID                string
	AuthoredID        string
	Display           StudioItemDisplay
	StartFrame        float64
	EndFrameExclusive float64
	// StackOrder is back-to-front timeline order. Ties preserve projection order. Selection raises only the timeline item.
	StackOrder   float64
	ElementRange *Range
	MarkerID     *string
	PresentID    *string
	// RenderIDs are the rendered parts belonging to this Item.
	RenderIDs []string
	// ParameterReferences are resolved author references that differ per derived Item, such as one Cue's actual Style.
	ParameterReferences map[string]string
	// SelectionGroup links disjoint displayed intervals belonging to one selectable author Item.
	SelectionGroup *string
	// Inspector holds deliberately selected facts, presented beside bound Inspector fields.
	Inspector    []StudioInspectorValue
	Presentation *StudioTimelinePresentation
	Temporal     *StudioTemporalLineage
	// Lane is an independent child Track partition; nil means this Track.
	Lane *string
}

// StudioInspectorObjectDraft is one non-timeline author object owned by a Track and presented in Inspector.
type StudioInspectorObjectDraft struct {
	ID                  string
	AuthoredID          string
	Title               string
	ElementRange        *Range
	ParameterReferences map[string]string
	Inspector           []StudioInspectorValue
}

// StudioTrackCompanionContext is the projection input for one resolved track.
type StudioTrackCompanionContext struct {
	Track     StudioResolvedTrack
	Placement *StudioPlacement
	// SurfacePreview is the package-owned Surface preview resolved by Studio from the selected domain.
	SurfacePreview *StudioMaterialPreview
	Spans          []StudioSpan
	Values         map[string]any
	// TemporalBindings are temporal values in this Track's actual executed dependency closure.
	TemporalBindings []StudioTemporalBinding
	TemporalDomains  []StudioTemporalDomainView
	Generic          func() []StudioItemDraft
}

// StudioInspectorObjectCompanion projects author objects that are not timeline occurrences.
type StudioInspectorObjectCompanion struct {
	ID        string
	Label     string
	Bindings  []StudioSourceBindingDeclaration
	Inspector []StudioInspectorFieldDeclaration
	Project   func(StudioTrackCompanionContext) []StudioInspectorObjectDraft
}

// StudioTrackOutput is the terminal output a Track Companion matches.
type StudioTrackOutput struct {
	Type        protocol.TypeRef
	Surface     *string
	Modules     []protocol.ModuleRef
	SiblingType *protocol.TypeRef
}

// StudioPoster opts root timeline Items into the component's package-owned Surface preview.
// Source is surface-preview.
type StudioPoster struct {
	Source string
}

// StudioTrackCompanion is one package-owned projection of a terminal track.
type StudioTrackCompanion struct {
	ID     string
	Role   string
	Output StudioTrackOutput
	Family *string
	// Tone is a visual token selected from Studio's finite palette.
	Tone        *string
	Label       *string
	Icon        *string
	Poster      *StudioPoster
	Attachments []StudioLaneAttachment
	// InspectorObjects are non-timeline author objects shown only in this Track's Inspector.
	InspectorObjects []StudioInspectorObjectCompanion
	// RequiredValues are same-Surface output values required to project this Track for Studio.
	RequiredValues []string
	Lane           *StudioLaneDescription
	// Bindings are the exact author endpoints required by Inspector fields or timeline inverses.
	Bindings []StudioSourceBindingDeclaration
	// Inspector is the visible field table. Undeclared bindings remain invisible.
	Inspector []StudioInspectorFieldDeclaration
	Project   func(StudioTrackCompanionContext) []StudioItemDraft
}

// StudioSurfaceMatch is a Module and Surface pair.
type StudioSurfaceMatch struct {
	Module  protocol.ModuleRef
	Surface string
}

// StudioParameterCompanion describes parameters owned by an authored object, independently of its consumers.
type StudioParameterCompanion struct {
	ID        string
	Match     StudioSurfaceMatch
	Bindings  []StudioSourceBindingDeclaration
	Inspector []StudioInspectorFieldDeclaration
}

// StudioCompanionContribution is the implementation of one Studio companion facet.
// Nil slices other than Tracks mean that property was omitted.
// Tracks nil means the property is not an array.
type StudioCompanionContribution struct {
	Format               string
	Tracks               []StudioTrackCompanion
	Films                []StudioFilmCompanion
	TemporalDomains      []StudioTemporalDomainCompanion
	TemporalDeclarations []StudioTemporalDeclarationCompanion
	TemporalRelations    []StudioTemporalRelationCompanion
	Parameters           []StudioParameterCompanion
}

// StudioFilmMatch is the declarative Film boundary match.
type StudioFilmMatch struct {
	Module     protocol.ModuleRef
	Surface    string
	OutputType protocol.TypeRef
}

// StudioFilmTimeSource is one reference the Film Companion allows Studio to follow.
type StudioFilmTimeSource struct {
	Attribute string
	Type      protocol.TypeRef
}

// StudioFilmTracks is the child Surface that contributes terminal tracks.
type StudioFilmTracks struct {
	ChildSurface    string
	SourceAttribute string
	Types           []protocol.TypeRef
}

// StudioFilmCompanion is a declarative Film boundary. Studio follows only the references named here.
type StudioFilmCompanion struct {
	ID          string
	Match       StudioFilmMatch
	TimeSources []StudioFilmTimeSource
	Tracks      StudioFilmTracks
}

// StudioTemporalDomainSourceMap is one observed domain source before projection.
type StudioTemporalDomainSourceMap struct {
	Companion  string
	DomainID   string
	SourcePath string
	Range      Range
	Content    Range
	// Data is an opaque package-owned observation; common Studio never interprets its fields.
	Data any
}

// StudioTemporalDomainAdjustment is a span or point inverse edit.
type StudioTemporalDomainAdjustment struct {
	Kind          string
	ItemID        string
	StartAnchorID string
	EndAnchorID   string
	AnchorID      string
}

// StudioTemporalDomainProjection is the part of a domain view produced by project.
type StudioTemporalDomainProjection struct {
	ID         string
	TimelineID string
	Lanes      []StudioTemporalDomainLane
	Anchors    []StudioTemporalDomainAnchor
	Items      []StudioTemporalDomainItem
	EditItems  []StudioTemporalDomainItem
}

// StudioFrameRate is the rational frame rate of a projection timeline.
type StudioFrameRate struct {
	Numerator   float64
	Denominator float64
}

// StudioTemporalProjectionTimeline is the absolute timeline passed to a domain projector.
type StudioTemporalProjectionTimeline struct {
	ID         string
	FrameRate  StudioFrameRate
	FrameCount float64
}

// StudioTemporalDomainProjectionInput is the executed input of a domain projector.
type StudioTemporalDomainProjectionInput struct {
	Source   StudioTemporalDomainSourceMap
	Values   []StudioObservedValue
	Timeline StudioTemporalProjectionTimeline
}

// StudioTemporalIdentifyInput is the value a domain Companion may claim.
type StudioTemporalIdentifyInput struct {
	Type  protocol.TypeRef
	Value any
}

// StudioTemporalDomainIdentity is the package-owned identity of one domain value.
type StudioTemporalDomainIdentity struct {
	DomainID string
	Kind     string
	ID       string
}

// StudioTemporalObserveInput is one authored element offered to a domain observer.
type StudioTemporalObserveInput struct {
	SourceName string
	// Source is raw Surface text. Structured Surfaces can rely on their decoded attributes and range.
	Source *string
	Tag    string
	// Range is the exact whole element range for raw and structured Surfaces alike.
	Range Range
	// ContentStart is a body boundary raw grammars may expose for package-owned parsing.
	ContentStart *float64
	// NextOffset is the authoritative end returned by the raw Surface after it parsed its own grammar.
	NextOffset *float64
	Attributes map[string]any
}

// StudioTemporalDomainObservation is a source map without the qualified companion id.
type StudioTemporalDomainObservation struct {
	DomainID   string
	SourcePath string
	Range      Range
	Content    Range
	Data       any
}

// StudioTemporalAdjustInput is one inverse edit of a domain's own source.
type StudioTemporalAdjustInput struct {
	SourceName string
	Source     string
	Adjustment StudioTemporalDomainAdjustment
}

// StudioTemporalDomainCompanion projects its own temporal facts and receives its own inverse edits.
type StudioTemporalDomainCompanion struct {
	ID    string
	Match StudioSurfaceMatch
	// ValueTypes are inline values required to build the domain view after execution.
	ValueTypes []protocol.TypeRef
	// SourceTypes are domain reference Types that may appear in executed Temporal lineage.
	SourceTypes  []protocol.TypeRef
	Presentation StudioTemporalDomainPresentation
	// Identify returns nil when the value is not this domain.
	Identify func(StudioTemporalIdentifyInput) *StudioTemporalDomainIdentity
	// Observe returns nil when the element is not this domain's source.
	Observe func(StudioTemporalObserveInput) *StudioTemporalDomainObservation
	Project func(StudioTemporalDomainProjectionInput) []StudioTemporalDomainProjection
	Adjust  func(StudioTemporalAdjustInput) string
}

// StudioTemporalDeclarationDraft is one package-owned author declaration whose primary result is an absolute Instant.
type StudioTemporalDeclarationDraft struct {
	ID    string
	Label *string
	// Output is the exact qualified logical output selected by the owning Surface Companion.
	Output string
	Range  *Range
}

// StudioTemporalDeclarationInput is the placement a declaration Companion reads.
type StudioTemporalDeclarationInput struct {
	Placement StudioPlacement
}

// StudioTemporalDeclarationCompanion declares author-visible temporal outputs without choosing their common Studio presentation.
type StudioTemporalDeclarationCompanion struct {
	ID      string
	Match   StudioSurfaceMatch
	Project func(StudioTemporalDeclarationInput) []StudioTemporalDeclarationDraft
}

// StudioTemporalConstraint is a desired value propagated while Studio reverses one executed temporal author graph.
// Kind is instant, extent, or span.
type StudioTemporalConstraint struct {
	Kind              string
	Frame             float64
	FrameCount        float64
	StartFrame        float64
	EndFrameExclusive float64
}

// StudioTemporalRelationValue is one executed input or output of a temporal relation.
type StudioTemporalRelationValue struct {
	Record string
	Type   protocol.TypeRef
	Value  any
}

// StudioTemporalInputConstraint assigns a desired constraint to one producer input.
type StudioTemporalInputConstraint struct {
	Input  string
	Target StudioTemporalConstraint
}

// StudioTemporalInputWrite replaces one author binding with an exact string.
type StudioTemporalInputWrite struct {
	Input       string
	Binding     string
	Replacement string
}

// StudioTemporalRelationInversePlan is one local inverse plan.
// Nil Constraints or Writes means that property was omitted.
type StudioTemporalRelationInversePlan struct {
	Constraints []StudioTemporalInputConstraint
	Writes      []StudioTemporalInputWrite
}

// StudioTemporalExtentTrace is a reconstructed extent.
// Authority nil means the property was omitted. When set, it is the parameter arm.
type StudioTemporalExtentTrace struct {
	Kind       string
	FrameCount float64
	Authority  *StudioTemporalAuthority
}

// StudioTemporalRelationTrace is a projection or an extent trace.
// Kind is instant, window, or extent.
// An extent uses FrameCount. Its Authority is absent when Authority.Kind is empty.
// Instant and window use the same fields as StudioTemporalProjection.
type StudioTemporalRelationTrace struct {
	Kind              string
	Expression        string
	Reference         string
	Frame             float64
	Source            StudioTemporalSource
	Authority         StudioTemporalAuthority
	Start             StudioTemporalInstantProjection
	End               StudioTemporalInstantProjection
	StartFrame        float64
	EndFrameExclusive float64
	FrameCount        float64
}

// StudioTemporalRelationMatch selects one Producer output.
type StudioTemporalRelationMatch struct {
	Producer protocol.ProducerRef
	Output   string
}

// StudioTemporalRelationInvertInput is the executed relation and the desired constraint.
type StudioTemporalRelationInvertInput struct {
	Target StudioTemporalConstraint
	Output StudioTemporalRelationValue
	Inputs map[string]StudioTemporalRelationValue
	// Desired is the constraint already imposed on a shared input by another output path.
	// A nil result means undefined.
	Desired func(input string) *StudioTemporalConstraint
}

// StudioTemporalRelationIdentity is the domain item identified from one relation input.
type StudioTemporalRelationIdentity struct {
	Companion string
	DomainID  string
	Kind      string
	ItemID    string
}

// StudioTemporalRelationTraceInput reconstructs lineage from exact input records.
type StudioTemporalRelationTraceInput struct {
	Output StudioTemporalRelationValue
	Inputs map[string]StudioTemporalRelationValue
	// Trace returns nil when that input has no relation trace.
	Trace func(input string) *StudioTemporalRelationTrace
	// Identify returns nil when that input is not a domain item.
	Identify func(input string) *StudioTemporalRelationIdentity
}

// StudioTemporalRelationCompanion is the package-owned inverse semantics for one Producer relation.
type StudioTemporalRelationCompanion struct {
	ID     string
	Match  StudioTemporalRelationMatch
	Invert func(StudioTemporalRelationInvertInput) []StudioTemporalRelationInversePlan
	// Trace reconstructs package-owned lineage without exposing private Producer vocabulary to Studio.
	// A nil Trace means the property was omitted. A nil result means undefined.
	Trace func(StudioTemporalRelationTraceInput) *StudioTemporalRelationTrace
}

// StudioLaneAttachment is an additional lane owned by a Track Companion.
type StudioLaneAttachment struct {
	ID        string
	Family    string
	Tone      *string
	Label     *string
	Icon      string
	Facet     string
	Lane      StudioLaneDescription
	Bindings  []StudioSourceBindingDeclaration
	Inspector []StudioInspectorFieldDeclaration
}

// StudioPackageContribution is every qualified companion collected from one physical package.
type StudioPackageContribution struct {
	Tracks               []StudioTrackCompanion
	Films                []StudioFilmCompanion
	TemporalDomains      []StudioTemporalDomainCompanion
	TemporalDeclarations []StudioTemporalDeclarationCompanion
	TemporalRelations    []StudioTemporalRelationCompanion
	Parameters           []StudioParameterCompanion
}

// StudioCompanionFacetInput is the object passed to CreateStudioCompanionFacet.
// A nil slice means that property was omitted.
type StudioCompanionFacetInput struct {
	Tracks               []StudioTrackCompanion
	Films                []StudioFilmCompanion
	TemporalDomains      []StudioTemporalDomainCompanion
	TemporalDeclarations []StudioTemporalDeclarationCompanion
	TemporalRelations    []StudioTemporalRelationCompanion
	Parameters           []StudioParameterCompanion
}

// StudioChildItem is one projected domain item passed to ChildItems.
type StudioChildItem struct {
	// ID is the exact identity of the projected domain item consumed by Temporal and renderer bindings.
	ID string
	// SubjectID is the exact author-owned Item realized by this projected item, when the identities differ.
	SubjectID         *string
	StartFrame        float64
	EndFrameExclusive float64
	StackOrder        float64
	// SourceTypes are the exact domain Spec types that may carry an omitted Source id.
	SourceTypes []protocol.TypeRef
}
