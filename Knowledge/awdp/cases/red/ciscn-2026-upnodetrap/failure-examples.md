# Failure Examples

## Failure A: runtime rabbit hole
- 花很长篇幅分析 Node/V8/ELF 启动路径
- 问题：没有碰到题目真正的业务漏洞

## Failure B: classify as pwn only
- 因为目录在 `PWN/` 下，就直接走纯 pwn 路线
- 问题：忽略了实际攻击面在 web upload

## Failure C: no overwrite target reasoning
- 只说“有路径穿越，可写文件”
- 问题：没有继续分析覆写哪个文件更稳、更符合比赛时效

