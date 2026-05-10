package hardware

import (
	"fmt"
	"os/exec"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func safeDerefUint32(p *uint32) uint32 {
	if p == nil {
		return 0
	}
	return *p
}

func safeDerefUint16(p *uint16) uint16 {
	if p == nil {
		return 0
	}
	return *p
}

func safeString(s string) string {
	return strings.TrimSpace(s)
}

func safeStringOr(s string, fallback string) string {
	v := strings.TrimSpace(s)
	if v == "" {
		return fallback
	}
	return v
}

func boolToChinese(b bool) string {
	if b {
		return "支持"
	}
	return "不支持"
}

func getMemoryTypeName(memType uint16) string {
	switch memType {
	case 20:
		return "DDR"
	case 21:
		return "DDR2"
	case 22:
		return "DDR2 FB-DIMM"
	case 24:
		return "DDR3"
	case 26:
		return "DDR4"
	case 27:
		return "LPDDR"
	case 28:
		return "LPDDR2"
	case 29:
		return "LPDDR3"
	case 30:
		return "LPDDR4"
	case 34:
		return "DDR5"
	case 35:
		return "LPDDR5"
	default:
		return fmt.Sprintf("未知(%d)", memType)
	}
}

func getDiskMediaType(mediaType string) string {
	t := strings.ToLower(mediaType)
	if strings.Contains(t, "ssd") || strings.Contains(t, "solid") {
		return "SSD"
	}
	if strings.Contains(t, "hdd") || strings.Contains(t, "hard") {
		return "HDD"
	}
	return strings.TrimSpace(mediaType)
}

// regReadString reads a string value from registry, returns fallback on error
func regReadString(k registry.Key, path, name, fallback string) string {
	subk, err := registry.OpenKey(k, path, registry.QUERY_VALUE)
	if err != nil {
		return fallback
	}
	defer subk.Close()

	val, _, err := subk.GetStringValue(name)
	if err != nil {
		return fallback
	}
	return safeStringOr(val, fallback)
}

// regReadStrings reads a multi-string value from registry
func regReadStrings(k registry.Key, path, name string) []string {
	subk, err := registry.OpenKey(k, path, registry.QUERY_VALUE)
	if err != nil {
		return nil
	}
	defer subk.Close()

	val, _, err := subk.GetStringsValue(name)
	if err != nil {
		return nil
	}
	return val
}

// regReadUint64 reads a uint64 value from registry
func regReadUint64(k registry.Key, path, name string) uint64 {
	subk, err := registry.OpenKey(k, path, registry.QUERY_VALUE)
	if err != nil {
		return 0
	}
	defer subk.Close()

	val, _, err := subk.GetIntegerValue(name)
	if err != nil {
		return 0
	}
	return val
}

// regSubKeyNames returns sub-key names under a registry path
func regSubKeyNames(k registry.Key, path string) []string {
	subk, err := registry.OpenKey(k, path, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil
	}
	defer subk.Close()

	names, err := subk.ReadSubKeyNames(0)
	if err != nil {
		return nil
	}
	return names
}

// runPowerShell executes a PowerShell command and returns trimmed output (no console flash)
func runPowerShell(script string) string {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	// Hide console window for subprocess
	cmd.SysProcAttr = &windows.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GlobalMemoryStatusEx wrapper
type memorystatusex struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

// getMemoryStatusEx calls GlobalMemoryStatusEx Windows API
func getMemoryStatusEx() (totalPhys, availPhys, cached uint64) {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	proc := kernel32.NewProc("GlobalMemoryStatusEx")

	var ms memorystatusex
	ms.dwLength = uint32(unsafe.Sizeof(ms))

	ret, _, _ := proc.Call(uintptr(unsafe.Pointer(&ms)))
	if ret != 0 {
		totalPhys = ms.ullTotalPhys
		availPhys = ms.ullAvailPhys
		// Cached = Total - Available - (Kernel usage estimate)
		// A better approximation: cached ≈ SystemCache + Standby
		// But from GlobalMemoryStatusEx alone, we can estimate:
		// cached ≈ total - avail - (used by processes)
		// Actually let's get it from WMI performance counters
		cached = 0
	}
	return
}

// getFirmwareType calls GetFirmwareType Windows API
func getFirmwareType() string {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	proc := kernel32.NewProc("GetFirmwareType")
	if proc.Find() != nil {
		return "N/A"
	}

	var ft uint32
	ret, _, _ := proc.Call(uintptr(unsafe.Pointer(&ft)))
	if ret != 0 {
		switch ft {
		case 1:
			return "Legacy BIOS"
		case 2:
			return "UEFI"
		default:
			return fmt.Sprintf("未知(%d)", ft)
		}
	}
	return "N/A"
}
