## Golang Uw/Utilware 离线开发工具包 (WIP)

算是 Utilware 的替代...因为 Utilware 里面由于都是社区包拼凑...目前已经乱成了大杂烩写起代码来感觉哪哪都不方便, 就干脆自己慢慢写组件得了...
除了一些关键的包引用会内置到 pkg 并且署名外, 其他的包都是直接引用的, 不会再像 Utilware 一样拼凑一个 100 MB 的 `sqlite` 包进来了...

#### 使用方式:

推荐拉到本地然后随便修改...因为这个包是我自己用的, 所以我会随便改...如果你不想随便改, 那么可以使用伪版本的方式引用

```bash

# 1.通常方式
# 这种方式后续更新只需要 git pull 就行了

git clone https://github.com/ClarkQAQ/uw
go mod edit replace uw => /xxx/uw

# 2.使用伪版本

replace uw => github.com/ClarkQAQ/uw v0.0.0-[时间]-[commit]

```

