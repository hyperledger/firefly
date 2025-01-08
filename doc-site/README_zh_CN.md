# Hyperledger FireFly 文档站

此目录基于 [Hyperledger 文档模板](https://github.com/hyperledger-labs/documentation-template)。 该文档使用 MkDocs ( 文档在 [mkdocs.org](https://www.mkdocs.org) ) 和 Material 的主题 (文档在[Material for MkDocs](https://squidfunk.github.io/mkdocs-material/))。 Material 主题给 MkDocs 添加了一系列的主题，让 Hyperledger 的代码库具有采取这些主题 [Insiders](https://squidfunk.github.io/mkdocs-material/insiders/) 的能力.

[Material for MkDocs]: https://squidfunk.github.io/mkdocs-material/
[Mike]: https://github.com/jimporter/mike

## 前置条件

需要以下工具去测试这些文档和升级已发布的站点:

- A Bash shell
- git
- Python 3
- [Material for Mkdocs] 主题.
- 用于发布到 gh-pages 的 [Mike] MkDocs 插件。
  - 非本地使用，但是需要在 `mkdocs.yml` 文件引用到， 并且在将站点部署到 gh-pages 的时候使用到。

### git

`git` 可以本地安装，你可以在 [Install Git Guide from GitHub](https://github.com/git-guides/install-git) 找到安装指引。

### Python 3

`Python 3` 可以本地安装，你可以在 [Python Getting Started guide](https://www.python.org/about/gettingstarted/) 找到安装指引。

### 虚拟环境

为了避免在你的本机环境上存在 python 安装的冲突，安装一个 python 虚拟环境是值得推荐的。这也能避免你在计算机上全局安装这个项目的依赖。


```bash
cd doc-site
python3 -m venv venv
source venv/bin/activate
```

### Mkdocs

Mkdocs 相关的内容也可以在本地安装，正如 [Material for Mkdocs] 指引所说。以下是简短的、针对具体版本的安装说明：


```bash
pip install -r requirements.txt
```

### 确认安装

为了确认安装的 `mkdocs` 生效， 你可以运行  `mkdocs --help` 看看有没有生成帮助文档。

## 一些有用的 MkDocs 命令

以下是 `mkdocs` 可以使用的命令:

- `mkdocs serve` - 启动一个动态刷新的文档服务器。
- `mkdocs build` - 构建这个文档站点。
- `mkdocs -h` - 打印帮助信息和退出。

## 添加内容

以下是在这个文档站中添加内容的基本流程：

- 在 `docs` 文件夹创建一个新的 markdown 文档。
- 将新文件添加到目录中 ( `mkdocs.yml` 文件 的 `nav` 标签 )

如果你使用这个模板去生成你自己的文档， 请点击 [自定义指引](./docs/index.md).

## 代码库布局

    mkdocs.yml    # 配置文件。
    docs/
        index.md  # 文档主页。
        ...       # 其他 markdown 页面，图片和文件。
