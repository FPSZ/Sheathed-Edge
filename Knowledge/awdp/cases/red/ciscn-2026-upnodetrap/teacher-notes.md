# Teacher Notes

- Alexander17 的复盘强调：不要一开始把精力浪费在整个 Node/V8 运行时；真正漏洞点在外部脚本 `app.js`。
- 本地 `app.js` 中 `/upload` 直接 `const filePath = path.join(uploadsDir, filename); fs.writeFile(filePath, content, ...)`，没有做 `resolve + base-dir` 限制。
- 公开线索与本地代码都指向：通过 `../` 可以覆盖 `app.js` 或其他入口资源，实现任意文件写 / 入口脚本替换。
- `GET /` 每次都会重新读取磁盘上的 `index.html`，所以覆盖静态资源能立即产生效果；覆盖 `app.js` 则依赖进程重启或部署环境重载。
- 这题训练上应重点记为『binary 外壳迷惑 + web 任意路径写』，让 agent 学会先找外部脚本和真实路由。
