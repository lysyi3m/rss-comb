package feed

import (
	"strings"
)

type Filterer struct{}

func NewFilterer() *Filterer {
	return &Filterer{}
}

func (f *Filterer) Run(items []Item, feedConfig *Config) []Item {
	if len(feedConfig.Filters) == 0 {
		return items
	}

	filtered := make([]Item, 0, len(items))
	for _, item := range items {
		item.IsFiltered = f.applyFilters(item, feedConfig.Filters)
		filtered = append(filtered, item)
	}

	return filtered
}

func (f *Filterer) applyFilters(item Item, filters []ConfigFilter) bool {
	for _, filter := range filters {
		for _, exclude := range filter.Excludes {
			if f.matchesFieldFilter(item, filter.Field, exclude) {
				return true
			}
		}

		if len(filter.Includes) > 0 {
			matched := false
			for _, include := range filter.Includes {
				if f.matchesFieldFilter(item, filter.Field, include) {
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

func (f *Filterer) matchesFieldFilter(item Item, field, pattern string) bool {
	switch field {
	case "title":
		return f.matchesPattern(item.Title, pattern)
	case "description":
		return f.matchesPattern(item.Description, pattern)
	case "content":
		return f.matchesPattern(item.Content, pattern)
	case "link":
		return f.matchesPattern(item.Link, pattern)
	case "authors":
		for _, author := range item.Authors {
			if f.matchesPattern(author, pattern) {
				return true
			}
		}
		return false
	case "categories":
		for _, category := range item.Categories {
			if f.matchesPattern(category, pattern) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (f *Filterer) matchesPattern(value, pattern string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(pattern))
}

