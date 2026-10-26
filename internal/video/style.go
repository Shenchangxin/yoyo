// 画面风格的预设表，以及把它们写入、列出和拼进提示。
// 短片工程用哪一种风格不在这里决定。表里的每条提示是数据，不另作说明。

package video

import (
	"fmt"
	"sort"
	"strings"
)

// PresetStyles 返回内置画面风格。核心表在前，extraPresetStyles 接在后面。
func PresetStyles() []Style {
	core := []Style{
		{Value: "3d", Name: "3D cinematic", Prompt: "high-quality 3D CG animation still, modern game-engine cinematic render, Unreal Engine and Pixar grade quality, semi-realistic stylized characters with refined facial features, clean sculpted anatomy, detailed skin shader with subtle subsurface scattering, PBR materials with crisp detailed textures, volumetric cinematic lighting with soft rim light, rich depth of field, polished film color grading, detailed environment art, sharp focus, consistent character design across shots, avoid flat lighting, avoid plastic waxy skin, avoid low-poly blurry look, avoid 2D flat cel shading, avoid anime line art", Description: "Stylized 3D, not live-action."},
		{Value: "anime", Name: "Anime", Prompt: "Japanese TV anime style, clean cel shading with hard-edged shadow shapes, crisp uniform black line art, vivid saturated color palette, expressive large-eyed character design with on-model proportions, detailed hand-painted anime backgrounds, dramatic anime key lighting with screentone highlights, key-visual poster quality, consistent character design across shots, avoid 3D CGI look, avoid painterly soft blending, avoid watercolor texture, avoid photorealism, avoid thick western comic outlines", Description: "2D anime look."},
		{Value: "ghibli", Name: "Ghibli", Prompt: "Studio Ghibli hand-drawn animation style, soft painterly brushwork with organic hand-crafted line quality, lush warm watercolor painted backgrounds, gentle natural daylight with nostalgic warm glow, muted earthy natural color palette, whimsical cozy storybook atmosphere, subtle film-grain softness, theatrical background art quality, consistent character design across shots, avoid hard cel shading, avoid 3D render look, avoid neon over-saturated colors, avoid sharp digital edges, avoid photorealism", Description: "Hand-painted Ghibli still."},
		{Value: "watercolor", Name: "Watercolor", Prompt: "delicate watercolor storybook illustration, soft translucent color washes, visible cold-press paper texture, fluid hand-painted brushstrokes with gentle pigment bleeds, light airy atmosphere, harmonious pastel palette, whimsical children book charm, loose expressive edges, consistent character design across shots, avoid bold black outlines, avoid digital airbrush look, avoid harsh contrast, avoid 3D rendering, avoid photorealism", Description: "Paper wash."},
		{Value: "comic", Name: "Graphic novel", Prompt: "Western graphic-novel comic book style, bold confident black ink outlines, halftone dot shading and screentone gradients, dynamic saturated colors with dramatic contrast, dramatic spotlight lighting, flat graphic print look, sharp inking details, dynamic cinematic composition, consistent character design across shots, avoid painterly soft blending, avoid watercolor washes, avoid photorealistic rendering, avoid 3D CGI look, avoid anime cel shading", Description: "Print comic."},
		{Value: "guofeng", Name: "Guofeng 2.5D", Prompt: "Chinese guofeng 2.5D illustration style, semi-realistic donghua-quality character art, elegant flowing line work, rich traditional Chinese aesthetic elements, layered ink-wash inspired atmospheric backgrounds, refined silk and fabric textures, soft luminous lighting with gentle haze, sophisticated muted jewel-tone palette, xianxia drama poster quality, consistent character design across shots, avoid flat cel shading, avoid western comic ink style, avoid photorealism, avoid plastic 3D look, avoid modern clothing and props unless specified", Description: "Donghua 2.5D."},
		{Value: "webtoon", Name: "Webtoon", Prompt: "Korean webtoon manhwa style, clean digital painting with soft gradient shading, slim elegant character proportions, large expressive eyes with detailed highlights, soft glowing skin rendering, romantic dreamy lighting, modern pastel-to-vivid color palette, detailed fashion and fabric rendering, webtoon key visual quality, consistent character design across shots, avoid heavy black ink outlines, avoid halftone dots, avoid 3D render look, avoid watercolor paper texture, avoid chibi proportions", Description: "Korean manhwa."},
		{Value: "noir", Name: "Noir ink", Prompt: "black and white manga illustration, high-contrast monochrome ink work, dynamic hatching and cross-hatching shading, bold solid blacks with dramatic negative space, screentone gray gradation, expressive confident ink linework, cinematic noir lighting, professional manga page quality, consistent character design across shots, strictly no color, avoid grayscale blur smudging, avoid painterly soft edges, avoid photorealism, avoid 3D render look", Description: "Monochrome manga."},
		{Value: "ink", Name: "Ink wash", Prompt: "ink-wash illustration, rice-paper grain, restrained palette, calligraphic line, atmospheric haze, not photoreal", Description: "East-Asian ink."},
		{Value: "clay", Name: "Stop-motion clay", Prompt: "stop-motion clay miniature, visible fingerprints, practical set lighting, tactile materials, not photoreal humans", Description: "Claymation."},
		{Value: "painterly", Name: "Painterly", Prompt: "oil-paint cinematic frame, visible brush, rich midtones, controlled palette, character likeness held across shots, not photoreal photo", Description: "Painted film."},
		{Value: "cyber", Name: "Neon noir", Prompt: "neon-noir 3D, wet streets, cyan-magenta practical lights, graphic silhouettes, stylized not photoreal faces", Description: "Night city."},
		{Value: "pixar", Name: "Pixar volume", Prompt: "Pixar-like stylized 3D animation still, appealing sculpted cartoon anatomy, large expressive eyes, subsurface-scattering skin, rich cloth and hair simulation, warm cinematic lighting, shallow depth of field, polished feature-animation color, consistent character design across shots, avoid photoreal human pores, avoid 2D line art, avoid plastic toy look, avoid low-poly game assets", Description: "Appealing 3D cartoon."},
		{Value: "disney", Name: "Storybook 3D", Prompt: "Disney fairy-tale 3D animation still, luminous storybook characters, silky animated hair, rounded appealing forms, magical warm rim light, richly painted environments, theatrical color, consistent character design across shots, avoid photorealism, avoid grimdark contrast, avoid flat 2D cel shading, avoid western comic ink", Description: "Fairy-tale volume."},
		{Value: "shinkai", Name: "Luminous city", Prompt: "Makoto Shinkai anime still, hyper-detailed luminous city bokeh, sparkling rain highlights, saturated twilight blues and golds, crisp refined linework, breathtaking painted sky, emotional close-up lighting, consistent character design across shots, avoid 3D CGI, avoid muddy colors, avoid photoreal faces, avoid thick western outlines", Description: "Luminous anime city."},
		{Value: "live", Name: "Live-action", Prompt: "photoreal live-action short-drama still, 35mm cinematic photography, natural contemporary East-Asian faces, practical tungsten and neon lights, wet asphalt, grounded Chinese city texture, shallow depth of field, film grain, consistent wardrobe and likeness across shots, avoid illustration, avoid anime eyes, avoid plastic CGI skin, avoid overcooked beauty filter", Description: "Cinematic live plate."},
		{Value: "dongman", Name: "Donghua 2D", Prompt: "Chinese 2D donghua key visual, elegant clean line art, refined cel shading, jewel-tone palette, expressive but restrained faces, detailed urban or period backgrounds, consistent character design across shots, avoid Japanese screentone, avoid photorealism, avoid 3D game render, avoid chibi proportions", Description: "Mainland 2D donghua."},
		{Value: "chibi", Name: "Chibi", Prompt: "super-deformed chibi Q-version character still, oversized head, huge sparkling eyes, tiny body, rounded cute volumes, pastel kawaii color, sticker-like clarity, consistent costume colors across shots, avoid realistic anatomy, avoid horror distortion, avoid photoreal skin, avoid dense adult drama lighting", Description: "Q-version cute."},
		{Value: "pixel", Name: "Pixel", Prompt: "strict 16-bit pixel-art cinematic still, visible square pixels, limited palette, dithered rain and shadows, hard pixel outlines, SNES RPG screenshot quality, consistent sprite design across shots, avoid smooth 3D rendering, avoid photorealism, avoid painterly blending, avoid vector smoothness", Description: "Retro pixel frame."},
		{Value: "shoujo", Name: "Shojo", Prompt: "classic shojo manga still, flowing hair, glittering large eyes, floral sparkle highlights, delicate screentone, romantic pastel palette, elegant fashion linework, consistent character design across shots, avoid grim seinen hatching, avoid 3D CGI, avoid photorealism, avoid thick superhero outlines", Description: "Romantic shojo."},
		{Value: "xianxia", Name: "Xianxia", Prompt: "Chinese xianxia spectacle still, embroidered flowing robes, mountain pavilion and cloud sea, jade and cinnabar palette, luminous spiritual haze, epic donghua poster quality, consistent character design and costume heraldry across shots, avoid European castles, avoid modern streetwear, avoid photoreal tourism photo, avoid plastic 3D armor", Description: "Xianxia spectacle."},
		{Value: "retro80", Name: "80s cel", Prompt: "1980s Japanese TV-anime cel still, analog film grain, warm cathode glow, hand-painted backgrounds, vintage cel shading, slightly soft VHS edges, consistent character model across shots, avoid modern digital smoothness, avoid 3D CGI, avoid photorealism, avoid neon cyberpunk overload", Description: "Vintage 80s cel."},
		{Value: "ukiyo", Name: "Ukiyo-e", Prompt: "Japanese ukiyo-e woodblock still, flat decorative perspective, mineral pigments, bold contour lines, patterned kimono and rain, Edo print paper texture, consistent character design across shots, avoid photorealism, avoid 3D shading, avoid western comic dots, avoid modern clothing", Description: "Woodblock print."},
		{Value: "sketch", Name: "Pencil sketch", Prompt: "refined graphite pencil storyboard still, visible paper grain, cross-hatching, soft charcoal shadows, monochrome drawing, consistent character construction across shots, avoid full color, avoid photoreal photo, avoid 3D render, avoid messy scribbles that lose likeness", Description: "Graphite board."},
		{Value: "kdrama", Name: "K-drama", Prompt: "Korean-drama cinematic still, glossy contemporary styling, teal-and-amber color grade, romantic city bokeh, luxury soft lighting, photoreal East-Asian faces, consistent wardrobe and makeup across shots, avoid anime illustration, avoid grimy documentary look, avoid oversaturated neon cyberpunk, avoid comic outlines", Description: "K-drama grade."},
		{Value: "wes", Name: "Pastel symmetry", Prompt: "Wes Anderson style still, perfectly symmetrical one-point composition, centered subject, flat even lighting, vintage pastel pink mustard and teal storefronts, meticulous production design, consistent costume and palette across shots, avoid handheld grit, avoid neon cyberpunk, avoid photoreal HDR, avoid chaotic staging", Description: "Pastel symmetry."},
		{Value: "mecha", Name: "Mecha", Prompt: "giant mecha cinematic still, towering robot over an industrial harbor, orange warning lights, steel and hydraulic detail, scale comparison with tiny cranes, dramatic low angle, consistent mech design across shots, avoid cute chibi, avoid photoreal humans filling the frame, avoid soft fairy-tale color", Description: "Giant robot."},
		{Value: "lego", Name: "Brick build", Prompt: "LEGO brick miniature still, visible plastic studs, toy-scale alley and minifig, saturated primary bricks, practical tabletop lighting, consistent brick architecture across shots, avoid smooth 3D skin, avoid photoreal streets, avoid painterly blending", Description: "Brick toy."},
		{Value: "lowpoly", Name: "Low poly", Prompt: "low-poly 3D still, faceted geometric mountain cabin, flat shaded triangles, limited color blocks, clean game-art lighting, consistent mesh language across shots, avoid high-poly realism, avoid 2D line art, avoid photographic texture", Description: "Faceted 3D."},
		{Value: "school", Name: "Campus anime", Prompt: "sunlit Japanese high-school anime still, empty corridor and cherry trees outside, warm afternoon light, clean cel shading, school uniforms on distant students only, consistent campus architecture across shots, avoid night rain, avoid photorealism, avoid 3D CGI", Description: "Campus anime."},
		{Value: "sports", Name: "Sports anime", Prompt: "sports anime key visual, night basketball court, motion streaks, sweat highlights, dramatic under-hoop camera, saturated team colors, consistent jersey design across shots, avoid photoreal faces filling frame, avoid rainy alley, avoid painterly oil", Description: "Sports key visual."},
		{Value: "action", Name: "Ink action", Prompt: "high-contrast ink-action still, black splashes and speed lines, two fighters mid-clash, graphic negative space, limited red accent, consistent costume silhouettes across shots, avoid soft romance sparkle, avoid photorealism, avoid 3D render", Description: "Ink fight."},
		{Value: "papercut", Name: "Paper-cut", Prompt: "Chinese red paper-cut still, flat layered silhouette of a dragon and full moon, knife-cut edges, red and black only, folk craft texture, consistent motif across shots, avoid shading, avoid photorealism, avoid 3D depth", Description: "Red paper-cut."},
		{Value: "stained", Name: "Stained glass", Prompt: "cathedral stained-glass still, jewel-tone glass fragments of a knight, thick lead cames, backlight glow, no photographic people, consistent window design across shots, avoid 3D sculpture, avoid rainy streets, avoid anime line art", Description: "Stained glass."},
		{Value: "gouache", Name: "Gouache", Prompt: "opaque gouache village festival still, matte chalky pigment, paper tooth, folk dancers and lanterns, daylight color, consistent painted shapes across shots, avoid glossy oil, avoid photorealism, avoid 3D CGI", Description: "Gouache poster."},
		{Value: "vaporwave", Name: "Vaporwave", Prompt: "vaporwave still, marble bust, chrome grid, sunset gradient, VHS date stamp avoided, pink and cyan, empty plaza, consistent graphic props across shots, avoid rainy alley, avoid photoreal portraits, avoid hand-drawn anime", Description: "Retro-futurist."},
		{Value: "wuxia", Name: "Wuxia", Prompt: "wuxia still, bamboo raft at dawn, two swordsmen in muted teal robes, misty river, practical martial-arts staging, consistent weapon and costume across shots, avoid glowing xianxia magic, avoid photoreal tourism, avoid modern clothing", Description: "Jianghu dawn."},
		{Value: "period", Name: "Period court", Prompt: "Tang-dynasty court banquet still, vermillion pillars, gold ritual vessels, layered ceremonial robes, candle and daylight mix, consistent rank costume across shots, avoid modern makeup, avoid xianxia flying swords, avoid photoreal hotel interiors", Description: "Court epic."},
		{Value: "fable", Name: "Animal fable", Prompt: "storybook animal-fable still, clothed red fox carrying a letter through autumn woods, storybook lighting, consistent creature design across shots, avoid human faces, avoid photoreal wildlife documentary, avoid horror", Description: "Talking-animal tale."},
		{Value: "horror", Name: "Horror", Prompt: "horror still, empty hospital corridor, sick green fluorescent, one distant figure, wet floor reflections, oppressive quiet, consistent location across shots, avoid gore close-ups, avoid cute characters, avoid neon cyberpunk glamour", Description: "Quiet horror."},
		{Value: "pastoral", Name: "Pastoral", Prompt: "pastoral still, rice terraces and a water buffalo at noon, gold-green daylight, rural houses, no fashion portrait, consistent village geography across shots, avoid night rain, avoid neon, avoid palace interiors", Description: "Rural noon."},
		{Value: "space", Name: "Space opera", Prompt: "space-opera still, astronaut silhouette against Saturn rings, starfield, spacecraft hull in foreground, cold cyan and black, consistent suit hardware across shots, avoid rainy city, avoid fantasy castles, avoid cute chibi", Description: "Orbital spectacle."},
		{Value: "steampunk", Name: "Steampunk", Prompt: "steampunk still, brass airship over Victorian docks, gears and copper pipes, sepia coal smoke, consistent vehicle design across shots, avoid neon cyberpunk, avoid photoreal modern city, avoid anime school", Description: "Brass airship."},
		{Value: "isometric", Name: "Isometric", Prompt: "isometric miniature city still, rainy night clinic as a cutaway diorama, tiny furniture, even three-quarter light, consistent tile scale across shots, avoid first-person portraits, avoid photoreal handheld, avoid 2D cel", Description: "Tiny diorama."},
		{Value: "documentary", Name: "Documentary", Prompt: "handheld documentary still, crowded wet market, available fluorescent light, real produce and steam, no posed beauty portrait, consistent location across shots, avoid cinematic teal-orange grade, avoid illustration, avoid studio lighting", Description: "Street documentary."},
		{Value: "felt", Name: "Felt puppet", Prompt: "wool-felt stop-motion still, needle-felt fox puppet in a woodland set, visible fiber texture, practical tabletop light, consistent puppet face across shots, avoid clay fingerprints, avoid photoreal fur, avoid 2D drawing", Description: "Felt puppet."},
		{Value: "risograph", Name: "Risograph", Prompt: "risograph print still, misregistered pink and yellow ink, grainy paper, limited spot colors, poster flatness, consistent graphic shapes across shots, avoid photoreal portraits, avoid rainy alley, avoid 3D CGI", Description: "Risograph print."},
		{Value: "crayon", Name: "Crayon", Prompt: "wax crayon illustration still, waxy strokes, kids-book hillside playground, thick pigment, consistent drawn shapes across shots, avoid photoreal faces, avoid rain alley, avoid digital smoothness", Description: "Crayon drawing."},
		{Value: "collage", Name: "Collage", Prompt: "analog paper collage still, torn magazine fragments and a paper moon, tape edges, layered cutouts, consistent collage language across shots, avoid seamless photoreal, avoid rainy heroine, avoid 3D render", Description: "Cut-paper collage."},
		{Value: "blueprint", Name: "Blueprint", Prompt: "cyanotype blueprint still, white construction lines on blue paper, architectural station plan, consistent drafting language across shots, avoid photoreal people, avoid color photography, avoid anime", Description: "Cyan blueprint."},
		{Value: "seinen", Name: "Seinen", Prompt: "seinen manga still, late-night empty office, heavy hatching, adult atmosphere, monochrome ink, consistent location across shots, avoid shojo sparkle, avoid chibi, avoid photoreal color", Description: "Adult manga."},
		{Value: "manhua", Name: "Manhua color", Prompt: "Chinese manhua color still, crowded night food street, saturated digital comic color, clean graphic staging, consistent street design across shots, avoid rainy umbrella heroine, avoid photoreal HDR, avoid Japanese screentone", Description: "Color manhua."},
		{Value: "gothic", Name: "Gothic", Prompt: "gothic cathedral nave still, candlelight, pointed arches, stone dust, consistent architecture across shots, avoid modern clothing, avoid rainy neon city, avoid cute chibi", Description: "Gothic nave."},
		{Value: "western", Name: "Western", Prompt: "western still, desert noon main street, wooden saloon, dust and hitching post, consistent town layout across shots, avoid rainy alley, avoid neon, avoid anime school", Description: "Desert western."},
		{Value: "underwater", Name: "Underwater", Prompt: "underwater still, sun shafts through teal water, a shipwreck ribcage, drifting particles, consistent marine geography across shots, avoid surface city, avoid umbrellas, avoid dry land portraits", Description: "Undersea."},
		{Value: "citypop", Name: "City pop", Prompt: "1980s Japanese city-pop still, sunset overpass and palm, cassette-era color, airbrushed sky, consistent city props across shots, avoid rainy night alley, avoid grim seinen, avoid photoreal modern phones", Description: "City-pop dusk."},
		{Value: "lofi", Name: "Lo-fi room", Prompt: "lo-fi bedroom still, rainy window as background only, warm lamp, vinyl and plants, cozy illustration, no close-up face, consistent room clutter across shots, avoid fashion portrait, avoid horror, avoid 3D plastic", Description: "Lo-fi room."},
		{Value: "origami", Name: "Origami", Prompt: "origami still, folded paper crane flock on a paper lake, visible crease geometry, studio craft light, consistent paper language across shots, avoid photoreal birds, avoid clay, avoid 2D sketch", Description: "Folded paper."},
	}
	return append(core, extraPresetStyles()...)
}

// seedStyles 把还没有的预设写进库。四条旧的短提示如果还没改过，换成现在的长提示。已经改过的不覆盖。
func (e *Engine) seedStyles() error {
	now := Now()
	seeds := PresetStyles()
	legacy := map[string]string{
		"3d":         "cinematic 3D animation, physically based materials, soft volumetric light, filmic contrast, coherent character design, no photoreal human skin",
		"anime":      "high-end anime still, clean linework, cel shading, expressive eyes, consistent costume color, cinematic lighting, not photoreal",
		"watercolor": "watercolor storyboard, paper texture, bleeding pigment, soft edges, consistent costume hues, not photoreal",
		"comic":      "graphic-novel panel, bold ink, limited flat color, dramatic shadow, consistent character model sheets, not photoreal",
	}
	existing := loadCol[styleRec](e, colStyles)
	byValue := map[string]styleRec{}
	for _, s := range existing {
		byValue[s.Value] = s
	}
	for i, s := range seeds {
		if cur, ok := byValue[s.Value]; ok {
			if old, hit := legacy[s.Value]; hit && cur.Prompt == old {
				cur.Name = s.Name
				cur.Prompt = s.Prompt
				cur.Description = s.Description
				cur.SortOrder = i
				cur.UpdatedAt = now
				if err := e.putDoc(colStyles, cur.ID, cur); err != nil {
					return err
				}
			}
			continue
		}
		rec := styleRec{Style: Style{ID: NewID(), Name: s.Name, Value: s.Value, Prompt: s.Prompt, Description: s.Description, SortOrder: i, IsActive: true}, Seeded: true, CreatedAt: now, UpdatedAt: now}
		if err := e.putDoc(colStyles, rec.ID, rec); err != nil {
			return err
		}
	}
	return nil
}

// ListStyles 只返回仍启用的风格，先按排序再按名字。没有时返回空切片。
func (e *Engine) ListStyles() ([]Style, error) {
	var out []Style
	for _, s := range loadCol[styleRec](e, colStyles) {
		if s.IsActive {
			out = append(out, s.Style)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].Name < out[j].Name
	})
	if out == nil {
		out = []Style{}
	}
	return out, nil
}

// ListAllStyles 连停用的一起返回，排序和 ListStyles 一样。没有时返回空切片。
func (e *Engine) ListAllStyles() ([]Style, error) {
	var out []Style
	for _, s := range loadCol[styleRec](e, colStyles) {
		out = append(out, s.Style)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].Name < out[j].Name
	})
	if out == nil {
		out = []Style{}
	}
	return out, nil
}

// StyleByValue 按 value 找一条。没有就返回 style not found。
func (e *Engine) StyleByValue(value string) (Style, error) {
	for _, s := range loadCol[styleRec](e, colStyles) {
		if s.Value == value {
			return s.Style, nil
		}
	}
	return Style{}, fmt.Errorf("style not found")
}

// PrefixStyle 把风格提示接在画面提示前面。库里没有就看预设表。都没有，或风格提示是空的，就原样返回画面提示。画面提示是空的就只返回风格提示。
func (e *Engine) PrefixStyle(value, prompt string) string {
	s, err := e.StyleByValue(value)
	if err != nil || s.Prompt == "" {
		for _, p := range PresetStyles() {
			if p.Value == value {
				s = p
				err = nil
				break
			}
		}
	}
	if err != nil || s.Prompt == "" {
		return prompt
	}
	if prompt == "" {
		return s.Prompt
	}
	return s.Prompt + ". " + prompt
}

// UpsertStyle 写入一条风格。value 空着就拒绝。没给编号时按 value 找已有的，仍没有就新建。
func (e *Engine) UpsertStyle(s Style) (Style, error) {
	now := Now()
	if s.Value == "" {
		return s, fmt.Errorf("style value required")
	}
	if s.ID == "" {
		for _, cur := range loadCol[styleRec](e, colStyles) {
			if cur.Value == s.Value {
				s.ID = cur.ID
				break
			}
		}
		if s.ID == "" {
			s.ID = NewID()
		}
	}
	rec, err := getDoc[styleRec](e, colStyles, s.ID)
	if err != nil {
		rec = styleRec{CreatedAt: now}
	}
	rec.Style = s
	rec.UpdatedAt = now
	if rec.CreatedAt == "" {
		rec.CreatedAt = now
	}
	return s, e.putDoc(colStyles, s.ID, rec)
}

// DeleteStyle 把风格标成停用，不删文档。
func (e *Engine) DeleteStyle(id string) error {
	rec, err := getDoc[styleRec](e, colStyles, id)
	if err != nil {
		return err
	}
	rec.IsActive = false
	rec.UpdatedAt = Now()
	return e.putDoc(colStyles, id, rec)
}

// styleValueKey 去掉 value 两端的空白。
func styleValueKey(v string) string { return strings.TrimSpace(v) }
