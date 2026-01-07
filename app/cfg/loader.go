package cfg

import (
	"cmp"
	"fmt"
	"time"

	"github.com/jessevdk/go-flags"
)

// Version is set at build time via -ldflags
var Version = "dev"

func GetVersion() string {
	return cmp.Or(Version, "unknown")
}

func Load() (*Cfg, error) {
	cfg := &Cfg{}

	parser := flags.NewParser(cfg, flags.Default)

	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok {
			if flagsErr.Type == flags.ErrHelp {
				return nil, nil
			}
		}
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	loc, err := loadTimezone(cfg.Timezone)
	if err != nil {
		fmt.Printf("Warning: Invalid timezone '%s', using UTC: %v\n", cfg.Timezone, err)
		loc = time.UTC
	}

	cfg.Version = GetVersion()
	cfg.Location = loc

	return cfg, nil
}

func loadTimezone(timezone string) (*time.Location, error) {
	if timezone == "" {
		return time.UTC, nil
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Timezone configured: %s\n", timezone)
	return loc, nil
}
