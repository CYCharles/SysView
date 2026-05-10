package hardware

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/StackExchange/wmi"
	"golang.org/x/sys/windows"
)

type win32OperatingSystem struct {
	Caption                 string
	Version                 string
	BuildNumber             string
	OSArchitecture          string
	TotalVisibleMemorySize  uint64
	FreePhysicalMemory      uint64
	InstallDate             string
	LastBootUpTime           string
	Manufacturer            string
	SerialNumber            string
	SystemDirectory         string
	Locale                  string
	OSLanguage              uint32
	ProductType             uint32
	MUILanguages            []string
}

type win32ComputerSystemInfo struct {
	Manufacturer string
	Model        string
	UserName     string
	Workgroup    string
	Domain       string
	PartOfDomain bool
}

type win32TimeZone struct {
	Caption      string
	StandardName string
	Bias         int32
}

type SystemInfo struct {
	OSName         string
	OSVersion      string
	BuildNumber    string
	OSArchitecture string
	Edition        string
	InstallDate    string
	Hostname       string
	Workgroup      string
	ComputerName   string
	Uptime         string
	Locale         string
	Timezone       string
	Username       string
	SystemDir      string
	ProductType    string
	SerialNumber   string
	FirmwareType   string
}

func GetSystemInfo() SystemInfo {
	info := SystemInfo{}

	var osDst []win32OperatingSystem
	err := wmi.Query("SELECT * FROM Win32_OperatingSystem", &osDst)
	if err == nil && len(osDst) > 0 {
		os := osDst[0]
		info.OSName = safeStringOr(strings.TrimSpace(os.Caption), "N/A")
		info.OSVersion = safeStringOr(os.Version, "N/A")
		info.BuildNumber = safeStringOr(os.BuildNumber, "N/A")
		info.OSArchitecture = safeStringOr(os.OSArchitecture, "N/A")
		info.SerialNumber = safeStringOr(strings.TrimSpace(os.SerialNumber), "N/A")
		info.SystemDir = safeStringOr(os.SystemDirectory, "N/A")

		// Install date
		installDate := strings.TrimSpace(os.InstallDate)
		if len(installDate) >= 14 {
			info.InstallDate = fmt.Sprintf("%s-%s-%s %s:%s",
				installDate[:4], installDate[4:6], installDate[6:8],
				installDate[8:10], installDate[10:12])
		} else {
			info.InstallDate = "N/A"
		}

		// Product type
		switch os.ProductType {
		case 1:
			info.ProductType = "工作站"
		case 2:
			info.ProductType = "域控制器"
		case 3:
			info.ProductType = "服务器"
		default:
			info.ProductType = "未知"
		}

		// Locale
		info.Locale = getLocaleName(os.OSLanguage)

		// Uptime from LastBootUpTime
		bootTime := strings.TrimSpace(os.LastBootUpTime)
		if len(bootTime) >= 14 {
			t, err := parseWMITime(bootTime)
			if err == nil {
				uptime := time.Since(t)
				days := int(uptime.Hours()) / 24
				hours := int(uptime.Hours()) % 24
				minutes := int(uptime.Minutes()) % 60
				if days > 0 {
					info.Uptime = fmt.Sprintf("%d天 %d小时 %d分钟", days, hours, minutes)
				} else {
					info.Uptime = fmt.Sprintf("%d小时 %d分钟", hours, minutes)
				}
			} else {
				info.Uptime = "N/A"
			}
		} else {
			info.Uptime = "N/A"
		}
	} else {
		info.OSName = "获取失败"
		info.OSVersion = "N/A"
		info.BuildNumber = "N/A"
		info.OSArchitecture = "N/A"
	}

	// Computer name
	hostname, err := windows.ComputerName()
	if err == nil {
		info.Hostname = hostname
		info.ComputerName = hostname
	} else {
		info.Hostname = "N/A"
		info.ComputerName = "N/A"
	}

	// Get computer system info
	var sysDst []win32ComputerSystemInfo
	err = wmi.Query("SELECT * FROM Win32_ComputerSystem", &sysDst)
	if err == nil && len(sysDst) > 0 {
		s := sysDst[0]
		info.Username = safeStringOr(strings.TrimSpace(s.UserName), "N/A")
		if s.PartOfDomain {
			info.Workgroup = safeStringOr(s.Domain, "N/A") + " (域)"
		} else {
			info.Workgroup = safeStringOr(s.Workgroup, "N/A")
		}
	} else {
		info.Username = "N/A"
		info.Workgroup = "N/A"
	}

	// Edition from OS name
	if info.OSName != "N/A" && info.OSName != "获取失败" {
		name := info.OSName
		switch {
		case strings.Contains(name, "Professional"):
			info.Edition = "专业版"
		case strings.Contains(name, "Enterprise"):
			info.Edition = "企业版"
		case strings.Contains(name, "Home"):
			info.Edition = "家庭版"
		case strings.Contains(name, "Education"):
			info.Edition = "教育版"
		case strings.Contains(name, "Pro"):
			info.Edition = "专业版"
		case strings.Contains(name, "Ultimate"):
			info.Edition = "旗舰版"
		case strings.Contains(name, "Server"):
			info.Edition = "服务器版"
		default:
			info.Edition = "N/A"
		}
	} else {
		info.Edition = "N/A"
	}

	// Timezone
	info.Timezone = getTimezone()

	// Firmware type
	info.FirmwareType = getFirmwareType()

	return info
}

// getTimezone gets the system timezone
func getTimezone() string {
	// Try WMI first
	var tzDst []win32TimeZone
	err := wmi.Query("SELECT Caption, StandardName, Bias FROM Win32_TimeZone", &tzDst)
	if err == nil && len(tzDst) > 0 {
		tz := tzDst[0]
		bias := tz.Bias
		sign := "+"
		if bias > 0 {
			sign = "-" // WMI Bias is inverted: positive = west of UTC
		} else {
			bias = -bias
		}
		hours := bias / 60
		minutes := bias % 60

		var offset string
		if minutes > 0 {
			offset = fmt.Sprintf("UTC%s%d:%02d", sign, hours, minutes)
		} else {
			offset = fmt.Sprintf("UTC%s%d", sign, hours)
		}

		return fmt.Sprintf("%s (%s)", tz.Caption, offset)
	}

	// Fallback: use Windows API
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getTZI := kernel32.NewProc("GetTimeZoneInformation")
	if getTZI.Find() != nil {
		return "N/A"
	}

	// DynamicTimeZoneInformation struct (simplified)
	type dynamicTZInfo struct {
		Bias                int32
		StandardName        [32]uint16
		StandardDate        [16]byte
		StandardBias        int32
		DaylightName        [32]uint16
		DaylightDate        [16]byte
		DaylightBias        int32
		TimeZoneKeyName     [128]uint16
		DynamicDaylightTime int32
	}

	var tzi dynamicTZInfo
	ret, _, _ := getTZI.Call(uintptr(unsafe.Pointer(&tzi)))
	if ret != 0 {
		name := strings.TrimRight(utf16ToString(tzi.TimeZoneKeyName[:]), "\x00")
		if name == "" {
			name = strings.TrimRight(utf16ToString(tzi.StandardName[:]), "\x00")
		}

		bias := tzi.Bias
		sign := "+"
		if bias > 0 {
			sign = "-"
		} else {
			bias = -bias
		}
		hours := bias / 60
		minutes := bias % 60

		var offset string
		if minutes > 0 {
			offset = fmt.Sprintf("UTC%s%d:%02d", sign, hours, minutes)
		} else {
			offset = fmt.Sprintf("UTC%s%d", sign, hours)
		}

		if name != "" {
			return fmt.Sprintf("%s (%s)", name, offset)
		}
		return offset
	}

	return "N/A"
}

// utf16ToString converts UTF-16 array to Go string
func utf16ToString(s []uint16) string {
	var result []rune
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			break
		}
		result = append(result, rune(s[i]))
	}
	return string(result)
}

func parseWMITime(wmiTime string) (time.Time, error) {
	if len(wmiTime) < 14 {
		return time.Time{}, fmt.Errorf("invalid WMI time format")
	}

	year := wmiTime[0:4]
	month := wmiTime[4:6]
	day := wmiTime[6:8]
	hour := wmiTime[8:10]
	minute := wmiTime[10:12]
	second := wmiTime[12:14]

	tStr := fmt.Sprintf("%s-%s-%sT%s:%s:%sZ", year, month, day, hour, minute, second)
	return time.Parse(time.RFC3339, tStr)
}

func getLocaleName(langID uint32) string {
	localeMap := map[uint32]string{
		0x0404: "中文(繁体-台湾)",
		0x0804: "中文(简体-中国)",
		0x0409: "English (US)",
		0x0809: "English (UK)",
		0x0411: "日本語",
		0x0412: "한국어",
		0x0413: "Nederlands",
		0x0414: "Norsk",
		0x0407: "Deutsch",
		0x040C: "Français",
		0x040A: "Español",
		0x0410: "Italiano",
		0x0816: "Português (PT)",
		0x0416: "Português (BR)",
		0x0419: "Русский",
		0x0C04: "中文(繁体-香港)",
		0x1004: "中文(简体-新加坡)",
		0x1404: "中文(繁体-澳门)",
	}
	if name, ok := localeMap[langID]; ok {
		return name
	}
	return fmt.Sprintf("0x%04X", langID)
}

// GetSystemFirmwareType returns the firmware type (UEFI or Legacy BIOS)
func GetSystemFirmwareType() string {
	return getFirmwareType()
}

// unused but kept for reference
var _ = syscall.Errno(0)
