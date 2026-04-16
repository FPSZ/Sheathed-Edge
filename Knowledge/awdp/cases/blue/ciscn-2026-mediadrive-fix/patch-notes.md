# Patch Notes

- 最小修补点：
  - 禁止直接反序列化用户 cookie；改用服务端 session 或签名后的固定结构。
  - 对最终 `convertedPath` 做规范化与目录约束，而不是只检查 raw path。
  - `preview` 与 `download` 统一路径校验语义。
- 为什么不走大改：AWDP 场景优先保证预览/下载还能用，不建议临时重写整套偏好系统。
- 回滚点：优先只改 `preview.php` 的路径验证链和 cookie 读取链。
