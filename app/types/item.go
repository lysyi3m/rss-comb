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
	IsFiltered              bool
	ContentExtractionStatus *string
	EnclosureURL    string
	EnclosureLength int64
	EnclosureType   string
	// iTunes podcast episode extension fields
	ITunesDuration    int    // Duration in seconds
	ITunesEpisode     int    // Episode number
	ITunesSeason      int    // Season number
	ITunesEpisodeType string // full/trailer/bonus
	ITunesImage       string // Episode-specific artwork
}
