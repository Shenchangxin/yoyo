package video

import (
	"fmt"
	"strings"
	"testing"
)

func TestStudioCatalogsSeed(t *testing.T) {
	dir := t.TempDir()
	e, err := Open(dir, &memCAS{}, &memVault{m: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	page := e.listCanvasSkills(map[string]any{"page": 1, "pageSize": 200, "scope": "public"})
	skills, _ := page["skills"].([]map[string]any)
	if len(skills) < 70 {
		t.Fatalf("want >=70 studio skills, got %d", len(skills))
	}
	tags := map[string]int{}
	for _, skill := range skills {
		if fmt.Sprint(skill["tag"]) == "" || fmt.Sprint(skill["skillName"]) == "" {
			t.Fatalf("skill missing tag/name: %+v", skill["skillId"])
		}
		if fmt.Sprint(skill["instruction"]) == "" {
			t.Fatalf("skill missing body: %s", skill["skillId"])
		}
		tags[fmt.Sprint(skill["tag"])]++
	}
	for _, need := range []string{"drama", "ecommerce", "creative", "social", "others"} {
		if tags[need] == 0 {
			t.Fatalf("missing skill tag %s", need)
		}
	}

	env := e.canvasHTTP(map[string]any{"method": "GET", "path": "skills?page=1&pageSize=8&scope=public"})
	if env.Code != 0 {
		t.Fatalf("skills query path: %+v", env)
	}
	data, _ := env.Data.(map[string]any)
	listed, _ := data["skills"].([]map[string]any)
	if len(listed) != 8 {
		t.Fatalf("paginated skills via URL query got %d", len(listed))
	}

	assets := e.listAssetsPage("", "", "", "active", "", 1, 200, false, "")
	if intAny(assets["total"]) < 40 {
		t.Fatalf("want >=40 studio assets, got %v", assets["total"])
	}
	for _, item := range assets["assets"].([]map[string]any) {
		if fmt.Sprint(item["status"]) == "ready" {
			t.Fatalf("asset %s still uses cloud-era status ready", item["id"])
		}
		if fmt.Sprint(item["coverUrl"]) == "" || item["data"] == nil {
			t.Fatalf("asset %s missing cover/data", item["id"])
		}
		kind := fmt.Sprint(item["kind"])
		if kind != "image" && kind != "text" && kind != "entity" && kind != "video" {
			t.Fatalf("unexpected kind %s", kind)
		}
	}

	library := e.listAssetsPage("", "", "", "active", "", 1, 200, false, "entity")
	if intAny(library["total"]) <= 0 {
		t.Fatalf("library excludeKind=entity returned empty")
	}
	for _, item := range library["assets"].([]map[string]any) {
		if fmt.Sprint(item["kind"]) == "entity" {
			t.Fatalf("library listed entity %s", item["id"])
		}
	}
	folders := e.listAssetFolders()
	folderIDs := map[string]bool{}
	for _, folder := range folders {
		folderIDs[fmt.Sprint(folder["id"])] = true
	}
	if folderIDs["studio-folder-cast"] {
		t.Fatal("studio-folder-cast should be removed after merging into 角色资产")
	}
	if !folderIDs["studio-folder-characters"] {
		t.Fatal("missing studio-folder-characters")
	}
	castCount := 0
	allAssets := e.listAssetsPage("", "", "", "active", "", 1, 1000, false, "")
	for _, item := range allAssets["assets"].([]map[string]any) {
		id := fmt.Sprint(item["id"])
		if !strings.HasPrefix(id, "studio-cast-") {
			continue
		}
		castCount++
		if fmt.Sprint(item["folderId"]) != "studio-folder-characters" {
			t.Fatalf("cast asset %s folderId=%v", id, item["folderId"])
		}
		if fmt.Sprint(item["category"]) != "character" {
			t.Fatalf("cast asset %s category=%v", id, item["category"])
		}
	}
	if castCount < 40 {
		t.Fatalf("want >=40 character stills in 角色资产, got %d", castCount)
	}

	folderRaw := library["folderCounts"]
	folderCounts := map[string]int{}
	switch raw := folderRaw.(type) {
	case map[string]int:
		folderCounts = raw
	case map[string]any:
		for k, v := range raw {
			folderCounts[k] = intAny(v)
		}
	default:
		t.Fatalf("folderCounts type %T", folderRaw)
	}
	folderSum := 0
	for _, n := range folderCounts {
		folderSum += n
	}
	if folderSum != intAny(library["total"]) {
		t.Fatalf("folderCounts sum %d != library total %v", folderSum, library["total"])
	}
	tagged := e.listAssetsPage("", "", "", "active", "霓虹", 1, 40, false, "entity")
	if intAny(tagged["total"]) < 1 {
		t.Fatalf("payload search for 霓虹 returned empty")
	}

	payload := e.listPluginPayload()
	plugins, _ := payload["plugins"].([]map[string]any)
	if len(plugins) < 8 {
		t.Fatalf("want bundled plugins, got %d", len(plugins))
	}
	withCaps := 0
	for _, plugin := range plugins {
		manifest, _ := plugin["manifest"].(map[string]any)
		contrib, _ := manifest["contributes"].(map[string]any)
		providers, _ := contrib["providers"].([]any)
		if len(providers) > 0 {
			withCaps++
			prov, _ := providers[0].(map[string]any)
			caps, _ := prov["capabilities"].([]string)
			if len(caps) == 0 {
				if raw, ok := prov["capabilities"].([]any); ok {
					caps = make([]string, 0, len(raw))
					for _, c := range raw {
						caps = append(caps, fmt.Sprint(c))
					}
				}
			}
			okCap := false
			for _, c := range caps {
				if c == "text" || c == "image" || c == "video" || c == "audio" {
					okCap = true
				}
			}
			if !okCap {
				t.Fatalf("plugin %s providers missing protocol capabilities: %+v", manifest["id"], prov["capabilities"])
			}
		}
	}
	if withCaps < 5 {
		t.Fatalf("too few protocol plugins with capabilities: %d", withCaps)
	}
}

func TestCastAssetsMigrateIntoCharactersFolder(t *testing.T) {
	e := testEngine(t)
	now := Now()
	legacy := canvasFolderRec{ID: "studio-folder-cast", Name: "角色设定图", Position: 7, CreatedAt: now, UpdatedAt: now}
	if err := e.putDoc(colFolders, legacy.ID, legacy); err != nil {
		t.Fatal(err)
	}
	e.upsertAsset(map[string]any{
		"id": "studio-cast-legacy-still", "kind": "image", "title": "遗留设定图", "coverUrl": "/character-cast/x.jpg",
		"folderId": "studio-folder-cast", "category": "character", "status": "confirmed",
		"createdAt": now, "updatedAt": now, "source": "Yoyo 工作室",
		"data": map[string]any{"dataUrl": "/character-cast/x.jpg", "width": 1280, "height": 720, "bytes": 12, "mimeType": "image/jpeg"},
	})
	if err := e.seedStudioAssets(); err != nil {
		t.Fatal(err)
	}
	item, err := e.getAsset("studio-cast-legacy-still")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(item["folderId"]) != "studio-folder-characters" {
		t.Fatalf("legacy cast asset stayed in %v", item["folderId"])
	}
	for _, folder := range e.listAssetFolders() {
		if fmt.Sprint(folder["id"]) == "studio-folder-cast" {
			t.Fatal("角色设定图 folder should be deleted after migrate")
		}
	}
}
