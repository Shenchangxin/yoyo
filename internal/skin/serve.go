// 按 /yoyo-skin/<id>/<路径> 提供皮肤文件。只接受 GET 和 HEAD，Cache-Control 为 no-store。

package skin

import (
	"net/http"
	"strings"
)

// Middleware 在路径以 AssetPrefix 开头时交给 Handler，否则交给 next。
// store 传给 Handler。
func Middleware(store *Store, next http.Handler) http.Handler {
	assets := Handler(store)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, AssetPrefix) {
			assets.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Handler 只接受 GET 和 HEAD。路径须是 <id>/<相对路径>。File 失败则 404。响应带上类型、no-store 和 nosniff。
func Handler(store *Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rel := strings.TrimPrefix(r.URL.Path, AssetPrefix)
		rel = strings.TrimPrefix(rel, "/")
		id, rest, ok := strings.Cut(rel, "/")
		if !ok || id == "" || rest == "" {
			http.NotFound(w, r)
			return
		}
		if store == nil {
			http.NotFound(w, r)
			return
		}
		ctype, body, err := store.File(id, rest)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ctype)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write(body)
	})
}
