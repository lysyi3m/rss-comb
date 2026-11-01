package cfg

import (
	"cmp"
	"fmt"
	"time"

	"github.com/jessevdk/go-flags"
)

// Version is set at build time via -ldflags
var Version = "dev"

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

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	loc, err := loadTimezone(cfg.Timezone)
	if err != nil {
		fmt.Printf("Warning: Invalid timezone '%s', using UTC: %v\n", cfg.Timezone, err)
		loc = time.UTC
	}

	cfg.Version = cmp.Or(Version, "unknown")
	cfg.Location = loc

	return cfg, nil
}

func (c *Cfg) validate() error {
	if c.WorkerCount < 1 {
		return fmt.Errorf("WORKER_COUNT must be at least 1, got %d", c.WorkerCount)
	}
	if c.SchedulerInterval < 1 {
		return fmt.Errorf("SCHEDULER_INTERVAL must be at least 1, got %d", c.SchedulerInterval)
	}
	return nil
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
