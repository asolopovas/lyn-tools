package lyn

import "strings"

type stringSet map[string]struct{}

func newStringSet(capacity int) stringSet {
	return make(stringSet, capacity)
}

func (s stringSet) add(value string) bool {
	if s.has(value) {
		return false
	}
	s[value] = struct{}{}
	return true
}

func (s stringSet) addFold(value string) bool {
	return s.add(strings.ToLower(value))
}

func (s stringSet) has(value string) bool {
	_, ok := s[value]
	return ok
}
