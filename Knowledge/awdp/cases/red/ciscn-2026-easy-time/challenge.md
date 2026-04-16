# Challenge

本地样本是 Flask 站点，含 `/login`、`/plugin/upload`、`/about` 等路由。

训练目标：让模型把『登录态 -> 上传面 -> 出网/内网访问点』串成一条链，而不是只在单点 payload 上卡住。
