package lyn

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreRecordsLaunchUsage(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "lyn.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	projects := []Project{
		{Name: "Alpha", Path: "/tmp/alpha", Kind: "go", DetectedAt: time.Now().UTC()},
		{Name: "Beta", Path: "/tmp/beta", Kind: "go", DetectedAt: time.Now().UTC()},
	}
	if err := store.UpsertProjects(ctx, projects); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordLaunch(ctx, "/tmp/beta"); err != nil {
		t.Fatal(err)
	}

	items, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected item count %d", len(items))
	}
	if items[0].Path != "/tmp/beta" || items[0].UsageCount != 1 || items[0].LastLaunchedAt.IsZero() {
		t.Fatalf("unexpected ranked item %#v", items[0])
	}
}

func TestStorePurgesDisabledSystemCommandsOnOpen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lyn.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, "INSERT INTO projects(path, name, kind, detected_at, updated_at) VALUES(?, ?, ?, ?, ?)", "lyn:system:logout", "Log Out", projectKindSystemCommand, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	items, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestStorePurgesVolatileVSCodeRecentsOnOpen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lyn.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	stale := []Project{
		{Name: "Old Recent", Path: "vscode-remote://ssh-remote+old/home/old", Kind: projectKindVSCodeRecent, DetectedAt: time.Now().UTC()},
		{Name: "Old Workspace", Path: "vscode-remote://ssh-remote+old/home/old.code-workspace", Kind: projectKindVSCodeWorkspace, DetectedAt: time.Now().UTC()},
		{Name: "Keep", Path: "/tmp/keep", Kind: projectKindGo, DetectedAt: time.Now().UTC()},
	}
	if err := store.UpsertProjects(ctx, stale); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	items, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Path != "/tmp/keep" {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestReplaceProjectsRemovesStaleItems(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "lyn.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	initial := []Project{
		{Name: "Keep", Path: "/tmp/keep", Kind: "go", DetectedAt: time.Now().UTC()},
		{Name: "Remove", Path: "/tmp/remove", Kind: "app", DetectedAt: time.Now().UTC()},
	}
	if err := store.UpsertProjects(ctx, initial); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceProjects(ctx, initial[:1]); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Path != "/tmp/keep" {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestReplaceProjectKindsOnlyRemovesStaleOwnedKinds(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "lyn.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	initial := []Project{
		{Name: "Keep Recent", Path: "/tmp/keep-recent", Kind: projectKindVSCodeRecent, DetectedAt: time.Now().UTC()},
		{Name: "Remove Recent", Path: "/tmp/remove-recent", Kind: projectKindVSCodeRecent, DetectedAt: time.Now().UTC()},
		{Name: "Keep App", Path: "/tmp/keep-app", Kind: projectKindApp, DetectedAt: time.Now().UTC()},
		{Name: "Keep Go", Path: "/tmp/keep-go", Kind: projectKindGo, DetectedAt: time.Now().UTC()},
	}
	if err := store.UpsertProjects(ctx, initial); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceProjectKinds(ctx, initial[:1], projectKindVSCodeRecent); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	for _, item := range items {
		paths[item.Path] = true
	}
	if !paths["/tmp/keep-recent"] || paths["/tmp/remove-recent"] || !paths["/tmp/keep-app"] || !paths["/tmp/keep-go"] {
		t.Fatalf("unexpected synced items %#v", items)
	}
}

func TestStorePreservesUsageAcrossUpserts(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "lyn.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	project := Project{Name: "Alpha", Path: "/tmp/alpha", Kind: "go", DetectedAt: time.Now().UTC()}
	if err := store.UpsertProjects(ctx, []Project{project}); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordLaunch(ctx, project.Path); err != nil {
		t.Fatal(err)
	}
	project.Name = "Alpha Renamed"
	if err := store.UpsertProjects(ctx, []Project{project}); err != nil {
		t.Fatal(err)
	}

	items, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected item count %d", len(items))
	}
	if items[0].UsageCount != 1 || items[0].Name != "Alpha Renamed" {
		t.Fatalf("usage was not preserved %#v", items[0])
	}
}
