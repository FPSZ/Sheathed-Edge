# Teacher Trace

## Route judgement
- `task_family=pwn`
- `shared_domain=binary`
- 但这题更准确地说是“事务/状态机型二进制服务”，不能简单按普通菜单堆题处理。

## Phase: intake
- 先从命令集认识题目模型，而不是先盯 malloc：
  - `SET`
  - `GET`
  - `CLONE`
  - `MULTI`
  - `EXEC`
  - `ABORT`
- 字符串里还出现 `TX START / TX EXEC / TX ABORT`，说明事务层是显式概念。

## Phase: triage
- 第一目标不是“找 free 在哪”，而是恢复对象关系：
  - key/object
  - value object
  - refcount
  - transaction cache / tx object
- 公开复盘的核心洞点全都和事务生命周期有关：
  - 重复 `SET` 时旧 value 处理错误
  - refcount 被绕过
  - tx chunk 未初始化
  - tx chunk 释放后仍被活动指针继续使用

## Phase: hypothesis
- 主假设：漏洞不在单次命令，而在“同一个 key 在 MULTI 阶段内被重复操作”时触发状态不一致。
- 次假设：事务对象和普通对象之间有共享指针或缓存复用，导致 use-after-free / stale pointer。
- 这题训练上的重点是“恢复状态转移图”，不是背某个命令序列。

## Phase: verification
- 验证顺序应该是：
  - 单步命令语义
  - MULTI 进入后对象如何缓存
  - 重复 `SET` / `CLONE` 后 refcount 是否变化异常
  - `ABORT/EXEC` 后 tx 对象是否正确释放并清理指针
- 如果还没把事务生命周期画清楚，就不要急着堆利用。

## Phase: exploit_or_patch
- 红队目标：把状态机漏洞收束成稳定的 UAF / stale pointer，再考虑泄露与改写。
- 蓝队目标：优先修事务分支里的对象释放逻辑，不要粗暴 NOP 掉全部 free。
- 如果某个补丁会导致事务语义崩坏或 checker 大面积失败，就说明修补层级选错了。

## Evidence required
- 本地至少要能证明：
  - 命令集和事务命令真实存在
  - tx 生命周期是独立层
  - 问题发生在事务阶段，而不是普通 `SET/GET`
- 如果后续要升 teacher case，必须补函数级对象布局和关键分支说明。

## Fallback if fail
- 如果直接利用走不通，先退回去画事务状态图。
- 如果状态图画不清，就先做弱监督样本：把这题标成“transaction-state binary service”。
- AWDP 比赛里这类题最忌讳在没看懂生命周期前反复试 payload。

