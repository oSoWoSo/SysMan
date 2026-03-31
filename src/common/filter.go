package common

import "strings"

// Filter filters items by state and search query.
// mode: 0 shows all, 1 shows only matched, 2 shows only unmatched.
// The isMatched function should return true for items in the "state A" category (e.g. installed, enabled, running).
// The matchesSearch function should return true if the item matches the search query.
func Filter[T any](
	items []T,
	mode int,
	search string,
	isMatched func(T) bool,
	matchesSearch func(T, string) bool,
) []T {
	var out []T
	q := strings.ToLower(search)
	for _, item := range items {
		switch mode {
		case 1:
			if !isMatched(item) {
				continue
			}
		case 2:
			if isMatched(item) {
				continue
			}
		}
		if q != "" && !matchesSearch(item, q) {
			continue
		}
		out = append(out, item)
	}
	return out
}
