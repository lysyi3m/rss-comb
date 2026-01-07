package types

import "time"

type Item struct {
	GUID            string
	Title           string
	Link            string
	Description     string
	Content         string
	PublishedAt     time.Time
	UpdatedAt       *time.Time
	Authors         []string
	Categories      []string
	ContentHash     string
	IsFiltered      bool
	EnclosureURL    string
	EnclosureLength int64
	EnclosureType   string
}
