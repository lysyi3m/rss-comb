package version

// Version is the current version of the application
// This variable is set at build time via -ldflags
var Version = "dev"

// GetVersion returns the current version of the application
func GetVersion() string {
	if Version == "" {
		return "unknown"
	}
	return Version
}