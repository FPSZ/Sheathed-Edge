---
name: rev-shell
description: 澶勭悊 reverse 棰樹腑鐨?PowerShell 涓?WSL bash 鍛戒护閫傞厤銆佸伐鍏风己澶?fallback 鍜屽父瑙佸熀纭€鍔ㄤ綔妯℃澘銆傜敤浜庨伩鍏?Linux/Windows 鍛戒护娣峰啓銆?z 缂哄け銆佽矾寰勮浆涔夊け璐ョ瓑闂銆?---

# Reverse Shell Adapter

## 浣曟椂瑙﹀彂

鍦ㄨ繖浜涙儏鍐佃Е鍙戯細

- 褰撳墠鐜鏄?Windows PowerShell
- 棰樼洰娴佺▼涓渶瑕佽В鍘嬨€佸垪鐩綍銆佹悳绱㈡枃鏈€佹壂瀛楃涓?- 宸ュ叿鏄惁瀛樺湪杩樹笉纭畾
- 鐢ㄦ埛鎴栨ā鍨嬪彲鑳戒細鎶?Linux 鍛戒护鐩存帴鍐欒繘 PowerShell

## 宸ヤ綔鐩爣

璁?reverse 鍩虹鍔ㄤ綔鍦ㄦ湰鍦?shell 涓ǔ瀹氬彲鎵ц锛屽苟鍦ㄥ伐鍏风己澶辨椂蹇€熷垏鍒板悎鐞?fallback銆?
## 寮哄埗瑙勫垯

1. 鍏堟槑纭綋鍓?shell锛屽啀鍐欏懡浠ゃ€?2. 涓嶆贩鍐?PowerShell 鍜?bash 璇硶銆?3. 宸ュ叿缂哄け鏃朵紭鍏堢敤绯荤粺鍐呭缓鑳藉姏 fallback銆?4. 鍏堥€夌ǔ瀹氬姩浣滐紝涓嶄负浜嗏€滄洿寮衡€濆幓寮曞叆棰濆渚濊禆銆?5. 鎵€鏈夊懡浠ら兘瑕佹湇鍔″綋鍓?challenge root锛屼笉璺戝亸鍒版暣涓粨搴撱€?
## 鍥哄畾閫傞厤鑼冨洿

绗竴鐗堣嚦灏戣鐩栵細

- 瑙ｅ帇
- 鍒楃洰褰?- 鎵归噺鏂囨湰鎼滅储
- 瀛楃涓叉壂鎻?- 鏂囦欢绫诲瀷鍒濈瓫

## 鍒嗛樁娈垫祦绋?
### 1. 纭 shell

鍏堝垽鏂綋鍓嶆槸锛?
- `powershell`
- `wsl-bash`

鑻ユ病鏈夋槑纭姹傦紝涓嶅亣璁惧彲浠ョ洿鎺ュ垏鍙︿竴绉?shell銆?
### 2. 纭宸ュ叿鏄惁瀛樺湪

瀵硅繖浜涘伐鍏峰厛鍋氬瓨鍦ㄦ€у垽鏂細

- `7z`
- `rg`
- `python`
- `file`
- `strings`

### 3. 璧扮ǔ瀹?fallback

榛樿 fallback 鎬濊矾锛?
- 瑙ｅ帇澶辫触锛氫紭鍏?PowerShell/.NET 瑙ｅ帇
- 娌℃湁 `rg`锛氶€€鍥?`Select-String`
- 娌℃湁 `strings`锛氱敤 PowerShell 瀛楄妭/鏂囨湰鏂瑰紡鍋氭湁闄愭壂鎻?- 娌℃湁 `file`锛氬厛闈犳墿灞曞悕銆佸ご閮ㄧ壒寰佸拰鐩綍缁撴瀯鍒濈瓫

鏇村鍔ㄤ綔寤鸿瑙侊細
- `references/powershell-reverse-cheatsheet.md`

## 杈撳叆瑕佹眰

鑷冲皯瑕佺煡閬擄細

- 褰撳墠 shell
- 褰撳墠 challenge root
- 鐩爣鍔ㄤ綔鏄粈涔?
## 杈撳嚭瑕佹眰

姣忔閫傞厤鑷冲皯瑕佺粰鍑猴細

- 褰撳墠 shell 鍒ゆ柇
- 閫夌敤鍛戒护
- 鑻ュけ璐ョ殑 fallback

## 绂佹浜嬮」

- 涓嶈鍦?PowerShell 涓洿鎺ヤ娇鐢?bash 绠￠亾鎬濈淮
- 涓嶈榛樿绯荤粺涓€瀹氳浜?`7z`銆乣strings`銆乣file`
- 涓嶈鍥犱负鍗曚釜鍛戒护澶辫触灏辩珛鍒绘崲涓€鏁村鐜

## 浣曟椂鍗囩骇

- 鍏ュ彛绋冲畾涓旈渶瑕侀鍨嬭瘑鍒椂锛氬崌绾у埌 `rev-triage`
- 宸茬粡涓嶅啀鏄?shell 闂锛岃€屾槸闈欐€佸垎鏋愰棶棰樻椂锛氬崌绾у埌 `rev-static`


