# Reverse 棰樿缁冭鍒?V1锛圫kills 浼樺厛锛岃繃绋嬬洃鐫ｈ瘯鐐癸級

## Summary

杩欎唤璁″垝浠ュ綋鍓嶉€嗗悜棰樿建杩逛负璧风偣锛屽厛鍋氫竴鏉?`reverse` 璇曠偣闂幆锛屽啀澶嶅埗鍒板叾浠栭鍨嬨€傚綋鍓嶈捣濮嬫牱鏈浐瀹氫负锛?
- `Feed/Histroy/chat-export-1774864835443.json`
- `Feed/Histroy/26-3-30/chat-浣犲ソ.txt`

鏍稿績鍐崇瓥鍥哄畾濡備笅锛?
- 绗竴闃舵涓嶇洿鎺ヤ笂寰皟锛屽厛鍋?`skills + eval + 杞ㄨ抗绾犻敊`
- `reverse` 鍏堜綔涓鸿瘯鐐癸紝涓嶄竴寮€濮嬮摵婊?`awdp/web/pwn` 鍏ㄥ煙
- 璁粌閲嶇偣涓嶆槸鍙拷鏈€缁堝仛瀵癸紝鑰屾槸绾犳杩囩▼璐ㄩ噺锛氬伐鍏烽€夋嫨銆乻hell 閫傞厤銆佽矾寰勭紪鐮併€佸け璐ユ仮澶嶃€佽瘉鎹暣鐞?- 绗竴闃舵娌跨敤鐜版湁 `awdp + pwn` 鑳藉姏闈紝涓嶅厛鏂板缓鐙珛 `reverse` 鎻掍欢
- 褰撳墠杩欓亾 `[re] 灏ょ毊路鍩冨厠鏂巻闄╄` 宸茶鐪嬭繃锛屽彧鑳借繘 `train/dev`锛屼笉鑳借繘 `eval`

## Key Changes

### 1. 寤虹珛鈥滈鐩?杞ㄨ抗-绾犻敊鈥濇暟鎹棴鐜?
鍘熷鑱婂ぉ璁板綍缁х画淇濈暀鍦?`Feed/Histroy`锛屼笉鐩存帴鎷胯剰杞ㄨ抗鍋氳缁冿紱鏂板涓€灞備汉宸ユ暣鐞嗗悗鐨?`reverse case` 璇枡锛屾渶灏戝寘鍚繖浜涘瓧娈碉細

- `case_id`
- `task_meta`
- `input_files`
- `student_trace`
- `teacher_trace`
- `error_tags`
- `gold_outcome`
- `skill_delta`
- `sft_ready`

姣忛亾棰樺浐瀹氳蛋杩欐潯娴佺▼锛?
1. 璁╁綋鍓嶆ā鍨嬬嫭绔嬪仛棰樺苟瀹屾暣璁板綍宸ュ叿杞ㄨ抗
2. 璁╂洿寮?`teacher` 鍚岄鐙珛姹傝В
3. 浜哄伐鍋氬樊寮傚鏌ワ紝杈撳嚭鈥滈敊鍦ㄥ摢銆佷负浠€涔堥敊銆佺悊鎯宠繃绋嬫槸浠€涔堚€?4. 鎶婁骇鐗╂媶鎴愪笁绫伙細
   - `skill` 淇椤?   - 杩囩▼鐩戠潱鏍锋湰
   - `eval` 鍊欓€夋牱鏈?
棣栨壒閿欒鏍囩鍏堜粠杩欐鏍锋湰閲屾娊璞★紝涓嶅仛娉涘寲绌鸿瘽锛?
- `shell_mismatch`
- `path_or_encoding_failure`
- `missing_dependency_fallback`
- `powershell_syntax_error`
- `low_signal_exploration`
- `bad_error_recovery`
- `wrong_tool_priority`
- `poor_evidence_summary`

### 2. 鍏堝啓灏忚€屽彲缁勫悎鐨?reverse skills锛屼笉鍐欎竴鍧ㄥぇ prompt

绗竴闃舵浼樺厛琛?`skills`锛屽洜涓哄綋鍓嶉棶棰樻洿鍍忊€滀笉浼氱ǔ瀹氭寜娴佺▼鍋氣€濓紝涓嶆槸鈥滃畬鍏ㄦ病鏈夌煡璇嗏€濄€俙skills` 鍥哄畾鎷嗘垚灏忓潡锛?
- `rev-intake`
  - 鍏堢‘璁ら鐩枃浠躲€佸帇缂╁寘銆佽緭鍑虹洰褰曘€佺紪鐮佷笌璺緞鍙揪鎬?- `rev-shell`
  - 鏄庣‘褰撳墠鏄?`powershell / wsl-bash`锛屽悓涓€鍛戒护鎸夊涓婚€傞厤锛屼笉娣?Linux/Windows 璇硶
- `rev-triage`
  - 鍘嬬缉鍖呰В鍖呫€佹枃浠剁被鍨嬭瘑鍒€佸叆鍙ｆ枃浠跺畾浣嶃€佸熀纭€瀛楃涓蹭笌鍏冧俊鎭鏌?- `rev-static`
  - 浣曟椂鍏?`strings`锛屼綍鏃惰繘 `radare2`锛屼綍鏃跺垏鍔ㄦ€侀獙璇?- `reverse-hypothesis-loop`
  - 姣忚疆鍙彁鍑哄皯閲忓亣璁撅紝瑕佹眰宸ュ叿杈撳嚭蹇呴』鑳芥敮鎸佷笅涓€姝ュ喅绛?- `rev-report`
  - 璇佹嵁銆佺粨璁恒€佹湭璇佸疄鍋囪銆佸鐜版楠ゅ垎寮€鍐?
姣忎釜 skill 鐨勬帴鍙ｅ浐瀹氬寘鍚細

- 瑙﹀彂鎻忚堪
- 鎵€闇€杈撳叆
- 缂栧彿娴佺▼
- 杈撳嚭鏍煎紡
- 鏈€缁堟鏌ラ」
- 蹇呰鏃堕檮鑴氭湰鎴栧弬鑰冩枃浠?
### 3. 璁粌閲嶅績鏀惧埌鈥滆繃绋嬬洃鐫ｂ€濓紝涓嶆槸鍙杺鏈€缁堥瑙?
璁粌鏍锋湰涓嶈鍙繚鐣欐渶缁堢瓟妗堟垨 `writeup`锛屽繀椤讳繚鐣欏彲鐩戠潱鐨勪腑闂村喅绛栥€俈1 涓嶇洿鎺ュ洖鐏屽師濮嬭嚜鐢遍摼璺€濈淮鏂囨湰锛岃€屾槸鍥哄畾鏀规垚鍙帶鐨勭粨鏋勫寲杩囩▼鏍囩锛?
- `phase`
- `goal`
- `chosen_tool`
- `why_this_tool`
- `expected_evidence`
- `observed_result`
- `fallback_if_fail`

鐩爣鏄妸鈥滄纭繃绋嬧€濇樉寮忔暀缁欐ā鍨嬶紝灏ゅ叾閽堝杩欐鏍锋湰閲屾毚闇茬殑澶氭澶辫銆傝繘鍏?`SFT` 涔嬪墠锛屽厛绱Н `teacher` 淇鍚庣殑澶氳疆杩囩▼鏍锋湰锛岃€屼笉鏄彧鍫嗘渶缁?`flag` 鎴栨渶缁堣В閲娿€?
### 4. 鍏堝仛 reverse 璇勬祴闆嗭紝鍐嶅喅瀹氭槸鍚﹀紑璁?
绗竴闃舵鍥哄畾鍏堝仛涓€涓皬鑰岀‖鐨?`reverse pilot` 鏁版嵁闆嗭紝寤鸿 20 棰樿捣姝ワ細

- `train/curation`: 10
- `dev`: 5
- `eval`: 5

璇勬祴涓嶅彧鐪嬧€滄渶鍚庤繃娌¤繃鈥濓紝杩樿鍥哄畾璁板綍杩欎簺杩囩▼鎸囨爣锛?
- `solve_rate`
- `time_to_first_useful_action`
- `invalid_tool_call_rate`
- `shell_mismatch_rate`
- `avg_tool_calls`
- `error_recovery_turns`
- `truncated_output_incidents`
- `final_evidence_quality`

鍒ゅ畾鏄惁杩涘叆涓嬩竴闃舵鐨勯棬妲涘浐瀹氫负锛?
- `reverse dev/eval` 宸茬ǔ瀹氬彲澶嶈窇
- 鑷冲皯绉疮 30 涓汉宸ユ牎姝ｈ繃鐨?`reverse case`
- 缁忚繃 `skill` 涓庡伐鍏锋弿杩拌凯浠ｅ悗锛屼富瑕佸け璐ユā寮忓凡鏀舵暃锛屼笉鍐嶆瘡棰橀兘鎹竴绉嶉敊娉?
### 5. 寰皟鍙綔涓虹浜岄樁娈碉紝涓嶅拰绗竴闃舵娣峰仛

杈惧埌涓婇潰闂ㄦ鍚庯紝鍐嶅噯澶囧閮ㄨ缁冦€傞粯璁ゅ彧鍋?`SFT`锛屼笉鍦?V1 涓婂仛 `DPO/RL`銆?
`SFT` 鏁版嵁瑙勫垯鍥哄畾涓猴細

- 淇濈暀澶氳疆瀵硅瘽涓庡伐鍏疯皟鐢ㄧ粨鏋?- 姣忔潯鏍锋湰甯︿笂褰撳墠鏈€浼樼郴缁熸彁绀轰笌 `skill` 涓婁笅鏂?- 璁粌闆嗗拰娴嬭瘯闆嗕弗鏍煎垎绂?- 楂樿川閲忓皯閲忔牱鏈紭鍏堜簬澶ч噺浣庤川閲忔牱鏈?- 鍏堣鈥滆繃绋嬬ǔ瀹氭€с€佸伐鍏疯皟鐢ㄥ亸濂姐€侀敊璇仮澶嶁€濓紝涓嶆槸鍏堣鈥滃啓寰楀儚涓嶅儚楂樻墜鈥?
`DPO` 鍙暀浣滃悗缁彲閫夐」锛?
- 閫傚悎鍋氣€滀袱涓繃绋嬮兘鑳借В锛屼絾鍝釜鏇寸ǔ鏇寸渷宸ュ叿鈥濈殑鍋忓ソ鎺掑簭
- 涓嶄綔涓虹涓€闃舵涓昏矾绾?
## Artifacts / Interfaces

鏈鍒掓柊澧炵殑绋冲畾鎺ュ彛涓嶆槸 HTTP API锛岃€屾槸璁粌宸ヤ欢鎺ュ彛锛?
- `ReverseCaseRecord`
  - 涓€棰樼殑瀹屾暣璁粌鍗曞厓
- `TraceReviewRecord`
  - `student` 涓?`teacher` 鐨勫樊寮傚鏌ョ粨鏋?- `SkillDeltaRecord`
  - 鏌愭澶辫触搴旇浆鍖栨垚鍝潯 `skill` 淇
- `ReverseEvalRecord`
  - 棰樼洰銆侀绠椼€佺粨鏋溿€佸伐鍏锋寚鏍囥€佷汉宸ヨ瘎鍒?
寤鸿鍚庣画钀界洏浣嶇疆鍥哄畾涓猴細

- 鍘熷杞ㄨ抗锛歚Feed/Histroy`
- 娓呮礂鏍锋湰锛歚Datasets/reverse`
- 璇勬祴闆嗭細`Eval/reverse`
- `skill` 鏂囨。锛氭部鐜版湁 `Core/awdp/skills` 涓?`Plugins/pwn/skills` 鍒嗗眰鏀剧疆

## Test Plan

- 鐢ㄥ綋鍓嶈繖娆℃牱鏈厛鍋?1 涓畬鏁寸ず鑼?`case`锛岀‘璁や粠鍘熷杞ㄨ抗鍒?`ReverseCaseRecord` 鐨勬暣鐞嗚鍒欏彲鎵ц
- 鍐嶉€?5 閬撴湰鍦?`reverse` 棰橈紝楠岃瘉鍚屼竴濂楅敊璇爣绛炬槸鍚﹀鐢紝蹇呰鏃跺彧澧炶ˉ灏戦噺鏍囩
- `skill` 鍒濈増瀹屾垚鍚庯紝閲嶈窇鍚屼竴鎵?`dev` 棰橈紝纭杩欎簺鎸囨爣纭疄鏀瑰杽锛?  - 閿?`shell` 鍛戒护鏄捐憲涓嬮檷
  - 璺緞/缂栫爜鐩稿叧澶辫触鏄捐憲涓嬮檷
  - 鍑洪敊鍚庤兘鍦?1-2 杞唴鍒囧埌鍚堢悊 `fallback`
  - 宸ュ叿璋冪敤鏇村皯浣嗘洿鏈夋晥
- 鑻?`skill` 杩唬宸叉槑鏄炬彁楂?`solve_rate` 鍜岃繃绋嬫寚鏍囷紝鍒欑户缁墿鏍凤紱鑻ヤ粛鏃犳槑鏄炬敼鍠勶紝鍐嶈繘鍏?`SFT` 鍑嗗

## Assumptions

- 杩欎唤鏂囨。鍏堜綔涓?`Docs/Plan` 涓嬬殑鏂拌缁冭鍒?- 褰撳墠鏍锋湰涓昏鏆撮湶鐨勬槸娴佺▼鎬х己闄凤紝涓嶆槸鍗曠函鐭ヨ瘑缂哄彛锛屽洜姝ら粯璁?`skills-first`
- 璇曠偣鑼冨洿鍥哄畾涓?`reverse`锛屽悗缁啀澶嶅埗鍒?`web/pwn`
- `reverse` 鍏堟寕鍦ㄧ幇鏈?`pwn` 鑳藉姏灞傦紝涓嶅湪绗竴闃舵鏂板紑鎻掍欢
- 璁粌涓庤瘎娴嬮兘瑕佹敮鎸佺绾跨幆澧冿紱姣旇禌鎬佷笉鑳戒緷璧栬仈缃戞绱?- 璇勬祴闆嗙粷涓嶅洖鐏岃缁?
## 鍙傝€冧緷鎹?
- OpenAI Skills锛氶€傚悎鈥滃彲閲嶅銆佸姝ラ銆侀渶瑕佸浐瀹氭牸寮忓拰妫€鏌ラ」鈥濈殑浠诲姟锛屼笖鎺ㄨ崘鎷嗘垚灏忕殑鍙粍鍚堝伐浣滄祦  
  <https://academy.openai.com/public/resources/skills>
- Anthropic Agent Skills锛氬缓璁€滃厛璇勬祴鍐嶈ˉ skill鈥濓紝骞舵牴鎹湡瀹炲け璐ヨ建杩瑰閲忚凯浠? 
  <https://claude.com/blog/equipping-agents-for-the-real-world-with-agent-skills>
- Anthropic Tools for Agents锛氬伐鍏锋敼杩涘簲璧拌瘎娴嬮┍鍔紝閲嶇偣鐪嬪伐鍏烽敊璇€佸啑浣欒皟鐢ㄣ€佹弿杩板拰杩斿洖鍊兼槸鍚﹂珮淇″彿  
  <https://www.anthropic.com/engineering/writing-tools-for-agents>
- OpenAI Evaluation Best Practices锛氬厛鍋氳创杩戠湡瀹炲垎甯冪殑 held-out eval锛屽苟鎸佺画浠庢棩蹇楁寲杈圭晫 case  
  <https://platform.openai.com/docs/guides/evaluation-best-practices>
- OpenAI Fine-tuning Best Practices锛氶珮璐ㄩ噺鏁版嵁浼樺厛锛岃缁?娴嬭瘯瑕佸厛鍒嗗紑锛屽苟鎶婃渶浼樻彁绀轰笂涓嬫枃淇濈暀杩涜缁冩牱鏈? 
  <https://platform.openai.com/docs/guides/fine-tuning-best-practices>
- OpenAI Process Supervision锛氬澶氭鎺ㄧ悊浠诲姟锛岀洃鐫ｈ繃绋嬮€氬父姣斿彧鐩戠潱缁撴灉鏇村彲闈? 
  <https://openai.com/research/improving-mathematical-reasoning-with-process-supervision>
- NIST CAISI Cyber Evals锛氬彲鍊熼壌鈥滀换鍔￠泦 + agent + 棰勭畻 + 宸ュ叿鎸囨爣鈥濈殑瀹夊叏璇勬祴缁勭粐鏂瑰紡  
  <https://github.com/usnistgov/caisi-cyber-evals>

