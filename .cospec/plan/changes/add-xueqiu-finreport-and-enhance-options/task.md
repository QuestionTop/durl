## 实施

- [x] 1.1 Content 接口新增 ToCSV 方法
     【目标对象】`internal/scraper/scraper.go` — `Content` 接口
     【修改目的】扩展统一格式化契约，支持 CSV 输出
     【修改方式】在 `Content` 接口中追加方法签名（新增）
     【相关依赖】所有实现 Content 接口的类型（PageContent、XueqiuContent、FinReportContent）均需同步实现
     【修改内容】
        - 在 `Content` 接口中追加 `ToCSV() (string, error)`

- [x] 1.2 formatter 新增 csv 格式分发
     【目标对象】`internal/formatter/formatter.go` — `Format()` 函数
     【修改目的】让 Format 函数能处理 "csv" 格式请求
     【修改方式】在 `Format()` 的 switch-case 中新增 `"csv"` 分支（新增）
     【相关依赖】`internal/scraper/scraper.go` 的 `Content` 接口（ToCSV 方法）
     【修改内容】
        - 在 switch 中新增 `case "csv": return content.ToCSV()`
        - 风格对齐：`"html"`/`"text"`/`"markdown"` 分支均直接 `return content.ToXxx()`，csv 同理，无需额外包裹

- [x] 1.3 PageContent 实现 ToCSV
     【目标对象】`internal/sites/generic/content.go` — `PageContent` 类型
     【修改目的】通用爬虫支持 CSV 输出，提取页面所有 HTML 表格转 CSV
     【修改方式】新增 `(p *PageContent) ToCSV() (string, error)` 方法（新增）
     【相关依赖】`github.com/PuerkitoBio/goquery`（已有依赖）；复用同文件已有的 `convertTablesInHTML` 和 `convertHTMLTableToMarkdown` 的 goquery 解析逻辑
     【修改内容】
        - 实现 `(p *PageContent) ToCSV() (string, error)`：
          - 调用 `p.extractor.Extract(p.level, p.selector)` 获取 HTML 字符串（与 `ToMarkdown()` 取内容方式一致）
          - 用 goquery 解析，查找所有 `table` 元素（复用 `convertHTMLTableToMarkdown` 中已有的 goquery 表格遍历逻辑）
          - 对每个 table：遍历 tr，提取 th/td 文本，使用标准库 `encoding/csv` 写入（自动处理含逗号/换行/引号的字段）
          - 多个表格之间用空行分隔，每个表格前加 `# Table N` 注释行（N 从 1 开始）
          - 若无表格，返回空字符串和 nil error

- [x] 2.1 parser.go 新增 ParseLast 函数
     【目标对象】`internal/sites/xueqiu/parser.go` — 文件末尾
     【修改目的】将 --last 字符串解析为 cutoff time.Time，支持 7d/1m/1y/202506/2024 格式
     【修改方式】在文件末尾新增 `ParseLast` 包级函数（新增）
     【相关依赖】标准库 `time`、`strconv`、`fmt`；`regexp` 已在文件中导入
     【修改内容】
        - 实现 `ParseLast(s string) (time.Time, error)` 函数：
          - 匹配 `(\d+)d`：N 天前 → `time.Now().AddDate(0, 0, -N)`
          - 匹配 `(\d+)m`：N 个月前 → `time.Now().AddDate(0, -N, 0)`
          - 匹配 `(\d+)y`：N 年前 → `time.Now().AddDate(-N, 0, 0)`
          - 匹配 6 位纯数字（如 `202506`）→ `time.Date(year, month, 1, 0, 0, 0, 0, time.Local)`
          - 匹配 4 位纯数字（如 `2024`）→ `time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)`
          - 数字解析失败（如 `strconv.Atoi` 出错）→ 返回对应 error
          - 其他格式 → 返回 `fmt.Errorf("invalid --last value: %q, supported: 7d, 1m, 1y, 202506, 2024", s)`
        - 注意：`strings` 已在文件中导入，`strconv` 需要新增到 import

- [x] 2.2 client.go 更新 FetchByURL/FetchDiscussions 签名
     【目标对象】`internal/sites/xueqiu/client.go` — `FetchByURL()` 和 `FetchDiscussions()` 方法
     【修改目的】将时间过滤从"天数整数"改为精确的截止时间点，支持 ParseLast 返回的任意 time.Time
     【修改方式】修改两个方法的签名及内部逻辑（修改）
     【相关依赖】无新依赖
     【修改内容】
        - `FetchDiscussions(code string, days int, ...)` → `FetchDiscussions(code string, cutoff time.Time, ...)`
          - 内部委托调用同步改为 `c.FetchByURL("https://xueqiu.com/S/"+code, cutoff, sort, maxPages)`
        - `FetchByURL(targetURL string, days int, ...)` → `FetchByURL(targetURL string, cutoff time.Time, ...)`
          - 删除原有的 `unlimitedDays := days < 0` 变量和 `cutoff := time.Now().AddDate(0, 0, -days)` 计算
          - 改为直接使用传入的 `cutoff` 参数
          - 边界处理：原 `!unlimitedDays && reachedCutoff` 判断改为 `!cutoff.IsZero() && reachedCutoff`（cutoff 零值表示无限制，对应原来 days=-1 的语义）
          - `extractVisibleItems(cutoff, seenIDs)` 调用无需修改，直接传入新 cutoff 即可

- [x] 2.3 XueqiuContent 更新 days→cutoff，新增 ToCSV
     【目标对象】`internal/sites/xueqiu/content.go` — `XueqiuContent` 结构体及其方法
     【修改目的】适配新的 cutoff 时间字段，并实现 CSV 输出
     【修改方式】修改结构体字段，更新 ToMarkdown 标题逻辑，新增 ToCSV 方法（修改 + 新增）
     【相关依赖】标准库 `encoding/csv`、`bytes`；`time` 需新增到 import
     【修改内容】
        - `XueqiuContent` 结构体：`days int` → `cutoff time.Time`
        - `NewXueqiuContent(discussions []Discussion, title string, days int)` → `NewXueqiuContent(discussions []Discussion, title string, cutoff time.Time)`，初始化字段同步更新
        - 更新 `ToMarkdown()` 标题行逻辑：
          - 若 `cutoff.IsZero()`：显示 `# {title} 讨论`（无时间限制）
          - 否则判断 cutoff 是否为整天（`cutoff.Hour() == 0 && cutoff.Minute() == 0 && cutoff.Second() == 0`）：
            - 是整天（对应 YYYYMM/YYYY 格式）：计算 `days := int(time.Since(cutoff).Hours()/24) + 1`，显示 `# {title} 近N天讨论`
            - 否则（对应 Nd/Nm/Ny 格式，带时分秒）：显示 `# {title} 截止到 {cutoff.Format("2006-01-02")} 的讨论`
          - 注意：原代码中 `days >= 0` 的判断逻辑需完整替换，不保留 `days` 字段
        - 实现 `(x *XueqiuContent) ToCSV() (string, error)`：
          - 使用 `encoding/csv` + `bytes.Buffer` 写入
          - 首行（header）：`时间,作者,内容,回复数,点赞数,链接`
          - 每条 Discussion 一行，字段顺序：`CreatedAt.Format("2006-01-02 15:04")`, `Author`, `Content`, `ReplyCount`, `LikeCount`, `URL`
          - 含逗号/换行/引号的字段由 `encoding/csv` 自动处理

- [x] 2.4 XueqiuScraper 改名并适配 --last
     【目标对象】`internal/sites/xueqiu/scraper.go` — `XueqiuScraper` 的 `Name()` 和 `Scrape()` 方法
     【修改目的】将 Name() 改为 "xueqiu.comment"，解析 --last 参数为 cutoff time.Time
     【修改方式】修改 Name() 返回值，替换 lastDays 解析逻辑（修改）
     【相关依赖】`internal/sites/xueqiu/parser.go` 的 `ParseLast`（同包，直接调用）
     【修改内容】
        - `Name()` 返回值：`"xueqiu"` → `"xueqiu.comment"`
        - 删除 `opts.Extra["last-days"]` 的读取、`lastDays int` 变量及 `strconv.Atoi` 调用
        - 新增：读取 `opts.Extra["last"]`（若不存在则默认 `"30d"`），调用 `ParseLast(lastStr)` 得到 `cutoff time.Time`；若 ParseLast 返回 error，则 `return nil, fmt.Errorf("invalid --last: %w", err)`
        - 将 `client.FetchByURL(target, lastDays, sort, maxPages)` 改为 `client.FetchByURL(target, cutoff, sort, maxPages)`
        - 将 `client.FetchDiscussions(code, lastDays, sort, maxPages)` 改为 `client.FetchDiscussions(code, cutoff, sort, maxPages)`
        - 将 `NewXueqiuContent(discussions, title, lastDays)` 改为 `NewXueqiuContent(discussions, title, cutoff)`
        - 清理 import：`strconv` 不再需要，从 import 中删除

- [x] 3.1 新建 finreport_client.go（财报抓取客户端）
     【目标对象】`internal/sites/xueqiu/finreport_client.go`（新建）
     【修改目的】实现雪球三张财报的逐页抓取与数据合并
     【修改方式】新建文件，package xueqiu（新建）
     【相关依赖】`internal/browser/browser.go`（已有）、`github.com/go-rod/rod`（已有）、`github.com/PuerkitoBio/goquery`（已有）
     【修改内容】
        - 定义报表配置切片（包级变量，非 const）：三项，分别为 `{id: "ZCFZB", name: "资产负债表"}`、`{id: "GSLRB", name: "利润表"}`、`{id: "XJLLB", name: "现金流量表"}`，使用匿名结构体切片
        - 定义 `FinReportTable` 结构体：`Type string`、`Headers []string`、`Rows []FinReportRow`
        - 定义 `FinReportRow` 结构体：`Name string`、`Values []string`
        - 定义 `FinReportClient` 结构体：持有 `browser *browser.Browser`、`page *rod.Page`（字段名风格对齐同包 `Client` 结构体）
        - 实现 `NewFinReportClient(b *browser.Browser) *FinReportClient`
        - 实现 `(c *FinReportClient) Init(timeout time.Duration) error`：
          - 调用 `c.browser.NewPage()` 创建页面，失败时返回 `fmt.Errorf("failed to create page: %w", err)`（风格对齐 client.go 的 Init）
          - 设置 UA（与 client.go 中 Init 使用相同的 UA 字符串）
          - 注入 stealth（与 client.go 中相同的 `EvalOnNewDocument` 调用）
          - 访问 `https://xueqiu.com`，等待 4 秒（与 client.go 的 Init 流程一致）
          - 验证页面可用（与 client.go 的 Init 验证逻辑一致）
        - 实现 `(c *FinReportClient) FetchAllReports(code string, timeout time.Duration) ([]FinReportTable, error)`：
          - 遍历报表配置切片，对每项调用 `c.fetchOneReport(code, def.id, def.name, timeout)`
          - 若某项出错，直接返回 error；全部成功则返回 tables 切片
        - 实现 `(c *FinReportClient) fetchOneReport(code, reportID, reportName string, timeout time.Duration) (FinReportTable, error)`：
          - 构造 URL：`https://xueqiu.com/snowman/S/{code}/detail#{reportID}` 并导航，失败返回 error
          - 等待 `.stock-info-content` 出现（最多 timeout），失败返回 error
          - 循环翻页：
            1. 调用 `c.extractTableData()` 提取当前页表格数据（headers + rows）
            2. 若是第一页（table.Headers 为空），初始化 FinReportTable：`Type = reportName`，`Headers = headers`，按 rows 构造 FinReportRow 切片
            3. 若是后续页，将新列追加：`Headers` 追加新列标题（跳过第0列，因为第0列是指标名），按行索引将新值追加到对应 FinReportRow.Values（行数不匹配时跳过多余行）
            4. 用 JS 查找"下一页"按钮：`Array.from(document.querySelectorAll('a,button')).find(el => el.textContent.trim() === '下一页' && !el.disabled && el.offsetParent !== null)`
            5. 若找到，点击并等待表格内容变化（等待首列第一个数据单元格文本变化，超时 1.5s 则继续）
            6. 若未找到"下一页"，退出循环
        - 实现 `(c *FinReportClient) extractTableData() (headers []string, rows [][]string, err error)`：
          - 用 JS 提取 `.stock-info-content table` 的所有行列数据
          - 第一行作为 headers（th/td 文本），后续行作为 rows（td 文本列表）
          - JS 执行失败或解析失败时返回 error

- [x] 3.2 新建 finreport_content.go（财报内容格式化）
     【目标对象】`internal/sites/xueqiu/finreport_content.go`（新建）
     【修改目的】将财报数据实现为 scraper.Content 接口，支持 CSV/Markdown/JSON/Text/HTML 输出
     【修改方式】新建文件，package xueqiu（新建）
     【相关依赖】`internal/scraper/scraper.go`（Content 接口）；标准库 `encoding/csv`、`encoding/json`、`bytes`、`strings`、`fmt`
     【修改内容】
        - 定义 `FinReportContent` 结构体：`stockCode string`、`stockName string`、`tables []FinReportTable`
        - 实现 `NewFinReportContent(stockCode, stockName string, tables []FinReportTable) *FinReportContent`
        - 实现 `(f *FinReportContent) ToCSV() (string, error)`：
          - 使用 `encoding/csv` + `bytes.Buffer`
          - 对每张报表（FinReportTable）：
            - 写入注释行（非 CSV 行，直接写字符串）：`# {table.Type}（{stockName} {stockCode}）\n`
            - 写入 header 行：`["指标", period1, period2, ...]`
            - 写入每个 FinReportRow：`[row.Name, val1, val2, ...]`
            - 各报表之间写入空行
          - 含特殊字符的字段由 `encoding/csv` 自动处理
        - 实现 `(f *FinReportContent) ToMarkdown() (string, error)`：
          - 对每张报表，生成 `## {table.Type}` 标题 + Markdown 表格（`| 指标 | 期间1 | 期间2 |` 格式，含分隔行 `|---|---|...|`）
          - 报表之间用空行分隔
        - 实现 `(f *FinReportContent) ToText() (string, error)`：委托 `f.ToMarkdown()`
        - 实现 `(f *FinReportContent) ToHTML() (string, error)`：调用 `f.ToMarkdown()` 后包裹在 `<pre>` 中返回（风格对齐 `XueqiuContent.ToHTML()`）
        - 实现 `(f *FinReportContent) ToJSON() ([]byte, error)`：序列化包含 `stockCode`、`stockName`、`tables` 的匿名结构体，使用 `json.Marshal`（风格对齐同包其他 ToJSON）

- [x] 3.3 新建 finreport_scraper.go（财报爬虫接口适配 + 注册）
     【目标对象】`internal/sites/xueqiu/finreport_scraper.go`（新建）
     【修改目的】将财报爬虫包装为 scraper.Scraper 接口，并通过 init() 自动注册
     【修改方式】新建文件，package xueqiu（新建）
     【相关依赖】`internal/scraper/`（scraper.Scraper 接口 + scraper.Register）、`internal/browser/`、同包 `finreport_client.go`、`finreport_content.go`、`parser.go`（ResolveStockCode）
     【修改内容】
        - 定义 `XueqiuFinReportScraper` 结构体（空结构体）
        - 实现 `(x *XueqiuFinReportScraper) Name() string`：返回 `"xueqiu.finreport"`
        - 实现 `(x *XueqiuFinReportScraper) Scrape(ctx context.Context, target string, opts scraper.Options) (scraper.Content, error)`：
          - 创建 browser（风格对齐 scraper.go：`browser.New(browser.Config{ProxyURL: opts.ProxyURL, Headless: !opts.ShowUI})`），失败返回 `fmt.Errorf("failed to create browser: %w", err)`
          - defer b.Close()
          - 创建 `NewFinReportClient(b)`，调用 `client.Init(opts.Timeout)`，失败返回 `fmt.Errorf("failed to init finreport client: %w", err)`
          - 判断 target 是否含 `"xueqiu.com"`：
            - 若是 URL：用正则 `/S/([A-Z0-9]+)/` 从 URL 中提取股票代码；若无匹配返回 error
            - 若非 URL：调用 `ResolveStockCode(target, client.page)` 解析（同包函数，直接调用），失败返回 `fmt.Errorf("failed to resolve stock code: %w", err)`
          - 调用 `client.FetchAllReports(code, opts.Timeout)`，失败返回 `fmt.Errorf("failed to fetch reports: %w", err)`
          - 返回 `NewFinReportContent(code, stockName, tables), nil`
        - 新增独立的 `init()` 函数（Go 同一 package 允许多个 init()，不修改 scraper.go 中已有的 init()）：
          - `scraper.Register(&XueqiuFinReportScraper{})`

- [x] 4.1 main.go 更新 --last 参数和 csv 格式
     【目标对象】`main.go` — 全局变量声明、`main()` flag 注册、`run()`、`validateFlags()`、`inferFormatFromExtension()`
     【修改目的】将 --last-days 升级为 --last（string），新增 csv 输出格式支持
     【修改方式】修改 flag 定义、validateFlags、inferFormatFromExtension、opts.Extra 构造、Example 注释（修改）
     【相关依赖】无新依赖；删除 `strconv` 对 `lastDays` 的使用后，若 `strconv` 仍被其他地方用到则保留，否则从 import 删除
     【修改内容】
        - 全局变量：删除 `lastDays int`，新增 `last string`
        - flag 注册：
          - 删除 `cmd.Flags().IntVar(&lastDays, "last-days", 30, ...)`
          - 新增 `rootCmd.Flags().StringVar(&last, "last", "30d", "time range: 7d, 1m, 1y, 202506, 2024")`
        - `validateFlags()` 更新：`validFormats` map 新增 `"csv": true`
        - `inferFormatFromExtension()` 更新：新增 `case ".csv": return "csv"`
        - `opts.Extra` 构造更新：
          - 删除 `"last-days": strconv.Itoa(lastDays)`
          - 新增 `"last": last`
        - Example 注释更新：将 `--last-days 180` 改为 `--last 180d`
        - 检查 `strconv` import：`strconv` 在 `opts.Extra` 中原用于 `strconv.Itoa(lastDays)` 和 `strconv.Itoa(maxPages)`；删除 lastDays 后，`strconv.Itoa(maxPages)` 仍在使用，保留 import

- [x] 4.2 验证编译通过
     【目标对象】项目根目录
     【修改目的】确保所有变更后代码可正常编译，无静态分析问题
     【修改方式】执行验证命令
     【相关依赖】所有前置任务完成
     【修改内容】
        - 执行 `go build ./...`，确认无编译错误
        - 执行 `go vet ./...`，确认无静态分析问题
        - 检查点：
          - Content 接口的所有实现类型（PageContent、XueqiuContent、FinReportContent）均已添加 ToCSV 方法
          - FetchByURL/FetchDiscussions 签名已统一从 `days int` 更新为 `cutoff time.Time`
          - xueqiu.comment 注册名已生效（registry key = "xueqiu.comment"）
          - xueqiu.finreport 已通过独立 init() 注册（registry key = "xueqiu.finreport"）
          - scraper.go 中 `strconv` import 已删除（不再使用）
