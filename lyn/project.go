package lyn

import (
	"cmp"
	"maps"
	"runtime"
	"slices"
	"strings"
	"time"
)

const (
	projectKindApp             = "app"
	projectKindGit             = "git"
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
	Distro         string    `json:"distro,omitempty"`
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
	if byKind := kindRank(a) - kindRank(b); byKind != 0 {
		return byKind
	}
	if byUsage := cmp.Compare(b.UsageCount, a.UsageCount); byUsage != 0 {
		return byUsage
	}
	if byLaunch := b.LastLaunchedAt.Compare(a.LastLaunchedAt); byLaunch != 0 {
		return byLaunch
	}
	return compareProjects(a, b)
}

func kindRank(p Project) int {
	switch {
	case p.Kind == projectKindVSCodeWorkspace:
		return 0
	case p.Kind == projectKindSystemCommand:
		return 3
	case isSSHRemoteProject(p):
		return 2
	default:
		return 1
	}
}

func isSSHRemoteProject(p Project) bool {
	lower := strings.ToLower(p.Path)
	return strings.HasPrefix(lower, "vscode-remote://ssh-remote+") ||
		strings.HasPrefix(lower, "vscode-remote://ssh-remote%2b")
}

type projectSet map[string]Project

func newProjectSet(capacity int) projectSet {
	return make(projectSet, capacity)
}

func projectKey(path string) string {
	return projectKeyForOS(path, runtime.GOOS)
}

func projectKeyForOS(path string, goos string) string {
	if goos != "windows" || isUnixPath(path) || isVSCodeRemoteURI(path) {
		return path
	}
	return strings.ToLower(path)
}

func (s projectSet) add(project Project) {
	s[projectKey(project.Path)] = project
}

func (s projectSet) addIfAbsent(project Project) {
	if _, ok := s[projectKey(project.Path)]; ok {
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
	if existing, ok := s[projectKey(project.Path)]; ok {
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
