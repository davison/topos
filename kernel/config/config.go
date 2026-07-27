package config

import (
	"fmt"
	"os"
	"os/user"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Load reads, expands, decodes and validates the config file at path.
//
// Secrets are never templated: the raw file bytes are expanded with
// os.Expand (${VAR} / $VAR references) before TOML decoding, and the
// expanded result is held in memory only — never written back to disk.
// A referenced-but-unset environment variable expands to the empty string
// and is reported by name in the returned error, not silently accepted.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	expanded, missing := expandEnv(string(raw))

	var cfg Config
	if err := toml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	applyDefaults(&cfg)

	if err := cfg.expandIndexPathHome(); err != nil {
		return nil, err
	}

	if err := cfg.Validate(missing); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// expandEnv expands ${VAR}/$VAR references in raw using the current
// environment, and returns the sorted, de-duplicated list of variable
// names that were referenced but not set (so callers can name them in
// validation errors instead of silently treating them as empty strings).
func expandEnv(raw string) (string, []string) {
	missingSet := map[string]struct{}{}
	expanded := os.Expand(raw, func(name string) string {
		v, ok := os.LookupEnv(name)
		if !ok {
			missingSet[name] = struct{}{}
		}
		return v
	})

	missing := make([]string, 0, len(missingSet))
	for name := range missingSet {
		missing = append(missing, name)
	}
	sort.Strings(missing)

	return expanded, missing
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = DefaultListen
	}
	if cfg.Index.Path == "" {
		cfg.Index.Path = DefaultIndexPath
	}
	if cfg.Plugins.Dir == "" {
		cfg.Plugins.Dir = DefaultPluginsDir
	}
}

// expandIndexPathHome expands a leading "~" in [index] path to the current
// user's home directory.
func (cfg *Config) expandIndexPathHome() error {
	if !strings.HasPrefix(cfg.Index.Path, "~") {
		return nil
	}
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("config: resolve home directory for index path: %w", err)
	}
	cfg.Index.Path = strings.Replace(cfg.Index.Path, "~", u.HomeDir, 1)
	return nil
}

// Validate checks structural correctness of the decoded config. missing is
// the list of environment variable names referenced in the raw file but
// unset, as returned by expandEnv — used to name the culprit when a
// required field is empty because of an unset variable rather than a
// simply-omitted key.
func (cfg *Config) Validate(missing []string) error {
	for name, ws := range cfg.Webspaces {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("config: webspace has empty name")
		}
		if len(ws.Keywords) == 0 {
			return fmt.Errorf("config: webspace %q declares zero keywords", name)
		}
		for _, kw := range ws.Keywords {
			if strings.TrimSpace(kw) == "" {
				return fmt.Errorf("config: webspace %q declares an empty or whitespace-only keyword", name)
			}
		}
	}

	for name, src := range cfg.Sources {
		if strings.TrimSpace(src.BaseURL) == "" {
			return fmt.Errorf("config: source %q has empty base_url%s", name, missingSuffix(missing))
		}
		if strings.TrimSpace(src.Token) == "" {
			return fmt.Errorf("config: source %q has empty token%s", name, missingSuffix(missing))
		}
	}

	return nil
}

func missingSuffix(missing []string) string {
	if len(missing) == 0 {
		return ""
	}
	return fmt.Sprintf(" (missing environment variable(s): %s)", strings.Join(missing, ", "))
}
