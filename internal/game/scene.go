package game

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/game/ir"
)

func (e *Engine) AnalyzeScene(sessionID, id, intent string) (Project, error) {
	p, err := e.resolve(sessionID, id)
	if err != nil {
		return p, err
	}
	if intent != "" {
		p.Intent = intent
	}
	seed := hashSeed(p.ID + p.Intent)
	scene := generateScene(seed, p.Genre, p.Intent)
	if e.Complete != nil {
		scene = e.enrichSceneNotes(scene, p.Intent)
	}
	if err := e.writeIR(p, "scene.json", scene); err != nil {
		return p, err
	}
	if err := e.writeWorld(p, scene); err != nil {
		return p, err
	}
	p.Stage = StageScene
	p.Status = "open"
	p.Error = ""
	return e.Update(p)
}

func (e *Engine) writeWorld(p Project, scene ir.ScenePacket) error {
	dir, err := e.projectDir(p)
	if err != nil {
		return err
	}
	return writeJSON(dir, "world/scene.json", scene.Terrain)
}

func generateScene(seed int64, genre, intent string) ir.ScenePacket {
	rng := newRNG(seed)
	w, d := 24, 24
	cell := 2.0
	heights := make([]float64, w*d)
	surfaces := make([]string, w*d)
	for z := 0; z < d; z++ {
		for x := 0; x < w; x++ {
			n := 0.35*math.Sin(float64(x)*0.35+rng.f()*0.2) + 0.25*math.Cos(float64(z)*0.28)
			h := math.Max(0, n)
			kind := "ground"
			cx, cz := float64(x-w/2)*cell, float64(z-d/2)*cell
			if math.Hypot(cx+8, cz-6) < 5 {
				kind = "water"
				h = -0.4
			}
			if rng.f() < 0.06 && kind == "ground" && math.Hypot(cx, cz) > 6 {
				kind = "rock"
				h += 1.2
			}
			if rng.f() < 0.04 && kind == "ground" && math.Hypot(cx, cz) > 8 {
				kind = "obstacle"
				h += 0.8
			}
			i := z*w + x
			heights[i] = h
			surfaces[i] = kind
		}
	}
	regions := []ir.Region{
		{ID: "center", Category: "clearing", Bounds: [4]float64{-8, -8, 8, 8}, Accessibility: "open", Affordances: []string{"walk", "place"}, Walkability: "good"},
		{ID: "water_east", Category: "water", Bounds: [4]float64{2, -12, 16, 2}, Accessibility: "blocked", Affordances: []string{"avoid"}, Walkability: "poor"},
		{ID: "ridge_north", Category: "rock", Bounds: [4]float64{-20, 8, 20, 22}, Accessibility: "partial", Affordances: []string{"landmark"}, Walkability: "medium"},
	}
	env := "temperate clearing"
	summary := "A walkable clearing with water to one side and scattered rocks."
	if looksLike(intent, "forest", "森", "dark") {
		env = "wooded hollow"
	}
	if looksLike(intent, "snow", "ski", "冰") {
		env = "snow field"
	}
	if looksLike(intent, "神庙", "temple", "石廊", "逃亡", "跑酷") {
		env = "stone temple corridor"
		summary = "A walkable stone temple corridor with collapsed pits to one side and broken columns along the walls."
		regions[0].Category = "nave"
		regions[1].Category = "pit"
		regions[2].Category = "ruin"
	}
	_ = genre
	return ir.ScenePacket{
		Status:      ir.Status{Usable: true, Confidence: 0.86, Reason: "procedural heightfield"},
		Summary:     summary,
		Environment: env,
		Terrain:     ir.Terrain{Width: w, Depth: d, CellSize: cell, Heights: heights, Surfaces: surfaces},
		Regions:     regions,
		Notes: []string{
			"Center is open and suitable for spawn and collectibles.",
			"Water is not walkable.",
			"Rocks are scenery; do not tag existing rocks as gameplay obstacles.",
		},
		Camera: ir.Camera{Preset: normalizeCamera(genre), Position: ir.Vec3{0, 8, -12}, Pitch: -18},
		Genre:  normalizeCamera(genre),
		Seed:   seed,
		Surfaces: map[string][]any{
			"ground_regions":        {"center"},
			"water_regions":         {"water_east"},
			"rock_or_slope_regions": {"ridge_north"},
		},
	}
}

func (e *Engine) enrichSceneNotes(scene ir.ScenePacket, intent string) ir.ScenePacket {
	payload := map[string]any{
		"intent":      intent,
		"summary":     scene.Summary,
		"environment": scene.Environment,
		"regions":     ir.SanitizeForModel(scene.Regions),
		"notes":       scene.Notes,
	}
	raw, _ := json.Marshal(payload)
	sys := `Describe only what is visible. Return JSON {overall_summary, environment_visual_type, visual_notes:[]}. No gameplay rules. No coordinates. Do not invent objects that are not in the regions.`
	text, err := e.Complete(sys, string(raw))
	if err != nil {
		return scene
	}
	var got struct {
		Summary     string   `json:"overall_summary"`
		Environment string   `json:"environment_visual_type"`
		Notes       []string `json:"visual_notes"`
	}
	if json.Unmarshal([]byte(extractJSON(text)), &got) != nil {
		return scene
	}
	if strings.TrimSpace(got.Summary) != "" {
		scene.Summary = strings.TrimSpace(got.Summary)
	}
	if strings.TrimSpace(got.Environment) != "" {
		scene.Environment = strings.TrimSpace(got.Environment)
	}
	if len(got.Notes) > 0 {
		scene.Notes = got.Notes
	}
	scene.Status.Reason = "procedural heightfield + visual notes"
	return scene
}

func hashSeed(s string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return int64(h.Sum64())
}

type rng struct{ s uint64 }

func newRNG(seed int64) *rng { return &rng{s: uint64(seed) | 1} }

func (r *rng) u() uint64 {
	r.s ^= r.s << 13
	r.s ^= r.s >> 7
	r.s ^= r.s << 17
	return r.s
}

func (r *rng) f() float64 { return float64(r.u()%10000) / 10000 }

func looksLike(s string, keys ...string) bool {
	low := fmt.Sprintf("%s", s)
	for _, k := range keys {
		if containsFold(low, k) {
			return true
		}
	}
	return false
}
