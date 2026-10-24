package remote

import (
	"encoding/json"
	"strings"
)

var secretFields = map[string]struct{}{
	"api_key": {}, "apikey": {}, "token": {}, "secret": {}, "authorization": {},
	"search_key": {}, "password": {}, "refresh_token": {}, "access_token": {},
	"private_key": {}, "api_key_value": {},
}

// Scrub removes vault material from a value that is about to leave the machine.
// has_key stays, so the phone can show whether a provider is ready.
func Scrub(v any) any {
	raw, err := json.Marshal(v)
	if err != nil || len(raw) == 0 {
		return v
	}
	var decoded any
	if json.Unmarshal(raw, &decoded) != nil {
		return v
	}
	return scrub(decoded)
}

func scrub(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if _, drop := secretFields[strings.ToLower(k)]; drop {
				continue
			}
			out[k] = scrub(val)
		}
		return out
	case []any:
		for i := range t {
			t[i] = scrub(t[i])
		}
		return t
	default:
		return v
	}
}
