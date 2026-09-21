package desktop

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/api"
)

func (s *Service) VideoCall(method string, params map[string]any) (any, error) {
	if params == nil {
		params = map[string]any{}
	}
	if s.RPC != nil {
		return s.call(method, params)
	}
	if s.App == nil {
		return nil, fmt.Errorf("video: no app")
	}
	raw, _ := json.Marshal(params)
	res := api.Dispatch(context.Background(), s.App, api.RPCRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: raw})
	if res.Error != nil {
		return nil, fmt.Errorf("%s", res.Error.Message)
	}
	return res.Result, nil
}
