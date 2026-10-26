package api

import (
	"context"
	"encoding/json"

	"github.com/Shenchangxin/yoyo/internal/app"
)

// RemoteHandler 处理手持网关不肯代签的操作方法，例如配对。方法不属于远程时 ok 为假。
type RemoteHandler func(ctx context.Context, a *app.App, method string, params json.RawMessage) (out any, ok bool, err error)

var remoteHandler RemoteHandler

// SetRemoteHandler 登记本进程的手持节点。callMethod 看到 remote. 前缀时先问这个处理函数。
func SetRemoteHandler(h RemoteHandler) {
	remoteHandler = h
}
