package skin

import (
	"net/http"
	"strings"
)

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
