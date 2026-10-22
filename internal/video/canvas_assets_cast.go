package video

import (
	"encoding/json"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

func (e *Engine) seedCastCharacterAssets(now string, existing map[string]bool) error {
	if err := e.migrateCastFolderIntoCharacters(now); err != nil {
		return err
	}
	// Stills live in the downloaded pack. A missing pack is not an error:
	// StartCharacterCast fills this in after the zip is installed.
	if err := e.importCharacterCast(now, existing); err != nil {
		diaglog.Warn("character cast import", "component", "video", "err", err)
	}
	return nil
}

func (e *Engine) migrateCastFolderIntoCharacters(now string) error {
	const oldFolder = "studio-folder-cast"
	const newFolder = "studio-folder-characters"
	for _, a := range loadCol[canvasAssetRec](e, colAssets) {
		if a.DeletedAt != "" {
			continue
		}
		payloadFolder := ""
		var extra map[string]any
		if a.PayloadJSON != "" {
			_ = json.Unmarshal([]byte(a.PayloadJSON), &extra)
			if extra != nil {
				payloadFolder, _ = extra["folderId"].(string)
			}
		}
		isCast := strings.HasPrefix(a.ID, "studio-cast-")
		inOld := a.FolderID == oldFolder || payloadFolder == oldFolder
		if !isCast && !inOld {
			continue
		}
		if a.FolderID == newFolder && payloadFolder == newFolder {
			continue
		}
		if err := e.setAssetFolder(a, newFolder, now); err != nil {
			return err
		}
	}
	if folder, err := getDoc[canvasFolderRec](e, colFolders, oldFolder); err == nil && folder.DeletedAt == "" {
		return e.deleteAssetFolder(oldFolder)
	}
	return nil
}
