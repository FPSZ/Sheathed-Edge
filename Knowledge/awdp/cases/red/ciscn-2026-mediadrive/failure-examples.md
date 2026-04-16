# Failure Examples

## Failure A: blacklist-first dead end
- 只输出“存在 `flag|/flag|..` 黑名单，建议换点”
- 问题：没有识别到编码转换造成的语义漂移

## Failure B: payload memorization
- 只记住 `ISO-2022-CN-EXT + fl%80ag`
- 问题：模型会在换题或换字符时直接失效，没学到“转换前后对象不一致”

## Failure C: ignore safer comparison target
- 完全不看 `download.php`
- 问题：丢失“同系统中不同 sink 策略不一致”的对照训练价值

