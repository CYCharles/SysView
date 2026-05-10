package gui

import (
	"fmt"

	"fyne.io/fyne/v2/container"
	"sysview/hardware"
)

func MakeSysTab(info hardware.SystemInfo) *container.Scroll {
	// Hero header shows OS name
	headerText := ""
	if info.OSName != "N/A" && info.OSName != "获取失败" {
		headerText = info.OSName
		if info.Edition != "N/A" && info.Edition != "" {
			headerText = fmt.Sprintf("%s %s", headerText, info.Edition)
		}
	}

	sections := []InfoSection{
		{
			Title: "操作系统",
			Rows: []InfoRow{
				{Label: "系统名称", Value: info.OSName},
				{Label: "版本", Value: info.OSVersion},
				{Label: "内部版本号", Value: info.BuildNumber},
				{Label: "系统位数", Value: info.OSArchitecture},
				{Label: "版本类型", Value: info.Edition},
				{Label: "产品类型", Value: info.ProductType},
			},
		},
		{
			Title: "计算机信息",
			Rows: []InfoRow{
				{Label: "主机名", Value: info.Hostname},
				{Label: "计算机名", Value: info.ComputerName},
				{Label: "工作组/域", Value: info.Workgroup},
				{Label: "当前用户", Value: info.Username},
			},
		},
		{
			Title: "运行信息",
			Rows: []InfoRow{
				{Label: "开机运行时长", Value: info.Uptime},
				{Label: "系统安装日期", Value: info.InstallDate},
				{Label: "区域语言", Value: info.Locale},
				{Label: "时区", Value: info.Timezone},
				{Label: "固件类型", Value: info.FirmwareType},
				{Label: "系统目录", Value: info.SystemDir},
				{Label: "序列号", Value: info.SerialNumber},
			},
		},
	}

	return makeInfoPanelWithHeader(sections, headerText)
}
