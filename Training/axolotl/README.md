# Axolotl Reverse Training Scaffold

## 鐩爣

杩欏鑴氭墜鏋剁敤浜庢妸 `Datasets/reverse` 涓凡缁忚繃浜哄伐瀹℃煡鐨勬牱鏈浆鎹㈡垚 Axolotl 鍙秷璐圭殑 `chat_template` 璁粌鏁版嵁锛屽苟鐢熸垚鍙洿鎺ヨ瘯璺戠殑閰嶇疆鏂囦欢銆?
鍥哄畾鍘熷垯锛?
- 鍙鍙栨竻娲楀悗鐨?`Datasets/reverse`
- 涓嶇洿鎺ヨ鍙?`Feed/Histroy`
- 涓嶈嚜鍔ㄤ慨鏀?`Eval/reverse`
- 涓嶈嚜鍔ㄦ妸鏈鏌ユ牱鏈姞鍏ヨ缁冮泦
- 榛樿鏀寔绂荤嚎鐜

## 鐩綍

- `configs/base/`: 鍏变韩璁粌鍩虹嚎
- `configs/tasks/`: reverse 浠诲姟绾︽潫
- `configs/profiles/`: 杩愯妗ｄ綅
- `configs/generated/`: 鐢熸垚鍚庣殑鏈€缁堥厤缃?- `datasets/`: 璁粌杞崲杈撳嚭
- `scripts/`: 鏍￠獙銆佽浆鎹€侀厤缃敓鎴愯剼鏈?- `outputs/`: Axolotl 璁粌杈撳嚭鐩綍

## 宸ヤ綔娴?
### 1. 鏍￠獙 reverse 鏍锋湰

```powershell
python Training/axolotl/scripts/validate_reverse_records.py
```

浣滅敤锛?
- 鏍￠獙 `case/review/skill-delta/eval` 鍩烘湰瀛楁
- 鎷︽埅閲嶅 `case_id`
- 鎷︽埅闈炴硶 split
- 鎷︽埅涓嶅厑璁歌繘鍏?`eval` 鐨?case 琚斁杩?`Eval/reverse/eval`

### 2. 鐢熸垚 Axolotl 璁粌闆?
```powershell
python Training/axolotl/scripts/build_reverse_sft_dataset.py ^
  --allow-reviewed ^
  --output-dir Training/axolotl/datasets/reverse_pilot
```

榛樿琛屼负锛?
- `train`: 鏉ヨ嚜 `Datasets/reverse` 涓?`audit.can_use_for_train=true` 鐨?case
- `dev`: 鏉ヨ嚜 `task_meta.visibility=train-dev-only|dev-only` 鐨?case
- `eval`: 涓嶇敓鎴愯缁?jsonl锛屽彧淇濈暀鍦?`Eval/reverse`

`--allow-reviewed` 鐢ㄤ簬绗竴闃舵鑴氭墜鏋堕獙璇侊細

- 鍏佽鎶?`reviewed but sft_ready=false` 鐨?case 杞垚璁粌鏍锋湰
- 渚夸簬鐢ㄥ綋鍓嶅凡鏈夌殑 `2026-newstar-yupi-adventure` 瀹屾垚杞崲闂幆

### 3. 鐢熸垚 Axolotl 閰嶇疆

```powershell
python Training/axolotl/scripts/render_axolotl_config.py ^
  --profile reverse-dry-run ^
  --train-file Training/axolotl/datasets/reverse_pilot/train.jsonl ^
  --val-file Training/axolotl/datasets/reverse_pilot/dev.jsonl ^
  --base-model Qwen/Qwen2.5-7B-Instruct ^
  --output-dir Training/axolotl/outputs/reverse-dry-run ^
  --save-path Training/axolotl/configs/generated/reverse-dry-run.yml
```

鏀寔鐨?profile锛?
- `reverse-dry-run`
- `reverse-small-sft`
- `reverse-formal-sft`

### 4. 杩愯 Axolotl

棰勫鐞嗭細

```powershell
python -m axolotl.cli.preprocess Training/axolotl/configs/generated/reverse-dry-run.yml
```

璁粌锛?
```powershell
accelerate launch -m axolotl.cli.train Training/axolotl/configs/generated/reverse-small-sft.yml
```

## 鏁版嵁鏍煎紡

杞崲鍚庣殑 jsonl 浣跨敤 `chat_template` 鍏煎鏍煎紡锛?
```json
{
  "messages": [
    {"role": "system", "content": "..."},
    {"role": "user", "content": "..."},
    {"role": "assistant", "content": "..."}
  ],
  "metadata": {
    "case_id": "...",
    "source_split": "train",
    "review_status": "reviewed",
    "skill_context": ["rev-shell"]
  }
}
```

## 璇勬祴鎵ц璇存槑

璇勬祴涓嶈蛋 Axolotl 鍐呯疆楠岃瘉锛岀户缁娇鐢ㄤ粨搴撶幇鏈?`Eval/reverse`锛?
1. 閫夊畾 dev 棰橀泦
2. 璁╁綋鍓嶆ā鍨嬫垨鏂版ā鍨嬭窇棰?3. 鎸?`ReverseEvalRecord` 璁板綍缁撴灉
4. 姣旇緝浠ヤ笅鎸囨爣锛?   - `solve_rate`
   - `time_to_first_useful_action`
   - `invalid_tool_call_rate`
   - `shell_mismatch_rate`
   - `error_recovery_turns`
   - `final_evidence_quality`

## 浜哄伐瀹℃煡娴佺▼

杩涘叆璁粌鍓嶏紝鏍锋湰蹇呴』婊¤冻锛?
- `case.json` 瀛樺湪
- `review.json` 瀛樺湪
- `skill-delta.json` 瀛樺湪
- `audit.can_use_for_train=true`

鎺ㄨ崘瀹℃煡椤哄簭锛?
1. 鏍稿棰樼洰鏉ユ簮鍜屽彲瑙佹€?2. 鏍稿 student/teacher 宸紓
3. 鍐欐竻 `error_tags`
4. 鍐欐竻 `skill_delta`
5. 鍐嶅喅瀹氭槸鍚︽妸 `sft_ready` 缃负 `true`

