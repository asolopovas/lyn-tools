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
