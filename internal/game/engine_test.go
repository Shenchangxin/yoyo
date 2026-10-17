package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/game/ir"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	ws := t.TempDir()
	e, err := Open(filepath.Join(t.TempDir(), "docs"), func() string { return ws })
	if err != nil {
		t.Fatal(err)
	}
	e.SkipImage = true
	return e
}

func TestSanitizeStripsCoordinates(t *testing.T) {
	c := ir.Candidate{ID: "c_1", WorldXYZ: ir.Vec3{3, 1, 4}, GridCell: "center", SurfaceType: "ground"}
	got, _ := json.Marshal(ir.SanitizeForModel(c))
	if strings.Contains(string(got), "world_xyz") {
		t.Fatalf("coordinates leaked: %s", got)
	}
	if !strings.Contains(string(got), "c_1") {
		t.Fatalf("id missing: %s", got)
	}
}

func TestValidateRulesAndLogicalAssets(t *testing.T) {
	bad := ir.RulePlan{Objective: "x", Randomness: ir.Randomness{FreeNavigation: false}, AllowedActions: ir.CardinalMoves(), Required: []ir.ElementReq{{ElementType: "collectible", RequiredCount: 1}}}
	if err := ir.ValidateRules(bad); err == nil {
		t.Fatal("expected free_navigation error")
	}
	ok := heuristicRules("collect three relics", ir.ScenePacket{})
	if err := ir.ValidateRules(ok); err != nil {
		t.Fatal(err)
	}
	if err := ir.ValidateAsset(ir.Asset{Role: "player_spawn", SkipMesh: false}); err == nil {
		t.Fatal("logical mesh should fail")
	}
}

func TestPipelineCompilesSealedPlayable(t *testing.T) {
	e := testEngine(t)
	p, err := e.Create(Project{Title: "Relics", Intent: "Collect three relics and reach the shrine."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AnalyzeScene("", p.ID, p.Intent); err != nil {
		t.Fatal(err)
	}
	if _, err := e.PlanRules("", p.ID, p.Intent); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ExtractCandidates("", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Place("", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RealizeAssets("", p.ID); err != nil {
		t.Fatal(err)
	}
	p, err = e.Compile("", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(e.workspace(), filepath.FromSlash(p.Rel()), "playable", "game.json"))
	if err != nil {
		t.Fatal(err)
	}
	var play ir.Playable
	if err := json.Unmarshal(raw, &play); err != nil {
		t.Fatal(err)
	}
	if play.NeedCollect < 1 || len(play.Elements) < 2 || len(play.Interactions) < 1 {
		t.Fatalf("thin playable: %+v", play)
	}
	html, err := os.ReadFile(filepath.Join(e.workspace(), filepath.FromSlash(p.Rel()), "playable", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !PlayableLooksSealed(string(html)) {
		t.Fatal("runtime not sealed into index.html")
	}
	if strings.Contains(string(html), "__YOYO_GAME_JSON__") {
		t.Fatal("placeholder left in html")
	}
}

func TestPlaceDoesNotInventIDs(t *testing.T) {
	e := testEngine(t)
	p, _ := e.Create(Project{Title: "P", Intent: "collect two relics"})
	_, _ = e.AnalyzeScene("", p.ID, p.Intent)
	_, _ = e.PlanRules("", p.ID, p.Intent)
	_, _ = e.ExtractCandidates("", p.ID)
	p, err := e.Place("", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	var set ir.CandidateSet
	var plan ir.PlacementPlan
	_ = e.readIR(p, "candidates.json", &set)
	_ = e.readIR(p, "placement.json", &plan)
	ids := map[string]bool{}
	for _, c := range set.Candidates {
		ids[c.ID] = true
	}
	if len(plan.Elements) == 0 {
		t.Fatal("no placements")
	}
	for _, el := range plan.Elements {
		if !ids[el.CandidateID] {
			t.Fatalf("invented candidate %s", el.CandidateID)
		}
		if el.WorldXYZ == (ir.Vec3{}) && el.ElementType != "" {
			// origin is legal; just ensure candidate restore happened
		}
	}
}

func TestSnapshotAdoptsPass(t *testing.T) {
	e := testEngine(t)
	p, err := e.Create(Project{Title: "Relic run", Intent: "collect three relics"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AnalyzeScene("", p.ID, p.Intent); err != nil {
		t.Fatal(err)
	}
	if _, err := e.PlanRules("", p.ID, p.Intent); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ExtractCandidates("", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Place("", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RealizeAssets("", p.ID); err != nil {
		t.Fatal(err)
	}
	p, err = e.Compile("", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := e.projectDir(p)
	if err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{"status": "PASS", "checks": map[string]any{"render": "PASS"}}
	raw, _ := json.Marshal(doc)
	if err := os.WriteFile(filepath.Join(dir, "qa", "verification.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	snap := e.Snapshot(p)
	if fmt.Sprint(snap["stage"]) != StageChecked {
		t.Fatalf("stage=%v", snap["stage"])
	}
	got, err := e.Get(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Stage != StageChecked || got.Status != "verified" {
		t.Fatalf("persisted stage=%s status=%s", got.Stage, got.Status)
	}
}

func TestRepairUsesVerification(t *testing.T) {
	e := testEngine(t)
	p, _ := e.Create(Project{Title: "R", Intent: "collect three relics"})
	_, _ = e.AnalyzeScene("", p.ID, "")
	_, _ = e.PlanRules("", p.ID, "")
	_, _ = e.ExtractCandidates("", p.ID)
	_, _ = e.Place("", p.ID)
	_, _ = e.RealizeAssets("", p.ID)
	p, err := e.Compile("", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := e.projectDir(p)
	doc := map[string]any{"status": "FAIL", "checks": map[string]any{"outcome": "FAIL", "render": "PASS"}}
	raw, _ := json.Marshal(doc)
	if err := os.WriteFile(filepath.Join(dir, "qa", "verification.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := e.Repair("", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "repaired" {
		t.Fatalf("status %s", got.Status)
	}
}

func TestPlaceWithModelRestoresCoordinates(t *testing.T) {
	e := testEngine(t)
	p, _ := e.Create(Project{Title: "M", Intent: "collect two relics"})
	_, _ = e.AnalyzeScene("", p.ID, p.Intent)
	_, _ = e.PlanRules("", p.ID, p.Intent)
	_, _ = e.ExtractCandidates("", p.ID)
	var set ir.CandidateSet
	_ = e.readIR(p, "candidates.json", &set)
	if len(set.Candidates) < 4 {
		t.Fatal("need candidates")
	}
	if len(set.Candidates) < 5 {
		t.Fatal("need 5 candidates")
	}
	ids := []string{set.Candidates[0].ID, set.Candidates[1].ID, set.Candidates[2].ID, set.Candidates[3].ID, set.Candidates[4].ID}
	called := false
	e.Complete = func(system, user string) (string, error) {
		called = true
		if !strings.Contains(system, "candidate_id") && !strings.Contains(user, "candidates") {
			return "", fmt.Errorf("not a place call")
		}
		return fmt.Sprintf(`{"elements":[
			{"element_type":"player_spawn","candidate_id":%q,"source_requirement_index":0},
			{"element_type":"collectible","candidate_id":%q,"source_requirement_index":1},
			{"element_type":"collectible","candidate_id":%q,"source_requirement_index":1},
			{"element_type":"goal_area","candidate_id":%q,"source_requirement_index":2},
			{"element_type":"hazard_zone","candidate_id":%q,"source_requirement_index":3}
		]}`, ids[0], ids[1], ids[2], ids[3], ids[4]), nil
	}
	p, err := e.Place("", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	var plan ir.PlacementPlan
	_ = e.readIR(p, "placement.json", &plan)
	byCand := map[string]ir.Candidate{}
	for _, c := range set.Candidates {
		byCand[c.ID] = c
	}
	if !called {
		t.Fatal("model picker not called")
	}
	if len(plan.Elements) < 4 {
		t.Fatalf("elements: %+v", plan.Elements)
	}
	for _, el := range plan.Elements {
		c := byCand[el.CandidateID]
		if el.WorldXYZ != c.WorldXYZ {
			t.Fatalf("xyz not restored for %s: got %v want %v", el.CandidateID, el.WorldXYZ, c.WorldXYZ)
		}
	}
}

func TestBindAndTools(t *testing.T) {
	e := testEngine(t)
	out, err := e.Tool("sess", "game_create", map[string]any{"title": "Demo", "seed": "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "playable") {
		t.Fatalf("demo should compile: %s", out)
	}
	id, ok := e.SessionBind("sess")
	if !ok || id == "" {
		t.Fatal("bind missing")
	}
	if _, err := e.Tool("sess", "game_get", nil); err != nil {
		t.Fatal(err)
	}
	plan, err := e.Tool("sess", "game_verify", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan, "browser") && !strings.Contains(plan, "open") {
		t.Fatalf("verify plan: %s", plan)
	}
}

func TestCreateReusesBoundWorld(t *testing.T) {
	e := testEngine(t)
	a, err := e.Create(Project{Title: "神庙逃亡"})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Bind("sess", a.ID); err != nil {
		t.Fatal(err)
	}
	out, err := e.Tool("sess", "game_create", map[string]any{
		"title": "B", "intent": "神庙逃亡主题的第三人称3D跑酷小游戏",
	})
	if err != nil {
		t.Fatal(err)
	}
	id, ok := e.SessionBind("sess")
	if !ok || id != a.ID {
		t.Fatalf("create forked a second world: %s ok=%v out=%s", id, ok, out)
	}
	if len(e.List()) != 1 {
		t.Fatalf("worlds=%d out=%s", len(e.List()), out)
	}
	if strings.Contains(out, `"ir"`) || strings.Contains(out, "preview_html") {
		t.Fatalf("tool receipt must stay compact: %d bytes", len(out))
	}
	got, err := e.Get(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "神庙逃亡" {
		t.Fatalf("title overwritten: %q", got.Title)
	}
	if got.Stage != StageBuilt && got.Stage != StageChecked {
		t.Fatalf("bound world should compile in place, stage=%s", got.Stage)
	}
}

func TestHeuristicTempleTheme(t *testing.T) {
	plan := heuristicRules("神庙逃亡主题的第三人称3D跑酷小游戏", ir.ScenePacket{})
	if plan.Title != "神庙逃亡" || !strings.Contains(plan.Objective, "祭坛") {
		t.Fatalf("%+v", plan)
	}
	if genericWorldTitle("神庙逃亡") || !genericWorldTitle("Relic run") {
		t.Fatal("generic title")
	}
	scene := generateScene(1, "tps", "神庙逃亡石廊跑酷")
	if scene.Environment != "stone temple corridor" {
		t.Fatalf("%s", scene.Environment)
	}
}

func TestReceiptOmitsIR(t *testing.T) {
	e := testEngine(t)
	p, err := e.Create(Project{Title: "Relics", Intent: "collect three relics"})
	if err != nil {
		t.Fatal(err)
	}
	rec := e.Receipt(p)
	if _, ok := rec["ir"]; ok {
		t.Fatal(rec)
	}
	if _, ok := rec["preview_html"]; ok {
		t.Fatal(rec)
	}
	if rec["id"] != p.ID {
		t.Fatalf("%v", rec)
	}
}

func TestUnbindLeavesWorlds(t *testing.T) {
	e := testEngine(t)
	a, err := e.Create(Project{Title: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.Create(Project{Title: "B"})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Bind("sess", a.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.Bind("sess", b.ID); err != nil {
		t.Fatal(err)
	}
	id, ok := e.SessionBind("sess")
	if !ok || id != b.ID {
		t.Fatalf("bind=%s ok=%v", id, ok)
	}
	if err := e.Unbind("sess"); err != nil {
		t.Fatal(err)
	}
	if _, ok := e.SessionBind("sess"); ok {
		t.Fatal("session should be free after unbind")
	}
	if got := e.List(); len(got) != 2 {
		t.Fatalf("worlds=%d", len(got))
	}
}
