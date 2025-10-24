package logger

// Config holds the configuration for the logger
type Config struct {
	// Level is the minimum logging level (debug, info, warn, error, fatal)
	Level string

	// Format is the output format (console or json)
	Format string

	// FilePath is the optional file path to write logs to
	// If empty, logs are only written to stdout
	FilePath string
}

// DefaultDevelopmentConfig returns a default configuration for development
func DefaultDevelopmentConfig() Config {
	return Config{
		Level:    "debug",
		Format:   "console",
		FilePath: "",
	}
}

// DefaultProductionConfig returns a default configuration for production
func DefaultProductionConfig() Config {
	return Config{
		Level:    "info",
		Format:   "json",
		FilePath: "",
	}
}
