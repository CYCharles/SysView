package gui

import (
	"fmt"

	"fyne.io/fyne/v2/container"
	"sysview/hardware"
)

func MakeMBTab(mb hardware.MotherboardInfo, bios hardware.BIOSInfo) *container.Scroll {
	// Hero header: "Manufacturer Model"
	headerText := ""
	if mb.Manufacturer != "N/A" && mb.Model != "N/A" {
		headerText = fmt.Sprintf("%s %s", mb.Manufacturer, mb.Model)
	} else if mb.Model != "N/A" {
		headerText = mb.Model
	}

	sections := []InfoSection{
		{
			Title: "主板信息",
			Rows: []InfoRow{
				{Label: "制造商", Value: mb.Manufacturer},
				{Label: "型号", Value: mb.Model},
				{Label: "版本", Value: mb.Version},
				{Label: "序列号", Value: mb.SerialNumber},
				{Label: "板型", Value: mb.FormFactor},
				{Label: "芯片组", Value: mb.Chipset},
				{Label: "南桥", Value: mb.Southbridge},
			},
		},
		{
			Title: "接口与扩展",
			Rows: []InfoRow{
				{Label: "PCIe 版本", Value: mb.PCIeVersion},
				{Label: "M.2 接口数", Value: mb.M2Count},
				{Label: "SATA 接口数", Value: mb.SATACount},
			},
		},
		{
			Title: "集成设备",
			Rows: []InfoRow{
				{Label: "集成声卡", Value: mb.AudioChip},
				{Label: "集成网卡", Value: mb.LANChip},
				{Label: "无线网卡", Value: mb.WiFiChip},
			},
		},
		{
			Title: "系统信息",
			Rows: []InfoRow{
				{Label: "系统制造商", Value: mb.SystemModel},
				{Label: "系统类型", Value: mb.SystemType},
			},
		},
		{
			Title: "BIOS 信息",
			Rows: []InfoRow{
				{Label: "BIOS 厂商", Value: bios.Vendor},
				{Label: "BIOS 版本", Value: bios.Version},
				{Label: "BIOS 日期", Value: bios.ReleaseDate},
				{Label: "启动模式", Value: bios.BootMode},
				{Label: "安全启动", Value: bios.SecureBoot},
				{Label: "BIOS 序列号", Value: bios.SerialNumber},
			},
		},
	}

	return makeInfoPanelWithHeader(sections, headerText)
}
