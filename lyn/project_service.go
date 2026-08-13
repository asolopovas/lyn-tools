package lyn

import (
	"context"
	"strings"
	"sync"
	"time"

	"lyn.tools/launcher/lyn/launch"
)

type projectService struct {
	mu              sync.Mutex
	scanMu          sync.Mutex
	selectionMu     sync.Mutex
	ctx             context.Context
	config          Config
	store           *Store
	closed          bool
	searchIndex     []searchProject
	launchSelection launch.Request
	debugLog        func(string, ...any)
	scanProjects    func(context.Context, ScannerConfig) ([]Project, []string, error)
	scanApps        func(context.Context) ([]Project, error)
	scanRecents     func(context.Context) ([]Project, error)
}

func newProjectService(config Config, debugLog func(string, ...any)) *projectService {
	return &projectService{
		config:       config,
		debugLog:     debugLog,
		scanProjects: ScanProjects,
		scanApps:     ScanApplications,
		scanRecents:  ScanVSCodeRecentProjects,
	}
}

func (s *projectService) configure(ctx context.Context, config Config) {
	s.mu.Lock()
	s.ctx = ctx
	s.config = config
	s.closed = false
	s.mu.Unlock()
}

func (s *projectService) useConfig(config Config) {
	s.mu.Lock()
	s.config = config
	s.mu.Unlock()
}

func (s *projectService) setStore(store *Store) {
	s.mu.Lock()
	s.store = store
	s.mu.Unlock()
}

func (s *projectService) snapshot() (context.Context, Config, *Store) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx, s.config, s.store
}

func (s *projectService) close() error {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	s.mu.Lock()
	store := s.store
	s.store = nil
	s.ctx = nil
	s.closed = true
	s.mu.Unlock()
	if store == nil {
		return nil
	}
	return store.Close()
}

func (s *projectService) currentSearchIndex() []searchProject {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.searchIndex
}

func (s *projectService) setSearchIndex(projects []Project) {
	index := newSearchIndex(projects)
	s.mu.Lock()
	s.searchIndex = index
	s.mu.Unlock()
}

func (s *projectService) indexProjects(cacheable, recent []Project) []Project {
	items := mergeProjects(cacheable, recent, systemCommands())
	s.setSearchIndex(items)
	return items
}

func (s *projectService) updateSearchIndexLaunch(path string) {
	index := s.currentSearchIndex()
	if len(index) == 0 {
		return
	}
	now := time.Now().UTC()
	projects := make([]Project, 0, len(index))
	for _, item := range index {
		project := item.project
		if project.Path == path {
			project.UsageCount++
			project.LastLaunchedAt = now
		}
		projects = append(projects, project)
	}
	s.setSearchIndex(projects)
}

func (s *projectService) projects() ([]Project, error) {
	ctx, _, store := s.snapshot()
	if store != nil {
		projects, err := store.ListProjects(ctx)
		if err != nil {
			return nil, err
		}
		if len(projects) > 0 {
			return s.indexProjects(projects, s.liveRecents(ctx)), nil
		}
	}
	items, sourceError, err := s.rescan()
	if err != nil {
		return items, err
	}
	return items, sourceError
}

func (s *projectService) search(query string) ([]Project, error) {
	index := s.currentSearchIndex()
	if len(index) == 0 {
		projects, err := s.projects()
		if err != nil {
			return projects, err
		}
		index = s.currentSearchIndex()
	}
	_, config, _ := s.snapshot()
	return searchProjectIndex(index, query, config.UI.WorkspaceQueryShortcut), nil
}

func (s *projectService) refresh() ([]Project, error) {
	ctx, _, store := s.snapshot()
	apps, applicationError := s.scanApps(ctx)
	recent, recentProjectError := s.scanRecents(ctx)
	sourceError := firstError(applicationError, recentProjectError)
	if store == nil {
		return s.indexProjects(apps, recent), sourceError
	}
	if err := updateProjectKind(ctx, store, apps, projectKindApp, applicationError); err != nil {
		return nil, err
	}
	projects, err := store.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	return s.indexProjects(mergeProjects(projects, apps), recent), sourceError
}

func (s *projectService) scan() ScanResult {
	items, sourceError, err := s.rescan()
	if err != nil {
		return ScanResult{Count: len(items), Error: err.Error()}
	}
	if sourceError != nil {
		return ScanResult{Count: len(items), Error: sourceError.Error()}
	}
	return ScanResult{Count: len(items)}
}

func (s *projectService) liveRecents(ctx context.Context) []Project {
	recent, err := s.scanRecents(ctx)
	if err != nil {
		s.debugLog("vscode-recent.live.error", "error", err)
		return nil
	}
	return recent
}

func (s *projectService) rescan() ([]Project, error, error) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return nil, nil, context.Canceled
	}
	ctx, config, store := s.snapshot()
	s.debugLog("scan.begin", "roots", len(config.Scanner.Roots), "maxDepth", config.Scanner.MaxDepth)
	items, skipped, err := s.scanProjects(ctx, config.Scanner)
	for _, root := range skipped {
		s.debugLog("scan.root.skip", "root", root)
	}
	apps, applicationError := s.scanApps(ctx)
	recent, recentProjectError := s.scanRecents(ctx)
	cacheItems := mergeProjects(items, apps)
	if store != nil {
		var storeError error
		if shouldReplaceCache(err, applicationError, skipped) {
			storeError = store.ReplaceProjects(ctx, cacheItems)
		} else {
			storeError = store.UpsertProjects(ctx, cacheItems)
		}
		if storeError != nil && err == nil {
			err = storeError
		}
	}
	sourceError := firstError(applicationError, recentProjectError)
	items = s.indexProjects(cacheItems, recent)
	s.debugLog("scan.end", "items", len(items), "skipped", len(skipped), "error", err, "sourceError", sourceError)
	return items, sourceError, err
}

func (s *projectService) launch(request launch.Request) launch.Result {
	s.debugLog("launch.begin", "action", launch.NormalizedAction(request.Action), "path", request.Path)
	target := s.resolveLaunchTarget(request)
	if target.Path != request.Path {
		s.debugLog("launch.retarget", "from", request.Path, "to", target.Path)
	}
	result := launchRequest(target)
	s.debugLog("launch.end", "command", result.Command, "args", strings.Join(result.Args, " "), "error", result.Error)
	if result.Error == "" && request.Action != "reveal" {
		launchAsync(func() { s.recordLaunch(request.Path) })
	}
	return result
}

func (s *projectService) recordLaunch(path string) {
	ctx, _, store := s.snapshot()
	if store == nil {
		return
	}
	logRuntimeError(ctx, store.RecordLaunch(ctx, path))
	s.updateSearchIndexLaunch(path)
}

func (s *projectService) setLaunchSelection(request launch.Request) {
	request.Path = strings.TrimSpace(request.Path)
	request.Action = launch.NormalizedAction(request.Action)
	s.selectionMu.Lock()
	s.launchSelection = request
	s.selectionMu.Unlock()
	s.debugLog("launch.selection", "action", request.Action, "path", request.Path)
}

func (s *projectService) hasCachedLaunchSelection() bool {
	s.selectionMu.Lock()
	path := s.launchSelection.Path
	s.selectionMu.Unlock()
	return launchablePath(path)
}

func (s *projectService) currentLaunchSelection() launch.Request {
	s.selectionMu.Lock()
	request := s.launchSelection
	s.selectionMu.Unlock()
	request.Action = launch.NormalizedAction(request.Action)
	if strings.TrimSpace(request.Path) != "" {
		if isSystemCommandPath(request.Path) {
			return launch.Request{}
		}
		return request
	}
	ctx, _, store := s.snapshot()
	if store == nil {
		return request
	}
	projects, err := store.ListProjects(ctx)
	if err != nil || len(projects) == 0 {
		return request
	}
	project := Project{}
	for _, candidate := range projects {
		if candidate.Kind != projectKindSystemCommand && !isSystemCommandPath(candidate.Path) {
			project = candidate
			break
		}
	}
	if project.Path == "" {
		return request
	}
	action := "code"
	if project.Kind == projectKindApp {
		action = "open"
	}
	s.debugLog("launch.native.fallback", "action", action, "path", project.Path, "error", err)
	return launch.Request{Path: project.Path, Action: action}
}

func updateProjectKind(ctx context.Context, store *Store, projects []Project, kind string, sourceError error) error {
	if sourceError == nil {
		return store.ReplaceProjectKinds(ctx, projects, kind)
	}
	if len(projects) > 0 {
		return store.UpsertProjects(ctx, projects)
	}
	return nil
}

func shouldReplaceCache(scanError, applicationError error, skippedRoots []string) bool {
	return scanError == nil && applicationError == nil && len(skippedRoots) == 0
}

func firstError(errorsToCheck ...error) error {
	for _, err := range errorsToCheck {
		if err != nil {
			return err
		}
	}
	return nil
}
