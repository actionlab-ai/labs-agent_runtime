# 历史归档：v0.2

本文件是历史归档，只记录当时从最小可运行版本走向工程化配置加载时的关键改动。

它不是当前状态说明。
当前真实状态请优先看：

- [README.md](../README.md)
- [README.md](README.md)
- [IMPLEMENTATION.md](IMPLEMENTATION.md)

## v0.2 当时做了什么

1. 移除自研 `internal/yamlish`
2. 配置读取改用 `github.com/spf13/viper`
3. skill frontmatter 改用 `gopkg.in/yaml.v3`
4. 增加默认值、环境变量覆盖和配置校验

## 为什么保留这个历史文件

因为这一步是项目从“先跑起来”转向“可维护 runtime”的分水岭，后面排查老配置或理解仓库历史时还有参考价值。

## 现在怎么看它

把它当成历史背景，不要把里面的示例模型、默认配置和阶段描述当作当前事实。
