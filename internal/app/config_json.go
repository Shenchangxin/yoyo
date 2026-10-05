package app

import "encoding/json"

// UnmarshalJSON merges into the existing Config so omitted keys survive a
// partial Wails/HTTP body. It also accepts both `locale` and Wails' `Locale`.
func (c *Config) UnmarshalJSON(b []byte) error {
	type plain Config
	p := plain(*c)
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
