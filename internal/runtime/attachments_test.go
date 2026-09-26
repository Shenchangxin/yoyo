package runtime

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestImagePartsUsesMultimodalNotBinaryDump(t *testing.T) {
	raw := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 1, 2, 3, 4}
	b64 := base64.StdEncoding.EncodeToString(raw)
	atts := []Attachment{{Name: "shot.png", MIME: "image/png", DataB64: b64}}
	text := ExpandAttachments("", atts, 2400)
	if strings.Contains(text, "binary ") {
		t.Fatalf("image should not dump as binary: %s", text)
	}
	if !strings.Contains(text, "multimodal") {
		t.Fatalf("want multimodal note, got %s", text)
	}
	parts := ImageParts("", atts)
	if len(parts) != 1 || parts[0].Type != "image_url" || !strings.HasPrefix(parts[0].ImageURL, "data:image/png") {
		t.Fatalf("%+v", parts)
	}
}
