package hardware

import (
	"fmt"
	"strconv"
	"strings"
)

type MemoryStick struct {
	FullName        string // 完整型号: 厂商 + PartNumber + 类型 + 容量
	Manufacturer    string
	PartNumber      string
	SerialNumber    string
	Capacity        string
	Speed           string
	ConfiguredSpeed string
	EquivalentSpeed string
	MemoryType      string
	DeviceLocator   string
	BankLabel       string
	DataWidth       string
	TotalWidth      string
	Voltage         string
	TimingCL        string
}

type MemoryInfo struct {
	TotalMemory  string
	UsedMemory   string
	FreeMemory   string
	CachedMemory string
	UsagePercent string
	ChannelMode  string
	TotalSlots   string
	UsedSlots    string
	FreeSlots    string
	Sticks       []MemoryStick
}

// GetMemoryInfo retrieves memory information using PowerShell (avoids COM conflicts)
func GetMemoryInfo() MemoryInfo {
	info := MemoryInfo{}

	// Get overall memory info from OS using PowerShell
	psOS := runPowerShell(`
		$os = Get-CimInstance Win32_OperatingSystem | Select-Object -First 1
		PSCustomObject @{
			TotalVisibleMemorySize = $os.TotalVisibleMemorySize
			FreePhysicalMemory     = $os.FreePhysicalMemory
		} | ConvertTo-Json -Compress
	`)

	if psOS != "" {
		totalKBStr := extractJSONField(psOS, "TotalVisibleMemorySize")
		freeKBStr := extractJSONField(psOS, "FreePhysicalMemory")
		if totalKB, err := strconv.ParseUint(totalKBStr, 10, 64); err == nil {
			if freeKB, err := strconv.ParseUint(freeKBStr, 10, 64); err == nil {
				info.TotalMemory = formatBytes(totalKB * 1024)
				info.FreeMemory = formatBytes(freeKB * 1024)
				usedKB := totalKB - freeKB
				info.UsedMemory = formatBytes(usedKB * 1024)
				if totalKB > 0 {
					pct := float64(usedKB) / float64(totalKB) * 100
					info.UsagePercent = fmt.Sprintf("%.1f%%", pct)
				}
			}
		}
	}

	if info.TotalMemory == "" {
		// Fallback: try GlobalMemoryStatusEx via API
		totalPhys, availPhys, _ := getMemoryStatusEx()
		if totalPhys > 0 {
			info.TotalMemory = formatBytes(totalPhys)
			info.FreeMemory = formatBytes(availPhys)
			if availPhys <= totalPhys {
				info.UsedMemory = formatBytes(totalPhys - availPhys)
			}
			if totalPhys > 0 {
				pct := float64(totalPhys-availPhys) / float64(totalPhys) * 100
				info.UsagePercent = fmt.Sprintf("%.1f%%", pct)
			}
		} else {
			info.TotalMemory = "获取失败"
			info.UsedMemory = "N/A"
			info.FreeMemory = "N/A"
			info.UsagePercent = "N/A"
		}
	}

	// Cached memory
	info.CachedMemory = getCachedMemory()

	// Get physical memory array (slot count)
	psArray := runPowerShell(`
		$arr = Get-CimInstance Win32_PhysicalMemoryArray | Select-Object -First 1
		if ($arr) { "$($arr.MemoryDevices)" } else { "" }
	`)
	if psArray != "" {
		if slots, err := strconv.Atoi(psArray); err == nil {
			info.TotalSlots = strconv.Itoa(slots)
		}
	}
	if info.TotalSlots == "" {
		info.TotalSlots = "N/A"
	}

	// Get per-stick memory info using PowerShell
	psMem := runPowerShell(`
		Get-CimInstance Win32_PhysicalMemory | Select-Object Manufacturer, PartNumber, SerialNumber, Capacity, Speed, ConfiguredClockSpeed, SMBIOSMemoryType, DeviceLocator, BankLabel, DataWidth, TotalWidth, ConfiguredVoltage | ConvertTo-Json -Compress
	`)

	if psMem == "" || psMem == "null" || psMem == "[]" {
		info.UsedSlots = "0"
		freeSlots := "N/A"
		if info.TotalSlots != "N/A" {
			freeSlots = info.TotalSlots
		}
		info.FreeSlots = freeSlots
		info.ChannelMode = "N/A"
		return info
	}

	// Parse JSON array of memory sticks
	// Simple parser for flat JSON array
	stickCount := countJSONArrayItems(psMem)
	info.UsedSlots = strconv.Itoa(stickCount)

	// Calculate free slots
	if info.TotalSlots != "N/A" {
		var totalSlotCount int
		fmt.Sscanf(info.TotalSlots, "%d", &totalSlotCount)
		freeCount := totalSlotCount - stickCount
		if freeCount < 0 {
			freeCount = 0
		}
		info.FreeSlots = strconv.Itoa(freeCount)
	} else {
		info.FreeSlots = "0"
	}

	// Parse each stick from the JSON array
	items := splitJSONArrayItems(psMem)
	for _, item := range items {
		manufacturer := cleanMemManufacturer(safeStringOr(extractJSONField(item, "Manufacturer"), "未知"))
		partNumber := safeStringOr(strings.TrimSpace(extractJSONField(item, "PartNumber")), "N/A")
		memTypeValStr := extractJSONField(item, "SMBIOSMemoryType")
		var memTypeVal uint16
		if v, err := strconv.Atoi(memTypeValStr); err == nil {
			memTypeVal = uint16(v)
		}
		memType := getMemoryTypeName(memTypeVal)
		capacityStr := extractJSONField(item, "Capacity")
		capacityUint := uint64(0)
		if c, err := strconv.ParseUint(capacityStr, 10, 64); err == nil {
			capacityUint = c
		}
		capacity := formatBytes(capacityUint)

		stick := MemoryStick{
			Manufacturer: manufacturer,
			PartNumber:   partNumber,
			SerialNumber: safeStringOr(strings.TrimSpace(extractJSONField(item, "SerialNumber")), "N/A"),
			Capacity:     capacity,
			MemoryType:   memType,
			DeviceLocator: safeStringOr(extractJSONField(item, "DeviceLocator"), "N/A"),
			BankLabel:     safeStringOr(extractJSONField(item, "BankLabel"), "N/A"),
		}

		// Data width
		dwStr := extractJSONField(item, "DataWidth")
		twStr := extractJSONField(item, "TotalWidth")
		if dw, err := strconv.Atoi(dwStr); err == nil {
			stick.DataWidth = fmt.Sprintf("%d bit", dw)
		}
		if tw, err := strconv.Atoi(twStr); err == nil {
			stick.TotalWidth = fmt.Sprintf("%d bit", tw)
		}

		// Build full name
		fullParts := []string{}
		if manufacturer != "未知" && manufacturer != "N/A" {
			fullParts = append(fullParts, manufacturer)
		}
		if partNumber != "N/A" {
			fullParts = append(fullParts, partNumber)
		}
		speedVal, _ := strconv.Atoi(extractJSONField(item, "Speed"))
		if memType != "" && memType != "N/A" && speedVal > 0 {
			fullParts = append(fullParts, fmt.Sprintf("%s-%d", memType, speedVal))
		}
		if capacity != "" && capacity != "N/A" {
			fullParts = append(fullParts, capacity)
		}
		if len(fullParts) > 0 {
			stick.FullName = strings.Join(fullParts, " ")
		} else {
			stick.FullName = "N/A"
		}

		// Speed
		if speedVal > 0 {
			stick.Speed = fmt.Sprintf("%d MHz", speedVal)
			stick.EquivalentSpeed = fmt.Sprintf("%d MHz (DDR%d等效)", speedVal*2, getDDRGeneration(memTypeVal))
		} else {
			stick.Speed = "N/A"
			stick.EquivalentSpeed = "N/A"
		}

		// Configured speed
		cfgSpeed, _ := strconv.Atoi(extractJSONField(item, "ConfiguredClockSpeed"))
		if cfgSpeed > 0 {
			stick.ConfiguredSpeed = fmt.Sprintf("%d MHz", cfgSpeed)
		} else {
			stick.ConfiguredSpeed = "N/A"
		}

		// Voltage
		voltStr := extractJSONField(item, "ConfiguredVoltage")
		if voltStr != "" && voltStr != "null" {
			if v, err := strconv.ParseFloat(voltStr, 32); err == nil && v > 0 {
				stick.Voltage = fmt.Sprintf("%.2f V", v)
			} else {
				stick.Voltage = "N/A"
			}
		} else {
			stick.Voltage = "N/A"
		}

		stick.TimingCL = "N/A"

		info.Sticks = append(info.Sticks, stick)
	}

	// Determine channel mode
	info.ChannelMode = detectChannelModeFromSticks(info.Sticks)

	return info
}

// detectChannelModeFromSticks determines channel mode from parsed sticks
func detectChannelModeFromSticks(sticks []MemoryStick) string {
	if len(sticks) == 0 {
		return "N/A"
	}

	capacities := make(map[string]int)
	for _, s := range sticks {
		capacities[s.Capacity]++
	}

	if len(capacities) == 1 {
		switch {
		case len(sticks) >= 4:
			return "四通道"
		case len(sticks) >= 2:
			return "双通道"
		default:
			return "单通道"
		}
	}

	switch {
	case len(sticks) >= 4:
		return "多通道 (混合)"
	case len(sticks) >= 2:
		return "双通道 (估计)"
	default:
		return "单通道"
	}
}

// cleanMemManufacturer cleans up memory manufacturer names
func cleanMemManufacturer(name string) string {
	manufacturerMap := map[string]string{
		"080D": "Crucial (Micron)",
		"802C": "Samsung",
		"80CE": "Samsung",
		"0198": "Hyundai (Hynix)",
		"AD00": "SK Hynix",
		"04CD": "SK Hynix",
		"029E": "Ramaxel",
		"8567": "Ramaxel",
		"859B": "Crucial",
		"2C00": "Micron",
		"5986": "KVR (Kingston)",
		"9E1A": "Corsair",
		"00CE": "Kingston",
		"00BM": "G.SKILL",
		// Full names that WMI may return directly
		"Crucial":          "Crucial",
		"Samsung":          "Samsung",
		"SK Hynix":         "SK Hynix",
		"Kingston":         "Kingston",
		"Corsair":          "Corsair",
		"G.SKILL":          "G.SKILL",
		"Micron Technology": "Micron",
		"Elpida":           "Elpida",
		"Nanya":            "Nanya",
	}

	name = strings.TrimSpace(name)
	if mapped, ok := manufacturerMap[name]; ok {
		return mapped
	}
	name = strings.TrimPrefix(name, "Unknown-")
	name = strings.TrimPrefix(name, "Unknown ")
	if name == "" {
		return "未知"
	}
	return name
}

// getDDRGeneration returns DDR generation number from memory type
func getDDRGeneration(memType uint16) int {
	switch {
	case memType == 20:
		return 1
	case memType >= 21 && memType <= 22:
		return 2
	case memType >= 24 && memType <= 25:
		return 3
	case memType >= 26 && memType <= 30:
		return 4
	case memType >= 34 && memType <= 35:
		return 5
	default:
		return 0
	}
}

// getCachedMemory retrieves cached memory
func getCachedMemory() string {
	result := runPowerShell(`
		try {
			$counter = Get-Counter '\Memory\Cache Bytes' -ErrorAction Stop
			if ($counter.CounterSamples.CookedValue -gt 0) {
				[math]::Round($counter.CounterSamples.CookedValue)
			} else { "" }
		} catch { "" }
	`)
	if result != "" {
		var bytes uint64
		cleanResult := strings.ReplaceAll(result, ",", "")
		fmt.Sscanf(cleanResult, "%d", &bytes)
		if bytes > 0 {
			return formatBytes(bytes)
		}
	}
	return "N/A"
}

// --- JSON parsing helpers for flat/simple JSON ---

// countJSONArrayItems counts items in a JSON array like [{"a":"1"},{"b":"2"}]
func countJSONArrayItems(jsonArr string) int {
	jsonArr = strings.TrimSpace(jsonArr)
	if !strings.HasPrefix(jsonArr, "[") || !strings.HasSuffix(jsonArr, "]") {
		return 0
	}
	inner := jsonArr[1 : len(jsonArr)-1]
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return 0
	}
	// Count objects by counting }{ patterns (simplified)
	count := 0
	depth := 0
	inString := false
	escaped := false
	for _, ch := range inner {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				count++
			}
		}
	}
	return count
}

// splitJSONArrayItems splits a JSON array into individual object strings
func splitJSONArrayItems(jsonArr string) []string {
	jsonArr = strings.TrimSpace(jsonArr)
	if !strings.HasPrefix(jsonArr, "[") || !strings.HasSuffix(jsonArr, "]") {
		return nil
	}
	inner := jsonArr[1 : len(jsonArr)-1]
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return nil
	}

	var items []string
	start := 0
	depth := 0
	inString := false
	escaped := false

	for i, ch := range inner {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '{' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				items = append(items, inner[start:i+1])
			}
		}
	}
	return items
}
