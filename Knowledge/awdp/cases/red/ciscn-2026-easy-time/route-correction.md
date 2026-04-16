# Route Correction

## Wrong route
- 只把这题看成单点上传题或单点 SSRF 题
- 忽略登录态、上传、出网访问之间的组合关系
- 看到“非预期”三个字就直接放弃结构化分析

## Why it is wrong
- 这题更适合训练组合链：cookie bypass + ZipSlip + SSRF
- 即使公开资料偏非预期，也依然有稳定的 route 学习价值
- 小模型最容易在这种题里只抓住一个点，丢掉整条链

## Correct route
- 先识别登录态是否真的由服务端 session 保护
- 再检查 zip 上传解压函数是否安全
- 再看远程头像/回连请求点是否真正受限
- 最后把三者串成完整攻链，而不是只停在其中一段

## Short correction phrase for model tuning
- 先拼链，再挑点；组合漏洞题不要只抓一个 sink。

