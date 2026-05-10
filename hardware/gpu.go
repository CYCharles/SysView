package hardware

import (
	"fmt"
	"strings"

	"github.com/StackExchange/wmi"
	"golang.org/x/sys/windows/registry"
)

type win32VideoController struct {
	Name                        string
	AdapterCompatibility        string
	AdapterRAM                  uint64
	DriverVersion               string
	DriverDate                  string
	VideoModeDescription        string
	VideoProcessor              string
	CurrentRefreshRate           uint32
	CurrentHorizontalResolution uint32
	CurrentVerticalResolution   uint32
	AdapterDACType              string
	VideoArchitecture           uint16
	Availability                uint16
	PNPDeviceID                 string
}

type GPUInfo struct {
	Model           string
	Codename        string
	Manufacturer    string
	VRAMType        string
	VRAMSize        string
	BusWidth        string
	VRAMFrequency   string
	CoreFrequency   string
	StreamProcessors string
	DriverVersion   string
	DirectXVersion  string
	Resolution      string
	RefreshRate     string
	Availability    string
}

func GetGPUInfo() []GPUInfo {
	var result []GPUInfo

	var dst []win32VideoController
	err := wmi.Query("SELECT * FROM Win32_VideoController", &dst)
	if err != nil || len(dst) == 0 {
		return []GPUInfo{{Model: "获取失败", Manufacturer: "N/A", VRAMSize: "N/A"}}
	}

	for _, g := range dst {
		info := GPUInfo{
			Model:        safeStringOr(g.Name, "N/A"),
			Manufacturer: safeStringOr(g.AdapterCompatibility, "N/A"),
		}

		// VRAM
		if g.AdapterRAM > 0 {
			info.VRAMSize = formatBytes(g.AdapterRAM)
		} else {
			// Try from registry
			info.VRAMSize = getVRAMFromRegistry(g.PNPDeviceID)
		}

		// Driver version - clean up format
		rawDriverVer := strings.TrimSpace(g.DriverVersion)
		info.DriverVersion = formatDriverVersion(rawDriverVer)

		// DirectX version from registry
		info.DirectXVersion = getDirectXVersion()

		// Resolution
		if g.CurrentHorizontalResolution > 0 && g.CurrentVerticalResolution > 0 {
			info.Resolution = fmt.Sprintf("%d x %d", g.CurrentHorizontalResolution, g.CurrentVerticalResolution)
		} else {
			info.Resolution = safeStringOr(g.VideoModeDescription, "N/A")
		}

		// Refresh rate
		if g.CurrentRefreshRate > 0 {
			info.RefreshRate = fmt.Sprintf("%d Hz", g.CurrentRefreshRate)
		} else {
			info.RefreshRate = "N/A"
		}

		// Video processor / Core frequency
		info.CoreFrequency = safeStringOr(strings.TrimSpace(g.VideoProcessor), "N/A")

		// Availability
		switch g.Availability {
		case 3:
			info.Availability = "正常运行"
		case 4:
			info.Availability = "警告"
		case 5:
			info.Availability = "错误"
		case 8:
			info.Availability = "离线"
		case 12:
			info.Availability = "已断开"
		default:
			info.Availability = "未知"
		}

		// Get additional info from registry
		gpuRegInfo := getGPURegInfo(g.PNPDeviceID)
		info.VRAMType = gpuRegInfo.VRAMType
		info.BusWidth = gpuRegInfo.BusWidth
		info.Codename = gpuRegInfo.Codename
		if info.VRAMSize == "N/A" && gpuRegInfo.VRAMSize != "" {
			info.VRAMSize = gpuRegInfo.VRAMSize
		}
		info.VRAMFrequency = gpuRegInfo.VRAMFrequency
		info.StreamProcessors = gpuRegInfo.StreamProcessors
		if info.CoreFrequency == "N/A" && gpuRegInfo.CoreFrequency != "" {
			info.CoreFrequency = gpuRegInfo.CoreFrequency
		}

		result = append(result, info)
	}

	return result
}

// GPURegInfo holds additional GPU info from registry
type GPURegInfo struct {
	VRAMType        string
	VRAMSize        string
	BusWidth        string
	VRAMFrequency   string
	CoreFrequency   string
	Codename        string
	StreamProcessors string
}

// getGPURegInfo reads GPU details from the display driver registry key
func getGPURegInfo(pnpDeviceID string) GPURegInfo {
	info := GPURegInfo{
		VRAMType:        "N/A",
		VRAMSize:        "N/A",
		BusWidth:        "N/A",
		VRAMFrequency:   "N/A",
		CoreFrequency:   "N/A",
		Codename:        "N/A",
		StreamProcessors: "N/A",
	}

	if pnpDeviceID == "" {
		return info
	}

	// Try to find the GPU in the video registry key
	// HKLM\SYSTEM\CurrentControlSet\Control\Video\{GUID}\0000
	videoPath := `SYSTEM\CurrentControlSet\Control\Video`
	videoSubkeys := regSubKeyNames(registry.LOCAL_MACHINE, videoPath)

	for _, vk := range videoSubkeys {
		subPath := videoPath + `\` + vk + `\0000`
		// Check if this matches our GPU
		matchID := regReadString(registry.LOCAL_MACHINE, subPath, "MatchingDeviceId", "")
		if matchID == "" {
			// Try ProviderName to filter
			provider := regReadString(registry.LOCAL_MACHINE, subPath, "ProviderName", "")
			if provider == "" {
				continue
			}
		}

		adapterDesc := regReadString(registry.LOCAL_MACHINE, subPath, "Device Description", "")
		if adapterDesc == "" {
			adapterDesc = regReadString(registry.LOCAL_MACHINE, subPath, "DriverDesc", "")
		}

		// Read hardware information
		info.VRAMType = readVRAMType(subPath)
		info.BusWidth = readBusWidth(subPath)

		// Memory size from registry
		memSize := regReadUint64(registry.LOCAL_MACHINE, subPath, "HardwareInformation.MemorySize")
		if memSize > 0 && info.VRAMSize == "N/A" {
			info.VRAMSize = formatBytes(memSize)
		}

		// Try to get more details from adapter string
		if adapterDesc != "" {
			// Try to infer codename from model name
			info.Codename = inferGPUCodename(adapterDesc)
		}

		// Only read from the first matching key
		if matchID != "" || adapterDesc != "" {
			break
		}
	}

	// Try to get info from PCI registry for this specific device
	info = getGPUInfoFromPCIRegistry(pnpDeviceID, info)

	return info
}

// readVRAMType reads VRAM type from registry
func readVRAMType(subPath string) string {
	// HardwareInformation.DacType can sometimes indicate VRAM type
	dacType := regReadString(registry.LOCAL_MACHINE, subPath, "HardwareInformation.DacType", "")
	if dacType != "" {
		lower := strings.ToLower(dacType)
		if strings.Contains(lower, "gddr6x") || strings.Contains(lower, "gddr6x") {
			return "GDDR6X"
		}
		if strings.Contains(lower, "gddr6") {
			return "GDDR6"
		}
		if strings.Contains(lower, "gddr5x") {
			return "GDDR5X"
		}
		if strings.Contains(lower, "gddr5") {
			return "GDDR5"
		}
		if strings.Contains(lower, "ddr4") {
			return "DDR4 (共享)"
		}
		if strings.Contains(lower, "ddr3") {
			return "DDR3 (共享)"
		}
	}

	// Try to infer from GPU model name
	return "N/A"
}

// readBusWidth reads memory bus width from registry
func readBusWidth(subPath string) string {
	// Not directly available in standard registry entries
	// Would need vendor-specific tools
	return "N/A"
}

// getGPUInfoFromPCIRegistry reads GPU info from PCI device registry
func getGPUInfoFromPCIRegistry(pnpDeviceID string, currentInfo GPURegInfo) GPURegInfo {
	// Convert PnP device ID to registry path
	// PnP ID format: PCI\VEN_10DE&DEV_1234&SUBSYS_...
	pciPath := `SYSTEM\CurrentControlSet\Enum\` + strings.ReplaceAll(pnpDeviceID, `\`, `\`)

	// Try to read the device description
	devDesc := regReadString(registry.LOCAL_MACHINE, pciPath, "DeviceDesc", "")
	if devDesc != "" {
		cleanDesc := cleanDeviceDesc(devDesc)
		if currentInfo.Codename == "N/A" {
			currentInfo.Codename = inferGPUCodename(cleanDesc)
		}
	}

	// Try vendor-specific registry keys for NVIDIA
	if strings.Contains(strings.ToUpper(pnpDeviceID), "VEN_10DE") {
		return getNVIDIAInfo(currentInfo, pnpDeviceID)
	}

	// AMD
	if strings.Contains(strings.ToUpper(pnpDeviceID), "VEN_1002") {
		return getAMDGPUInfo(currentInfo, pnpDeviceID)
	}

	// Intel
	if strings.Contains(strings.ToUpper(pnpDeviceID), "VEN_8086") {
		return getIntelGPUInfo(currentInfo, pnpDeviceID)
	}

	return currentInfo
}

// getNVIDIAInfo reads NVIDIA-specific GPU info
func getNVIDIAInfo(info GPURegInfo, pnpDeviceID string) GPURegInfo {
	// Try reading from NVIDIA registry
	nvPath := `SOFTWARE\NVIDIA Corporation\Global`
	subkeys := regSubKeyNames(registry.LOCAL_MACHINE, nvPath)
	_ = subkeys

	// Try driver registry
	driverStorePath := `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`
	classSubkeys := regSubKeyNames(registry.LOCAL_MACHINE, driverStorePath)

	for _, ck := range classSubkeys {
		fullPath := driverStorePath + `\` + ck
		provider := regReadString(registry.LOCAL_MACHINE, fullPath, "ProviderName", "")
		if strings.Contains(strings.ToLower(provider), "nvidia") {
			// Found NVIDIA driver key
			driverDesc := regReadString(registry.LOCAL_MACHINE, fullPath, "DriverDesc", "")
			if driverDesc != "" {
				// Try to extract info from description
				if info.Codename == "N/A" {
					info.Codename = inferGPUCodename(driverDesc)
				}
			}
		}
	}

	// Default VRAM type for modern NVIDIA
	if info.VRAMType == "N/A" {
		// Try to infer from model
		info.VRAMType = inferNVIDIAVRAMType(pnpDeviceID)
	}

	return info
}

// getAMDGPUInfo reads AMD-specific GPU info
func getAMDGPUInfo(info GPURegInfo, pnpDeviceID string) GPURegInfo {
	// Try driver registry
	driverStorePath := `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`
	classSubkeys := regSubKeyNames(registry.LOCAL_MACHINE, driverStorePath)

	for _, ck := range classSubkeys {
		fullPath := driverStorePath + `\` + ck
		provider := regReadString(registry.LOCAL_MACHINE, fullPath, "ProviderName", "")
		if strings.Contains(strings.ToLower(provider), "amd") ||
			strings.Contains(strings.ToLower(provider), "advanced micro") {
			driverDesc := regReadString(registry.LOCAL_MACHINE, fullPath, "DriverDesc", "")
			if driverDesc != "" && info.Codename == "N/A" {
				info.Codename = inferGPUCodename(driverDesc)
			}
		}
	}

	if info.VRAMType == "N/A" {
		info.VRAMType = inferAMDVRAMType(pnpDeviceID)
	}

	return info
}

// getIntelGPUInfo reads Intel-specific GPU info
func getIntelGPUInfo(info GPURegInfo, pnpDeviceID string) GPURegInfo {
	info.VRAMType = "共享系统内存"
	info.VRAMSize = "共享系统内存"

	driverStorePath := `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`
	classSubkeys := regSubKeyNames(registry.LOCAL_MACHINE, driverStorePath)

	for _, ck := range classSubkeys {
		fullPath := driverStorePath + `\` + ck
		provider := regReadString(registry.LOCAL_MACHINE, fullPath, "ProviderName", "")
		if strings.Contains(strings.ToLower(provider), "intel") {
			driverDesc := regReadString(registry.LOCAL_MACHINE, fullPath, "DriverDesc", "")
			if driverDesc != "" && info.Codename == "N/A" {
				info.Codename = inferGPUCodename(driverDesc)
			}
		}
	}

	return info
}

// inferGPUCodename guesses GPU codename from model name
func inferGPUCodename(name string) string {
	upper := strings.ToUpper(name)

	// NVIDIA
	if strings.Contains(upper, "RTX 5090") {
		return "Blackwell (GB202)"
	}
	if strings.Contains(upper, "RTX 5080") {
		return "Blackwell (GB203)"
	}
	if strings.Contains(upper, "RTX 4090") {
		return "Ada Lovelace (AD102)"
	}
	if strings.Contains(upper, "RTX 4080") || strings.Contains(upper, "RTX 4080 SUPER") {
		return "Ada Lovelace (AD103)"
	}
	if strings.Contains(upper, "RTX 4070") {
		return "Ada Lovelace (AD104)"
	}
	if strings.Contains(upper, "RTX 4060") {
		return "Ada Lovelace (AD107)"
	}
	if strings.Contains(upper, "RTX 3090") {
		return "Ampere (GA102)"
	}
	if strings.Contains(upper, "RTX 3080") {
		return "Ampere (GA102)"
	}
	if strings.Contains(upper, "RTX 3070") {
		return "Ampere (GA104)"
	}
	if strings.Contains(upper, "RTX 3060") {
		return "Ampere (GA106)"
	}
	if strings.Contains(upper, "RTX 2080") {
		return "Turing (TU104)"
	}
	if strings.Contains(upper, "RTX 2070") {
		return "Turing (TU106)"
	}
	if strings.Contains(upper, "RTX 2060") {
		return "Turing (TU106)"
	}
	if strings.Contains(upper, "GTX 1660") {
		return "Turing (TU116)"
	}
	if strings.Contains(upper, "GTX 1080") {
		return "Pascal (GP104)"
	}
	if strings.Contains(upper, "GTX 1070") {
		return "Pascal (GP104)"
	}
	if strings.Contains(upper, "GTX 1060") {
		return "Pascal (GP106)"
	}

	// AMD
	if strings.Contains(upper, "RX 9070") {
		return "RDNA 4"
	}
	if strings.Contains(upper, "RX 7900") {
		return "RDNA 3 (Navi 31)"
	}
	if strings.Contains(upper, "RX 7800") || strings.Contains(upper, "RX 7700") {
		return "RDNA 3 (Navi 32)"
	}
	if strings.Contains(upper, "RX 7600") {
		return "RDNA 3 (Navi 33)"
	}
	if strings.Contains(upper, "RX 6900") || strings.Contains(upper, "RX 6800") {
		return "RDNA 2 (Navi 21)"
	}
	if strings.Contains(upper, "RX 6700") || strings.Contains(upper, "RX 6750") {
		return "RDNA 2 (Navi 22)"
	}
	if strings.Contains(upper, "RX 6600") {
		return "RDNA 2 (Navi 23)"
	}
	if strings.Contains(upper, "RX 5700") || strings.Contains(upper, "RX 5800") {
		return "RDNA (Navi 10)"
	}
	if strings.Contains(upper, "RX 5600") {
		return "RDNA (Navi 10)"
	}
	if strings.Contains(upper, "RX VEGA") {
		return "Vega (Vega 10/20)"
	}
	if strings.Contains(upper, "RX 580") || strings.Contains(upper, "RX 570") {
		return "Polaris (Polaris 20)"
	}

	// Intel Arc
	if strings.Contains(upper, "ARC A770") {
		return "Alchemist (ACM-G10)"
	}
	if strings.Contains(upper, "ARC A750") {
		return "Alchemist (ACM-G10)"
	}
	if strings.Contains(upper, "ARC A580") {
		return "Alchemist (ACM-G10)"
	}
	if strings.Contains(upper, "ARC A380") {
		return "Alchemist (ACM-G11)"
	}
	if strings.Contains(upper, "ARC B") {
		return "Battlemage"
	}

	// Intel Integrated
	if strings.Contains(upper, "UHD") && strings.Contains(upper, "770") {
		return "Xe-LPG (Rocket Lake-S)"
	}
	if strings.Contains(upper, "UHD") && strings.Contains(upper, "730") {
		return "Xe-LPG (Rocket Lake-S)"
	}
	if strings.Contains(upper, "IRIS") && strings.Contains(upper, "XE") {
		return "Xe Graphics"
	}

	return "N/A"
}

// inferNVIDIAVRAMType guesses VRAM type from NVIDIA GPU
func inferNVIDIAVRAMType(pnpDeviceID string) string {
	upper := strings.ToUpper(pnpDeviceID)
	if strings.Contains(upper, "RTX 40") || strings.Contains(upper, "RTX 50") {
		return "GDDR6X"
	}
	if strings.Contains(upper, "RTX 30") {
		return "GDDR6X"
	}
	if strings.Contains(upper, "RTX 20") || strings.Contains(upper, "GTX 16") {
		return "GDDR6"
	}
	return "GDDR6"
}

// inferAMDVRAMType guesses VRAM type from AMD GPU
func inferAMDVRAMType(pnpDeviceID string) string {
	upper := strings.ToUpper(pnpDeviceID)
	if strings.Contains(upper, "RX 7") || strings.Contains(upper, "RX 9") {
		return "GDDR6"
	}
	if strings.Contains(upper, "RX 6") {
		return "GDDR6"
	}
	if strings.Contains(upper, "RX 5") {
		return "GDDR6 / GDDR5"
	}
	return "GDDR6"
}

// getVRAMFromRegistry tries to get VRAM from registry when WMI fails
func getVRAMFromRegistry(pnpDeviceID string) string {
	videoPath := `SYSTEM\CurrentControlSet\Control\Video`
	videoSubkeys := regSubKeyNames(registry.LOCAL_MACHINE, videoPath)

	for _, vk := range videoSubkeys {
		subPath := videoPath + `\` + vk + `\0000`
		memSize := regReadUint64(registry.LOCAL_MACHINE, subPath, "HardwareInformation.MemorySize")
		if memSize > 0 {
			return formatBytes(memSize)
		}
	}

	return "N/A"
}

// formatDriverVersion cleans up NVIDIA driver version numbers
func formatDriverVersion(ver string) string {
	if ver == "" {
		return "N/A"
	}

	// NVIDIA driver versions are often like "31.0.15.3667" - convert to readable format
	// The last 5 digits (before any dots) are the actual driver version
	parts := strings.Split(ver, ".")
	if len(parts) >= 4 {
		// Try to extract meaningful version
		// NVIDIA: 3rd and 4th parts combined often give the version
		// e.g., "31.0.15.3667" -> "536.67"
		lastParts := parts[len(parts)-2] + "." + parts[len(parts)-1]
		// Remove leading zeros
		lastParts = strings.TrimPrefix(lastParts, "0")
		if len(lastParts) > 2 {
			// Insert dot for readability: "53667" -> "536.67"
			if len(parts[len(parts)-1]) >= 3 {
				main := parts[len(parts)-1][:len(parts[len(parts)-1])-2]
				minor := parts[len(parts)-1][len(parts[len(parts)-1])-2:]
				if main != "" {
					return main + "." + minor
				}
			}
		}
		return lastParts
	}

	return ver
}

// getDirectXVersion reads the DirectX version from registry
func getDirectXVersion() string {
	// Check DirectX version from registry
	dxVer := regReadString(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\DirectX`,
		"Version", "")
	if dxVer != "" {
		// Format: "4.09.00.0904" etc.
		parts := strings.Split(dxVer, ".")
		if len(parts) >= 2 {
			major := parts[0]
			minor := parts[1]
			switch {
			case major == "4" && minor == "10":
				return "DirectX 12"
			case major == "4" && minor == "09":
				// Check sub-version
				if len(parts) >= 4 {
					sub := parts[3]
					if strings.HasPrefix(sub, "1") {
						return "DirectX 11"
					}
				}
				return "DirectX 9.0c"
			default:
				// Try to determine from OS version
			}
		}
	}

	// Windows 10/11 always supports DirectX 12
	// Check Windows version
	dxFeatureLevel := regReadString(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\DirectX`,
		"InstalledVersion", "")

	_ = dxFeatureLevel

	// Modern Windows always has DX12
	osVer := runPowerShell(`[System.Environment]::OSVersion.Version.Major`)
	if osVer == "10" {
		// Check if it's Windows 11 (build 22000+)
		build := regReadString(registry.LOCAL_MACHINE,
			`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
			"CurrentBuildNumber", "")
		if build != "" {
			var buildNum int
			fmt.Sscanf(build, "%d", &buildNum)
			if buildNum >= 22000 {
				return "DirectX 12 (Ultimate)"
			}
		}
		return "DirectX 12"
	}

	return "N/A"
}
