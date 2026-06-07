package lyn

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	defaultHotkeyBinding     = "Ctrl+Space"
	defaultMaxDepth          = 5
	defaultScannerTimeout    = Duration("20s")
	defaultTheme             = "power-run"
	defaultBackgroundOpacity = 0.98
	defaultWindowPlacement   = "center"
	defaultWorkspaceShortcut = "{"
)

var defaultScannerRoots = []string{"~/src", "~/www"}

type Duration string

type Config struct {
	Path    string        `json:"path"`
	Hotkey  HotkeyConfig  `json:"hotkey"`
	Cache   CacheConfig   `json:"cache"`
	Scanner ScannerConfig `json:"scanner"`
	UI      UIConfig      `json:"ui"`
	Startup StartupConfig `json:"startup"`
}

type HotkeyConfig struct {
	Binding string `json:"binding"`
}

type CacheConfig struct {
	Dir string `json:"dir"`
}

type ScannerConfig struct {
	Roots       []string `json:"roots"`
	MaxDepth    int      `json:"maxDepth"`
	Concurrency int      `json:"concurrency"`
	Timeout     Duration `json:"timeout"`
	Watch       bool     `json:"watch"`
}

type UIConfig struct {
	Theme                  string  `json:"theme"`
	BackgroundOpacity      float64 `json:"backgroundOpacity"`
	WindowPlacement        string  `json:"windowPlacement"`
	ClearQueryOnShow       bool    `json:"clearQueryOnShow"`
	WorkspaceQueryShortcut string  `json:"workspaceQueryShortcut"`
}

type StartupConfig struct {
	Enabled     bool `json:"enabled"`
	StartHidden bool `json:"startHidden"`
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		path = ConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg.Path = path
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	cfg.Path = path
	cfg, err = NormalizeConfig(cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

func SaveConfig(cfg Config) (Config, error) {
	cfg, err := NormalizeConfig(cfg)
	if err != nil {
		return cfg, err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o755); err != nil {
		return cfg, err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return cfg, err
	}
	data = append(data, '\n')
	return cfg, os.WriteFile(cfg.Path, data, 0o644)
}

func ConfigPath() string {
	dir, err := os.UserConfigDir()
	if err == nil && dir != "" {
		return filepath.Join(dir, "lyn", "lyn.json")
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".config", "lyn", "lyn.json")
	}
	return filepath.Join(".config", "lyn", "lyn.json")
}

func DefaultConfig() Config {
	return Config{
		Path:   ConfigPath(),
		Hotkey: HotkeyConfig{Binding: defaultHotkeyBinding},
		Cache:  CacheConfig{Dir: defaultCacheDir()},
		Scanner: ScannerConfig{
			Roots:       append([]string(nil), defaultScannerRoots...),
			MaxDepth:    defaultMaxDepth,
			Concurrency: runtime.NumCPU(),
			Timeout:     defaultScannerTimeout,
			Watch:       true,
		},
		UI: UIConfig{
			Theme:                  defaultTheme,
			BackgroundOpacity:      defaultBackgroundOpacity,
			WindowPlacement:        defaultWindowPlacement,
			ClearQueryOnShow:       true,
			WorkspaceQueryShortcut: defaultWorkspaceShortcut,
		},
		Startup: StartupConfig{Enabled: false, StartHidden: true},
	}
}

func NormalizeConfig(cfg Config) (Config, error) {
	if cfg.Path == "" {
		cfg.Path = ConfigPath()
	}
	if cfg.Cache.Dir == "" {
		cfg.Cache.Dir = defaultCacheDir()
	}
	if cfg.Scanner.MaxDepth < 1 {
		cfg.Scanner.MaxDepth = defaultMaxDepth
	}
	if cfg.Scanner.Concurrency < 1 {
		cfg.Scanner.Concurrency = runtime.NumCPU()
	}
	if cfg.Scanner.Timeout == "" {
		cfg.Scanner.Timeout = defaultScannerTimeout
	}
	if len(cfg.Scanner.Roots) == 0 {
		cfg.Scanner.Roots = append([]string(nil), defaultScannerRoots...)
	}
	if cfg.Hotkey.Binding == "" {
		cfg.Hotkey.Binding = defaultHotkeyBinding
	}
	if cfg.UI.Theme == "" {
		cfg.UI.Theme = defaultTheme
	}
	if cfg.UI.BackgroundOpacity <= 0 || cfg.UI.BackgroundOpacity > 1 {
		cfg.UI.BackgroundOpacity = defaultBackgroundOpacity
	}
	cfg.UI.WindowPlacement = defaultWindowPlacement
	cfg.UI.WorkspaceQueryShortcut = normalizeShortcutCharacter(cfg.UI.WorkspaceQueryShortcut, defaultWorkspaceShortcut)
	if cfg.Scanner.Timeout.Duration() <= 0 {
		return cfg, errors.New("scanner timeout must be positive")
	}
	return cfg, nil
}

func normalizeShortcutCharacter(value string, fallback string) string {
	for _, char := range value {
		if char == ' ' || char == '\t' || char == '\n' || char == '\r' {
			continue
		}
		return string(char)
	}
	return fallback
}

func (d Duration) Duration() time.Duration {
	value, err := time.ParseDuration(string(d))
	if err != nil {
		return 0
	}
	return value
}

func defaultCacheDir() string {
	return filepath.Join(userCacheDir(), "lyn")
}

func userCacheDir() string {
	dir, err := os.UserCacheDir()
	if err == nil && dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".cache")
	}
	return ".cache"
}
