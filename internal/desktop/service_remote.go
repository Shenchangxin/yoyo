package desktop

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/api"
)

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

// RemoteStatus reports the handheld listener, pending phones, and paired devices.
func (s *Service) RemoteStatus() (map[string]any, error) {
	return s.remoteMap("remote.status", nil)
}

// RemotePair opens the listener and returns a two-minute pairing QR.
func (s *Service) RemotePair() (map[string]any, error) {
	return s.remoteMap("remote.pair.begin", nil)
}

// RemoteApprove lets a phone that scanned the code join this workstation.
func (s *Service) RemoteApprove(id string) (map[string]any, error) {
	return s.remoteMap("remote.pair.approve", map[string]any{"id": id})
}

// RemoteDeny rejects a phone waiting on the pairing code.
func (s *Service) RemoteDeny(id string) (map[string]any, error) {
	return s.remoteMap("remote.pair.deny", map[string]any{"id": id})
}

// RemoteRevoke removes a phone. It can pair again with a new code.
func (s *Service) RemoteRevoke(id string) (map[string]any, error) {
	return s.remoteMap("remote.revoke", map[string]any{"id": id})
}

// RemoteDisable closes the handheld listener.
func (s *Service) RemoteDisable() (map[string]any, error) {
	return s.remoteMap("remote.disable", nil)
}
