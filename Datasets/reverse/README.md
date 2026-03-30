# Reverse Dataset Layout

训练前只先固定目录和模板，不直接回灌历史脏轨迹。

- `success_cases/`: 已解出且审核通过的 case
- `failure_cases/`: 失败样本与失败轨迹
- `review_queue/`: 待人工差异审查的样本
- `templates/`: `ReverseCaseRecord` 等模板

命名建议：

- case 目录：`<year>-<source>-<slug>`
- 主要记录：`case.json`
- teacher review：`review.json`
- 附件：原题文件、截图、日志放在对应 case 目录内
