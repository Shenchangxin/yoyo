// Copyright 2026 Hypit.AI
// Translated from @hypit/result/test/result.test.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package result

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

var videoType = protocol.TypeRef{Module: protocol.ModuleRef{Name: "example.media", Version: "1"}, Name: "Video"}
var compositeType = protocol.TypeRef{Module: protocol.ModuleRef{Name: "example.speech", Version: "1"}, Name: "CompositeValue"}

type outputBinding struct {
	output string
	record string
}

func testState(records []protocol.TypedRecord, bindings []outputBinding) protocol.BuildState {
	out := make([]protocol.BuildOutputBinding, len(bindings))
	for index, binding := range bindings {
		typ := videoType
		for _, record := range records {
			if record.ID == binding.record {
				typ = record.Type
				break
			}
		}
		out[index] = protocol.BuildOutputBinding{Output: binding.output, Record: binding.record, Type: typ}
	}
	return protocol.BuildState{Status: "active", Records: records, Plan: protocol.BuildPlan{OutputBindings: out}}
}

func strPtr(value string) *string { return &value }

type resourceOpener func(protocol.BlobRef) (io.Reader, error)

func (f resourceOpener) Open(artifact protocol.BlobRef) (io.Reader, error) { return f(artifact) }

type recordingTarget struct {
	resources func(path string, artifact protocol.BlobRef) error
	values    func(path string) error
}

func (t recordingTarget) WriteResource(path string, artifact protocol.BlobRef, _ BuildResultResourceSource) error {
	if t.resources == nil {
		return nil
	}
	return t.resources(path, artifact)
}

func (t recordingTarget) WriteValue(path string, _ BuildResultValueDocument) error {
	if t.values == nil {
		return nil
	}
	return t.values(path)
}

func mustFail(t *testing.T, fn func() error, pattern string) {
	t.Helper()
	re := regexp.MustCompile(pattern)
	var recovered any
	err := func() (err error) {
		defer func() { recovered = recover() }()
		return fn()
	}()
	var message string
	switch {
	case recovered != nil:
		message = panicMessage(recovered)
	case err != nil:
		message = err.Error()
	default:
		t.Fatalf("expected failure matching %s", pattern)
	}
	if !re.MatchString(message) {
		t.Fatalf("failure %q does not match %s", message, pattern)
	}
}

func readHandle(t *testing.T, file *os.File) []byte {
	t.Helper()
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, info.Size())
	if info.Size() == 0 {
		return buf
	}
	if _, err := file.ReadAt(buf, 0); err != nil {
		t.Fatal(err)
	}
	return buf
}

func sortedOutputNames(outputs map[string]BuildResultOutput) []string {
	names := make([]string, 0, len(outputs))
	for name := range outputs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func TestInternalProgressLeavesResultFilesUntouched(t *testing.T) {
	root := t.TempDir()
	result, err := CreateFileBuildResult(root, BuildResultSeed{
		ID: "bld_20260902T100000000Z_0000000001", Source: BuildResultSource{ID: "/project/main.svml"},
		Targets:          []string{"answer"},
		PublishedOutputs: []BuildResultPublishedOutput{{Name: "answer", Output: "logical:answer", DisplayName: strPtr("Answer")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{filepath.Join(result.Directory, "result.json"), filepath.Join(result.Directory, ".writer.json")}
	resources := resourceOpener(func(protocol.BlobRef) (io.Reader, error) {
		return nil, errors.New("scalar Outputs have no Resource bytes")
	})
	snapshot := func(ready bool) protocol.BuildState {
		bindings := []outputBinding{}
		if ready {
			bindings = []outputBinding{{output: "logical:answer", record: "record:answer"}}
		}
		return testState([]protocol.TypedRecord{{
			ID: "record:answer", Type: videoType, Value: protocol.StoredValue{Kind: "inline", Value: 42},
		}}, bindings)
	}
	unchanged := func(ready bool) {
		t.Helper()
		past := time.Unix(1, 0)
		before := make([]os.FileInfo, len(paths))
		for index, path := range paths {
			if err := os.Chtimes(path, past, past); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			before[index] = info
		}
		if _, err := result.Sync(BuildResultSync{State: snapshot(ready), Resources: resources}); err != nil {
			t.Fatal(err)
		}
		for index, path := range paths {
			after, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if !os.SameFile(before[index], after) || !before[index].ModTime().Equal(after.ModTime()) {
				t.Fatalf("result file %s was rewritten", path)
			}
		}
	}
	unchanged(false)
	read, err := result.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Outputs) != 0 {
		t.Fatalf("outputs %#v", read.Outputs)
	}
	if _, err := result.Sync(BuildResultSync{State: snapshot(true), Resources: resources}); err != nil {
		t.Fatal(err)
	}
	read, err = result.Read()
	if err != nil {
		t.Fatal(err)
	}
	answer := read.Outputs["answer"]
	if answer.Value.Kind != "inline" || answer.Value.Inline != float64(42) {
		t.Fatalf("answer %#v", answer.Value)
	}
	if answer.DisplayName == nil || *answer.DisplayName != "Answer" {
		t.Fatalf("display name %#v", answer.DisplayName)
	}
	unchanged(true)
	failure := "a later Need failed"
	finished, err := result.Finish(BuildResultFinish{Outcome: "failed", Failure: &failure})
	if err != nil {
		t.Fatal(err)
	}
	if finished.Outcome != "failed" || finished.Outputs["answer"].DisplayName == nil || *finished.Outputs["answer"].DisplayName != "Answer" {
		t.Fatalf("%#v", finished)
	}
	if finished.Outputs["answer"].Value.Kind != "inline" || finished.Outputs["answer"].Value.Inline != float64(42) {
		t.Fatalf("%#v", finished.Outputs["answer"].Value)
	}
	entries, err := os.ReadDir(result.Directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == ".writer.json" {
			t.Fatal("writer state remains")
		}
	}
}

func TestPublishingPreservesOpenReaders(t *testing.T) {
	root := t.TempDir()
	result, err := CreateFileBuildResult(root, BuildResultSeed{
		ID: "bld_20260902T100000000Z_0000000001", Source: BuildResultSource{ID: "/project/main.svml"},
		Targets:          []string{"answer"},
		PublishedOutputs: []BuildResultPublishedOutput{{Name: "answer", Output: "logical:answer"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(result.Directory, "result.json")
	oldManifest, err := openShared(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	defer oldManifest.Close()
	writerPath := filepath.Join(result.Directory, ".writer.json")
	oldWriter, err := openShared(writerPath)
	if err != nil {
		t.Fatal(err)
	}
	defer oldWriter.Close()
	writerBytes, err := os.ReadFile(writerPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := result.Sync(BuildResultSync{
		State: testState([]protocol.TypedRecord{{
			ID: "record:answer", Type: videoType, Value: protocol.StoredValue{Kind: "inline", Value: 42},
		}}, []outputBinding{{output: "logical:answer", record: "record:answer"}}),
		Resources: resourceOpener(func(protocol.BlobRef) (io.Reader, error) {
			return nil, errors.New("unexpected Resource read")
		}),
	}); err != nil {
		t.Fatal(err)
	}
	var oldParsed map[string]any
	if err := json.Unmarshal(readHandle(t, oldManifest), &oldParsed); err != nil {
		t.Fatal(err)
	}
	outputs, _ := oldParsed["outputs"].(map[string]any)
	if len(outputs) != 0 {
		t.Fatalf("old manifest outputs %#v", outputs)
	}
	if !bytes.Equal(readHandle(t, oldWriter), writerBytes) {
		t.Fatal("old writer changed")
	}
	read, err := result.Read()
	if err != nil {
		t.Fatal(err)
	}
	if read.Outputs["answer"].Value.Kind != "inline" || read.Outputs["answer"].Value.Inline != float64(42) {
		t.Fatalf("%#v", read.Outputs["answer"].Value)
	}
	published, err := openShared(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	defer published.Close()
	if _, err := result.Finish(BuildResultFinish{Outcome: "complete"}); err != nil {
		t.Fatal(err)
	}
	var publishedParsed map[string]any
	if err := json.Unmarshal(readHandle(t, published), &publishedParsed); err != nil {
		t.Fatal(err)
	}
	if _, ok := publishedParsed["outcome"]; ok {
		t.Fatalf("published reader saw outcome %#v", publishedParsed["outcome"])
	}
	read, err = result.Read()
	if err != nil {
		t.Fatal(err)
	}
	if read.Outcome != "complete" {
		t.Fatalf("outcome %s", read.Outcome)
	}
}

func TestSameBuildResourceBacksVideoAndComposite(t *testing.T) {
	root := t.TempDir()
	payload := []byte{1, 2, 3, 4}
	video := protocol.BlobRef{Kind: "blob", Resource: "res_same-build-video", Size: int64(len(payload)), MediaType: "video/mp4"}
	id := "bld_20260902T100000000Z_0000000001"
	result, err := CreateFileBuildResult(root, BuildResultSeed{
		ID: id, Source: BuildResultSource{ID: "/project/main.svml"}, Run: &BuildResultSource{ID: "/project/build.svrun"},
		Targets: []string{"shot.video"},
		PublishedOutputs: []BuildResultPublishedOutput{
			{Name: "shot.video", Output: "logical:shot-video"},
			{Name: "shot.take", Output: "logical:shot-take"},
			{Name: "shot.take-copy", Output: "logical:shot-take-copy"},
			{Name: "unused.video", Output: "logical:unused"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Directory != filepath.Join(root, "2026-09-02", id) {
		t.Fatalf("directory %s", result.Directory)
	}
	stored, err := os.ReadFile(filepath.Join(result.Directory, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	var storedManifest map[string]any
	if err := json.Unmarshal(stored, &storedManifest); err != nil {
		t.Fatal(err)
	}
	if _, ok := storedManifest["id"]; ok {
		t.Fatal("stored manifest repeats its Build id")
	}
	read, err := result.Read()
	if err != nil || read.ID != id {
		t.Fatalf("id %v %v", read.ID, err)
	}
	manifest, err := result.Sync(BuildResultSync{
		State: testState([]protocol.TypedRecord{
			{ID: "record:video", Type: videoType, Value: protocol.StoredValue{Kind: "blob", Resource: video.Resource, Size: video.Size, MediaType: video.MediaType}},
			{ID: "record:take", Type: compositeType, Value: protocol.StoredValue{Kind: "inline", Value: map[string]any{
				"media": map[string]any{"visual": map[string]any{"artifact": video}},
			}}},
		}, []outputBinding{
			{output: "logical:shot-video", record: "record:video"},
			{output: "logical:shot-take", record: "record:take"},
			{output: "logical:shot-take-copy", record: "record:take"},
			{output: "logical:unused", record: "record:unused"},
		}),
		Resources: resourceOpener(func(artifact protocol.BlobRef) (io.Reader, error) {
			if artifact.Resource != video.Resource {
				t.Fatalf("resource %s", artifact.Resource)
			}
			return bytes.NewReader(payload), nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sortedOutputNames(manifest.Outputs), []string{"shot.take", "shot.take-copy", "shot.video"}) {
		t.Fatalf("names %v", sortedOutputNames(manifest.Outputs))
	}
	if manifest.Outputs["shot.video"].Value.Kind != "build-file" || manifest.Outputs["shot.take"].Value.Kind != "value" {
		t.Fatalf("%#v", manifest.Outputs)
	}
	videoPath := manifest.Outputs["shot.video"].Value.File.Path
	takePath := manifest.Outputs["shot.take"].Value.Path
	if manifest.Outputs["shot.take-copy"].Value.Path != takePath || videoPath != "files/file-0001.mp4" {
		t.Fatalf("video %s take %s copy %s", videoPath, takePath, manifest.Outputs["shot.take-copy"].Value.Path)
	}
	takeBytes, err := os.ReadFile(filepath.Join(result.Directory, takePath))
	if err != nil {
		t.Fatal(err)
	}
	var take map[string]any
	if err := json.Unmarshal(takeBytes, &take); err != nil {
		t.Fatal(err)
	}
	if take["format"] != "hypit.result-value@1" {
		t.Fatal(take["format"])
	}
	visual := take["value"].(map[string]any)["media"].(map[string]any)["visual"].(map[string]any)
	if visual["artifact"] != nil {
		t.Fatalf("artifact %#v", visual["artifact"])
	}
	binding := take["resources"].([]any)[0].(map[string]any)
	if !reflect.DeepEqual(binding["at"], []any{"media", "visual", "artifact"}) {
		t.Fatalf("at %#v", binding["at"])
	}
	if binding["file"].(map[string]any)["path"] != videoPath {
		t.Fatalf("file %#v", binding["file"])
	}
	files, err := os.ReadDir(filepath.Join(result.Directory, "files"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name() != "file-0001.mp4" {
		t.Fatalf("files %#v", files)
	}
	failure := "final render failed"
	failed, err := result.Finish(BuildResultFinish{Outcome: "failed", Failure: &failure})
	if err != nil {
		t.Fatal(err)
	}
	if failed.Outcome != "failed" || failed.Failure == nil || *failed.Failure != failure {
		t.Fatalf("%#v", failed)
	}
	if !reflect.DeepEqual(sortedOutputNames(failed.Outputs), []string{"shot.take", "shot.take-copy", "shot.video"}) {
		t.Fatal(sortedOutputNames(failed.Outputs))
	}
	if _, err := result.Finish(BuildResultFinish{Outcome: "failed", Failure: &failure}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(result.Directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == ".writer.json" {
			t.Fatal("writer state remains")
		}
	}
	mustFail(t, func() error {
		_, err := result.Finish(BuildResultFinish{Outcome: "complete"})
		return err
	}, "already finished with a different outcome")
	repository := NewFileBuildResultRepository(root)
	presented, err := repository.UpdatePresentation(id, BuildResultPresentationUpdate{
		SetOutputDisplayNames: true,
		OutputDisplayNames:    map[string]*string{"shot.video": strPtr("Opening portrait")},
		SetTitle:              true,
		Title:                 strPtr("Episode 12 opening"),
		SetNote:               true,
		Note:                  strPtr("Use the quieter take."),
		SetHighlightedOutputs: true,
		HighlightedOutputs:    []string{"shot.video", "shot.take", "shot.video"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if presented.Title == nil || *presented.Title != "Episode 12 opening" {
		t.Fatal(presented.Title)
	}
	if presented.Outputs["shot.video"].DisplayName == nil || *presented.Outputs["shot.video"].DisplayName != "Opening portrait" {
		t.Fatal(presented.Outputs["shot.video"].DisplayName)
	}
	if !reflect.DeepEqual(presented.Outputs["shot.video"].Value, failed.Outputs["shot.video"].Value) {
		t.Fatalf("%#v vs %#v", presented.Outputs["shot.video"].Value, failed.Outputs["shot.video"].Value)
	}
	if presented.Outputs["shot.take"].DisplayName != nil || presented.Note == nil || *presented.Note != "Use the quieter take." {
		t.Fatalf("%#v", presented)
	}
	if !reflect.DeepEqual(presented.HighlightedOutputs, []string{"shot.video", "shot.take"}) {
		t.Fatal(presented.HighlightedOutputs)
	}
	again, err := repository.Read(presented.ID)
	if err != nil || again.Outputs["shot.video"].DisplayName == nil || *again.Outputs["shot.video"].DisplayName != "Opening portrait" {
		t.Fatal(again)
	}
	mustFail(t, func() error {
		_, err := repository.UpdatePresentation(presented.ID, BuildResultPresentationUpdate{
			SetOutputDisplayNames: true, OutputDisplayNames: map[string]*string{"missing": strPtr("Oops")},
		})
		return err
	}, "has no Output")
	mustFail(t, func() error {
		_, err := repository.UpdatePresentation(presented.ID, BuildResultPresentationUpdate{
			SetOutputDisplayNames: true, OutputDisplayNames: map[string]*string{"shot.video": strPtr("  ")},
		})
		return err
	}, "must not be empty")
	mustFail(t, func() error {
		_, err := repository.UpdatePresentation(id, BuildResultPresentationUpdate{
			SetHighlightedOutputs: true, HighlightedOutputs: []string{"missing.output"},
		})
		return err
	}, `has no Output missing\.output`)
	cleared, err := repository.UpdatePresentation(id, BuildResultPresentationUpdate{
		SetOutputDisplayNames: true, OutputDisplayNames: map[string]*string{"shot.video": nil},
		SetTitle: true, SetNote: true, SetHighlightedOutputs: true, HighlightedOutputs: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Outputs["shot.video"].DisplayName != nil || cleared.Title != nil || cleared.Note != nil || cleared.HighlightedOutputs != nil {
		t.Fatalf("%#v", cleared)
	}
}

func TestFilesystemRepositoryStreamsOneByteRange(t *testing.T) {
	root := t.TempDir()
	build := "bld_20260902T100000009Z_0000000001"
	directory := BuildResultDirectory(root, build)
	if err := os.MkdirAll(filepath.Join(directory, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	payload := []byte("abcdefghij")
	if err := os.WriteFile(filepath.Join(directory, "files", "video.mp4"), payload, 0o644); err != nil {
		t.Fatal(err)
	}
	repository := NewFileBuildResultRepository(root)
	stream, err := repository.OpenFile(build, BuildResultFileRef{
		Kind: "build-file", Path: "files/video.mp4", Size: int64(len(payload)), MediaType: "video/mp4",
	}, &BuildResultFileRange{Start: 3, EndExclusive: 7})
	if err != nil || stream == nil {
		t.Fatalf("stream %v %v", stream, err)
	}
	defer stream.Close()
	got, err := io.ReadAll(stream)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "defg" {
		t.Fatalf("%q", got)
	}
}

func TestFilesystemDecodesResultAndWriterData(t *testing.T) {
	root := t.TempDir()
	malformed := "bld_20260902T100000010Z_0000000001"
	unfinished := "bld_20260902T100000011Z_0000000001"
	directory := BuildResultDirectory(root, malformed)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{
      "format": "hypit.build-result@1",
      "source": { "id": "main.svml" },
      "targets": ["video"],
      "outputs": {
        "video": {
          "type": {"module":{"name":"example.media","version":"1"},"name":"Video"},
          "value": { "kind": "build-file", "path": "../outside.mp4", "size": 1, "mediaType": "video/mp4" }
        }
      }
    }`
	if err := os.WriteFile(filepath.Join(directory, "result.json"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	repository := NewFileBuildResultRepository(root)
	mustFail(t, func() error {
		_, err := repository.Read(malformed)
		return err
	}, `outputs\["video"\]\.value\.path is not a Result-relative path`)
	writer, err := repository.Create(BuildResultSeed{
		ID: unfinished, Source: BuildResultSource{ID: "main.svml"}, Targets: []string{}, PublishedOutputs: []BuildResultPublishedOutput{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(BuildResultDirectory(root, unfinished), ".writer.json"), []byte(`{"resources":null,"values":{},"publishedOutputs":[],"forwards":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	mustFail(t, func() error {
		_, err := writer.Sync(BuildResultSync{
			State: testState(nil, nil), Resources: resourceOpener(func(protocol.BlobRef) (io.Reader, error) { return nil, nil }),
		})
		return err
	}, `\.writer\.json\.resources must be an object`)
}

func TestReservedDomainObjectsRemainOrdinaryCompositeData(t *testing.T) {
	root := t.TempDir()
	id := "bld_20260902T100000000Z_0000000001"
	directory := BuildResultDirectory(root, id)
	if err := os.MkdirAll(filepath.Join(directory, "values"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
      "format": "hypit.build-result@1",
      "source": { "id": "main.svml" },
      "targets": ["value"],
      "outcome": "complete",
      "finishedAt": 1,
      "outputs": { "value": { "type": {"module":{"name":"example.media","version":"1"},"name":"Video"}, "value": { "kind": "value", "path": "values/value-0001.json" } } }
    }`
	document := `{
      "format": "hypit.result-value@1",
      "value": { "nested": { "kind": "build-output", "build": "` + id + `", "output": "value" } },
      "resources": []
    }`
	if err := os.WriteFile(filepath.Join(directory, "result.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "values", "value-0001.json"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	resolved, err := NewFileBuildResultRepository(root).Resolve(id, "value")
	if err != nil {
		t.Fatal(err)
	}
	if resolved == nil || resolved.Value.Kind != "value" || resolved.Value.Document == nil {
		t.Fatalf("%#v", resolved)
	}
	if !reflect.DeepEqual(resolved.Value.Document.Value, map[string]any{
		"nested": map[string]any{"kind": "build-output", "build": id, "output": "value"},
	}) {
		t.Fatalf("%#v", resolved.Value.Document.Value)
	}
}

func TestDescribingCompositeOutputReadsOnlyItsManifest(t *testing.T) {
	root := t.TempDir()
	id := "bld_20260902T100000020Z_0000000001"
	directory := BuildResultDirectory(root, id)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{
      "format": "hypit.build-result@1",
      "source": { "id": "main.svml" },
      "targets": ["take"],
      "outcome": "complete",
      "finishedAt": 1,
      "outputs": { "take": { "type": {"module":{"name":"example.speech","version":"1"},"name":"CompositeValue"}, "value": { "kind": "value", "path": "values/missing.json" } } }
    }`
	if err := os.WriteFile(filepath.Join(directory, "result.json"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	repository := NewFileBuildResultRepository(root)
	description, err := repository.DescribeOutput(id, "take")
	if err != nil || description == nil || description.Kind != "composite" {
		t.Fatalf("%#v %v", description, err)
	}
	mustFail(t, func() error {
		_, err := repository.Resolve(id, "take")
		return err
	}, `missing\.json`)
}

func TestReusedPublicOutputIsAForwardReference(t *testing.T) {
	root := t.TempDir()
	prior, err := CreateFileBuildResult(root, BuildResultSeed{
		ID: "bld_20260902T100000001Z_0000000001", Source: BuildResultSource{ID: "/project/main.svml"},
		Targets:          []string{"opening.video"},
		PublishedOutputs: []BuildResultPublishedOutput{{Name: "opening.video", Output: "logical:opening-video"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prior.Sync(BuildResultSync{
		State: testState([]protocol.TypedRecord{{
			ID: "record:opening", Type: videoType,
			Value: protocol.StoredValue{Kind: "blob", Resource: "res_opening", Size: 10, MediaType: "video/mp4"},
		}}, []outputBinding{{output: "logical:opening-video", record: "record:opening"}}),
		Resources: resourceOpener(func(protocol.BlobRef) (io.Reader, error) {
			return bytes.NewReader(make([]byte, 10)), nil
		}),
	}); err != nil {
		t.Fatal(err)
	}
	reused := BuildResultSeed{
		ID: "bld_20260902T100000002Z_0000000002", Source: BuildResultSource{ID: "/project/main.svml"},
		Targets:          []string{"shot.video"},
		PublishedOutputs: []BuildResultPublishedOutput{{Name: "shot.video", Output: "logical:shot-video"}},
		Forwards: []BuildResultForward{{
			Output: "logical:shot-video", Build: "bld_20260902T100000001Z_0000000001", SourceOutput: "opening.video",
		}},
	}
	mustFail(t, func() error {
		_, err := CreateFileBuildResult(root, reused)
		return err
	}, "is not a finished Result")
	if _, err := prior.Finish(BuildResultFinish{Outcome: "complete"}); err != nil {
		t.Fatal(err)
	}
	mustFail(t, func() error {
		return NewFileBuildResultRepository(root).RemoveIncomplete("bld_20260902T100000001Z_0000000001")
	}, "cannot be removed")
	result, err := CreateFileBuildResult(root, reused)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := result.Sync(BuildResultSync{
		State: testState([]protocol.TypedRecord{{
			ID: "record:prior", Type: videoType,
			Value: protocol.StoredValue{Kind: "blob", Resource: "res_prior_video", Size: 10, MediaType: "video/mp4"},
		}}, []outputBinding{{output: "logical:shot-video", record: "record:prior"}}),
		Resources: resourceOpener(func(protocol.BlobRef) (io.Reader, error) {
			return nil, errors.New("a forwarded output must not copy its historical bytes")
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	value := manifest.Outputs["shot.video"].Value
	if value.Kind != "build-output" || value.Build != "bld_20260902T100000001Z_0000000001" || value.Output != "opening.video" {
		t.Fatalf("%#v", value)
	}
}

func TestForwardOnlyCompositePublishesWithoutOpeningValueDocument(t *testing.T) {
	root := t.TempDir()
	priorID := "bld_20260902T100000030Z_0000000001"
	nextID := "bld_20260902T100000031Z_0000000001"
	prior, err := CreateFileBuildResult(root, BuildResultSeed{
		ID: priorID, Source: BuildResultSource{ID: "/project/main.svml"},
		Targets:          []string{"layout"},
		PublishedOutputs: []BuildResultPublishedOutput{{Name: "layout", Output: "logical:layout"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prior.Sync(BuildResultSync{
		State: testState([]protocol.TypedRecord{{
			ID: "record:layout", Type: videoType, Value: protocol.StoredValue{Kind: "inline", Value: map[string]any{"title": "ready"}},
		}}, []outputBinding{{output: "logical:layout", record: "record:layout"}}),
		Resources: resourceOpener(func(protocol.BlobRef) (io.Reader, error) {
			return nil, errors.New("fixture has no files")
		}),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := prior.Finish(BuildResultFinish{Outcome: "complete"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(prior.Directory, "values", "value-0001.json")); err != nil {
		t.Fatal(err)
	}
	forwardType := videoType
	forwarded, err := CreateFileBuildResult(root, BuildResultSeed{
		ID: nextID, Source: BuildResultSource{ID: "/project/main.svml"},
		Targets:          []string{"layout"},
		PublishedOutputs: []BuildResultPublishedOutput{{Name: "layout", Output: "logical:layout"}},
		Forwards: []BuildResultForward{{
			Output: "logical:layout", Build: priorID, SourceOutput: "layout", Type: &forwardType,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := forwarded.Sync(BuildResultSync{
		State: testState(nil, nil),
		Resources: resourceOpener(func(protocol.BlobRef) (io.Reader, error) {
			return nil, errors.New("a forward-only Result has no bytes to open")
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	layout := manifest.Outputs["layout"]
	if !reflect.DeepEqual(layout.Type, videoType) || layout.DisplayName != nil {
		t.Fatalf("%#v", layout)
	}
	if layout.Value.Kind != "build-output" || layout.Value.Build != priorID || layout.Value.Output != "layout" {
		t.Fatalf("%#v", layout.Value)
	}
}

func TestFailedFilesystemSyncDoesNotInheritStaleBytes(t *testing.T) {
	root := t.TempDir()
	result, err := CreateFileBuildResult(root, BuildResultSeed{
		ID: "bld_20260924T100000000Z_0000000001", Source: BuildResultSource{ID: "/project/main.svml"},
		Targets: []string{"a", "b", "z"},
		PublishedOutputs: []BuildResultPublishedOutput{
			{Name: "a", Output: "a"}, {Name: "b", Output: "b"}, {Name: "z", Output: "z"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := func(names []string) protocol.BuildState {
		records := make([]protocol.TypedRecord, len(names))
		bindings := make([]outputBinding, len(names))
		for index, name := range names {
			records[index] = protocol.TypedRecord{
				ID: name, Type: videoType,
				Value: protocol.StoredValue{Kind: "blob", Resource: protocol.ResourceId("res_" + name), Size: 1, MediaType: "video/mp4"},
			}
			bindings[index] = outputBinding{output: name, record: name}
		}
		return testState(records, bindings)
	}
	fail := true
	resources := resourceOpener(func(artifact protocol.BlobRef) (io.Reader, error) {
		if fail && artifact.Resource == "res_z" {
			return nil, errors.New("intentional source failure")
		}
		last := artifact.Resource[len(artifact.Resource)-1]
		if last >= 'a' && last <= 'z' {
			last = last - 'a' + 'A'
		}
		return bytes.NewReader([]byte{last}), nil
	})
	mustFail(t, func() error {
		_, err := result.Sync(BuildResultSync{State: snapshot([]string{"b", "z"}), Resources: resources})
		return err
	}, "intentional source failure")
	read, err := result.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Outputs) != 0 {
		t.Fatalf("%#v", read.Outputs)
	}
	fail = false
	if _, err := result.Sync(BuildResultSync{State: snapshot([]string{"a", "b", "z"}), Resources: resources}); err != nil {
		t.Fatal(err)
	}
	read, err = result.Read()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b", "z"} {
		value := read.Outputs[name].Value
		if value.Kind != "build-file" || value.File == nil {
			t.Fatalf("%s %#v", name, value)
		}
		body, err := os.ReadFile(filepath.Join(result.Directory, value.File.Path))
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != string([]byte{name[0] - 'a' + 'A'}) {
			t.Fatalf("%s %q", name, body)
		}
	}
}

func TestResultWriterRecoversNumericCursor(t *testing.T) {
	artifact := protocol.BlobRef{Kind: "blob", Resource: "res_new-image", Size: 1, MediaType: "image/png"}
	var resourcePaths []string
	var valuePaths []string
	updated, err := SyncBuildResultOutputs(BuildResultSyncInput{
		Manifest: BuildResultManifest{
			Format: "hypit.build-result@1", ID: "bld_20260924T110000000Z_0000000001",
			Source: BuildResultSource{ID: "main.svml"}, Targets: []string{"image"}, Outputs: map[string]BuildResultOutput{},
		},
		Writer: BuildResultWriterState{
			Resources:        map[string]string{"old": "files/file-9999.mp4"},
			Values:           map[string]string{"old": "values/value-9999.json"},
			PublishedOutputs: []BuildResultPublishedOutput{{Name: "image", Output: "logical:image"}},
			Forwards:         []BuildResultForward{},
		},
		Sync: BuildResultSync{
			State: testState([]protocol.TypedRecord{{
				ID: "record:image", Type: videoType, Value: protocol.StoredValue{Kind: "inline", Value: map[string]any{"artifact": artifact}},
			}}, []outputBinding{{output: "logical:image", record: "record:image"}}),
			Resources: resourceOpener(func(protocol.BlobRef) (io.Reader, error) {
				return nil, errors.New("target owns this test write")
			}),
		},
		Target: recordingTarget{
			resources: func(path string, _ protocol.BlobRef) error {
				resourcePaths = append(resourcePaths, path)
				return nil
			},
			values: func(path string) error {
				valuePaths = append(valuePaths, path)
				return nil
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resourcePaths, []string{"files/file-10000.png"}) || !reflect.DeepEqual(valuePaths, []string{"values/value-10000.json"}) {
		t.Fatalf("resources %v values %v", resourcePaths, valuePaths)
	}
	if updated.Manifest.Outputs["image"].Value.Kind != "value" {
		t.Fatal(updated.Manifest.Outputs["image"].Value.Kind)
	}
}

func TestResultWriterBoundsResourceWrites(t *testing.T) {
	artifacts := make([]protocol.BlobRef, 12)
	values := make([]any, len(artifacts))
	for index := range artifacts {
		artifacts[index] = protocol.BlobRef{
			Kind: "blob", Resource: protocol.ResourceId("res_" + string(rune('0'+(index/10))) + string(rune('0'+(index%10)))),
			Size: 1, MediaType: "image/png",
		}
		values[index] = artifacts[index]
	}
	input := func(target BuildResultWriteTarget) (BuildResultSyncResult, error) {
		return SyncBuildResultOutputs(BuildResultSyncInput{
			Manifest: BuildResultManifest{
				Format: "hypit.build-result@1", ID: "bld_20260924T110000001Z_0000000001",
				Source: BuildResultSource{ID: "main.svml"}, Targets: []string{"images"}, Outputs: map[string]BuildResultOutput{},
			},
			Writer: BuildResultWriterState{
				Resources: map[string]string{}, Values: map[string]string{},
				PublishedOutputs: []BuildResultPublishedOutput{{Name: "images", Output: "logical:images"}},
				Forwards:         []BuildResultForward{},
			},
			Sync: BuildResultSync{
				State: testState([]protocol.TypedRecord{{
					ID: "record:images", Type: videoType, Value: protocol.StoredValue{Kind: "inline", Value: values},
				}}, []outputBinding{{output: "logical:images", record: "record:images"}}),
				Resources: resourceOpener(func(protocol.BlobRef) (io.Reader, error) {
					return nil, errors.New("target owns this test write")
				}),
			},
			Target: target,
		})
	}
	var active, maximum, completed atomic.Int32
	if _, err := input(recordingTarget{
		resources: func(string, protocol.BlobRef) error {
			cur := active.Add(1)
			for {
				max := maximum.Load()
				if cur <= max || maximum.CompareAndSwap(max, cur) {
					break
				}
			}
			defer active.Add(-1)
			time.Sleep(2 * time.Millisecond)
			completed.Add(1)
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if maximum.Load() != 4 || completed.Load() != int32(len(artifacts)) {
		t.Fatalf("maximum %d completed %d", maximum.Load(), completed.Load())
	}
	active.Store(0)
	maximum.Store(0)
	var started, valueWrites atomic.Int32
	mustFail(t, func() error {
		_, err := input(recordingTarget{
			resources: func(_ string, artifact protocol.BlobRef) error {
				started.Add(1)
				cur := active.Add(1)
				for {
					max := maximum.Load()
					if cur <= max || maximum.CompareAndSwap(max, cur) {
						break
					}
				}
				defer active.Add(-1)
				if artifact.Resource == artifacts[0].Resource {
					time.Sleep(2 * time.Millisecond)
					return errors.New("intentional resource failure")
				}
				time.Sleep(20 * time.Millisecond)
				return nil
			},
			values: func(string) error {
				valueWrites.Add(1)
				return nil
			},
		})
		return err
	}, "intentional resource failure")
	if maximum.Load() != 4 {
		t.Fatalf("maximum %d", maximum.Load())
	}
	if started.Load() != 4 {
		t.Fatalf("started %d, no queued write starts after the first failure", started.Load())
	}
	if active.Load() != 0 {
		t.Fatal("all already-started writes settle before sync rejects")
	}
	if valueWrites.Load() != 0 {
		t.Fatal("a value document cannot publish before all of its files exist")
	}
}

func TestFilesystemResultsBrowseNewestFirst(t *testing.T) {
	root := t.TempDir()
	repository := NewFileBuildResultRepository(root)
	ids := []string{
		"bld_20260901T100000001Z_0000000001",
		"bld_20260902T100000002Z_0000000001",
		"bld_20260903T100000003Z_0000000001",
	}
	mustFail(t, func() error {
		_, err := repository.Create(BuildResultSeed{
			ID: "bld_20260902T100000000Z_0000000001", Source: BuildResultSource{ID: "/project/main.svml"},
			Targets: []string{"video"},
			PublishedOutputs: []BuildResultPublishedOutput{
				{Name: "video", Output: "logical:video"},
				{Name: "duplicate", Output: "logical:video"},
			},
		})
		return err
	}, "has more than one published name")
	failure := "ordering fixture"
	for _, id := range ids {
		writer, err := repository.Create(BuildResultSeed{
			ID: id, Source: BuildResultSource{ID: "/project/main.svml"},
			Targets:          []string{"video"},
			PublishedOutputs: []BuildResultPublishedOutput{{Name: "video", Output: "logical:video"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Finish(BuildResultFinish{Outcome: "failed", Failure: &failure}); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	if !reflect.DeepEqual(names, []string{"2026-09-01", "2026-09-02", "2026-09-03"}) {
		t.Fatalf("buckets %v", names)
	}
	first, err := repository.Browse(BuildResultBrowseRequest{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(first.Results))
	for index, item := range first.Results {
		got[index] = item.ID
	}
	if !reflect.DeepEqual(got, []string{ids[2], ids[1]}) || first.Next == nil || *first.Next != ids[1] {
		t.Fatalf("%v next %v", got, first.Next)
	}
	second, err := repository.Browse(BuildResultBrowseRequest{Limit: 2, Before: first.Next})
	if err != nil {
		t.Fatal(err)
	}
	got = make([]string, len(second.Results))
	for index, item := range second.Results {
		got[index] = item.ID
	}
	if !reflect.DeepEqual(got, []string{ids[0]}) || second.Next != nil {
		t.Fatalf("%v next %v", got, second.Next)
	}
}
