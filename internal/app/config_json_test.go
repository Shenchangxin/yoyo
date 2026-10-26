// 配置 JSON 对 locale 键和缺省字段的合并。

package app

import "testing"

// TestConfigUnmarshalLocaleKeys 接受小写 locale，也接受 Pascal 的 Locale。
func TestConfigUnmarshalLocaleKeys(t *testing.T) {
	var a Config
	if err := a.UnmarshalJSON([]byte(`{"locale":"zh-CN","model":"x"}`)); err != nil {
		t.Fatal(err)
	}
	if a.Locale != "zh-CN" || a.Model != "x" {
		t.Fatalf("tagged locale: %+v", a)
	}
	var b Config
	if err := b.UnmarshalJSON([]byte(`{"Locale":"zh-CN","Model":"y"}`)); err != nil {
		t.Fatal(err)
	}
	if b.Locale != "zh-CN" {
		t.Fatalf("pascal locale: %+v", b)
	}
}

// TestConfigUnmarshalKeepsOtherFields 确认 provider、base_url 和 locale 一起被读入。
func TestConfigUnmarshalKeepsOtherFields(t *testing.T) {
	var c Config
	if err := c.UnmarshalJSON([]byte(`{"provider":"deepseek","base_url":"https://api.deepseek.com","locale":"zh-CN"}`)); err != nil {
		t.Fatal(err)
	}
	if c.Provider != "deepseek" || c.BaseURL != "https://api.deepseek.com" || c.Locale != "zh-CN" {
		t.Fatalf("got %+v", c)
	}
}

// TestConfigUnmarshalMergesOmitted 确认补丁没写到的 SearchURL 和 CrashResume 还在。
func TestConfigUnmarshalMergesOmitted(t *testing.T) {
	c := Config{SearchURL: "https://keep.example/{q}", CrashResume: true, Model: "old"}
	if err := c.UnmarshalJSON([]byte(`{"model":"new","theme":"dark"}`)); err != nil {
		t.Fatal(err)
	}
	if c.Model != "new" || c.Theme != "dark" {
		t.Fatalf("patched %+v", c)
	}
	if c.SearchURL != "https://keep.example/{q}" || !c.CrashResume {
		t.Fatalf("omitted wiped %+v", c)
	}
}
