package lyn

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"lyn.tools/launcher/lyn/launch"
)

func newTestProjectService(ctx context.Context, config Config) *projectService {
	service := newProjectService(config, func(string, ...any) {})
	service.configure(ctx, config)
	service.scanApps = func(context.Context) ([]Project, error) { return nil, nil }
	service.scanRecents = func(context.Context) ([]Project, error) { return nil, nil }
	return service
}

func TestProjectServicePartialScanPreservesCachedProjects(t *testing.T) {
	ctx := t.Context()
	service := newTestProjectService(ctx, DefaultConfig())
	store, err := OpenStore(filepath.Join(t.TempDir(), "lyn.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.close() })
	cached := newProject("cached", filepath.Join(t.TempDir(), "cached"), projectKindGo)
	if err := store.UpsertProjects(ctx, []Project{cached}); err != nil {
		t.Fatal(err)
	}
	service.setStore(store)
	discovered := newProject("discovered", filepath.Join(t.TempDir(), "discovered"), projectKindGo)
	service.scanProjects = func(context.Context, ScannerConfig) ([]Project, []string, error) {
		return []Project{discovered}, []string{"unavailable"}, errors.New("partial scan")
	}

	items, _, scanError := service.rescan()
	if scanError == nil {
		t.Fatal("expected partial scan error")
	}
	items = withoutSystemCommands(items)
	if len(items) != 1 || items[0].Path != discovered.Path {
		t.Fatalf("unexpected live scan result %#v", items)
	}
	stored, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 2 {
		t.Fatalf("partial scan replaced cached data: %#v", stored)
	}
}

func TestProjectServiceRefreshKeepsSuccessfulAppsFromPartialSource(t *testing.T) {
	ctx := t.Context()
	service := newTestProjectService(ctx, DefaultConfig())
	store, err := OpenStore(filepath.Join(t.TempDir(), "lyn.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.close() })
	service.setStore(store)
	app := newProject("example", filepath.Join(t.TempDir(), "example.exe"), projectKindApp)
	service.scanApps = func(context.Context) ([]Project, error) {
		return []Project{app}, errors.New("one application source failed")
	}

	items, err := service.refresh()
	if err == nil || err.Error() != "one application source failed" {
		t.Fatalf("unexpected source error %v", err)
	}
	items = withoutSystemCommands(items)
	if len(items) != 1 || items[0].Path != app.Path {
		t.Fatalf("successful application result was lost: %#v", items)
	}
	stored, listErr := store.ListProjects(ctx)
	if listErr != nil || len(stored) != 1 || stored[0].Path != app.Path {
		t.Fatalf("successful application was not cached: %#v, %v", stored, listErr)
	}
}

func TestProjectServiceLaunchRefreshesRankingState(t *testing.T) {
	originalRequest := launchRequest
	originalAsync := launchAsync
	launchRequest = func(request launch.Request) launch.Result {
		return launch.Result{Command: "example", Args: []string{request.Path}}
	}
	launchAsync = func(fn func()) { fn() }
	t.Cleanup(func() {
		launchRequest = originalRequest
		launchAsync = originalAsync
	})
	ctx := t.Context()
	service := newTestProjectService(ctx, DefaultConfig())
	store, err := OpenStore(filepath.Join(t.TempDir(), "lyn.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.close() })
	project := newProject("example", filepath.Join(t.TempDir(), "example"), projectKindGo)
	if err := store.UpsertProjects(ctx, []Project{project}); err != nil {
		t.Fatal(err)
	}
	service.setStore(store)
	service.setSearchIndex([]Project{project})

	service.launch(launch.Request{Path: project.Path, Action: "code"})

	indexed, ok := service.findIndexedProject(project.Path)
	if !ok || indexed.UsageCount != 1 || indexed.LastLaunchedAt.IsZero() {
		t.Fatalf("launch did not refresh ranking state: %#v", indexed)
	}
}

func TestProjectServiceCloseReleasesStoreAndIsIdempotent(t *testing.T) {
	service := newTestProjectService(t.Context(), DefaultConfig())
	store, err := OpenStore(filepath.Join(t.TempDir(), "lyn.db"))
	if err != nil {
		t.Fatal(err)
	}
	service.setStore(store)
	if err := service.close(); err != nil {
		t.Fatal(err)
	}
	ctx, _, current := service.snapshot()
	if ctx != nil || current != nil {
		t.Fatalf("service retained shutdown state: ctx=%v store=%v", ctx, current)
	}
	if err := service.close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}

func TestProjectServiceSerializesScans(t *testing.T) {
	service := newTestProjectService(t.Context(), DefaultConfig())
	entered := make(chan struct{}, 2)
	release := make(chan struct{}, 2)
	service.scanProjects = func(context.Context, ScannerConfig) ([]Project, []string, error) {
		entered <- struct{}{}
		<-release
		return nil, nil, nil
	}
	done := make(chan struct{}, 2)
	run := func() {
		_, _, _ = service.rescan()
		done <- struct{}{}
	}
	go run()
	<-entered
	go run()
	select {
	case <-entered:
		t.Fatal("second scan entered before the first completed")
	case <-time.After(100 * time.Millisecond):
	}
	release <- struct{}{}
	<-entered
	release <- struct{}{}
	<-done
	<-done
}
