// Config 的 JSON 反序列化：缺省键保留原值，并接受另一种 locale 写法。

package app

import "encoding/json"

// UnmarshalJSON 在现有 Config 上合并 JSON，缺省键保留。locale 仍为空时再接受 Locale 或 language。
// b 传给 json.Unmarshal，先合并进 Config，Locale 仍为空时再按 Locale 和 language 解一次。
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
