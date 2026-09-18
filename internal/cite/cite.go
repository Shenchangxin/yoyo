package cite

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Source struct {
	URL     string `json:"url"`
	Excerpt string `json:"excerpt"`
	Hash    string `json:"hash"`
}

func HashExcerpt(url, excerpt string) Source {
	sum := sha256.Sum256([]byte(url + "\n" + excerpt))
	return Source{URL: url, Excerpt: excerpt, Hash: hex.EncodeToString(sum[:8])}
}

func Write(path string, srcs []Source) error {
	b, err := json.MarshalIndent(srcs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func Validate(report, citesPath string) error {
	b, err := os.ReadFile(citesPath)
	if err != nil {
		return err
	}
	var srcs []Source
	if err := json.Unmarshal(b, &srcs); err != nil {
		return err
	}
	if len(srcs) == 0 {
		return fmt.Errorf("cite: empty sources")
	}
	low := strings.ToLower(report)
	for _, s := range srcs {
		if s.URL == "" || !strings.HasPrefix(s.URL, "http") {
			return fmt.Errorf("cite: bad url")
		}
		if s.Hash == "" {
			return fmt.Errorf("cite: missing hash for %s", s.URL)
		}
		if s.Excerpt != "" && !strings.Contains(low, strings.ToLower(s.Excerpt[:min(40, len(s.Excerpt))])) && !strings.Contains(low, s.URL) {
			return fmt.Errorf("cite: report does not mention %s", s.URL)
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
