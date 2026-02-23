# 变更：架构重构——建立可扩展的多站点爬虫框架

## 原因

当前代码存在职责混乱、硬编码扩展点、接口缺失等问题：`main.go` 臃肿（370行）且硬编码站点路由；`xueqiu.go` 单文件467行混杂浏览器操作/DOM解析/时间解析/格式化输出；`output` 包与 `fetcher.FetchResult` 强耦合导致专项爬虫无法复用输出管道；代理地址硬编码为 `127.0.0.1:7890`。随着新站点（新浪等）的加入，现有设计将持续劣化。

## 变更内容

### 架构核心：引入统一的 Scraper + Content 接口

```
┌─────────────────────────────────────────────────────────────┐
│  main.go  （CLI 入口，cobra，参数解析，路由分发）             │
└──────────────────────┬──────────────────────────────────────┘
                       │ 依赖接口，不依赖实现
┌──────────────────────▼──────────────────────────────────────┐
│  internal/scraper/   （核心抽象层）                          │
│  ├── scraper.go      Scraper 接口 + Content 接口 + Options  │
│  └── registry.go     注册表（name → Scraper）               │
└──────┬───────────────────────────────────┬──────────────────┘
       │                                   │
┌──────▼──────────┐             ┌──────────▼──────────────────┐
│ internal/sites/ │             │ internal/formatter/          │
│ ├── generic/    │             │ └── formatter.go             │
│ │   ├── scraper.go            │   Format(Content, format)    │
│ │   ├── fetcher.go            └─────────────────────────────┘
│ │   ├── extractor.go
│ │   └── content.go
│ └── xueqiu/
│     ├── scraper.go  ← init() 注册
│     ├── client.go
│     ├── parser.go
│     └── content.go
└─────────────────┘
       │
┌──────▼──────────────────────────────────────────────────────┐
│  internal/browser/browser.go  （基础设施层，精简工厂函数）    │
└─────────────────────────────────────────────────────────────┘
```

### 具体变更列表

- **新增** `internal/scraper/scraper.go`：定义 `Scraper` 接口、`Content` 接口、`Options` 结构体
- **新增** `internal/scraper/registry.go`：基于 map 的注册表，`Register()`/`Get()`
- **新增** `internal/formatter/formatter.go`：接受 `Content` 接口的格式化分发（替代 `output` 包）
- **新增** `internal/sites/generic/`：通用爬虫（组合现有 fetcher + extractor），实现 `Scraper` + `Content` 接口
- **新增** `internal/sites/xueqiu/`：雪球爬虫拆分为4个文件，实现 `Scraper` + `Content` 接口，`init()` 自动注册
- **移动** `cmd/durl/main.go` → `main.go`（根目录），精简为纯 CLI 胶水层
- **重构** `internal/browser/browser.go`：4个工厂函数精简为单一 `New(cfg Config)` 函数
- **删除** `internal/output/`、`internal/fetcher/`、`internal/extractor/`、`internal/xueqiu/`、`cmd/` 目录
- **新增** `--proxy` flag + `DURL_PROXY` 环境变量支持（替代硬编码 `127.0.0.1:7890`）

### 设计原则说明

| 原则 | 体现 |
|------|------|
| **OCP 开闭原则** | 新增站点只需在 `sites/` 下新建包 + `blank import`，不修改 main.go |
| **SRP 单一职责** | xueqiu 拆为4文件：接口适配/浏览器操作/纯函数解析/内容格式化 |
| **DIP 依赖倒置** | main.go 和 formatter 依赖 `scraper.Content` 接口，不依赖具体实现 |
| **统一输出管道** | 通用爬虫和专项爬虫都通过 `Content` 接口支持 `-f/-o` flag |

### 接口定义（核心契约）

```go
// internal/scraper/scraper.go

// Scraper 是站点爬虫的统一接口
type Scraper interface {
    Name() string  // 对应 --site 参数值，如 "xueqiu"
    Scrape(ctx context.Context, target string, opts Options) (Content, error)
}

// Content 是爬取结果的统一格式化接口
// 各站点实现自己的内容类型，不强制统一数据模型
type Content interface {
    ToHTML() (string, error)
    ToText() (string, error)
    ToMarkdown() (string, error)
    ToJSON() ([]byte, error)
}

// Options 包含所有爬取选项
type Options struct {
    Method     string
    Headers    map[string]string
    Body       string
    WaitFor    string
    WaitTarget string
    Timeout    time.Duration
    Level      string       // full/html/body/content/xpath/css
    Selector   string
    ShowUI     bool
    ProxyURL   string       // --proxy flag 或 DURL_PROXY 环境变量
    Extra      map[string]string  // 站点专属参数（last-days/max-pages/sort 等）
}
```

### browser.Config 设计

```go
// internal/browser/browser.go

type Config struct {
    ProxyURL string  // 空字符串表示不使用代理
    Headless bool    // true = headless（默认），false = 有界面
}

func New(cfg Config) (*Browser, error)
```

## 影响

- **受影响的规范**：CLI 接口、站点扩展机制、输出格式化管道
- **受影响的代码**：
  - `cmd/durl/main.go` → 移动并重构为 `main.go`（根目录）
  - `internal/browser/browser.go`：精简工厂函数
  - `internal/fetcher/fetcher.go`：迁移至 `internal/sites/generic/fetcher.go`
  - `internal/extractor/extractor.go`：迁移至 `internal/sites/generic/extractor.go`
  - `internal/output/output.go`：重构为 `internal/formatter/formatter.go`
  - `internal/xueqiu/xueqiu.go`：拆分至 `internal/sites/xueqiu/` 下4个文件
  - `go.mod`：模块路径保持 `durl` 不变（根目录 main.go 与模块名一致）
  - `.vscode/launch.json`：更新调试路径
