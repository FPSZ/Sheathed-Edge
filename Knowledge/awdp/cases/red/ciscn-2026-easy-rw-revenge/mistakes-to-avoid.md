# Mistakes To Avoid

- 不要一看到堆题就直接套 one_gadget 模板
- 不要在没证明 UAF 稳定前就进入 largebin/tcache 细节
- 不要忽略服务型约束；AWDP 场景下“能复用、能稳定读 flag”比最花哨链条更重要
- 不要粗暴把所有 free NOP 掉，这在蓝队修补里很容易导致 checker 或协议崩坏
- 不要把“公开复盘的具体 exp”当成唯一思路，训练目标是原语与阶段，不是背脚本

