package gui

import (
	"fmt"

	"fyne.io/fyne/v2/container"
	"sysview/hardware"
)

func MakeDiskTab(disks []hardware.DiskInfo) *container.Scroll {
	var sections []InfoSection

	// Header shows disk count
	headerText := ""
	if len(disks) > 0 {
		headerText = fmt.Sprintf("%d 块磁盘", len(disks))
	}

	for i, disk := range disks {
		// Include model name in section title
		title := fmt.Sprintf("磁盘 #%d", i+1)
		if disk.Model != "N/A" && disk.Model != "" {
			title = fmt.Sprintf("磁盘 #%d — %s", i+1, disk.Model)
		}

		rows := []InfoRow{
			{Label: "型号", Value: disk.Model},
			{Label: "序列号", Value: disk.SerialNumber},
			{Label: "固件版本", Value: disk.Firmware},
			{Label: "容量", Value: disk.Size},
			{Label: "接口类型", Value: disk.InterfaceType},
			{Label: "媒体类型", Value: disk.MediaType},
			{Label: "分区数", Value: disk.PartitionCount},
			{Label: "通电时长", Value: disk.PowerOnHours},
			{Label: "通电次数", Value: disk.PowerCycleCount},
			{Label: "健康度", Value: disk.HealthPercent},
			{Label: "温度", Value: disk.Temperature},
		}

		sections = append(sections, InfoSection{
			Title: title,
			Rows:  rows,
		})

		// Add partition info
		if len(disk.Partitions) > 0 {
			var partRows []InfoRow
			for _, p := range disk.Partitions {
				partRows = append(partRows, []InfoRow{
					{Label: "盘符", Value: p.Letter},
					{Label: "卷标", Value: p.Label},
					{Label: "文件系统", Value: p.FileSystem},
					{Label: "总大小", Value: p.TotalSize},
					{Label: "已使用", Value: p.UsedSpace},
					{Label: "剩余", Value: p.FreeSpace},
					{Label: "使用率", Value: p.UsagePercent},
				}...)

				// Add separator between partitions
				partRows = append(partRows, InfoRow{Label: "", Value: "─ 分隔 ─"})
			}

			sections = append(sections, InfoSection{
				Title: fmt.Sprintf("磁盘 #%d 分区列表", i+1),
				Rows:  partRows,
			})
		}
	}

	if len(sections) == 0 {
		sections = append(sections, InfoSection{
			Title: "磁盘信息",
			Rows:  []InfoRow{{Label: "状态", Value: "未检测到磁盘"}},
		})
	}

	return makeInfoPanelWithHeader(sections, headerText)
}
