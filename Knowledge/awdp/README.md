# AWDP Knowledge Base

这层知识库不直接等于 prompt，也不等于训练集。

目标只有三个：
- 把可复用的红蓝打法沉淀成可检索 playbooks
- 把做过且人工确认过的案例沉淀成 cases
- 把索引和元数据单独做成 machine-friendly indexes，避免把几十上百题全文塞进 prompt

## 目录约定

- `playbooks/`
  - 抽象打法、检查表、失败回退点
- `cases/`
  - 已做过的题、服务、攻防样例
- `indexes/`
  - 给检索、评测、训练脚本消费的 JSONL 索引

## 维护原则

- playbook 只写模式，不写某道题的硬编码答案
- case 只收老师可确认、可复现、可复盘的材料
- indexes 可重建，不手工维护为主
- prompt 只放路由和纪律，不把这里全文灌进去

## 后续 AI 主要改哪里

- 新增/改打法：`Knowledge/awdp/playbooks/...`
- 补案例：`Knowledge/awdp/cases/...`
- 重建索引：`Workflows/testing/bootstrap_awdp_knowledge_base.py`
- 改 AWDP 行为纪律：`Core/awdp/prompts/awdp-core.md`
- 改红蓝 skill：`Plugins/awdp-red/skills/awdp-red-skills.md` / `Plugins/awdp-blue/skills/awdp-blue-skills.md`
