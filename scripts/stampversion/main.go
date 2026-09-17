package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 || strings.TrimSpace(os.Args[1]) == "" {
		fmt.Fprintln(os.Stderr, "usage: stampversion <version>")
		os.Exit(2)
	}
	v := strings.TrimPrefix(strings.TrimSpace(os.Args[1]), "v")
	base := strings.SplitN(v, "-", 2)[0]
	asm := fourPart(base)

	if err := stampWindowsInfo("build/windows/info.json", v, base); err != nil {
		fatal(err)
	}
	if err := stampManifest("build/windows/wails.exe.manifest", asm); err != nil {
		fatal(err)
	}
	if err := stampPlist("build/darwin/Info.plist", base); err != nil {
		fatal(err)
	}
	if err := stampNFPM("build/linux/nfpm/nfpm.yaml", v); err != nil {
		fatal(err)
	}
	if err := stampNSIS("build/windows/nsis/wails_tools.nsh", base); err != nil {
		fatal(err)
	}
	fmt.Printf("stamped version %s (bundle %s, assembly %s)\n", v, base, asm)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func fourPart(v string) string {
	p := strings.Split(v, ".")
	for len(p) < 4 {
		p = append(p, "0")
	}
	return strings.Join(p[:4], ".")
}

func stampWindowsInfo(path, product, fileVer string) error {
	type infoJSON struct {
		Fixed map[string]string            `json:"fixed"`
		Info  map[string]map[string]string `json:"info"`
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc infoJSON
	if err := json.Unmarshal(b, &doc); err != nil {
		return err
	}
	if doc.Fixed == nil {
		doc.Fixed = map[string]string{}
	}
	doc.Fixed["file_version"] = fileVer
	if doc.Info == nil {
		doc.Info = map[string]map[string]string{}
	}
	block := doc.Info["0000"]
	if block == nil {
		block = map[string]string{}
	}
	block["ProductVersion"] = product
	doc.Info["0000"] = block
	out, err := json.MarshalIndent(doc, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

func stampManifest(path, asm string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	re := regexp.MustCompile(`(<assemblyIdentity\b[^>]*\bversion=")[^"]+(")`)
	out := re.ReplaceAll(b, []byte(`${1}`+asm+`${2}`))
	if string(out) == string(b) {
		return fmt.Errorf("%s: assemblyIdentity version not found", path)
	}
	return os.WriteFile(path, out, 0o644)
}

func stampPlist(path, ver string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	keys := []string{"CFBundleShortVersionString", "CFBundleVersion"}
	out := string(b)
	for _, key := range keys {
		re := regexp.MustCompile(`(?s)(<key>` + regexp.QuoteMeta(key) + `</key>\s*<string>)[^<]+(</string>)`)
		next := re.ReplaceAllString(out, `${1}`+ver+`${2}`)
		if next == out {
			return fmt.Errorf("%s: %s not found", path, key)
		}
		out = next
	}
	return os.WriteFile(path, []byte(out), 0o644)
}

func stampNFPM(path, ver string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	re := regexp.MustCompile(`(?m)^version:\s*".*"`)
	out := re.ReplaceAll(b, []byte(`version: "`+ver+`"`))
	if string(out) == string(b) {
		return fmt.Errorf("%s: version field not found", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

func stampNSIS(path, ver string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	re := regexp.MustCompile(`(!define INFO_PRODUCTVERSION\s+")[^"]+(")`)
	out := re.ReplaceAll(b, []byte(`${1}`+ver+`${2}`))
	if string(out) == string(b) {
		return fmt.Errorf("%s: INFO_PRODUCTVERSION not found", path)
	}
	return os.WriteFile(path, out, 0o644)
}
