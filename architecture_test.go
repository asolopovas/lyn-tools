package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestNoSingleFileNonDocumentationFolders(t *testing.T) {
	allowed := map[string]bool{
		".git":                            true,
		"build":                           true,
		filepath.Join("build", "bin"):     true,
		"frontend":                        true,
		filepath.Join("frontend", "src"):  true,
		filepath.Join("frontend", "dist"): true,
		filepath.Join("frontend", "node_modules"): true,
	}
	if err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() || path == "." {
			return nil
		}
		clean := strings.TrimPrefix(filepath.Clean(path), "."+string(filepath.Separator))
		base := filepath.Base(clean)
		if base == ".git" || base == ".github" || base == ".claude" || base == "graphify-out" || base == "node_modules" || clean == filepath.Join("frontend", "wailsjs") || strings.HasPrefix(clean, "docs") {
			return filepath.SkipDir
		}
		if allowed[clean] {
			return nil
		}
		count := 0
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, item := range entries {
			if !item.IsDir() {
				count++
			}
		}
		if count == 1 {
			t.Fatalf("single-file non-documentation folder %s", clean)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDomainPackagesDoNotImportParentOrSiblings(t *testing.T) {
	const parent = "lyn.tools/launcher/lyn"
	domains := []string{"hotkey", "launch", "startup", "tray"}
	for _, domain := range domains {
		dir := filepath.Join("lyn", domain)
		own := parent + "/" + domain
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, imp := range file.Imports {
				value := strings.Trim(imp.Path.Value, `"`)
				if value == parent {
					t.Fatalf("%s imports parent package %s: domain packages must not depend on the orchestrator", path, value)
				}
				if strings.HasPrefix(value, parent+"/") && value != own {
					t.Fatalf("%s imports sibling domain %s: domain packages must not depend on each other", path, value)
				}
			}
		}
	}
}

func TestDocLinksResolve(t *testing.T) {
	linkPattern := regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	skipDir := map[string]bool{".git": true, "node_modules": true, "build": true, "graphify-out": true, ".claude": true}
	if err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skipDir[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range linkPattern.FindAllStringSubmatch(string(data), -1) {
			target := strings.TrimSpace(match[1])
			if target == "" || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") || strings.Contains(target, "://") {
				continue
			}
			if cut := strings.IndexAny(target, "#?"); cut >= 0 {
				target = target[:cut]
			}
			if target == "" {
				continue
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(path), target)); err != nil {
				t.Errorf("%s links to %s, which does not resolve", path, target)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
