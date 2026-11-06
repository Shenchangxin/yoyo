// Copyright 2026 Hypit.AI
// Translated from @hypit/result/src/types.ts (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package result

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// BuildResultOutcome is complete, failed, or cancelled.
type BuildResultOutcome = string

// BuildResultFileRef is one Result file or an explicit external file.
type BuildResultFileRef struct {
	Kind      string  `json:"kind"`
	Size      int64   `json:"size"`
	MediaType string  `json:"mediaType"`
	Build     *string `json:"build,omitempty"`
	Path      string  `json:"path,omitempty"`
	URI       string  `json:"uri,omitempty"`
}

// MarshalJSON omits the address fields that do not belong to this kind.
func (f BuildResultFileRef) MarshalJSON() ([]byte, error) {
	if f.Kind == "external-file" {
		return jsonMarshal(struct {
			Kind      string `json:"kind"`
			URI       string `json:"uri"`
			Size      int64  `json:"size"`
			MediaType string `json:"mediaType"`
		}{f.Kind, f.URI, f.Size, f.MediaType})
	}
	return jsonMarshal(struct {
		Kind      string  `json:"kind"`
		Path      string  `json:"path"`
		Size      int64   `json:"size"`
		MediaType string  `json:"mediaType"`
		Build     *string `json:"build,omitempty"`
	}{f.Kind, f.Path, f.Size, f.MediaType, f.Build})
}

// BuildResultFileRange is a half-open byte range within one Result file.
type BuildResultFileRange struct {
	Start        int64
	EndExclusive int64
}

// HistoricalBuildOutputRef points at a terminal Output of an earlier Build.
type HistoricalBuildOutputRef struct {
	Kind   string `json:"kind"`
	Build  string `json:"build"`
	Output string `json:"output"`
}

// BuildResultOutputValue is the persisted value of one public Output.
type BuildResultOutputValue struct {
	Kind   string
	File   *BuildResultFileRef
	Build  string
	Output string
	Inline any
	Path   string
}

// MarshalJSON writes one Result value kind and only its own fields.
func (v BuildResultOutputValue) MarshalJSON() ([]byte, error) {
	switch v.Kind {
	case "build-file", "external-file":
		if v.File == nil {
			return []byte("null"), nil
		}
		return jsonMarshal(v.File)
	case "build-output":
		return jsonMarshal(HistoricalBuildOutputRef{Kind: "build-output", Build: v.Build, Output: v.Output})
	case "inline":
		return jsonMarshal(struct {
			Kind  string `json:"kind"`
			Value any    `json:"value"`
		}{Kind: "inline", Value: v.Inline})
	case "value":
		return jsonMarshal(struct {
			Kind string `json:"kind"`
			Path string `json:"path"`
		}{Kind: "value", Path: v.Path})
	default:
		return nil, fmt.Errorf("%s", v.Kind)
	}
}

// BuildResultResourceBinding is one Resource slot inside a value document.
type BuildResultResourceBinding struct {
	At   []any              `json:"at"`
	File BuildResultFileRef `json:"file"`
}

// BuildResultValueDocument is the durable encoding of one composite value.
type BuildResultValueDocument struct {
	Format    string                       `json:"format"`
	Value     any                          `json:"value"`
	Resources []BuildResultResourceBinding `json:"resources"`
}

// BuildResultOutput is one public Output in a Result manifest.
type BuildResultOutput struct {
	DisplayName *string                `json:"displayName,omitempty"`
	Type        protocol.TypeRef       `json:"type"`
	Value       BuildResultOutputValue `json:"value"`
}

// BuildResultOperationNeed is the Need recorded on one operation receipt.
type BuildResultOperationNeed struct {
	ID         string           `json:"id"`
	Capability protocol.TypeRef `json:"capability"`
}

// BuildResultProgress is the last observed progress of one operation.
type BuildResultProgress struct {
	Phase     string   `json:"phase"`
	Completed *float64 `json:"completed,omitempty"`
	Total     *float64 `json:"total,omitempty"`
	Unit      *string  `json:"unit,omitempty"`
}

// BuildResultCancellation is the acknowledgement of a cancellation request.
type BuildResultCancellation struct {
	Outcome string  `json:"outcome"`
	Message *string `json:"message,omitempty"`
}

// BuildResultCredential names a credential without copying its secret.
type BuildResultCredential struct {
	Store string `json:"store"`
	Key   string `json:"key"`
}

// BuildResultReceipt is the remote handle of one operation.
type BuildResultReceipt struct {
	ID  string  `json:"id"`
	URL *string `json:"url,omitempty"`
}

// BuildResultOperationFailure is the remote failure of one operation.
type BuildResultOperationFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BuildResultOperation is one non-secret execution fact.
type BuildResultOperation struct {
	Need           *BuildResultOperationNeed        `json:"need,omitempty"`
	CreatedAt      *int64                           `json:"createdAt,omitempty"`
	AcknowledgedAt *int64                           `json:"acknowledgedAt,omitempty"`
	EndedAt        *int64                           `json:"endedAt,omitempty"`
	Progress       *BuildResultProgress             `json:"progress,omitempty"`
	Cancellation   *BuildResultCancellation         `json:"cancellation,omitempty"`
	Operation      string                           `json:"operation"`
	Command        string                           `json:"command"`
	Endpoint       string                           `json:"endpoint"`
	Status         string                           `json:"status"`
	Pool           *string                          `json:"pool,omitempty"`
	Credentials    map[string]BuildResultCredential `json:"credentials,omitempty"`
	Receipt        *BuildResultReceipt              `json:"receipt,omitempty"`
	Failure        *BuildResultOperationFailure     `json:"failure,omitempty"`
}

// BuildResultSource is an opaque Source or Run identity.
type BuildResultSource struct {
	ID string `json:"id"`
}

// BuildResultManifest is one Build Result document plus the repository's Build id.
type BuildResultManifest struct {
	Format             string                       `json:"format"`
	ID                 string                       `json:"id,omitempty"`
	Title              *string                      `json:"title,omitempty"`
	Note               *string                      `json:"note,omitempty"`
	HighlightedOutputs []string                     `json:"highlightedOutputs,omitempty"`
	Source             BuildResultSource            `json:"source"`
	Run                *BuildResultSource           `json:"run,omitempty"`
	Targets            []string                     `json:"targets"`
	FinishedAt         *int64                       `json:"finishedAt,omitempty"`
	Outcome            string                       `json:"outcome,omitempty"`
	Failure            *string                      `json:"failure,omitempty"`
	Outputs            map[string]BuildResultOutput `json:"outputs"`
	Operations         *[]BuildResultOperation      `json:"operations,omitempty"`
	ExecutionLog       *BuildResultFileRef          `json:"executionLog,omitempty"`
}

// FinishedBuildResultManifest is a manifest with an outcome and finishedAt.
type FinishedBuildResultManifest = BuildResultManifest

// BuildResultPublishedOutput is one public name for a Logical Output.
type BuildResultPublishedOutput struct {
	Name        string  `json:"name"`
	Output      string  `json:"output"`
	DisplayName *string `json:"displayName,omitempty"`
}

// BuildResultForward carries one Logical Output from an earlier Result.
type BuildResultForward struct {
	Output       string            `json:"output"`
	Build        string            `json:"build"`
	SourceOutput string            `json:"sourceOutput"`
	Type         *protocol.TypeRef `json:"type,omitempty"`
}

// RepositoryBuildResultOutputLocation is a terminal Output address.
type RepositoryBuildResultOutputLocation struct {
	Build  string
	Output string
	Type   protocol.TypeRef
}

// BuildResultSeed is the public surface recorded before storage is touched.
type BuildResultSeed struct {
	ID                 string
	Title              *string
	Source             BuildResultSource
	Run                *BuildResultSource
	Targets            []string
	PublishedOutputs   []BuildResultPublishedOutput
	Forwards           []BuildResultForward
	ResourceReferences map[string]BuildResultFileRef
}

// BuildResultResourceSource opens admitted Resource bytes.
type BuildResultResourceSource interface {
	Open(artifact protocol.BlobRef) (io.Reader, error)
}

// BuildResultSync is the Runtime view used to publish newly accepted Outputs.
type BuildResultSync struct {
	State     protocol.BuildState
	Resources BuildResultResourceSource
}

// BuildResultFinish closes one Result.
type BuildResultFinish struct {
	ExecutionLog io.Reader
	Operations   *[]BuildResultOperation
	Outcome      string
	Failure      *string
}

// BuildResultPresentationUpdate edits human presentation on one finished Result.
// A set flag distinguishes omission from an explicit null.
type BuildResultPresentationUpdate struct {
	OutputDisplayNames    map[string]*string
	SetOutputDisplayNames bool
	Title                 *string
	SetTitle              bool
	Note                  *string
	SetNote               bool
	HighlightedOutputs    []string
	SetHighlightedOutputs bool
}

// BuildResultBrowseRequest pages finished Results, newest first.
type BuildResultBrowseRequest struct {
	Before *string
	Limit  int
}

// BuildResultPage is one page of finished Results.
type BuildResultPage struct {
	Results []FinishedBuildResultManifest
	Next    *string
}

// ResolvedBuildResultValue is a terminal Output value, with a document when it is composite.
type ResolvedBuildResultValue struct {
	BuildResultOutputValue
	Document *BuildResultValueDocument
}

// ResolvedBuildResultOutput is one resolved public Output plus its owning directory.
type ResolvedBuildResultOutput struct {
	Build     string
	Output    string
	Directory string
	Type      protocol.TypeRef
	Value     ResolvedBuildResultValue
}

// RepositoryBuildResultOutput is a resolved Output without a directory.
type RepositoryBuildResultOutput struct {
	Build  string
	Output string
	Type   protocol.TypeRef
	Value  ResolvedBuildResultValue
}

// RepositoryBuildResultOutputDescription describes a terminal Output without opening it.
type RepositoryBuildResultOutputDescription struct {
	Type      protocol.TypeRef
	Kind      string
	Size      int64
	MediaType string
}

// BuildResultWriter publishes Outputs and the terminal outcome of one Build.
type BuildResultWriter interface {
	Read() (BuildResultManifest, error)
	Sync(input BuildResultSync) (BuildResultManifest, error)
	Finish(input BuildResultFinish) (BuildResultManifest, error)
}

// BuildResultReader reads one manifest by Build id.
type BuildResultReader interface {
	Read(build string) (*BuildResultManifest, error)
}

// AssertBuildResultPath panics unless value is a portable Result-relative path.
func AssertBuildResultPath(value any, subject string) {
	text, ok := value.(string)
	if !ok || !resultPathOK(text) {
		panicErr(subject + " is not a Result-relative path")
	}
}

func resultPathOK(value string) bool {
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, `\`) || strings.Contains(value, "\x00") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

// AssertBuildResultFileRef panics unless value is a Build or external file reference.
func AssertBuildResultFileRef(value any, subject string) {
	fileRefFromAny(value, subject)
}

func fileRefFromAny(value any, subject string) BuildResultFileRef {
	item, ok := asObject(value)
	if !ok {
		panicErr(subject + " is not a Build file reference")
	}
	kind, _ := item["kind"].(string)
	size, sizeOK := asSafeNonNegative(item["size"])
	media, _ := item["mediaType"].(string)
	if (kind != "build-file" && kind != "external-file") || !sizeOK || media == "" {
		panicErr(subject + " is not a valid Build file reference")
	}
	ref := BuildResultFileRef{Kind: kind, Size: size, MediaType: media}
	if kind == "external-file" {
		uri, _ := item["uri"].(string)
		if uri == "" {
			panicErr(subject + ".uri is missing")
		}
		assertURL(uri)
		ref.URI = uri
		return ref
	}
	AssertBuildResultPath(item["path"], subject+".path")
	ref.Path = item["path"].(string)
	if raw, exists := item["build"]; exists {
		build, ok := raw.(string)
		if !ok {
			protocol.AssertOrderedBuildId(fmt.Sprint(raw))
		}
		protocol.AssertOrderedBuildId(build)
		ref.Build = &build
	}
	return ref
}

func assertFileRef(file BuildResultFileRef, subject string) {
	if (file.Kind != "build-file" && file.Kind != "external-file") || file.Size < 0 || file.Size > maxSafeInteger || file.MediaType == "" {
		panicErr(subject + " is not a valid Build file reference")
	}
	if file.Kind == "external-file" {
		if file.URI == "" {
			panicErr(subject + ".uri is missing")
		}
		assertURL(file.URI)
		return
	}
	AssertBuildResultPath(file.Path, subject+".path")
	if file.Build != nil {
		protocol.AssertOrderedBuildId(*file.Build)
	}
}

func assertURL(raw string) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || !parsed.IsAbs() {
		panicErr("Invalid URL")
	}
}

func assertCanonicalValue(value any, subject string) {
	switch item := value.(type) {
	case nil, bool, string:
		return
	case float64:
		if !scalar(item) {
			panicErr(subject + " contains a non-finite number")
		}
	case float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		if !scalar(item) {
			panicErr(subject + " contains a non-finite number")
		}
	case []any:
		for index, child := range item {
			assertCanonicalValue(child, fmt.Sprintf("%s[%d]", subject, index))
		}
	case map[string]any:
		for key, child := range item {
			assertCanonicalValue(child, subject+"."+key)
		}
	default:
		panicErr(subject + " is not canonical data")
	}
}

func valueAtPath(value any, path []any, subject string) any {
	current := value
	for _, segment := range path {
		if text, ok := segment.(string); ok {
			item, ok := current.(map[string]any)
			child, exists := item[text]
			if !ok || !exists {
				panicErr(subject + " does not address a value slot")
			}
			current = child
			continue
		}
		index, ok := asSafeNonNegative(segment)
		items, isArray := current.([]any)
		if !ok || !isArray || index >= int64(len(items)) {
			panicErr(subject + " does not address a value slot")
		}
		current = items[index]
	}
	return current
}

// AssertBuildResultValueDocument panics unless value is a Result value document.
func AssertBuildResultValueDocument(value any, subject ...string) {
	sub := subjectOr("Build Result value", subject)
	item, ok := asObject(value)
	resources, isArray := []any(nil), false
	if ok {
		resources, isArray = item["resources"].([]any)
	}
	if !ok || item["format"] != "hypit.result-value@1" || !isArray {
		panicErr(sub + " is not a Result value document")
	}
	canonical := item["value"]
	assertCanonicalValue(canonical, sub+".value")
	occupied := map[string]struct{}{}
	for index, raw := range resources {
		binding, ok := asObject(raw)
		if !ok {
			panicErr(fmt.Sprintf("%s.resources[%d] is invalid", sub, index))
		}
		at, isArray := binding["at"].([]any)
		if !isArray || len(at) == 0 {
			panicErr(fmt.Sprintf("%s.resources[%d].at is invalid", sub, index))
		}
		for _, segment := range at {
			if _, ok := segment.(string); ok {
				continue
			}
			if _, ok := asSafeNonNegative(segment); !ok {
				panicErr(fmt.Sprintf("%s.resources[%d].at is invalid", sub, index))
			}
		}
		encoded := jsonStringify(at)
		if _, exists := occupied[encoded]; exists {
			panicErr(sub + " binds " + encoded + " more than once")
		}
		occupied[encoded] = struct{}{}
		atSubject := fmt.Sprintf("%s.resources[%d].at", sub, index)
		if valueAtPath(canonical, at, atSubject) != nil {
			panicErr(atSubject + " must address a null Resource slot")
		}
		AssertBuildResultFileRef(binding["file"], fmt.Sprintf("%s.resources[%d].file", sub, index))
	}
}

// AssertBuildResultSeed panics unless each public Output has one name.
func AssertBuildResultSeed(seed BuildResultSeed) {
	if strings.TrimSpace(seed.Source.ID) == "" {
		panicErr("Build Result Source identity must not be empty")
	}
	if seed.Run != nil && strings.TrimSpace(seed.Run.ID) == "" {
		panicErr("Build Result Run identity must not be empty")
	}
	if seed.Title != nil && strings.TrimSpace(*seed.Title) == "" {
		panicErr("Build Result title must not be empty")
	}
	for resource, file := range seed.ResourceReferences {
		if resource == "" {
			panicErr("Resource reference has no identity")
		}
		assertFileRef(file, "Resource "+resource)
		if file.Kind == "build-file" && file.Build == nil {
			panicErr("Resource " + resource + " needs its owning Build")
		}
	}
	names := map[string]struct{}{}
	outputs := map[string]struct{}{}
	for _, published := range seed.PublishedOutputs {
		if strings.TrimSpace(published.Name) == "" {
			panicErr("Published Output name must not be empty")
		}
		if strings.TrimSpace(published.Output) == "" {
			panicErr("Published Output " + published.Name + " has no Logical Output")
		}
		if _, exists := names[published.Name]; exists {
			panicErr("Published Output name " + published.Name + " is repeated")
		}
		if _, exists := outputs[published.Output]; exists {
			panicErr("Logical Output " + published.Output + " has more than one published name")
		}
		names[published.Name] = struct{}{}
		outputs[published.Output] = struct{}{}
	}
	targets := map[string]struct{}{}
	for _, target := range seed.Targets {
		if _, ok := names[target]; !ok {
			panicErr("Target " + target + " is not a published Output")
		}
		if _, exists := targets[target]; exists {
			panicErr("Target " + target + " is repeated")
		}
		targets[target] = struct{}{}
	}
	forwards := map[string]struct{}{}
	for _, forward := range seed.Forwards {
		if _, ok := outputs[forward.Output]; !ok {
			panicErr("Forwarded Logical Output " + forward.Output + " is not public")
		}
		if _, exists := forwards[forward.Output]; exists {
			panicErr("Logical Output " + forward.Output + " has more than one forward source")
		}
		if strings.TrimSpace(forward.Build) == "" || strings.TrimSpace(forward.SourceOutput) == "" {
			panicErr("Logical Output " + forward.Output + " has an incomplete forward source")
		}
		if forward.Build == seed.ID {
			panicErr("Logical Output " + forward.Output + " cannot forward to its own Build")
		}
		forwards[forward.Output] = struct{}{}
	}
}
