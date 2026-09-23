package video

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

func (e *Engine) ListProviders(serviceType string) ([]Provider, error) {
	q := `SELECT id, service_type, provider, name, base_url, vault_key, model, models, priority, is_default, is_active, settings, created_at, updated_at FROM providers`
	var args []any
	if serviceType != "" {
		q += ` WHERE service_type = ?`
		args = append(args, serviceType)
	}
	q += ` ORDER BY priority DESC, created_at ASC`
	rows, err := e.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Provider
	for rows.Next() {
		p, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		if e.Vault != nil {
			if _, err := e.Vault.Get(p.VaultKey); err == nil {
				p.HasKey = true
			}
		}
		out = append(out, p)
	}
	if out == nil {
		out = []Provider{}
	}
	return out, rows.Err()
}

func scanProvider(rows *sql.Rows) (Provider, error) {
	var p Provider
	var def, active int
	err := rows.Scan(&p.ID, &p.ServiceType, &p.Provider, &p.Name, &p.BaseURL, &p.VaultKey, &p.Model, &p.Models, &p.Priority, &def, &active, &p.Settings, &p.CreatedAt, &p.UpdatedAt)
	p.IsDefault = def != 0
	p.IsActive = active != 0
	return p, err
}

func (e *Engine) GetProvider(id string) (Provider, error) {
	row := e.DB.QueryRow(`SELECT id, service_type, provider, name, base_url, vault_key, model, models, priority, is_default, is_active, settings, created_at, updated_at FROM providers WHERE id = ?`, id)
	var p Provider
	var def, active int
	err := row.Scan(&p.ID, &p.ServiceType, &p.Provider, &p.Name, &p.BaseURL, &p.VaultKey, &p.Model, &p.Models, &p.Priority, &def, &active, &p.Settings, &p.CreatedAt, &p.UpdatedAt)
	p.IsDefault = def != 0
	p.IsActive = active != 0
	if err != nil {
		return p, fmt.Errorf("provider not found")
	}
	if e.Vault != nil {
		if _, err := e.Vault.Get(p.VaultKey); err == nil {
			p.HasKey = true
		}
	}
	return p, nil
}

func (e *Engine) ActiveProvider(serviceType, id string) (Provider, error) {
	if id != "" {
		if p, err := e.GetProvider(id); err == nil {
			return p, nil
		}
	}
	list, err := e.ListProviders(serviceType)
	if err != nil {
		return Provider{}, err
	}
	var keyed, anyActive Provider
	var hasKeyed, hasAny bool
	for _, p := range list {
		if !p.IsActive {
			continue
		}
		if !hasAny {
			anyActive = p
			hasAny = true
		}
		if p.HasKey {
			if p.IsDefault {
				return p, nil
			}
			if !hasKeyed {
				keyed = p
				hasKeyed = true
			}
		}
	}
	if hasKeyed {
		return keyed, nil
	}
	if hasAny {
		return anyActive, nil
	}
	return Provider{}, fmt.Errorf("no active %s provider — add one in Settings", serviceType)
}

func ProviderModels(p Provider) []string {
	var out []string
	seen := map[string]bool{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			return
		}
		seen[v] = true
		out = append(out, v)
	}
	add(p.Model)
	raw := strings.TrimSpace(p.Models)
	if strings.HasPrefix(raw, "[") {
		var arr []string
		if json.Unmarshal([]byte(raw), &arr) == nil {
			for _, m := range arr {
				add(m)
			}
		}
	} else if raw != "" {
		for _, m := range strings.Split(raw, ",") {
			add(m)
		}
	}
	return out
}

func ResolveProviderModel(p Provider, want string) string {
	want = strings.TrimSpace(want)
	if want != "" {
		return want
	}
	if p.Model != "" {
		return p.Model
	}
	if ms := ProviderModels(p); len(ms) > 0 {
		return ms[0]
	}
	return ""
}

func (e *Engine) uniqueProviderName(serviceType, name, skipID string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		return base
	}
	candidate := base
	for n := 2; n < 100; n++ {
		var id string
		err := e.DB.QueryRow(`SELECT id FROM providers WHERE service_type = ? AND name = ? AND id != ? LIMIT 1`, serviceType, candidate, skipID).Scan(&id)
		if err != nil {
			return candidate
		}
		candidate = fmt.Sprintf("%s %d", base, n)
	}
	return candidate
}

func (e *Engine) UpsertProvider(p Provider, apiKey string) (Provider, error) {
	now := Now()
	if p.ID != "" {
		if cur, err := e.GetProvider(p.ID); err == nil {
			if p.VaultKey == "" {
				p.VaultKey = cur.VaultKey
			}
			if p.CreatedAt == "" {
				p.CreatedAt = cur.CreatedAt
			}
		}
	}
	if p.ID == "" {
		p.ID = NewID()
		p.CreatedAt = now
		p.Name = e.uniqueProviderName(p.ServiceType, p.Name, p.ID)
	}
	p.UpdatedAt = now
	if p.VaultKey == "" {
		p.VaultKey = "video." + p.ServiceType + "." + p.ID
	}
	if p.Settings == "" {
		p.Settings = "{}"
	}
	if p.Models == "" {
		p.Models = "[]"
	}
	def, active := 0, 0
	if p.IsDefault {
		def = 1
	}
	if p.IsActive {
		active = 1
	}
	if p.IsDefault {
		_, _ = e.DB.Exec(`UPDATE providers SET is_default = 0 WHERE service_type = ? AND id != ?`, p.ServiceType, p.ID)
	}
	_, err := e.DB.Exec(`INSERT INTO providers(id, service_type, provider, name, base_url, vault_key, model, models, priority, is_default, is_active, settings, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET service_type=excluded.service_type, provider=excluded.provider, name=excluded.name, base_url=excluded.base_url, vault_key=excluded.vault_key, model=excluded.model, models=excluded.models, priority=excluded.priority, is_default=excluded.is_default, is_active=excluded.is_active, settings=excluded.settings, updated_at=excluded.updated_at`,
		p.ID, p.ServiceType, p.Provider, p.Name, p.BaseURL, p.VaultKey, p.Model, p.Models, p.Priority, def, active, p.Settings, coalesce(p.CreatedAt, now), p.UpdatedAt)
	if err != nil {
		return p, err
	}
	if apiKey != "" && e.Vault != nil {
		e.Vault.Set(p.VaultKey, apiKey)
		p.HasKey = true
	}
	return e.GetProvider(p.ID)
}

func (e *Engine) DeleteProvider(id string) error {
	_, err := e.DB.Exec(`DELETE FROM providers WHERE id = ?`, id)
	return err
}

func (e *Engine) TestProvider(id string) map[string]any {
	p, err := e.GetProvider(id)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	key, err := e.leaseProvider(p)
	if err != nil || key == "" {
		return map[string]any{"ok": false, "error": "missing vault key"}
	}
	url := strings.TrimRight(p.BaseURL, "/")
	req, err := httpGet(url, key)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	res, err := e.HTTP.Do(req)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	defer res.Body.Close()
	ok := res.StatusCode == 200 || res.StatusCode == 400 || res.StatusCode == 401 || res.StatusCode == 403 || res.StatusCode == 404
	return map[string]any{"ok": ok, "status": res.StatusCode, "reachable": ok}
}

func coalesce(v, fb string) string {
	if v == "" {
		return fb
	}
	return v
}
