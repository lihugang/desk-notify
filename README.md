# DeskNotify 班级桌面通知投递系统

> 一体化轻量级班级桌面通知系统。专为教室多媒体大屏、白板与讲台电脑设计，后端采用 Go 语言构建，前端采用 Vue 3 + TypeScript + Tailwind CSS，支持 PWA 教师手机管理端，依托 frp 原生穿透与 SQLite3 嵌入式持久化存储。

---

## 🌟 核心特性

- 🖥️ **一体化运行架构**：班级 Windows 电脑运行单文件绿色版 `.exe`，后台静默工作，托盘常驻。
- 🔔 **桌面置顶卡片**：醒目、大字号通知悬浮窗（`alwaysOnTop`），自适应 Markdown 渲染，多图平铺与课件附件卡片展示，支持音效与倒计时自动隐去。
- 📱 **教师端 PWA 移动应用**：
  - 手机扫码即可直接访问，支持安装至手机桌面（PWA Standalone）。
  - 隐藏原始 Markdown 语法，提供直观的可视化轻量排版。
  - 支持截图后按 `Ctrl+V` 直接粘贴上传图片，支持多文件拖拽。
- 🔒 **智能网络与安全鉴权**：
  - 校园局域网内直接访问：默认免密。
  - 公网通过 frp 穿透访问：强制口令验证并下发 JWT 会话。
- 📂 **双层只读存储与硬链接**：
  - 通知附件与多媒体数据存入软件 `%APPDATA%/DeskNotify/data` 目录。
  - 自动通过 Win32 只读硬链接（`os.Link`）镜像至用户 `Documents/班级通知` 目录，防止误删与篡改。
- 🗄️ **嵌入式纯 Go SQLite3**：内置轻量高效的 SQLite3 存储引擎（无需安装任何外部数据库驱动）。
- ⚡ **Go Multi-Module Workspace 隔离机制**：本地开发不引入繁重庞大的 FRP 依赖包，秒级编译热重载；仅在 CI 构建最终产物时下载并静态链接 FRP SDK。

---

## 📁 目录结构

```text
desk-notify/
├── .github/
│   └── workflows/
│       └── release.yml          # GitHub Actions 自动化编译与 Release 流水线
├── core/                        # 核心 Go 应用
│   ├── main.go                  # 应用入口与生命周期管理
│   ├── embed.go                 # 嵌入打包好的静态 Web 前端资源
│   └── pkg/
│       ├── api/                 # HTTP/WebSocket 接口服务
│       ├── auth/                # 内网检测与 JWT 动态鉴权中间件
│       ├── config/              # 配置读写管理器 (config.json)
│       ├── db/                  # SQLite3 纯 Go 存储驱动与通知数据模型
│       ├── storage/             # 文件保存与 AppData/Documents 只读硬链接
│       └── tunnel/              # 穿透抽象接口与本地 Mock 实现
├── modules/
│   └── tunnel-frp/              # 独立的 FRP SDK 依赖模块 (隔离开发环境)
│       ├── go.mod
│       └── frp.go               # 基于 github.com/fatedier/frp 的内嵌穿透实现
├── frontend/                    # Vue 3 + TypeScript + Vite + TailwindCSS
│   ├── src/
│   │   ├── teacher/             # 教师管理与投递端 (PWA)
│   │   ├── card/                # 桌面置顶弹窗卡片 UI
│   │   ├── settings/            # 托盘设置与扫码投递二维码窗口
│   │   └── types/               # 全局 TypeScript 类型定义
│   ├── index.html               # 教师端 HTML 入口
│   ├── card.html                # 卡片端 HTML 入口
│   └── settings.html            # 设置端 HTML 入口
├── go.work                      # Go 工作区配置
└── .gitignore
```

---

## 🚀 本地开发指南 (Linux)

### 1. 前端开发与构建
```bash
cd frontend

# 安装依赖
pnpm install

# 启动开发热更新服务器 (默认端口 5173)
pnpm dev

# 编译并同步到后端 core/dist
pnpm run build
```

### 2. 后端核心开发与运行
由于配置了 Go Workspace 隔离机制，本地开发默认无需下载庞大的 FRP 仓库：
```bash
cd core

# 安装后端依赖
go mod tidy

# 启动服务 (访问 http://localhost:8920)
go run main.go
```

访问地址：
- **教师投递端 (PWA)**: `http://localhost:8920`
- **桌面通知卡片窗口**: `http://localhost:8920/card`
- **系统设置与二维码**: `http://localhost:8920/settings`

---

## 📦 GitHub Actions 编译与发布流程

只需在本地为 Git 提交打上版本标签并推送到 GitHub，即可自动触发 Windows 原生编译流水线：

```bash
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions 会自动在 `windows-latest` 环境中原生编译生成单个独立的绿色版 `DeskNotify.exe`，并自动发布到 GitHub Releases 页面提供下载。
