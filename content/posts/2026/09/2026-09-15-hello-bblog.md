---
title: 用 Go + React 从零搭一个博客
date: 2026-09-15 21:30:00
tags: [go, react, 建站]
summary: 记录这个博客的架构：Markdown 文件是唯一真相，Go 负责服务端渲染，React 负责交互，内容通过 rsync 发布。
---

## 为什么自己写

现成的静态站点生成器都试过一圈，最后想要的是：**文章就是磁盘上的 Markdown 文件**，发布只要一条命令，同时正文由服务端直出，搜索引擎和微信抓取都能拿到内容。

## 架构

| 层 | 选型 | 职责 |
|---|---|---|
| 内容 | Markdown / HTML 文件 | 唯一真相 |
| 渲染 | Go + goldmark + chroma | 服务端直出 HTML |
| 交互 | React + TypeScript | 列表、标签、归档、搜索 |
| 部署 | 单容器 + 宿主 nginx | `/blog/` 反代到容器 8090 |

## 服务端渲染的代码路径

```go
doc := md.Parser().Parse(text.NewReader(src), parser.WithContext(ctx))
var buf bytes.Buffer
if err := md.Renderer().Render(&buf, src, doc); err != nil {
    return Result{}, err
}
```

## 行内公式与块级公式

行内公式写成 $E = mc^2$，块级公式单独成段：

$$
\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}
$$

## 一段原始 HTML

下面这段是 Markdown 里内嵌的 HTML，渲染时原样输出：

<div class="callout">
  <strong>提示：</strong>原始 HTML 不做转义，因为写入通道只有 SSH 与 rsync。
</div>

## 小结

内容与代码分离，发布与部署分离，是这套方案最舒服的地方。
