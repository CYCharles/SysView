package gui

import (
	"fmt"

	"fyne.io/fyne/v2/container"
	"sysview/hardware"
)

func MakeMemTab(info hardware.MemoryInfo) *container.Scroll {
	// Hero header shows total memory summary
	headerText := ""
	if info.TotalMemory != "N/A" && info.TotalMemory != "获取失败" {
		headerText = fmt.Sprintf("%s %s × %s %s", info.TotalMemory, info.ChannelMode, info.UsedSlots, "条")
	}

	sections := []InfoSection{
		{
			Title: "整机内存概览",
			Rows: []InfoRow{
				{Label: "总容量", Value: info.TotalMemory},
				{Label: "已使用", Value: info.UsedMemory},
				{Label: "空闲", Value: info.FreeMemory},
				{Label: "缓存", Value: info.CachedMemory},
				{Label: "占用率", Value: info.UsagePercent},
			},
		},
		{
			Title: "插槽与通道",
			Rows: []InfoRow{
				{Label: "通道模式", Value: info.ChannelMode},
				{Label: "总插槽数", Value: info.TotalSlots},
				{Label: "已使用插槽", Value: info.UsedSlots},
				{Label: "空闲插槽", Value: info.FreeSlots},
			},
		},
	}

	// Add per-stick info
	for i, stick := range info.Sticks {
		// Section title includes the full model name
		sectionTitle := fmt.Sprintf("内存条 #%d", i+1)
		if stick.FullName != "N/A" && stick.FullName != "" {
			sectionTitle = fmt.Sprintf("内存条 #%d — %s", i+1, stick.FullName)
		}

		sections = append(sections, InfoSection{
			Title: sectionTitle,
			Rows: []InfoRow{
				{Label: "完整型号", Value: stick.FullName},
				{Label: "厂商", Value: stick.Manufacturer},
				{Label: "部件号", Value: stick.PartNumber},
				{Label: "序列号", Value: stick.SerialNumber},
				{Label: "容量", Value: stick.Capacity},
				{Label: "类型", Value: stick.MemoryType},
				{Label: "标称频率", Value: stick.Speed},
				{Label: "等效频率", Value: stick.EquivalentSpeed},
				{Label: "工作频率", Value: stick.ConfiguredSpeed},
				{Label: "数据宽度", Value: stick.DataWidth},
				{Label: "总宽度", Value: stick.TotalWidth},
				{Label: "电压", Value: stick.Voltage},
				{Label: "时序CL", Value: stick.TimingCL},
				{Label: "插槽位置", Value: stick.DeviceLocator},
				{Label: "Bank 标签", Value: stick.BankLabel},
			},
		})
	}

	return makeInfoPanelWithHeader(sections, headerText)
}
