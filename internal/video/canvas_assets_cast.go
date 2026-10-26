// 行为：把旧卡司文件夹迁进角色资产，并导入已下载的卡司静帧。清单不存在时导入返回 nil，不算错误。

package video

import (
	"encoding/json"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

// seedCastCharacterAssets 先迁移旧卡司文件夹，再导入已下载的静帧。导入失败只记警告，包缺失时导入本身返回 nil。
// now 传给 migrateCastFolderIntoCharacters 和 importCharacterCast。
// existing 传给 importCharacterCast。
func (e *Engine) seedCastCharacterAssets(now string, existing map[string]bool) error {
	if err := e.migrateCastFolderIntoCharacters(now); err != nil {
		return err
	}
	// 静帧在已下载的卡司包里。包缺失不算错误，StartCharacterCast 会在 zip 装好后再导入。
	if err := e.importCharacterCast(now, existing); err != nil {
		diaglog.Warn("character cast import", "component", "video", "err", err)
	}
	return nil
}

// migrateCastFolderIntoCharacters 把卡司前缀或旧文件夹里的素材改到角色资产，并删除旧文件夹。
// now 传给 setAssetFolder。
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
