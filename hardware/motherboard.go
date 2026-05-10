package hardware

import (
	"fmt"
	"strings"

	"github.com/StackExchange/wmi"
	"golang.org/x/sys/windows/registry"
)

type win32BaseBoard struct {
	Manufacturer string
	Product      string
	Version      string
	SerialNumber string
	HostingBoard bool
}

type win32ComputerSystemBoard struct {
	Manufacturer string
	Model        string
	SystemType   string
}

type win32PnPEntity struct {
	Name              string
	DeviceID          string
	Manufacturer      string
	PNPClass          string
	Description       string
	HardwareID        []string
	CompatibleID      []string
	Status            string
	ConfigManagerErrorCode uint32
}

type MotherboardInfo struct {
	Manufacturer string
	Model        string
	Version      string
	SerialNumber string
	FormFactor   string
	Chipset      string
	Southbridge  string
	PCIeVersion  string
	M2Count      string
	SATACount    string
	AudioChip    string
	LANChip      string
	WiFiChip     string
	SystemModel  string
	SystemType   string
}

type win32BIOS struct {
	Manufacturer string
	Name         string
	Version      string
	ReleaseDate  string
	SerialNumber string
	Status       string
}

type BIOSInfo struct {
	Vendor       string
	Version      string
	ReleaseDate  string
	BootMode     string
	SecureBoot   string
	SerialNumber string
}

func GetMotherboardInfo() MotherboardInfo {
	info := MotherboardInfo{}

	var boardDst []win32BaseBoard
	err := wmi.Query("SELECT * FROM Win32_BaseBoard", &boardDst)
	if err == nil && len(boardDst) > 0 {
		b := boardDst[0]
		info.Manufacturer = safeStringOr(b.Manufacturer, "N/A")
		info.Model = safeStringOr(b.Product, "N/A")
		info.Version = safeStringOr(b.Version, "N/A")
		info.SerialNumber = safeStringOr(strings.TrimSpace(b.SerialNumber), "N/A")
	} else {
		info.Manufacturer = "获取失败"
		info.Model = "获取失败"
		info.Version = "N/A"
		info.SerialNumber = "N/A"
	}

	var sysDst []win32ComputerSystemBoard
	err = wmi.Query("SELECT Manufacturer, Model, SystemType FROM Win32_ComputerSystem", &sysDst)
	if err == nil && len(sysDst) > 0 {
		s := sysDst[0]
		info.SystemModel = safeStringOr(s.Model, "N/A")
		info.SystemType = safeStringOr(s.SystemType, "N/A")
	} else {
		info.SystemModel = "N/A"
		info.SystemType = "N/A"
	}

	// Get chipset, audio, LAN, WiFi from PnP devices
	info.Chipset = getChipsetFromPnP()
	info.AudioChip = getAudioChipFromPnP()
	info.LANChip = getLANChipFromPnP()
	info.WiFiChip = getWiFiChipFromPnP()
	info.Southbridge = getSouthbridgeFromPnP(info.Chipset)

	// Get form factor from registry
	info.FormFactor = getFormFactorFromRegistry()

	// PCIe version and interface counts - estimate from chipset
	info.PCIeVersion = estimatePCIeVersion(info.Chipset)
	info.M2Count = countM2Slots()
	info.SATACount = countSATAPorts()

	return info
}

// getChipsetFromPnP finds the chipset name from PnP devices
func getChipsetFromPnP() string {
	var dst []win32PnPEntity
	err := wmi.Query("SELECT Name, DeviceID, Manufacturer FROM Win32_PnPEntity WHERE PNPClass='System' OR PNPClass='Processor'", &dst)
	if err != nil {
		return getChipsetFromRegistry()
	}

	for _, d := range dst {
		name := strings.TrimSpace(d.Name)
		// Look for chipset controller names
		nameLower := strings.ToLower(name)
		if strings.Contains(nameLower, "chipset") ||
			strings.Contains(nameLower, "lakse") ||
			strings.Contains(nameLower, "cannon point") ||
			strings.Contains(nameLower, "comet point") ||
			strings.Contains(nameLower, "union point") ||
			strings.Contains(nameLower, "sunrise point") ||
			strings.Contains(nameLower, "promontory") ||
			strings.Contains(nameLower, "fch") ||
			strings.Contains(nameLower, "x370") ||
			strings.Contains(nameLower, "b350") ||
			strings.Contains(nameLower, "b450") ||
			strings.Contains(nameLower, "b550") ||
			strings.Contains(nameLower, "b650") ||
			strings.Contains(nameLower, "x470") ||
			strings.Contains(nameLower, "x570") ||
			strings.Contains(nameLower, "x670") ||
			strings.Contains(nameLower, "z370") ||
			strings.Contains(nameLower, "z390") ||
			strings.Contains(nameLower, "z490") ||
			strings.Contains(nameLower, "z590") ||
			strings.Contains(nameLower, "z690") ||
			strings.Contains(nameLower, "z790") ||
			strings.Contains(nameLower, "b360") ||
			strings.Contains(nameLower, "b460") ||
			strings.Contains(nameLower, "b560") ||
			strings.Contains(nameLower, "b660") ||
			strings.Contains(nameLower, "b760") ||
			strings.Contains(nameLower, "h310") ||
			strings.Contains(nameLower, "h410") ||
			strings.Contains(nameLower, "h510") ||
			strings.Contains(nameLower, "h610") ||
			strings.Contains(nameLower, "h770") {
			return name
		}
	}

	// Try matching "PCI standard" or "System controller"
	for _, d := range dst {
		name := strings.TrimSpace(d.Name)
		nameLower := strings.ToLower(name)
		if strings.Contains(nameLower, "pci express") ||
			strings.Contains(nameLower, "root complex") ||
			strings.Contains(nameLower, "system controller") {
			continue
		}
		deviceID := strings.ToLower(d.DeviceID)
		if strings.Contains(deviceID, "pci") &&
			(strings.Contains(nameLower, "controller") || strings.Contains(nameLower, "bridge")) {
			// Check for specific chipset vendors
			if strings.Contains(deviceID, "ven_8086") {
				// Intel chipset
				return extractIntelChipset(deviceID, name)
			} else if strings.Contains(deviceID, "ven_1022") {
				// AMD chipset
				return extractAMDChipset(deviceID, name)
			}
		}
	}

	return getChipsetFromRegistry()
}

func extractIntelChipset(deviceID, name string) string {
	// Intel chipset device IDs
	chipsetIDs := map[string]string{
		"9d4b": "Sunrise Point-LP (100系列)",
		"9d4e": "Sunrise Point-LP (100系列)",
		"a145": "CM238 (200系列)",
		"a150": "Cannon Point-LP (300系列)",
		"a1c1": "Z390",
		"a1c2": "Z370",
		"a1c3": "H370",
		"a1c4": "B360",
		"a1c5": "H310",
		"a1c6": "Z390",
		"a1c7": "Q370",
		"a1ca": "B365",
		"a1cb": "H310C",
		"a303": "Comet Point (400系列)",
		"a308": "Comet Point (400系列)",
		"0685": "Comet Point (400系列)",
		"4381": "AMD Promontory/X370",
		"43b5": "B550",
		"7a82": "LPC Controller (500系列)",
		"7a84": "Z590",
		"7a86": "H570",
		"7a88": "B560",
		"7a8a": "H510",
		"7a02": "Alder Lake (600系列)",
		"7a04": "Alder Lake (600系列)",
		"7a06": "Alder Lake (600系列)",
		"7a08": "Alder Lake (600系列)",
		"7a0c": "Alder Lake (600系列)",
		"7a14": "Alder Lake (600系列)",
		"7a16": "Alder Lake (600系列)",
		"7a22": "Raptor Lake (700系列)",
		"7a24": "Raptor Lake (700系列)",
		"7a26": "Raptor Lake (700系列)",
		"7a28": "Raptor Lake (700系列)",
		"7a2a": "Raptor Lake (700系列)",
		"7a2c": "Raptor Lake (700系列)",
	}

	for id, name := range chipsetIDs {
		if strings.Contains(strings.ToLower(deviceID), id) {
			return name
		}
	}

	// Return the name if it looks informative
	if name != "" && !strings.Contains(strings.ToLower(name), "pci standard") {
		return name
	}

	return "Intel 芯片组"
}

func extractAMDChipset(deviceID, name string) string {
	chipsetIDs := map[string]string{
		"1453": "X370",
		"1483": "B350",
		"1484": "B350",
		"1485": "A320",
		"1487": "X399",
		"1489": "A320",
		"1490": "X399",
		"7900": "X570",
		"7901": "X570",
		"7904": "X570",
		"7907": "B550",
		"7908": "B550",
		"7909": "B550",
		"790b": "A520",
		"790c": "A520",
	}

	for id, name := range chipsetIDs {
		if strings.Contains(strings.ToLower(deviceID), id) {
			return name
		}
	}

	if name != "" && !strings.Contains(strings.ToLower(name), "pci standard") {
		return name
	}

	return "AMD 芯片组"
}

// getChipsetFromRegistry reads chipset from PCI registry entries
func getChipsetFromRegistry() string {
	// Look in HKLM\SYSTEM\CurrentControlSet\Enum\PCI for chipset devices
	basePath := `SYSTEM\CurrentControlSet\Enum\PCI`
	subkeys := regSubKeyNames(registry.LOCAL_MACHINE, basePath)

	for _, sub := range subkeys {
		// Sub-key format: VEN_xxxx&DEV_xxxx&...
		lower := strings.ToLower(sub)
		// Look for Intel or AMD chipset VIDs
		if strings.Contains(lower, "ven_8086") || strings.Contains(lower, "ven_1022") {
			fullPath := basePath + `\` + sub
			funcSubkeys := regSubKeyNames(registry.LOCAL_MACHINE, fullPath)
			for _, fsub := range funcSubkeys {
				funcPath := fullPath + `\` + fsub
				devDesc := regReadString(registry.LOCAL_MACHINE, funcPath, "DeviceDesc", "")
				if devDesc != "" {
					devLower := strings.ToLower(devDesc)
					if strings.Contains(devLower, "chipset") ||
						strings.Contains(devLower, "controller") ||
						(strings.Contains(devLower, "bridge") && !strings.Contains(devLower, "pci standard") && !strings.Contains(devLower, "root")) {
						// Clean up the description (sometimes has format like @oemXX.inf,%xxx%;Real Name)
						return cleanDeviceDesc(devDesc)
					}
				}
			}
		}
	}

	return "N/A"
}

// cleanDeviceDesc cleans registry DeviceDesc format
func cleanDeviceDesc(desc string) string {
	// Format: @oemXX.inf,%id%;Friendly Name
	parts := strings.Split(desc, ";")
	if len(parts) >= 3 {
		return strings.TrimSpace(parts[2])
	}
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(desc)
}

// getAudioChipFromPnP finds audio chip from PnP devices
func getAudioChipFromPnP() string {
	var dst []win32PnPEntity
	err := wmi.Query("SELECT Name, Manufacturer FROM Win32_PnPEntity WHERE PNPClass='MEDIA' OR PNPClass='AudioEndpoint'", &dst)
	if err != nil {
		return getAudioChipFromRegistry()
	}

	var audioChips []string
	for _, dev := range dst {
		name := strings.TrimSpace(dev.Name)
		if name == "" {
			continue
		}
		lower := strings.ToLower(name)
		// Skip generic entries
		if strings.Contains(lower, "audio endpoint") ||
			strings.Contains(lower, "speaker") ||
			strings.Contains(lower, "microphone") ||
			strings.Contains(lower, "headphone") ||
			strings.Contains(lower, "line") ||
			strings.Contains(lower, "stereo mix") {
			continue
		}
		// This is likely the audio chip/driver
		audioChips = append(audioChips, name)
	}

	if len(audioChips) > 0 {
		return strings.Join(audioChips, " / ")
	}

	return getAudioChipFromRegistry()
}

func getAudioChipFromRegistry() string {
	// Check registry for audio devices
	basePath := `SYSTEM\CurrentControlSet\Enum\PCI`
	subkeys := regSubKeyNames(registry.LOCAL_MACHINE, basePath)

	for _, sub := range subkeys {
		lower := strings.ToLower(sub)
		// Common audio vendor IDs: Intel (8086), Realtek (10EC), Creative (1102), NVIDIA (10DE)
		if strings.Contains(lower, "ven_10ec") ||
			strings.Contains(lower, "ven_1102") ||
			strings.Contains(lower, "ven_8086") && strings.Contains(lower, "dev_") {
			fullPath := basePath + `\` + sub
			funcSubkeys := regSubKeyNames(registry.LOCAL_MACHINE, fullPath)
			for _, fsub := range funcSubkeys {
				funcPath := fullPath + `\` + fsub
				class := regReadString(registry.LOCAL_MACHINE, funcPath, "Class", "")
				if class == "MEDIA" {
					devDesc := regReadString(registry.LOCAL_MACHINE, funcPath, "DeviceDesc", "")
					if devDesc != "" {
						return cleanDeviceDesc(devDesc)
					}
				}
			}
		}
	}

	return "N/A"
}

// getLANChipFromPnP finds LAN chip from PnP devices
func getLANChipFromPnP() string {
	var dst []win32PnPEntity
	err := wmi.Query("SELECT Name, Manufacturer FROM Win32_PnPEntity WHERE PNPClass='Net' AND PhysicalAdapter=True", &dst)
	if err != nil {
		return getLANChipFromRegistry()
	}

	var lanChips []string
	for _, dev := range dst {
		name := strings.TrimSpace(dev.Name)
		if name == "" {
			continue
		}
		lower := strings.ToLower(name)
		// Filter: include only wired Ethernet adapters
		if strings.Contains(lower, "ethernet") ||
			strings.Contains(lower, "gigabit") ||
			strings.Contains(lower, "fast ethernet") ||
			strings.Contains(lower, "realtek") && !strings.Contains(lower, "wireless") ||
			strings.Contains(lower, "intel") && (strings.Contains(lower, "ethernet") || strings.Contains(lower, "gigabit") || strings.Contains(lower, "i219") || strings.Contains(lower, "i225") || strings.Contains(lower, "i226")) ||
			strings.Contains(lower, "killer") && !strings.Contains(lower, "wireless") ||
			strings.Contains(lower, "broadcom") && !strings.Contains(lower, "wireless") ||
			strings.Contains(lower, "qualcomm") && strings.Contains(lower, "ethernet") {
			lanChips = append(lanChips, name)
		}
	}

	if len(lanChips) > 0 {
		return strings.Join(lanChips, " / ")
	}

	return getLANChipFromRegistry()
}

func getLANChipFromRegistry() string {
	basePath := `SYSTEM\CurrentControlSet\Enum\PCI`
	subkeys := regSubKeyNames(registry.LOCAL_MACHINE, basePath)

	for _, sub := range subkeys {
		lower := strings.ToLower(sub)
		// Common LAN vendor IDs: Realtek (10EC), Intel (8086), Broadcom (14E4), Qualcomm (1969)
		if strings.Contains(lower, "ven_10ec") ||
			strings.Contains(lower, "ven_14e4") ||
			strings.Contains(lower, "ven_1969") ||
			(strings.Contains(lower, "ven_8086") && (strings.Contains(lower, "dev_15") || strings.Contains(lower, "dev_10d") || strings.Contains(lower, "dev_125") || strings.Contains(lower, "dev_1565"))) {
			fullPath := basePath + `\` + sub
			funcSubkeys := regSubKeyNames(registry.LOCAL_MACHINE, fullPath)
			for _, fsub := range funcSubkeys {
				funcPath := fullPath + `\` + fsub
				class := regReadString(registry.LOCAL_MACHINE, funcPath, "Class", "")
				if class == "Net" {
					devDesc := regReadString(registry.LOCAL_MACHINE, funcPath, "DeviceDesc", "")
					if devDesc != "" {
						return cleanDeviceDesc(devDesc)
					}
				}
			}
		}
	}

	return "N/A"
}

// getWiFiChipFromPnP finds WiFi chip from PnP devices
func getWiFiChipFromPnP() string {
	var dst []win32PnPEntity
	err := wmi.Query("SELECT Name, Manufacturer FROM Win32_PnPEntity WHERE PNPClass='Net' AND PhysicalAdapter=True", &dst)
	if err != nil {
		return "N/A"
	}

	var wifiChips []string
	for _, dev := range dst {
		name := strings.TrimSpace(dev.Name)
		if name == "" {
			continue
		}
		lower := strings.ToLower(name)
		// Filter for wireless adapters
		if strings.Contains(lower, "wireless") ||
			strings.Contains(lower, "wi-fi") ||
			strings.Contains(lower, "wifi") ||
			strings.Contains(lower, "wlan") ||
			strings.Contains(lower, "802.11") ||
			strings.Contains(lower, "bluetooth") && strings.Contains(lower, "wireless") ||
			strings.Contains(lower, "ax201") ||
			strings.Contains(lower, "ax200") ||
			strings.Contains(lower, "ax210") ||
			strings.Contains(lower, "ax211") ||
			strings.Contains(lower, "ac 9") ||
			strings.Contains(lower, "killer") && strings.Contains(lower, "wireless") ||
			strings.Contains(lower, "realtek") && strings.Contains(lower, "wireless") ||
			strings.Contains(lower, "mediatek") && strings.Contains(lower, "wireless") {
			wifiChips = append(wifiChips, name)
		}
	}

	if len(wifiChips) > 0 {
		return strings.Join(wifiChips, " / ")
	}

	return "N/A (无无线网卡)"
}

// getSouthbridgeFromPnP tries to find southbridge info
func getSouthbridgeFromPnP(chipset string) string {
	// On modern platforms, southbridge is integrated into the chipset
	// or PCH (Platform Controller Hub). The chipset field often covers both.
	if chipset == "N/A" {
		return "N/A"
	}

	// For Intel, PCH IS the chipset on modern platforms
	lower := strings.ToLower(chipset)
	if strings.Contains(lower, "series") || strings.Contains(lower, "z390") ||
		strings.Contains(lower, "z370") || strings.Contains(lower, "b360") ||
		strings.Contains(lower, "z490") || strings.Contains(lower, "z590") ||
		strings.Contains(lower, "b460") || strings.Contains(lower, "b560") ||
		strings.Contains(lower, "z690") || strings.Contains(lower, "z790") ||
		strings.Contains(lower, "b660") || strings.Contains(lower, "b760") {
		return "集成于 PCH"
	}

	// For AMD, chipset is separate from southbridge
	if strings.Contains(lower, "x370") || strings.Contains(lower, "b350") {
		return "AMD Promontory"
	}
	if strings.Contains(lower, "x470") || strings.Contains(lower, "b450") {
		return "AMD Promontory"
	}
	if strings.Contains(lower, "x570") || strings.Contains(lower, "b550") {
		return "AMD Promontory"
	}
	if strings.Contains(lower, "x670") || strings.Contains(lower, "b650") {
		return "AMD Promontory"
	}

	return "N/A"
}

// getFormFactorFromRegistry reads board form factor
func getFormFactorFromRegistry() string {
	// Try to read from registry - board type
	boardType := regReadString(registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\BIOS`,
		"BaseBoardManufacturer", "")

	_ = boardType // Not directly useful

	// Try reading from WMI - BaseBoard has no form factor field
	// Try checking the chassis type instead
	result := runPowerShell(`
		$chassis = Get-CimInstance Win32_SystemEnclosure | Select-Object -ExpandProperty ChassisTypes
		if ($chassis) {
			switch ($chassis[0]) {
				3 { "Desktop" }
				4 { "Low Profile Desktop" }
				5 { "Pizza Box" }
				6 { "Mini Tower" }
				7 { "Tower" }
				8 { "Portable" }
				9 { "Laptop" }
				10 { "Notebook" }
				13 { "All in One" }
				14 { "Sub Notebook" }
				30 { "Tablet" }
				31 { "Convertible" }
				35 { "Mini PC" }
				default { "" }
			}
		} else { "" }
	`)
	if result != "" {
		return result
	}

	return "N/A"
}

// estimatePCIeVersion estimates PCIe version based on chipset
func estimatePCIeVersion(chipset string) string {
	lower := strings.ToLower(chipset)

	// Intel chipsets
	if strings.Contains(lower, "z790") || strings.Contains(lower, "z690") ||
		strings.Contains(lower, "b760") || strings.Contains(lower, "b660") ||
		strings.Contains(lower, "700系列") || strings.Contains(lower, "600系列") ||
		strings.Contains(lower, "raptor lake") || strings.Contains(lower, "alder lake") {
		return "PCIe 5.0 / 4.0"
	}
	if strings.Contains(lower, "z590") || strings.Contains(lower, "z490") ||
		strings.Contains(lower, "b560") || strings.Contains(lower, "b460") ||
		strings.Contains(lower, "500系列") || strings.Contains(lower, "400系列") ||
		strings.Contains(lower, "comet") {
		return "PCIe 4.0 / 3.0"
	}
	if strings.Contains(lower, "z390") || strings.Contains(lower, "z370") ||
		strings.Contains(lower, "b360") || strings.Contains(lower, "300系列") ||
		strings.Contains(lower, "sunrise") || strings.Contains(lower, "cannon") {
		return "PCIe 3.0"
	}

	// AMD chipsets
	if strings.Contains(lower, "x670") || strings.Contains(lower, "b650") ||
		strings.Contains(lower, "x870") || strings.Contains(lower, "b850") {
		return "PCIe 5.0 / 4.0"
	}
	if strings.Contains(lower, "x570") {
		return "PCIe 4.0"
	}
	if strings.Contains(lower, "b550") || strings.Contains(lower, "x470") ||
		strings.Contains(lower, "b450") || strings.Contains(lower, "x370") ||
		strings.Contains(lower, "b350") {
		return "PCIe 4.0 / 3.0"
	}

	return "N/A"
}

// countM2Slots counts M.2 slots (estimated from chipset)
func countM2Slots() string {
	// M.2 slot count is not available through standard WMI/registry
	// This requires motherboard-specific data
	// We'll estimate based on chipset or return N/A
	return "N/A (需主板规格)"
}

// countSATAPorts counts SATA ports (estimated from chipset)
func countSATAPorts() string {
	// Try to count from registry
	count := 0
	basePath := `SYSTEM\CurrentControlSet\Enum\SCSI`
	subkeys := regSubKeyNames(registry.LOCAL_MACHINE, basePath)
	for _, sub := range subkeys {
		lower := strings.ToLower(sub)
		if strings.Contains(lower, "sata") || strings.Contains(lower, "ahci") {
			count++
		}
	}

	if count > 0 {
		return fmt.Sprintf("≥%d (估计)", count)
	}

	return "N/A (需主板规格)"
}

func GetBIOSInfo() BIOSInfo {
	info := BIOSInfo{}

	var biosDst []win32BIOS
	err := wmi.Query("SELECT * FROM Win32_BIOS", &biosDst)
	if err == nil && len(biosDst) > 0 {
		b := biosDst[0]
		info.Vendor = safeStringOr(b.Manufacturer, "N/A")
		info.Version = safeStringOr(strings.TrimSpace(b.Name), "N/A")
		info.SerialNumber = safeStringOr(strings.TrimSpace(b.SerialNumber), "N/A")

		// Parse BIOS release date
		dateStr := strings.TrimSpace(b.ReleaseDate)
		if len(dateStr) >= 8 {
			datePart := dateStr[:8]
			info.ReleaseDate = fmt.Sprintf("%s-%s-%s", datePart[:4], datePart[4:6], datePart[6:8])
		} else {
			info.ReleaseDate = safeStringOr(dateStr, "N/A")
		}
	} else {
		info.Vendor = "获取失败"
		info.Version = "N/A"
		info.ReleaseDate = "N/A"
		info.SerialNumber = "N/A"
	}

	// Detect UEFI vs Legacy using Windows API
	info.BootMode = getFirmwareType()
	info.SecureBoot = detectSecureBoot()

	return info
}

func detectSecureBoot() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\SecureBoot\State`, registry.QUERY_VALUE)
	if err != nil {
		return "不支持 / 未知"
	}
	defer k.Close()

	val, _, err := k.GetIntegerValue("UEFISecureBootEnabled")
	if err != nil {
		return "未知"
	}
	if val == 1 {
		return "已开启"
	}
	return "已关闭"
}
