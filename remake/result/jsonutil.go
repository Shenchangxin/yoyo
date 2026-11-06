// Copyright 2026 Hypit.AI
// Translated from @hypit/result (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package result

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const maxSafeInteger int64 = 9007199254740991

func panicErr(message string) {
	panic(errors.New(message))
}

func panicMessage(value any) string {
	switch v := value.(type) {
	case error:
		return v.Error()
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func asObject(value any) (map[string]any, bool) {
	if value == nil {
		return nil, false
	}
	item, ok := value.(map[string]any)
	return item, ok
}

func object(value any, subject string) map[string]any {
	item, ok := asObject(value)
	if !ok {
		panicErr(subject + " must be an object")
	}
	return item
}

func text(value any, subject string) string {
	item, ok := value.(string)
	if !ok || item == "" {
		panicErr(subject + " must be a non-empty string")
	}
	return item
}

func optionalText(value any, present bool, subject string) *string {
	if !present {
		return nil
	}
	item := text(value, subject)
	return &item
}

func optionalString(value any, present bool, subject string) *string {
	if !present {
		return nil
	}
	item, ok := value.(string)
	if !ok {
		panicErr(subject + " must be a string")
	}
	return &item
}

func texts(value any, subject string) []string {
	items, ok := value.([]any)
	if !ok {
		panicErr(subject + " must be an array")
	}
	out := make([]string, len(items))
	for index, item := range items {
		out[index] = text(item, subject+"["+strconv.Itoa(index)+"]")
	}
	return out
}

func subjectOr(fallback string, subject []string) string {
	if len(subject) == 0 {
		return fallback
	}
	return subject[0]
}

func asSafeInteger(value any) (int64, bool) {
	switch n := value.(type) {
	case int:
		return safeInt(int64(n))
	case int8:
		return int64(n), true
	case int16:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return safeInt(n)
	case uint8:
		return int64(n), true
	case uint16:
		return int64(n), true
	case uint32:
		return safeInt(int64(n))
	case uint:
		if uint64(n) > uint64(maxSafeInteger) {
			return 0, false
		}
		return int64(n), true
	case uint64:
		if n > uint64(maxSafeInteger) {
			return 0, false
		}
		return int64(n), true
	case float32:
		return asSafeInteger(float64(n))
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) || n != math.Trunc(n) {
			return 0, false
		}
		if n > float64(maxSafeInteger) || n < -float64(maxSafeInteger) {
			return 0, false
		}
		return int64(n), true
	case json.Number:
		parsed, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return safeInt(parsed)
	default:
		return 0, false
	}
}

func safeInt(n int64) (int64, bool) {
	if n > maxSafeInteger || n < -maxSafeInteger {
		return 0, false
	}
	return n, true
}

func asSafeNonNegative(value any) (int64, bool) {
	n, ok := asSafeInteger(value)
	if !ok || n < 0 {
		return 0, false
	}
	return n, true
}

func asFinite(value any) (float64, bool) {
	switch n := value.(type) {
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return 0, false
		}
		return n, true
	case float32:
		return asFinite(float64(n))
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case json.Number:
		parsed, err := n.Float64()
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func scalar(value any) bool {
	switch n := value.(type) {
	case nil, bool, string:
		return true
	case float64:
		return !math.IsNaN(n) && !math.IsInf(n, 0)
	case float32:
		return !math.IsNaN(float64(n)) && !math.IsInf(float64(n), 0)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case json.Number:
		parsed, err := n.Float64()
		return err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0)
	default:
		return false
	}
}

func jsonMarshal(value any) ([]byte, error) {
	return encodeJSON(value, false)
}

func jsonIndent(value any) ([]byte, error) {
	return encodeJSON(value, true)
}

func encodeJSON(value any, indent bool) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	out := buf.Bytes()
	if !indent {
		out = bytes.TrimSuffix(out, []byte("\n"))
	}
	return escapeJSONSeparators(out), nil
}

func escapeJSONSeparators(in []byte) []byte {
	text := string(in)
	text = strings.ReplaceAll(text, "\u2028", `\u2028`)
	text = strings.ReplaceAll(text, "\u2029", `\u2029`)
	return []byte(text)
}

func jsonStringify(value any) string {
	raw, err := jsonMarshal(value)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func hasKey(item map[string]any, key string) bool {
	_, ok := item[key]
	return ok
}

func sameStringPtr(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func stringMapValues(in map[string]string) []string {
	out := make([]string, 0, len(in))
	for _, value := range in {
		out = append(out, value)
	}
	return out
}

func externalFileAccess(external []ExternalFileAccess) ExternalFileAccess {
	if len(external) == 0 || external[0] == nil {
		return LocalExternalFiles
	}
	return external[0]
}
