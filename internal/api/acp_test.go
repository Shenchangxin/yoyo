package api

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestACPInitialize(t *testing.T) {
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	var out bytes.Buffer
	if err := ServeACP(context.Background(), nil, in, &out); err != nil {
		t.Fatal(err)
	}
	var res RPCResponse
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Error != nil {
		t.Fatal(res.Error)
	}
}
