package hardware

import (
	"fmt"
	"net"
	"strings"

	"github.com/StackExchange/wmi"
)

type win32NetworkAdapter struct {
	Name               string
	Manufacturer       string
	MACAddress         string
	AdapterType        string
	Speed              uint64
	NetConnectionID    string
	NetConnectionStatus uint16
	PhysicalAdapter    bool
	Product            string
	Description        string
	DeviceID           string
}

type win32NetworkAdapterConfig struct {
	IPAddress         []string
	MACAddress        string
	DefaultIPGateway  []string
	IPSubnet          []string
	DNSServerSearchOrder []string
	DHCPEnabled       bool
	Description       string
	SettingID         string
}

type win32SoundDevice struct {
	Name         string
	Manufacturer string
	Status       string
	StatusInfo   uint16
}

type NetworkAdapter struct {
	Name         string
	Manufacturer string
	MACAddress   string
	IPAddress    string
	SubnetMask   string
	Gateway      string
	AdapterType  string
	Speed        string
	ConnectionID string
	Status       string
	DHCPEnabled  string
}

type AudioDevice struct {
	Name         string
	Manufacturer string
	Status       string
}

func GetNetworkInfo() []NetworkAdapter {
	var result []NetworkAdapter

	var adapterDst []win32NetworkAdapter
	err := wmi.Query("SELECT * FROM Win32_NetworkAdapter WHERE PhysicalAdapter=True OR NetConnectionID IS NOT NULL", &adapterDst)
	if err != nil || len(adapterDst) == 0 {
		// Fallback: try all adapters
		err = wmi.Query("SELECT * FROM Win32_NetworkAdapter", &adapterDst)
		if err != nil || len(adapterDst) == 0 {
			return []NetworkAdapter{{Name: "获取失败"}}
		}
	}

	// Get configuration for each adapter
	var configDst []win32NetworkAdapterConfig
	_ = wmi.Query("SELECT * FROM Win32_NetworkAdapterConfiguration WHERE IPEnabled=True", &configDst)

	configMap := make(map[string]win32NetworkAdapterConfig)
	for _, c := range configDst {
		mac := strings.TrimSpace(c.MACAddress)
		if mac != "" {
			configMap[mac] = c
		}
	}

	for _, a := range adapterDst {
		// Skip non-physical or disconnected adapters with no useful info
		name := strings.TrimSpace(a.Name)
		if name == "" {
			continue
		}

		// Skip WAN Miniport, Bluetooth, etc.
		lowerName := strings.ToLower(name)
		if strings.Contains(lowerName, "wan miniport") ||
			strings.Contains(lowerName, "bluetooth") ||
			strings.Contains(lowerName, "ras") ||
			strings.Contains(lowerName, "6to4") ||
			strings.Contains(lowerName, "isatap") ||
			strings.Contains(lowerName, "teredo") {
			continue
		}

		info := NetworkAdapter{
			Name:         name,
			Manufacturer: safeStringOr(a.Manufacturer, "N/A"),
			MACAddress:   safeStringOr(strings.TrimSpace(a.MACAddress), "N/A"),
			AdapterType:  safeStringOr(a.AdapterType, "N/A"),
			ConnectionID: safeStringOr(a.NetConnectionID, "N/A"),
		}

		// Connection status
		switch a.NetConnectionStatus {
		case 0:
			info.Status = "已断开"
		case 1:
			info.Status = "正在连接"
		case 2:
			info.Status = "已连接"
		case 3:
			info.Status = "正在断开"
		case 4:
			info.Status = "硬件不存在"
		case 5:
			info.Status = "硬件被禁用"
		case 6:
			info.Status = "硬件故障"
		case 7:
			info.Status = "媒体断开"
		case 8:
			info.Status = "正在验证"
		case 9:
			info.Status = "验证成功"
		default:
			info.Status = "未知"
		}

		// Speed
		if a.Speed > 0 {
			speed := a.Speed
			if speed >= 1_000_000_000 {
				info.Speed = fmt.Sprintf("%.1f Gbps", float64(speed)/1_000_000_000.0)
			} else if speed >= 1_000_000 {
				info.Speed = fmt.Sprintf("%.0f Mbps", float64(speed)/1_000_000.0)
			} else {
				info.Speed = fmt.Sprintf("%d bps", speed)
			}
		} else {
			info.Speed = "N/A"
		}

		// Try to match with config
		mac := strings.TrimSpace(a.MACAddress)
		if config, ok := configMap[mac]; ok {
			if len(config.IPAddress) > 0 {
				info.IPAddress = strings.Join(config.IPAddress, ", ")
			} else {
				info.IPAddress = "N/A"
			}
			if len(config.IPSubnet) > 0 {
				info.SubnetMask = config.IPSubnet[0]
			} else {
				info.SubnetMask = "N/A"
			}
			if len(config.DefaultIPGateway) > 0 {
				info.Gateway = config.DefaultIPGateway[0]
			} else {
				info.Gateway = "N/A"
			}
			if config.DHCPEnabled {
				info.DHCPEnabled = "已开启"
			} else {
				info.DHCPEnabled = "已关闭"
			}
		} else {
			info.IPAddress = "N/A"
			info.SubnetMask = "N/A"
			info.Gateway = "N/A"
			info.DHCPEnabled = "N/A"
		}

		// Format MAC address with colons
		if info.MACAddress != "N/A" {
			info.MACAddress = formatMAC(info.MACAddress)
		}

		result = append(result, info)
	}

	if len(result) == 0 {
		result = append(result, NetworkAdapter{Name: "未检测到网络适配器"})
	}

	return result
}

func GetAudioInfo() []AudioDevice {
	var result []AudioDevice

	var dst []win32SoundDevice
	err := wmi.Query("SELECT * FROM Win32_SoundDevice", &dst)
	if err != nil || len(dst) == 0 {
		return []AudioDevice{{Name: "获取失败"}}
	}

	for _, d := range dst {
		info := AudioDevice{
			Name:         safeStringOr(d.Name, "N/A"),
			Manufacturer: safeStringOr(d.Manufacturer, "N/A"),
		}

		switch d.Status {
		case "OK":
			info.Status = "正常"
		case "Error":
			info.Status = "错误"
		case "Degraded":
			info.Status = "降级"
		default:
			info.Status = safeStringOr(d.Status, "未知")
		}

		result = append(result, info)
	}

	return result
}

func formatMAC(mac string) string {
	// Normalize MAC address format
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return mac
	}
	return hw.String()
}
