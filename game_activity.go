package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

type knownGame struct {
	ApplicationID string
	Name          string
}

var defaultKnownGames = map[string]knownGame{
	"wow.exe":                           {ApplicationID: "356875762940379136", Name: "World of Warcraft"},
	"wowclassic.exe":                    {ApplicationID: "356875762940379136", Name: "World of Warcraft"},
	"wowt.exe":                          {ApplicationID: "356875762940379136", Name: "World of Warcraft"},
	"wowb.exe":                          {ApplicationID: "356875762940379136", Name: "World of Warcraft"},
	"league of legends.exe":             {ApplicationID: "401518687463948290", Name: "League of Legends"},
	"valorant.exe":                      {ApplicationID: "700144211132645406", Name: "VALORANT"},
	"overwatch.exe":                     {ApplicationID: "356867200780468224", Name: "Overwatch"},
	"csgo.exe":                          {ApplicationID: "738864303494791248", Name: "Counter-Strike 2"},
	"cs2.exe":                           {ApplicationID: "738864303494791248", Name: "Counter-Strike 2"},
	"dota2.exe":                         {ApplicationID: "738864293411684352", Name: "Dota 2"},
	"gta5.exe":                          {ApplicationID: "436993026818867200", Name: "Grand Theft Auto V"},
	"minecraft.exe":                     {ApplicationID: "356875127150903296", Name: "Minecraft"},
	"rocketleague.exe":                  {ApplicationID: "356877028164632576", Name: "Rocket League"},
	"fortniteclient-win64-shipping.exe": {ApplicationID: "432980957394370572", Name: "Fortnite"},
	"genshinimpact.exe":                 {ApplicationID: "762434991303950386", Name: "Genshin Impact"},
	"starrail.exe":                      {ApplicationID: "1100344445853245480", Name: "Honkai: Star Rail"},
	"ffxiv_dx11.exe":                    {ApplicationID: "468936993781252096", Name: "FINAL FANTASY XIV"},
	"r5apex.exe":                        {ApplicationID: "542385150820417537", Name: "Apex Legends"},
}

var (
	cachedDetectableMu sync.RWMutex
	cachedDetectable   map[string]knownGame
)

func resolveKnownGame(exeName string) (knownGame, bool) {
	lower := strings.ToLower(exeName)
	if filepath.Ext(lower) == "" {
		lower += ".exe"
	}
	cachedDetectableMu.RLock()
	if cachedDetectable != nil {
		if game, ok := cachedDetectable[lower]; ok {
			cachedDetectableMu.RUnlock()
			return game, true
		}
	}
	cachedDetectableMu.RUnlock()
	if game, ok := defaultKnownGames[lower]; ok {
		return game, true
	}
	return knownGame{}, false
}

func initDetectableCache() {
	go func() {
		cachePath, err := detectableCachePath()
		if err == nil {
			if data, err := os.ReadFile(cachePath); err == nil {
				parseAndSetDetectable(data)
			}
		}
		needFetch := true
		if info, err := os.Stat(cachePath); err == nil {
			if time.Since(info.ModTime()) < 7*24*time.Hour {
				needFetch = false
			}
		}
		if needFetch {
			client := &http.Client{Timeout: 15 * time.Second}
			resp, err := client.Get("https://discord.com/api/v9/applications/detectable")
			if err == nil && resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err == nil && len(body) > 0 {
					if parseAndSetDetectable(body) {
						_ = os.WriteFile(cachePath, body, 0644)
					}
				}
			}
		}
	}()
}

func detectableCachePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "Moreno", "DiscordLite")
	_ = os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "detectable.json"), nil
}

type detectableApp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Executables []struct {
		Name string `json:"name"`
		OS   string `json:"os"`
	} `json:"executables"`
}

func parseAndSetDetectable(data []byte) bool {
	var apps []detectableApp
	if err := json.Unmarshal(data, &apps); err != nil {
		return false
	}
	m := make(map[string]knownGame, len(apps)*2)
	for _, app := range apps {
		for _, exe := range app.Executables {
			if exe.OS == "win32" || exe.OS == "" {
				base := strings.ToLower(filepath.Base(strings.ReplaceAll(exe.Name, "/", "\\")))
				if base != "" {
					m[base] = knownGame{ApplicationID: app.ID, Name: app.Name}
				}
			}
		}
	}
	cachedDetectableMu.Lock()
	cachedDetectable = m
	cachedDetectableMu.Unlock()
	return true
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
	go func() {
		for {
			if d.window != nil {
				if state, err := d.gameActivityState(); err == nil {
					encoded, _ := json.Marshal(state)
					d.window.ExecJS("if(window.__vcGameActivityApply)window.__vcGameActivityApply(" + string(encoded) + ");")
				}
			}
			time.Sleep(3 * time.Second)
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
	candidates := make([]string, 0, len(config.GamesSeen))
	for _, entry := range config.GamesSeen {
		entryID := normalizeGamePath(entry.Path)
		if config.IgnoredGames != nil && config.IgnoredGames[entryID] {
			continue
		}
		if ignoredGameProcess(filepath.Base(entry.Path)) {
			continue
		}
		if entry.Source == "manual" || config.Overrides[entryID] != "" || likelyGamePath(entry.Path) {
			candidates = append(candidates, entry.Path)
		}
	}
	running, err := enumerateGameProcesses(candidates...)
	if err != nil {
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
		exe := strings.ToLower(filepath.Base(entry.Path))
		if ignoredGameProcess(exe) {
			needsSave = true
			continue
		}
		if (exe == "javaw.exe" || exe == "java.exe") && entry.Name == "Minecraft" && entry.Source == "detected" {
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
	if known, ok := resolveKnownGame(base); ok && known.Name != "" {
		return known.Name
	}
	clean := strings.TrimSuffix(base, filepath.Ext(base))
	if clean == "" {
		return base
	}
	return clean
}

func ignoredGameProcess(name string) bool {
	lower := strings.ToLower(name)
	switch lower {
	case "discord.exe", "discordcanary.exe", "discordptb.exe", "msedgewebview2.exe", "crashpad_handler.exe", "explorer.exe", "dwm.exe", "applicationframehost.exe", "searchhost.exe", "startmenuexperiencehost.exe", "shellexperiencehost.exe", "textinputhost.exe", "taskmgr.exe", "powershell.exe", "pwsh.exe", "cmd.exe", "conhost.exe", "git.exe", "gh.exe", "node.exe", "go.exe", "python.exe", "bash.exe", "sh.exe", "wt.exe", "wsl.exe", "wslhost.exe":
		return true
	}
	if strings.Contains(lower, "crashhandler") || strings.Contains(lower, "crashreporter") || strings.Contains(lower, "crashpad") || strings.Contains(lower, "crashmailer") || strings.Contains(lower, "werfault") || strings.Contains(lower, "errorreport") || strings.HasPrefix(lower, "unins") || strings.Contains(lower, "setup") || strings.Contains(lower, "installer") {
		return true
	}
	return false
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
		return gameActivityState{}, err
	}
	refreshGameActivityProcesses()
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
		candidates := make([]string, 0, len(config.GamesSeen))
		for _, entry := range config.GamesSeen {
			if entry.Source == "manual" || config.Overrides[normalizeGamePath(entry.Path)] != "" || likelyGamePath(entry.Path) {
				candidates = append(candidates, entry.Path)
			}
		}
		if running, err := enumerateGameProcesses(candidates...); err == nil {
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
