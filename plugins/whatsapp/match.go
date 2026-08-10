package main

import "strings"

// matchesAnyKeyword reports whether name case-insensitively equals any of
// keywords via Unicode simple case folding (strings.EqualFold) — exact
// match only, no substring or prefix matching. Mirrors
// plugins/signal/match.go's identical function.
func matchesAnyKeyword(name string, keywords []string) bool {
	if name == "" {
		return false
	}
	for _, kw := range keywords {
		if strings.EqualFold(name, kw) {
			return true
		}
	}
	return false
}

// candidateNames returns the name(s) c is eligible to match a webspace
// keyword against. In this plan, only a group's OWN subject is ever a
// candidate (T-08-01's mitigation) — a 1:1's system/nickname candidates
// are Plan 08-02's D-05 widening, deliberately absent here.
func candidateNames(c chatRecord) []string {
	if !c.IsGroup {
		return nil
	}
	if c.Name == "" {
		return nil
	}
	return []string{c.Name}
}

// matchesChat reports whether c has at least one candidate name matching
// any of keywords.
func matchesChat(c chatRecord, keywords []string) bool {
	for _, candidate := range candidateNames(c) {
		if matchesAnyKeyword(candidate, keywords) {
			return true
		}
	}
	return false
}

// eligibleChats filters chats to groups only (this plan's scope — see
// candidateNames), returning only those matching at least one of
// keywords. An empty keyword list returns zero matches.
func eligibleChats(chats []chatRecord, keywords []string) []chatRecord {
	var out []chatRecord
	for _, c := range chats {
		if matchesChat(c, keywords) {
			out = append(out, c)
		}
	}
	return out
}
