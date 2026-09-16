package app

import "testing"

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

func TestConfigUnmarshalKeepsOtherFields(t *testing.T) {
	var c Config
	if err := c.UnmarshalJSON([]byte(`{"provider":"deepseek","base_url":"https://api.deepseek.com","locale":"zh-CN"}`)); err != nil {
		t.Fatal(err)
	}
	if c.Provider != "deepseek" || c.BaseURL != "https://api.deepseek.com" || c.Locale != "zh-CN" {
		t.Fatalf("got %+v", c)
	}
}
