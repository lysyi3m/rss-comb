package feed

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/types"
)

func TestBuildStaysValidXMLForHostileValues(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"CDATA terminator", "a]]>b", "a]]>b"},
		{"repeated terminator", "]]>]]>", "]]>]]>"},
		{"control character", "a\x01b", "a�b"},
		{"invalid UTF-8", "a\xffb", "a�b"},
		{"plain markup", "<p>x</p>", "<p>x</p>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := database.Item{Item: types.Item{
				GUID:          "g",
				Title:         "t",
				Description:   "d",
				Content:       tt.content,
				PublishedAt:   time.Unix(0, 0),
				EnclosureURL:  "https://example.com/a\x01.mp3?a=1&b=\"2\"",
				EnclosureType: "audio/mpeg",
				ITunesImage:   "https://example.com/\x01.png",
			}}
			out, err := podcastType{}.Build(database.Feed{Name: "f"}, []database.Item{item}, &cfg.Cfg{Location: time.UTC})
			if err != nil {
				t.Fatal(err)
			}

			var doc struct {
				Content string `xml:"channel>item>encoded"`
			}
			if err := xml.NewDecoder(strings.NewReader(out)).Decode(&doc); err != nil {
				t.Fatalf("invalid XML: %v\n%s", err, out)
			}
			if doc.Content != tt.want {
				t.Errorf("content = %q, want %q", doc.Content, tt.want)
			}
		})
	}
}
