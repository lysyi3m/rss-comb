package feed

import (
	"strings"

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
	return strings.Contains(strings.ToLower(value), strings.ToLower(pattern))
}
