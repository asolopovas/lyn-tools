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

func TestSearchProjectsRanksWorkspacesAboveFolders(t *testing.T) {
	projects := []Project{
		{Name: "app", Path: "/app", Kind: projectKindGo, UsageCount: 9},
		{Name: "app", Path: "/app.code-workspace", Kind: projectKindVSCodeWorkspace},
	}
	matches := searchProjects(projects, "{", "{")
	if len(matches) != 2 || matches[0].Kind != projectKindVSCodeWorkspace {
		t.Fatalf("expected workspace ranked above folder, got %#v", matches)
	}
}

func TestSearchProjectsRanksNamePrefixAbovePathMatch(t *testing.T) {
	projects := []Project{
		{Name: "Alpha", Path: `/opt/widgets/alpha`, Kind: projectKindApp, UsageCount: 50},
		{Name: "Widget", Path: `/opt/alpha/widget`, Kind: projectKindApp, UsageCount: 1},
	}
	matches := searchProjects(projects, "wi", "{")
	if len(matches) < 2 || matches[0].Name != "Widget" {
		t.Fatalf("expected name prefix ranked above path match, got %#v", matches)
	}
}

func TestSearchProjectsRanksNamePrefixByUsage(t *testing.T) {
	projects := []Project{
		{Name: "Widgetbar", Path: `/opt/widgetbar`, Kind: projectKindApp, UsageCount: 2},
		{Name: "Widgetbox", Path: `/opt/widgetbox`, Kind: projectKindApp, UsageCount: 9},
	}
	matches := searchProjects(projects, "wi", "{")
	if len(matches) < 2 || matches[0].Name != "Widgetbox" {
		t.Fatalf("expected most-used prefix match first, got %#v", matches)
	}
}

func TestSearchProjectsMatchesSubsequence(t *testing.T) {
	projects := []Project{
		{Name: "Widescribe", Path: `/opt/widescribe`, Kind: projectKindApp},
		{Name: "Notepad", Path: `/opt/notepad`, Kind: projectKindApp},
	}
	matches := searchProjects(projects, "wdscrb", "{")
	if len(matches) != 1 || matches[0].Name != "Widescribe" {
		t.Fatalf("expected subsequence match for Widescribe, got %#v", matches)
	}
}

func TestSearchProjectsSubsequenceRanksTighterMatchFirst(t *testing.T) {
	projects := []Project{
		{Name: "Wonderful Diagnostic Scribe", Path: `/opt/wds`, Kind: projectKindApp, UsageCount: 9},
		{Name: "Widescribe", Path: `/opt/widescribe`, Kind: projectKindApp, UsageCount: 1},
	}
	matches := searchProjects(projects, "wdscrb", "{")
	if len(matches) < 2 || matches[0].Name != "Widescribe" {
		t.Fatalf("expected the tighter subsequence match first, got %#v", matches)
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
	for _, query := range []string{"{examplessh", "{sshexample"} {
		matches := searchProjects(projects, query, "{")
		if len(matches) != 1 || matches[0].Path != "vscode-remote://ssh-remote+examplehost/srv/www/example" {
			t.Fatalf("query %q: expected only the SSH recent, got %#v", query, matches)
		}
	}
	for _, query := range []string{"{examplewsl", "{wslexample"} {
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
	if matches := searchProjects(projects, "{sshexample", "{"); len(matches) != 0 {
		t.Fatalf("expected no match for a local project, got %#v", matches)
	}
}

func TestSearchProjectsMainIndexHidesUnopenedFolders(t *testing.T) {
	projects := []Project{
		{Name: "Editor", Path: "/editor", Kind: projectKindApp},
		{Name: "opened", Path: "/opened", Kind: projectKindGo, UsageCount: 2},
		{Name: "fresh", Path: "/fresh", Kind: projectKindGo},
		{Name: "recent", Path: "/recent", Kind: projectKindVSCodeRecent},
	}
	matches := searchProjects(projects, "", "{")
	if len(matches) != 2 {
		t.Fatalf("expected app and opened folder only, got %#v", matches)
	}
	for _, match := range matches {
		if match.UsageCount == 0 && isWorkspaceSearchProject(match) {
			t.Fatalf("unopened folder leaked into main index: %#v", matches)
		}
	}
}

func TestSearchProjectsWorkspaceShortcutShowsUnopenedFolders(t *testing.T) {
	projects := []Project{
		{Name: "fresh", Path: "/fresh", Kind: projectKindGo},
	}
	matches := searchProjects(projects, "{", "{")
	if len(matches) != 1 || matches[0].Name != "fresh" {
		t.Fatalf("expected unopened folder under workspace shortcut, got %#v", matches)
	}
}

func TestSearchProjectsWithoutShortcutShowsUnopenedFolders(t *testing.T) {
	projects := []Project{
		{Name: "fresh", Path: "/fresh", Kind: projectKindGo},
	}
	matches := searchProjects(projects, "", "")
	if len(matches) != 1 || matches[0].Name != "fresh" {
		t.Fatalf("expected folder shown when shortcut disabled, got %#v", matches)
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
