## 实施

- [x] 1.1 定义核心抽象接口
     【目标对象】`internal/scraper/scraper.go`（新建）
     【修改目的】建立整个框架的核心契约：Scraper 接口、Content 接口、Options 结构体
     【修改方式】新建文件，定义接口和结构体
     【相关依赖】无（纯接口定义，不依赖任何内部包）
     【修改内容】
        - 定义 `Scraper` 接口：`Name() string`、`Scrape(ctx, target, opts) (Content, error)`
        - 定义 `Content` 接口：`ToHTML() (string, error)`、`ToText() (string, error)`、`ToMarkdown() (string, error)`、`ToJSON() ([]byte, error)`
        - 定义 `Options` 结构体：Method/Headers/Body/WaitFor/WaitTarget/Timeout/Level/Selector/ShowUI/ProxyURL/Extra

- [x] 1.2 实现站点注册表
     【目标对象】`internal/scraper/registry.go`（新建）
     【修改目的】提供全局注册机制，实现 OCP：新增站点无需修改 main.go
     【修改方式】新建文件，实现基于 map 的注册表
     【相关依赖】`internal/scraper/scraper.go` 的 Scraper 接口
     【修改内容】
        - 定义包级私有 `registry map[string]Scraper`
        - 实现 `Register(s Scraper)`：以 `strings.ToLower(s.Name())` 为 key 注册
        - 实现 `Get(name string) (Scraper, bool)`：按名称查找

- [x] 1.3 精简 browser 包工厂函数
     【目标对象】`internal/browser/browser.go`
     【修改目的】消除4个功能重叠的工厂函数，统一为单一入口，使调用方通过 Config 显式传参
     【修改方式】修改现有文件：保留 `Browser` 结构体及其 `NewPage()`/`Close()`/`GetProxyURL()` 方法，将私有 `newBrowser()` 函数改为由 `New()` 调用，替换全部公开工厂函数
     【相关依赖】被 `internal/sites/generic/scraper.go` 和 `internal/sites/xueqiu/scraper.go` 使用
     【修改内容】
        - 新增 `Config` 结构体：`ProxyURL string`（空字符串表示不使用代理）、`Headless bool`（`true` 表示 headless，调用方需显式传 `true` 以启用 headless 模式，`false` 为有界面模式）
        - 新增单一公开工厂函数 `New(cfg Config) (*Browser, error)`：内部调用现有私有 `newBrowser(cfg.ProxyURL, cfg.Headless)`
        - 删除 `NewBrowser()`、`NewBrowserWithProxy()`、`NewBrowserHeaded()`、`NewBrowserWithOptions()` 四个公开函数
        - 保留私有 `newBrowser(proxyURL string, headless bool)` 函数不变（作为 `New` 的内部实现）

- [x] 2.1 实现 formatter 包（替代 output 包）
     【目标对象】`internal/formatter/formatter.go`（新建）
     【修改目的】将格式化分发逻辑与具体数据结构解耦，main.go 统一通过此函数输出，不再直接调用各站点的格式化方法
     【修改方式】新建文件，仅包含格式化分发函数，不迁移任何 HTML 转换工具函数
     【相关依赖】`internal/scraper/scraper.go` 的 `Content` 接口
     【修改内容】
        - 实现 `Format(content scraper.Content, format string) (string, error)` 函数
        - 按 format 分发：`"html"` 调用 `content.ToHTML()`，`"text"` 调用 `content.ToText()`，`"markdown"` 调用 `content.ToMarkdown()`，`"json"` 调用 `content.ToJSON()` 并转为 `string` 返回
        - `"json"` 分支：`ToJSON()` 返回 `[]byte`，直接转换为 `string` 后返回
        - 未知 format 返回 `fmt.Errorf("unsupported output format: %s", format)`（与现有 `output.go` 的错误信息保持一致）
        - 注意：HTML→Markdown 的 `convertTablesInHTML` 等工具函数不在此文件，它们属于 `internal/sites/generic/content.go` 的私有实现细节

- [x] 2.2 实现通用爬虫的 fetcher（迁移）
     【目标对象】`internal/sites/generic/fetcher.go`（新建）
     【修改目的】将通用页面抓取逻辑迁移至 sites/generic 包，解除旧 fetcher 包依赖
     【修改方式】将 `internal/fetcher/fetcher.go` 全部内容复制迁移，调整包名为 `generic`
     【相关依赖】`internal/browser/browser.go`（`*browser.Browser`）、`github.com/go-rod/rod`
     【修改内容】
        - 迁移 `WaitStrategy` 类型及 `WaitStrategyLoad`/`WaitStrategyElement`/`WaitStrategyTime` 三个常量
        - 迁移 `FetchResult` 结构体（`Page *rod.Page`/`Title string`/`URL string`/`LoadTime time.Duration`）
        - 迁移 `Fetcher` 结构体（`browser *browser.Browser` 字段）及 `NewFetcher(browser *browser.Browser) *Fetcher`
        - 迁移 `SetBrowser(browser *browser.Browser)` 方法（main.go 代理重试时需要替换浏览器实例）
        - 迁移 `Fetch()`、`applyWaitStrategy()`、`headersToJS()`、`stringToJS()` 方法，逻辑不变
        - 包名改为 `generic`，将 `import "durl/internal/browser"` 保持不变（路径不变）

- [x] 2.3 实现通用爬虫的 extractor（迁移）
     【目标对象】`internal/sites/generic/extractor.go`（新建）
     【修改目的】将内容提取逻辑迁移至 sites/generic 包
     【修改方式】将 `internal/extractor/extractor.go` 内容迁移，调整包名为 `generic`
     【相关依赖】`go-rod/rod`（rod.Page）
     【修改内容】
        - 迁移 `Extractor` 结构体及其所有方法（Extract/extractFull/extractHTML/extractBody/extractContent/extractByXPath/extractByCSS）
        - 包名改为 `generic`

- [x] 2.4 实现通用爬虫的 Content
     【目标对象】`internal/sites/generic/content.go`（新建）
     【修改目的】将通用爬虫的格式化逻辑封装为实现 scraper.Content 接口的类型，替代 output.Output
     【修改方式】新建文件，从 `internal/output/output.go` 迁移核心格式化逻辑，适配 `scraper.Content` 接口签名
     【相关依赖】`internal/scraper/scraper.go`（`Content` 接口）、`internal/sites/generic/extractor.go`（`Extractor`、`NewExtractor`）、`internal/sites/generic/fetcher.go`（`FetchResult`）、`github.com/JohannesKaufmann/html-to-markdown`、`github.com/PuerkitoBio/goquery`
     【修改内容】
        - 定义 `PageContent` 结构体：持有 `extractor *Extractor`、`result FetchResult`（含 title/url/loadTime）、`level string`、`selector string`
        - 提供 `NewPageContent(result FetchResult, level, selector string) *PageContent` 工厂函数：内部调用 `NewExtractor(result.Page)` 初始化 extractor
        - 实现 `ToHTML() (string, error)`：若 level 为 `"body"` 则改用 `"html"` 级别调用 `extractor.Extract`（与 `output.OutputHTML` 逻辑一致）；其他 level 直接调用 `extractor.Extract(level, selector)`
        - 实现 `ToText() (string, error)`：level 为 `"body"` 时直接调用 `extractor.Extract("body", "")` 返回纯文本；其他 level 先取 HTML 再用 `html-to-markdown` 转换（与 `output.OutputText` 逻辑一致）
        - 实现 `ToMarkdown() (string, error)`：迁移 `output.OutputMarkdown` 逻辑，含调用私有 `convertTablesInHTML()` 预处理表格、清理注释标记
        - 实现 `ToJSON() ([]byte, error)`：迁移 `output.OutputJSON` 逻辑，构造含 html/text/markdown/title/url/load_time 的 JSON 结构体，返回 `json.MarshalIndent` 结果
        - 迁移私有函数 `convertTablesInHTML()`、`convertHTMLTableToMarkdown()` 至本文件（包内私有，供 `ToMarkdown` 使用）

- [x] 2.5 实现通用爬虫的 Scraper
     【目标对象】`internal/sites/generic/scraper.go`（新建）
     【修改目的】将通用爬虫包装为实现 scraper.Scraper 接口的类型，供 main.go 在无 --site 时直接调用
     【修改方式】新建文件，组合 `browser.New` + `Fetcher` + `NewPageContent`
     【相关依赖】`internal/scraper/scraper.go`（`Scraper`/`Content`/`Options` 接口）、`internal/browser/browser.go`（`Config`/`New`）、`internal/sites/generic/fetcher.go`（`Fetcher`/`WaitStrategy`）、`internal/sites/generic/content.go`（`NewPageContent`）
     【修改内容】
        - 定义 `GenericScraper` 结构体：持有 `cfg browser.Config`（不预先创建 browser，每次 Scrape 时按需创建）
        - 提供 `NewGenericScraper(cfg browser.Config) *GenericScraper` 工厂函数
        - 实现 `Name() string`：返回 `"generic"`
        - 实现 `Scrape(ctx context.Context, target string, opts scraper.Options) (scraper.Content, error)`：
          - 使用 `browser.New(cfg)` 创建浏览器（`cfg.ProxyURL` 和 `cfg.Headless` 由调用方（main.go）在构造 `GenericScraper` 时传入）
          - `defer b.Close()` 确保浏览器资源释放
          - 创建 `NewFetcher(b)`，调用 `Fetch(target, opts.Method, opts.Headers, opts.Body, WaitStrategy(opts.WaitFor), opts.WaitTarget, opts.Timeout)` 获取 `FetchResult`
          - 返回 `NewPageContent(result, opts.Level, opts.Selector)`
          - 错误处理：browser 创建失败和 Fetch 失败均用 `fmt.Errorf("...: %w", err)` 包装后返回
        - 注意：代理重试逻辑不在此处，由 main.go 负责（先用无代理 Config 调用，失败后用带代理 Config 重新构造 GenericScraper 再调用）

- [x] 3.1 拆分雪球爬虫：client.go（浏览器操作）
     【目标对象】`internal/sites/xueqiu/client.go`（新建）
     【修改目的】将雪球的浏览器操作逻辑从 xueqiu.go 中分离，职责单一（仅保留页面交互，不含股票代码解析和格式化）
     【修改方式】从 `internal/xueqiu/xueqiu.go` 迁移浏览器操作相关代码，包名保持 `xueqiu`
     【相关依赖】`internal/browser/browser.go`（`*browser.Browser`/`New`）、`github.com/go-rod/rod`、`github.com/go-rod/rod/lib/proto`
     【修改内容】
        - 迁移 `Discussion` 结构体（ID/Author/Content/CreatedAt/RelativeTime/ReplyCount/LikeCount/URL 字段）
        - 迁移 `Client` 结构体（`browser *browser.Browser`、`page *rod.Page` 字段）
        - 迁移 `NewClient(b *browser.Browser) *Client`
        - 迁移 `Close()` 方法（关闭 page）
        - 迁移 `Init(timeout time.Duration) error`（UA 设置、webdriver 属性覆盖、访问 xueqiu.com 主页、等待页面就绪）
        - 迁移 `FetchDiscussions(code string, days int, sort string, maxPages int) ([]Discussion, error)`（构造 URL 后调用 FetchByURL）
        - 迁移 `FetchByURL(targetURL string, days int, sort string, maxPages int) ([]Discussion, error)`（分页抓取主流程，含排序切换、登录弹窗关闭、cutoff 截止逻辑）
        - 迁移私有方法 `extractVisibleItems(cutoff time.Time, seenIDs map[string]bool) ([]Discussion, bool, error)`（DOM 提取，调用 `parseRelativeTime` 和 `cleanStatText`——这两个函数在 parser.go 中定义）
        - 迁移私有方法 `advancePage() (string, error)`（优先点击分页按钮，无效则降级滚动）
        - 迁移私有方法 `firstVisibleItemID() string`、`countVisibleItems() int`（DOM 辅助查询）
        - 迁移 `waitForPageChange(firstIDBefore string, timeout time.Duration) bool`
        - 迁移 `waitForMoreItems(countBefore int, timeout time.Duration) bool`
        - 注意：`ResolveStockCode`、`searchStockByPage` 不在此文件，它们属于 parser.go

- [x] 3.2 拆分雪球爬虫：parser.go（纯函数 + 页面搜索）
     【目标对象】`internal/sites/xueqiu/parser.go`（新建）
     【修改目的】将股票代码解析、相对时间解析等可独立测试的函数从 xueqiu.go 中分离
     【修改方式】从 `internal/xueqiu/xueqiu.go` 迁移相关函数，并调整函数签名以脱离 Client 结构体
     【相关依赖】`github.com/go-rod/rod`（`*rod.Page`，供 `searchStockByPage` 使用）
     【修改内容】
        - 将 `Client.ResolveStockCode(query string)` 改写为包级函数 `ResolveStockCode(query string, page *rod.Page) (code, name string, err error)`：将原方法中对 `c.searchStockByPage(query)` 的调用改为 `searchStockByPage(page, query)`，其余4条规则逻辑不变（SZ/SH/HK前缀、6位纯数字、2-5位数字、名称搜索）
        - 将 `Client.searchStockByPage(query string)` 改写为包级私有函数 `searchStockByPage(page *rod.Page, query string) (string, string, error)`：将原方法中对 `c.page` 的引用改为参数 `page`，逻辑不变
        - 迁移包级私有函数 `parseRelativeTime(relTime string) time.Time`（支持秒/分钟/小时/天/周/月/年及日期格式，逻辑不变）
        - 迁移包级私有函数 `cleanStatText(s string) string`（去除统计数字前的"·"前缀，供 client.go 的 `extractVisibleItems` 调用）
        - 注意：`FormatDiscussions` 和 `FormatDiscussionsWithTitle` 不在此文件，它们属于 content.go

- [x] 3.3 拆分雪球爬虫：content.go（格式化输出）
     【目标对象】`internal/sites/xueqiu/content.go`（新建）
     【修改目的】将雪球内容格式化封装为实现 scraper.Content 接口的类型，替代旧 `FormatDiscussionsWithTitle` 函数
     【修改方式】从 `internal/xueqiu/xueqiu.go` 迁移 `FormatDiscussionsWithTitle` 逻辑，适配 `scraper.Content` 接口
     【相关依赖】`internal/scraper/scraper.go`（`Content` 接口）、`internal/sites/xueqiu/client.go`（`Discussion` 结构体）、`encoding/json`
     【修改内容】
        - 定义 `XueqiuContent` 结构体：持有 `discussions []Discussion`、`title string`、`days int`（`days >= 0` 时标题显示"近N天讨论"，`days < 0` 时显示"讨论"）
        - 提供 `NewXueqiuContent(discussions []Discussion, title string, days int) *XueqiuContent` 工厂函数
        - 实现 `ToMarkdown() (string, error)`：迁移 `FormatDiscussionsWithTitle` 逻辑，**修复原函数中 `sb.WriteString(fmt.Sprintf("共 %d 条讨论\n\n", ...))` 被重复写入两次的 bug**（只写一次）；格式：标题行 → 共N条讨论 → `---` 分隔线 → 每条讨论（时间-作者二级标题、正文、统计数字+链接、`---`）
        - 实现 `ToText() (string, error)`：直接调用 `ToMarkdown()` 返回（Markdown 纯文本可读性足够）
        - 实现 `ToHTML() (string, error)`：将 `ToMarkdown()` 结果包装在 `<pre>` 标签中返回（`"<pre>" + md + "</pre>"`）
        - 实现 `ToJSON() ([]byte, error)`：将 `discussions []Discussion` 直接用 `json.Marshal` 序列化为 JSON 数组返回

- [x] 3.4 拆分雪球爬虫：scraper.go（接口适配 + 注册）
     【目标对象】`internal/sites/xueqiu/scraper.go`（新建）
     【修改目的】将雪球爬虫包装为实现 scraper.Scraper 接口的类型，并通过 init() 自动注册，使 main.go 无需感知 xueqiu 实现细节
     【修改方式】新建文件，实现 `Scraper` 接口并在 `init()` 中注册
     【相关依赖】`internal/scraper/scraper.go`（`Scraper`/`Content`/`Options`）、`internal/scraper/registry.go`（`Register`）、`internal/browser/browser.go`（`Config`/`New`）、`internal/sites/xueqiu/client.go`（`NewClient`/`Client`）、`internal/sites/xueqiu/parser.go`（`ResolveStockCode`）、`internal/sites/xueqiu/content.go`（`NewXueqiuContent`）
     【修改内容】
        - 定义 `XueqiuScraper` 结构体（无字段，无状态）
        - 实现 `Name() string`：返回 `"xueqiu"`
        - 实现 `Scrape(ctx context.Context, target string, opts scraper.Options) (scraper.Content, error)`：
          - 从 `opts.Extra` 读取参数：`last-days`（转 int，默认 30）、`max-pages`（转 int，默认 -1）、`sort`（默认 `"hot"`）
          - 使用 `browser.New(browser.Config{ProxyURL: opts.ProxyURL, Headless: !opts.ShowUI})` 创建浏览器，`defer b.Close()`
          - 创建 `client := NewClient(b)`，`defer client.Close()`
          - 调用 `client.Init(opts.Timeout)` 初始化雪球会话，失败则返回错误
          - 判断 target 是否为雪球 URL（`strings.Contains(target, "xueqiu.com")`）：是则直接调用 `client.FetchByURL(target, lastDays, sort, maxPages)`，title 设为 target；否则调用 `ResolveStockCode(target, client.page)` 解析股票代码，再调用 `client.FetchDiscussions(code, lastDays, sort, maxPages)`，title 由 name+code 组合
          - 注意：`client.page` 是私有字段，需在 `Client` 上暴露 `Page() *rod.Page` 方法，或将 `ResolveStockCode` 调用改为通过 `Client` 的公开方法封装（推荐在 `Client` 上新增 `ResolveStockCode(query string) (code, name string, err error)` 方法，内部调用 `parser.ResolveStockCode(query, c.page)`）
          - 返回 `NewXueqiuContent(discussions, title, lastDays)`
        - 在 `init()` 中调用 `registry.Register(&XueqiuScraper{})`

- [x] 4.1 重构 main.go（移动 + 精简）
     【目标对象】`main.go`（根目录，新建；替代 `cmd/durl/main.go`）
     【修改目的】将 main.go 移至根目录，精简为纯 CLI 胶水层，消除硬编码站点路由和硬编码代理地址
     【修改方式】基于 `cmd/durl/main.go` 重写，保留 cobra 框架和全部 flag，替换核心 run 逻辑
     【相关依赖】`internal/scraper/registry.go`（`Get`）、`internal/scraper/scraper.go`（`Options`）、`internal/sites/generic/scraper.go`（`NewGenericScraper`）、`internal/formatter/formatter.go`（`Format`）、`internal/browser/browser.go`（`Config`）、`_ "durl/internal/sites/xueqiu"`（blank import 触发注册）
     【修改内容】
        - 保留所有现有 CLI flag：`-X`/`-H`/`-d`/`-f`/`-o`/`-w`/`-T`/`-t`/`-l`/`-s`/`--site`/`--last-days`/`--max-pages`/`--sort`/`--showui`
        - **新增** `--proxy`（`-p`）flag：`string` 类型，默认值读取 `os.Getenv("DURL_PROXY")`
        - `run()` 函数核心逻辑重写：
          - 构造 `scraper.Options`（将所有 flag 值填入对应字段，`Extra` 中放入 `"last-days"`/`"max-pages"`/`"sort"` 的字符串值）
          - 若 `--site` 非空：调用 `registry.Get(site)` 获取 Scraper，未找到则返回错误；若 `--site` 为空：使用 `generic.NewGenericScraper(browser.Config{Headless: !showUI})`（无代理初始 Config）
          - 调用 `s.Scrape(ctx, url, opts)` 获取 `content`
          - 通用模式（无 --site）的代理重试逻辑：若第一次 Scrape 失败且 `proxyURL` 非空，用带代理的 Config 重新构造 `GenericScraper` 后重试；若 `proxyURL` 为空则直接返回错误（不重试）
          - 专项站点模式（有 --site）：不做代理重试，`opts.ProxyURL` 已通过 `Options` 传入，由各站点 Scraper 自行处理
          - 调用 `formatter.Format(content, outputFormat)` 获取输出字符串
          - 文件输出：若 `-o` 非空则写文件（`os.WriteFile`），否则 `fmt.Println`
          - 非 JSON 格式且输出到 stdout 时，向 stderr 输出 Title/URL/LoadTime 元数据（仅通用模式有此元数据，专项模式可跳过）
        - 保留辅助函数：`normalizeURL`、`inferFormatFromExtension`、`parseHeaders`、`isXueqiuURL`（供 xueqiu 模式 URL 判断，或移至 xueqiu/scraper.go 后可删除）、`validateFlags`
        - 通过 `_ "durl/internal/sites/xueqiu"` blank import 触发 xueqiu 的 `init()` 注册

- [x] 4.2 删除旧文件和目录
     【目标对象】旧目录结构
     【修改目的】清理被替代的旧文件，保持代码库整洁
     【修改方式】删除文件和目录
     【相关依赖】无（在所有新文件创建完成后执行）
     【修改内容】
        - 删除 `cmd/` 目录（含 `cmd/durl/main.go`）
        - 删除 `internal/fetcher/` 目录
        - 删除 `internal/extractor/` 目录
        - 删除 `internal/output/` 目录
        - 删除 `internal/xueqiu/` 目录

- [x] 4.3 更新 .vscode/launch.json
     【目标对象】`.vscode/launch.json`
     【修改目的】更新调试配置，指向根目录的 main.go
     【修改方式】修改 program 路径
     【相关依赖】无
     【修改内容】
        - 将 `"program": "${workspaceFolder}/cmd/durl/main.go"` 改为 `"program": "${workspaceFolder}/main.go"`

- [x] 4.4 验证编译通过
     【目标对象】项目根目录（`go.mod` 所在目录）
     【修改目的】确保重构后所有包均可正常编译，无残留的旧包引用
     【修改方式】执行验证命令（不修改代码文件）
     【相关依赖】所有前置任务（1.1～4.3）全部完成后执行
     【修改内容】
        - 运行 `go build ./...`：确认无编译错误；若有错误，定位到具体文件和行号后修复对应任务中的遗漏
        - 运行 `go vet ./...`：确认无静态分析问题（重点检查接口实现完整性）
        - 检查点：确认旧包路径（`durl/internal/fetcher`、`durl/internal/extractor`、`durl/internal/output`、`durl/internal/xueqiu`、`durl/cmd/durl`）已无任何文件引用
