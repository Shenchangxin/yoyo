// Copyright 2026 Hypit.AI
// Translated from @hypit/result/src/writer.ts (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package result

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// BuildResultWriterState is the private record of an unfinished Result.
type BuildResultWriterState struct {
	Resources          map[string]string             `json:"resources"`
	Values             map[string]string             `json:"values"`
	PublishedOutputs   []BuildResultPublishedOutput  `json:"publishedOutputs"`
	ResourceReferences map[string]BuildResultFileRef `json:"resourceReferences,omitempty"`
	Forwards           []BuildResultForward          `json:"forwards"`
}

func (s BuildResultWriterState) normalized() BuildResultWriterState {
	if s.Resources == nil {
		s.Resources = map[string]string{}
	}
	if s.Values == nil {
		s.Values = map[string]string{}
	}
	if s.PublishedOutputs == nil {
		s.PublishedOutputs = []BuildResultPublishedOutput{}
	}
	if s.Forwards == nil {
		s.Forwards = []BuildResultForward{}
	}
	return s
}

// BuildResultWriteTarget writes bytes and value documents at paths chosen by the encoder.
type BuildResultWriteTarget interface {
	WriteResource(path string, artifact protocol.BlobRef, source BuildResultResourceSource) error
	WriteValue(path string, document BuildResultValueDocument) error
}

// BuildResultSyncResult is one publication pass.
type BuildResultSyncResult struct {
	Changed  bool
	Manifest BuildResultManifest
	Writer   BuildResultWriterState
}

// BuildResultSyncInput is the encoder input shared by repository adapters.
type BuildResultSyncInput struct {
	Manifest BuildResultManifest
	Writer   BuildResultWriterState
	Sync     BuildResultSync
	Target   BuildResultWriteTarget
}

type pathAllocator struct {
	directory string
	stem      string
	next      int64
	occupied  map[string]struct{}
}

var numberedPathPattern = regexp.MustCompile(`^(\d+)\.[^/]+$`)
var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)
var edgeDash = regexp.MustCompile(`^-+|-+$`)

func numberedPathAllocator(directory, stem string, paths []string) *pathAllocator {
	occupied := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		occupied[path] = struct{}{}
	}
	prefix := directory + "/" + stem + "-"
	var highest int64
	for path := range occupied {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		match := numberedPathPattern.FindStringSubmatch(path[len(prefix):])
		if match == nil {
			continue
		}
		index, ok := parseSafeIndex(match[1])
		if !ok {
			panicErr("Build Result path " + path + " has an unsafe generated index")
		}
		if index > highest {
			highest = index
		}
	}
	if highest >= maxSafeInteger {
		panicErr("Build Result " + directory + " path index is exhausted")
	}
	return &pathAllocator{directory: directory, stem: stem, next: highest + 1, occupied: occupied}
}

func parseSafeIndex(raw string) (int64, bool) {
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return safeInt(parsed)
}

func (a *pathAllocator) nextPath(extension string) string {
	for {
		if a.next > maxSafeInteger {
			panicErr("Build Result " + a.directory + " path index is exhausted")
		}
		candidate := fmt.Sprintf("%s/%s-%04d%s", a.directory, a.stem, a.next, extension)
		a.next++
		if _, exists := a.occupied[candidate]; exists {
			continue
		}
		a.occupied[candidate] = struct{}{}
		return candidate
	}
}

func runBounded(count, concurrency int, perform func(int) error) error {
	if count == 0 {
		return nil
	}
	if concurrency > count {
		concurrency = count
	}
	var mu sync.Mutex
	next := 0
	failed := false
	var failure error
	var failurePanic any
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for worker := 0; worker < concurrency; worker++ {
		go func() {
			defer wg.Done()
			for {
				mu.Lock()
				if failed {
					mu.Unlock()
					return
				}
				index := next
				next++
				mu.Unlock()
				if index >= count {
					return
				}
				func() {
					defer func() {
						if recovered := recover(); recovered != nil {
							mu.Lock()
							if !failed {
								failed = true
								failurePanic = recovered
							}
							mu.Unlock()
						}
					}()
					if err := perform(index); err != nil {
						mu.Lock()
						if !failed {
							failed = true
							failure = err
						}
						mu.Unlock()
					}
				}()
			}
		}()
	}
	wg.Wait()
	if failurePanic != nil {
		panic(failurePanic)
	}
	return failure
}

func mediaExtension(mediaType string) string {
	parts := strings.SplitN(mediaType, "/", 2)
	if len(parts) < 2 {
		return ".bin"
	}
	subtype := parts[1]
	if semi := strings.IndexByte(subtype, ';'); semi >= 0 {
		subtype = subtype[:semi]
	}
	subtype = strings.ToLower(strings.TrimSpace(subtype))
	if subtype == "" {
		return ".bin"
	}
	conventional := subtype
	switch subtype {
	case "jpeg":
		conventional = "jpg"
	case "x-wav":
		conventional = "wav"
	}
	if plus := strings.IndexByte(conventional, '+'); plus >= 0 {
		conventional = conventional[:plus]
	}
	safe := edgeDash.ReplaceAllString(nonAlnum.ReplaceAllString(conventional, "-"), "")
	if safe == "" {
		return ".bin"
	}
	return "." + safe
}

func asBlobRef(value any) (protocol.BlobRef, bool) {
	switch item := value.(type) {
	case protocol.BlobRef:
		if blobRefOK(item) {
			return item, true
		}
	case map[string]any:
		kind, _ := item["kind"].(string)
		resource, _ := item["resource"].(string)
		media, _ := item["mediaType"].(string)
		size, ok := asSafeNonNegative(item["size"])
		if kind == "blob" && resource != "" && ok && media != "" {
			return protocol.BlobRef{
				Kind:      "blob",
				Resource:  protocol.ResourceId(resource),
				Size:      size,
				MediaType: media,
			}, true
		}
	}
	return protocol.BlobRef{}, false
}

func blobRefOK(value protocol.BlobRef) bool {
	return value.Kind == "blob" && value.Resource != "" && value.Size >= 0 && value.Size <= maxSafeInteger && value.MediaType != ""
}

func outputRecord(input BuildResultSync, output string) *protocol.TypedRecord {
	var recordID string
	found := false
	for _, binding := range input.State.Plan.OutputBindings {
		if binding.Output == output {
			recordID = binding.Record
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	for index := range input.State.Records {
		if input.State.Records[index].ID == recordID {
			return &input.State.Records[index]
		}
	}
	return nil
}

type readyOutput struct {
	published BuildResultPublishedOutput
	typ       protocol.TypeRef
	record    *protocol.TypedRecord
	forward   *HistoricalBuildOutputRef
}

func outputRank(record *protocol.TypedRecord, forward *HistoricalBuildOutputRef) int {
	if forward != nil {
		return 0
	}
	if record != nil && record.Value.Kind == "blob" {
		return 1
	}
	if record != nil && scalar(record.Value.Value) {
		return 2
	}
	return 3
}

func appendPath(path []any, segment any) []any {
	next := make([]any, len(path)+1)
	copy(next, path)
	next[len(path)] = segment
	return next
}

// SyncBuildResultOutputs encodes accepted public Outputs once.
// The target only writes bytes and documents at the paths chosen here.
func SyncBuildResultOutputs(input BuildResultSyncInput) (BuildResultSyncResult, error) {
	resources := cloneStringMap(input.Writer.Resources)
	values := cloneStringMap(input.Writer.Values)
	filePaths := numberedPathAllocator("files", "file", stringMapValues(resources))
	valuePaths := numberedPathAllocator("values", "value", stringMapValues(values))
	outputs := cloneOutputs(input.Manifest.Outputs)
	forwards := map[string]BuildResultForward{}
	for _, forward := range input.Writer.Forwards {
		forwards[forward.Output] = forward
	}

	materialize := func(artifacts []protocol.BlobRef) ([]BuildResultFileRef, error) {
		type pendingWrite struct {
			path     string
			artifact protocol.BlobRef
		}
		var writes []pendingWrite
		result := make([]BuildResultFileRef, len(artifacts))
		for index, artifact := range artifacts {
			if reference, ok := input.Writer.ResourceReferences[string(artifact.Resource)]; ok {
				reference.MediaType = artifact.MediaType
				result[index] = reference
				continue
			}
			identity := string(artifact.Resource)
			if identity == "" {
				panicErr("Build resource has no instance identity")
			}
			path, exists := resources[identity]
			if !exists {
				path = filePaths.nextPath(mediaExtension(artifact.MediaType))
				resources[identity] = path
				writes = append(writes, pendingWrite{path: path, artifact: artifact})
			}
			result[index] = BuildResultFileRef{
				Kind:      "build-file",
				Path:      path,
				Size:      artifact.Size,
				MediaType: artifact.MediaType,
			}
		}
		err := runBounded(len(writes), 4, func(index int) error {
			return input.Target.WriteResource(writes[index].path, writes[index].artifact, input.Sync.Resources)
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	var encodeComposite func(any, []any, *[]struct {
		at       []any
		artifact protocol.BlobRef
	}) any
	encodeComposite = func(value any, path []any, artifacts *[]struct {
		at       []any
		artifact protocol.BlobRef
	}) any {
		if ref, ok := asBlobRef(value); ok {
			*artifacts = append(*artifacts, struct {
				at       []any
				artifact protocol.BlobRef
			}{path, ref})
			return nil
		}
		if scalar(value) {
			return value
		}
		switch item := value.(type) {
		case []any:
			out := make([]any, len(item))
			for index, child := range item {
				out[index] = encodeComposite(child, appendPath(path, index), artifacts)
			}
			return out
		case map[string]any:
			keys := make([]string, 0, len(item))
			for key := range item {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			out := make(map[string]any, len(item))
			for _, key := range keys {
				out[key] = encodeComposite(item[key], appendPath(path, key), artifacts)
			}
			return out
		default:
			panicErr("Build Output contains a non-canonical value")
			return nil
		}
	}

	outputValueOf := func(record protocol.TypedRecord) (BuildResultOutputValue, error) {
		if record.Value.Kind == "blob" {
			files, err := materialize([]protocol.BlobRef{{
				Kind:      "blob",
				Resource:  record.Value.Resource,
				Size:      record.Value.Size,
				MediaType: record.Value.MediaType,
			}})
			if err != nil {
				return BuildResultOutputValue{}, err
			}
			return fileOutput(files[0]), nil
		}
		if scalar(record.Value.Value) {
			return inlineOutput(record.Value.Value), nil
		}
		if existing, ok := values[record.ID]; ok {
			return BuildResultOutputValue{Kind: "value", Path: existing}, nil
		}
		path := valuePaths.nextPath(".json")
		values[record.ID] = path
		var artifacts []struct {
			at       []any
			artifact protocol.BlobRef
		}
		encoded := encodeComposite(record.Value.Value, nil, &artifacts)
		blobs := make([]protocol.BlobRef, len(artifacts))
		for index, artifact := range artifacts {
			blobs[index] = artifact.artifact
		}
		files, err := materialize(blobs)
		if err != nil {
			return BuildResultOutputValue{}, err
		}
		bindings := make([]BuildResultResourceBinding, len(artifacts))
		for index, artifact := range artifacts {
			bindings[index] = BuildResultResourceBinding{At: artifact.at, File: files[index]}
		}
		if err := input.Target.WriteValue(path, BuildResultValueDocument{
			Format:    "hypit.result-value@1",
			Value:     encoded,
			Resources: bindings,
		}); err != nil {
			return BuildResultOutputValue{}, err
		}
		return BuildResultOutputValue{Kind: "value", Path: path}, nil
	}

	ready := make([]readyOutput, 0)
	for _, published := range input.Writer.PublishedOutputs {
		if _, exists := outputs[published.Name]; exists {
			continue
		}
		if source, ok := forwards[published.Output]; ok {
			typ := protocol.TypeRef{}
			if source.Type != nil {
				typ = *source.Type
			}
			ready = append(ready, readyOutput{
				published: published,
				typ:       typ,
				forward: &HistoricalBuildOutputRef{
					Kind:   "build-output",
					Build:  source.Build,
					Output: source.SourceOutput,
				},
			})
			continue
		}
		record := outputRecord(input.Sync, published.Output)
		if record == nil {
			continue
		}
		ready = append(ready, readyOutput{published: published, typ: record.Type, record: record})
	}
	sort.SliceStable(ready, func(i, j int) bool {
		left := outputRank(ready[i].record, ready[i].forward)
		right := outputRank(ready[j].record, ready[j].forward)
		if left != right {
			return left < right
		}
		return ready[i].published.Name < ready[j].published.Name
	})
	if len(ready) == 0 {
		return BuildResultSyncResult{Changed: false, Manifest: input.Manifest, Writer: input.Writer}, nil
	}
	for _, item := range ready {
		var value BuildResultOutputValue
		if item.forward != nil {
			value = BuildResultOutputValue{Kind: "build-output", Build: item.forward.Build, Output: item.forward.Output}
		} else {
			var err error
			value, err = outputValueOf(*item.record)
			if err != nil {
				return BuildResultSyncResult{}, err
			}
		}
		output := BuildResultOutput{Type: item.typ, Value: value}
		if item.published.DisplayName != nil {
			output.DisplayName = item.published.DisplayName
		}
		outputs[item.published.Name] = output
	}
	manifest := input.Manifest
	manifest.Outputs = outputs
	writer := input.Writer
	writer.Resources = resources
	writer.Values = values
	return BuildResultSyncResult{Changed: true, Manifest: manifest, Writer: writer}, nil
}

func cloneOutputs(in map[string]BuildResultOutput) map[string]BuildResultOutput {
	out := make(map[string]BuildResultOutput, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
