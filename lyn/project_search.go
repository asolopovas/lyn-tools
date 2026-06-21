package lyn

import (
	"slices"
	"strings"
	"unicode"
)

const (
	maxSearchMatches = 12
	noSearchMatch    = 99
	minLooseTermPart = 2
)

const (
	scoreExact = iota
	scoreNamePrefix
	scoreWordPrefix
	scoreLabelSubstr
	scoreTextSubstr
	scoreAllTerms
	scoreApprox
	scoreSubseq
)

const (
	bonusBoundary = 16
	bonusCamel    = 12
	bonusConsec   = 8
	penaltyGap    = 2
)

type searchProject struct {
	project    Project
	text       string
	words      []string
	label      string
	labelRaw   string
	labelWords []string
}

type scoredProject struct {
	project Project
	rank    int
	quality int
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
	workspaceEnabled := workspaceShortcut != ""
	workspaceMode := workspaceEnabled && strings.HasPrefix(raw, workspaceShortcut)
	text := strings.ToLower(raw)
	if workspaceMode {
		text = strings.ToLower(strings.TrimSpace(raw[len(workspaceShortcut):]))
	}
	scored := make([]scoredProject, 0, len(index))
	for _, item := range index {
		if !includeInSearch(item.project, workspaceMode, workspaceEnabled) {
			continue
		}
		rank, quality := projectSearchScore(item, text)
		if rank == noSearchMatch {
			continue
		}
		scored = append(scored, scoredProject{project: item.project, rank: rank, quality: quality})
	}
	slices.SortStableFunc(scored, func(a, b scoredProject) int {
		if a.rank != b.rank {
			return a.rank - b.rank
		}
		return b.quality - a.quality
	})
	result := make([]Project, 0, maxSearchMatches)
	for _, item := range scored {
		result = append(result, item.project)
		if len(result) == maxSearchMatches {
			break
		}
	}
	return result
}

func newSearchProject(project Project) searchProject {
	text := project.Name + " " + project.DisplayName + " " + project.Kind + " " + project.Path
	labelRaw := strings.TrimSpace(project.Name + " " + project.DisplayName)
	label := strings.ToLower(labelRaw)
	return searchProject{
		project:    project,
		text:       strings.ToLower(text),
		words:      searchWords(text),
		label:      label,
		labelRaw:   labelRaw,
		labelWords: searchWords(label),
	}
}

func projectSearchScore(item searchProject, text string) (int, int) {
	if text == "" {
		return scoreExact, 0
	}
	if score := labelScore(item, text); score != noSearchMatch {
		return score, 0
	}
	if strings.Contains(item.text, text) {
		return scoreTextSubstr, 0
	}
	if rank := termScore(item, text); rank != noSearchMatch {
		return rank, 0
	}
	if quality, ok := subsequenceScore(item, text); ok {
		return scoreSubseq, quality
	}
	return noSearchMatch, 0
}

func labelScore(item searchProject, text string) int {
	if item.label == text {
		return scoreExact
	}
	if strings.HasPrefix(item.label, text) {
		return scoreNamePrefix
	}
	if slices.ContainsFunc(item.labelWords, func(word string) bool { return strings.HasPrefix(word, text) }) {
		return scoreWordPrefix
	}
	if strings.Contains(item.label, text) {
		return scoreLabelSubstr
	}
	return noSearchMatch
}

func termScore(item searchProject, text string) int {
	words := searchWords(text)
	if len(words) == 0 {
		return noSearchMatch
	}
	approximate := false
	for _, term := range words {
		if termInWords(item.words, term) {
			continue
		}
		if slices.ContainsFunc(item.words, func(word string) bool { return fuzzyWordMatch(term, word) }) {
			approximate = true
			continue
		}
		if looseTermMatch(item.words, term) {
			approximate = true
			continue
		}
		return noSearchMatch
	}
	if approximate {
		return scoreApprox
	}
	return scoreAllTerms
}

func subsequenceScore(item searchProject, text string) (int, bool) {
	query := strings.ReplaceAll(text, " ", "")
	if query == "" {
		return 0, false
	}
	return fuzzyMatchScore(item.labelRaw, query)
}

func fuzzyMatchScore(target string, query string) (int, bool) {
	chars := []rune(target)
	score := 0
	at := 0
	previous := -1
	for _, q := range query {
		found := -1
		for ; at < len(chars); at++ {
			if unicode.ToLower(chars[at]) == unicode.ToLower(q) {
				found = at
				break
			}
		}
		if found == -1 {
			return 0, false
		}
		score += charMatchScore(chars, found, found == previous+1)
		previous = found
		at = found + 1
	}
	return score, true
}

func charMatchScore(chars []rune, at int, consecutive bool) int {
	score := 1
	switch {
	case at == 0 || isSearchSeparator(chars[at-1]):
		score += bonusBoundary
	case unicode.IsLower(chars[at-1]) && unicode.IsUpper(chars[at]):
		score += bonusCamel
	}
	if consecutive {
		score += bonusConsec
	} else if at > 0 {
		score -= penaltyGap
	}
	return score
}

func isSearchSeparator(r rune) bool {
	switch r {
	case ' ', '/', '\\', '-', '_', '.':
		return true
	default:
		return false
	}
}

func termInWords(words []string, term string) bool {
	return slices.ContainsFunc(words, func(word string) bool { return strings.Contains(word, term) })
}

func looseTermMatch(words []string, term string) bool {
	if len(term) < 2*minLooseTermPart {
		return false
	}
	for cut := minLooseTermPart; cut <= len(term)-minLooseTermPart; cut++ {
		if termInWords(words, term[:cut]) && termInWords(words, term[cut:]) {
			return true
		}
	}
	return false
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

func includeInSearch(project Project, workspaceMode bool, workspaceEnabled bool) bool {
	folder := isWorkspaceSearchProject(project)
	if workspaceMode {
		return folder
	}
	if !folder {
		return true
	}
	return !workspaceEnabled || project.UsageCount > 0
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
