package lyn

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

func ScanProjects(ctx context.Context, cfg ScannerConfig) ([]Project, []string, error) {
	if timeout := cfg.Timeout.Duration(); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	roots := expandRoots(cfg.Roots)
	limit := max(cfg.Concurrency, 1)
	type job struct {
		path  string
		depth int
	}
	seen := newProjectSet(0)
	var mu sync.Mutex

	queue := struct {
		items   []job
		pending int
		mu      sync.Mutex
		cond    *sync.Cond
	}{items: make([]job, 0, limit*2)}
	queue.cond = sync.NewCond(&queue.mu)

	addJob := func(path string, depth int) bool {
		if ctx.Err() != nil {
			return false
		}
		queue.mu.Lock()
		defer queue.mu.Unlock()
		if ctx.Err() != nil {
			return false
		}
		queue.items = append(queue.items, job{path: path, depth: depth})
		queue.pending++
		queue.cond.Signal()
		return true
	}

	nextJob := func() (job, bool) {
		queue.mu.Lock()
		defer queue.mu.Unlock()
		for len(queue.items) == 0 && queue.pending > 0 && ctx.Err() == nil {
			queue.cond.Wait()
		}
		if len(queue.items) == 0 {
			return job{}, false
		}
		item := queue.items[0]
		queue.items[0] = job{}
		queue.items = queue.items[1:]
		return item, true
	}

	doneJob := func() {
		queue.mu.Lock()
		defer queue.mu.Unlock()
		queue.pending--
		if queue.pending == 0 || ctx.Err() != nil {
			queue.cond.Broadcast()
		}
	}

	var skipped []string
	for _, root := range roots {
		info, err := os.Stat(root)
		if err == nil && info.IsDir() {
			if !addJob(root, 0) {
				break
			}
			continue
		}
		if err == nil {
			if project, ok := DetectProject(root); ok {
				seen.add(project)
			}
			continue
		}
		if ctx.Err() == nil {
			skipped = append(skipped, root)
		}
	}

	var workers sync.WaitGroup
	worker := func() {
		defer workers.Done()
		for {
			item, ok := nextJob()
			if !ok {
				return
			}
			visitPath(ctx, cfg, item.path, item.depth, addJob, seen, &mu)
			doneJob()
		}
	}

	for range limit {
		workers.Add(1)
		go worker()
	}
	workers.Wait()

	items := seen.sorted()
	if ctx.Err() != nil {
		return items, skipped, ctx.Err()
	}
	return items, skipped, nil
}

func visitPath(ctx context.Context, cfg ScannerConfig, path string, depth int, addJob func(string, int) bool, seen projectSet, mu *sync.Mutex) {
	if ctx.Err() != nil {
		return
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}
	if project, ok := detectProjectDirEntries(path, entries); ok {
		mu.Lock()
		seen.add(project)
		mu.Unlock()
		return
	}
	if depth >= cfg.MaxDepth {
		return
	}
	for _, entry := range entries {
		if ctx.Err() != nil {
			return
		}
		child := filepath.Join(path, entry.Name())
		if !entry.IsDir() {
			if project, ok := DetectWorkspaceFile(child); ok {
				mu.Lock()
				seen.add(project)
				mu.Unlock()
			}
			continue
		}
		if shouldSkip(entry.Name()) {
			continue
		}
		if !addJob(child, depth+1) {
			return
		}
	}
}

func DetectProject(path string) (Project, bool) {
	if project, ok := DetectWorkspaceFile(path); ok {
		return project, true
	}
	return DetectProjectDir(path)
}

func DetectWorkspaceFile(path string) (Project, bool) {
	if !isWorkspaceFile(path) {
		return Project{}, false
	}
	return newProject(workspaceName(path), path, projectKindVSCodeWorkspace), true
}

type projectDirCheck struct {
	kind  string
	files []string
}

var projectDirChecks = []projectDirCheck{
	{kind: projectKindWordPress, files: []string{"wp-config.php", "wp-content"}},
	{kind: projectKindLaravel, files: []string{"artisan", "composer.json", "app"}},
	{kind: projectKindGo, files: []string{"go.mod"}},
	{kind: projectKindNode, files: []string{"package.json"}},
	{kind: projectKindRust, files: []string{"Cargo.toml"}},
	{kind: projectKindGit, files: []string{".git"}},
}

func DetectProjectDir(path string) (Project, bool) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return Project{}, false
	}
	return detectProjectDirEntries(path, entries)
}

func detectProjectDirEntries(path string, entries []os.DirEntry) (Project, bool) {
	names := newStringSet(len(entries))
	for _, entry := range entries {
		names.add(entry.Name())
	}
	for _, check := range projectDirChecks {
		matched := true
		for _, file := range check.files {
			if !names.has(file) {
				matched = false
				break
			}
		}
		if matched {
			return newProject(filepath.Base(path), path, check.kind), true
		}
	}
	return Project{}, false
}

func isWorkspaceFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".code-workspace")
}

func workspaceName(path string) string {
	name := filepath.Base(path)
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func expandRoots(roots []string) []string {
	return expandRootsForOS(roots, runtime.GOOS, wslPathToWindows)
}

func expandRootsForOS(roots []string, goos string, convertWsl func(string) (string, bool)) []string {
	home, _ := os.UserHomeDir()
	items := make([]string, 0, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if strings.HasPrefix(root, "~/") && home != "" {
			root = filepath.Join(home, root[2:])
		}
		if goos == "windows" && isUnixPath(root) && !isUNCRoot(root) {
			if converted, ok := convertWsl(root); ok {
				root = converted
			}
		}
		items = append(items, filepath.Clean(root))
	}
	return items
}

func isUNCRoot(path string) bool {
	return strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, "//")
}

func isUnixPath(path string) bool {
	return strings.HasPrefix(filepath.ToSlash(path), "/")
}

func wslPathToWindows(path string) (string, bool) {
	output, err := exec.Command("wsl.exe", "wslpath", "-w", path).Output()
	if err != nil {
		return "", false
	}
	converted := strings.TrimSpace(string(output))
	return converted, converted != ""
}
