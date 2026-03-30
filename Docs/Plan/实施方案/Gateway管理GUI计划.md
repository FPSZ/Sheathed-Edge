# Gateway 管理 GUI 计划

## Summary

在 [Docs/Plan](D:\AI\Local\Docs\Plan) 下新增一份 GUI 规划文档，归类到 `实施方案` 目录。  
这份文档聚焦一个新的 **浏览器优先 Admin 控制台**，不替代 Open WebUI 聊天入口，只负责本机系统控制、状态观测和日志查看。

V1 固定定位如下：

- 形态：本地浏览器管理台
- 角色：`Gateway Admin Console`
- 风格来源：
  - `Memori-Vault`：负责视觉语言、控制台气质、卡片层次、面板密度
  - `AstrBot`：负责后台布局、导航结构、配置入口、插件/连接器/日志的信息架构
- 控制范围：模型状态、模型启停/切换、Gateway / Tool Router 状态、模式与插件状态、会话/工具日志
- 不做：替代 Open WebUI、直接聊天主入口、prompt 编辑器、远程多用户管理

## Key Changes

### 1. 固定 GUI 架构

GUI 方案固定为三层，不再留实现分歧：

- 前端：`React + Vite + Tailwind CSS v4`
  - 浏览器优先
  - 视觉语言向 `Memori-Vault` 靠拢
  - 布局和后台信息组织吸收 `AstrBot` 的管理台思路
  - 不做 Tauri 桌面壳
- 控制面：继续复用现有 `Gateway (Go)`
  - 新增 admin API
  - 负责聚合状态、日志、模式信息、工具状态
- 宿主机控制代理：新增一个本机 `Windows Host Control Agent (Rust)`
  - 只负责 `llama-server` 的启动、停止、重启、切换配置
  - 不承载聊天、不承载工具编排

### 2. 固定 V1 GUI 页面结构

V1 页面结构固定为 5 个一级视图，并按 `AstrBot` 式后台思路组织导航：

1. 总览 Dashboard
- 显示 `llama-server / Gateway / Tool Router / Open WebUI` 状态
- 显示当前激活模型、上下文、监听地址、最近错误
- 显示最近 10 条会话摘要和最近 10 条工具摘要
- 顶部保留系统总状态条，强调本机系统是否可用

2. 模型控制 Models
- 显示当前活动模型 profile
- 支持启动、停止、重启 `llama-server`
- 支持在预定义 profile 间切换
- 支持查看模型文件路径、量化档位、ctx、端口、状态
- 右侧详情抽屉显示 profile 配置与最近一次启动结果

3. 模式与插件 Modes
- 显示 `awdp` 主模式
- 显示 `web / pwn` 插件状态与说明
- 显示每个模式/插件关联的 prompt、tool scopes、retrieval roots
- V1 只做查看，不做在线编辑
- 页面结构参考 `AstrBot` 的插件与配置页组织方式，不做纯静态列表

4. 日志与审计 Logs
- 查看会话日志
- 查看工具调用日志
- 查看最近失败原因
- 支持按时间、request_id、工具名过滤
- 支持查看单次调用详情，右侧抽屉显示上下游状态与失败摘要

5. 系统设置 Settings
- 显示 Gateway 上游地址、Tool Router 地址、Host Agent 地址
- 显示当前模型 profile 源配置
- 显示连接器/控制代理状态摘要
- V1 只读展示配置摘要，不做网页内编辑

### 3. 固定视觉与布局风格

GUI 风格固定为“`Memori-Vault` 视觉语言 + `AstrBot` 布局方式”的混合方案：

- 整体为浅色、高密度信息面板
- 左侧垂直导航栏
- 顶部保留简洁状态栏与页面级操作区
- 中央主工作区为卡片式控制台
- 右侧上下文抽屉用于详情、状态、操作确认
- 页面层次按后台产品组织，不做单页堆卡片

视觉关键词：

- 本地优先
- 工程化
- 冷静、克制
- 面板化而非营销页
- 后台产品感，而非通用 admin 模板感

样式约束：

- 不做通用后台模板感
- 不使用默认紫色系
- 不做大面积渐变宣传风
- 重点用状态色、边框层次、信息密度建立风格
- 图标、间距、面板密度、侧栏导航的节奏参考 `AstrBot`
- 字体、色块控制、卡片语言、信息精度参考 `Memori-Vault`

### 4. 固定接口与控制边界

新增接口固定分两组：

Gateway Admin API（Go）：

- `GET /internal/admin/overview`
- `GET /internal/admin/services`
- `GET /internal/admin/models`
- `GET /internal/admin/modes`
- `GET /internal/admin/logs/sessions`
- `GET /internal/admin/logs/tools`
- `POST /internal/admin/models/switch`
- `POST /internal/admin/llama/start`
- `POST /internal/admin/llama/stop`
- `POST /internal/admin/llama/restart`

Windows Host Control Agent（Rust）：

- `GET /healthz`
- `GET /internal/host/llama/status`
- `POST /internal/host/llama/start`
- `POST /internal/host/llama/stop`
- `POST /internal/host/llama/restart`
- `POST /internal/host/llama/switch`

固定边界：

- 浏览器只访问 Gateway
- Gateway 不直接自己杀/起 Windows 进程
- Gateway 通过 Host Control Agent 控制 `llama-server`
- Host Control Agent 只绑定本机地址，不对外网开放
- Open WebUI 保持聊天入口身份，不承接管理职责

### 5. 固定模型管理方式

为 GUI 的模型切换新增固定配置源：

- 新增 `Agent/model-profiles.json`
- 只允许切换预定义 profile
- 不允许在 GUI 中手填任意模型路径
- 每个 profile 固定包含：
  - `id`
  - `label`
  - `model_path`
  - `quant`
  - `ctx_size`
  - `parallel`
  - `threads`
  - `n_gpu_layers`
  - `flash_attn`
  - `enabled`
- 当前活动模型状态由 Host Control Agent 持有并暴露给 Gateway

V1 预期 profile：

- `deepseek-r1-70b-q4-competition`
- `qwen3.5-35b-a3b-q8-experimental`

### 6. 固定安全与部署约束

V1 安全策略固定如下：

- 管理台只绑定 `127.0.0.1`
- 不做登录，不做多用户
- 不做局域网访问
- 不开放 prompt 在线编辑
- 不开放 registry 在线编辑
- 不开放任意命令执行
- 只允许通过预定义 profile 启停/切换模型

开发与部署固定为：

- 前端开发：Vite dev server
- 前端生产：构建后由 Gateway 静态托管到 `/admin`
- 管理 API 与聊天 API 共用 Gateway 端口
- Host Control Agent 独立运行在 Windows 本机端口，例如 `127.0.0.1:8098`

## Test Plan

### 1. 页面与风格验证

- Admin GUI 能在浏览器打开
- 视觉风格与 `Memori-Vault` 一致性明显
- 页面信息组织与 `AstrBot` 类后台一致性明显
- 左侧导航、顶部状态条、主内容区、右侧详情抽屉都可用
- 页面在桌面分辨率下正常，不做移动端优先

### 2. 状态链路验证

- Dashboard 能显示 `llama-server / Gateway / Tool Router / Open WebUI` 状态
- `llama-server` 停止后状态能实时反映
- `Gateway` 与 Host Control Agent 断开时能显示明确错误

### 3. 模型控制验证

- 点击启动后可成功拉起 `llama-server`
- 点击停止后 `llama-server` 进程消失
- 切换 profile 后能以新模型配置启动
- Open WebUI 经 Gateway 调用时能看到新模型别名/能力变化

### 4. 日志与模式验证

- 会话日志可显示最近记录
- 工具日志可显示最近调用
- Modes 页面能正确显示 `awdp / web / pwn`
- 插件信息展示不依赖手工硬编码

## Assumptions

- 新文档路径固定为 [Gateway管理GUI计划.md](D:\AI\Local\Docs\Plan\实施方案\Gateway管理GUI计划.md)
- GUI 第一版是浏览器管理台，不做桌面壳
- Open WebUI 继续保留为聊天前端
- 高性能语言的落点固定为：
  - `Gateway` 继续用 Go
  - `Windows Host Control Agent` 用 Rust
- GUI 风格参考来源固定为：
  - `Memori-Vault`：视觉语言、卡片系统、控制台气质
  - `AstrBot`：后台布局、导航结构、配置与日志页面组织
- 参考来源：
  - Memori-Vault: https://github.com/FPSZ/Memori-Vault
  - AstrBot WebUI docs: https://docs.astrbot.app/en/use/webui.html
