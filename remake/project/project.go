// Copyright 2026 Hypit.AI
// Translated from @hypit/project (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

// Package project finds a directory whose package.json sets hypit.project to true.
package project

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// ErrNoProject is returned when no marked project exists above the start directory.
const noProjectMessage = "No Hypit project was found; run inside a package.json with hypit.project set to true, or pass --project <directory>"

// FindProjectRoot walks upward from cwd for a package.json with hypit.project set to true.
// The returned path is the real path of that directory. An empty string means no project exists.
// Missing package.json files are skipped. Any other filesystem or JSON error is returned.
func FindProjectRoot(cwd string) (string, error) {
	selected, err := nearestProjectPackageRoot(cwd)
	if err != nil || selected == "" {
		return "", err
	}
	return filepath.EvalSymlinks(selected)
}

// ResolveProjectRoot uses projectRoot when it is set, otherwise FindProjectRoot.
// projectRoot is resolved against cwd before the real path is taken.
func ResolveProjectRoot(cwd, projectRoot string, hasProjectRoot bool) (string, error) {
	if !hasProjectRoot {
		selected, err := FindProjectRoot(cwd)
		if err != nil {
			return "", err
		}
		if selected == "" {
			return "", errors.New(noProjectMessage)
		}
		return selected, nil
	}
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	selected := projectRoot
	if !filepath.IsAbs(selected) {
		selected = filepath.Join(cwd, selected)
	}
	return filepath.EvalSymlinks(selected)
}

func nearestProjectPackageRoot(start string) (string, error) {
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	directory, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(directory, "package.json")
		info, err := os.Stat(candidate)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", err
			}
		} else if info.Mode().IsRegular() {
			manifest, err := os.ReadFile(candidate)
			if err != nil {
				return "", err
			}
			var value any
			if err := json.Unmarshal(manifest, &value); err != nil {
				return "", err
			}
			if isProjectManifest(value) {
				return directory, nil
			}
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", nil
		}
		directory = parent
	}
}

func isProjectManifest(value any) bool {
	obj, ok := value.(map[string]any)
	if !ok {
		return false
	}
	hypit, ok := obj["hypit"].(map[string]any)
	if !ok {
		return false
	}
	project, ok := hypit["project"].(bool)
	return ok && project
}
