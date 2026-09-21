package observe

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

func exportOTLP(endpoint string, s Span) {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return
	}
	code := 1
	if s.Status == "error" {
		code = 2
	}
	attrs := []map[string]any{}
	for k, v := range s.Attrs {
		item := map[string]any{"key": k}
		switch t := v.(type) {
		case string:
			item["value"] = map[string]any{"stringValue": diaglog.Redact(t)}
		case int:
			item["value"] = map[string]any{"intValue": strconv.Itoa(t)}
		case int64:
			item["value"] = map[string]any{"intValue": strconv.FormatInt(t, 10)}
		case float64:
			item["value"] = map[string]any{"intValue": strconv.FormatInt(int64(t), 10)}
		case bool:
			item["value"] = map[string]any{"boolValue": t}
		default:
			item["value"] = map[string]any{"stringValue": diaglog.Redact(stringify(v))}
		}
		attrs = append(attrs, item)
	}
	if s.SessionID != "" {
		attrs = append(attrs, map[string]any{"key": "session_id", "value": map[string]any{"stringValue": s.SessionID}})
	}
	if s.TurnID != "" {
		attrs = append(attrs, map[string]any{"key": "turn_id", "value": map[string]any{"stringValue": s.TurnID}})
	}
	end := s.End
	if end.IsZero() {
		end = time.Now().UTC()
	}
	body := map[string]any{
		"resourceSpans": []any{
			map[string]any{
				"resource": map[string]any{
					"attributes": []any{
						map[string]any{"key": "service.name", "value": map[string]any{"stringValue": "yoyo"}},
					},
				},
				"scopeSpans": []any{
					map[string]any{
						"spans": []any{
							map[string]any{
								"traceId":           s.TraceID,
								"spanId":            s.SpanID,
								"parentSpanId":      s.ParentSpanID,
								"name":              s.Name,
								"startTimeUnixNano": strconv.FormatInt(s.Start.UnixNano(), 10),
								"endTimeUnixNano":   strconv.FormatInt(end.UnixNano(), 10),
								"status":            map[string]any{"code": code},
								"attributes":        attrs,
							},
						},
					},
				},
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return
	}
	raw = diaglog.RedactBytes(raw)
	req, err := http.NewRequest(http.MethodPost, endpoint+"/v1/traces", bytes.NewReader(raw))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	cli := &http.Client{Timeout: 5 * time.Second}
	res, err := cli.Do(req)
	if err != nil {
		return
	}
	_ = res.Body.Close()
}

func stringify(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
