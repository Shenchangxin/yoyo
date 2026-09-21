package video

func (e *Engine) seedStyles() error {
	now := Now()
	seeds := []Style{
		{Value: "3d", Name: "3D cinematic", Prompt: "cinematic 3D animation, physically based materials, soft volumetric light, filmic contrast, coherent character design, no photoreal human skin", Description: "Stylized 3D, not live-action."},
		{Value: "anime", Name: "Anime", Prompt: "high-end anime still, clean linework, cel shading, expressive eyes, consistent costume color, cinematic lighting, not photoreal", Description: "2D anime look."},
		{Value: "ink", Name: "Ink wash", Prompt: "ink-wash illustration, rice-paper grain, restrained palette, calligraphic line, atmospheric haze, not photoreal", Description: "East-Asian ink."},
		{Value: "comic", Name: "Graphic novel", Prompt: "graphic-novel panel, bold ink, limited flat color, dramatic shadow, consistent character model sheets, not photoreal", Description: "Print comic."},
		{Value: "clay", Name: "Stop-motion clay", Prompt: "stop-motion clay miniature, visible fingerprints, practical set lighting, tactile materials, not photoreal humans", Description: "Claymation."},
		{Value: "painterly", Name: "Painterly", Prompt: "oil-paint cinematic frame, visible brush, rich midtones, controlled palette, character likeness held across shots, not photoreal photo", Description: "Painted film."},
		{Value: "watercolor", Name: "Watercolor", Prompt: "watercolor storyboard, paper texture, bleeding pigment, soft edges, consistent costume hues, not photoreal", Description: "Paper wash."},
		{Value: "cyber", Name: "Neon noir", Prompt: "neon-noir 3D, wet streets, cyan-magenta practical lights, graphic silhouettes, stylized not photoreal faces", Description: "Night city."},
	}
	for i, s := range seeds {
		_, err := e.DB.Exec(`INSERT OR IGNORE INTO style_presets(id, name, value, prompt, description, sort_order, is_active, seeded, created_at, updated_at)
			VALUES(?,?,?,?,?,?,1,1,?,?)`, NewID(), s.Name, s.Value, s.Prompt, s.Description, i, now, now)
		if err != nil {
			return err
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
	if s.ID == "" {
		s.ID = NewID()
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
