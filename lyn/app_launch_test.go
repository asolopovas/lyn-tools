package lyn

import (
	"context"
	"path/filepath"
	"testing"

	"lyn.tools/launcher/lyn/launch"
)

func TestAppLaunchRoutesFrontendRequestAndRecordsUsage(t *testing.T) {
	original := launchRequest
	originalAsync := launchAsync
	launchAsync = func(fn func()) { fn() }
	t.Cleanup(func() { launchRequest = original; launchAsync = originalAsync })
	var got launch.Request
	launchRequest = func(request launch.Request) launch.Result {
		got = request
		return launch.Result{Command: "code", Args: []string{request.Path}}
	}
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "lyn.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	request := launch.Request{Path: `C:\src\lyn-tools`, Action: "code"}
	if err := store.UpsertProjects(ctx, []Project{{Name: "lyn-tools", Path: request.Path, Kind: "go"}}); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.projects.configure(ctx, DefaultConfig())
	app.projects.setStore(store)
	result := app.Launch(request)
	if result.Error != "" || result.Command != "code" {
		t.Fatalf("unexpected result %#v", result)
	}
	if got != request {
		t.Fatalf("frontend request was not routed unchanged: %#v", got)
	}
	projects, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].Path != request.Path || projects[0].UsageCount != 1 {
		t.Fatalf("launch usage was not recorded: %#v", projects)
	}
}

func TestAppLaunchDoesNotRecordReveal(t *testing.T) {
	original := launchRequest
	t.Cleanup(func() { launchRequest = original })
	launchRequest = func(request launch.Request) launch.Result {
		return launch.Result{Command: "explorer", Args: []string{request.Path}}
	}
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "lyn.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	app := NewApp()
	app.projects.configure(ctx, DefaultConfig())
	app.projects.setStore(store)
	result := app.Launch(launch.Request{Path: `C:\src\lyn-tools`, Action: "reveal"})
	if result.Error != "" {
		t.Fatalf("unexpected result %#v", result)
	}
	projects, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 0 {
		t.Fatalf("reveal should not record usage: %#v", projects)
	}
}
