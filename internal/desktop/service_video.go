package desktop

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/api"
)

// VideoCall 把方法名原样交给后端。params 为 nil 时换成空 map。RPC 非空时 call 这个方法。没有 App 时返回 video: no app。否则用 Dispatch 调同一方法，并返回 Result。
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
