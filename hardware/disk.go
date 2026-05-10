package hardware

import (
	"fmt"
	"strings"

	"github.com/StackExchange/wmi"
	"golang.org/x/sys/windows/registry"
)

type win32DiskDrive struct {
	Model            string
	SerialNumber     string
	FirmwareRevision string
	Size             uint64
	InterfaceType    string
	MediaType        string
	Partitions       uint32
	BytesPerSector   uint32
	Index            uint32
	PNPDeviceID     string
}

type win32LogicalDisk struct {
	DeviceID    string
	FileSystem  string
	Size        uint64
	FreeSpace   uint64
	VolumeName  string
	DriveType   uint32
	Description string
}

type win32DiskPartition struct {
	DeviceID      string
	DiskIndex    uint32
	Index        uint32
	Bootable     bool
	BootPartition bool
	Size         uint64
	Type         string
}

type win32LogicalDiskToPartition struct {
	Antecedent string
	Dependent  string
}

// SMART-related WMI classes
type msStorageDriverFailurePredictStatus struct {
	InstanceName    string
	PredictFailure  bool
	Reason          uint8
}

type msStorageDriverFailurePredictData struct {
	InstanceName    string
	VendorSpecific  []uint8
}

type PartitionInfo struct {
	Letter       string
	Label        string
	FileSystem   string
	TotalSize    string
	UsedSpace    string
	FreeSpace    string
	UsagePercent string
}

type DiskInfo struct {
	Model           string
	SerialNumber    string
	Firmware        string
	Size            string
	InterfaceType   string
	MediaType       string
	PartitionCount  string
	PowerOnHours    string
	PowerCycleCount string
	HealthPercent   string
	Temperature     string
	Partitions      []PartitionInfo
}

func GetDiskInfo() []DiskInfo {
	var result []DiskInfo

	// Get disk drives
	var diskDst []win32DiskDrive
	err := wmi.Query("SELECT * FROM Win32_DiskDrive", &diskDst)
	if err != nil || len(diskDst) == 0 {
		return []DiskInfo{{
			Model:      "获取失败",
			Size:       "N/A",
			Partitions: []PartitionInfo{},
		}}
	}

	// Get SMART health status
	healthMap := getSmartHealthStatus()

	// Get partition-to-disk mapping
	partitionMap := getDiskPartitionMapping()

	for _, d := range diskDst {
		info := DiskInfo{
			Model:          safeStringOr(strings.TrimSpace(d.Model), "N/A"),
			SerialNumber:   safeStringOr(strings.TrimSpace(d.SerialNumber), "N/A"),
			Firmware:       safeStringOr(strings.TrimSpace(d.FirmwareRevision), "N/A"),
			InterfaceType:  safeStringOr(d.InterfaceType, "N/A"),
			PartitionCount: fmt.Sprintf("%d", d.Partitions),
		}

		// Size
		if d.Size > 0 {
			info.Size = formatBytes(d.Size)
		} else {
			info.Size = "N/A"
		}

		// Media type - improved detection
		info.MediaType = detectMediaType(d)

		// Update interface type for NVMe
		if strings.Contains(strings.ToLower(d.Model), "nvme") ||
			strings.Contains(strings.ToLower(d.PNPDeviceID), "nvme") {
			info.InterfaceType = "NVMe"
		}

		// SMART data
		pnpID := strings.ToLower(d.PNPDeviceID)
		if health, ok := healthMap[pnpID]; ok {
			info.HealthPercent = health
		} else {
			info.HealthPercent = getSmartHealthFromRegistry(d.PNPDeviceID)
		}

		// Power on hours and cycle count from SMART
		powerOn, powerCycle, temp := getSmartDetails(d.PNPDeviceID, d.Model)
		info.PowerOnHours = powerOn
		info.PowerCycleCount = powerCycle
		info.Temperature = temp

		// Get partitions for this disk
		if partitions, ok := partitionMap[d.Index]; ok {
			info.Partitions = partitions
		} else {
			// Fallback: get all partitions
			info.Partitions = getPartitions()
		}

		result = append(result, info)
	}

	return result
}

// detectMediaType determines if a disk is SSD or HDD
func detectMediaType(d win32DiskDrive) string {
	modelLower := strings.ToLower(d.Model)
	pnpLower := strings.ToLower(d.PNPDeviceID)
	mediaLower := strings.ToLower(d.MediaType)
	ifaceLower := strings.ToLower(d.InterfaceType)

	// NVMe is always SSD
	if strings.Contains(modelLower, "nvme") || strings.Contains(pnpLower, "nvme") {
		return "SSD (NVMe)"
	}

	// Check model name for SSD indicators
	if strings.Contains(modelLower, "ssd") {
		return "SSD"
	}

	// Check WMI MediaType
	if strings.Contains(mediaLower, "solid") || strings.Contains(mediaLower, "ssd") {
		return "SSD"
	}

	// Check registry for SSD detection
	// Windows 8+ stores this in the disk's registry key
	ssd := regReadString(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Enum\`+strings.ReplaceAll(d.PNPDeviceID, `\`, `\`)+`\Device Parameters`,
		"SolidStateDrive", "")
	if strings.EqualFold(ssd, "1") || strings.EqualFold(ssd, "true") {
		return "SSD"
	}

	// Try rotation rate from WMI (HDDs have >0)
	// This would need an additional WMI query field

	// Check interface type hints
	if strings.Contains(ifaceLower, "nvme") {
		return "SSD (NVMe)"
	}

	// Default based on common HDD patterns
	if strings.Contains(mediaLower, "fixed") || strings.Contains(mediaLower, "hard") {
		return "HDD"
	}

	return safeStringOr(strings.TrimSpace(d.MediaType), "N/A")
}

// getSmartHealthStatus gets SMART health from WMI
func getSmartHealthStatus() map[string]string {
	result := make(map[string]string)

	var dst []msStorageDriverFailurePredictStatus
	err := wmi.QueryNamespace("SELECT InstanceName, PredictFailure FROM MSStorageDriver_FailurePredictStatus", &dst, `root\wmi`)
	if err != nil {
		return result
	}

	for _, d := range dst {
		if d.PredictFailure {
			result[strings.ToLower(d.InstanceName)] = "警告 - 预测故障"
		} else {
			result[strings.ToLower(d.InstanceName)] = "良好 (100%)"
		}
	}

	return result
}

// getSmartHealthFromRegistry tries to get SMART status from registry
func getSmartHealthFromRegistry(pnpDeviceID string) string {
	if pnpDeviceID == "" {
		return "N/A"
	}

	// Try to read from the disk's WMI instance name pattern
	// The instance name in MSStorageDriver matches the PnP device ID
	pnpLower := strings.ToLower(pnpDeviceID)

	_ = pnpLower

	// Try PowerShell as fallback
	model := runPowerShell(`
		$disk = Get-CimInstance MSStorageDriver_FailurePredictStatus -Namespace root\wmi -ErrorAction SilentlyContinue |
			Where-Object { $_.InstanceName -like "*` + extractDiskInstanceID(pnpDeviceID) + `*" }
		if ($disk) {
			if ($disk.PredictFailure) { "警告" } else { "良好" }
		} else { "" }
	`)
	if model != "" {
		if model == "良好" {
			return "良好 (100%)"
		}
		return model
	}

	return "N/A"
}

// extractDiskInstanceID extracts a simplified instance ID from PnP ID
func extractDiskInstanceID(pnpDeviceID string) string {
	parts := strings.Split(pnpDeviceID, `\`)
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return pnpDeviceID
}

// getSmartDetails tries to get power-on hours, cycle count, temperature from SMART
func getSmartDetails(pnpDeviceID, model string) (powerOn, powerCycle, temperature string) {
	// Use PowerShell to read SMART data
	result := runPowerShell(fmt.Sprintf(`
		try {
			$disk = Get-CimInstance Win32_DiskDrive | Where-Object { $_.PNPDeviceID -eq '%s' }
			if ($disk) {
				$index = $disk.Index
				# Try to get SMART data from storage WMI
				$predictData = Get-CimInstance MSStorageDriver_FailurePredictData -Namespace root\wmi -ErrorAction SilentlyContinue |
					Where-Object { $_.InstanceName -like "*$($disk.PNPDeviceID.Split('\')[-1])*" }

				if ($predictData -and $predictData.VendorSpecific) {
					$vs = $predictData.VendorSpecific
					# Parse SMART attributes (vendor-specific format)
					# Each attribute: ID(1) + Flags(2) + Value(1) + Worst(1) + Raw(6) = 12 bytes
					$powerOnHours = ""
					$powerCycleCount = ""
					$temp = ""

					for ($i = 2; $i -lt $vs.Length - 12; $i += 12) {
						$attrId = $vs[$i]
						$rawValue = [BitConverter]::ToUInt32($vs, $i + 6)

						switch ($attrId) {
							9 { $powerOnHours = "$rawValue 小时" }
							12 { $powerCycleCount = "$rawValue 次" }
							194 { $temp = "$rawValue °C" }
						}
					}

					"$powerOnHours|$powerCycleCount|$temp"
				} else { "" }
			} else { "" }
		} catch { "" }
	`, pnpDeviceID))

	if result != "" {
		parts := strings.Split(result, "|")
		if len(parts) >= 3 {
			if parts[0] != "" {
				powerOn = parts[0]
			} else {
				powerOn = "N/A"
			}
			if parts[1] != "" {
				powerCycle = parts[1]
			} else {
				powerCycle = "N/A"
			}
			if parts[2] != "" {
				temperature = parts[2]
			} else {
				temperature = "N/A"
			}
			return
		}
	}

	return "N/A", "N/A", "N/A"
}

// getDiskPartitionMapping maps disk index to partition info
func getDiskPartitionMapping() map[uint32][]PartitionInfo {
	result := make(map[uint32][]PartitionInfo)

	// Get disk partitions
	var partDst []win32DiskPartition
	err := wmi.Query("SELECT * FROM Win32_DiskPartition", &partDst)
	if err != nil || len(partDst) == 0 {
		return result
	}

	// Get logical disk to partition associations
	var ldpDst []win32LogicalDiskToPartition
	_ = wmi.Query("SELECT Antecedent, Dependent FROM Win32_LogicalDiskToPartition", &ldpDst)

	// Build partition-to-logical-disk mapping
	partToLogical := make(map[string]string) // partition device ID -> logical disk letter
	for _, ldp := range ldpDst {
		// Parse the association strings
		// Antecedent: \\Computer\root\cimv2:Win32_DiskPartition.DeviceID="Disk #0, Partition #0"
		// Dependent: \\Computer\root\cimv2:Win32_LogicalDisk.DeviceID="C:"
		partRef := extractWMIRef(ldp.Antecedent, "DeviceID=")
		logRef := extractWMIRef(ldp.Dependent, "DeviceID=")

		if partRef != "" && logRef != "" {
			// Clean up the references
			partRef = strings.Trim(partRef, `"`)
			logRef = strings.Trim(logRef, `"`)
			partToLogical[partRef] = logRef
		}
	}

	// Get logical disk info for sizes
	var logDst []win32LogicalDisk
	_ = wmi.Query("SELECT * FROM Win32_LogicalDisk WHERE DriveType=3", &logDst)

	logDiskMap := make(map[string]win32LogicalDisk)
	for _, ld := range logDst {
		logDiskMap[ld.DeviceID] = ld
	}

	// Build partition info per disk
	for _, part := range partDst {
		// Find the logical disk letter for this partition
		partKey := part.DeviceID
		var letter, label, fileSystem, totalSize, usedSpace, freeSpace, usagePercent string

		if logLetter, ok := partToLogical[partKey]; ok {
			letter = logLetter
			if ld, ok2 := logDiskMap[logLetter]; ok2 {
				label = safeStringOr(strings.TrimSpace(ld.VolumeName), "本地磁盘")
				fileSystem = safeStringOr(ld.FileSystem, "N/A")

				if ld.Size > 0 {
					totalSize = formatBytes(ld.Size)
					if ld.FreeSpace <= ld.Size {
						used := ld.Size - ld.FreeSpace
						usedSpace = formatBytes(used)
						freeSpace = formatBytes(ld.FreeSpace)
						pct := float64(used) / float64(ld.Size) * 100
						usagePercent = fmt.Sprintf("%.1f%%", pct)
					}
				}
			}
		} else {
			letter = fmt.Sprintf("分区 %d", part.Index)
		}

		if totalSize == "" {
			if part.Size > 0 {
				totalSize = formatBytes(part.Size)
			} else {
				totalSize = "N/A"
			}
		}

		bootFlag := ""
		if part.BootPartition {
			bootFlag = " (启动)"
		}

		pInfo := PartitionInfo{
			Letter:       letter + bootFlag,
			Label:        label,
			FileSystem:   fileSystem,
			TotalSize:    totalSize,
			UsedSpace:    usedSpace,
			FreeSpace:    freeSpace,
			UsagePercent: usagePercent,
		}

		result[part.DiskIndex] = append(result[part.DiskIndex], pInfo)
	}

	return result
}

// extractWMIRef extracts a value from a WMI reference string
func extractWMIRef(ref, key string) string {
	idx := strings.Index(ref, key)
	if idx < 0 {
		return ""
	}
	value := ref[idx+len(key):]
	// Extract up to the next comma or closing paren
	end := strings.IndexAny(value, ",)")
	if end > 0 {
		return value[:end]
	}
	return value
}

// getPartitions is the fallback partition getter
func getPartitions() []PartitionInfo {
	var result []PartitionInfo

	var dst []win32LogicalDisk
	err := wmi.Query("SELECT * FROM Win32_LogicalDisk WHERE DriveType=3", &dst)
	if err != nil || len(dst) == 0 {
		return result
	}

	for _, d := range dst {
		p := PartitionInfo{
			Letter:     safeStringOr(d.DeviceID, "N/A"),
			Label:      safeStringOr(strings.TrimSpace(d.VolumeName), "本地磁盘"),
			FileSystem: safeStringOr(d.FileSystem, "N/A"),
		}

		if d.Size > 0 {
			p.TotalSize = formatBytes(d.Size)
		} else {
			p.TotalSize = "N/A"
		}

		if d.FreeSpace > 0 {
			p.FreeSpace = formatBytes(d.FreeSpace)
		} else {
			p.FreeSpace = "N/A"
		}

		if d.Size > 0 && d.FreeSpace <= d.Size {
			used := d.Size - d.FreeSpace
			p.UsedSpace = formatBytes(used)
			if d.Size > 0 {
				pct := float64(used) / float64(d.Size) * 100
				p.UsagePercent = fmt.Sprintf("%.1f%%", pct)
			} else {
				p.UsagePercent = "N/A"
			}
		} else {
			p.UsedSpace = "N/A"
					p.UsagePercent = "N/A"
		}

		result = append(result, p)
	}

	return result
}
