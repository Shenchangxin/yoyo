package api

import (
	"context"
	"encoding/json"

	"github.com/Shenchangxin/yoyo/internal/app"
)

// RemoteHandler serves operator methods that the handheld gateway also
// refuses to mint (pairing). ok is false when the method is not remote.
type RemoteHandler func(ctx context.Context, a *app.App, method string, params json.RawMessage) (out any, ok bool, err error)

var remoteHandler RemoteHandler

// SetRemoteHandler registers the process-local handheld node.
func SetRemoteHandler(h RemoteHandler) {
	remoteHandler = h
}
