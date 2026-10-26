package desktop

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/api"
)

// remoteMap 在 Service 或 App 都不可用且没有 RPC 时返回 remote unavailable。RPC 非空时 call 再解码成 map。同进程用 Dispatch 调同一方法，再把 Result 解回 map。
func (s *Service) remoteMap(method string, params any) (map[string]any, error) {
	if s == nil {
		return nil, fmt.Errorf("remote unavailable")
	}
	if s.RPC != nil {
		return decode[map[string]any](s.call(method, params))
	}
	if s.App == nil {
		return nil, fmt.Errorf("remote unavailable")
	}
	raw := []byte("{}")
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	res := api.Dispatch(context.Background(), s.App, api.RPCRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: raw})
	if res.Error != nil {
		return nil, fmt.Errorf("%s", res.Error.Message)
	}
	b, err := json.Marshal(res.Result)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RemoteStatus 调用 remote.status，返回监听、待配对和已配对设备。
func (s *Service) RemoteStatus() (map[string]any, error) {
	return s.remoteMap("remote.status", nil)
}

// RemotePair 调用 remote.pair.begin。配对会在需要时打开监听，并返回两分钟有效的二维码。
func (s *Service) RemotePair() (map[string]any, error) {
	return s.remoteMap("remote.pair.begin", nil)
}

// RemoteApprove 调用 remote.pair.approve，让扫过配对码的手机加入这台工作站。
func (s *Service) RemoteApprove(id string) (map[string]any, error) {
	return s.remoteMap("remote.pair.approve", map[string]any{"id": id})
}

// RemoteDeny 调用 remote.pair.deny，拒绝还在等配对码的手机。
func (s *Service) RemoteDeny(id string) (map[string]any, error) {
	return s.remoteMap("remote.pair.deny", map[string]any{"id": id})
}

// RemoteRevoke 调用 remote.revoke，摘掉一台手机。它可以再用新的配对码加入。
func (s *Service) RemoteRevoke(id string) (map[string]any, error) {
	return s.remoteMap("remote.revoke", map[string]any{"id": id})
}

// RemoteDisable 调用 remote.disable，关掉手持端监听。
func (s *Service) RemoteDisable() (map[string]any, error) {
	return s.remoteMap("remote.disable", nil)
}
