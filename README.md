<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Platform-Windows-0078D4?style=flat-square&logo=windows" alt="Platform">
  <img src="https://img.shields.io/badge/Architecture-x86__x64-green?style=flat-square" alt="Architecture">
  <img src="https://img.shields.io/badge/License-MIT-yellow?style=flat-square" alt="License">
</p>

# SysView

Windows 轻量级原生硬件信息查看工具。仿 CPU-Z 分页标签式 UI，全维度硬件参数一览，单文件免安装、零依赖、秒启动，支持 Win7/10/11 32/64 位。

## 功能特性

### 🖥 CPU 处理器
- 品牌/型号/代号/架构/微架构
- 物理/逻辑核心数、超线程状态
- 基础主频、睿频、总线频率、倍频
- 制程工艺、L1/L2/L3 缓存
- 指令集支持（MMX / SSE / AVX / AVX2 / AVX-512，基于 CPUID 运行时检测）
- TDP、插槽类型、处理器 ID、当前负载

### 🧠 内存
- 整机容量、已用/空闲/缓存内存、占用率
- 内存代数（DDR3/DDR4/DDR5）、通道模式（单/双/四通道自动检测）
- 每条内存：厂商、型号、序列号、容量、频率、等效频率、电压
- 插槽总数、已占用/空闲插槽数

### 🔧 主板 & BIOS
- 主板制造商/型号/版本/序列号/板型
- 芯片组识别（Intel 100-700 系列 / AMD 芯片组设备 ID 映射）
- 集成声卡/网卡/无线网卡型号
- BIOS 厂商/版本/日期、启动模式（UEFI/Legacy）、安全启动状态

### 🎮 显卡
- 显卡型号/厂商/核心代号（NVIDIA Ada/Ampere/Turing、AMD RDNA/Vega、Intel Arc）
- 显存类型/容量、驱动版本、DirectX 版本
- 分辨率/刷新率、可用状态
- 多显卡支持（独显+集显分别展示）

### 💾 硬盘存储
- 所有磁盘列表：SSD/HDD/NVMe 自动识别
- 磁盘型号/序列号/固件版本/容量/接口类型
- SMART 健康状态、通电时长、通电次数、温度
- 分区信息：盘符/标签/文件系统/容量/已用/剩余

### 🌐 网络 & 音频
- 有线/无线网卡：型号/MAC/IP/子网掩码/网关/DHCP/连接速度
- 自动过滤虚拟适配器（WAN Miniport、Bluetooth 等）
- 音频设备：型号/厂商/状态

### 🖲 系统信息
- 操作系统版本/版本类型（专业版/家庭版等）/内部版本号
- 计算算机名/主机名/用户名/工作组
- 开机运行时长、系统安装日期
- 区域语言、时区、系统目录

## 技术栈

| 组件 | 技术 |
|------|------|
| 开发语言 | Go 1.22+ |
| GUI 框架 | [Fyne v2](https://fyne.io/) — 跨平台原生渲染 |
| 硬件采集 | WMI + Windows 注册表 + PowerShell + Windows API |
| 指令集检测 | `golang.org/x/sys/cpu` — CPUID 原生检测 |
| 编译方式 | CGO + 静态链接，单 EXE 输出 |

## 项目结构

```
├── main.go               # 程序入口 & 窗口初始化
├── gui/
│   ├── loading.go        # 启动加载动画
│   ├── tab_cpu.go        # CPU 页面
│   ├── tab_mem.go        # 内存页面
│   ├── tab_mb.go         # 主板 + BIOS 页面
│   ├── tab_gpu.go        # 显卡页面
│   ├── tab_disk.go       # 硬盘存储页面
│   ├── tab_net.go        # 网络 & 音频页面
│   └── tab_sys.go        # 系统信息页面
├── hardware/
│   ├── cpu.go            # CPU 信息采集
│   ├── memory.go         # 内存信息采集
│   ├── motherboard.go    # 主板 & BIOS 采集
│   ├── gpu.go            # 显卡采集
│   ├── disk.go           # 硬盘 & 分区采集
│   ├── network.go        # 网络 & 音频采集
│   └── system.go         # 系统信息采集
├── go.mod
└── go.sum
```

## 快速开始

### 环境要求

- Go 1.22+
- CGO 编译环境（Windows 需要 [MinGW-w64](https://www.mingw-w64.org/)）
- 仅支持 Windows

### 安装依赖

```bash
go mod download
```

### 本地运行

```bash
go run .
```

### 编译打包

**64 位：**
```bash
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w -H windowsgui" -o SysView_x64.exe
```

**32 位：**（需要 32 位 MinGW，如 `i686` 版本）
```bash
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=386
go build -ldflags="-s -w -H windowsgui" -o SysView_x86.exe
```

> `-s -w` 去除调试信息压缩体积，`-H windowsgui` 隐藏控制台黑框窗口。

## 核心优势

- **极度轻量** — 单文件约 24 MB，启动毫秒级
- **零依赖** — 不依赖 .NET / VC++ / Electron / 浏览器内核
- **权限友好** — 无需管理员权限，不篡改系统
- **隐私安全** — 纯本地读取，无网络请求、不上传任何硬件信息
- **全兼容** — Win7 ~ Win11 全系 32 位 / 64 位适配
- **专业 UI** — 仿 CPU-Z 分页标签式界面，分类清晰、参数整齐

## 安全说明

- 仅**只读读取**硬件公开参数，无写入、无修改、无系统篡改
- 不读写敏感注册表项、不后台驻留、不弹窗广告
- 无进程后台残留，关闭即退出
- 纯离线本地工具，无任何数据上报、无隐私收集
- 绿色便携版，可放 U 盘直接运行

## License

[MIT](LICENSE)
