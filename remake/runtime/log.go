// Copyright 2026 Hypit.AI
// Translated from @hypit/runtime/src/log.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package runtime

import (
	"encoding/json"
	"errors"
	"unicode/utf8"
)

// ExecutionDiagnostic is provider-authored text suitable for users.
type ExecutionDiagnostic struct {
	Level   string
	Message string
	Stream  *string
}

// ExecutionLogEvent is one provider log event.
// Kind is started, completed, failed, phase, or diagnostic.
type ExecutionLogEvent struct {
	Endpoint string
	Kind     string
	Message  string
	Phase    string
	Level    string
	Stream   *string
}

// ExecutionLogRecord is one persisted log line.
type ExecutionLogRecord struct {
	ExecutionLogEvent
	Format  string
	Time    float64
	Command string
}

// ExecutionLogView is a bounded tail of a log.
type ExecutionLogView struct {
	Records []ExecutionLogRecord
	Total   int
}

// ReadExecutionLog reads a bounded tail without keeping more than limit records.
// chunks are the successive byte pieces of an async iterable. Only complete lines are visible.
func ReadExecutionLog(chunks [][]byte, limit int) ExecutionLogView {
	if !safeNumber(float64(limit)) || limit < 1 {
		panic(errors.New("Log line count must be a positive integer"))
	}
	records := make([]ExecutionLogRecord, 0)
	slots := make([]ExecutionLogRecord, limit)
	var decoder utf8Stream
	pending := ""
	total := 0
	consume := func(line string) {
		if jsTrim(line) == "" {
			return
		}
		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			panic(err)
		}
		format, _ := raw["format"].(string)
		if format != "hypit.execution-log@1" {
			panic(errors.New("Unsupported execution log format"))
		}
		record := executionRecord(raw)
		slots[total%limit] = record
		total++
	}
	for _, chunk := range chunks {
		pending += decoder.write(chunk)
		for {
			newline := indexByte(pending, '\n')
			if newline < 0 {
				break
			}
			consume(pending[:newline])
			pending = pending[newline+1:]
		}
	}
	offset := 0
	if total > limit {
		offset = total % limit
	}
	count := total
	if count > limit {
		count = limit
	}
	records = make([]ExecutionLogRecord, 0, count)
	if total == 0 {
		return ExecutionLogView{Records: records, Total: 0}
	}
	if total <= limit {
		records = append(records, slots[:total]...)
		return ExecutionLogView{Records: records, Total: total}
	}
	records = append(records, slots[offset:]...)
	records = append(records, slots[:offset]...)
	return ExecutionLogView{Records: records, Total: total}
}

func executionRecord(raw map[string]any) ExecutionLogRecord {
	record := ExecutionLogRecord{Format: "hypit.execution-log@1"}
	record.Endpoint, _ = raw["endpoint"].(string)
	record.Kind, _ = raw["kind"].(string)
	record.Message, _ = raw["message"].(string)
	record.Phase, _ = raw["phase"].(string)
	record.Level, _ = raw["level"].(string)
	record.Command, _ = raw["command"].(string)
	if stream, ok := raw["stream"].(string); ok {
		record.Stream = &stream
	}
	if time, ok := raw["time"].(float64); ok {
		record.Time = time
	}
	return record
}

func indexByte(value string, needle byte) int {
	for i := 0; i < len(value); i++ {
		if value[i] == needle {
			return i
		}
	}
	return -1
}

// utf8Stream is TextDecoder with stream:true and fatal:false.
// A trailing incomplete sequence stays buffered and is not flushed.
type utf8Stream struct {
	partial []byte
}

func (d *utf8Stream) write(chunk []byte) string {
	data := append(append([]byte{}, d.partial...), chunk...)
	d.partial = nil
	var out []byte
	for len(data) > 0 {
		if !utf8.FullRune(data) {
			d.partial = append([]byte{}, data...)
			break
		}
		r, size := utf8.DecodeRune(data)
		out = utf8.AppendRune(out, r)
		data = data[size:]
	}
	return string(out)
}
