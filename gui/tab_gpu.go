package gui

import (
	"fmt"

	"fyne.io/fyne/v2/container"
	"sysview/hardware"
)

func MakeGPUTab(gpus []hardware.GPUInfo) *container.Scroll {
	var sections []InfoSection

	// Build header from first GPU model name
	headerText := ""
	if len(gpus) > 0 && gpus[0].Model != "N/A" && gpus[0].Model != "" {
		headerText = gpus[0].Model
	}

	for i, gpu := range gpus {
		title := "显卡信息"
		if len(gpus) > 1 {
			title = fmt.Sprintf("显卡 #%d", i+1)
		}

		// Include model name in section title
		if gpu.Model != "N/A" && gpu.Model != "" {
			title = fmt.Sprintf("%s — %s", title, gpu.Model)
		}

		sections = append(sections, InfoSection{
			Title: title,
			Rows: []InfoRow{
				{Label: "型号", Value: gpu.Model},
				{Label: "核心代号", Value: gpu.Codename},
				{Label: "厂商", Value: gpu.Manufacturer},
				{Label: "显存类型", Value: gpu.VRAMType},
				{Label: "显存容量", Value: gpu.VRAMSize},
				{Label: "显存位宽", Value: gpu.BusWidth},
				{Label: "显存频率", Value: gpu.VRAMFrequency},
				{Label: "核心频率", Value: gpu.CoreFrequency},
				{Label: "流处理器数量", Value: gpu.StreamProcessors},
				{Label: "驱动版本", Value: gpu.DriverVersion},
				{Label: "DirectX/驱动日期", Value: gpu.DirectXVersion},
				{Label: "当前分辨率", Value: gpu.Resolution},
				{Label: "刷新率", Value: gpu.RefreshRate},
				{Label: "状态", Value: gpu.Availability},
			},
		})
	}

	if len(sections) == 0 {
		sections = append(sections, InfoSection{
			Title: "显卡信息",
			Rows:  []InfoRow{{Label: "状态", Value: "未检测到显卡"}},
		})
	}

	return makeInfoPanelWithHeader(sections, headerText)
}
