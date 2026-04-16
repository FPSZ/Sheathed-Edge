# Local Diff Check

- 本地 `app.js` 确认存在 `POST /upload`，且 `path.join(uploadsDir, filename)` 后直接 `fs.writeFile`。
- 本地 `GET /` 确认每次从磁盘读取 `index.html`。
- 本地目录里同时存在 `pwn` ELF、`app.js`、`index.html`，与复盘中“表面大 ELF，核心在 app.js”完全一致。
- 结论：本地与公开线索一致度高，但公开资料更偏思路摘要，因此先按 medium-confidence case 管理。
