# Challenge

本地样本是带 `proxy` 的 pwn 服务，二进制字符串里能看到 `RTSP server listening`、`MD5`、`seccomp`、`add/delete/edit/show`。

训练目标：让模型先识别这是服务型堆题，再进入多阶段利用或最小修复。
