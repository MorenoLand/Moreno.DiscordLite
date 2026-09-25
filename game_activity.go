package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type gameProcess struct {
	PID           uint32 `json:"pid"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	WindowTitle   string `json:"windowTitle"`
	Start         int64  `json:"start,omitempty"`
	ApplicationID string `json:"applicationId,omitempty"`
}

type detectableExecutable struct {
	Name       string `json:"name"`
	OS         string `json:"os"`
	Arguments  string `json:"arguments"`
	IsLauncher bool   `json:"isLauncher"`
}

type detectableApp struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Executables []detectableExecutable `json:"executables"`
}

type detectableExclusions struct {
	Executables []string `json:"executables"`
	Patterns    []string `json:"patterns"`
}

type gameCandidate struct {
	ApplicationID string
	Name          string
	Executable    string
	Arguments     string
}

type gameCandidateIndex map[string][]gameCandidate

var (
	cachedDetectableMu       sync.RWMutex
	cachedDetectableApps     []detectableApp
	cachedCandidateIndex     gameCandidateIndex
	cachedNonGameIDs         map[string]bool
	cachedBlockedExecutables map[string]bool
	cachedBlockedPatterns    []*regexp.Regexp
	detectableCatalogReady   bool
)

func detectableCachePath(name string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "Moreno", "DiscordLite")
	_ = os.MkdirAll(dir, 0700)
	return filepath.Join(dir, name), nil
}

func initDetectableCache() {
	go func() {
		client := &http.Client{Timeout: 20 * time.Second}
		resources := []struct {
			name string
			urls []string
			set  func([]byte) bool
		}{
			{"detectable-games-v1.json", []string{"https://cdn.discordapp.com/detectables/games-v1.json", "https://discord.com/api/v9/applications/detectable"}, parseAndSetDetectable},
			{"detectable-non-games-v1.json", []string{"https://cdn.discordapp.com/detectables/non-games-v1.json", "https://discord.com/api/v9/applications/non-games/detectable"}, parseAndSetNonGames},
			{"detectable-exclusions.json", []string{"https://discord.com/api/v9/games/detectable/exclusions"}, parseAndSetDetectableExclusions},
		}
		for _, resource := range resources {
			cachePath, err := detectableCachePath(resource.name)
			if err != nil {
				continue
			}
			if data, err := os.ReadFile(cachePath); err == nil {
				resource.set(data)
			}
			info, err := os.Stat(cachePath)
			if err == nil && time.Since(info.ModTime()) < 24*time.Hour {
				continue
			}
			for _, url := range resource.urls {
				resp, err := client.Get(url)
				if err != nil {
					continue
				}
				body, readErr := io.ReadAll(resp.Body)
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK && readErr == nil && len(body) > 0 && resource.set(body) {
					_ = os.WriteFile(cachePath, body, 0600)
					break
				}
			}
		}
	}()
}

func parseAndSetDetectable(data []byte) bool {
	var apps []detectableApp
	if err := json.Unmarshal(data, &apps); err != nil || len(apps) == 0 {
		return false
	}
	cachedDetectableMu.Lock()
	cachedDetectableApps = apps
	detectableCatalogReady = true
	rebuildCachedCandidateIndex()
	cachedDetectableMu.Unlock()
	return true
}

func parseAndSetNonGames(data []byte) bool {
	var apps []detectableApp
	if err := json.Unmarshal(data, &apps); err != nil || len(apps) == 0 {
		return false
	}
	ids := make(map[string]bool, len(apps))
	for _, app := range apps {
		if app.ID != "" {
			ids[app.ID] = true
		}
	}
	cachedDetectableMu.Lock()
	cachedNonGameIDs = ids
	rebuildCachedCandidateIndex()
	cachedDetectableMu.Unlock()
	return true
}

func parseAndSetDetectableExclusions(data []byte) bool {
	var exclusions detectableExclusions
	if err := json.Unmarshal(data, &exclusions); err != nil || len(exclusions.Executables)+len(exclusions.Patterns) == 0 {
		return false
	}
	executables := make(map[string]bool, len(exclusions.Executables))
	for _, executable := range exclusions.Executables {
		executables[normalizeCandidateExecutable(executable)] = true
	}
	patterns := make([]*regexp.Regexp, 0, len(exclusions.Patterns))
	for _, pattern := range exclusions.Patterns {
		if compiled, err := regexp.Compile("(?i)" + pattern); err == nil {
			patterns = append(patterns, compiled)
		}
	}
	cachedDetectableMu.Lock()
	cachedBlockedExecutables = executables
	cachedBlockedPatterns = patterns
	cachedDetectableMu.Unlock()
	return true
}

func rebuildCachedCandidateIndex() {
	candidates := make(gameCandidateIndex)
	for _, app := range cachedDetectableApps {
		if cachedNonGameIDs[app.ID] {
			continue
		}
		for _, executable := range app.Executables {
			if (strings.EqualFold(executable.OS, "win32") || executable.OS == "") && executable.Name != "" {
				candidate := gameCandidate{ApplicationID: app.ID, Name: app.Name, Executable: executable.Name, Arguments: executable.Arguments}
				name := filepath.Base(normalizeCandidateExecutable(executable.Name))
				candidates[name] = append(candidates[name], candidate)
			}
		}
	}
	cachedCandidateIndex = candidates
}

func cachedGameCandidates() gameCandidateIndex {
	cachedDetectableMu.RLock()
	defer cachedDetectableMu.RUnlock()
	return cachedCandidateIndex
}

func executableMatchesPath(pathName, executable string) bool {
	pathName = normalizeGamePath(pathName)
	executable = normalizeCandidateExecutable(executable)
	if pathName == "" || executable == "" {
		return false
	}
	if strings.Contains(executable, `\`) {
		return pathName == executable || strings.HasSuffix(pathName, `\`+executable)
	}
	return strings.EqualFold(filepath.Base(pathName), executable)
}

func normalizeCandidateExecutable(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(value, ">"))
	value = strings.ReplaceAll(value, "/", `\`)
	value = strings.TrimLeft(value, `\`)
	return strings.ToLower(value)
}

func candidateIsAllowed(candidate gameCandidate, pathName string) bool {
	cachedDetectableMu.RLock()
	defer cachedDetectableMu.RUnlock()
	if cachedNonGameIDs[candidate.ApplicationID] {
		return false
	}
	name := normalizeCandidateExecutable(candidate.Executable)
	path := normalizeGamePath(pathName)
	if cachedBlockedExecutables[name] || cachedBlockedExecutables[path] || cachedBlockedExecutables[strings.ToLower(filepath.Base(path))] {
		return false
	}
	for _, pattern := range cachedBlockedPatterns {
		if pattern.MatchString(path) || pattern.MatchString(filepath.Base(path)) || pattern.MatchString(name) {
			return false
		}
	}
	return true
}

func catalogContainsPath(pathName string, candidates gameCandidateIndex) bool {
	for _, candidate := range candidates[strings.ToLower(filepath.Base(normalizeGamePath(pathName)))] {
		if executableMatchesPath(pathName, candidate.Executable) && candidateIsAllowed(candidate, pathName) {
			return true
		}
	}
	return false
}

func gameActivityCandidates(config gameActivityConfig) (gameCandidateIndex, map[string]string) {
	manual := make(map[string]string)
	for _, entry := range config.GamesSeen {
		id := normalizeGamePath(entry.Path)
		if id == "" || config.IgnoredGames[id] {
			continue
		}
		if entry.Source == "manual" || config.Overrides[id] != "" {
			name := config.Overrides[id]
			if name == "" {
				name = entry.Name
			}
			manual[id] = name
		}
	}
	return cachedGameCandidates(), manual
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
	DisabledGames    map[string]bool     `json:"disabledGames"`
	IgnoredGames     map[string]bool     `json:"ignoredGames,omitempty"`
}

type gameActivityState struct {
	DetectionEnabled bool                `json:"detectionEnabled"`
	RunningGames     []gameProcess       `json:"runningGames"`
	GamesSeen        []gameActivityEntry `json:"gamesSeen"`
	Overrides        map[string]string   `json:"overrides"`
	DisabledGames    map[string]bool     `json:"disabledGames"`
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

type bridgeGameActivityToggleGameArgs struct {
	Path    string `json:"path"`
	Enabled bool   `json:"enabled"`
}

var gameActivityMu sync.Mutex
var gameActivityRuntimeMu sync.RWMutex
var gameActivityRunning []gameProcess

func (d *discordApp) startGameActivityObserver() {
	d.gameActivityOnce.Do(func() {
		go func() {
			wasRunning := false
			for {
				if d.window != nil {
					if state, err := d.gameActivityState(); err == nil {
						encoded, _ := json.Marshal(state)
						isRunning := len(state.RunningGames) > 0
						processExit := wasRunning && !isRunning
						application.InvokeAsync(func() {
							if d.window == nil {
								return
							}
							d.window.ExecJS("if(window.__vcGameActivityApply)window.__vcGameActivityApply(" + string(encoded) + ");")
							if processExit {
								d.window.ExecJS("if(window.__vcGameActivityProcessExit)window.__vcGameActivityProcessExit();")
							}
						})
						wasRunning = isRunning
					}
				}
				time.Sleep(3 * time.Second)
			}
		}()
	})
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

func refreshGameActivityProcesses() []gameProcess {
	gameActivityMu.Lock()
	config, err := loadGameActivityConfig()
	gameActivityMu.Unlock()
	if err != nil {
		return nil
	}
	if !config.DetectionEnabled {
		setCachedGameActivityProcesses(nil)
		return nil
	}
	candidates, manual := gameActivityCandidates(config)
	running, err := enumerateGameProcesses(candidates, manual)
	if err != nil {
		setCachedGameActivityProcesses(nil)
		return nil
	}
	filteredRunning := make([]gameProcess, 0, len(running))
	for _, p := range running {
		pID := normalizeGamePath(p.Path)
		if config.IgnoredGames == nil || !config.IgnoredGames[pID] {
			filteredRunning = append(filteredRunning, p)
		}
	}
	running = filteredRunning
	setCachedGameActivityProcesses(running)
	gameActivityMu.Lock()
	defer gameActivityMu.Unlock()
	config, err = loadGameActivityConfig()
	if err != nil || !config.DetectionEnabled {
		return running
	}
	changed := false
	cachedDetectableMu.RLock()
	catalogReady := detectableCatalogReady
	cachedDetectableMu.RUnlock()
	if catalogReady {
		kept := config.GamesSeen[:0]
		for _, entry := range config.GamesSeen {
			id := normalizeGamePath(entry.Path)
			if entry.Source == "detected" && config.Overrides[id] == "" && !catalogContainsPath(entry.Path, candidates) {
				changed = true
				continue
			}
			kept = append(kept, entry)
		}
		config.GamesSeen = kept
	}
	for _, process := range running {
		id := normalizeGamePath(process.Path)
		if id == "" || (config.IgnoredGames != nil && config.IgnoredGames[id]) {
			continue
		}
		found := false
		for i := range config.GamesSeen {
			if normalizeGamePath(config.GamesSeen[i].Path) == id {
				found = true
				if time.Since(config.GamesSeen[i].LastSeen) > 5*time.Minute {
					config.GamesSeen[i].LastSeen = time.Now()
					changed = true
				}
				break
			}
		}
		if !found {
			name := process.Name
			if override := config.Overrides[id]; override != "" {
				name = override
			}
			config.GamesSeen = append(config.GamesSeen, gameActivityEntry{ID: id, Name: name, Path: process.Path, Source: "detected", LastSeen: time.Now()})
			changed = true
		}
	}
	if changed {
		_ = saveGameActivityConfig(config)
	}
	return running
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
	if config.IgnoredGames == nil {
		config.IgnoredGames = map[string]bool{}
	}
	cleanIgnored := map[string]bool{}
	for k, v := range config.IgnoredGames {
		if v && normalizeGamePath(k) != "" {
			cleanIgnored[normalizeGamePath(k)] = true
		}
	}
	config.IgnoredGames = cleanIgnored
	cleanDisabled := map[string]bool{}
	for k, v := range config.DisabledGames {
		if v && normalizeGamePath(k) != "" {
			cleanDisabled[normalizeGamePath(k)] = true
		}
	}
	config.DisabledGames = cleanDisabled
	needsSave := false
	for k, v := range config.Overrides {
		if strings.ContainsAny(v, `\/`) {
			config.Overrides[k] = gameDisplayName(v)
			needsSave = true
		}
	}
	filteredSeen := make([]gameActivityEntry, 0, len(config.GamesSeen))
	for _, entry := range config.GamesSeen {
		if config.IgnoredGames[normalizeGamePath(entry.Path)] {
			needsSave = true
			continue
		}
		if strings.ContainsAny(entry.Name, `\/`) {
			entry.Name = gameDisplayName(entry.Name)
			needsSave = true
		}
		filteredSeen = append(filteredSeen, entry)
	}
	config.GamesSeen = filteredSeen
	if needsSave {
		_ = saveGameActivityConfig(config)
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
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}
	_ = os.Remove(filePath)
	return os.Rename(tmpPath, filePath)
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
		return ""
	}
	base := filepath.Base(name)
	clean := strings.TrimSuffix(base, filepath.Ext(base))
	if clean == "" {
		return base
	}
	return clean
}
func (d *discordApp) gameActivityState() (gameActivityState, error) {
	refreshGameActivityProcesses()
	gameActivityMu.Lock()
	defer gameActivityMu.Unlock()
	config, err := loadGameActivityConfig()
	if err != nil {
		return gameActivityState{}, err
	}
	return d.gameActivityStateUnlocked(config)
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
	pathName := strings.TrimSpace(args.Path)
	if pathName == "" {
		if d.app == nil {
			gameActivityMu.Unlock()
			return gameActivityState{}, fmt.Errorf("application is not ready")
		}
		selected, err := d.app.Dialog.OpenFile().SetTitle("Add a registered game").SetMessage("Select the game's executable").CanChooseFiles(true).CanChooseDirectories(false).AddFilter("Windows executables", "*.exe").AllowsOtherFileTypes(false).AttachToWindow(window).PromptForSingleSelection()
		if err != nil {
			gameActivityMu.Unlock()
			return gameActivityState{}, err
		}
		pathName = selected
	}
	pathName = strings.Trim(strings.TrimSpace(pathName), "\"")
	info, err := os.Stat(pathName)
	if err != nil {
		gameActivityMu.Unlock()
		return gameActivityState{}, err
	}
	if info.IsDir() || !strings.EqualFold(filepath.Ext(pathName), ".exe") {
		gameActivityMu.Unlock()
		return gameActivityState{}, fmt.Errorf("select a Windows executable")
	}
	config, err := loadGameActivityConfig()
	if err != nil {
		gameActivityMu.Unlock()
		return gameActivityState{}, err
	}
	id := normalizeGamePath(pathName)
	if config.IgnoredGames != nil {
		delete(config.IgnoredGames, id)
	}
	name := strings.TrimSpace(args.Name)
	if name == "" || strings.ContainsAny(name, `\/`) {
		name = gameDisplayName(pathName)
	}
	config.Overrides[id] = name
	newEntry := gameActivityEntry{ID: id, Name: name, Path: pathName, Source: "manual", LastSeen: time.Now()}
	updated := false
	for i := range config.GamesSeen {
		if normalizeGamePath(config.GamesSeen[i].Path) == id {
			config.GamesSeen[i] = newEntry
			updated = true
			break
		}
	}
	if !updated {
		config.GamesSeen = append(config.GamesSeen, newEntry)
	}
	config.GamesSeen = deduplicateGameEntries(config.GamesSeen, config.Overrides)
	if err := saveGameActivityConfig(config); err != nil {
		gameActivityMu.Unlock()
		return gameActivityState{}, err
	}
	gameActivityMu.Unlock()
	refreshGameActivityProcesses()
	gameActivityMu.Lock()
	defer gameActivityMu.Unlock()
	config, err = loadGameActivityConfig()
	if err != nil {
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
	if config.IgnoredGames == nil {
		config.IgnoredGames = map[string]bool{}
	}
	if id != "" {
		config.IgnoredGames[id] = true
	}
	delete(config.Overrides, id)
	for k := range config.DisabledGames {
		if normalizeGamePath(k) == id || strings.EqualFold(k, pathName) {
			delete(config.DisabledGames, k)
		}
	}
	delete(config.DisabledGames, id)
	filtered := make([]gameActivityEntry, 0, len(config.GamesSeen))
	for _, entry := range config.GamesSeen {
		if normalizeGamePath(entry.Path) != id && !strings.EqualFold(entry.Path, pathName) && !strings.EqualFold(entry.ID, id) {
			filtered = append(filtered, entry)
		}
	}
	config.GamesSeen = filtered
	if err := saveGameActivityConfig(config); err != nil {
		return gameActivityState{}, err
	}
	running := cachedGameActivityProcesses()
	filteredRunning := make([]gameProcess, 0, len(running))
	for _, p := range running {
		if normalizeGamePath(p.Path) != id && !strings.EqualFold(p.Path, pathName) {
			filteredRunning = append(filteredRunning, p)
		}
	}
	setCachedGameActivityProcesses(filteredRunning)
	return d.gameActivityStateUnlocked(config)
}

func (d *discordApp) toggleGameActivity(pathName string, enabled bool) (gameActivityState, error) {
	gameActivityMu.Lock()
	defer gameActivityMu.Unlock()
	config, err := loadGameActivityConfig()
	if err != nil {
		return gameActivityState{}, err
	}
	id := normalizeGamePath(pathName)
	if config.DisabledGames == nil {
		config.DisabledGames = map[string]bool{}
	}
	if enabled {
		for k := range config.DisabledGames {
			if normalizeGamePath(k) == id || strings.EqualFold(k, pathName) {
				delete(config.DisabledGames, k)
			}
		}
		delete(config.DisabledGames, id)
	} else {
		config.DisabledGames[id] = true
	}
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
	if !enabled {
		setCachedGameActivityProcesses(nil)
	}
	if err := saveGameActivityConfig(config); err != nil {
		return gameActivityState{}, err
	}
	if enabled {
		candidates, manual := gameActivityCandidates(config)
		if running, err := enumerateGameProcesses(candidates, manual); err == nil {
			setCachedGameActivityProcesses(running)
		}
	}
	return d.gameActivityStateUnlocked(config)
}

func (d *discordApp) gameActivityStateUnlocked(config gameActivityConfig) (gameActivityState, error) {
	running := []gameProcess{}
	if config.DetectionEnabled {
		allRunning := cachedGameActivityProcesses()
		for _, p := range allRunning {
			if config.IgnoredGames == nil || !config.IgnoredGames[normalizeGamePath(p.Path)] {
				running = append(running, p)
			}
		}
	}
	for i := range running {
		if override := config.Overrides[normalizeGamePath(running[i].Path)]; override != "" {
			running[i].Name = override
		}
	}
	if config.DisabledGames == nil {
		config.DisabledGames = map[string]bool{}
	}
	gamesSeen := deduplicateGameEntries(config.GamesSeen, config.Overrides)
	sort.Slice(gamesSeen, func(i, j int) bool {
		return strings.ToLower(gamesSeen[i].Name) < strings.ToLower(gamesSeen[j].Name)
	})
	return gameActivityState{DetectionEnabled: config.DetectionEnabled, RunningGames: running, GamesSeen: gamesSeen, Overrides: config.Overrides, DisabledGames: config.DisabledGames}, nil
}
