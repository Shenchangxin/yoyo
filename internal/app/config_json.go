package app

import "encoding/json"

// UnmarshalJSON accepts both `locale` and Wails' `Locale` key so native menus
// follow the UI language even when the bound payload uses Go field names.
func (c *Config) UnmarshalJSON(b []byte) error {
	type plain Config
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*c = Config(p)
	if c.Locale != "" {
		return nil
	}
	var alt struct {
		Locale string `json:"Locale"`
		Lang   string `json:"language"`
	}
	if err := json.Unmarshal(b, &alt); err != nil {
		return nil
	}
	if alt.Locale != "" {
		c.Locale = alt.Locale
	} else if alt.Lang != "" {
		c.Locale = alt.Lang
	}
	return nil
}
