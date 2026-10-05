package video

import (
	"fmt"
	"sort"
	"strings"
)

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

func (e *Engine) StyleByValue(value string) (Style, error) {
	for _, s := range loadCol[styleRec](e, colStyles) {
		if s.Value == value {
			return s.Style, nil
		}
	}
	return Style{}, fmt.Errorf("style not found")
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

func (e *Engine) DeleteStyle(id string) error {
	rec, err := getDoc[styleRec](e, colStyles, id)
	if err != nil {
		return err
	}
	rec.IsActive = false
	rec.UpdatedAt = Now()
	return e.putDoc(colStyles, id, rec)
}

func styleValueKey(v string) string { return strings.TrimSpace(v) }
