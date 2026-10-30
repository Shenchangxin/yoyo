package im

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// runOpGateway 跑 Discord 和 QQ 这一类 op/t/d 网关：先等 hello，再 identify，并按间隔心跳。
func (rt *Runtime) runOpGateway(ctx context.Context, acct Account, rawURL string, header http.Header, identify func() (any, error), onDispatch func(t string, d map[string]any)) error {
	c, err := dialWS(ctx, rawURL, header)
	if err != nil {
		return err
	}
	defer closeWS(c)
	sock := rt.putSocket(acct.ID, c)
	defer func() {
		if cur := rt.socket(acct.ID); cur != nil && cur.c == c {
			rt.sockets.Delete(acct.ID)
		}
	}()
	var seq atomic.Int64
	stopHB := func() {}
	defer func() { stopHB() }()
	started := false
	for {
		raw, err := readWS(ctx, c)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return gatewayFatal(err)
		}
		var frame struct {
			Op int             `json:"op"`
			T  string          `json:"t"`
			S  *int64          `json:"s"`
			D  json.RawMessage `json:"d"`
		}
		if json.Unmarshal(raw, &frame) != nil {
			continue
		}
		if frame.S != nil {
			seq.Store(*frame.S)
		}
		d := map[string]any{}
		if len(bytesTrim(frame.D)) > 0 && frame.D[0] == '{' {
			d, _ = decodeJSON(frame.D)
		}
		switch frame.Op {
		case 10:
			if started {
				continue
			}
			started = true
			body, err := identify()
			if err != nil {
				return err
			}
			if err := sock.write(ctx, body); err != nil {
				return err
			}
			ms := jnum(d["heartbeat_interval"])
			if ms <= 0 {
				ms = 40000
			}
			hbCtx, cancel := context.WithCancel(ctx)
			stopHB = cancel
			local := sock
			go pumpJSON(hbCtx, local, time.Duration(ms)*time.Millisecond, func() any {
				var seqVal any
				if n := seq.Load(); n > 0 {
					seqVal = n
				}
				return map[string]any{"op": 1, "d": seqVal}
			})
		case 0:
			if onDispatch != nil {
				onDispatch(frame.T, d)
			}
		case 7, 9:
			return fmt.Errorf("gateway asked to reconnect")
		}
	}
}

func bytesTrim(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\n' || b[0] == '\t') {
		b = b[1:]
	}
	return b
}
