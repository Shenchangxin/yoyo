package home

import (
	"os"
	"path/filepath"
)

const (
	EnvHome = "YOYO_HOME"
	DirName = ".yoyo"
)

// Dir is the on-disk layout of a Yoyo home.
type Dir struct {
	Root string
}

func Open(root string) (*Dir, error) {
	if root == "" {
		var err error
		root, err = Default()
		if err != nil {
			return nil, err
		}
	}
	d := &Dir{Root: root}
	for _, p := range []string{
		d.CAS(),
		d.Refs(),
		d.Sessions(),
		d.Journal(),
		d.EvalRuns(),
		d.Archive(),
		d.Plugins(),
		d.Tmp(),
		d.Updates(),
		d.Skills(),
		d.Workspace(),
	} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func Default() (string, error) {
	if v := os.Getenv(EnvHome); v != "" {
		return filepath.Abs(v)
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, DirName), nil
}

func (d *Dir) CAS() string      { return filepath.Join(d.Root, "cas", "objects") }
func (d *Dir) Refs() string     { return filepath.Join(d.Root, "refs") }
func (d *Dir) Sessions() string { return filepath.Join(d.Root, "sessions") }
func (d *Dir) Journal() string  { return filepath.Join(d.Root, "journal") }
func (d *Dir) EvalRuns() string { return filepath.Join(d.Root, "evals", "runs") }
func (d *Dir) Archive() string  { return filepath.Join(d.Root, "archive") }
func (d *Dir) Plugins() string  { return filepath.Join(d.Root, "plugins") }
func (d *Dir) Tmp() string      { return filepath.Join(d.Root, "tmp") }
func (d *Dir) Updates() string  { return filepath.Join(d.Root, "updates") }
func (d *Dir) Skills() string  { return filepath.Join(d.Root, "skills") }
func (d *Dir) Workspace() string {
	return filepath.Join(d.Root, "workspace")
}
func (d *Dir) Config() string { return filepath.Join(d.Root, "config.yaml") }
func (d *Dir) ModelsRefs() string {
	return filepath.Join(d.Refs(), "models")
}
