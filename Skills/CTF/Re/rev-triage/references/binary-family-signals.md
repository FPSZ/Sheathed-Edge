# Binary Family Signals

## PE

- `.exe/.dll`
- Windows 导入表
- 常见 CRT、WinAPI 痕迹

## ELF

- Linux 可执行或 `.so`
- section / interpreter / glibc 痕迹

## .NET

- `mscoree`、托管元数据、明显托管程序集结构

## Unity

- `UnityPlayer.dll`
- `GameAssembly.dll`
- `*_Data/` 目录

## APK

- `AndroidManifest.xml`
- `classes.dex`
- `res/`

## Packed / Obfuscated

- 明显 `UPX`
- 节区异常
- imports 极少但行为像完整程序
- 入口非常短或跳转异常密集

