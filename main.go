package main

import (
	"sysview/gui"
	"sysview/hardware"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

func main() {
	a := app.NewWithID("com.sysview.app")
	a.Settings().SetTheme(theme.DefaultTheme())

	w := a.NewWindow("SysView")
	w.Resize(fyne.NewSize(850, 580))
	// Prevent resize during loading for cleaner appearance
	w.SetFixedSize(true)

	// Show loading screen first
	loading, setStatus := gui.LoadingScreen(
"SysView",
		"正在初始化...",
	)
	w.SetContent(loading)
	w.Show()

	// Load all data asynchronously and update UI when done
	go func() {
		setStatus("正在采集 CPU 信息...")
		cpuInfo := hardware.GetCPUInfo()

		setStatus("正在采集内存信息...")
		memInfo := hardware.GetMemoryInfo()

		setStatus("正在采集主板与 BIOS 信息...")
		mbInfo := hardware.GetMotherboardInfo()
		biosInfo := hardware.GetBIOSInfo()

		setStatus("正在采集显卡信息...")
		gpuInfo := hardware.GetGPUInfo()

		setStatus("正在采集硬盘信息...")
		diskInfo := hardware.GetDiskInfo()

		setStatus("正在采集网络与音频设备...")
		netInfo := hardware.GetNetworkInfo()
		audioInfo := hardware.GetAudioInfo()

		setStatus("正在采集系统信息...")
		sysInfo := hardware.GetSystemInfo()

		setStatus("加载完成！")

		// Build main tabs on the main thread
		a.Lifecycle().SetOnStopped(nil) // no-op to satisfy type

		tabs := container.NewAppTabs(
			container.NewTabItem("CPU", gui.MakeCPUTab(cpuInfo)),
			container.NewTabItem("内存", gui.MakeMemTab(memInfo)),
			container.NewTabItem("主板", gui.MakeMBTab(mbInfo, biosInfo)),
			container.NewTabItem("显卡", gui.MakeGPUTab(gpuInfo)),
			container.NewTabItem("硬盘", gui.MakeDiskTab(diskInfo)),
			container.NewTabItem("网络/音频", gui.MakeNetTab(netInfo, audioInfo)),
			container.NewTabItem("系统", gui.MakeSysTab(sysInfo)),
		)

		// Switch to main content on UI thread
		w.SetContent(tabs)
		w.SetFixedSize(false)
	}()

	a.Run()
}
