//go:build windows

package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	processQueryLimitedInformation = 0x1000
	th32csSnapProcess              = 0x00000002
	maxPath                        = 260
)

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	user32                    = syscall.NewLazyDLL("user32.dll")
	createToolhelp32Snapshot  = kernel32.NewProc("CreateToolhelp32Snapshot")
	process32First            = kernel32.NewProc("Process32FirstW")
	process32Next             = kernel32.NewProc("Process32NextW")
	openProcess               = kernel32.NewProc("OpenProcess")
	queryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
	getProcessTimes           = kernel32.NewProc("GetProcessTimes")
	closeHandle               = kernel32.NewProc("CloseHandle")
	enumWindows               = user32.NewProc("EnumWindows")
	openInputDesktop          = user32.NewProc("OpenInputDesktop")
	enumDesktopWindows        = user32.NewProc("EnumDesktopWindows")
	closeDesktop              = user32.NewProc("CloseDesktop")
	isWindowVisible           = user32.NewProc("IsWindowVisible")
	getWindowThreadProcessID  = user32.NewProc("GetWindowThreadProcessId")
	getWindowTextLength       = user32.NewProc("GetWindowTextLengthW")
	getWindowText             = user32.NewProc("GetWindowTextW")
)

type processEntry32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	Priority        int32
	Flags           uint32
	Executable      [maxPath]uint16
}

func enumerateGameProcesses(candidatePaths ...string) ([]gameProcess, error) {
	windows := visibleWindowTitles()
	candidates := map[string]bool{}
	for _, candidate := range candidatePaths {
		candidates[normalizeGamePath(candidate)] = true
	}
	snapshot, _, _ := createToolhelp32Snapshot.Call(th32csSnapProcess, 0)
	if snapshot == 0 || snapshot == ^uintptr(0) {
		return []gameProcess{}, nil
	}
	defer closeHandle.Call(snapshot)
	entry := processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
	byPath := map[string]gameProcess{}
	currentPID := uint32(os.Getpid())
	first, _, _ := process32First.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	for first != 0 {
		pid := entry.ProcessID
		pathName, startTime := processPathAndStartTime(pid)
		name := filepath.Base(pathName)
		normPath := normalizeGamePath(pathName)
		if pid != 0 && pid != currentPID && normPath != "" && !ignoredGameProcess(name) {
			title := windows[pid]
			isCandidate := candidates[normPath]
			isLikely := likelyGamePath(pathName)
			known, isKnown := resolveKnownGame(name)
			if isCandidate || isKnown || (isLikely && title != "") {
				display := gameDisplayName(name)
				if isKnown && known.Name != "" {
					display = known.Name
				} else if title != "" && !isCandidate {
					display = title
				}
				appID := "0"
				if isKnown && known.ApplicationID != "" {
					appID = known.ApplicationID
				}
				existing, exists := byPath[normPath]
				if !exists || (existing.WindowTitle == "" && title != "") {
					byPath[normPath] = gameProcess{
						PID:           pid,
						Name:          display,
						Path:          pathName,
						WindowTitle:   title,
						Start:         startTime,
						ApplicationID: appID,
					}
				}
			}
		}
		entry = processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
		first, _, _ = process32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}
	result := make([]gameProcess, 0, len(byPath))
	for _, p := range byPath {
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool { return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name) })
	return result, nil
}

func likelyGamePath(pathName string) bool {
	value := strings.ToLower(filepath.Clean(pathName))
	for _, marker := range []string{`\steamapps\common\`, `\epic games\`, `\gog galaxy\games\`, `\riot games\`, `\xboxgames\`, `\minecraft\`, `\roblox\`, `\games\`} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func processPathAndStartTime(pid uint32) (string, int64) {
	handle, _, _ := openProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if handle == 0 {
		return "", 0
	}
	defer closeHandle.Call(handle)
	buffer := make([]uint16, 32768)
	length := uint32(len(buffer))
	result, _, _ := queryFullProcessImageName.Call(handle, 0, uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&length)))
	if result == 0 || length == 0 {
		return "", 0
	}
	path := syscall.UTF16ToString(buffer[:length])
	var creation, exit, kernel, user syscall.Filetime
	var startMs int64
	if ret, _, _ := getProcessTimes.Call(handle, uintptr(unsafe.Pointer(&creation)), uintptr(unsafe.Pointer(&exit)), uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&user))); ret != 0 {
		startMs = creation.Nanoseconds() / 1e6
	}
	return path, startMs
}

var (
	enumWindowsMu       sync.Mutex
	currentEnumTitles   map[uint32]string
	enumWindowsCallback = syscall.NewCallback(enumWindowsProc)
)

func enumWindowsProc(hwnd uintptr, _ uintptr) uintptr {
	visible, _, _ := isWindowVisible.Call(hwnd)
	if visible == 0 {
		return 1
	}
	var pid uint32
	getWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return 1
	}
	length, _, _ := getWindowTextLength.Call(hwnd)
	if int32(length) <= 0 || int32(length) > 1024 {
		return 1
	}
	buffer := make([]uint16, length+1)
	getWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if title := strings.TrimSpace(syscall.UTF16ToString(buffer)); title != "" {
		if currentEnumTitles[pid] == "" || len(title) > len(currentEnumTitles[pid]) {
			currentEnumTitles[pid] = title
		}
	}
	return 1
}

func visibleWindowTitles() map[uint32]string {
	enumWindowsMu.Lock()
	defer enumWindowsMu.Unlock()
	currentEnumTitles = map[uint32]string{}
	enumWindows.Call(enumWindowsCallback, 0)
	inputDesk, _, _ := openInputDesktop.Call(0, 0, 0x0040)
	if inputDesk != 0 {
		enumDesktopWindows.Call(inputDesk, enumWindowsCallback, 0)
		closeDesktop.Call(inputDesk)
	}
	titles := currentEnumTitles
	currentEnumTitles = nil
	return titles
}

func ignoredGameProcess(name string) bool {
	switch strings.ToLower(name) {
	case "discord.exe", "discordcanary.exe", "discordptb.exe", "msedgewebview2.exe", "crashpad_handler.exe", "explorer.exe", "dwm.exe", "applicationframehost.exe", "searchhost.exe", "startmenuexperiencehost.exe", "shellexperiencehost.exe", "textinputhost.exe", "taskmgr.exe", "powershell.exe", "pwsh.exe", "cmd.exe", "conhost.exe":
		return true
	default:
		return false
	}
}
