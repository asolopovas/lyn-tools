package lyn

import (
	"testing"
	"time"
)

func TestSearchProjectsRanksEmptyQueryByUsage(t *testing.T) {
	now := time.Now()
	projects := []Project{
		{Name: "Alpha", Path: "/alpha", UsageCount: 1, LastLaunchedAt: now},
		{Name: "Beta", Path: "/beta", UsageCount: 5, LastLaunchedAt: now.Add(-time.Hour)},
		{Name: "Gamma", Path: "/gamma", UsageCount: 1, LastLaunchedAt: now.Add(time.Hour)},
	}
	matches := searchProjects(projects, "", "{")
	if len(matches) != 3 || matches[0].Name != "Beta" || matches[1].Name != "Gamma" || matches[2].Name != "Alpha" {
		t.Fatalf("unexpected ranking %#v", matches)
	}
}

func TestSearchProjectsMatchesTypos(t *testing.T) {
	projects := []Project{
		{Name: "Calendar", Path: "/calendar", Kind: projectKindApp},
		{Name: "Calculator", Path: "/calculator", Kind: projectKindApp},
	}
	matches := searchProjects(projects, "calclator", "{")
	if len(matches) == 0 || matches[0].Name != "Calculator" {
		t.Fatalf("expected Calculator, got %#v", matches)
	}
}

func TestSearchProjectsPrefersExactMatches(t *testing.T) {
	projects := []Project{
		{Name: "Calculator", Path: "/calculator", Kind: projectKindApp, UsageCount: 1},
		{Name: "Calclator", Path: "/calclator", Kind: projectKindApp},
	}
	matches := searchProjects(projects, "calclator", "{")
	if len(matches) < 2 || matches[0].Name != "Calclator" || matches[1].Name != "Calculator" {
		t.Fatalf("expected exact before fuzzy, got %#v", matches)
	}
}

func TestSearchProjectsMatchesGluedRemoteTerms(t *testing.T) {
	projects := []Project{
		{Name: "example", Path: "/home/me/src/example", Kind: projectKindGo},
		{Name: "example", Path: "vscode-remote://ssh-remote+examplehost/srv/www/example", Kind: projectKindVSCodeRecent},
		{Name: "example", Path: `\\wsl.localhost\Ubuntu\home\me\src\example`, Kind: projectKindVSCodeRecent},
	}
	for _, query := range []string{"examplessh", "sshexample"} {
		matches := searchProjects(projects, query, "{")
		if len(matches) != 1 || matches[0].Path != "vscode-remote://ssh-remote+examplehost/srv/www/example" {
			t.Fatalf("query %q: expected only the SSH recent, got %#v", query, matches)
		}
	}
	for _, query := range []string{"examplewsl", "wslexample"} {
		matches := searchProjects(projects, query, "{")
		if len(matches) != 1 || matches[0].Path != `\\wsl.localhost\Ubuntu\home\me\src\example` {
			t.Fatalf("query %q: expected only the WSL recent, got %#v", query, matches)
		}
	}
}

func TestSearchProjectsGluedTermIgnoresUnrelatedProjects(t *testing.T) {
	projects := []Project{
		{Name: "example", Path: "/home/me/src/example", Kind: projectKindGo},
	}
	if matches := searchProjects(projects, "sshexample", "{"); len(matches) != 0 {
		t.Fatalf("expected no match for a local project, got %#v", matches)
	}
}

func TestSearchProjectsWorkspaceShortcutExcludesApps(t *testing.T) {
	projects := []Project{
		{Name: "Calculator", Path: "/calculator", Kind: projectKindApp},
		{Name: "Calculator Workspace", Path: "/calculator.code-workspace", Kind: projectKindVSCodeWorkspace},
	}
	matches := searchProjects(projects, "{calclator", "{")
	if len(matches) != 1 || matches[0].Kind != projectKindVSCodeWorkspace {
		t.Fatalf("expected workspace only, got %#v", matches)
	}
}
