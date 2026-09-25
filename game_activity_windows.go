//go:build windows

package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	ntdll                     = syscall.NewLazyDLL("ntdll.dll")
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
	getForegroundWindow       = user32.NewProc("GetForegroundWindow")
	openDesktop               = user32.NewProc("OpenDesktopW")
	isIconic                  = user32.NewProc("IsIconic")
	ntQueryInformationProcess = ntdll.NewProc("NtQueryInformationProcess")
	readProcessMemory         = kernel32.NewProc("ReadProcessMemory")
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

func enumerateGameProcesses(candidates gameCandidateIndex, manual map[string]string) ([]gameProcess, error) {
	windows := visibleWindowTitles()
	snapshot, _, _ := createToolhelp32Snapshot.Call(th32csSnapProcess, 0)
	if snapshot == 0 || snapshot == ^uintptr(0) {
		return []gameProcess{}, nil
	}
	defer closeHandle.Call(snapshot)
	entry := processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
	byProcess := map[string]gameProcess{}
	currentPID := uint32(os.Getpid())
	first, _, _ := process32First.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	for first != 0 {
		pid := entry.ProcessID
		pathName, startTime := processPathAndStartTime(pid)
		name := filepath.Base(pathName)
		normPath := normalizeGamePath(pathName)
		if pid != 0 && pid != currentPID && normPath != "" {
			display := manual[normPath]
			appID := "0"
			if display == "" && name != "" && name != "." {
				commandLine := ""
				commandLineRead := false
				for _, candidate := range candidates[strings.ToLower(name)] {
					if !executableMatchesPath(pathName, candidate.Executable) || !candidateIsAllowed(candidate, pathName) {
						continue
					}
					if candidate.Arguments != "" {
						if !commandLineRead {
							commandLine = processCommandLine(pid)
							commandLineRead = true
						}
						if !strings.Contains(strings.ToLower(commandLine), strings.ToLower(candidate.Arguments)) {
							continue
						}
					}
					display = candidate.Name
					appID = candidate.ApplicationID
					break
				}
			}
			if display != "" {
				title := windows[pid]
				if title == "" {
					title = display
				}
				key := normPath + "|" + strconv.FormatUint(uint64(pid), 10)
				byProcess[key] = gameProcess{PID: pid, Name: display, Path: pathName, WindowTitle: title, Start: startTime, ApplicationID: appID}
			}
		}
		entry = processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
		first, _, _ = process32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}
	result := make([]gameProcess, 0, len(byProcess))
	for _, process := range byProcess {
		result = append(result, process)
	}
	fgPID := foregroundProcessID()
	sort.Slice(result, func(i, j int) bool {
		if result[i].PID == fgPID && result[j].PID != fgPID {
			return true
		}
		if result[j].PID == fgPID && result[i].PID != fgPID {
			return false
		}
		if result[i].Start != result[j].Start {
			return result[i].Start > result[j].Start
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result, nil
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

func processCommandLine(pid uint32) string {
	if pid == 0 {
		return ""
	}
	handle, _, _ := openProcess.Call(0x0410, 0, uintptr(pid))
	if handle == 0 {
		return ""
	}
	defer closeHandle.Call(handle)
	type processBasicInformation struct {
		ExitStatus                   int32
		PebBaseAddress               uintptr
		AffinityMask                 uintptr
		BasePriority                 int32
		UniqueProcessID              uintptr
		InheritedFromUniqueProcessID uintptr
	}
	var basic processBasicInformation
	var returned uint32
	status, _, _ := ntQueryInformationProcess.Call(handle, 0, uintptr(unsafe.Pointer(&basic)), unsafe.Sizeof(basic), uintptr(unsafe.Pointer(&returned)))
	if int32(status) < 0 || basic.PebBaseAddress == 0 {
		return ""
	}
	pebAddress := basic.PebBaseAddress
	pointerSize := int(unsafe.Sizeof(uintptr(0)))
	if pointerSize == 8 {
		var wow64PEB uintptr
		status, _, _ = ntQueryInformationProcess.Call(handle, 26, uintptr(unsafe.Pointer(&wow64PEB)), unsafe.Sizeof(wow64PEB), uintptr(unsafe.Pointer(&returned)))
		if int32(status) >= 0 && wow64PEB != 0 {
			pebAddress = wow64PEB
			pointerSize = 4
		}
	}
	read := func(address uintptr, buffer []byte) bool {
		if len(buffer) == 0 {
			return false
		}
		var bytesRead uintptr
		result, _, _ := readProcessMemory.Call(handle, address, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)), uintptr(unsafe.Pointer(&bytesRead)))
		return result != 0 && bytesRead == uintptr(len(buffer))
	}
	processParametersOffset := uintptr(0x10)
	commandLineOffset := uintptr(0x40)
	if pointerSize == 8 {
		processParametersOffset = 0x20
		commandLineOffset = 0x70
	}
	var peb [0x28]byte
	if !read(pebAddress, peb[:processParametersOffset+uintptr(pointerSize)]) {
		return ""
	}
	var processParameters uintptr
	if pointerSize == 8 {
		processParameters = uintptr(binary.LittleEndian.Uint64(peb[processParametersOffset:]))
	} else {
		processParameters = uintptr(binary.LittleEndian.Uint32(peb[processParametersOffset:]))
	}
	if processParameters == 0 {
		return ""
	}
	unicodeStringSize := pointerSize * 2
	var unicodeString [16]byte
	if !read(processParameters+commandLineOffset, unicodeString[:unicodeStringSize]) {
		return ""
	}
	length := binary.LittleEndian.Uint16(unicodeString[:2])
	if length == 0 || length%2 != 0 {
		return ""
	}
	var commandLineAddress uintptr
	if pointerSize == 8 {
		commandLineAddress = uintptr(binary.LittleEndian.Uint64(unicodeString[8:16]))
	} else {
		commandLineAddress = uintptr(binary.LittleEndian.Uint32(unicodeString[4:8]))
	}
	if commandLineAddress == 0 {
		return ""
	}
	commandLine := make([]uint16, int(length)/2)
	if !read(commandLineAddress, unsafe.Slice((*byte)(unsafe.Pointer(&commandLine[0])), int(length))) {
		return ""
	}
	return syscall.UTF16ToString(commandLine)
}

var (
	enumWindowsMu       sync.Mutex
	currentEnumTitles   map[uint32]string
	enumWindowsCallback = syscall.NewCallback(enumWindowsProc)
)

func enumWindowsProc(hwnd uintptr, _ uintptr) uintptr {
	visible, _, _ := isWindowVisible.Call(hwnd)
	iconic, _, _ := isIconic.Call(hwnd)
	if visible == 0 && iconic == 0 {
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
	defDesk, _, _ := openDesktop.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Default"))), 0, 0, 0x0040)
	if defDesk != 0 {
		enumDesktopWindows.Call(defDesk, enumWindowsCallback, 0)
		closeDesktop.Call(defDesk)
	}
	titles := currentEnumTitles
	currentEnumTitles = nil
	return titles
}

func foregroundProcessID() uint32 {
	hwnd, _, _ := getForegroundWindow.Call()
	if hwnd == 0 {
		return 0
	}
	var pid uint32
	getWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid
}
