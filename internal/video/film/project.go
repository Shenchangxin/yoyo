// 翻拍工程的文档：简介、词、镜头、待办和产物。
// 其他产品的标识、命令名和托管网关不放在这里。

package film

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// Brief 是以后可以单独换掉的那一部分：宿主、产品、语言、画幅、开头、价格和收口。
type Brief struct {
	Host     string `json:"host,omitempty"`
	Product  string `json:"product,omitempty"`
	Language string `json:"language,omitempty"`
	Aspect   string `json:"aspect,omitempty"`
	Hook     string `json:"hook,omitempty"`
	Price    string `json:"price,omitempty"`
	CTA      string `json:"cta,omitempty"`
}

// Word 是词轴上的一个词。同一行的词会被一次改写一起换掉。
type Word struct {
	ID      string  `json:"id"`
	LineID  string  `json:"line_id"`
	Text    string  `json:"text"`
	Speaker string  `json:"speaker,omitempty"`
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
}

// Box 是画面里的一块区域，坐标和宽高都是 0 到 1。
type Box struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
	Label string  `json:"label,omitempty"`
}

// Shot 是一段画面，时间锚在词上。
type Shot struct {
	ID      string   `json:"id"`
	LineID  string   `json:"line_id,omitempty"`
	Visual  string   `json:"visual"`
	WordIDs []string `json:"word_ids,omitempty"`
	Start   float64  `json:"start"`
	End     float64  `json:"end"`
	Region  *Box     `json:"region,omitempty"`
	// Beat 为真表示这一镜的起点落在词上。
	Beat bool `json:"beat,omitempty"`
}

// Entry 是排行榜的一行，或揭示里的一档。
type Entry struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Sticker 是钉在某一个词上的评论。
type Sticker struct {
	ID     string `json:"id"`
	WordID string `json:"word_id"`
	Text   string `json:"text"`
}

// Need 是构建时的一步。InputHash 盖住会改变产物字节的字段。没变时 AssetHash 会留下。
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

// Artifact 是构建留下的一份结果。
type Artifact struct {
	Name      string `json:"name"`
	Hash      string `json:"hash,omitempty"`
	MediaType string `json:"media_type,omitempty"`
}

// Task 是最近一次构建里某一步的证据。失败的一步不会抹掉其余产物。
type Task struct {
	ID     string `json:"id"`
	NeedID string `json:"need_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Reference 是操作者有权使用的参考片。
type Reference struct {
	Hash     string  `json:"hash,omitempty"`
	URL      string  `json:"url,omitempty"`
	Duration float64 `json:"duration,omitempty"`
}

// Project 是一部可再跑的翻拍。
type Project struct {
	ID        string    `json:"id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Title     string    `json:"title"`
	Structure string    `json:"structure"`
	Brief     Brief     `json:"brief"`
	Reference Reference `json:"reference"`
	// Bed 是已经存好的垫乐。空着才会再计划配乐。
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

// Cue 是读参考片时的一句带时间的话，还没拆成词。
type Cue struct {
	Start   float64
	End     float64
	Text    string
	Speaker string
}

// Input 是工程还没建起来时，对参考片或一段描述已知的全部材料。
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

// digest 把值编成 JSON 再取 SHA-256 的前 12 个字节，当成输入摘要。编码失败时用错误文本。
// v 先 Marshal，失败时改用错误文本，再取 SHA-256 的前 12 个字节。
func digest(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		b = []byte(err.Error())
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:12])
}

// clean 去掉字符串两端的空白。
// s 传给 TrimSpace。
func clean(s string) string { return strings.TrimSpace(s) }
