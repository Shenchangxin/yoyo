// Copyright 2026 Hypit.AI
// Translated from @hypit/protocol (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

func Canonicalize(value any) any {
	return normalize(value, "$input")
}

func CanonicalStringify(value any) string {
	return stringify(Canonicalize(value))
}

func IsResourceId(value string) bool {
	if len(value) < 5 || value[:4] != "res_" {
		return false
	}
	for _, r := range value[4:] {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == ':' || r == '-':
		default:
			return false
		}
	}
	return len(value) > 4
}

func normalize(value any, path string) any {
	if value == nil {
		return nil
	}
	if _, ok := value.(Undefined); ok {
		panic(&SvmlError{Code: "NON_CANONICAL_UNDEFINED", Message: path + " is undefined"})
	}
	switch v := value.(type) {
	case bool, string:
		return v
	case float64:
		return normalizeNumber(v, path)
	case float32:
		return normalizeNumber(float64(v), path)
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return normalizeNumber(float64(v), path)
	case uint:
		return normalizeNumber(float64(v), path)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return normalizeNumber(float64(v), path)
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			panic(&SvmlError{Code: "NON_CANONICAL_NUMBER", Message: path + " must be finite"})
		}
		return normalizeNumber(f, path)
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = normalize(rv.Index(i).Interface(), fmt.Sprintf("%s[%d]", path, i))
		}
		return out
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			panic(&SvmlError{Code: "NON_CANONICAL_VALUE", Message: path + " contains " + rv.Kind().String()})
		}
		keys := make([]string, 0, rv.Len())
		for _, key := range rv.MapKeys() {
			keys = append(keys, key.String())
		}
		sortKeys(keys)
		out := make(map[string]any, len(keys))
		for _, key := range keys {
			item := rv.MapIndex(reflect.ValueOf(key)).Interface()
			if _, ok := item.(Undefined); ok {
				panic(&SvmlError{Code: "NON_CANONICAL_UNDEFINED", Message: path + "." + key + " is undefined"})
			}
			out[key] = normalize(item, path+"."+key)
		}
		return out
	default:
		panic(&SvmlError{Code: "NON_CANONICAL_VALUE", Message: path + " contains " + rv.Kind().String()})
	}
}

func normalizeNumber(value float64, path string) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		panic(&SvmlError{Code: "NON_CANONICAL_NUMBER", Message: path + " must be finite"})
	}
	if value == 0 {
		return 0
	}
	return value
}

func stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case string:
		return jsonString(v)
	case float64:
		return jsonNumber(v)
	case []any:
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i, item := range v {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(stringify(item))
		}
		buf.WriteByte(']')
		return buf.String()
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sortKeys(keys)
		var buf bytes.Buffer
		buf.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(jsonString(key))
			buf.WriteByte(':')
			buf.WriteString(stringify(v[key]))
		}
		buf.WriteByte('}')
		return buf.String()
	default:
		panic(&SvmlError{Code: "NON_CANONICAL_VALUE", Message: "$input contains " + reflect.TypeOf(value).String()})
	}
}

func jsonString(value string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		panic(&SvmlError{Code: "NON_CANONICAL_VALUE", Message: err.Error()})
	}
	out := buf.Bytes()
	if len(out) > 0 && out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	// JSON.stringify escapes these two separators; encoding/json does not.
	text := strings.ReplaceAll(string(out), "\u2028", `\u2028`)
	return strings.ReplaceAll(text, "\u2029", `\u2029`)
}

func jsonNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
