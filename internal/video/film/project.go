// Package film is Yoyo's production project for 翻拍.
// It stores a rerunnable film: words, shots, tracks of work, and artifacts.
// Identity strings, command names, and hosted gateways from other products do not belong here.
package film

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// Brief is the part of a film a later variant may replace on its own.
type Brief struct {
	Host     string `json:"host,omitempty"`
	Product  string `json:"product,omitempty"`
	Language string `json:"language,omitempty"`
	Aspect   string `json:"aspect,omitempty"`
	Hook     string `json:"hook,omitempty"`
	Price    string `json:"price,omitempty"`
	CTA      string `json:"cta,omitempty"`
}

// Word is one timed token. Lines group the tokens a rewrite replaces together.
type Word struct {
	ID      string  `json:"id"`
	LineID  string  `json:"line_id"`
	Text    string  `json:"text"`
	Speaker string  `json:"speaker,omitempty"`
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
}

// Box is a region inside the frame, in 0..1 coordinates.
type Box struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
	Label string  `json:"label,omitempty"`
}

// Shot is one picture span anchored to words.
type Shot struct {
	ID      string   `json:"id"`
	LineID  string   `json:"line_id,omitempty"`
	Visual  string   `json:"visual"`
	WordIDs []string `json:"word_ids,omitempty"`
	Start   float64  `json:"start"`
	End     float64  `json:"end"`
	Region  *Box     `json:"region,omitempty"`
	Beat    bool     `json:"beat,omitempty"`
}

// Entry is one row on a board or one stage of a reveal.
type Entry struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Sticker is a comment pinned to one word.
type Sticker struct {
	ID     string `json:"id"`
	WordID string `json:"word_id"`
	Text   string `json:"text"`
}

// Need is one unit of work. InputHash covers the fields that would change its bytes.
type Need struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Kind      string `json:"kind"`
	Prompt    string `json:"prompt,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Model     string `json:"model,omitempty"`
	VaultKey  string `json:"vault_key,omitempty"`
	InputHash string `json:"input_hash,omitempty"`
	AssetHash string `json:"asset_hash,omitempty"`
	Cost      string `json:"cost,omitempty"`
	Status    string `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Artifact is a stored result of a build.
type Artifact struct {
	Name      string `json:"name"`
	Hash      string `json:"hash,omitempty"`
	MediaType string `json:"media_type,omitempty"`
}

// Task is the evidence for one need on the latest build.
type Task struct {
	ID     string `json:"id"`
	NeedID string `json:"need_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Reference is the source the operator has rights to use.
type Reference struct {
	Hash     string  `json:"hash,omitempty"`
	URL      string  `json:"url,omitempty"`
	Duration float64 `json:"duration,omitempty"`
}

// Project is one rerunnable film.
type Project struct {
	ID        string     `json:"id"`
	ParentID  string     `json:"parent_id,omitempty"`
	Title     string     `json:"title"`
	Structure string     `json:"structure"`
	Brief     Brief      `json:"brief"`
	Reference Reference  `json:"reference"`
	Bed       string     `json:"bed,omitempty"`
	Words     []Word     `json:"words,omitempty"`
	Shots     []Shot     `json:"shots,omitempty"`
	Entries   []Entry    `json:"entries,omitempty"`
	Stickers  []Sticker  `json:"stickers,omitempty"`
	Needs     []Need     `json:"needs,omitempty"`
	Artifacts []Artifact `json:"artifacts,omitempty"`
	Tasks     []Task     `json:"tasks,omitempty"`
	Note      string     `json:"note,omitempty"`
}

// Cue is a timed line used while reading a reference.
type Cue struct {
	Start   float64
	End     float64
	Text    string
	Speaker string
}

// Input is everything known about a reference or a description before a project exists.
type Input struct {
	ID          string
	Title       string
	Structure   string
	Description string
	Hash        string
	URL         string
	Duration    float64
	Scenes      []float64
	Cues        []Cue
	Words       []Word
	Regions     []Box
}

func digest(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		b = []byte(err.Error())
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:12])
}

func clean(s string) string { return strings.TrimSpace(s) }
