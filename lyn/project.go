package lyn

import (
	"cmp"
	"maps"
	"slices"
	"time"
)

const (
	projectKindApp             = "app"
	projectKindGo              = "go"
	projectKindLaravel         = "laravel"
	projectKindNode            = "node"
	projectKindRust            = "rust"
	projectKindSystemCommand   = "system-command"
	projectKindVSCodeRecent    = "vscode-recent"
	projectKindVSCodeWorkspace = "vscode-workspace"
	projectKindWordPress       = "wordpress"
)

type Project struct {
	Name           string    `json:"name"`
	Path           string    `json:"path"`
	Kind           string    `json:"kind"`
	DisplayName    string    `json:"displayName,omitempty"`
	DetectedAt     time.Time `json:"detectedAt"`
	UsageCount     int       `json:"usageCount"`
	LastLaunchedAt time.Time `json:"lastLaunchedAt"`
}

func newProject(name string, path string, kind string) Project {
	return Project{Name: name, Path: path, Kind: kind, DetectedAt: time.Now().UTC()}
}

func compareProjects(a, b Project) int {
	if byName := cmp.Compare(a.Name, b.Name); byName != 0 {
		return byName
	}
	return cmp.Compare(a.Path, b.Path)
}

func compareRankedProjects(a, b Project) int {
	if byUsage := cmp.Compare(b.UsageCount, a.UsageCount); byUsage != 0 {
		return byUsage
	}
	if byLaunch := b.LastLaunchedAt.Compare(a.LastLaunchedAt); byLaunch != 0 {
		return byLaunch
	}
	return compareProjects(a, b)
}

type projectSet map[string]Project

func newProjectSet(capacity int) projectSet {
	return make(projectSet, capacity)
}

func (s projectSet) add(project Project) {
	s[project.Path] = project
}

func (s projectSet) addIfAbsent(project Project) {
	if _, ok := s[project.Path]; ok {
		return
	}
	s.add(project)
}

func (s projectSet) addAll(projects []Project) {
	for _, project := range projects {
		s.add(project)
	}
}

func (s projectSet) addMerged(project Project) {
	if existing, ok := s[project.Path]; ok {
		if existing.UsageCount > project.UsageCount {
			project.UsageCount = existing.UsageCount
		}
		if existing.LastLaunchedAt.After(project.LastLaunchedAt) {
			project.LastLaunchedAt = existing.LastLaunchedAt
		}
	}
	s.add(project)
}

func (s projectSet) sorted() []Project {
	return slices.SortedFunc(maps.Values(s), compareProjects)
}

func mergeProjects(groups ...[]Project) []Project {
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	seen := newProjectSet(total)
	for _, group := range groups {
		for _, item := range group {
			seen.addMerged(item)
		}
	}
	return seen.sorted()
}
