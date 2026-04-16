# Teacher Trace

## Route judgement
- `task_family=pwn`
- `shared_domain=binary`
- `primary_skill=binary-core -> mitigation-and-primitive-judgement -> exploit-path-selection`

## Phase: intake
- 先把它识别成“服务型二进制”，不是本地一次性交互题。
- 从字符串与公开线索先确认几个事实：
  - 有 `MD5`
  - 有 `add/delete/edit/show`
  - 有 `RTSP server listening`
  - 有 `seccomp_rule_add execve`
- 这意味着：
  - 交互是网络服务风格
  - shellcode/execve 受限
  - 攻击思路更可能是 ORW / 栈劫持 / ROP

## Phase: triage
- 先恢复菜单语义和参数约束，再看漏洞原语。
- 公开复盘指出的关键点是：
  - 鉴权处 `strcmp(MD5_raw, digest)` 有绕过空间
  - `add(-1)` 会走到 free 后 malloc 失败，留下 UAF
- 这里最重要的是“先确认 UAF 是否稳定”，不要一上来套 largebin 模板。

## Phase: hypothesis
- 主假设：服务端菜单对象存在可重用释放对象，后续 edit/show 可继续命中已释放对象。
- 次假设：由于 seccomp 禁了 execve，最终目标更偏向读 flag 而不是直接 system('/bin/sh')。
- 从公开复盘看，这题的中后期链路是：
  - UAF
  - largebin attack
  - tcache poisoning
  - 任意分配到栈 / 返回地址控制

## Phase: verification
- 先验证三件事：
  - 鉴权能否稳定过
  - UAF 是否由 `add(-1)` 或同类异常参数触发
  - show/edit 是否能观测或改写已释放对象
- 只有 leak 和写原语都稳定后，再进入 largebin/tcache 组合利用。
- 如果当前目标是 AWDP 红队实战，不必第一时间追求“最优雅 exp”，而要优先追求“最快可复用链路”。

## Phase: exploit_or_patch
- 红队输出目标：
  - 一条可重复利用的服务型堆利用链
  - 尽量偏 ORW / 读 flag，而非依赖 execve
- 蓝队输出目标：
  - 明确危险 `call free` 所在点
  - 评估是否可改为已有安全 delete 路径，减少行为改动
  - 保持服务协议和 checker 兼容

## Evidence required
- 本地至少要能拿出：
  - `MD5` / `RTSP server listening` / `seccomp` / `add delete edit show` 字符串证据
  - 一处异常参数到 UAF 的证据
  - 一处读 / 写原语证据

## Fallback if fail
- 如果 largebin 链总是断，先退回去确认：
  - UAF 是否真稳定
  - 是否已有更短的 tcache/unsorted 可打链
  - 服务读取 flag 是否必须走 ORW
- 如果红队链路太长，AWDP 场景下优先转去补 blue patch case，别在单题上过度沉没成本。

