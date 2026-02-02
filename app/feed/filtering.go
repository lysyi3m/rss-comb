package feed

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"github.com/lysyi3m/rss-comb/app/types"
)

func Filter(items []types.Item, filters []types.Filter) []types.Item {
	if len(filters) == 0 {
		return items
	}

	filtered := make([]types.Item, 0, len(items))
	for _, item := range items {
		item.IsFiltered = applyFilters(item, filters)
		filtered = append(filtered, item)
	}

	return filtered
}

func applyFilters(item types.Item, filters []types.Filter) bool {
	for _, filter := range filters {
		for _, exclude := range filter.Excludes {
			if matchesFieldFilter(item, filter.Field, exclude) {
				return true
			}
		}

		if len(filter.Includes) > 0 {
			matched := false
			for _, include := range filter.Includes {
				if matchesFieldFilter(item, filter.Field, include) {
					matched = true
					break
				}
			}
			if !matched {
				return true
			}
		}
	}

	return false
}

func matchesFieldFilter(item types.Item, field, pattern string) bool {
	switch field {
	case "title":
		return matchesPattern(item.Title, pattern)
	case "description":
		return matchesPattern(item.Description, pattern)
	case "content":
		return matchesPattern(item.Content, pattern)
	case "link":
		return matchesPattern(item.Link, pattern)
	case "authors":
		for _, author := range item.Authors {
			if matchesPattern(author, pattern) {
				return true
			}
		}
		return false
	case "categories":
		for _, category := range item.Categories {
			if matchesPattern(category, pattern) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func matchesPattern(value, pattern string) bool {
	// Normalize Unicode to canonical form (NFC)
	normalizedValue := normalizeUnicode(normalizeWhitespace(strings.ToLower(value)))
	normalizedPattern := normalizeUnicode(normalizeWhitespace(strings.ToLower(pattern)))
	return strings.Contains(normalizedValue, normalizedPattern)
}

func normalizeUnicode(s string) string {
	if !utf8.ValidString(s) {
		return s
	}
	// Use NFC (Canonical Decomposition followed by Canonical Composition)
	// This converts both "й" (U+0439) and "и+combining breve" (U+0438+U+0306) to the same form
	return norm.NFC.String(s)
}

func normalizeWhitespace(s string) string {
	// Replace all types of whitespace with regular spaces
	s = strings.ReplaceAll(s, "\u00a0", " ") // non-breaking space
	s = strings.ReplaceAll(s, "\u2009", " ") // thin space
	s = strings.ReplaceAll(s, "\u202f", " ") // narrow no-break space
	s = strings.ReplaceAll(s, "\t", " ")     // tab
	s = strings.ReplaceAll(s, "\n", " ")     // newline
	s = strings.ReplaceAll(s, "\r", " ")     // carriage return

	// Collapse multiple spaces into single space
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}

	return strings.TrimSpace(s)
}
