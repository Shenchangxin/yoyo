package remake

import (
	"strings"
	"testing"
)

func TestCatalogIsThePinnedDistribution(t *testing.T) {
	seen := map[string]bool{}
	gates := map[int]int{}
	layers := map[string]int{}
	for _, pkg := range Catalog() {
		if seen[pkg.Name] {
			t.Fatalf("duplicate package %s", pkg.Name)
		}
		seen[pkg.Name] = true
		if pkg.Gate < 1 || pkg.Gate > 6 {
			t.Fatalf("%s gate %d", pkg.Name, pkg.Gate)
		}
		gates[pkg.Gate]++
		layers[pkg.Layer]++
		switch pkg.Status {
		case StatusPending, StatusPartial, StatusTranslated:
		default:
			t.Fatalf("%s status %s", pkg.Name, pkg.Status)
		}
		if pkg.Status == StatusPartial && pkg.Note == "" {
			t.Fatalf("%s is partial without a note", pkg.Name)
		}
	}
	if len(seen) != 86 {
		t.Fatalf("catalog has %d packages, pinned distribution has 86", len(seen))
	}
	wantLayers := map[string]int{
		"kernel-language": 15,
		"execution":       8,
		"time-picture":    29,
		"model":           14,
		"local-tool":      14,
		"shell":           3,
		"optional":        3,
	}
	for layer, n := range wantLayers {
		if layers[layer] != n {
			t.Fatalf("layer %s has %d, want %d", layer, layers[layer], n)
		}
	}
	if gates[1] != 15 || gates[2] != 8 || gates[3] != 29 || gates[4] != 28 || gates[5] != 3 || gates[6] != 3 {
		t.Fatalf("gates %#v", gates)
	}
	for _, name := range strings.Fields(pinnedPackages) {
		if !seen[name] {
			t.Fatalf("pinned package %s is missing from the catalog", name)
		}
		delete(seen, name)
	}
	if len(seen) != 0 {
		t.Fatalf("catalog has packages that are not in the pinned distribution: %#v", seen)
	}
}

const pinnedPackages = `
admission atomic-file audio-track author blob browser-capture caption caption-fine
cli comment-sticker compiler composition credential-store-env credential-store-local
depth-stack elevenlabs-speech endpoint executor facet film fishaudio-speech fontsource
generation gpt-image gpt-image-kits grok-imagine html-document html-program html-video
image-operations interview-emoji-reveal kernel loader markup media media-local
media-operations mimo-speech minimax-h3 nano-banana narrative narrative-caption
narrative-speech-alignment narrative-temporal pixverse producer project protocol
provider-beatapi provider-hiapi provider-html-local provider-hypihub
provider-image-opencv-local provider-monid provider-pollo provider-tokendance
provider-whisperx-local ranking recipe region-evidence result run runtime runtime-local
script seedance seedance-kits seedream source spatial speech-estimate speech-evidence
studio studio-companion temporal text text-fine timeline timeline-author video
visual-track volcengine-matting wan whisperx workspace yt-dlp
`

func TestTranslationIsNotInTheInstaller(t *testing.T) {
	if IncludeInInstaller() {
		t.Fatal("remake stays out of third-party builds; shipping 翻拍 is Yoyo's own cut flow")
	}
	if UpstreamCommit != "e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16" {
		t.Fatal("upstream pin changed without a catalog review")
	}
}
