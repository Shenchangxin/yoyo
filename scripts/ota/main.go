// Command ota builds signed ModelScope channel JSON from a GitHub Release dist/
// directory (SHA256SUMS.txt + named installers).
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/update"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("ota", flag.ContinueOnError)
	keygen := fs.Bool("keygen", false, "print a new Ed25519 keypair and exit")
	dist := fs.String("dist", "dist", "directory with release artifacts and SHA256SUMS.txt")
	version := fs.String("version", "", "release version (no leading v)")
	notes := fs.String("notes", "", "release notes")
	repo := fs.String("repo", "", "ModelScope dataset id owner/name")
	endpoint := fs.String("endpoint", "", "ModelScope endpoint")
	out := fs.String("out", "ota", "output directory")
	priv := fs.String("privkey", "", "Ed25519 private key hex (default YOYO_UPDATE_PRIVKEY)")
	channels := fs.String("channels", "", "comma-separated channels (default derived from version)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *keygen {
		return genKeys()
	}
	ver := strings.TrimPrefix(strings.TrimSpace(*version), "v")
	if ver == "" {
		return fmt.Errorf("ota: -version is required")
	}
	if *repo != "" {
		_ = os.Setenv(update.EnvRepo, *repo)
	}
	if *endpoint != "" {
		_ = os.Setenv(update.EnvEndpoint, *endpoint)
	}
	key, err := loadPriv(*priv)
	if err != nil {
		return err
	}
	sums, err := readSHA256SUMS(filepath.Join(*dist, "SHA256SUMS.txt"))
	if err != nil {
		return err
	}
	platforms, err := buildPlatforms(ver, *dist, sums)
	if err != nil {
		return err
	}
	chs := parseChannels(*channels, ver)
	if len(chs) == 0 {
		return fmt.Errorf("ota: no channels for %s", ver)
	}
	relDir := filepath.Join(*out, "releases", "v"+ver)
	if err := copyRelease(*dist, relDir); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, ch := range chs {
		m := update.Manifest{
			Schema:    1,
			Version:   ver,
			Channel:   string(ch),
			PubDate:   now,
			Notes:     strings.TrimSpace(*notes),
			Platforms: platforms,
		}
		body, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return err
		}
		body = append(body, '\n')
		sig := update.Sign(key, body)
		chanDir := filepath.Join(*out, "channels")
		if err := os.MkdirAll(chanDir, 0o755); err != nil {
			return err
		}
		base := filepath.Join(chanDir, string(ch)+".json")
		if err := os.WriteFile(base, body, 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(base+".sig", sig, 0o644); err != nil {
			return err
		}
		fmt.Printf("wrote %s (%s %s)\n", base, ch, ver)
	}
	return nil
}

func genKeys() error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	fmt.Printf("public  %s\n", hex.EncodeToString(pub))
	fmt.Printf("private %s\n", hex.EncodeToString(priv))
	return nil
}

func loadPriv(explicit string) (ed25519.PrivateKey, error) {
	raw := strings.TrimSpace(explicit)
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv(update.EnvPrivkey))
	}
	if raw == "" {
		return nil, fmt.Errorf("ota: set YOYO_UPDATE_PRIVKEY or pass -privkey")
	}
	if b, err := os.ReadFile(raw); err == nil {
		raw = string(b)
	}
	return update.ParsePrivateKey(raw)
}

func parseChannels(raw, version string) []update.Channel {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return update.ChannelsForVersion(version)
	}
	var out []update.Channel
	for _, p := range strings.Split(raw, ",") {
		ch := update.NormalizeChannel(p)
		out = append(out, ch)
	}
	return out
}

func readSHA256SUMS(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		sum, name, ok := strings.Cut(line, "  ")
		if !ok {
			sum, name, ok = strings.Cut(line, " *")
		}
		if !ok {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			sum, name = fields[0], fields[len(fields)-1]
		}
		name = filepath.Base(strings.TrimSpace(name))
		out[name] = strings.ToLower(strings.TrimSpace(sum))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("ota: empty SHA256SUMS")
	}
	return out, nil
}

func buildPlatforms(ver, dist string, sums map[string]string) (map[string]update.Asset, error) {
	type spec struct {
		key, file, kind string
	}
	wanted := []spec{
		{"windows-amd64", "Yoyo-" + ver + "-windows-x64-setup.exe", "nsis"},
		{"windows-amd64", "Yoyo-" + ver + "-windows-amd64-setup.exe", "nsis"},
		{"darwin-universal", "Yoyo-" + ver + "-darwin-universal.dmg", "dmg"},
		{"linux-amd64", "Yoyo-" + ver + "-linux-amd64.AppImage", "appimage"},
		{"windows-amd64-cli", "yoyo-cli-" + ver + "-windows-amd64.zip", "zip"},
		{"linux-amd64-cli", "yoyo-cli-" + ver + "-linux-amd64.tar.gz", "tar.gz"},
		{"linux-arm64-cli", "yoyo-cli-" + ver + "-linux-arm64.tar.gz", "tar.gz"},
		{"darwin-amd64-cli", "yoyo-cli-" + ver + "-darwin-amd64.tar.gz", "tar.gz"},
		{"darwin-arm64-cli", "yoyo-cli-" + ver + "-darwin-arm64.tar.gz", "tar.gz"},
	}
	out := map[string]update.Asset{}
	for _, s := range wanted {
		if _, ok := out[s.key]; ok {
			continue
		}
		path := filepath.Join(dist, s.file)
		st, err := os.Stat(path)
		if err != nil {
			continue
		}
		sum := sums[s.file]
		if sum == "" {
			return nil, fmt.Errorf("ota: missing sha256 for %s", s.file)
		}
		out[s.key] = update.Asset{
			URL:      update.ResolveURL("releases/v" + ver + "/" + s.file),
			Filename: s.file,
			SHA256:   sum,
			Size:     st.Size(),
			Kind:     s.kind,
		}
	}
	if _, ok := out["windows-amd64"]; !ok {
		return nil, fmt.Errorf("ota: missing windows installer in %s", dist)
	}
	if _, ok := out["darwin-universal"]; !ok {
		return nil, fmt.Errorf("ota: missing macOS dmg in %s", dist)
	}
	if _, ok := out["linux-amd64"]; !ok {
		return nil, fmt.Errorf("ota: missing linux AppImage in %s", dist)
	}
	return out, nil
}

func copyRelease(dist, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(dist)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(dist, e.Name())
		dst := filepath.Join(dest, e.Name())
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
