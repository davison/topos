// configpath_test.go pins the config-path precedence chain 14-01-PLAN.md
// Task 1 adds: --config flag > TOPOS_CONFIG env var > $XDG_CONFIG_HOME >
// $HOME/.config (configPath()'s own unchanged fallback chain). Every case
// uses t.Setenv for TOPOS_CONFIG, XDG_CONFIG_HOME and HOME so no case ever
// reaches the developer's real environment, and drives resolveConfigPath
// directly — no config file, no index, no plugin subprocess.
package main

import (
	"path/filepath"
	"testing"
)

func TestResolveConfigPath_Precedence(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		toposEnv  string
		xdgEnv    string
		home      string
		want      string
	}{
		{
			name:      "flag beats env",
			flagValue: "/from/flag/config.toml",
			toposEnv:  "/from/env/config.toml",
			xdgEnv:    "/from/xdg",
			home:      "/home/testuser",
			want:      "/from/flag/config.toml",
		},
		{
			name:      "flag beats XDG",
			flagValue: "/from/flag/config.toml",
			toposEnv:  "",
			xdgEnv:    "/from/xdg",
			home:      "/home/testuser",
			want:      "/from/flag/config.toml",
		},
		{
			name:      "env beats XDG",
			flagValue: "",
			toposEnv:  "/from/env/config.toml",
			xdgEnv:    "/from/xdg",
			home:      "/home/testuser",
			want:      "/from/env/config.toml",
		},
		{
			name:      "XDG beats home fallback",
			flagValue: "",
			toposEnv:  "",
			xdgEnv:    "/from/xdg",
			home:      "/home/testuser",
			want:      filepath.Join("/from/xdg", "topos", "config.toml"),
		},
		{
			name:      "empty TOPOS_CONFIG treated as unset, falls through to XDG",
			flagValue: "",
			toposEnv:  "",
			xdgEnv:    "/from/xdg",
			home:      "/home/testuser",
			want:      filepath.Join("/from/xdg", "topos", "config.toml"),
		},
		{
			name:      "neither flag, env, nor XDG set falls to home fallback",
			flagValue: "",
			toposEnv:  "",
			xdgEnv:    "",
			home:      "/home/testuser",
			want:      filepath.Join("/home/testuser", ".config", "topos", "config.toml"),
		},
		{
			name:      "relative flag value is returned unchanged",
			flagValue: "config.dev.toml",
			toposEnv:  "",
			xdgEnv:    "/from/xdg",
			home:      "/home/testuser",
			want:      "config.dev.toml",
		},
		{
			name:      "absolute flag value is returned unchanged",
			flagValue: "/abs/path/config.toml",
			toposEnv:  "",
			xdgEnv:    "/from/xdg",
			home:      "/home/testuser",
			want:      "/abs/path/config.toml",
		},
		{
			name:      "relative TOPOS_CONFIG value is returned unchanged",
			flagValue: "",
			toposEnv:  "config.dev.toml",
			xdgEnv:    "/from/xdg",
			home:      "/home/testuser",
			want:      "config.dev.toml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TOPOS_CONFIG", tt.toposEnv)
			t.Setenv("XDG_CONFIG_HOME", tt.xdgEnv)
			t.Setenv("HOME", tt.home)

			got := resolveConfigPath(tt.flagValue)
			if got != tt.want {
				t.Errorf("resolveConfigPath(%q) = %q, want %q", tt.flagValue, got, tt.want)
			}
		})
	}
}
