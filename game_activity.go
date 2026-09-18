package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type gameProcess struct {
	PID         uint32 `json:"pid"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	WindowTitle string `json:"windowTitle"`
}

type gameActivityEntry struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Source   string    `json:"source"`
	LastSeen time.Time `json:"lastSeen"`
}

type gameActivityConfig struct {
	DetectionEnabled bool                `json:"detectionEnabled"`
	GamesSeen        []gameActivityEntry `json:"gamesSeen"`
	Overrides        map[string]string   `json:"overrides"`
}

type gameActivityState struct {
	DetectionEnabled bool                `json:"detectionEnabled"`
	RunningGames     []gameProcess       `json:"runningGames"`
	GamesSeen        []gameActivityEntry `json:"gamesSeen"`
	Overrides        map[string]string   `json:"overrides"`
}

type bridgeGameActivityAddArgs struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type bridgeGameActivityPathArgs struct {
	Path string `json:"path"`
}

type bridgeGameActivityDetectionArgs struct {
	Enabled bool `json:"enabled"`
}

var gameActivityMu sync.Mutex
var gameActivityRuntimeMu sync.RWMutex
var gameActivityRunning []gameProcess

func startGameActivityObserver() {
	go func() {
		for {
			refreshGameActivityProcesses()
			time.Sleep(5 * time.Second)
		}
	}()
}

func cachedGameActivityProcesses() []gameProcess {
	gameActivityRuntimeMu.RLock()
	defer gameActivityRuntimeMu.RUnlock()
	return append([]gameProcess(nil), gameActivityRunning...)
}

func setCachedGameActivityProcesses(processes []gameProcess) {
	gameActivityRuntimeMu.Lock()
	gameActivityRunning = append([]gameProcess(nil), processes...)
	gameActivityRuntimeMu.Unlock()
}

func refreshGameActivityProcesses() {
	gameActivityMu.Lock()
	config, err := loadGameActivityConfig()
	gameActivityMu.Unlock()
	if err != nil {
		return
	}
	if !config.DetectionEnabled {
		setCachedGameActivityProcesses(nil)
		return
	}
	candidates := make([]string, 0, len(config.GamesSeen))
	for _, entry := range config.GamesSeen {
		candidates = append(candidates, entry.Path)
	}
	running, err := enumerateGameProcesses(candidates...)
	if err != nil {
		return
	}
	setCachedGameActivityProcesses(running)
	gameActivityMu.Lock()
	defer gameActivityMu.Unlock()
	config, err = loadGameActivityConfig()
	if err != nil || !config.DetectionEnabled {
		return
	}
	changed := false
	for _, process := range running {
		id := normalizeGamePath(process.Path)
		if id == "" || containsGameEntry(config.GamesSeen, id) {
			continue
		}
		name := process.Name
		if override := config.Overrides[id]; override != "" {
			name = override
		}
		config.GamesSeen = append(config.GamesSeen, gameActivityEntry{ID: id, Name: name, Path: process.Path, Source: "detected", LastSeen: time.Now()})
		changed = true
	}
	if changed {
		_ = saveGameActivityConfig(config)
	}
}

func gameActivityPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "Moreno", "DiscordLite")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "game-activity.json"), nil
}

func loadGameActivityConfig() (gameActivityConfig, error) {
	config := gameActivityConfig{DetectionEnabled: true, Overrides: map[string]string{}}
	filePath, err := gameActivityPath()
	if err != nil {
		return config, err
	}
	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return config, nil
	}
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	if config.Overrides == nil {
		config.Overrides = map[string]string{}
	}
	return config, nil
}

func saveGameActivityConfig(config gameActivityConfig) error {
	filePath, err := gameActivityPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0600)
}

func normalizeGamePath(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "\"")
	if value == "" {
		return ""
	}
	return strings.ToLower(filepath.Clean(value))
}

func gameDisplayName(value string) string {
	name := strings.TrimSpace(value)
	if name == "" {
		name = filepath.Base(value)
	}
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func (d *discordApp) gameActivityState() (gameActivityState, error) {
	gameActivityMu.Lock()
	defer gameActivityMu.Unlock()
	config, err := loadGameActivityConfig()
	if err != nil {
		return gameActivityState{}, err
	}
	running := []gameProcess{}
	if config.DetectionEnabled {
		running = cachedGameActivityProcesses()
	}
	config.GamesSeen = deduplicateGameEntries(config.GamesSeen, config.Overrides)
	changed := false
	for i := range running {
		id := normalizeGamePath(running[i].Path)
		if id == "" {
			continue
		}
		if override := config.Overrides[id]; override != "" {
			running[i].Name = override
		}
		if !containsGameEntry(config.GamesSeen, id) {
			config.GamesSeen = append(config.GamesSeen, gameActivityEntry{ID: id, Name: running[i].Name, Path: running[i].Path, Source: "detected", LastSeen: time.Now()})
			changed = true
		}
	}
	if changed {
		if err := saveGameActivityConfig(config); err != nil {
			return gameActivityState{}, err
		}
	}
	sort.Slice(config.GamesSeen, func(i, j int) bool {
		return strings.ToLower(config.GamesSeen[i].Name) < strings.ToLower(config.GamesSeen[j].Name)
	})
	return gameActivityState{DetectionEnabled: config.DetectionEnabled, RunningGames: running, GamesSeen: config.GamesSeen, Overrides: config.Overrides}, nil
}

func deduplicateGameEntries(entries []gameActivityEntry, overrides map[string]string) []gameActivityEntry {
	result := make([]gameActivityEntry, 0, len(entries))
	seen := map[string]bool{}
	for _, entry := range entries {
		entry.ID = normalizeGamePath(entry.Path)
		if entry.ID == "" || seen[entry.ID] {
			continue
		}
		seen[entry.ID] = true
		if override := overrides[entry.ID]; override != "" {
			entry.Name = override
		}
		if entry.Name == "" {
			entry.Name = gameDisplayName(entry.Path)
		}
		result = append(result, entry)
	}
	return result
}

func containsGameEntry(entries []gameActivityEntry, id string) bool {
	for _, entry := range entries {
		if normalizeGamePath(entry.Path) == id {
			return true
		}
	}
	return false
}

func (d *discordApp) addGameActivity(window application.Window, args bridgeGameActivityAddArgs) (gameActivityState, error) {
	gameActivityMu.Lock()
	defer gameActivityMu.Unlock()
	pathName := strings.TrimSpace(args.Path)
	if pathName == "" {
		if d.app == nil {
			return gameActivityState{}, fmt.Errorf("application is not ready")
		}
		selected, err := d.app.Dialog.OpenFile().SetTitle("Add a registered game").SetMessage("Select the game's executable").CanChooseFiles(true).CanChooseDirectories(false).AddFilter("Windows executables", "*.exe").AllowsOtherFileTypes(false).AttachToWindow(window).PromptForSingleSelection()
		if err != nil {
			return gameActivityState{}, err
		}
		pathName = selected
	}
	pathName = strings.Trim(strings.TrimSpace(pathName), "\"")
	info, err := os.Stat(pathName)
	if err != nil {
		return gameActivityState{}, err
	}
	if info.IsDir() || !strings.EqualFold(filepath.Ext(pathName), ".exe") {
		return gameActivityState{}, fmt.Errorf("select a Windows executable")
	}
	config, err := loadGameActivityConfig()
	if err != nil {
		return gameActivityState{}, err
	}
	id := normalizeGamePath(pathName)
	name := gameDisplayName(args.Name)
	if args.Name == "" {
		name = gameDisplayName(pathName)
	}
	config.Overrides[id] = name
	config.GamesSeen = deduplicateGameEntries(append(config.GamesSeen, gameActivityEntry{ID: id, Name: name, Path: pathName, Source: "manual", LastSeen: time.Now()}), config.Overrides)
	if err := saveGameActivityConfig(config); err != nil {
		return gameActivityState{}, err
	}
	return d.gameActivityStateUnlocked(config)
}

func (d *discordApp) removeGameActivity(pathName string) (gameActivityState, error) {
	gameActivityMu.Lock()
	defer gameActivityMu.Unlock()
	config, err := loadGameActivityConfig()
	if err != nil {
		return gameActivityState{}, err
	}
	id := normalizeGamePath(pathName)
	delete(config.Overrides, id)
	filtered := make([]gameActivityEntry, 0, len(config.GamesSeen))
	for _, entry := range config.GamesSeen {
		if normalizeGamePath(entry.Path) != id {
			filtered = append(filtered, entry)
		}
	}
	config.GamesSeen = filtered
	if err := saveGameActivityConfig(config); err != nil {
		return gameActivityState{}, err
	}
	return d.gameActivityStateUnlocked(config)
}

func (d *discordApp) setGameActivityDetection(enabled bool) (gameActivityState, error) {
	gameActivityMu.Lock()
	defer gameActivityMu.Unlock()
	config, err := loadGameActivityConfig()
	if err != nil {
		return gameActivityState{}, err
	}
	config.DetectionEnabled = enabled
	if err := saveGameActivityConfig(config); err != nil {
		return gameActivityState{}, err
	}
	return d.gameActivityStateUnlocked(config)
}

func (d *discordApp) gameActivityStateUnlocked(config gameActivityConfig) (gameActivityState, error) {
	running := []gameProcess{}
	if config.DetectionEnabled {
		running = cachedGameActivityProcesses()
	}
	for i := range running {
		if override := config.Overrides[normalizeGamePath(running[i].Path)]; override != "" {
			running[i].Name = override
		}
	}
	return gameActivityState{DetectionEnabled: config.DetectionEnabled, RunningGames: running, GamesSeen: deduplicateGameEntries(config.GamesSeen, config.Overrides), Overrides: config.Overrides}, nil
}
