// Copyright © 2026 The Homeport Team
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"

	"github.com/permhoot/pennypinch/internal/color"
	"github.com/permhoot/pennypinch/internal/model"
)

// Settings holds application-level settings.
type Settings struct {
	Theme            string `yaml:"theme" json:"theme"`
	CurrencySymbol   string `yaml:"currency_symbol" json:"currency_symbol"`
	CurrencyPosition string `yaml:"currency_position" json:"currency_position"`
	CurrencyInCells  bool   `yaml:"currency_in_cells" json:"currency_in_cells"`
	DateFormat       string `yaml:"date_format" json:"date_format"`
}

// Config is the root of config.yaml.
type Config struct {
	Categories []model.Category `yaml:"categories" json:"categories"`
	Settings   Settings         `yaml:"settings" json:"settings"`
}

// DefaultConfig returns a Config with sane defaults and an empty category list.
func DefaultConfig() *Config {
	return &Config{
		Categories: []model.Category{},
		Settings: Settings{
			Theme:            "dark",
			CurrencySymbol:   "$",
			CurrencyPosition: "left",
			CurrencyInCells:  true,
			DateFormat:       "YYYY-MM-DD",
		},
	}
}

// Manager owns the in-memory config, handles persistence (atomic writes) and
// file watching (fsnotify) for live reload.
type Manager struct {
	mu      sync.RWMutex
	writeMu sync.Mutex
	path    string
	config  *Config
	watcher *fsnotify.Watcher
}

// LoadManager reads config from path, creating a default file if missing, and
// starts a file watcher for live reload. Returns the manager and the loaded
// config. The watcher must be closed via Close.
func LoadManager(path string) (*Manager, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	cfg, err := loadOrCreate(path)
	if err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}
	if err := watcher.Add(path); err != nil {
		_ = watcher.Close()
		return nil, fmt.Errorf("watch config: %w", err)
	}

	m := &Manager{
		path:    path,
		config:  cfg,
		watcher: watcher,
	}
	go m.watchLoop()
	return m, nil
}

func loadOrCreate(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // Path from CLI flag; acceptable for self-hosted homelab tool
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if err := writeAtomic(path, cfg); err != nil {
				return nil, fmt.Errorf("create default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Categories == nil {
		cfg.Categories = []model.Category{}
	}
	if cfg.Settings.Theme == "" {
		cfg.Settings.Theme = "dark"
	}
	if cfg.Settings.CurrencySymbol == "" {
		cfg.Settings.CurrencySymbol = "$"
	}
	if cfg.Settings.CurrencyPosition == "" {
		cfg.Settings.CurrencyPosition = "left"
	}
	if cfg.Settings.DateFormat == "" {
		cfg.Settings.DateFormat = "YYYY-MM-DD"
	}
	return cfg, nil
}

func writeAtomic(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

func (m *Manager) watchLoop() {
	for {
		select {
		case ev, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) != 0 {
				m.reload()
			}
		case _, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func (m *Manager) reload() {
	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	cfg, err := loadOrCreate(m.path)
	if err != nil {
		return
	}

	m.mu.Lock()
	m.config = cfg
	m.mu.Unlock()
}

// Close stops the file watcher.
func (m *Manager) Close() error {
	if m.watcher != nil {
		return m.watcher.Close()
	}
	return nil
}

// Snapshot returns a copy of the current config.
func (m *Manager) Snapshot() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c := *m.config
	c.Categories = append([]model.Category(nil), m.config.Categories...)
	return &c
}

// RestoreCategories replaces the category list with the given snapshot and
// atomically persists it. Used to roll back a failed import.
func (m *Manager) RestoreCategories(categories []model.Category) error {
	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.Categories = append([]model.Category(nil), categories...)
	return writeAtomic(m.path, m.config)
}

// EnsureCategories adds any missing category names to the config, assigning
// the next sequential palette color, and atomically persists the config. It
// returns the newly added categories (in order).
func (m *Manager) EnsureCategories(names []string) ([]model.Category, error) {
	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	existing := make(map[string]bool, len(m.config.Categories))
	for _, c := range m.config.Categories {
		existing[c.Name] = true
	}
	origLen := len(m.config.Categories)

	added := make([]model.Category, 0, len(names))
	for _, n := range names {
		if existing[n] {
			continue
		}
		color := color.NextPaletteColor(origLen + len(added))
		cat := model.Category{Name: n, Color: color}
		m.config.Categories = append(m.config.Categories, cat)
		existing[n] = true
		added = append(added, cat)
	}

	if len(added) == 0 {
		return added, nil
	}

	if err := writeAtomic(m.path, m.config); err != nil {
		return nil, err
	}
	return added, nil
}
