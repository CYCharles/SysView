package hardware

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sys/cpu"
	"golang.org/x/sys/windows/registry"
)

type CPUInfo struct {
	Brand            string
	Model            string
	Codename         string
	Architecture     string
	Cores            string
	Threads          string
	HyperThreading   string
	BaseClockMHz     string
	MaxBoostClockMHz string
	BusSpeedMHz      string
	Multiplier       string
	Process          string
	L1Cache          string
	L2Cache          string
	L3Cache          string
	MMX              string
	SSE              string
	AVX              string
	AVX2             string
	AVX512           string
	TDP              string
	Socket           string
	Voltage          string
	LoadPercent      string
	ProcessorID      string
}

// cleanCPUBrand maps WMI manufacturer strings to brand names
func cleanCPUBrand(manufacturer string) string {
	m := strings.TrimSpace(manufacturer)
	brandMap := map[string]string{
		"AuthenticAMD":   "AMD",
		"GenuineIntel":   "Intel",
		"CentaurHauls":   "VIA",
		"CyrixInstead":   "Cyrix",
		"TransmetaCPU":   "Transmeta",
		"GenuineTMx86":   "Transmeta",
		"RiseRiseRise":   "Rise",
		"SiS SiS SiS":    "SiS",
		"UMC UMC UMC":    "UMC",
		"VIA VIA VIA":    "VIA",
	}
	if mapped, ok := brandMap[m]; ok {
		return mapped
	}
	if m == "" {
		return "N/A"
	}
	return m
}

// cleanCPUName removes trailing whitespace and normalizes CPU name
func cleanCPUName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return "N/A"
	}
	return s
}

// extractCPUSeries extracts the 4-digit model number from a CPU name
func extractCPUSeries(name string) string {
	upper := strings.ToUpper(name)
	for i := 0; i < len(upper)-3; i++ {
		if upper[i] >= '0' && upper[i] <= '9' &&
			upper[i+1] >= '0' && upper[i+1] <= '9' &&
			upper[i+2] >= '0' && upper[i+2] <= '9' &&
			upper[i+3] >= '0' && upper[i+3] <= '9' {
			if i > 0 && upper[i-1] >= '0' && upper[i-1] <= '9' {
				continue
			}
			end := i + 4
			if end < len(upper) && upper[end] >= 'A' && upper[end] <= 'Z' {
				end++
			}
			return string(upper[i:end])
		}
	}
	return ""
}

func inferCodename(name string) string {
	upper := strings.ToUpper(name)

	// Intel Core Ultra
	if strings.Contains(upper, "ULTRA") {
		if strings.Contains(upper, "200") {
			return "Lunar Lake / Arrow Lake"
		}
		return "Meteor Lake"
	}

	// Intel Core - extract the model number
	if strings.Contains(upper, "CORE") {
		series := extractCPUSeries(name)
		if len(series) >= 4 {
			firstDigit := series[0]
			switch firstDigit {
			case '1':
				if series[1] == '4' {
					return "Raptor Lake Refresh"
				}
				if series[1] == '3' {
					return "Raptor Lake"
				}
				if series[1] == '2' {
					return "Alder Lake"
				}
				if series[1] == '1' {
					return "Rocket Lake / Tiger Lake"
				}
			case '9':
				return "Coffee Lake Refresh"
			case '8':
				return "Coffee Lake"
			case '7':
				return "Kaby Lake"
			case '6':
				return "Skylake"
			}
		}

		// Fallback: check generation keywords
		if strings.Contains(upper, "14TH") {
			return "Raptor Lake Refresh"
		}
		if strings.Contains(upper, "13TH") {
			return "Raptor Lake"
		}
		if strings.Contains(upper, "12TH") {
			return "Alder Lake"
		}
		if strings.Contains(upper, "11TH") {
			return "Rocket Lake / Tiger Lake"
		}
		if strings.Contains(upper, "10TH") {
			return "Comet Lake / Ice Lake"
		}
		if strings.Contains(upper, "9TH") {
			return "Coffee Lake Refresh"
		}
		if strings.Contains(upper, "8TH") {
			return "Coffee Lake"
		}
		if strings.Contains(upper, "7TH") {
			return "Kaby Lake"
		}
		if strings.Contains(upper, "6TH") {
			return "Skylake"
		}
	}

	// AMD Ryzen - use the first digit of the 4-digit model number
	if strings.Contains(upper, "RYZEN") {
		series := extractCPUSeries(name)
		if len(series) >= 4 {
			firstDigit := series[0]
			switch firstDigit {
			case '9':
				return "Zen 5 (Granite Ridge)"
			case '7':
				return "Zen 4 (Raphael)"
			case '5':
				if strings.Contains(upper, "G") || strings.Contains(upper, "GE") {
					return "Zen 3 (Cezanne)"
				}
				return "Zen 3 (Vermeer)"
			case '3':
				return "Zen 2 (Matisse)"
			case '2':
				return "Zen+ (Pinnacle Ridge)"
			case '1':
				return "Zen (Summit Ridge)"
			}
		}
		return "Zen"
	}

	// AMD EPYC
	if strings.Contains(upper, "EPYC") {
		series := extractCPUSeries(name)
		if len(series) >= 4 {
			switch series[0] {
			case '9':
				return "Zen 4 (Genoa)"
			case '7':
				return "Zen 3 (Milan)"
			case '3':
				return "Zen 2 (Rome)"
			}
		}
		return "Zen"
	}

	return "N/A"
}

// inferProcessFromCPUName tries to guess the manufacturing process from CPU name
func inferProcessFromCPUName(name string) string {
	upper := strings.ToUpper(name)

	// Intel Core Ultra
	if strings.Contains(upper, "ULTRA") {
		if strings.Contains(upper, "200") {
			return "3nm (TSMC) / 6nm (Intel 4)"
		}
		return "Intel 4 (7nm)"
	}

	// Intel Core - use series number
	if strings.Contains(upper, "CORE") {
		series := extractCPUSeries(name)
		if len(series) >= 4 {
			switch series[0] {
			case '1':
				if series[1] >= '2' {
					return "Intel 7 (10nm ESF)"
				}
				if series[1] == '1' {
					return "14nm++ / 10nm SF"
				}
			case '9':
				return "14nm++"
			case '8':
				return "14nm++"
			case '7':
				return "14nm+"
			case '6':
				return "14nm"
			}
		}
	}

	// AMD Ryzen - use series number
	if strings.Contains(upper, "RYZEN") {
		series := extractCPUSeries(name)
		if len(series) >= 4 {
			switch series[0] {
			case '9':
				return "4nm (TSMC)"
			case '7':
				return "5nm (TSMC)"
			case '5':
				return "7nm (TSMC)"
			case '3':
				return "7nm (TSMC) / 12nm I/O"
			case '2':
				return "12nm (TSMC) / 14nm I/O"
			case '1':
				return "14nm (GlobalFoundries)"
			}
		}
	}

	return "N/A"
}

// GetCPUInfo retrieves CPU information using PowerShell (avoids COM conflicts with Fyne GUI)
func GetCPUInfo() CPUInfo {
	info := CPUInfo{}

	// Use PowerShell to get all CPU data at once (avoids wmi package COM issues)
	psResult := runPowerShell(`
		$cpu = Get-CimInstance Win32_Processor | Select-Object -First 1
		[PSCustomObject]@{
			Name                       = $cpu.Name
			Manufacturer               = $cpu.Manufacturer
			Architecture               = $cpu.Architecture
			NumberOfCores              = $cpu.NumberOfCores
			NumberOfLogicalProcessors  = $cpu.NumberOfLogicalProcessors
			MaxClockSpeed             = $cpu.MaxClockSpeed
			CurrentClockSpeed         = $cpu.CurrentClockSpeed
			L2CacheSize               = $cpu.L2CacheSize
			L3CacheSize               = $cpu.L3CacheSize
			LoadPercentage            = if ($cpu.LoadPercentage) { $cpu.LoadPercentage } else { $null }
			ProcessorID               = $cpu.ProcessorID
			Voltage                   = if ($cpu.Voltage) { $cpu.Voltage } else { $null }
			UpgradeMethod             = $cpu.UpgradeMethod
		} | ConvertTo-Json -Compress
	`)

	if psResult == "" {
		info.Brand = "获取失败"
		info.Model = "获取失败"
		return info
	}

	// Parse JSON result - simple manual parsing to avoid extra dependencies
	name := extractJSONField(psResult, "Name")
	manufacturer := extractJSONField(psResult, "Manufacturer")
	architectureStr := extractJSONField(psResult, "Architecture")
	coresStr := extractJSONField(psResult, "NumberOfCores")
	threadsStr := extractJSONField(psResult, "NumberOfLogicalProcessors")
	maxClockStr := extractJSONField(psResult, "MaxClockSpeed")
	currentClockStr := extractJSONField(psResult, "CurrentClockSpeed")
	l2CacheStr := extractJSONField(psResult, "L2CacheSize")
	l3CacheStr := extractJSONField(psResult, "L3CacheSize")
	loadStr := extractJSONField(psResult, "LoadPercentage")
	processorID := extractJSONField(psResult, "ProcessorID")
	voltageStr := extractJSONField(psResult, "Voltage")
	upgradeMethodStr := extractJSONField(psResult, "UpgradeMethod")

	if name == "" || manufacturer == "" {
		info.Brand = "获取失败"
		info.Model = "获取失败"
		return info
	}

	info.Brand = cleanCPUBrand(manufacturer)
	info.Model = cleanCPUName(name)
	info.Codename = inferCodename(name)

	// Cores and threads
	if cores, err := strconv.Atoi(coresStr); err == nil {
		info.Cores = strconv.Itoa(cores)
	}
	if threads, err := strconv.Atoi(threadsStr); err == nil {
		info.Threads = strconv.Itoa(threads)
	}
	if info.Cores != "" && info.Threads != "" {
		c, _ := strconv.Atoi(info.Cores)
		t, _ := strconv.Atoi(info.Threads)
		if t > c {
			info.HyperThreading = "已开启"
		} else {
			info.HyperThreading = "未开启"
		}
	}

	// Clock speeds
	if currentClock, err := strconv.Atoi(currentClockStr); err == nil {
		info.BaseClockMHz = fmt.Sprintf("%d MHz", currentClock)
	}
	if maxClock, err := strconv.Atoi(maxClockStr); err == nil {
		info.MaxBoostClockMHz = fmt.Sprintf("%d MHz", maxClock)
	}

	// Bus speed from registry
	busSpeed := regReadString(registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor\0`,
		"BusSpeed", "")
	if busSpeed != "" {
		info.BusSpeedMHz = busSpeed + " MHz"
	} else {
		info.BusSpeedMHz = "N/A"
	}

	// Multiplier
	if currentClock, err := strconv.Atoi(currentClockStr); err == nil {
		if currentClock > 0 {
			info.Multiplier = fmt.Sprintf("x%d", currentClock/100) // approximate
		} else {
			info.Multiplier = "N/A"
		}
	}

	// Cache sizes
	l2, _ := strconv.Atoi(l2CacheStr)
	l3, _ := strconv.Atoi(l3CacheStr)
	c, _ := strconv.Atoi(info.Cores)
	if l2 > 0 {
		info.L2Cache = fmt.Sprintf("%d KB", l2)
	} else if c > 0 {
		info.L2Cache = fmt.Sprintf("%d KB", 32*c*4) // rough estimate
	} else {
		info.L2Cache = "N/A"
	}
	if l3 > 0 {
		if l3 >= 1024 {
			info.L3Cache = fmt.Sprintf("%.1f MB", float64(l3)/1024.0)
		} else {
			info.L3Cache = fmt.Sprintf("%d KB", l3)
		}
	} else {
		info.L3Cache = "N/A"
	}
	if c > 0 {
		info.L1Cache = fmt.Sprintf("%d KB", 64*c) // typical L1 per core
	} else {
		info.L1Cache = "N/A"
	}

	// Architecture
	if arch, err := strconv.Atoi(architectureStr); err == nil {
		switch arch {
		case 0:
			info.Architecture = "x86"
		case 5:
			info.Architecture = "ARM"
		case 9:
			info.Architecture = "x64"
		case 12:
			info.Architecture = "ARM64"
		default:
			info.Architecture = fmt.Sprintf("未知(%d)", arch)
		}
	}

	// Socket type
	socketMap := map[string]string{
		"1":  "Other", "2": "Unknown", "3": "Daughter Board", "4": "ZIF Socket",
		"5":  "Replacement/Piggy Back", "6": "None", "7": "LIF Socket",
		"8":  "Slot 1", "9": "Slot 2", "10": "370 PIN Socket",
		"11": "Slot A", "12": "Slot M", "13": "Socket 423",
		"14": "Socket A (462)", "15": "Socket 478", "16": "Socket 754",
		"17": "Socket 940", "18": "Socket 939", "19": "Socket mPGA604",
		"20": "Socket LGA771", "21": "Socket LGA775", "22": "Socket S1",
		"23": "Socket AM2", "24": "Socket F (1207)", "25": "Socket LGA1366",
		"26": "Socket G1", "27": "Socket AM3", "28": "Socket LGA1156",
		"29": "Socket LGA1567", "30": "Socket PGA988A", "31": "Socket BGA1288",
		"32": "Socket rPGA988B", "33": "Socket BGA1023", "34": "Socket BGA1224",
		"35": "Socket LGA1155", "36": "Socket LGA2011", "37": "Socket FS1",
		"38": "Socket FS2", "39": "Socket FM1", "40": "Socket FM2",
		"41": "Socket LGA2011-3", "42": "Socket LGA1356-3", "43": "Socket LGA1150",
		"44": "Socket BGA1168", "45": "Socket BGA1234", "46": "Socket BGA1364",
		"47": "Socket AM4", "48": "Socket LGA1151", "49": "Socket BGA1356",
		"50": "Socket BGA1440", "51": "Socket BGA1515", "52": "Socket LGA2066",
		"53": "Socket BGA1392", "54": "Socket BGA1510", "55": "Socket BGA1528",
	}
	if s, ok := socketMap[upgradeMethodStr]; ok {
		info.Socket = s
	} else {
		info.Socket = "N/A"
	}

	// Voltage
	if voltageStr != "" && voltageStr != "null" {
		if v, err := strconv.ParseFloat(voltageStr, 64); err == nil {
			volts := v / 10.0
			if volts > 20 {
				volts = v
			}
			info.Voltage = fmt.Sprintf("%.2f V", volts)
		} else {
			info.Voltage = "N/A"
		}
	} else {
		info.Voltage = "N/A"
	}

	// Load percentage
	if loadStr != "" && loadStr != "null" {
		if load, err := strconv.Atoi(loadStr); err == nil {
			info.LoadPercent = fmt.Sprintf("%d%%", load)
		} else {
			info.LoadPercent = "N/A"
		}
	} else {
		info.LoadPercent = "N/A"
	}

	// Instruction sets
	info.MMX = boolToChinese(cpu.X86.HasSSE2) // MMX is universal on modern CPUs
	info.SSE = boolToChinese(cpu.X86.HasSSE2)
	info.AVX = boolToChinese(cpu.X86.HasAVX)
	info.AVX2 = boolToChinese(cpu.X86.HasAVX2)
	info.AVX512 = boolToChinese(cpu.X86.HasAVX512F)

	info.ProcessorID = safeStringOr(processorID, "N/A")
	info.Process = inferProcessFromCPUName(name)
	info.TDP = getTDPFromRegistry()

	return info
}

// extractJSONField extracts a field value from simple flat JSON (no nested objects)
func extractJSONField(jsonStr, field string) string {
	// Look for "FieldName":"value" or "FieldName":number pattern
	searchKey := `"` + field + `":`
	idx := strings.Index(jsonStr, searchKey)
	if idx < 0 {
		return ""
	}
	start := idx + len(searchKey)
	remaining := jsonStr[start:]

	// Skip whitespace
	for len(remaining) > 0 && (remaining[0] == ' ' || remaining[0] == '\t') {
		remaining = remaining[1:]
		start++
	}
	if len(remaining) == 0 {
		return ""
	}

	if remaining[0] == '"' {
		// String value - find closing quote
		end := strings.Index(remaining[1:], `"`)
		if end < 0 {
			return ""
		}
		return remaining[1 : end+1]
	} else if remaining[0] == 'n' && strings.HasPrefix(remaining, "null") {
		return "null"
	} else {
		// Number value - find comma or closing brace
		var val []byte
		for _, ch := range remaining {
			if ch == ',' || ch == '}' || ch == ']' {
				break
			}
			val = append(val, byte(ch))
		}
		return strings.TrimSpace(string(val))
	}
}

func getTDPFromRegistry() string {
	tdp := regReadString(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\Class\{50127dc3-0f36-415e-a6cc-4cb3be871b7d}`,
		"TDP", "")
	if tdp != "" {
		return tdp + " W"
	}

	result := runPowerShell(`
		$cpu = Get-CimInstance Win32_Processor
		if ($cpu.ThermalDesignPower) { "$($cpu.ThermalDesignPower) W" } else { "" }
	`)
	if result != "" {
		return result
	}

	return "N/A"
}
