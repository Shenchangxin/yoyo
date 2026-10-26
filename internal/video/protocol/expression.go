// expression.go 用少量 $ 运算符把清单模板求值成供应商请求，不执行插件代码。

package protocol

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// evaluateManifestValue 求值清单模板。数字和布尔原样返回。整段是 ${路径} 的字符串按路径取值，其他字符串原样返回。数组丢掉 nil。只有一个 $ 键的对象当成运算符。普通对象逐项求值并丢掉 nil。其余类型先转成 JSON 再求值。
// template 为 nil、布尔或数字时原样返回；字符串整段是 ${路径} 时从 env 取值；数组逐项求值并丢掉 nil；只有一个 $ 键的对象交给 evaluateManifestOperator。
// env 传给 manifestPathValue、递归求值和 evaluateManifestOperator。
func evaluateManifestValue(template any, env map[string]any) (any, error) {
	switch value := template.(type) {
	case nil, bool, float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return value, nil
	case string:
		trimmed := strings.TrimSpace(value)
		if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") {
			return manifestPathValue(env, strings.TrimSpace(trimmed[2:len(trimmed)-1])), nil
		}
		return value, nil
	case []any:
		result := make([]any, 0, len(value))
		for _, item := range value {
			evaluated, err := evaluateManifestValue(item, env)
			if err != nil {
				return nil, err
			}
			if evaluated != nil {
				result = append(result, evaluated)
			}
		}
		return result, nil
	case map[string]any:
		if len(value) == 1 {
			for operator, operand := range value {
				if strings.HasPrefix(operator, "$") {
					return evaluateManifestOperator(operator, operand, env)
				}
			}
		}
		result := make(map[string]any, len(value))
		for key, item := range value {
			evaluated, err := evaluateManifestValue(item, env)
			if err != nil {
				return nil, fmt.Errorf("evaluate %s: %w", key, err)
			}
			if evaluated != nil {
				result[key] = evaluated
			}
		}
		return result, nil
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("unsupported manifest template value %T", template)
		}
		var normalized any
		if err := json.Unmarshal(encoded, &normalized); err != nil {
			return nil, err
		}
		return evaluateManifestValue(normalized, env)
	}
}

// evaluateManifestOperator 实现 $ref、$literal、$coalesce、$concat、$map、$filter、$indexObject、$first、$last、$at、$len、$split、$if、$switch、$omitEmpty、大小写和类型转换、$merge、$sortByOrder、四则和比较。未知运算符返回 unsupported manifest operator。$divide 的除数不能为 0。$ceilStep 的步长必须为正。$json 把值编成 JSON 文本。$dataMime 和 $dataPayload 交给同名函数处理 data URL。
// operand 在 $ref 时要求是字符串路径，在 $literal 时原样返回，在多数运算符里先求值或要求是数组、对象。
// env 传给 manifestPathValue、evaluateManifestValue 和 cloneManifestEnv。
func evaluateManifestOperator(operator string, operand any, env map[string]any) (any, error) {
	switch operator {
	case "$ref":
		path, ok := operand.(string)
		if !ok {
			return nil, fmt.Errorf("$ref requires a string path")
		}
		return manifestPathValue(env, path), nil
	case "$literal":
		return operand, nil
	case "$coalesce", "$default":
		values, ok := operand.([]any)
		if !ok {
			return nil, fmt.Errorf("%s requires an array", operator)
		}
		for _, item := range values {
			value, err := evaluateManifestValue(item, env)
			if err != nil {
				return nil, err
			}
			if !manifestEmpty(value) {
				return value, nil
			}
		}
		return nil, nil
	case "$concat":
		values, ok := operand.([]any)
		if !ok {
			return nil, fmt.Errorf("$concat requires an array")
		}
		var result strings.Builder
		for _, item := range values {
			value, err := evaluateManifestValue(item, env)
			if err != nil {
				return nil, err
			}
			if value != nil {
				result.WriteString(manifestString(value))
			}
		}
		return result.String(), nil
	case "$concatArrays":
		values, ok := operand.([]any)
		if !ok {
			return nil, fmt.Errorf("$concatArrays requires an array")
		}
		result := make([]any, 0)
		for _, item := range values {
			value, err := evaluateManifestValue(item, env)
			if err != nil {
				return nil, err
			}
			result = append(result, manifestArray(value)...)
		}
		return result, nil
	case "$map", "$filter":
		spec, ok := operand.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s requires an object", operator)
		}
		from, err := evaluateManifestValue(spec["from"], env)
		if err != nil {
			return nil, err
		}
		alias := strings.TrimSpace(manifestString(spec["as"]))
		if alias == "" {
			alias = "item"
		}
		items := manifestArray(from)
		result := make([]any, 0, len(items))
		for index, item := range items {
			child := cloneManifestEnv(env)
			child[alias] = item
			child[alias+"Index"] = index
			if operator == "$filter" {
				condition, err := evaluateManifestValue(spec["where"], child)
				if err != nil {
					return nil, err
				}
				if manifestTruthy(condition) {
					result = append(result, item)
				}
				continue
			}
			mapped, err := evaluateManifestValue(spec["in"], child)
			if err != nil {
				return nil, err
			}
			if mapped != nil {
				result = append(result, mapped)
			}
		}
		return result, nil
	case "$indexObject":
		spec, ok := operand.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("$indexObject requires an object")
		}
		from, err := evaluateManifestValue(spec["from"], env)
		if err != nil {
			return nil, err
		}
		alias := strings.TrimSpace(manifestString(spec["as"]))
		if alias == "" {
			alias = "item"
		}
		prefixValue, err := evaluateManifestValue(spec["prefix"], env)
		if err != nil {
			return nil, err
		}
		prefix := manifestString(prefixValue)
		maxItems := len(manifestArray(from))
		if spec["max"] != nil {
			value, err := evaluateManifestValue(spec["max"], env)
			if err != nil {
				return nil, err
			}
			if configured := manifestInt(value); configured >= 0 && configured < maxItems {
				maxItems = configured
			}
		}
		result := make(map[string]any, maxItems)
		for index, item := range manifestArray(from)[:maxItems] {
			child := cloneManifestEnv(env)
			child[alias] = item
			child[alias+"Index"] = index
			value, err := evaluateManifestValue(spec["value"], child)
			if err != nil {
				return nil, err
			}
			if value != nil && !manifestEmpty(value) {
				result[prefix+strconv.Itoa(index)] = value
			}
		}
		return result, nil
	case "$first", "$last":
		value, err := evaluateManifestValue(operand, env)
		if err != nil {
			return nil, err
		}
		items := manifestArray(value)
		if len(items) == 0 {
			return nil, nil
		}
		if operator == "$last" {
			return items[len(items)-1], nil
		}
		return items[0], nil
	case "$at":
		values, ok := operand.([]any)
		if !ok || len(values) != 2 {
			return nil, fmt.Errorf("$at requires [array, index]")
		}
		value, err := evaluateManifestValue(values[0], env)
		if err != nil {
			return nil, err
		}
		indexValue, err := evaluateManifestValue(values[1], env)
		if err != nil {
			return nil, err
		}
		items := manifestArray(value)
		index := manifestInt(indexValue)
		if index < 0 || index >= len(items) {
			return nil, nil
		}
		return items[index], nil
	case "$len":
		value, err := evaluateManifestValue(operand, env)
		if err != nil {
			return nil, err
		}
		switch typed := value.(type) {
		case []any:
			return len(typed), nil
		case map[string]any:
			return len(typed), nil
		case string:
			return len([]rune(typed)), nil
		default:
			return 0, nil
		}
	case "$split":
		spec, ok := operand.([]any)
		if !ok || len(spec) != 2 {
			return nil, fmt.Errorf("$split requires [value, separator]")
		}
		value, err := evaluateManifestValue(spec[0], env)
		if err != nil {
			return nil, err
		}
		separator, err := evaluateManifestValue(spec[1], env)
		if err != nil {
			return nil, err
		}
		sep := manifestString(separator)
		if sep == "" {
			return nil, fmt.Errorf("$split separator must not be empty")
		}
		parts := strings.Split(manifestString(value), sep)
		result := make([]any, 0, len(parts))
		for _, part := range parts {
			result = append(result, part)
		}
		return result, nil
	case "$if":
		spec, ok := operand.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("$if requires an object")
		}
		condition, err := evaluateManifestValue(spec["condition"], env)
		if err != nil {
			return nil, err
		}
		if manifestTruthy(condition) {
			return evaluateManifestValue(spec["then"], env)
		}
		return evaluateManifestValue(spec["else"], env)
	case "$switch":
		spec, ok := operand.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("$switch requires an object")
		}
		for _, rawCase := range manifestArray(spec["cases"]) {
			item, _ := rawCase.(map[string]any)
			when, err := evaluateManifestValue(item["when"], env)
			if err != nil {
				return nil, err
			}
			if manifestTruthy(when) {
				return evaluateManifestValue(item["then"], env)
			}
		}
		return evaluateManifestValue(spec["default"], env)
	case "$omitEmpty":
		value, err := evaluateManifestValue(operand, env)
		if err != nil || manifestEmpty(value) {
			return nil, err
		}
		return value, nil
	case "$lower", "$upper", "$trim", "$toString", "$toInt", "$toFloat", "$toBool", "$dataMime", "$dataPayload", "$json":
		value, err := evaluateManifestValue(operand, env)
		if err != nil {
			return nil, err
		}
		text := manifestString(value)
		switch operator {
		case "$lower":
			return strings.ToLower(text), nil
		case "$upper":
			return strings.ToUpper(text), nil
		case "$trim":
			return strings.TrimSpace(text), nil
		case "$toString":
			return text, nil
		case "$toInt":
			return manifestInt(value), nil
		case "$toFloat":
			parsed, ok := manifestFloatValue(value)
			if !ok {
				return nil, nil
			}
			return parsed, nil
		case "$toBool":
			return manifestTruthy(value), nil
		case "$dataMime":
			return dataMIME(text), nil
		case "$dataPayload":
			return dataPayload(text), nil
		default:
			encoded, err := json.Marshal(value)
			return string(encoded), err
		}
	case "$merge":
		values, ok := operand.([]any)
		if !ok {
			return nil, fmt.Errorf("$merge requires an array")
		}
		result := make(map[string]any)
		for _, item := range values {
			value, err := evaluateManifestValue(item, env)
			if err != nil {
				return nil, err
			}
			for key, entry := range manifestObject(value) {
				result[key] = entry
			}
		}
		return result, nil
	case "$sortByOrder":
		value, err := evaluateManifestValue(operand, env)
		if err != nil {
			return nil, err
		}
		items := append([]any(nil), manifestArray(value)...)
		sort.SliceStable(items, func(i, j int) bool {
			return manifestInt(manifestPathValue(items[i], "order")) < manifestInt(manifestPathValue(items[j], "order"))
		})
		return items, nil
	case "$add", "$multiply", "$divide", "$min", "$max":
		values, ok := operand.([]any)
		if !ok || len(values) == 0 {
			return nil, fmt.Errorf("%s requires an array", operator)
		}
		if operator == "$divide" {
			if len(values) != 2 {
				return nil, fmt.Errorf("$divide requires [numerator, denominator]")
			}
			numerator, err := evaluateManifestValue(values[0], env)
			if err != nil {
				return nil, err
			}
			denominator, err := evaluateManifestValue(values[1], env)
			if err != nil {
				return nil, err
			}
			denom := manifestFloat(denominator)
			if denom == 0 {
				return nil, fmt.Errorf("$divide denominator must not be zero")
			}
			return normalizeManifestNumber(manifestFloat(numerator) / denom), nil
		}
		result := 0.0
		if operator == "$multiply" {
			result = 1
		} else if operator == "$min" {
			result = math.Inf(1)
		} else if operator == "$max" {
			result = math.Inf(-1)
		}
		for _, item := range values {
			value, err := evaluateManifestValue(item, env)
			if err != nil {
				return nil, err
			}
			number := manifestFloat(value)
			switch operator {
			case "$multiply":
				result *= manifestFloat(value)
			case "$min":
				result = math.Min(result, number)
			case "$max":
				result = math.Max(result, number)
			default:
				result += manifestFloat(value)
			}
		}
		return normalizeManifestNumber(result), nil
	case "$ceilStep":
		values, ok := operand.([]any)
		if !ok || len(values) != 2 {
			return nil, fmt.Errorf("$ceilStep requires [value, step]")
		}
		value, err := evaluateManifestValue(values[0], env)
		if err != nil {
			return nil, err
		}
		step, err := evaluateManifestValue(values[1], env)
		if err != nil {
			return nil, err
		}
		stepValue := manifestFloat(step)
		if stepValue <= 0 {
			return nil, fmt.Errorf("$ceilStep step must be positive")
		}
		return normalizeManifestNumber(math.Ceil(manifestFloat(value)/stepValue) * stepValue), nil
	case "$eq", "$ne", "$gt", "$gte", "$lt", "$lte", "$in", "$and", "$or":
		return evaluateManifestComparison(operator, operand, env)
	case "$not":
		value, err := evaluateManifestValue(operand, env)
		return !manifestTruthy(value), err
	default:
		return nil, fmt.Errorf("unsupported manifest operator %q", operator)
	}
}

// evaluateManifestComparison 对 $and、$or 做真值组合。$eq 和 $ne 比较两边的 JSON 文本。大小比较用浮点数。$in 看左边的 JSON 是否出现在右边数组里。除逻辑运算外必须正好两个操作数。
// operator 为 $and 或 $or 时做真值组合；$eq、$ne 比较 JSON 文本；$gt、$gte、$lt、$lte 用浮点数；$in 看左边的 JSON 是否出现在右边数组里。
// operand 必须是数组，否则失败；除 $and、$or 外求值后必须正好两项。
// env 传给每一项的 evaluateManifestValue。
func evaluateManifestComparison(operator string, operand any, env map[string]any) (any, error) {
	values, ok := operand.([]any)
	if !ok {
		return nil, fmt.Errorf("%s requires an array", operator)
	}
	evaluated := make([]any, 0, len(values))
	for _, item := range values {
		value, err := evaluateManifestValue(item, env)
		if err != nil {
			return nil, err
		}
		evaluated = append(evaluated, value)
	}
	if operator == "$and" || operator == "$or" {
		result := operator == "$and"
		for _, value := range evaluated {
			if operator == "$and" {
				result = result && manifestTruthy(value)
			} else {
				result = result || manifestTruthy(value)
			}
		}
		return result, nil
	}
	if len(evaluated) != 2 {
		return nil, fmt.Errorf("%s requires two operands", operator)
	}
	left, right := evaluated[0], evaluated[1]
	switch operator {
	case "$eq", "$ne":
		leftJSON, _ := json.Marshal(left)
		rightJSON, _ := json.Marshal(right)
		equal := string(leftJSON) == string(rightJSON)
		if operator == "$ne" {
			return !equal, nil
		}
		return equal, nil
	case "$gt":
		return manifestFloat(left) > manifestFloat(right), nil
	case "$gte":
		return manifestFloat(left) >= manifestFloat(right), nil
	case "$lt":
		return manifestFloat(left) < manifestFloat(right), nil
	case "$lte":
		return manifestFloat(left) <= manifestFloat(right), nil
	case "$in":
		for _, item := range manifestArray(right) {
			leftJSON, _ := json.Marshal(left)
			itemJSON, _ := json.Marshal(item)
			if string(leftJSON) == string(itemJSON) {
				return true, nil
			}
		}
		return false, nil
	}
	return false, nil
}

// interpolateManifestString 把 {{路径}} 换成环境里该路径的文本。没有成对的 }} 时留下原文。
// value 里成对的 {{路径}} 换成文本，没有成对的 }} 时留下原文。
// env 传给 manifestPathValue。
func interpolateManifestString(value string, env map[string]any) string {
	result := value
	for {
		start := strings.Index(result, "{{")
		if start < 0 {
			return result
		}
		endOffset := strings.Index(result[start+2:], "}}")
		if endOffset < 0 {
			return result
		}
		end := start + 2 + endOffset
		path := strings.TrimSpace(result[start+2 : end])
		result = result[:start] + manifestString(manifestPathValue(env, path)) + result[end+2:]
	}
}

// manifestPathValue 按点号路径取值。空路径返回根。数组下标不是非负整数或越界时返回 nil。
// root 在 path 去掉空白后为空时原样返回，否则按点号逐段往下取。
// path 去掉首尾点号后按点号分段；数组下标小于 0 或越界时返回 nil。
func manifestPathValue(root any, path string) any {
	if strings.TrimSpace(path) == "" {
		return root
	}
	value := root
	for _, part := range strings.Split(strings.Trim(path, "."), ".") {
		switch current := value.(type) {
		case map[string]any:
			value = current[part]
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(current) {
				return nil
			}
			value = current[index]
		default:
			return nil
		}
	}
	return value
}

// cloneManifestEnv 浅拷贝求值环境。
// env 的键值浅拷贝进新 map，容量是原长度加 2。
func cloneManifestEnv(env map[string]any) map[string]any {
	result := make(map[string]any, len(env)+2)
	for key, value := range env {
		result[key] = value
	}
	return result
}

// manifestArray 在值已是数组时原样返回，nil 返回 nil，其他值包成单元素数组。
// value 为 nil 时返回 nil，已是数组时原样返回，其他值包成单元素数组。
func manifestArray(value any) []any {
	if value == nil {
		return nil
	}
	if items, ok := value.([]any); ok {
		return items
	}
	return []any{value}
}

// manifestObject 把值当成对象。不是对象时返回 nil。
// value 做 map[string]any 断言，失败返回 nil。
func manifestObject(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

// manifestEmpty 把 nil、空白字符串、空数组和空对象当成空。0 和 false 不算空。
// value 为 nil、去掉空白后的空串、空数组或空对象时返回真，其余返回假。
func manifestEmpty(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	default:
		return false
	}
}

// manifestTruthy 把 nil、假、0、空集合，以及 false、0、no、null 这些文本当成假。其他值当成真。
// value 为 nil、false、0、空串、空数组、空对象，或文本 false、0、no、null 时返回假，其余返回真。
func manifestTruthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		normalized := strings.ToLower(strings.TrimSpace(typed))
		return normalized != "" && normalized != "false" && normalized != "0" && normalized != "no" && normalized != "null"
	case float64:
		return typed != 0
	case float32:
		return typed != 0
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}

// manifestString 把 nil 收成空串，数字和布尔收成文本，其余值收成 JSON。
// value 为 nil 时返回空串，字符串原样返回，数字和布尔收成文本，其余 Marshal 成 JSON。
func manifestString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		return strconv.FormatBool(typed)
	default:
		encoded, _ := json.Marshal(value)
		return string(encoded)
	}
}

// manifestFloat 调用 manifestFloatValue，解析失败时返回 0。
// value 传给 manifestFloatValue，解析失败时返回 0。
func manifestFloat(value any) float64 {
	parsed, _ := manifestFloatValue(value)
	return parsed
}

// manifestFloatValue 接受浮点、整数、json.Number 和去掉空白后的数字字符串。其他类型返回 0 和假。
// value 为 float64、float32、int、int64 时转成 float64；json.Number 和去掉空白后的数字字符串用解析结果；其他类型返回 0 和假。
func manifestFloatValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		result, err := typed.Float64()
		return result, err == nil
	case string:
		result, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return result, err == nil
	default:
		return 0, false
	}
}

// manifestInt 把 manifestFloat 的结果截成 int。
// value 传给 manifestFloat，结果截成 int。
func manifestInt(value any) int {
	return int(manifestFloat(value))
}

// normalizeManifestNumber 在数值没有小数部分时返回 int，否则返回 float64。
// value 与 Trunc 相等时返回 int，否则返回自身。
func normalizeManifestNumber(value float64) any {
	if math.Trunc(value) == value {
		return int(value)
	}
	return value
}
