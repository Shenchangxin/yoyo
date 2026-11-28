package connector

import (
	"net/http"
	"testing"
)

// TestProbeReadsLiveOnly 确认自检用真实令牌读：读到几封就报几条；
// 没有令牌时报失败，不像 Read 那样退回本地样例；找不到的账号也报失败。
func TestProbeReadsLiveOnly(t *testing.T) {
	b, _ := Open(t.TempDir())
	b.SetTokens(memTokens{})
	b.SetHTTP(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"value":[{"id":"m1","subject":"Hi","from":{"emailAddress":{"address":"a@b.com"}}},`+
			`{"id":"m2","subject":"Yo","from":{"emailAddress":{"address":"c@d.com"}}}]}`), nil
	})})
	live := b.Connect(Account{Provider: "outlook", Kind: KindMail, Label: "o"})
	_ = b.SaveToken(live.VaultKey, Token{AccessToken: "tok"})
	bare := b.Connect(Account{Provider: "gmail", Kind: KindMail, Label: "g"})

	if got := b.Probe(live.ID); !got.OK || got.Items != 2 || got.Error != "" {
		t.Fatalf("live probe = %+v", got)
	}
	if got := b.Probe(bare.ID); got.OK || got.Error == "" {
		t.Fatalf("no token must fail, not fall back to samples: %+v", got)
	}
	all := b.ProbeAll("")
	if len(all) != 2 || all[0].ID != live.ID || all[1].ID != bare.ID {
		t.Fatalf("probe all = %+v", all)
	}
	if got := b.ProbeAll("nope"); len(got) != 1 || got[0].OK {
		t.Fatalf("unknown id = %+v", got)
	}
}
