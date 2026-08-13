package lyn

import (
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"

	"lyn.tools/launcher/lyn/launch"
)

func (s *projectService) resolveLaunchTarget(req launch.Request) launch.Request {
	if launch.NormalizedAction(req.Action) != "code" {
		return req
	}
	project, ok := s.findIndexedProject(strings.TrimSpace(req.Path))
	if !ok || project.Kind != projectKindWordPress {
		return req
	}
	if target, ok := wordpressCodeTarget(project, s.workspacePathsUnder(project)); ok {
		req.Path = target
	}
	return req
}

func (s *projectService) findIndexedProject(path string) (Project, bool) {
	if path == "" {
		return Project{}, false
	}
	for _, item := range s.currentSearchIndex() {
		if item.project.Path == path {
			return item.project, true
		}
	}
	return Project{}, false
}

func (s *projectService) workspacePathsUnder(project Project) []string {
	var paths []string
	for _, item := range s.currentSearchIndex() {
		p := item.project
		if p.Kind == projectKindVSCodeWorkspace && p.Distro == project.Distro && isLaunchPathUnder(p.Path, project.Path) {
			paths = append(paths, p.Path)
		}
	}
	return paths
}

func wordpressCodeTarget(project Project, workspacePaths []string) (string, bool) {
	if target, ok := mostRecentLaunchPath(project.Distro, workspacePaths); ok {
		return target, true
	}
	return mostRecentLaunchPath(project.Distro, themeLaunchPaths(project))
}

func themeLaunchPaths(project Project) []string {
	themes := joinLaunchPath(project, "wp-content", "themes")
	entries, err := os.ReadDir(launchFSPath(project.Distro, themes))
	if err != nil {
		return nil
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			paths = append(paths, joinLaunchPath(project, "wp-content", "themes", entry.Name()))
		}
	}
	return paths
}

func mostRecentLaunchPath(distro string, paths []string) (string, bool) {
	best := ""
	var bestModTime time.Time
	for _, path := range paths {
		info, err := os.Stat(launchFSPath(distro, path))
		if err != nil {
			continue
		}
		if best == "" || info.ModTime().After(bestModTime) {
			best = path
			bestModTime = info.ModTime()
		}
	}
	return best, best != ""
}

func joinLaunchPath(project Project, parts ...string) string {
	segments := append([]string{project.Path}, parts...)
	if project.Distro != "" {
		return pathpkg.Join(segments...)
	}
	return filepath.Join(segments...)
}

func launchFSPath(distro string, launchPath string) string {
	if distro != "" {
		return wslWindowsRoot(distro, launchPath)
	}
	return launchPath
}

func isLaunchPathUnder(child string, parent string) bool {
	for _, sep := range []string{"/", string(filepath.Separator), `\`} {
		if strings.HasPrefix(child, parent+sep) {
			return true
		}
	}
	return false
}
