package lyn

import (
	"slices"
	"strings"
	"unicode"
)

const (
	maxSearchMatches = 12
	noSearchMatch    = 99
)

type searchProject struct {
	project Project
	text    string
	words   []string
}

func searchProjects(projects []Project, query string, workspaceShortcut string) []Project {
	return searchProjectIndex(newSearchIndex(projects), query, workspaceShortcut)
}

func newSearchIndex(projects []Project) []searchProject {
	index := make([]searchProject, 0, len(projects))
	for _, project := range projects {
		index = append(index, newSearchProject(project))
	}
	slices.SortStableFunc(index, func(a, b searchProject) int {
		return compareRankedProjects(a.project, b.project)
	})
	return index
}

func searchProjectIndex(index []searchProject, query string, workspaceShortcut string) []Project {
	raw := strings.TrimSpace(query)
	workspaceMode := workspaceShortcut != "" && strings.HasPrefix(raw, workspaceShortcut)
	text := strings.ToLower(raw)
	if workspaceMode {
		text = strings.ToLower(strings.TrimSpace(raw[len(workspaceShortcut):]))
	}
	tiers := [3][]Project{}
	for _, item := range index {
		if workspaceMode && !isWorkspaceSearchProject(item.project) {
			continue
		}
		score := projectSearchScore(item, text)
		if score == noSearchMatch || len(tiers[score]) == maxSearchMatches {
			continue
		}
		tiers[score] = append(tiers[score], item.project)
	}
	result := make([]Project, 0, maxSearchMatches)
	for _, tier := range tiers {
		result = append(result, tier...)
		if len(result) >= maxSearchMatches {
			return result[:maxSearchMatches]
		}
	}
	return result
}

func newSearchProject(project Project) searchProject {
	text := project.Name + " " + project.Kind + " " + project.Path
	return searchProject{project: project, text: strings.ToLower(text), words: searchWords(text)}
}

func projectSearchScore(item searchProject, text string) int {
	if text == "" || strings.Contains(item.text, text) {
		return 0
	}
	words := searchWords(text)
	usedFuzzy := false
	for _, term := range words {
		if slices.ContainsFunc(item.words, func(word string) bool { return strings.Contains(word, term) }) {
			continue
		}
		if slices.ContainsFunc(item.words, func(word string) bool { return fuzzyWordMatch(term, word) }) {
			usedFuzzy = true
			continue
		}
		return noSearchMatch
	}
	if usedFuzzy {
		return 2
	}
	return 1
}

func searchWords(text string) []string {
	words := strings.FieldsFunc(splitCamel(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	seen := newStringSet(len(words))
	result := make([]string, 0, len(words))
	for _, word := range words {
		word = strings.ToLower(word)
		if word == "" || !seen.add(word) {
			continue
		}
		result = append(result, word)
	}
	return result
}

func splitCamel(text string) string {
	var b strings.Builder
	var previous rune
	for _, current := range text {
		if previous != 0 && unicode.IsLower(previous) && unicode.IsUpper(current) {
			b.WriteByte(' ')
		}
		b.WriteRune(current)
		previous = current
	}
	return b.String()
}

func fuzzyWordMatch(term string, word string) bool {
	maxDistance := typoDistance(term)
	if maxDistance == 0 || len(term) > 64 || word == "" {
		return false
	}
	if abs(len(word)-len(term)) <= maxDistance && boundedDamerauDistance(term, word, maxDistance) <= maxDistance {
		return true
	}
	if len(word) <= len(term) {
		return false
	}
	if boundedDamerauDistance(term, word[:len(term)], maxDistance) <= maxDistance {
		return true
	}
	prefixLength := min(len(word), len(term)+maxDistance)
	return boundedDamerauDistance(term, word[:prefixLength], maxDistance) <= maxDistance
}

func typoDistance(term string) int {
	if len(term) < 4 {
		return 0
	}
	if len(term) < 7 {
		return 1
	}
	return 2
}

func boundedDamerauDistance(a string, b string, maxDistance int) int {
	if a == b {
		return 0
	}
	if abs(len(a)-len(b)) > maxDistance {
		return maxDistance + 1
	}
	prevPrev := make([]int, len(b)+1)
	prev := make([]int, len(b)+1)
	for j := range prevPrev {
		prevPrev[j] = maxDistance + 1
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		current := make([]int, len(b)+1)
		current[0] = i
		rowMin := current[0]
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			distance := min(prev[j]+1, current[j-1]+1, prev[j-1]+cost)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				distance = min(distance, prevPrev[j-2]+1)
			}
			current[j] = distance
			rowMin = min(rowMin, distance)
		}
		if rowMin > maxDistance {
			return maxDistance + 1
		}
		prevPrev = prev
		prev = current
	}
	return prev[len(b)]
}

func isWorkspaceSearchProject(project Project) bool {
	return project.Kind != projectKindApp && project.Kind != projectKindSystemCommand
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
