# CISCN 半决赛 AWDP / ISW 公开题解线索整理（基于本地目录）

## Summary

本文件用于把本地目录 `D:\AWDP\CISCN半决赛\攻防` 与可搜索到的公开赛后资料做一轮映射。

目标不是立刻把所有题都当成“标准答案”喂训练，而是先分清三类：

- `A类：公开资料足够完整，可作为 teacher reference 候选`
- `B类：只有部分 exploit / fix / 复盘线索，可作为弱监督资料`
- `C类：目前几乎没找到足够公开资料，暂不直接入训练`

---

## 本地目录盘点

### 攻防 / PWN
- `broken_manager`
- `catchme`
- `easy_rw_revenge`
- `minidb`
- `UpNodeTrap`

### 攻防 / WEB
- `easy_time`
- `MediaDrive`

### 渗透
- `WatchAgent`

---

## 结果总览

| 本地题目 | 类型 | 公开线索状态 | 训练建议 |
|---|---|---:|---|
| `MediaDrive` | web / awdp | A | 可做 case 候选 |
| `easy_time` | web / awdp | B | 可做弱监督 case |
| `UpNodeTrap` | 更像 web + binary 包装 | B | 可做弱监督 case |
| `broken_manager` | pwn / awdp | A | 可做 case 候选 |
| `easy_rw_revenge` | pwn / awdp | A/B | 可做 case 候选 |
| `catchme` | pwn / awdp | B | 可做弱监督 case |
| `minidb` | pwn / awdp | A/B | 可做 case 候选 |
| `WatchAgent` | isw / 渗透 | B | 可做渗透弱监督 case |

---

## 分题整理

### 1. `MediaDrive`
**结论：A 类，公开资料较完整。**

已找到公开资料：
- 博客园《2025 CISCN 半决赛-WEB全解 / ISW入口》
  - 来源：`https://www.cnblogs.com/Linyuuy/p/19771049`
  - 关键线索：
    - `/preview.php?f=` 文件读取
    - 可控 `user` cookie 反序列化对象
    - `encoding` / `basePath` 影响路径转换
    - 通过特定编码转换把 `/fl%80ag` 转成 `/flag`
- idocdown 对应整理页
  - 来源：`https://idocdown.com/app/articles/blogs/detail/18104`
  - 关键线索：
    - 明确提到 `preview.php` 文件读取
    - 给出 fix 方向和利用描述

训练建议：
- 适合放入 `Knowledge/awdp/cases/red/...` 作为 web 文件读取 / 编码绕过 case
- 也适合抽一个 blue fix case，强调 checker-safe 文件读取修补

### 2. `easy_time`
**结论：B 类，有明显利用线索，但“预期解”公开不完整。**

已找到公开资料：
- 博客园《2025 CISCN 半决赛-WEB全解 / ISW入口》
  - 来源：`https://www.cnblogs.com/Linyuuy/p/19771049`
  - 关键线索：
    - 非预期利用只需三次 HTTP 请求
    - `Cookie: visited=yes; user=admin` 绕过登录
    - 构造 zip 路径穿越，把 `s.php` 写到 `/var/www/html/s.php`
    - 再通过 `/about` 处 SSRF 触发 `s.php?cmd=cat /flag`

限制：
- 该文明确写了“预期解法省略”
- 因此当前更适合做“非预期 exploit case”，不适合直接当唯一标准答案

训练建议：
- 可先作为 `web-upload-to-rce` / `path traversal zip` 弱监督样本
- 标记 `status=draft` 或 `confidence=medium`

### 3. `UpNodeTrap`
**结论：B 类，有较完整思路，但当前更像复盘摘要。**

已找到公开资料：
- 博客园《2025 CISCN 半决赛-WEB全解 / ISW入口》
  - 来源：`https://www.cnblogs.com/Linyuuy/p/19771049`
  - 关键线索：
    - 核心接口是 `POST /upload`
    - `path.join(uploadsDir, filename)` 无法阻止 `../`
    - 可任意路径写，例如覆盖 `../app.js`
    - 利用思路是“改服务入口脚本 + 让进程重启”
- 博客园《2026 CISCNx长城杯半决赛复盘》
  - 来源：`https://www.cnblogs.com/seyedog/p/19814241`
  - 关键线索：
    - 同样提到 `/upload` 路径穿越
    - 给出修复方向：限制路径，固定到受控上传目录

限制：
- 公开资料更偏利用链说明，不一定包含完整可复现实战脚本

训练建议：
- 适合做 `web-upload-to-rce` / `entrypoint overwrite` 弱监督样本
- 也适合作为 “binary 外壳迷惑、实则 web 任意路径写” 的 route 纠偏样本

### 4. `broken_manager`
**结论：A 类，攻击资料较完整。**

已找到公开资料：
- 博客园《CISCN&长城杯 半决赛awdp pwn-all-break》
  - 来源：`https://www.cnblogs.com/alexander17/p/19761028`
  - 关键线索：
    - 自定义 allocator
    - double free + UAF
    - 通过 `SIGSEGV` 崩溃信息恢复 arena 地址
    - 借助 `sigaltstack` 重入 `main`
    - 最终做 freelist poisoning 命中 altstack 返回地址

训练建议：
- 可作为 `pwn / custom allocator / sigaltstack reentry` case 候选
- 但要注意：这题偏技术细节，适合做高质量 teacher trace，不适合粗暴抽短模板

### 5. `easy_rw_revenge`
**结论：A/B 类，公开 attack 与 fix 线索都能找到。**

已找到公开资料：
- 博客园《CISCN&长城杯 半决赛awdp pwn-all-break》
  - 来源：`https://www.cnblogs.com/alexander17/p/19761028`
  - 关键线索：
    - 攻击方向与 ORW 相关
    - 文中给了 `easy_rw_revenge local exploit` 结构和 `/flag` 读取思路
- 博客园《2026 CISCNx长城杯半决赛复盘》
  - 来源：`https://www.cnblogs.com/seyedog/p/19814241`
  - 关键线索：
    - 修复思路是把 UAF 处 `call free` 换成调用现成 delete 逻辑

训练建议：
- 非常适合做红蓝双视角 case：
  - red：ORW / UAF
  - blue：最小修补 call 目标替换

### 6. `catchme`
**结论：B 类，修复线索较清晰，利用方向也有摘要。**

已找到公开资料：
- 博客园《2026 CISCNx长城杯半决赛复盘》
  - 来源：`https://www.cnblogs.com/seyedog/p/19814241`
  - 关键线索：
    - `delete` UAF
    - 修补要点：free 后清空 `heap[idx]`
    - 可通过 `eh_frame` 或直接改 free 后汇编补清零逻辑
- 博客园《[2026 CCB&ciscn] 半决赛AWDP复盘》
  - 来源：`https://www.cnblogs.com/S1nyer/p/19788479`
  - 关键线索：
    - 5 槽位 glibc 2.27 菜单堆题
    - `adopt / release / inspect / engrave / leave / purge`
    - `release` 造成 UAF
    - 攻击方向是 `house of storm`

训练建议：
- 可先做 blue 修复 case
- red 攻击 case 也能做，但当前更适合弱监督而不是直接当满分 teacher

### 7. `minidb`
**结论：A/B 类，有旧题同名公开题解，也有 2026 复盘。**

已找到公开资料：
- CN-SEC《ciscn国赛华东北分区赛WriteUp分享》
  - 来源：`https://cn-sec.com/archives/1912130.html`
  - 关键线索：
    - MiniDB 结构、对象布局、UAF/越界思路
    - 给出利用方向和 exp 轮廓
- 博客园《[2026 CCB&ciscn] 半决赛AWDP复盘》
  - 来源：`https://www.cnblogs.com/S1nyer/p/19788479`
  - 关键线索：
    - 本地 2026 版 MiniDB 支持 `SET / GET / CLONE / MULTI / EXEC / ABORT`
    - 明确指出 `setval` 附近存在 UAF
    - 修补可改 `call _free` 为未引用的 `sub_13BE`

风险：
- “minidb” 这个名字历史上可能复用过，不能只凭同名旧题就直接当答案
- 需要本地样本比对命令集、结构体和漏洞点是否一致后再完全入库

训练建议：
- 可以先做 `case candidate`
- 先本地静态比对，再决定是否升成高置信 teacher case

### 8. `WatchAgent`
**结论：B 类，渗透方向线索明确，但需要本地样本核对。**

已找到公开资料：
- 博客园《2026 CISCNx长城杯半决赛复盘》
  - 来源：`https://www.cnblogs.com/seyedog/p/19814241`
  - 关键线索：
    - 提到类似 `watchagent` 服务
    - 通过 socket/RPC 连接可达成命令执行
    - 认证 token 为 `RCE_AUTH_2026`
- 搜索摘要中还出现：
  - “通过构建 RPC 连接可以达成命令执行”
  - 与 ISW/渗透场景一致

训练建议：
- 可作为 `isw / lateral-rce / agent-service` 弱监督 case
- 在没有本地复现前，不建议把 token 和命令链直接当成标准答案模板硬喂

---

## 当前最适合先入训练 / review 的题

### 第一批（优先）
- `MediaDrive`
- `broken_manager`
- `easy_rw_revenge`
- `minidb`（先本地核一遍）

### 第二批（弱监督）
- `easy_time`
- `UpNodeTrap`
- `catchme`
- `WatchAgent`

---

## 建议的训练策略

### 不建议
- 直接把博客全文塞进 prompt
- 把“同名旧题”没核过就当标准答案
- 把非预期利用当唯一正解

### 建议
- 先把这些公开资料沉淀成 `teacher-notes`
- 本地做一遍源码/二进制比对
- 只把“题目一致 + 漏洞点一致 + 利用链一致”的题升格为高置信 case
- 对其余题标记 `confidence=medium/low`，只做弱监督和 route 指导

---

## 一句话判断

你这批 `D:\AWDP\CISCN半决赛\攻防` 题里，已经有一部分能在网上找到比较像样的赛后资料，尤其是 `MediaDrive`、`broken_manager`、`easy_rw_revenge`、`catchme`、`UpNodeTrap`、`minidb`。但在正式“训练”之前，还是应该先做一轮本地样本与公开资料的对齐核验，把 `teacher case` 和 `弱监督线索` 分开。


---

## 本轮落地（2026-04-16）

已基于上面的公开线索，先把 8 个 red 候选案例落到了本地知识库：

- `Knowledge/awdp/cases/red/ciscn-2026-mediadrive`
- `Knowledge/awdp/cases/red/ciscn-2026-easy-time`
- `Knowledge/awdp/cases/red/ciscn-2026-upnodetrap`
- `Knowledge/awdp/cases/red/ciscn-2026-broken-manager`
- `Knowledge/awdp/cases/red/ciscn-2026-easy-rw-revenge`
- `Knowledge/awdp/cases/red/ciscn-2026-catchme`
- `Knowledge/awdp/cases/red/ciscn-2026-minidb`
- `Knowledge/awdp/cases/red/ciscn-2026-watchagent`

每个候选目录当前至少包含：
- `meta.json`
- `challenge.md`
- `teacher-notes.md`
- `source-links.md`
- `local-diff-check.md`
- `wp.md`
- `artifacts/`

其中：
- `MediaDrive`、`easy_rw_revenge`、`minidb` 当前可优先视作高价值训练前候选
- `easy_time`、`UpNodeTrap`、`catchme`、`WatchAgent` 当前更适合弱监督 / 路由纠偏
- `broken_manager` 资料强，但本地还应补一次函数级核对，再升格 teacher case

后续 AI 如果继续做这批题，优先修改和补充这些位置：
- 继续补 AWDP 候选案例：`D:\AI\Local\Knowledgewdp\casesed\...`
- 继续补蓝队修复案例：`D:\AI\Local\Knowledgewdp\caseslue\...`
- 重建索引：`D:\AI\Local\Workflows	estingootstrap_awdp_knowledge_base.py`
- 如果要调整红队/蓝队行为纪律：
  - `D:\AI\Local\Corewdp\promptswdp-core.md`
  - `D:\AI\Local\Pluginswdp-red\skillswdp-red-skills.md`
  - `D:\AI\Local\Pluginswdp-blue\skillswdp-blue-skills.md`
