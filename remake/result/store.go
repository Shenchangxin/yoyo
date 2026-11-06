// Copyright 2026 Hypit.AI
// Translated from @hypit/result/src/node/store.ts (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package result

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/remake/atomic-file"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

const manifestName = "result.json"
const writerStateName = ".writer.json"

var dateBucketPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ApplyBuildResultPresentation edits title, note, highlights, and display names.
func ApplyBuildResultPresentation(manifest BuildResultManifest, update BuildResultPresentationUpdate) BuildResultManifest {
	if manifest.Outcome == "" {
		panicErr("Build Result " + manifest.ID + " is not finished")
	}
	base := manifest
	if update.SetTitle {
		if update.Title == nil {
			base.Title = nil
		} else {
			title := strings.TrimSpace(*update.Title)
			if title == "" {
				panicErr("Build Result title must not be empty")
			}
			base.Title = &title
		}
	}
	if update.SetNote {
		if update.Note == nil {
			base.Note = nil
		} else {
			note := strings.TrimSpace(*update.Note)
			if note == "" {
				panicErr("Build Result note must not be empty")
			}
			base.Note = &note
		}
	}
	if update.SetHighlightedOutputs {
		highlighted := dedupeStrings(update.HighlightedOutputs)
		for _, output := range highlighted {
			if _, ok := manifest.Outputs[output]; !ok {
				panicErr("Build Result " + manifest.ID + " has no Output " + output)
			}
		}
		if len(highlighted) == 0 {
			base.HighlightedOutputs = nil
		} else {
			base.HighlightedOutputs = highlighted
		}
	}
	if update.SetOutputDisplayNames {
		outputs := cloneOutputs(manifest.Outputs)
		for name, value := range update.OutputDisplayNames {
			entry, ok := manifest.Outputs[name]
			if !ok {
				panicErr("Build Result " + manifest.ID + " has no Output " + name)
			}
			entry.DisplayName = nil
			if value != nil {
				displayName := strings.TrimSpace(*value)
				if displayName == "" {
					panicErr("Output display name must not be empty")
				}
				entry.DisplayName = &displayName
			}
			outputs[name] = entry
		}
		base.Outputs = outputs
	}
	return base
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func randomUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func writeJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := jsonIndent(prepareJSON(value))
	if err != nil {
		return err
	}
	temporary := path + ".part-" + randomUUID()
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		os.Remove(temporary)
		return writeErr
	}
	if closeErr != nil {
		os.Remove(temporary)
		return closeErr
	}
	if err := atomicfile.ReplaceFile(temporary, path); err != nil {
		os.Remove(temporary)
		return err
	}
	return nil
}

func prepareJSON(value any) any {
	switch item := value.(type) {
	case BuildResultManifest:
		if item.Targets == nil {
			item.Targets = []string{}
		}
		if item.Outputs == nil {
			item.Outputs = map[string]BuildResultOutput{}
		}
		return item
	case BuildResultWriterState:
		return item.normalized()
	case BuildResultValueDocument:
		if item.Resources == nil {
			item.Resources = []BuildResultResourceBinding{}
		}
		return item
	default:
		return value
	}
}

func writeManifestAtomic(path string, manifest BuildResultManifest) error {
	return writeJSONAtomic(path, EncodeBuildResultManifest(manifest))
}

func writeStreamAtomic(destination string, input io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	temporary := destination + ".part-" + randomUUID()
	output, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		os.Remove(temporary)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(temporary)
		return closeErr
	}
	if err := atomicfile.ReplaceFile(temporary, destination); err != nil {
		os.Remove(temporary)
		return err
	}
	return nil
}

func copyArtifactAtomic(source BuildResultResourceSource, artifact protocol.BlobRef, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	input, err := source.Open(artifact)
	if err != nil {
		return err
	}
	if input == nil {
		panicErr("Build resource " + string(artifact.Resource) + " is unavailable")
	}
	return writeStreamAtomic(destination, input)
}

func removeFile(path string) error {
	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func readBytes(path string) ([]byte, error) {
	file, err := openShared(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// LocateRepositoryBuildResultOutput follows explicit forwarding to the terminal Output.
func LocateRepositoryBuildResultOutput(repository BuildResultReader, build, output string) (*RepositoryBuildResultOutputLocation, error) {
	seen := map[string]struct{}{}
	currentBuild := build
	currentOutput := output
	for {
		address := currentBuild + "\x00" + currentOutput
		if _, ok := seen[address]; ok {
			panicErr("Build Output forwarding repeats " + currentBuild + " / " + currentOutput)
		}
		seen[address] = struct{}{}
		manifest, err := repository.Read(currentBuild)
		if err != nil {
			return nil, err
		}
		if manifest == nil {
			return nil, nil
		}
		if manifest.Outcome == "" {
			panicErr("Build " + currentBuild + " is not a finished Result")
		}
		entry, ok := manifest.Outputs[currentOutput]
		if !ok {
			return nil, nil
		}
		if entry.Value.Kind == "build-output" {
			currentBuild = entry.Value.Build
			currentOutput = entry.Value.Output
			continue
		}
		return &RepositoryBuildResultOutputLocation{Build: currentBuild, Output: currentOutput, Type: entry.Type}, nil
	}
}

// LocateBuildResultOutput follows forwarding inside one filesystem repository.
func LocateBuildResultOutput(root, build, output string) (*RepositoryBuildResultOutputLocation, error) {
	return LocateRepositoryBuildResultOutput(directoryBuildResultReader{root: root}, build, output)
}

func buildResultDateBucket(build string) (string, bool) {
	created, ok := protocol.BuildIdCreatedAt(build)
	if !ok {
		return "", false
	}
	return time.UnixMilli(created).UTC().Format("2006-01-02"), true
}

// BuildResultDirectory is the absolute directory of one Build under root.
func BuildResultDirectory(root, build string) string {
	protocol.AssertOrderedBuildId(build)
	directory, err := filepath.Abs(root)
	if err != nil {
		panic(err)
	}
	bucket, ok := buildResultDateBucket(build)
	if !ok {
		panicErr("Build id must contain its UTC submission time and a ten-character nonce")
	}
	target := filepath.Join(directory, bucket, build)
	relation, err := filepath.Rel(directory, target)
	if err != nil || relation == "" || relation == ".." || strings.HasPrefix(relation, ".."+string(filepath.Separator)) {
		panicErr("Build " + build + " leaves Build Result root " + directory)
	}
	return target
}

// ReadBuildResult reads result.json, or returns nil when the file is absent.
func ReadBuildResult(directory string) (*BuildResultManifest, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(absolute, manifestName)
	payload, err := readBytes(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	manifest := DecodeBuildResultManifest(DecodeBuildResultJson(payload, path), filepath.Base(absolute), path)
	return &manifest, nil
}

func containedResultPath(directory, path string) string {
	if filepath.IsAbs(path) {
		panicErr("Build Result path " + path + " must be relative")
	}
	root, err := filepath.Abs(directory)
	if err != nil {
		panic(err)
	}
	absolute := filepath.Clean(filepath.Join(root, path))
	relation, err := filepath.Rel(root, absolute)
	if err != nil || relation == ".." || strings.HasPrefix(relation, ".."+string(filepath.Separator)) {
		panicErr("Build Result path " + path + " leaves " + root)
	}
	return absolute
}

// BrowseBuildResults lists finished Results, newest Build id first.
func BrowseBuildResults(root string, request BuildResultBrowseRequest) (BuildResultPage, error) {
	if request.Limit <= 0 || int64(request.Limit) > maxSafeInteger {
		panicErr("Build Result browse limit must be positive")
	}
	if request.Before != nil {
		protocol.AssertOrderedBuildId(*request.Before)
	}
	directory, err := filepath.Abs(root)
	if err != nil {
		return BuildResultPage{}, err
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			entries = nil
		} else {
			return BuildResultPage{}, err
		}
	}
	var beforeBucket string
	if request.Before != nil {
		beforeBucket, _ = buildResultDateBucket(*request.Before)
	}
	var buckets []string
	for _, entry := range entries {
		if !entry.IsDir() || !dateBucketPattern.MatchString(entry.Name()) {
			continue
		}
		if beforeBucket != "" && entry.Name() > beforeBucket {
			continue
		}
		buckets = append(buckets, entry.Name())
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i] > buckets[j] })
	var found []BuildResultManifest
	for _, bucket := range buckets {
		bucketDirectory := filepath.Join(directory, bucket)
		builds, err := os.ReadDir(bucketDirectory)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return BuildResultPage{}, err
		}
		var ordered []string
		for _, entry := range builds {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			got, ok := buildResultDateBucket(name)
			if !ok || got != bucket {
				continue
			}
			if request.Before != nil && name >= *request.Before {
				continue
			}
			ordered = append(ordered, name)
		}
		sort.Slice(ordered, func(i, j int) bool { return ordered[i] > ordered[j] })
		for _, build := range ordered {
			manifest, err := ReadBuildResult(filepath.Join(bucketDirectory, build))
			if err != nil {
				return BuildResultPage{}, err
			}
			if manifest == nil || manifest.Outcome == "" || manifest.FinishedAt == nil {
				continue
			}
			found = append(found, *manifest)
			if len(found) > request.Limit {
				break
			}
		}
		if len(found) > request.Limit {
			break
		}
	}
	page := BuildResultPage{Results: []BuildResultManifest{}}
	if len(found) > request.Limit {
		page.Results = found[:request.Limit]
		if len(page.Results) > 0 {
			next := page.Results[len(page.Results)-1].ID
			page.Next = &next
		}
	} else if found != nil {
		page.Results = found
	}
	return page, nil
}

// ResolveBuildResultOutput follows forwarding and opens a composite value document.
func ResolveBuildResultOutput(root, build, output string, external ...ExternalFileAccess) (*ResolvedBuildResultOutput, error) {
	access := externalFileAccess(external)
	seen := map[string]struct{}{}
	currentBuild := build
	currentOutput := output
	for {
		address := currentBuild + "\x00" + currentOutput
		if _, ok := seen[address]; ok {
			panicErr("Build Output forwarding repeats " + currentBuild + " / " + currentOutput)
		}
		seen[address] = struct{}{}
		directory := BuildResultDirectory(root, currentBuild)
		manifest, err := ReadBuildResult(directory)
		if err != nil {
			return nil, err
		}
		if manifest == nil {
			return nil, nil
		}
		entry, ok := manifest.Outputs[currentOutput]
		if !ok {
			return nil, nil
		}
		if entry.Value.Kind == "build-output" {
			currentBuild = entry.Value.Build
			currentOutput = entry.Value.Output
			continue
		}
		if entry.Value.Kind == "value" {
			path := containedResultPath(directory, entry.Value.Path)
			payload, err := readBytes(path)
			if err != nil {
				return nil, err
			}
			document := DecodeBuildResultValueDocument(
				DecodeBuildResultJson(payload, path),
				"Build "+currentBuild+" Output "+currentOutput,
			)
			return &ResolvedBuildResultOutput{
				Build: currentBuild, Output: currentOutput, Directory: directory, Type: entry.Type,
				Value: ResolvedBuildResultValue{BuildResultOutputValue: entry.Value, Document: &document},
			}, nil
		}
		kind := entry.Value.Kind
		if kind != "build-file" && kind != "external-file" && kind != "inline" {
			panicErr("Build " + currentBuild + " Output " + currentOutput + " has unsupported Result value kind " + kind)
		}
		value := entry.Value
		if kind == "external-file" && value.File != nil {
			current, err := CurrentFileReference(*value.File, access)
			if err != nil {
				return nil, err
			}
			value.File = &current
		}
		return &ResolvedBuildResultOutput{
			Build: currentBuild, Output: currentOutput, Directory: directory, Type: entry.Type,
			Value: ResolvedBuildResultValue{BuildResultOutputValue: value},
		}, nil
	}
}

// DescribeBuildResultOutput reports the terminal kind without opening content.
func DescribeBuildResultOutput(root, build, output string, external ...ExternalFileAccess) (*RepositoryBuildResultOutputDescription, error) {
	access := externalFileAccess(external)
	seen := map[string]struct{}{}
	currentBuild := build
	currentOutput := output
	for {
		address := currentBuild + "\x00" + currentOutput
		if _, ok := seen[address]; ok {
			panicErr("Build Output forwarding repeats " + currentBuild + " / " + currentOutput)
		}
		seen[address] = struct{}{}
		manifest, err := ReadBuildResult(BuildResultDirectory(root, currentBuild))
		if err != nil {
			return nil, err
		}
		if manifest == nil {
			return nil, nil
		}
		entry, ok := manifest.Outputs[currentOutput]
		if !ok {
			return nil, nil
		}
		if entry.Value.Kind == "build-output" {
			currentBuild = entry.Value.Build
			currentOutput = entry.Value.Output
			continue
		}
		file := entry.Value
		if file.Kind == "external-file" && file.File != nil {
			current, err := CurrentFileReference(*file.File, access)
			if err != nil {
				return nil, err
			}
			file.File = &current
		}
		if file.Kind == "build-file" || file.Kind == "external-file" {
			description := RepositoryBuildResultOutputDescription{Type: entry.Type, Kind: "resource"}
			if file.File != nil {
				description.Size = file.File.Size
				description.MediaType = file.File.MediaType
			}
			return &description, nil
		}
		if entry.Value.Kind == "value" {
			return &RepositoryBuildResultOutputDescription{Type: entry.Type, Kind: "composite"}, nil
		}
		return &RepositoryBuildResultOutputDescription{Type: entry.Type, Kind: "scalar"}, nil
	}
}

// NormalizeBuildResultForwards keeps only finished historical Results and their terminal Outputs.
func NormalizeBuildResultForwards(repository BuildResultReader, forwards []BuildResultForward) ([]BuildResultForward, error) {
	out := make([]BuildResultForward, len(forwards))
	for index, forward := range forwards {
		source, err := repository.Read(forward.Build)
		if err != nil {
			return nil, err
		}
		if source == nil || source.Outcome == "" {
			panicErr("Build " + forward.Build + " is not a finished Result")
		}
		resolved, err := LocateRepositoryBuildResultOutput(repository, forward.Build, forward.SourceOutput)
		if err != nil {
			return nil, err
		}
		if resolved == nil {
			panicErr("Build " + forward.Build + " has no Output " + forward.SourceOutput)
		}
		owner := source
		if resolved.Build != source.ID {
			owner, err = repository.Read(resolved.Build)
			if err != nil {
				return nil, err
			}
		}
		if owner == nil || owner.Outcome == "" {
			panicErr("Build " + resolved.Build + " is not a finished Result")
		}
		if forward.Type != nil && !protocol.SameType(*forward.Type, resolved.Type) {
			panicErr("Build " + forward.Build + " Output " + forward.SourceOutput + " has the wrong type for its forwarded Logical Output")
		}
		resolvedType := resolved.Type
		out[index] = BuildResultForward{
			Output:       forward.Output,
			Build:        resolved.Build,
			SourceOutput: resolved.Output,
			Type:         &resolvedType,
		}
	}
	if out == nil {
		out = []BuildResultForward{}
	}
	return out, nil
}

type directoryBuildResultReader struct {
	root string
}

func (r directoryBuildResultReader) Read(build string) (*BuildResultManifest, error) {
	return ReadBuildResult(BuildResultDirectory(r.root, build))
}

// FileBuildResult is one Build directory.
type FileBuildResult struct {
	Directory string
}

// CreateFileBuildResult prepares a new unfinished Result directory.
func CreateFileBuildResult(root string, seed BuildResultSeed, external ...ExternalFileAccess) (*FileBuildResult, error) {
	_ = externalFileAccess(external)
	protocol.AssertOrderedBuildId(seed.ID)
	AssertBuildResultSeed(seed)
	forwards, err := NormalizeBuildResultForwards(directoryBuildResultReader{root: root}, seed.Forwards)
	if err != nil {
		return nil, err
	}
	directory := BuildResultDirectory(root, seed.ID)
	if err := os.MkdirAll(filepath.Dir(directory), 0o755); err != nil {
		return nil, err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var temporary string
	success := false
	defer func() {
		if !success && temporary != "" {
			os.RemoveAll(temporary)
		}
	}()
	temporary = filepath.Join(rootAbs, ".preparing-"+seed.ID+"-"+randomUUID())
	if err := os.Mkdir(temporary, 0o755); err != nil {
		return nil, err
	}
	targets := append([]string{}, seed.Targets...)
	if targets == nil {
		targets = []string{}
	}
	manifest := BuildResultManifest{
		Format:  "hypit.build-result@1",
		ID:      seed.ID,
		Title:   seed.Title,
		Source:  seed.Source,
		Run:     seed.Run,
		Targets: targets,
		Outputs: map[string]BuildResultOutput{},
	}
	published := append([]BuildResultPublishedOutput{}, seed.PublishedOutputs...)
	if published == nil {
		published = []BuildResultPublishedOutput{}
	}
	writer := BuildResultWriterState{
		Resources:          map[string]string{},
		Values:             map[string]string{},
		PublishedOutputs:   published,
		ResourceReferences: seed.ResourceReferences,
		Forwards:           forwards,
	}
	if err := writeManifestAtomic(filepath.Join(temporary, manifestName), manifest); err != nil {
		return nil, err
	}
	if err := writeJSONAtomic(filepath.Join(temporary, writerStateName), writer); err != nil {
		return nil, err
	}
	if err := os.Rename(temporary, directory); err != nil {
		return nil, err
	}
	success = true
	return &FileBuildResult{Directory: directory}, nil
}

// OpenFileBuildResult opens a directory that already contains result.json.
func OpenFileBuildResult(directory string) (*FileBuildResult, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	manifest, err := ReadBuildResult(absolute)
	if err != nil {
		return nil, err
	}
	if manifest == nil {
		panicErr(absolute + " has no Build Result")
	}
	return &FileBuildResult{Directory: absolute}, nil
}

// Read returns the current manifest.
func (r *FileBuildResult) Read() (BuildResultManifest, error) {
	manifest, err := ReadBuildResult(r.Directory)
	if err != nil {
		return BuildResultManifest{}, err
	}
	if manifest == nil {
		panicErr(r.Directory + " has no Build Result")
	}
	return *manifest, nil
}

// Sync publishes newly accepted public Outputs.
func (r *FileBuildResult) Sync(input BuildResultSync) (BuildResultManifest, error) {
	manifest, err := r.Read()
	if err != nil {
		return BuildResultManifest{}, err
	}
	if manifest.Outcome != "" {
		return manifest, nil
	}
	writerPath := filepath.Join(r.Directory, writerStateName)
	payload, err := readBytes(writerPath)
	if err != nil {
		return BuildResultManifest{}, err
	}
	writer := DecodeBuildResultWriterState(DecodeBuildResultJson(payload, writerPath), writerPath)
	directory := r.Directory
	updated, err := SyncBuildResultOutputs(BuildResultSyncInput{
		Manifest: manifest,
		Writer:   writer,
		Sync:     input,
		Target: fileWriteTarget{
			writeResource: func(path string, artifact protocol.BlobRef, source BuildResultResourceSource) error {
				return copyArtifactAtomic(source, artifact, filepath.Join(directory, path))
			},
			writeValue: func(path string, document BuildResultValueDocument) error {
				return writeJSONAtomic(filepath.Join(directory, path), document)
			},
		},
	})
	if err != nil {
		return BuildResultManifest{}, err
	}
	if !updated.Changed {
		return manifest, nil
	}
	if err := writeJSONAtomic(filepath.Join(r.Directory, writerStateName), updated.Writer); err != nil {
		return BuildResultManifest{}, err
	}
	if err := writeManifestAtomic(filepath.Join(r.Directory, manifestName), updated.Manifest); err != nil {
		return BuildResultManifest{}, err
	}
	return updated.Manifest, nil
}

// Finish records the terminal outcome and drops private writer state.
func (r *FileBuildResult) Finish(input BuildResultFinish) (BuildResultManifest, error) {
	manifest, err := r.Read()
	if err != nil {
		return BuildResultManifest{}, err
	}
	writerPath := filepath.Join(r.Directory, writerStateName)
	if manifest.Outcome != "" {
		if manifest.Outcome != input.Outcome || !sameStringPtr(manifest.Failure, input.Failure) {
			panicErr("Build Result " + manifest.ID + " is already finished with a different outcome")
		}
		if err := removeFile(writerPath); err != nil {
			return BuildResultManifest{}, err
		}
		return manifest, nil
	}
	executionLog, err := PreserveExecutionLog(input.ExecutionLog, func(path string, chunks io.Reader, _ string) error {
		return writeStreamAtomic(filepath.Join(r.Directory, path), chunks)
	})
	if err != nil {
		return BuildResultManifest{}, err
	}
	now := time.Now().UnixMilli()
	updated := manifest
	updated.Outcome = input.Outcome
	if executionLog != nil {
		updated.ExecutionLog = executionLog
	}
	if input.Operations != nil {
		updated.Operations = input.Operations
	}
	if updated.FinishedAt == nil {
		finishedAt := now
		updated.FinishedAt = &finishedAt
	}
	if input.Failure != nil {
		updated.Failure = input.Failure
	}
	if err := writeManifestAtomic(filepath.Join(r.Directory, manifestName), updated); err != nil {
		return BuildResultManifest{}, err
	}
	if err := removeFile(writerPath); err != nil {
		return BuildResultManifest{}, err
	}
	return updated, nil
}

type fileWriteTarget struct {
	writeResource func(path string, artifact protocol.BlobRef, source BuildResultResourceSource) error
	writeValue    func(path string, document BuildResultValueDocument) error
}

func (t fileWriteTarget) WriteResource(path string, artifact protocol.BlobRef, source BuildResultResourceSource) error {
	return t.writeResource(path, artifact, source)
}

func (t fileWriteTarget) WriteValue(path string, document BuildResultValueDocument) error {
	return t.writeValue(path, document)
}

// FileBuildResultRepository is the default directory-backed Result repository.
type FileBuildResultRepository struct {
	ExternalFiles ExternalFileAccess
	Root          string
}

// NewFileBuildResultRepository resolves root and keeps the external-file reader.
func NewFileBuildResultRepository(root string, external ...ExternalFileAccess) *FileBuildResultRepository {
	if strings.TrimSpace(root) == "" {
		panicErr("Build Result root must not be empty")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		panic(err)
	}
	return &FileBuildResultRepository{ExternalFiles: externalFileAccess(external), Root: absolute}
}

// Create prepares one unfinished Result.
func (r *FileBuildResultRepository) Create(seed BuildResultSeed) (*FileBuildResult, error) {
	return CreateFileBuildResult(r.Root, seed, r.ExternalFiles)
}

// OpenWriter opens an existing Result, or returns nil when it is absent.
func (r *FileBuildResultRepository) OpenWriter(build string) (*FileBuildResult, error) {
	directory := BuildResultDirectory(r.Root, build)
	manifest, err := ReadBuildResult(directory)
	if err != nil || manifest == nil {
		return nil, err
	}
	return OpenFileBuildResult(directory)
}

// RemoveIncomplete deletes one exact unfinished Result.
func (r *FileBuildResultRepository) RemoveIncomplete(build string) error {
	manifest, err := r.Read(build)
	if err != nil || manifest == nil {
		return err
	}
	if manifest.Outcome != "" {
		panicErr("Finished Build Result " + build + " cannot be removed")
	}
	return os.RemoveAll(BuildResultDirectory(r.Root, build))
}

// Read returns one manifest, or nil when it does not exist.
func (r *FileBuildResultRepository) Read(build string) (*BuildResultManifest, error) {
	return ReadBuildResult(BuildResultDirectory(r.Root, build))
}

// UpdatePresentation edits human presentation stored in this finished Result.
func (r *FileBuildResultRepository) UpdatePresentation(build string, update BuildResultPresentationUpdate) (BuildResultManifest, error) {
	manifest, err := r.Read(build)
	if err != nil {
		return BuildResultManifest{}, err
	}
	if manifest == nil {
		panicErr("Build Result " + build + " does not exist")
	}
	updated := ApplyBuildResultPresentation(*manifest, update)
	if err := writeManifestAtomic(filepath.Join(BuildResultDirectory(r.Root, build), manifestName), updated); err != nil {
		return BuildResultManifest{}, err
	}
	return updated, nil
}

// Browse lists finished Results in descending Build-id order.
func (r *FileBuildResultRepository) Browse(request BuildResultBrowseRequest) (BuildResultPage, error) {
	return BrowseBuildResults(r.Root, request)
}

// DescribeOutput describes a terminal Output without opening its content.
func (r *FileBuildResultRepository) DescribeOutput(build, output string) (*RepositoryBuildResultOutputDescription, error) {
	return DescribeBuildResultOutput(r.Root, build, output, r.ExternalFiles)
}

// Resolve follows forwarding and returns the portable terminal Output.
func (r *FileBuildResultRepository) Resolve(build, output string) (*RepositoryBuildResultOutput, error) {
	resolved, err := ResolveBuildResultOutput(r.Root, build, output, r.ExternalFiles)
	if err != nil || resolved == nil {
		return nil, err
	}
	return &RepositoryBuildResultOutput{
		Build: resolved.Build, Output: resolved.Output, Type: resolved.Type, Value: resolved.Value,
	}, nil
}

// DescribeFile reads current metadata for a file reference.
func (r *FileBuildResultRepository) DescribeFile(_ string, file BuildResultFileRef) (BuildResultFileRef, error) {
	return CurrentFileReference(file, r.ExternalFiles)
}

// OpenFile streams a Result file or an external file, optionally over a byte range.
func (r *FileBuildResultRepository) OpenFile(build string, file BuildResultFileRef, fileRange *BuildResultFileRange) (io.ReadCloser, error) {
	if file.Kind == "external-file" {
		return r.ExternalFiles.Open(file.URI, fileRange)
	}
	owner := build
	if file.Build != nil {
		owner = *file.Build
	}
	path := containedResultPath(BuildResultDirectory(r.Root, owner), file.Path)
	if !fileExists(path) {
		return nil, nil
	}
	if fileRange == nil {
		return openShared(path)
	}
	if !safeRangeBound(fileRange.Start) {
		panicErr("Build Result file range start is invalid")
	}
	if !safeRangeBound(fileRange.EndExclusive) || fileRange.EndExclusive <= fileRange.Start {
		panicErr("Build Result file range end is invalid")
	}
	if fileRange.EndExclusive > file.Size {
		panicErr("Build Result file range exceeds the declared file size")
	}
	opened, err := openShared(path)
	if err != nil {
		return nil, err
	}
	if _, err := opened.Seek(fileRange.Start, io.SeekStart); err != nil {
		opened.Close()
		return nil, err
	}
	return sectionCloser{Reader: io.LimitReader(opened, fileRange.EndExclusive-fileRange.Start), Closer: opened}, nil
}
