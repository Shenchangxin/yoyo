package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/app"
)

// mountVideo 把 /api/video/ 转到 a.VideoCall。GET 只用查询参数；其他方法先读正文，上限 32MiB，查询参数只填正文里还没有的键。videoHTTPMethod 返回空则 404。
// mux 用 HandleFunc 挂上 /api/video/。
func mountVideo(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("/api/video/", func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/video/")
		params := map[string]any{}
		if r.Method != http.MethodGet && r.Body != nil {
			raw, _ := io.ReadAll(io.LimitReader(r.Body, 32<<20))
			if len(raw) > 0 {
				_ = json.Unmarshal(raw, &params)
			}
		}
		for k, v := range r.URL.Query() {
			if len(v) > 0 && params[k] == nil {
				params[k] = v[0]
			}
		}
		method := videoHTTPMethod(rest, r.Method)
		if method == "" {
			http.NotFound(w, r)
			return
		}
		out, err := a.VideoCall(method, params)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, out)
	})
}

// videoHTTPMethod 把 /api/video/ 后面的路径和 HTTP 方法映射成 video.、drama. 或 connections. 方法。providers、settings、dramas、episodes、connections、connections/defaults 在 POST 时走写入方法，其他方法走读取。未知路径返回空字符串。
// rest 去掉首尾斜杠后按路径选择返回值。对不上时返回空字符串。
// httpMethod 只在 providers、settings、dramas、episodes、connections、connections/defaults 上与 http.MethodPost 比较。
func videoHTTPMethod(rest, httpMethod string) string {
	rest = strings.Trim(rest, "/")
	switch rest {
	case "status":
		return "video.status"
	case "ffmpeg/install":
		return "video.ffmpeg.install"
	case "ytdlp/install":
		return "video.ytdlp.install"
	case "studio/remotion/status":
		return "video.studio.remotion.status"
	case "studio/remotion/install":
		return "video.studio.remotion.install"
	case "modes":
		return "video.modes"
	case "templates":
		return "video.templates"
	case "providers":
		if httpMethod == http.MethodPost {
			return "video.providers.upsert"
		}
		return "video.providers.list"
	case "providers/test":
		return "video.providers.test"
	case "providers/delete":
		return "video.providers.delete"
	case "styles":
		return "video.styles"
	case "styles/all":
		return "video.styles.all"
	case "styles/upsert":
		return "video.styles.upsert"
	case "styles/delete":
		return "video.styles.delete"
	case "settings":
		if httpMethod == http.MethodPost {
			return "video.settings.set"
		}
		return "video.settings.get"
	case "jobs":
		return "video.jobs.list"
	case "jobs/cancel":
		return "video.jobs.cancel"
	case "jobs/retry":
		return "video.jobs.retry"
	case "jobs/apply":
		return "video.jobs.apply"
	case "dramas":
		if httpMethod == http.MethodPost {
			return "drama.create"
		}
		return "drama.list"
	case "drama":
		return "drama.get"
	case "drama/update":
		return "drama.update"
	case "drama/delete":
		return "drama.delete"
	case "episodes":
		if httpMethod == http.MethodPost {
			return "drama.episode.create"
		}
		return "drama.episodes"
	case "episode":
		return "drama.episode.get"
	case "episode/update":
		return "drama.episode.update"
	case "episode/delete":
		return "drama.episode.delete"
	case "bundle":
		return "drama.bundle"
	case "bind":
		return "drama.bind"
	case "stage":
		return "drama.stage.run"
	case "assets/save":
		return "drama.assets.save"
	case "assets/create":
		return "drama.assets.create"
	case "assets/delete":
		return "drama.assets.delete"
	case "assets/generate":
		return "drama.assets.generate"
	case "assets/generate-missing":
		return "drama.assets.generate_missing"
	case "assets/upload":
		return "drama.assets.upload"
	case "shots/save":
		return "drama.shots.save"
	case "shots/update":
		return "drama.shots.update"
	case "shots/delete":
		return "drama.shots.delete"
	case "shots/generate":
		return "drama.shots.generate"
	case "shots/generate-missing":
		return "drama.shots.generate_missing"
	case "merge":
		return "drama.merge"
	case "import":
		return "drama.import"
	case "skip-rewrite":
		return "drama.skip_rewrite"
	case "connections":
		if httpMethod == http.MethodPost {
			return "connections.upsert"
		}
		return "connections.list"
	case "connections/test":
		return "connections.test"
	case "connections/delete":
		return "connections.delete"
	case "connections/defaults":
		if httpMethod == http.MethodPost {
			return "connections.setDefault"
		}
		return "connections.defaults"
	case "connections/templates":
		return "connections.templates"
	}
	return ""
}
