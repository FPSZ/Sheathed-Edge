# Re - calc

## 解题过程

### 1. 样本识别

样本路径 `./calc`，从导入函数 `KERNEL32.dll`、`VCRUNTIME140D.dll` 可判断为 Windows PE 程序。

```
strings ./calc | grep -i "flag\|utflag"
```

输出包含完整 flag 字符串：`utflag{str1ngs_1s_y0ur_fr13nd}`

### 2. 入口定位

程序入口为 `main` 函数，调用 `printf` 输出提示信息：

```
Enter an operator (+, -, *,):
Enter two operands:
```

未直接验证入口地址，但字符串证据已足够定位程序功能为简单计算器。

### 3. 逻辑还原

程序逻辑为：
1. 提示输入运算符
2. 提示输入两个操作数
3. 执行计算并输出结果

flag 未参与任何加密或解密逻辑，直接以明文形式硬编码在二进制文件中。

关键证据：`strings` 扫描直接返回完整 flag，无需逆向分析计算逻辑。

### 4. 脚本验证

未直接验证，flag 已通过 `strings` 工具直接提取，无需额外脚本验证。

## Flag

```
utflag{str1ngs_1s_y0ur_fr13nd}
```
