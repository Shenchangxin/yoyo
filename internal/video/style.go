package video

import "fmt"

func (e *Engine) seedStyles() error {
	now := Now()
	seeds := []Style{
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
	}
	legacy := map[string]string{
		"3d":         "cinematic 3D animation, physically based materials, soft volumetric light, filmic contrast, coherent character design, no photoreal human skin",
		"anime":      "high-end anime still, clean linework, cel shading, expressive eyes, consistent costume color, cinematic lighting, not photoreal",
		"watercolor": "watercolor storyboard, paper texture, bleeding pigment, soft edges, consistent costume hues, not photoreal",
		"comic":      "graphic-novel panel, bold ink, limited flat color, dramatic shadow, consistent character model sheets, not photoreal",
	}
	for i, s := range seeds {
		_, err := e.DB.Exec(`INSERT OR IGNORE INTO style_presets(id, name, value, prompt, description, sort_order, is_active, seeded, created_at, updated_at)
			VALUES(?,?,?,?,?,?,1,1,?,?)`, NewID(), s.Name, s.Value, s.Prompt, s.Description, i, now, now)
		if err != nil {
			return err
		}
		if old, ok := legacy[s.Value]; ok {
			_, _ = e.DB.Exec(`UPDATE style_presets SET name=?, prompt=?, description=?, sort_order=?, updated_at=? WHERE value=? AND prompt=?`,
				s.Name, s.Prompt, s.Description, i, now, s.Value, old)
		}
	}
	return nil
}

func (e *Engine) ListStyles() ([]Style, error) {
	rows, err := e.DB.Query(`SELECT id, name, value, prompt, description, sort_order, is_active FROM style_presets WHERE is_active = 1 ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Style
	for rows.Next() {
		var s Style
		var active int
		if err := rows.Scan(&s.ID, &s.Name, &s.Value, &s.Prompt, &s.Description, &s.SortOrder, &active); err != nil {
			return nil, err
		}
		s.IsActive = active != 0
		out = append(out, s)
	}
	if out == nil {
		out = []Style{}
	}
	return out, nil
}

func (e *Engine) ListAllStyles() ([]Style, error) {
	rows, err := e.DB.Query(`SELECT id, name, value, prompt, description, sort_order, is_active FROM style_presets ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Style
	for rows.Next() {
		var s Style
		var active int
		if err := rows.Scan(&s.ID, &s.Name, &s.Value, &s.Prompt, &s.Description, &s.SortOrder, &active); err != nil {
			return nil, err
		}
		s.IsActive = active != 0
		out = append(out, s)
	}
	if out == nil {
		out = []Style{}
	}
	return out, nil
}

func (e *Engine) StyleByValue(value string) (Style, error) {
	var s Style
	var active int
	err := e.DB.QueryRow(`SELECT id, name, value, prompt, description, sort_order, is_active FROM style_presets WHERE value = ?`, value).
		Scan(&s.ID, &s.Name, &s.Value, &s.Prompt, &s.Description, &s.SortOrder, &active)
	s.IsActive = active != 0
	return s, err
}

func (e *Engine) PrefixStyle(value, prompt string) string {
	s, err := e.StyleByValue(value)
	if err != nil || s.Prompt == "" {
		return prompt
	}
	if prompt == "" {
		return s.Prompt
	}
	return s.Prompt + ". " + prompt
}

func (e *Engine) UpsertStyle(s Style) (Style, error) {
	now := Now()
	if s.Value == "" {
		return s, fmt.Errorf("style value required")
	}
	if s.ID == "" {
		var existing string
		if e.DB.QueryRow(`SELECT id FROM style_presets WHERE value = ?`, s.Value).Scan(&existing) == nil {
			s.ID = existing
		} else {
			s.ID = NewID()
		}
	}
	active := 1
	if !s.IsActive {
		active = 0
	}
	_, err := e.DB.Exec(`INSERT INTO style_presets(id, name, value, prompt, description, sort_order, is_active, seeded, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,0,?,?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, value=excluded.value, prompt=excluded.prompt, description=excluded.description, sort_order=excluded.sort_order, is_active=excluded.is_active, updated_at=excluded.updated_at`,
		s.ID, s.Name, s.Value, s.Prompt, s.Description, s.SortOrder, active, now, now)
	return s, err
}

func (e *Engine) DeleteStyle(id string) error {
	_, err := e.DB.Exec(`UPDATE style_presets SET is_active = 0, updated_at = ? WHERE id = ?`, Now(), id)
	return err
}
