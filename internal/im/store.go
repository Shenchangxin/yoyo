package im

import (
	"encoding/json"
	"os"
	"strings"
	"time"
)

type fileState struct {
	Settings Settings          `json:"settings"`
	Accounts []Account         `json:"accounts"`
	Bindings map[string]string `json:"bindings"`
}

func (rt *Runtime) load() error {
	b, err := os.ReadFile(rt.path)
	if err != nil {
		if os.IsNotExist(err) {
			rt.settings = DefaultSettings()
			rt.accounts = nil
			rt.bindings = map[string]string{}
			return nil
		}
		return err
	}
	var st fileState
	if err := json.Unmarshal(b, &st); err != nil {
		return err
	}
	if st.Settings.ListenPort == 0 && st.Settings.ListenHost == "" && len(st.Settings.WakePrefixes) == 0 {
		st.Settings = DefaultSettings()
	}
	rt.settings = st.Settings.normalized()
	rt.accounts = st.Accounts
	rt.bindings = st.Bindings
	if rt.bindings == nil {
		rt.bindings = map[string]string{}
	}
	for i := range rt.accounts {
		rt.accounts[i].HasSecret = nil
	}
	return nil
}

func (rt *Runtime) flush() error {
	st := fileState{Settings: rt.settings, Accounts: make([]Account, len(rt.accounts)), Bindings: rt.bindings}
	for i, a := range rt.accounts {
		a.HasSecret = nil
		st.Accounts[i] = a
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := rt.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, rt.path)
}

func applyDefaults(a Account) Account {
	spec, ok := specByID(a.Platform)
	if !ok {
		return a
	}
	if a.Config == nil {
		a.Config = map[string]string{}
	}
	for _, field := range spec.Fields {
		if field.Secret || field.Default == "" {
			continue
		}
		if strings.TrimSpace(a.Config[field.Key]) == "" {
			a.Config[field.Key] = field.Default
		}
	}
	return a
}

func cleanID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func newAccountID(platform string) string {
	return cleanID(platform) + "-" + time.Now().UTC().Format("150405")
}

func secretName(id, key string) string {
	return "im." + id + "." + key
}
