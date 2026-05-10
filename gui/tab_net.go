package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2/container"
	"sysview/hardware"
)

func MakeNetTab(nets []hardware.NetworkAdapter, audios []hardware.AudioDevice) *container.Scroll {
	var sections []InfoSection

	// Header shows network adapter count
	headerText := ""
	if len(nets) > 0 {
		headerText = fmt.Sprintf("%d 个网络适配器", len(nets))
	}

	// Network adapters
	for i, net := range nets {
		title := "网络适配器"
		if len(nets) > 1 {
			title = fmt.Sprintf("网络适配器 #%d", i+1)
		}

		// Include adapter name in section title
		if net.Name != "N/A" && net.Name != "" {
			title = fmt.Sprintf("%s — %s", title, net.Name)
		}

		adapterType := "有线"
		name := net.Name
		if strings.Contains(name, "Wireless") || strings.Contains(name, "Wi-Fi") ||
			strings.Contains(name, "WLAN") || strings.Contains(name, "802.11") ||
			strings.Contains(name, "无线") {
			adapterType = "无线"
		}

		rows := []InfoRow{
			{Label: "名称", Value: net.Name},
			{Label: "类型", Value: adapterType},
			{Label: "厂商", Value: net.Manufacturer},
			{Label: "MAC 地址", Value: net.MACAddress},
			{Label: "IP 地址", Value: net.IPAddress},
			{Label: "子网掩码", Value: net.SubnetMask},
			{Label: "网关", Value: net.Gateway},
			{Label: "连接速度", Value: net.Speed},
			{Label: "连接名", Value: net.ConnectionID},
			{Label: "状态", Value: net.Status},
			{Label: "DHCP", Value: net.DHCPEnabled},
		}

		sections = append(sections, InfoSection{
			Title: title,
			Rows:  rows,
		})
	}

	if len(nets) == 0 {
		sections = append(sections, InfoSection{
			Title: "网络适配器",
			Rows:  []InfoRow{{Label: "状态", Value: "未检测到网络适配器"}},
		})
	}

	// Audio devices
	for i, audio := range audios {
		title := "音频设备"
		if len(audios) > 1 {
			title = fmt.Sprintf("音频设备 #%d", i+1)
		}

		// Include device name in section title
		if audio.Name != "N/A" && audio.Name != "" {
			title = fmt.Sprintf("%s — %s", title, audio.Name)
		}

		sections = append(sections, InfoSection{
			Title: title,
			Rows: []InfoRow{
				{Label: "设备名称", Value: audio.Name},
				{Label: "制造商", Value: audio.Manufacturer},
				{Label: "状态", Value: audio.Status},
			},
		})
	}

	if len(audios) == 0 {
		sections = append(sections, InfoSection{
			Title: "音频设备",
			Rows:  []InfoRow{{Label: "状态", Value: "未检测到音频设备"}},
		})
	}

	return makeInfoPanelWithHeader(sections, headerText)
}
