// 数据：黑神话短片的静帧、提示词和实体种子。seedBlackMythShortAssets 把尚未存在的条目写入黑神话文件夹。

package video

// seedBlackMythShortAssets 把尚未存在的黑神话静帧、提示词和实体写入黑神话文件夹。
// now 写入每条素材的 createdAt 和 updatedAt。
// existing 里已有的 id 直接跳过，写入后把该 id 标成已有。
func (e *Engine) seedBlackMythShortAssets(now string, existing map[string]bool) error {
	const folder = "studio-folder-black-myth"
	const pub = "/black-myth-shorts/"
	put := func(id string, record map[string]any) {
		if existing[id] {
			return
		}
		e.upsertAsset(record)
		existing[id] = true
	}
	type still struct {
		file, id, title, note, category string
		tags                            []string
		bytes, w, h                     int
	}
	stills := []still{
		{"bm-env-buddha-cliff.jpg", "studio-bm-still-sh01-buddha", "SH01 开场·巨佛崖山", "横屏开场空镜：雾海中的巨佛崖。画布第一张，接角色特写。", "environment", []string{"黑神话", "镜头01", "空镜", "横屏"}, 441646, 1280, 720},
		{"bm-hero-portrait.jpg", "studio-bm-still-sh02-portrait", "SH02 角色·行者特写", "行者定妆特写。后续有他的镜头都用这张保脸。", "character", []string{"黑神话", "镜头02", "角色", "横屏"}, 393334, 1280, 720},
		{"bm-env-ruined-temple.jpg", "studio-bm-still-sh03-temple", "SH03 空镜·破庙残阳", "破庙黄昏空镜，取棒之前的空间建立。", "environment", []string{"黑神话", "镜头03", "空镜", "横屏"}, 384993, 1280, 720},
		{"bm-shot-staff-take.jpg", "studio-bm-still-sh04-staff", "SH04 动作·祭坛取棒", "行者在祭坛前握住金箍棒。", "material", []string{"黑神话", "镜头04", "动作", "横屏"}, 373245, 1280, 720},
		{"bm-shot-walk-mist.jpg", "studio-bm-still-sh05-walk", "SH05 行进·踏雾崖道", "背影走入黄雾，转场用。", "material", []string{"黑神话", "镜头05", "行进", "横屏"}, 331717, 1280, 720},
		{"bm-shot-confrontation.jpg", "studio-bm-still-sh06-faceoff", "SH06 对峙·石妖现身", "行者与山门石妖对峙。", "material", []string{"黑神话", "镜头06", "对峙", "横屏"}, 407798, 1280, 720},
		{"bm-shot-combat-clean.jpg", "studio-bm-still-sh07-combat", "SH07 交锋·金棒破影", "单人格斗动作板，金棒扫碎石臂。", "material", []string{"黑神话", "镜头07", "动作", "横屏"}, 409592, 1280, 720},
		{"bm-shot-leap.jpg", "studio-bm-still-sh08-leap", "SH08 一跃·雷火裂谷", "低机位仰拍纵跃，接峰顶。", "material", []string{"黑神话", "镜头08", "动作", "横屏"}, 358362, 1280, 720},
		{"bm-shot-summit.jpg", "studio-bm-still-sh09-summit", "SH09 收势·云海峰顶", "战后站在峰顶看云海。", "material", []string{"黑神话", "镜头09", "收势", "横屏"}, 332255, 1280, 720},
		{"bm-shot-ending-clouds.jpg", "studio-bm-still-sh10-ending", "SH10 收束·云中佛面", "片尾空镜，云海托出佛面。", "environment", []string{"黑神话", "镜头10", "空镜", "横屏"}, 450694, 1280, 720},
		{"bm-hero-fullbody.jpg", "studio-bm-still-hero-fullbody", "角色全身·行者定妆", "全身定妆：青铜札甲、赤袍金云纹、右肩金甲。", "character", []string{"黑神话", "角色", "定妆"}, 408016, 1280, 720},
		{"bm-hero-turnaround.jpg", "studio-bm-still-hero-turnaround", "角色转面·行者四面", "四面转面，保服装一致性。", "character", []string{"黑神话", "角色", "转面"}, 356840, 1280, 720},
		{"bm-env-mist-path.jpg", "studio-bm-still-mist-path", "场景·黄雾崖道", "无人崖道空镜。", "environment", []string{"黑神话", "场景", "空镜"}, 356630, 1280, 720},
		{"bm-env-waterfall-cave.jpg", "studio-bm-still-waterfall-cave", "场景·水帘洞府", "水帘洞空镜，油灯与玉池。", "environment", []string{"黑神话", "场景", "空镜"}, 431092, 1280, 720},
		{"bm-env-ancient-village.jpg", "studio-bm-still-ancient-village", "场景·山村暮色", "山村蓝调空镜。", "environment", []string{"黑神话", "场景", "空镜"}, 352790, 1280, 720},
		{"bm-env-thunder-peak.jpg", "studio-bm-still-thunder-peak", "场景·雷裂峰门", "雷暴山门空镜。", "environment", []string{"黑神话", "场景", "空镜"}, 307137, 1280, 720},
		{"bm-prop-golden-staff.jpg", "studio-bm-still-prop-staff", "道具·金箍棒", "铁芯金箍棒静物。", "prop", []string{"黑神话", "道具"}, 369349, 1280, 720},
		{"bm-prop-purple-gourd.jpg", "studio-bm-still-prop-gourd", "道具·紫金葫芦", "紫金葫芦静物。", "prop", []string{"黑神话", "道具"}, 300297, 1280, 720},
		{"bm-prop-opera-mask.jpg", "studio-bm-still-prop-mask", "道具·残面", "开裂神面静物。", "prop", []string{"黑神话", "道具"}, 417737, 1280, 720},
		{"bm-shot-cave-oil.jpg", "studio-bm-still-cave-oil", "加镜头·洞中静立", "水帘洞内油灯勾甲，可插在 SH03 与 SH04 之间。", "material", []string{"黑神话", "加镜头"}, 437384, 1280, 720},
		{"bm-shot-combat.jpg", "studio-bm-still-combat-alt", "加镜头·棒势特写", "金棒挥击备选板。", "material", []string{"黑神话", "加镜头", "动作"}, 417054, 1280, 720},
		{"bm-v-buddha-cliff.jpg", "studio-bm-still-v-sh01-buddha", "竖屏 SH01 巨佛崖", "竖屏开场，接竖屏特写。", "environment", []string{"黑神话", "竖屏", "镜头01"}, 440168, 720, 1280},
		{"bm-v-hero-portrait.jpg", "studio-bm-still-v-sh02-portrait", "竖屏 SH02 行者特写", "竖屏角色特写。", "character", []string{"黑神话", "竖屏", "镜头02"}, 422758, 720, 1280},
		{"bm-v-walk-mist.jpg", "studio-bm-still-v-sh03-walk", "竖屏 SH03 踏雾", "竖屏背影踏雾。", "material", []string{"黑神话", "竖屏", "镜头03"}, 375523, 720, 1280},
		{"bm-v-combat-solo.jpg", "studio-bm-still-v-sh04-combat", "竖屏 SH04 纵跃", "竖屏单人格斗纵跃。", "material", []string{"黑神话", "竖屏", "镜头04"}, 318413, 720, 1280},
		{"bm-v-summit.jpg", "studio-bm-still-v-sh05-summit", "竖屏 SH05 云海收势", "竖屏片尾收势。", "material", []string{"黑神话", "竖屏", "镜头05"}, 355791, 720, 1280},
	}
	for _, s := range stills {
		src := pub + s.file
		put(s.id, map[string]any{
			"id": s.id, "kind": "image", "title": s.title, "coverUrl": src, "tags": s.tags,
			"folderId": folder, "category": s.category, "status": "confirmed",
			"createdAt": now, "updatedAt": now, "source": "Yoyo 工作室", "note": s.note,
			"data": map[string]any{"dataUrl": src, "width": s.w, "height": s.h, "bytes": s.bytes, "mimeType": "image/jpeg"},
		})
	}
	type clip struct {
		file, id, title, cover, category string
		tags                             []string
		bytes, w, h, durationMs          int
	}
	clips := []clip{
		{"clip-sh01-buddha.mp4", "studio-bm-clip-sh01-buddha", "镜头视频 SH01 巨佛崖", "bm-env-buddha-cliff.jpg", "environment", []string{"黑神话", "视频", "镜头01", "横屏"}, 3035967, 1920, 1080, 4000},
		{"clip-sh02-portrait.mp4", "studio-bm-clip-sh02-portrait", "镜头视频 SH02 行者特写", "bm-hero-portrait.jpg", "character", []string{"黑神话", "视频", "镜头02", "横屏"}, 2708735, 1920, 1080, 4000},
		{"clip-sh03-temple.mp4", "studio-bm-clip-sh03-temple", "镜头视频 SH03 破庙", "bm-env-ruined-temple.jpg", "environment", []string{"黑神话", "视频", "镜头03", "横屏"}, 2636183, 1920, 1080, 4000},
		{"clip-sh04-staff.mp4", "studio-bm-clip-sh04-staff", "镜头视频 SH04 取棒", "bm-shot-staff-take.jpg", "material", []string{"黑神话", "视频", "镜头04", "横屏"}, 2612875, 1920, 1080, 4000},
		{"clip-sh05-walk.mp4", "studio-bm-clip-sh05-walk", "镜头视频 SH05 踏雾", "bm-shot-walk-mist.jpg", "material", []string{"黑神话", "视频", "镜头05", "横屏"}, 2367146, 1920, 1080, 4000},
		{"clip-sh06-faceoff.mp4", "studio-bm-clip-sh06-faceoff", "镜头视频 SH06 对峙", "bm-shot-confrontation.jpg", "material", []string{"黑神话", "视频", "镜头06", "横屏"}, 2874122, 1920, 1080, 4000},
		{"clip-sh07-combat.mp4", "studio-bm-clip-sh07-combat", "镜头视频 SH07 交锋", "bm-shot-combat-clean.jpg", "material", []string{"黑神话", "视频", "镜头07", "横屏"}, 2513767, 1920, 1080, 4000},
		{"clip-sh08-leap.mp4", "studio-bm-clip-sh08-leap", "镜头视频 SH08 一跃", "bm-shot-leap.jpg", "material", []string{"黑神话", "视频", "镜头08", "横屏"}, 2371880, 1920, 1080, 4000},
		{"clip-sh09-summit.mp4", "studio-bm-clip-sh09-summit", "镜头视频 SH09 收势", "bm-shot-summit.jpg", "material", []string{"黑神话", "视频", "镜头09", "横屏"}, 2218578, 1920, 1080, 4000},
		{"clip-sh10-ending.mp4", "studio-bm-clip-sh10-ending", "镜头视频 SH10 收束", "bm-shot-ending-clouds.jpg", "environment", []string{"黑神话", "视频", "镜头10", "横屏"}, 3246956, 1920, 1080, 4000},
		{"clip-v-sh01-buddha.mp4", "studio-bm-clip-v-sh01-buddha", "竖屏视频 SH01 巨佛崖", "bm-v-buddha-cliff.jpg", "environment", []string{"黑神话", "视频", "竖屏", "镜头01"}, 3281810, 1080, 1920, 4000},
		{"clip-v-sh02-portrait.mp4", "studio-bm-clip-v-sh02-portrait", "竖屏视频 SH02 特写", "bm-v-hero-portrait.jpg", "character", []string{"黑神话", "视频", "竖屏", "镜头02"}, 2785069, 1080, 1920, 4000},
		{"clip-v-sh03-walk.mp4", "studio-bm-clip-v-sh03-walk", "竖屏视频 SH03 踏雾", "bm-v-walk-mist.jpg", "material", []string{"黑神话", "视频", "竖屏", "镜头03"}, 2736375, 1080, 1920, 4000},
		{"clip-v-sh04-combat.mp4", "studio-bm-clip-v-sh04-combat", "竖屏视频 SH04 纵跃", "bm-v-combat-solo.jpg", "material", []string{"黑神话", "视频", "竖屏", "镜头04"}, 2140383, 1080, 1920, 4000},
		{"clip-v-sh05-summit.mp4", "studio-bm-clip-v-sh05-summit", "竖屏视频 SH05 收势", "bm-v-summit.jpg", "material", []string{"黑神话", "视频", "竖屏", "镜头05"}, 2617582, 1080, 1920, 4000},
		{"assemble-landscape.mp4", "studio-bm-assemble-landscape", "成片预览·天命归处 横屏", "bm-shot-summit.jpg", "other", []string{"黑神话", "成片", "横屏"}, 26242671, 1920, 1080, 40000},
		{"assemble-vertical.mp4", "studio-bm-assemble-vertical", "成片预览·天命归处 竖屏", "bm-v-summit.jpg", "other", []string{"黑神话", "成片", "竖屏"}, 13343400, 1080, 1920, 20000},
	}
	for _, c := range clips {
		put(c.id, map[string]any{
			"id": c.id, "kind": "video", "title": c.title, "coverUrl": pub + c.cover, "tags": c.tags,
			"folderId": folder, "category": c.category, "status": "confirmed",
			"createdAt": now, "updatedAt": now, "source": "Yoyo 工作室",
			"note": "由高质量静帧升格 1080p 缓慢推镜，可直接插入画布视频节点。",
			"data": map[string]any{
				"url": pub + c.file, "width": c.w, "height": c.h, "durationMs": c.durationMs,
				"hasAudio": false, "bytes": c.bytes, "mimeType": "video/mp4",
			},
		})
	}
	texts := []struct {
		id, title, cover, content string
		tags                      []string
	}{
		{"studio-bm-text-wiring", "连线说明书·天命归处", "bm-shot-summit.jpg",
			"《天命归处》画布连线（横屏 40s）\nSH01 巨佛崖 → SH02 行者特写 → SH03 破庙残阳 → SH04 祭坛取棒 → SH05 踏雾崖道 → SH06 石妖对峙 → SH07 金棒破影 → SH08 雷火一跃 → SH09 云海收势 → SH10 云中佛面\n\n竖屏 20s：竖屏SH01 巨佛 → 竖屏SH02 特写 → 竖屏SH03 踏雾 → 竖屏SH04 纵跃 → 竖屏SH05 收势\n\n用法：素材分类选「黑神话短片」。图片当首帧参考，视频可直接插入时间线。角色保脸请始终引用「SH02 角色·行者特写」和「角色全身·行者定妆」。",
			[]string{"黑神话", "分镜", "连线"}},
		{"studio-bm-text-bible", "角色卡·行者", "bm-hero-portrait.jpg",
			"角色卡：行者（原创神话武者，非任何游戏官方造型）\n物种：成年雄性猿形武者。毛色金棕，眼周炭灰纹，左眉细疤，虹膜琥珀金。\n甲胄：风化青铜札甲带铜绿，赤绸内袍金云纹领，红绦腰带，右肩金色龙纹披膊。赤足。\n兵器：铁芯金箍棒，云纹浮雕，一节金箍有凹痕。\n禁用：现代服装、字幕、水印、商标、网红脸、第二套发型。\n光线：体积雾 + 金边逆光 + 冷雾补光。材质：湿石、铜绿、兽毛、丝绸。",
			[]string{"黑神话", "角色", "一致性"}},
		{"studio-bm-text-color", "色彩脚本·黄雾青铜", "bm-env-buddha-cliff.jpg",
			"色彩脚本\n60% 墨松、湿岩、青铜暗部；25% 黄雾与黄昏金；10% 赤袍；5% 雷电冷白只给 SH07/SH08。\n肤色/毛色保持金棕，不要漂成灰色。禁用全片霓虹和现代青橙。",
			[]string{"黑神话", "色彩"}},
	}
	for _, t := range texts {
		put(t.id, map[string]any{
			"id": t.id, "kind": "text", "title": t.title, "coverUrl": pub + t.cover, "tags": t.tags,
			"folderId": folder, "category": "other", "status": "confirmed",
			"createdAt": now, "updatedAt": now, "source": "Yoyo 工作室",
			"data": map[string]any{"content": t.content},
		})
	}
	entities := []struct {
		id, title, cover, category string
		def                        map[string]any
	}{
		{"studio-bm-entity-xingzhe", "角色·行者", "bm-hero-fullbody.jpg", "character", map[string]any{
			"type": "character", "name": "行者", "age": "成年",
			"wardrobe": []string{"风化青铜札甲", "赤绸内袍金云纹", "红绦腰带", "右肩金披膊"},
			"marker":   "左眉细疤 + 琥珀金虹膜",
		}},
		{"studio-bm-entity-staff", "道具·金箍棒", "bm-prop-golden-staff.jpg", "prop", map[string]any{
			"type": "prop", "name": "金箍棒", "duty": "SH04 取棒后全程识别", "wear": "一节金箍有凹痕",
		}},
		{"studio-bm-entity-gourd", "道具·紫金葫芦", "bm-prop-purple-gourd.jpg", "prop", map[string]any{
			"type": "prop", "name": "紫金葫芦", "duty": "行囊识别物", "wear": "口沿缺一角",
		}},
		{"studio-bm-entity-buddha", "场景·巨佛崖", "bm-env-buddha-cliff.jpg", "environment", map[string]any{
			"type": "location", "name": "巨佛崖山", "time": "黄昏/雾", "light": "体积黄雾 + 破云金光",
		}},
		{"studio-bm-entity-temple", "场景·破庙", "bm-env-ruined-temple.jpg", "environment", map[string]any{
			"type": "location", "name": "破庙残阳", "time": "黄昏", "light": "破瓦漏光 + 香炉残烟",
		}},
	}
	for _, ent := range entities {
		put(ent.id, map[string]any{
			"id": ent.id, "kind": "entity", "title": ent.title, "coverUrl": pub + ent.cover, "tags": []string{"黑神话", "实体", "可复用"},
			"folderId": folder, "category": ent.category, "status": "confirmed",
			"createdAt": now, "updatedAt": now, "source": "Yoyo 工作室",
			"data": map[string]any{"definition": ent.def},
		})
	}
	return nil
}
