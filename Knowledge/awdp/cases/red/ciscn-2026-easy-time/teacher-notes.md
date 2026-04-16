# Teacher Notes

- 公开复盘明确说这题公开的是非预期链，原预期未完全展开，因此这题更适合做弱监督 case。
- 本地代码中 `is_logged_in()` 仅检查 `visited=yes` 且 `user` cookie 非空，没有服务端 session 绑定；这是登录态绕过的重要前提。
- `/plugin/upload` 最终调用 `safe_upload`，而 `safe_upload` 直接 `os.path.join(dest_dir, info.filename)` 写文件，没有像 `safe_extract_zip` 那样做 commonpath 校验，存在 ZipSlip / 路径穿越。
- `/about` 会调用 `fetch_remote_avatar_info(avatar_url)` 主动请求用户给的 URL，但代码没有调用 `_host_is_public` 进行限制，因此 SSRF 点实际可用。
- 公开资料中的思路是：伪造管理员 cookie -> 上传带路径穿越的 zip 将脚本写到 Web 根 -> 再通过 `/about` 的远程头像抓取去访问并触发 shell。
- 由于公开复盘自己写了“预期解法省略”，所以这里不应升格为唯一标准答案，只能作为高价值 exploit candidate。
