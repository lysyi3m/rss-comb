package types

import "time"

type Settings struct {
	RefreshInterval int  `yaml:"refresh_interval" json:"refresh_interval"`
	MaxItems        int  `yaml:"max_items" json:"max_items"`
	Timeout         int  `yaml:"timeout" json:"timeout"`
	ExtractContent  bool `yaml:"extract_content" json:"extract_content"`
	ExtractMedia bool `yaml:"extract_media" json:"extract_media"`
}

type Filter struct {
	Field    string   `yaml:"field" json:"field"`
	Includes []string `yaml:"includes" json:"includes"`
	Excludes []string `yaml:"excludes" json:"excludes"`
}

type Metadata struct {
	Title           string
	Link            string
	Description     string
	ImageURL        string
	Language        string
	FeedPublishedAt *time.Time
	FeedUpdatedAt   *time.Time
	// iTunes podcast extension fields
	ITunesAuthor     string
	ITunesImage      string
	ITunesExplicit   string
	ITunesOwnerName  string
	ITunesOwnerEmail string
}
