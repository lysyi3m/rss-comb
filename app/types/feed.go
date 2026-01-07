package types

type Settings struct {
	RefreshInterval int  `yaml:"refresh_interval" json:"refresh_interval"`
	MaxItems        int  `yaml:"max_items" json:"max_items"`
	Timeout         int  `yaml:"timeout" json:"timeout"`
	ExtractContent  bool `yaml:"extract_content" json:"extract_content"`
}

type Filter struct {
	Field    string   `yaml:"field" json:"field"`
	Includes []string `yaml:"includes" json:"includes"`
	Excludes []string `yaml:"excludes" json:"excludes"`
}
