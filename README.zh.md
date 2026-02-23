# durl

类似 curl 的动态网页抓取工具，支持 JavaScript 渲染。

## 功能特性

- **动态渲染**：通过 Playwright 抓取需要 JavaScript 执行的页面内容
- **多级内容提取**：支持多种提取层级（full、html、body、content、xpath、css）
- **多种输出格式**：支持 HTML、Text、Markdown、JSON 和 CSV
- **站点专属模式**：内置针对特定网站的爬虫（雪球评论、财务报告、必应搜索、百度搜索）
- **代理支持**：内置代理支持，失败时自动重试
- **灵活的等待策略**：支持等待页面加载、特定元素或自定义超时
- **分页支持**：站点专属爬虫自动处理分页

## 安装

### 前置条件

- Go 1.23.4 或更高版本
- Playwright 浏览器（首次使用时自动下载）

### 从源码构建

```bash
go build .
```

## 使用方法

### 基本用法

使用默认设置抓取 URL：

```bash
durl https://example.com
```

### 通过选择器提取内容

使用 CSS 选择器提取内容并保存为 Markdown 文件：

```bash
durl -l css -s ".stock-info-content" -o out.md https://xueqiu.com/snowman/S/SZ300454/detail#/GSLRB
```

### 输出格式

抓取内容并以不同格式导出：

```bash
# Markdown
durl -f markdown https://example.com

# JSON
durl -f json https://example.com

# CSV（适用于表格数据）
durl -f csv https://xueqiu.com/snowman/S/SZ300454/detail#/GCFZB
```

### 等待策略

控制内容提取的时机：

```bash
# 等待页面加载（默认）
durl -w load https://example.com

# 等待特定元素
durl -w element -T "#main-content" https://example.com

# 等待指定时间（毫秒）
durl -w time -T 3000 https://example.com
```

### 代理支持

通过代理抓取内容：

```bash
# 使用 --proxy 参数
durl -p http://127.0.0.1:7890 https://example.com

# 使用环境变量
export DURL_PROXY=http://127.0.0.1:7890
durl https://example.com
```

## 站点专属模式

### 必应搜索

搜索必应并获取结果：

```bash
durl --site bing "golang 教程"
durl --site bing "golang 教程" -f json
durl --site bing "golang 教程" -f markdown -o results.md
```

### 百度搜索

搜索百度并获取结果：

```bash
durl --site baidu "golang 教程"
durl --site baidu "golang 教程" -f json
durl --site baidu "golang 教程" -f markdown -o results.md
```

### 雪球评论

搜索并提取雪球讨论：

```bash
# 按股票名称搜索
durl --site xueqiu.comment "燕京啤酒"

# 按股票代码搜索
durl --site xueqiu.comment "SZ000729"

# 指定时间范围和排序方式
durl --site xueqiu.comment --last 180d --sort new "09988"

# 时间范围格式：7d、1m、1y、202506、2024
```

### 雪球财务报告

提取财务报告并导出为 CSV：

```bash
# 导出资产负债表
durl --site xueqiu.finreport SZ300454 -f csv -o report.csv

# 导出利润表
durl --site xueqiu.finreport -o income.csv "https://xueqiu.com/snowman/S/SZ300454/detail#/GSLRB"
```

## 命令选项

| 选项 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--method` | `-X` | HTTP 方法（GET、POST、PUT、DELETE 等） | GET |
| `--header` | `-H` | HTTP 请求头（可多次使用） | - |
| `--data` | `-d` | 请求体数据 | - |
| `--format` | `-f` | 输出格式（html、text、markdown、json、csv） | text |
| `--output` | `-o` | 输出文件路径 | - |
| `--wait-for` | `-w` | 等待策略（load、element、time） | load |
| `--wait-target` | `-T` | 等待目标（选择器或毫秒数） | - |
| `--timeout` | `-t` | 请求超时时间 | 30s |
| `--level` | `-l` | 内容层级（full、html、body、content、xpath、css） | body |
| `--selector` | `-s` | xpath 或 css 层级的选择器 | - |
| `--site` | - | 站点专属模式（如 xueqiu.comment） | - |
| `--last` | - | 时间范围（7d、1m、1y、202506、2024） | 30d |
| `--max-pages` | - | 最大分页数（-1 表示不限制） | -1 |
| `--sort` | - | 排序方式：hot 或 new | hot |
| `--showui` | - | 显示浏览器界面（禁用无头模式） | false |
| `--proxy` | `-p` | 代理 URL | $DURL_PROXY |

## 内容层级

- **full**：完整页面，包含 JS、CSS 及所有资源
- **html**：仅 HTML 文档
- **body**：页面 body 内容
- **content**：智能提取主要内容（默认）
- **xpath**：使用 XPath 选择器提取内容（需要 --selector）
- **css**：使用 CSS 选择器提取内容（需要 --selector）

## 架构

项目采用模块化架构：

```
durl/
├── main.go                 # 基于 Cobra 的 CLI 入口
├── internal/
│   ├── browser/           # 浏览器抽象层
│   ├── scraper/           # Scraper 接口与注册表
│   ├── formatter/         # 输出格式化
│   └── sites/             # 站点专属爬虫
│       ├── generic/        # 通用网页爬虫
│       ├── bing/           # 必应搜索爬虫
│       ├── baidu/          # 百度搜索爬虫
│       └── xueqiu/         # 雪球专属爬虫
│           ├── comment/   # 评论爬虫
│           └── finreport/ # 财务报告爬虫
```

### 扩展性

新增站点专属爬虫的步骤：

1. 在 `internal/sites/` 下创建新包
2. 实现 `Scraper` 接口
3. 使用 `scraper.Register()` 注册爬虫
4. 在 `main.go` 中导入该包

## 依赖

- [go-rod](https://github.com/go-rod/rod) - 兼容 Playwright 的浏览器自动化库
- [cobra](https://github.com/spf13/cobra) - 命令行界面框架
- [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown) - HTML 转 Markdown

## 许可证

详见 LICENSE 文件。

## 贡献

欢迎贡献代码！请随时提交 Pull Request。
