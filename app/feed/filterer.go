package feed

import (
	"fmt"
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
		isFiltered, filterReason := f.applyFilters(item, feedConfig.Filters)
		item.IsFiltered = isFiltered
		item.FilterReason = filterReason
		filtered = append(filtered, item)
	}

	return filtered
}

func (f *Filterer) applyFilters(item Item, filters []ConfigFilter) (bool, string) {
	for _, filter := range filters {
		value := f.getFieldValue(item, filter.Field)

		for _, exclude := range filter.Excludes {
			if f.matchesFilter(value, exclude) {
				return true, fmt.Sprintf("Excluded by %s filter: contains '%s'", filter.Field, exclude)
			}
		}

		if len(filter.Includes) > 0 {
			matched := false
			for _, include := range filter.Includes {
				if f.matchesFilter(value, include) {
					matched = true
					break
				}
			}
			if !matched {
				return true, fmt.Sprintf("Excluded by %s filter: does not contain any of %v", filter.Field, filter.Includes)
			}
		}
	}

	return false, ""
}

func (f *Filterer) matchesFilter(value, pattern string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(pattern))
}

func (f *Filterer) getFieldValue(item Item, field string) string {
	switch field {
	case "title":
		return item.Title
	case "description":
		return item.Description
	case "content":
		return item.Content
	case "authors":
		return strings.Join(item.Authors, " ")
	case "link":
		return item.Link
	case "categories":
		return strings.Join(item.Categories, " ")
	default:
		return ""
	}
}
