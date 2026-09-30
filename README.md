# FnOSDiskInfo

A CrystalDiskInfo-like disk health viewer for fnOS, packaged as a native FPK app.
Backend: Go (stdlib only) + `smartctl`; frontend: dependency-free HTML/CSS/JS with a
dense, CDI-style UI.

把 CrystalDiskInfo 的信息密度和交互搬到 fnOS：Go 后端调用系统 `smartctl` 采集，
无依赖网页面板展示，支持导入 CDI 主题包与 Shizuku 风格的几何布局。

## 功能

- 磁盘选择条（CDI 风格按钮、健康/温度/寿命状态、分页）
- 健康引擎：good / caution / bad / unknown，可配置 caution 阈值，带原因列表
- SMART 属性表：CDI 语义的左侧 LED、斑马纹、属性名本地化（en / zh-CN / zh-TW / ja，
  按 CDI 的厂商表匹配：Kingston / WDC / Samsung / Intel / Kioxia / SK hynix / Micron 等）
- NVMe：由 SMART/Health 日志合成 CDI 的 15 行伪属性表（含 LED 判定）、NVM Express 标准与
  主机读写总量；SATA SSD：CDI 同款寿命 / 主机读写 / NAND 写入规则
- 温度、寿命、转速/SSD、传输模式、标准、特性；信息格悬停显示完整值/说明（传输模式、标准、特性等）
- 历史曲线（Graph / 图表选项，移植 CDI 的 HTML 对话框，内置 flot）
- 告警（横幅 + 告警列表）、SMART 自检 / 中止、AAM / APM 查询与设置
- 报告导出（CDI 风格 report.txt、原始 smartctl JSON）、当前视图导出 PNG
- 主题：内置 Light / Dark / Follow fnOS；可导入 CDI 主题包（zip），支持父主题与
  回退素材链（与 CDI `IP()` 的四级回退一致），菜单栏跟随 fnOS 深浅色
- 显示偏好：缩放、字体、温标、原始值格式、隐藏序列号/表格、磁盘排序等

## 安装

1. 从 [Nightly release](https://github.com/jlj1102/FnOSDiskInfo/releases/tag/nightly)
   下载 `cdifnos.fpk`
2. fnOS 应用中心 → 手动安装 → 选择该 fpk
3. 打开应用（默认端口 7817）

依赖系统自带的 `/usr/sbin/smartctl`（fnOS 自带 7.3，无需额外安装）。

## 运行方式与权限

- 应用以 **root 模式**运行，但拆成两个进程：root 的 `collect` 子进程调用 smartctl 并
  原子写入 `$TRIM_PKGVAR/cache.json`；网页服务 `serve` 通过 `runuser` 降权到应用用户运行，
  不接触磁盘也不调用 smartctl
- 跟随宿主深浅色使用 fnOS 官方 JS SDK，需要 fnOS ≥ 1.2.0401（App ≥ 1.34.0）；
  更低版本自动退回系统 `prefers-color-scheme`

## 已知限制 / 未验证

> 以下功能尚未在真实硬件上验证过，欢迎提交样本或反馈问题。

- **NVMe：解析已按真机样本（SK hynix BC501，smartctl 7.3）验证**，属性表、LED、寿命、
  主机读写量与 NVM Express 标准均已接线；但 PCIe 传输速率（smartctl JSON 不提供）仍显示
  `--`，VolatileWriteCache 特性无数据来源，其他 NVMe 设备/桥接的兼容性未验证
- **USB：未验证。** USB 硬盘盒依赖 `smartctl --scan-open` 报出的 `-d sat/usb` 类型
  透传，没有做任何专门处理，可能可用但未测试
- **SAS / HBA / RAID 卡：没有专门支持**，仅按 smartctl 默认路径工作
- SSD 厂商识别与数值规则是从 CDI 移植的尽力而为版本（含 36 个厂商判定），未覆盖到的
  型号回落通用表（SmartSsd）与 smartctl 英文名
- ATA 特性里的 DevSleep / Streaming 没有数据来源，不显示；NCQ 由
  NCQ Command Error 日志（GP log 0x10）推断
- 导出的 PNG 不包含毛玻璃模糊（canvas 无法复制 `backdrop-filter`）
- 主题兼容为尽力而为（CDI L1/L2），不追求 Windows 像素级一致

## 开发 / 构建

```
src/        Go 1.26 源码（仅标准库），web 前端以 go:embed 打包
cdifnos/    FPK 包根（manifest、cmd、config、ui、图标）
build.ps1   vet + test + 交叉编译 linux/amd64 + fnpack 打包
```

- 构建：`./build.ps1`。fnpack（仅 Windows 版）不随仓库分发，构建/CI 时从官方地址
  `https://static2.fnnas.com/fnpack/fnpack-1.2.3-windows-amd64` 下载到 `tools/`
- 本地调试：`go -C src build -o cdi-dev.exe .`，并用
  `go -C src build -o ..\fake\smartctl.exe ./testdata/fake-smartctl` 生成假 smartctl，
  把 `fake\` 加到 PATH 后运行即可（无需真实硬盘）
- 测试：`go -C src vet ./... && go -C src test ./...`
- 发布：push 到 `master` 后由 GitHub Actions 自动打包并更新 `nightly` pre-release

## 第三方与许可

本项目以 MIT 许可发布（见 `LICENSE`）。

- [CrystalDiskInfo](https://github.com/hiyohiyo/CrystalDiskInfo)（MIT）：SMART 属性名、
  LED 图标（`src/web/icons/led_*.png` 由 CDI 的 `res/*.ico` 转换）、健康判定与
  UI 行为/几何布局的参考实现。**仓库不包含任何 CDI 角色/主题素材**，主题请自行导入
  你本地的 CDI 主题包
- jQuery + flot（MIT）：内置在 `src/web/flot/`，用于 Graph 窗口
- fnOS 官方 JS SDK `@trimjs/web-app`（内置在 `src/web/trim/`）：读取宿主主题
