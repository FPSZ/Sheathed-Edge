# Route Correction

## Wrong route
- 被目录里的大 ELF 和 Node runtime 吓住
- 把题判成纯 binary reverse / pwn
- 花大量时间在 runtime 细节，忽略外部 `app.js`

## Why it is wrong
- 公开复盘已经明确提示：关键逻辑在 `app.js`
- 本地 `POST /upload` 的路径拼接和写文件逻辑才是真漏洞入口
- 这题的训练价值就在于“binary 外壳迷惑下的 web 入口纠偏”

## Correct route
- 先识别同时存在 ELF 与外部脚本时，应该检查谁才是业务入口
- 看到 `app.js` 后先扫路由和文件写入点
- 锁定 `/upload -> path.join(uploadsDir, filename) -> fs.writeFile`
- 再决定是覆盖资源文件还是覆盖入口脚本

## Short correction phrase for model tuning
- 有外部业务脚本时，先判真正入口，不要被运行时外壳带偏。

