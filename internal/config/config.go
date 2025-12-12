// Package config handles configuration loading from CLI flags, environment variables, and TOML files.
// CRC: crc-Config.md
// Spec: deployment.md
package config

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds all configuration settings for the UI server.
type Config struct {
	Server  ServerConfig  `toml:"server"`
	Storage StorageConfig `toml:"storage"`
	Lua     LuaConfig     `toml:"lua"`
	Session SessionConfig `toml:"session"`
	Logging LoggingConfig `toml:"logging"`
}

// ServerConfig holds server-related settings.
type ServerConfig struct {
	Host   string `toml:"host"`
	Port   int    `toml:"port"`
	Socket string `toml:"socket"`
	Dir    string `toml:"-"` // Custom site directory (CLI only, not in config file)
}

// StorageConfig holds storage-related settings.
type StorageConfig struct {
	Type string `toml:"type"` // "memory", "sqlite", "postgresql"
	Path string `toml:"path"` // SQLite file path
	URL  string `toml:"url"`  // PostgreSQL connection URL
}

// LuaConfig holds Lua runtime settings.
type LuaConfig struct {
	Enabled bool   `toml:"enabled"`
	Path    string `toml:"path"`
}

// SessionConfig holds session-related settings.
type SessionConfig struct {
	Timeout Duration `toml:"timeout"` // Session expiration (0 = never)
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level     string `toml:"level"`     // "debug", "info", "warn", "error"
	Verbosity int    `toml:"verbosity"` // 0=none, 1=connections, 2=messages, 3=variables, 4=values
}

// verbosityCounter implements flag.Value for counting -v flags.
type verbosityCounter int

func (v *verbosityCounter) String() string {
	return fmt.Sprintf("%d", *v)
}

func (v *verbosityCounter) Set(string) error {
	*v++
	return nil
}

func (v *verbosityCounter) IsBoolFlag() bool {
	return true
}

// expandVerbosityFlags preprocesses args to expand -vvv into -v -v -v.
// This allows both "-v -v -v" and "-vvv" styles to work.
func expandVerbosityFlags(args []string) []string {
	result := make([]string, 0, len(args))
	for _, arg := range args {
		// Check if this is a -v... flag (but not --verbose or -version etc.)
		if len(arg) > 2 && arg[0] == '-' && arg[1] != '-' && arg[1] == 'v' {
			// Check if all remaining chars are 'v'
			allV := true
			for _, c := range arg[1:] {
				if c != 'v' {
					allV = false
					break
				}
			}
			if allV {
				// Expand -vvv into -v -v -v
				for range arg[1:] {
					result = append(result, "-v")
				}
				continue
			}
		}
		result = append(result, arg)
	}
	return result
}

// Duration is a time.Duration that can be unmarshaled from TOML strings.
type Duration time.Duration

// UnmarshalText implements encoding.TextUnmarshaler for Duration.
func (d *Duration) UnmarshalText(text []byte) error {
	duration, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
}

// Duration returns the underlying time.Duration.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// String returns the duration as a string.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// DefaultConfig returns a Config with all default values.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:   "0.0.0.0",
			Port:   8080,
			Socket: defaultSocketPath(),
		},
		Storage: StorageConfig{
			Type: "memory",
			Path: "ui.db",
		},
		Lua: LuaConfig{
			Enabled: true,
			Path:    "lua/",
		},
		Session: SessionConfig{
			Timeout: Duration(24 * time.Hour),
		},
		Logging: LoggingConfig{
			Level:     "info",
			Verbosity: 0,
		},
	}
}

// defaultSocketPath returns the platform-specific default socket path.
func defaultSocketPath() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\ui`
	}
	return "/tmp/ui.sock"
}

// Load loads configuration from CLI flags, environment variables, and TOML file.
// Priority: CLI flags > env vars > TOML file > defaults
func Load(args []string) (*Config, error) {
	cfg := DefaultConfig()

	// Preprocess args to expand -vvv into -v -v -v
	args = expandVerbosityFlags(args)

	// Parse CLI flags first to get --dir if specified
	fs := flag.NewFlagSet("ui", flag.ContinueOnError)
	dir := fs.String("dir", "", "Serve from directory instead of embedded site")

	// Server flags
	host := fs.String("host", "", "Browser listen address")
	port := fs.Int("port", 0, "Browser listen port")
	socket := fs.String("socket", "", "Backend API socket path")

	// Storage flags
	storage := fs.String("storage", "", "Storage type: memory, sqlite, postgresql")
	storagePath := fs.String("storage-path", "", "SQLite database path")
	storageURL := fs.String("storage-url", "", "PostgreSQL connection URL")

	// Lua flags
	lua := fs.Bool("lua", true, "Enable Lua backend")
	luaPath := fs.String("lua-path", "", "Lua scripts directory")

	// Session flags
	sessionTimeout := fs.Duration("session-timeout", 0, "Session expiration (0=never)")

	// Logging flags
	logLevel := fs.String("log-level", "", "Log level: debug, info, warn, error")
	var verbosity verbosityCounter
	fs.Var(&verbosity, "v", "Verbosity level (use -v, -vv, or -vvv)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Load TOML config if exists (from config/ subdirectory)
	configPath := "config/config.toml"
	if *dir != "" {
		configPath = *dir + "/config/config.toml"
	}
	if err := cfg.loadTOML(configPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Apply environment variables
	cfg.applyEnv()

	// Apply CLI flags (highest priority)
	if *host != "" {
		cfg.Server.Host = *host
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *socket != "" {
		cfg.Server.Socket = *socket
	}
	if *storage != "" {
		cfg.Storage.Type = *storage
	}
	if *storagePath != "" {
		cfg.Storage.Path = *storagePath
	}
	if *storageURL != "" {
		cfg.Storage.URL = *storageURL
	}
	if fs.Lookup("lua").Value.String() != "true" {
		cfg.Lua.Enabled = *lua
	}
	if *luaPath != "" {
		cfg.Lua.Path = *luaPath
	}
	if *sessionTimeout != 0 {
		cfg.Session.Timeout = Duration(*sessionTimeout)
	}
	if *logLevel != "" {
		cfg.Logging.Level = *logLevel
	}
	if verbosity > 0 {
		cfg.Logging.Verbosity = int(verbosity)
	}

	// Store dir in config (not from TOML, only CLI)
	cfg.Server.Dir = *dir

	return cfg, nil
}

// loadTOML loads configuration from a TOML file.
func (c *Config) loadTOML(path string) error {
	_, err := toml.DecodeFile(path, c)
	return err
}

// applyEnv applies environment variable overrides.
func (c *Config) applyEnv() {
	if v := os.Getenv("UI_HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv("UI_PORT"); v != "" {
		var port int
		if _, err := parseEnvInt(v, &port); err == nil {
			c.Server.Port = port
		}
	}
	if v := os.Getenv("UI_SOCKET"); v != "" {
		c.Server.Socket = v
	}
	if v := os.Getenv("UI_STORAGE"); v != "" {
		c.Storage.Type = v
	}
	if v := os.Getenv("UI_STORAGE_PATH"); v != "" {
		c.Storage.Path = v
	}
	if v := os.Getenv("UI_STORAGE_URL"); v != "" {
		c.Storage.URL = v
	}
	if v := os.Getenv("UI_LUA"); v != "" {
		c.Lua.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("UI_LUA_PATH"); v != "" {
		c.Lua.Path = v
	}
	if v := os.Getenv("UI_SESSION_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Session.Timeout = Duration(d)
		}
	}
	if v := os.Getenv("UI_LOG_LEVEL"); v != "" {
		c.Logging.Level = v
	}
	if v := os.Getenv("UI_VERBOSITY"); v != "" {
		if verbosity, err := strconv.Atoi(v); err == nil {
			c.Logging.Verbosity = verbosity
		}
	}
}

// parseEnvInt parses an environment variable as an integer.
func parseEnvInt(s string, result *int) (bool, error) {
	var v int
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		v = v*10 + int(c-'0')
	}
	*result = v
	return true, nil
}

// Verbosity returns the configured verbosity level (0-3).
func (c *Config) Verbosity() int {
	return c.Logging.Verbosity
}
