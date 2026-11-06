package remake

// Status of one upstream package in this tree.
const (
	StatusPending    = "pending"
	StatusPartial    = "partial"
	StatusTranslated = "translated"
)

// Package is one directory under hypit/packages at UpstreamCommit.
type Package struct {
	Name   string
	Layer  string
	Gate   int
	Status string
	// Note records a remaining parity gap. Empty when there is nothing open.
	Note string
}

// Catalog is the full distribution. Gates follow the translation order:
// 1 language and kernel, 2 execution, 3 time and picture, 4 models and local
// tools, 5 product shell, 6 optional components.
func Catalog() []Package {
	out := make([]Package, len(catalog))
	copy(out, catalog)
	return out
}

var catalog = []Package{
	{Name: "admission", Layer: "kernel-language", Gate: 1, Status: StatusTranslated},
	{Name: "atomic-file", Layer: "kernel-language", Gate: 1, Status: StatusTranslated},
	{Name: "author", Layer: "kernel-language", Gate: 1, Status: StatusPartial, Note: "author and fragment tests pass; bindExports walks binding names by localeCompare because Go maps have no insertion order; the fragment fixture does not link the full video-contract closure"},
	{Name: "blob", Layer: "kernel-language", Gate: 1, Status: StatusTranslated},
	{Name: "compiler", Layer: "kernel-language", Gate: 1, Status: StatusPending},
	{Name: "facet", Layer: "kernel-language", Gate: 1, Status: StatusTranslated},
	{Name: "kernel", Layer: "kernel-language", Gate: 1, Status: StatusPartial, Note: "link, graph, plan, reducer, machine, and slice are translated; upstream demand.test.ts is not ported yet"},
	{Name: "loader", Layer: "kernel-language", Gate: 1, Status: StatusPending},
	{Name: "markup", Layer: "kernel-language", Gate: 1, Status: StatusPending},
	{Name: "project", Layer: "kernel-language", Gate: 1, Status: StatusTranslated},
	{Name: "protocol", Layer: "kernel-language", Gate: 1, Status: StatusPartial, Note: "upstream canonical tests pass; Han-versus-Latin key order still differs from Node ICU localeCompare"},
	{Name: "result", Layer: "kernel-language", Gate: 1, Status: StatusPartial, Note: "writer, log, and repository pass the upstream tests; nested resource numbers follow sorted keys; output tie-breaks and browse use byte order instead of localeCompare; encoded JSON key order differs from JSON.stringify; file URLs follow net/url, so localhost and dot segments differ from Node fileURLToPath"},
	{Name: "run", Layer: "kernel-language", Gate: 1, Status: StatusPending},
	{Name: "source", Layer: "kernel-language", Gate: 1, Status: StatusPartial, Note: "header, mask, and UTF-8 decode match the upstream text tests; a UTF-16 split in MaskSourceHeader becomes U+FFFD, and resolvers cannot return a Promise"},
	{Name: "workspace", Layer: "kernel-language", Gate: 1, Status: StatusPartial, Note: "filesystem session matches the upstream workspace tests; host resolvers return their result directly and cannot return a Promise"},

	{Name: "cli", Layer: "execution", Gate: 2, Status: StatusPending},
	{Name: "credential-store-env", Layer: "execution", Gate: 2, Status: StatusPartial, Note: "resolve and activation match; CredentialRef and config helpers stay local until runtime-local exists; several rejected config keys follow Go map iteration"},
	{Name: "credential-store-local", Layer: "execution", Gate: 2, Status: StatusPending},
	{Name: "endpoint", Layer: "execution", Gate: 2, Status: StatusPending},
	{Name: "executor", Layer: "execution", Gate: 2, Status: StatusPending},
	{Name: "producer", Layer: "execution", Gate: 2, Status: StatusPartial, Note: "handlers and package facets match; pending-input order follows localeCompare because Go maps do not keep insertion order"},
	{Name: "runtime", Layer: "execution", Gate: 2, Status: StatusPartial, Note: "scheduler, credentials, capacity, and the execution-log tail match the translated branches; log input is a chunk slice, and composite resolve calls stores in order"},
	{Name: "runtime-local", Layer: "execution", Gate: 2, Status: StatusPending},

	{Name: "audio-track", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "caption", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "caption-fine", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "composition", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "depth-stack", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "film", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "fontsource", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "generation", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "html-document", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "html-program", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "html-video", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "image-operations", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "media", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "media-operations", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "narrative", Layer: "time-picture", Gate: 3, Status: StatusPartial, Note: "identity, selection, schemas, and activation are translated; admission still expects Go structs rather than canonical maps"},
	{Name: "narrative-caption", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "narrative-speech-alignment", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "narrative-temporal", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "recipe", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "region-evidence", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "script", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "spatial", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "speech-evidence", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "temporal", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "text", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "text-fine", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "timeline", Layer: "time-picture", Gate: 3, Status: StatusTranslated},
	{Name: "timeline-author", Layer: "time-picture", Gate: 3, Status: StatusPending},
	{Name: "visual-track", Layer: "time-picture", Gate: 3, Status: StatusPending},

	{Name: "elevenlabs-speech", Layer: "model", Gate: 4, Status: StatusPending},
	{Name: "fishaudio-speech", Layer: "model", Gate: 4, Status: StatusPending},
	{Name: "gpt-image", Layer: "model", Gate: 4, Status: StatusPending},
	{Name: "gpt-image-kits", Layer: "model", Gate: 4, Status: StatusPending},
	{Name: "grok-imagine", Layer: "model", Gate: 4, Status: StatusPending},
	{Name: "mimo-speech", Layer: "model", Gate: 4, Status: StatusPending},
	{Name: "minimax-h3", Layer: "model", Gate: 4, Status: StatusPending},
	{Name: "nano-banana", Layer: "model", Gate: 4, Status: StatusPending},
	{Name: "pixverse", Layer: "model", Gate: 4, Status: StatusPending},
	{Name: "seedance", Layer: "model", Gate: 4, Status: StatusPending},
	{Name: "seedance-kits", Layer: "model", Gate: 4, Status: StatusPending},
	{Name: "seedream", Layer: "model", Gate: 4, Status: StatusPending},
	{Name: "volcengine-matting", Layer: "model", Gate: 4, Status: StatusPending},
	{Name: "wan", Layer: "model", Gate: 4, Status: StatusPending},

	{Name: "browser-capture", Layer: "local-tool", Gate: 4, Status: StatusPending},
	{Name: "media-local", Layer: "local-tool", Gate: 4, Status: StatusPending},
	{Name: "provider-beatapi", Layer: "local-tool", Gate: 4, Status: StatusPending},
	{Name: "provider-hiapi", Layer: "local-tool", Gate: 4, Status: StatusPending},
	{Name: "provider-html-local", Layer: "local-tool", Gate: 4, Status: StatusPending},
	{Name: "provider-hypihub", Layer: "local-tool", Gate: 4, Status: StatusPending},
	{Name: "provider-image-opencv-local", Layer: "local-tool", Gate: 4, Status: StatusPending},
	{Name: "provider-monid", Layer: "local-tool", Gate: 4, Status: StatusPending},
	{Name: "provider-pollo", Layer: "local-tool", Gate: 4, Status: StatusPending},
	{Name: "provider-tokendance", Layer: "local-tool", Gate: 4, Status: StatusPending},
	{Name: "provider-whisperx-local", Layer: "local-tool", Gate: 4, Status: StatusPending},
	{Name: "speech-estimate", Layer: "local-tool", Gate: 4, Status: StatusPending},
	{Name: "whisperx", Layer: "local-tool", Gate: 4, Status: StatusPending},
	{Name: "yt-dlp", Layer: "local-tool", Gate: 4, Status: StatusPending},

	{Name: "studio", Layer: "shell", Gate: 5, Status: StatusPending},
	{Name: "studio-companion", Layer: "shell", Gate: 5, Status: StatusPartial, Note: "identity tests pass; attribute and value maps have no insertion order, and omitted arrays do not distinguish null from undefined"},
	{Name: "video", Layer: "shell", Gate: 5, Status: StatusPending},

	{Name: "comment-sticker", Layer: "optional", Gate: 6, Status: StatusPending},
	{Name: "interview-emoji-reveal", Layer: "optional", Gate: 6, Status: StatusPending},
	{Name: "ranking", Layer: "optional", Gate: 6, Status: StatusPending},
}
