package gui

import (
	"fyne.io/fyne/v2/container"
	"sysview/hardware"
)

func MakeCPUTab(info hardware.CPUInfo) *container.Scroll {
	// Hero header shows the full CPU model name prominently
	headerText := info.Model
	if headerText == "N/A" || headerText == "获取失败" {
		headerText = ""
	}

	sections := []InfoSection{
		{
			Title: "处理器基本信息",
			Rows: []InfoRow{
				{Label: "品牌", Value: info.Brand},
				{Label: "型号", Value: info.Model},
				{Label: "代号", Value: info.Codename},
				{Label: "架构", Value: info.Architecture},
			},
		},
		{
			Title: "核心与线程",
			Rows: []InfoRow{
				{Label: "物理核心数", Value: info.Cores},
				{Label: "逻辑线程数", Value: info.Threads},
				{Label: "超线程", Value: info.HyperThreading},
			},
		},
		{
			Title: "频率与缓存",
			Rows: []InfoRow{
				{Label: "当前主频", Value: info.BaseClockMHz},
				{Label: "最大睿频", Value: info.MaxBoostClockMHz},
				{Label: "总线频率", Value: info.BusSpeedMHz},
				{Label: "倍频", Value: info.Multiplier},
				{Label: "L1 缓存", Value: info.L1Cache},
				{Label: "L2 缓存", Value: info.L2Cache},
				{Label: "L3 缓存", Value: info.L3Cache},
			},
		},
		{
			Title: "指令集支持",
			Rows: []InfoRow{
				{Label: "MMX", Value: info.MMX},
				{Label: "SSE/SSE2", Value: info.SSE},
				{Label: "AVX", Value: info.AVX},
				{Label: "AVX2", Value: info.AVX2},
				{Label: "AVX-512", Value: info.AVX512},
			},
		},
		{
			Title: "其他参数",
			Rows: []InfoRow{
				{Label: "制程工艺", Value: info.Process},
				{Label: "TDP 功耗", Value: info.TDP},
				{Label: "插槽类型", Value: info.Socket},
				{Label: "电压", Value: info.Voltage},
				{Label: "当前负载", Value: info.LoadPercent},
				{Label: "处理器ID", Value: info.ProcessorID},
			},
		},
	}

	return makeInfoPanelWithHeader(sections, headerText)
}
