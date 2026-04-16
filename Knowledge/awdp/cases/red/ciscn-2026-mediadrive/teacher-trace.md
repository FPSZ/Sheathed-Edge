# Teacher Trace

## Route judgement
- `task_family=web`
- `competition_mode=awdp`
- `primary_skill=route-and-parameter-enumeration -> sink-and-filter-judgement`

## Phase: intake
- 先不要急着打 payload，先确认站点是 PHP 文件型后台。
- 关键入口文件：`preview.php`、`download.php`、`profile.php`、`lib/User.php`。
- 看到 `@unserialize($_COOKIE["user"])` 时，要立刻把它记为“用户可控对象状态”，而不是只当一个普通 cookie。

## Phase: triage
- 先找真正产生文件访问的 sink：`file_get_contents`、`readfile`、`realpath`、路径拼接。
- `preview.php` 中的关键流程：
  - `$rawPath = $user->basePath . $f`
  - 对 `$rawPath` 做黑名单检查
  - `iconv($user->encoding, "UTF-8//IGNORE", $rawPath)`
  - `file_get_contents($convertedPath)`
- 这一步要得出的结论不是“有黑名单”，而是“黑名单打在转换前，读取发生在转换后”。

## Phase: hypothesis
- 主假设：如果 `encoding` 可控，就可能构造“原始路径不过滤、转换后命中敏感路径”的绕过。
- 次假设：如果 `basePath` 可控，还可能放大攻击面，不必局限于默认 uploads 目录。
- 这题最重要的抽象不是 `%80` 本身，而是：
  - 规范化 / 编码转换前后的对象不一致
  - 安全校验没有落在最终消费值上

## Phase: verification
- 先用低风险样例验证路径读取是否真实存在，例如公开资料里的 `/preview.php?f=etc/passwd`。
- 再验证转换差异是否真的可观测：页面会显示 `Raw path` 与 `Converted`。
- 公开线索里最关键的验证样例是：
  - 可控 `user` cookie
  - `encoding = ISO-2022-CN-EXT`
  - `f = fl%80ag`
- 如果页面调试区出现“raw 不是 `/flag`，converted 变成 `/flag`”，说明假设成立。

## Phase: exploit_or_patch
- 红队利用目标：把路径检查绕过到 flag 或敏感文件读取。
- 蓝队修补要点：
  - 禁止对用户 cookie 反序列化
  - 路径检查必须落在最终转换、规范化后的路径上
  - 预览和下载统一限制在受控目录，不允许用户状态直接影响根目录

## Evidence required
- 必须能指向这些本地证据：
  - `preview.php` 中 `unserialize(cookie)`
  - `preview.php` 中黑名单位置在 `iconv` 之前
  - `file_get_contents($convertedPath)`
  - `User.php` 中 `encoding / basePath`

## Fallback if fail
- 如果编码绕过没打通，不要立刻换题；退回去重新核：
  - 编码白名单有哪些
  - 黑名单是在 raw path 还是 converted path 上
  - `download.php` 是否比 `preview.php` 更安全
- 如果 `preview` 打不通但 `download` 存在真实路径约束差异，就转做“预览链与下载链不一致”的分析样本。

