# Failure Examples

## Failure A: gadget-first reasoning
- 还没证明 leak / write primitive，就开始列 one_gadget
- 问题：把利用结果当成分析起点

## Failure B: ignore service semantics
- 完全不提网络协议、监听端口、seccomp
- 问题：得出的 exp 与真实服务环境不匹配

## Failure C: overfit public writeup chain
- 生搬 largebin+tcache 细节
- 问题：一旦本地环境差一点，模型就不会回退到“重新确认原语”

