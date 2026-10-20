package video

import (
	"encoding/json"
	"strings"
)

func (e *Engine) seedCastCharacterAssets(now string, existing map[string]bool) error {
	const folder = "studio-folder-characters"
	const pub = "/character-cast/"
	if err := e.migrateCastFolderIntoCharacters(now); err != nil {
		return err
	}
	put := func(id string, record map[string]any) {
		if existing[id] {
			return
		}
		e.upsertAsset(record)
		existing[id] = true
	}
	type still struct {
		file, id, title, note string
		tags                  []string
		bytes, w, h           int
	}
	stills := []still{
		{"cast-ea-child-storybook.jpg", "studio-cast-ea-child-storybook", "东亚幼年·纸鸢绘本", "绘本风东亚儿童，绿围巾与纸鸢。", []string{"角色", "设定图", "幼年", "东亚", "绘本"}, 290878, 1280, 720},
		{"cast-ea-teen-guoman.jpg", "studio-cast-ea-teen-guoman", "东亚少年·国漫校服", "国漫 2D 东亚少女，藏青外套。", []string{"角色", "设定图", "少年", "东亚", "国漫2D"}, 142452, 1280, 720},
		{"cast-ea-young-period.jpg", "studio-cast-ea-young-period", "东亚青年·唐风襦裙", "年代写实东亚女性，靛青襦裙。", []string{"角色", "设定图", "青年", "东亚", "年代"}, 204277, 1280, 720},
		{"cast-ea-youngman-cyber.jpg", "studio-cast-ea-youngman-cyber", "东亚青年·赛博夜行", "赛博写实东亚男性，青边光风衣。", []string{"角色", "设定图", "青年", "东亚", "赛博"}, 212746, 1280, 720},
		{"cast-ea-youngman-office.jpg", "studio-cast-ea-youngman-office", "东亚青年·通勤衬衫", "纪实东亚男性，松领带通勤。", []string{"角色", "设定图", "青年", "东亚", "纪实"}, 184902, 1280, 720},
		{"cast-ea-youngwoman-ink.jpg", "studio-cast-ea-youngwoman-ink", "东亚青年·水墨旅袍", "水墨叙事东亚女性，灰白旅袍。", []string{"角色", "设定图", "青年", "东亚", "水墨"}, 322263, 1280, 720},
		{"cast-ea-mid-ink.jpg", "studio-cast-ea-mid-ink", "东亚中年·墨色学者", "水墨东亚中年男性，油纸伞。", []string{"角色", "设定图", "中年", "东亚", "水墨"}, 348893, 1280, 720},
		{"cast-ea-midwoman-real.jpg", "studio-cast-ea-midwoman-real", "东亚中年·通勤风衣", "纪实东亚中年女性，炭色风衣。", []string{"角色", "设定图", "中年", "东亚", "纪实"}, 220028, 1280, 720},
		{"cast-ea-elder-photo.jpg", "studio-cast-ea-elder-photo", "东亚老年·卡其开衫", "写实东亚老年女性，金耳钉。", []string{"角色", "设定图", "老年", "东亚", "写实"}, 200582, 1280, 720},
		{"cast-ea-elderman-photo.jpg", "studio-cast-ea-elderman-photo", "东亚老年·折扇唐装", "写实东亚老年男性，灰唐装。", []string{"角色", "设定图", "老年", "东亚", "写实"}, 214506, 1280, 720},
		{"cast-sa-child-storybook.jpg", "studio-cast-sa-child-storybook", "南亚幼年·木陀螺", "卡通绘本南亚儿童，姜黄古尔塔。", []string{"角色", "设定图", "幼年", "南亚", "绘本"}, 419125, 1280, 720},
		{"cast-sa-teen-guoman.jpg", "studio-cast-sa-teen-guoman", "南亚少年·国漫卫衣", "国漫 2D 南亚少年，姜黄卫衣。", []string{"角色", "设定图", "少年", "南亚", "国漫2D"}, 132345, 1280, 720},
		{"cast-sa-young-cinematic.jpg", "studio-cast-sa-young-cinematic", "南亚青年·锈色丝绸", "电影感南亚女性，鼻钉与手镯。", []string{"角色", "设定图", "青年", "南亚", "写实"}, 160254, 1280, 720},
		{"cast-sa-mid-period.jpg", "studio-cast-sa-mid-period", "南亚中年·靛蓝头巾", "年代南亚中年男性，奶油色长袍。", []string{"角色", "设定图", "中年", "南亚", "年代"}, 192889, 1280, 720},
		{"cast-sa-elder-photo.jpg", "studio-cast-sa-elder-photo", "南亚老年·绛红纱丽", "写实南亚老年女性，棉纱丽。", []string{"角色", "设定图", "老年", "南亚", "写实"}, 139954, 1280, 720},
		{"cast-af-child-cartoon.jpg", "studio-cast-af-child-cartoon", "非洲裔幼年·三维卡通", "三维卡通非洲裔儿童，青绿卫衣。", []string{"角色", "设定图", "幼年", "非洲裔", "三维卡通"}, 161897, 1280, 720},
		{"cast-af-teen-street.jpg", "studio-cast-af-teen-street", "非洲裔少年·街头纪实", "纪实非洲裔少年，橄榄外套。", []string{"角色", "设定图", "少年", "非洲裔", "纪实"}, 290185, 1280, 720},
		{"cast-af-young-cinematic.jpg", "studio-cast-af-young-cinematic", "混血青年·驼色大衣", "电影感混血女性，短发金环。", []string{"角色", "设定图", "青年", "混血", "写实"}, 149211, 1280, 720},
		{"cast-af-youngwoman-photo.jpg", "studio-cast-af-youngwoman-photo", "非洲裔青年·橄榄夹克", "写实深肤色非洲裔女性，拳辫。", []string{"角色", "设定图", "青年", "非洲裔", "写实"}, 196584, 1280, 720},
		{"cast-af-youngman-fantasy.jpg", "studio-cast-af-youngman-fantasy", "非洲裔青年·青铜斥候", "奇幻三维非洲裔男性，皮甲短披。", []string{"角色", "设定图", "青年", "非洲裔", "奇幻3D"}, 248262, 1280, 720},
		{"cast-af-midwoman-real.jpg", "studio-cast-af-midwoman-real", "非洲裔中年·诊所夜班", "纪实非洲裔中年女性，护士服。", []string{"角色", "设定图", "中年", "非洲裔", "纪实"}, 141068, 1280, 720},
		{"cast-af-elder-photo.jpg", "studio-cast-af-elder-photo", "非洲裔老年·灯芯绒", "写实非洲裔老年男性，手杖。", []string{"角色", "设定图", "老年", "非洲裔", "写实"}, 255149, 1280, 720},
		{"cast-af-elderwoman-photo.jpg", "studio-cast-af-elderwoman-photo", "非洲裔老年·紫红大衣", "写实非洲裔老年女性，银白短发。", []string{"角色", "设定图", "老年", "非洲裔", "写实"}, 184481, 1280, 720},
		{"cast-eu-child-storybook.jpg", "studio-cast-eu-child-storybook", "欧洲裔幼年·锈色冬衣", "绘本北欧儿童，雀斑与松果。", []string{"角色", "设定图", "幼年", "欧洲裔", "绘本"}, 285577, 1280, 720},
		{"cast-eu-young-noir.jpg", "studio-cast-eu-young-noir", "欧洲裔青年·黑白风衣", "黑白黑色电影欧洲裔女性。", []string{"角色", "设定图", "青年", "欧洲裔", "黑色电影"}, 193425, 1280, 720},
		{"cast-eu-youngman-scifi.jpg", "studio-cast-eu-youngman-scifi", "欧洲裔青年·月背技工", "太空歌剧欧洲裔男性，工作连体衣。", []string{"角色", "设定图", "青年", "欧洲裔", "科幻"}, 165654, 1280, 720},
		{"cast-eu-young-wheelchair.jpg", "studio-cast-eu-young-wheelchair", "欧洲裔青年·画廊轮椅", "写实欧洲裔男性，现代轮椅。", []string{"角色", "设定图", "青年", "欧洲裔", "写实"}, 165260, 1280, 720},
		{"cast-eu-mid-period.jpg", "studio-cast-eu-mid-period", "欧洲裔中年·木工围裙", "年代欧洲裔中年男性，羊毛马甲。", []string{"角色", "设定图", "中年", "欧洲裔", "年代"}, 265401, 1280, 720},
		{"cast-eu-plus-mid-warm.jpg", "studio-cast-eu-plus-mid-warm", "欧洲裔中年·面包房围裙", "写实丰满欧洲裔中年女性。", []string{"角色", "设定图", "中年", "欧洲裔", "写实"}, 196701, 1280, 720},
		{"cast-eu-elder-warm.jpg", "studio-cast-eu-elder-warm", "欧洲裔老年·铁锈开衫", "写实欧洲裔老年女性，眼镜链。", []string{"角色", "设定图", "老年", "欧洲裔", "写实"}, 187524, 1280, 720},
		{"cast-me-child-storybook-v2.jpg", "studio-cast-me-child-storybook-v2", "中东幼年·泥鸟", "绘本中东男童，过大凉鞋。", []string{"角色", "设定图", "幼年", "中东", "绘本"}, 319774, 1280, 720},
		{"cast-me-child-storybook.jpg", "studio-cast-me-child-storybook", "中东青年·亚麻长袍", "绘本感中东青年，泥鸟与长袍。", []string{"角色", "设定图", "青年", "中东", "绘本"}, 358199, 1280, 720},
		{"cast-me-young-cinematic.jpg", "studio-cast-me-young-cinematic", "中东青年·尘粉长袖", "电影感中东女性，长袖连衣裙。", []string{"角色", "设定图", "青年", "中东", "写实"}, 173962, 1280, 720},
		{"cast-me-mid-period.jpg", "studio-cast-me-mid-period", "中东中年·棕色bisht", "年代中东中年男性，羊毛大氅。", []string{"角色", "设定图", "中年", "中东", "年代"}, 216646, 1280, 720},
		{"cast-me-elder-photo.jpg", "studio-cast-me-elder-photo", "中东老年·素巾外袍", "写实中东老年女性，素色头巾。", []string{"角色", "设定图", "老年", "中东", "写实"}, 202522, 1280, 720},
		{"cast-lat-child-storybook.jpg", "studio-cast-lat-child-storybook", "拉丁裔幼年·纸花", "绘本拉丁裔儿童，红开衫背带裤。", []string{"角色", "设定图", "幼年", "拉丁裔", "绘本"}, 366976, 1280, 720},
		{"cast-lat-teen-urban.jpg", "studio-cast-lat-teen-urban", "拉丁裔少年·姜黄帽衫", "街头纪实拉丁裔少年。", []string{"角色", "设定图", "少年", "拉丁裔", "纪实"}, 319388, 1280, 720},
		{"cast-lat-young-cinematic.jpg", "studio-cast-lat-young-cinematic", "拉丁裔青年·白衬衫", "电影感拉丁裔男性，瓜亚贝拉。", []string{"角色", "设定图", "青年", "拉丁裔", "写实"}, 154384, 1280, 720},
		{"cast-lat-mid-doc.jpg", "studio-cast-lat-mid-doc", "拉丁裔中年·牛仔衬衫", "纪实拉丁裔中年女性，灰白发丝。", []string{"角色", "设定图", "中年", "拉丁裔", "纪实"}, 210026, 1280, 720},
		{"cast-sea-child-storybook.jpg", "studio-cast-sea-child-storybook", "东南亚幼年·纸船", "绘本东南亚儿童，浅蓝短袖。", []string{"角色", "设定图", "幼年", "东南亚", "绘本"}, 407430, 1280, 720},
		{"cast-sea-young-cinematic.jpg", "studio-cast-sea-young-cinematic", "东南亚青年·鼠尾草衬衣", "电影感东南亚女性，亚麻长裤。", []string{"角色", "设定图", "青年", "东南亚", "写实"}, 246018, 1280, 720},
		{"cast-sea-elder-photo.jpg", "studio-cast-sea-elder-photo", "东南亚老年·蜡染衬衫", "写实东南亚老年男性，草帽。", []string{"角色", "设定图", "老年", "东南亚", "写实"}, 282985, 1280, 720},
		{"cast-ca-young-cinematic.jpg", "studio-cast-ca-young-cinematic", "中亚青年·炭色大衣", "电影感中亚男性，高颧骨。", []string{"角色", "设定图", "青年", "中亚", "写实"}, 235726, 1280, 720},
		{"cast-pi-youngman-photo.jpg", "studio-cast-pi-youngman-photo", "太平洋裔青年·船务衬衫", "写实太平洋裔男性，宽肩工装。", []string{"角色", "设定图", "青年", "太平洋裔", "写实"}, 133756, 1280, 720},
		{"cast-mixed-young-citypop.jpg", "studio-cast-mixed-young-citypop", "东亚青年·城市流行", "八十年代城市流行风东亚女性。", []string{"角色", "设定图", "青年", "东亚", "城市流行"}, 286612, 1280, 720},
		{"cast-indig-mid-cinematic.jpg", "studio-cast-indig-mid-cinematic", "中年女性·暮色风衣", "电影感中年女性，辫发与松石耳钉。", []string{"角色", "设定图", "中年", "原住民/当代", "写实"}, 177239, 1280, 720},
		{"cast-clay-child-mousekeeper.jpg", "studio-cast-clay-child-mousekeeper", "黏土幼年·黄雨衣", "黏土定格儿童角色，黄雨衣红靴。", []string{"角色", "设定图", "幼年", "风格化", "黏土定格"}, 320627, 1280, 720},
		{"cast-comic-youngman.jpg", "studio-cast-comic-youngman", "青年男性·漫画红外套", "漫画块面青年，红风衣信使包。", []string{"角色", "设定图", "青年", "东亚", "漫画"}, 252191, 1280, 720},
	}
	for _, s := range stills {
		src := pub + s.file
		put(s.id, map[string]any{
			"id": s.id, "kind": "image", "title": s.title, "coverUrl": src, "tags": s.tags,
			"folderId": folder, "category": "character", "status": "confirmed",
			"createdAt": now, "updatedAt": now, "source": "Yoyo 工作室", "note": s.note,
			"data": map[string]any{"dataUrl": src, "width": s.w, "height": s.h, "bytes": s.bytes, "mimeType": "image/jpeg"},
		})
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
