// Copyright 2026 Hypit.AI
// Translated from @hypit/result/src/decode.ts (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package result

import (
	"encoding/json"
	"fmt"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// DecodeBuildResultJson parses Result JSON and keeps invalid UTF-8 inside the Result boundary.
func DecodeBuildResultJson(value []byte, subject string) any {
	var out any
	if err := json.Unmarshal(value, &out); err != nil {
		panicErr(subject + " is not valid JSON: " + err.Error())
	}
	return out
}

func buildID(value any, subject string) string {
	id := text(value, subject)
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		protocol.AssertOrderedBuildId(id)
	}()
	if recovered != nil {
		panicErr(subject + " is invalid: " + panicMessage(recovered))
	}
	return id
}

func typeRef(value any, subject string) protocol.TypeRef {
	item := object(value, subject)
	module := object(item["module"], subject+".module")
	return protocol.TypeRef{
		Module: protocol.ModuleRef{
			Name:    text(module["name"], subject+".module.name"),
			Version: text(module["version"], subject+".module.version"),
		},
		Name: text(item["name"], subject+".name"),
	}
}

func resultPath(value any, subject string) string {
	AssertBuildResultPath(value, subject)
	return value.(string)
}

func outputValue(value any, subject string) BuildResultOutputValue {
	item := object(value, subject)
	kind, _ := item["kind"].(string)
	switch kind {
	case "build-file", "external-file":
		return fileOutput(fileRefFromAny(item, subject))
	case "build-output":
		return BuildResultOutputValue{
			Kind:   "build-output",
			Build:  buildID(item["build"], subject+".build"),
			Output: text(item["output"], subject+".output"),
		}
	case "inline":
		scalarValue, present := item["value"]
		if !present || !scalar(scalarValue) {
			panicErr(subject + ".value must be a scalar")
		}
		return inlineOutput(scalarValue)
	case "value":
		return BuildResultOutputValue{Kind: "value", Path: resultPath(item["path"], subject+".path")}
	default:
		panicErr(subject + ".kind is not a Result value kind")
		return BuildResultOutputValue{}
	}
}

func fileOutput(file BuildResultFileRef) BuildResultOutputValue {
	return BuildResultOutputValue{Kind: file.Kind, File: &file}
}

func inlineOutput(value any) BuildResultOutputValue {
	return BuildResultOutputValue{Kind: "inline", Inline: value}
}

func outputs(value any, subject string) map[string]BuildResultOutput {
	raw := object(value, subject)
	out := make(map[string]BuildResultOutput, len(raw))
	for name, child := range raw {
		text(name, subject+" Output name")
		label := jsonStringify(name)
		item := object(child, subject+"["+label+"]")
		output := BuildResultOutput{
			Type:  typeRef(item["type"], subject+"["+label+"].type"),
			Value: outputValue(item["value"], subject+"["+label+"].value"),
		}
		if displayName, present := item["displayName"]; present {
			output.DisplayName = optionalText(displayName, true, subject+".displayName")
		}
		out[name] = output
	}
	return out
}

func optionalTime(value any, present bool, subject string) *int64 {
	if !present {
		return nil
	}
	parsed, ok := asSafeNonNegative(value)
	if !ok {
		panicErr(subject + " must be a non-negative integer timestamp")
	}
	return &parsed
}

func progressNumber(value any, subject string) float64 {
	parsed, ok := asFinite(value)
	if !ok || parsed < 0 {
		panicErr(subject + " must be non-negative")
	}
	return parsed
}

func operationReceipts(value any, subject string) []BuildResultOperation {
	items, ok := value.([]any)
	if !ok {
		panicErr(subject + " must be an array")
	}
	out := make([]BuildResultOperation, len(items))
	for index, raw := range items {
		item := object(raw, subject)
		status, _ := item["status"].(string)
		switch status {
		case "pending", "completed", "failed", "cancelled":
		default:
			panicErr(subject + " has an invalid Operation status")
		}
		op := BuildResultOperation{
			Operation: text(item["operation"], "Operation id"),
			Command:   text(item["command"], "Operation command"),
			Endpoint:  text(item["endpoint"], "Operation Endpoint"),
			Status:    status,
		}
		if raw, present := item["need"]; present {
			need := object(raw, subject+".need")
			op.Need = &BuildResultOperationNeed{
				ID:         text(need["id"], "Need id"),
				Capability: typeRef(need["capability"], "Need capability"),
			}
		}
		if raw, present := item["createdAt"]; present {
			op.CreatedAt = optionalTime(raw, true, "Operation createdAt")
		}
		if raw, present := item["acknowledgedAt"]; present {
			op.AcknowledgedAt = optionalTime(raw, true, "Operation acknowledgedAt")
		}
		if raw, present := item["endedAt"]; present {
			op.EndedAt = optionalTime(raw, true, "Operation endedAt")
		}
		if raw, present := item["progress"]; present {
			progress := object(raw, subject+".progress")
			parsed := BuildResultProgress{Phase: text(progress["phase"], "Operation phase")}
			if completed, ok := progress["completed"]; ok {
				n := progressNumber(completed, "Operation completed units")
				parsed.Completed = &n
			}
			if total, ok := progress["total"]; ok {
				n := progressNumber(total, "Operation total units")
				parsed.Total = &n
			}
			if unit, ok := progress["unit"]; ok {
				parsed.Unit = optionalText(unit, true, "Operation unit")
			}
			op.Progress = &parsed
		}
		if raw, present := item["cancellation"]; present {
			cancellation := object(raw, subject+".cancellation")
			outcome, _ := cancellation["outcome"].(string)
			switch outcome {
			case "confirmed", "accepted", "unsupported", "too-late", "failed":
			default:
				panicErr(subject + " has an invalid cancellation acknowledgement")
			}
			parsed := BuildResultCancellation{Outcome: outcome}
			if message, ok := cancellation["message"]; ok {
				parsed.Message = optionalText(message, true, "Cancellation error")
			}
			op.Cancellation = &parsed
		}
		if raw, present := item["pool"]; present {
			op.Pool = optionalText(raw, true, "Operation pool")
		}
		if raw, present := item["credentials"]; present {
			creds := object(raw, subject+".credentials")
			parsed := make(map[string]BuildResultCredential, len(creds))
			for slot, rawRef := range creds {
				ref := object(rawRef, subject+".credential")
				parsed[slot] = BuildResultCredential{
					Store: text(ref["store"], "credential store"),
					Key:   text(ref["key"], "credential reference"),
				}
			}
			op.Credentials = parsed
		}
		if raw, present := item["receipt"]; present {
			receipt := object(raw, subject+".receipt")
			parsed := BuildResultReceipt{ID: text(receipt["id"], "Remote receipt id")}
			if rawURL, ok := receipt["url"]; ok {
				parsed.URL = optionalText(rawURL, true, "Remote receipt URL")
			}
			op.Receipt = &parsed
		}
		if raw, present := item["failure"]; present {
			failure := object(raw, subject+".failure")
			op.Failure = &BuildResultOperationFailure{
				Code:    text(failure["code"], "Operation failure code"),
				Message: text(failure["message"], "Operation failure message"),
			}
		}
		out[index] = op
	}
	return out
}

// DecodeBuildResultManifest decodes one Result document and attaches the repository Build id.
func DecodeBuildResultManifest(value any, build string, subject ...string) BuildResultManifest {
	sub := subjectOr("Build "+build+" result.json", subject)
	id := buildID(build, "Build Result address")
	item := object(value, sub)
	if item["format"] != "hypit.build-result@1" {
		panicErr(sub + " is not a Hypit Build Result")
	}
	source := object(item["source"], sub+".source")
	manifest := BuildResultManifest{
		Format:  "hypit.build-result@1",
		ID:      id,
		Source:  BuildResultSource{ID: text(source["id"], sub+".source.id")},
		Targets: texts(item["targets"], sub+".targets"),
		Outputs: outputs(item["outputs"], sub+".outputs"),
	}
	if raw, present := item["title"]; present {
		manifest.Title = optionalText(raw, true, sub+".title")
	}
	if raw, present := item["note"]; present {
		manifest.Note = optionalText(raw, true, sub+".note")
	}
	if raw, present := item["highlightedOutputs"]; present {
		manifest.HighlightedOutputs = texts(raw, sub+".highlightedOutputs")
	}
	if raw, present := item["run"]; present {
		run := object(raw, sub+".run")
		manifest.Run = &BuildResultSource{ID: text(run["id"], sub+".run.id")}
	}
	finishedAt := optionalTime(item["finishedAt"], hasKey(item, "finishedAt"), sub+".finishedAt")
	var outcome string
	if raw, present := item["outcome"]; present {
		parsed, _ := raw.(string)
		if parsed != "complete" && parsed != "failed" && parsed != "cancelled" {
			panicErr(sub + ".outcome is not a Build Result outcome")
		}
		outcome = parsed
	}
	if (outcome == "") != (finishedAt == nil) {
		panicErr(sub + " must be either unfinished or have both outcome and finishedAt")
	}
	manifest.FinishedAt = finishedAt
	manifest.Outcome = outcome
	if raw, present := item["failure"]; present {
		manifest.Failure = optionalString(raw, true, sub+".failure")
	}
	if raw, present := item["executionLog"]; present {
		ref := fileRefFromAny(raw, sub+".executionLog")
		manifest.ExecutionLog = &ref
	}
	if raw, present := item["operations"]; present {
		ops := operationReceipts(raw, sub+".operations")
		manifest.Operations = &ops
	}
	return manifest
}

// EncodeBuildResultManifest drops the repository-injected Build id.
func EncodeBuildResultManifest(manifest BuildResultManifest) BuildResultManifest {
	manifest.ID = ""
	return manifest
}

func pathMap(value any, subject string) map[string]string {
	item := object(value, subject)
	out := make(map[string]string, len(item))
	for identity, path := range item {
		key := text(identity, subject+" identity")
		out[key] = resultPath(path, subject+"["+jsonStringify(identity)+"]")
	}
	return out
}

// DecodeBuildResultWriterState decodes the private unfinished-Result writer state.
func DecodeBuildResultWriterState(value any, subject ...string) BuildResultWriterState {
	sub := subjectOr("Build Result writer state", subject)
	item := object(value, sub)
	rawPublished, ok := item["publishedOutputs"].([]any)
	if !ok {
		panicErr(sub + ".publishedOutputs must be an array")
	}
	rawForwards, ok := item["forwards"].([]any)
	if !ok {
		panicErr(sub + ".forwards must be an array")
	}
	names := map[string]struct{}{}
	logical := map[string]struct{}{}
	published := make([]BuildResultPublishedOutput, len(rawPublished))
	for index, raw := range rawPublished {
		entry := object(raw, fmt.Sprintf("%s.publishedOutputs[%d]", sub, index))
		name := text(entry["name"], fmt.Sprintf("%s.publishedOutputs[%d].name", sub, index))
		output := text(entry["output"], fmt.Sprintf("%s.publishedOutputs[%d].output", sub, index))
		if _, exists := names[name]; exists {
			panicErr(sub + " repeats published name " + name)
		}
		if _, exists := logical[output]; exists {
			panicErr(sub + " publishes Logical Output " + output + " more than once")
		}
		names[name] = struct{}{}
		logical[output] = struct{}{}
		published[index] = BuildResultPublishedOutput{Name: name, Output: output}
		if displayName, present := entry["displayName"]; present {
			published[index].DisplayName = optionalText(displayName, true, sub+".displayName")
		}
	}
	forwarded := map[string]struct{}{}
	forwards := make([]BuildResultForward, len(rawForwards))
	for index, raw := range rawForwards {
		entry := object(raw, fmt.Sprintf("%s.forwards[%d]", sub, index))
		output := text(entry["output"], fmt.Sprintf("%s.forwards[%d].output", sub, index))
		if _, exists := forwarded[output]; exists {
			panicErr(sub + " forwards Logical Output " + output + " more than once")
		}
		forwarded[output] = struct{}{}
		parsedType := typeRef(entry["type"], fmt.Sprintf("%s.forwards[%d].type", sub, index))
		forwards[index] = BuildResultForward{
			Output:       output,
			Build:        buildID(entry["build"], fmt.Sprintf("%s.forwards[%d].build", sub, index)),
			SourceOutput: text(entry["sourceOutput"], fmt.Sprintf("%s.forwards[%d].sourceOutput", sub, index)),
			Type:         &parsedType,
		}
	}
	state := BuildResultWriterState{
		Resources:        pathMap(item["resources"], sub+".resources"),
		Values:           pathMap(item["values"], sub+".values"),
		PublishedOutputs: published,
		Forwards:         forwards,
	}
	if rawRefs, present := item["resourceReferences"]; present {
		refs := object(rawRefs, sub+".resourceReferences")
		state.ResourceReferences = make(map[string]BuildResultFileRef, len(refs))
		for resource, reference := range refs {
			state.ResourceReferences[resource] = fileRefFromAny(reference, sub+".resourceReferences."+resource)
		}
	}
	return state
}

// DecodeBuildResultValueDocument validates and returns one composite value document.
func DecodeBuildResultValueDocument(value any, subject ...string) BuildResultValueDocument {
	sub := subjectOr("Build Result value", subject)
	AssertBuildResultValueDocument(value, sub)
	item := value.(map[string]any)
	rawResources := item["resources"].([]any)
	resources := make([]BuildResultResourceBinding, len(rawResources))
	for index, raw := range rawResources {
		binding := raw.(map[string]any)
		resources[index] = BuildResultResourceBinding{
			At:   append([]any{}, binding["at"].([]any)...),
			File: fileRefFromAny(binding["file"], fmt.Sprintf("%s.resources[%d].file", sub, index)),
		}
	}
	if resources == nil {
		resources = []BuildResultResourceBinding{}
	}
	return BuildResultValueDocument{
		Format:    "hypit.result-value@1",
		Value:     item["value"],
		Resources: resources,
	}
}
