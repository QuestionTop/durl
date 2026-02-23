# 变更：新增雪球财报爬取、CSV 输出、--last 时间格式增强

## 原因

需要扩展三项能力：(1) 将雪球站点拆分为 `xueqiu.comment`（讨论）和 `xueqiu.finreport`（财报）两个独立爬虫；(2) 新增 CSV 输出格式，使财报数据可直接导出为表格文件；(3) 将 `--last-days`（仅支持整数天）升级为 `--last`，支持 `7d/1m/1y/202506/2024` 等多种时间格式，精确控制数据截止时间。

## 变更内容

### 1. `--site` 值变更与新增

| 旧值 | 新值 | 说明 |
|------|------|------|
| `xueqiu` | `xueqiu.comment` | 原有讨论爬取功能，仅改名 |
| _(无)_ | `xueqiu.finreport` | 新增：获取资产负债表/利润表/现金流量表 |

**财报爬取规格**：
- 三张报表 URL：`/snowman/S/{CODE}/detail#/ZCFZB`（资产负债表）、`#/GSLRB`（利润表）、`#/XJLLB`（现金流量表）
- 内容选择器：`.stock-info-content table`
- 翻页：用 JS 查找文本含"下一页"的按钮，点击后等待表格数据更新，循环直到无"下一页"
- 多页数据合并：按行名称（指标名）对齐，将新列（期间）追加到已有数据

**财报数据结构**：
```go
type FinReportTable struct {
    Type    string     // "资产负债表" / "利润表" / "现金流量表"
    Headers []string   // 列标题（期间，如 "2024Q3"）
    Rows    []FinReportRow
}
type FinReportRow struct {
    Name   string   // 指标名称（如"货币资金"）
    Values []string // 各期间对应值
}
```

### 2. CSV 输出格式

- `scraper.Content` 接口新增 `ToCSV() (string, error)` 方法
- `formatter.Format()` 新增 `"csv"` 格式分发
- `main.go` `-f` 有效值新增 `"csv"`，`inferFormatFromExtension` 新增 `.csv → csv`
- 各 Content 实现：
  - `FinReportContent.ToCSV()`：每张报表一个 CSV 块（以空行分隔），首列为指标名，后续列为各期间值
  - `XueqiuContent.ToCSV()`：讨论列表转 CSV，字段：时间、作者、内容、回复数、点赞数、URL
  - `PageContent.ToCSV()`：提取页面所有 HTML 表格，每个表格转为 CSV 块

### 3. `--last` 时间格式增强

| 格式 | 示例 | 含义 |
|------|------|------|
| `Nd` | `7d` | N 天前 |
| `Nm` | `1m` | N 个月前 |
| `Ny` | `1y` | N 年前 |
| `YYYYMM` | `202506` | 截止到 2025 年 6 月 1 日 |
| `YYYY` | `2024` | 截止到 2024 年 1 月 1 日 |

- `--last-days int` → `--last string`（默认值 `"30d"`）
- 新增 `ParseLast(s string) (time.Time, error)` 纯函数（位于 `parser.go`）
- `client.FetchByURL` / `FetchDiscussions` 签名：`days int` → `cutoff time.Time`
- `XueqiuContent` 的 `days int` 字段改为 `cutoff time.Time`，标题显示：若 cutoff 距今整天数，显示"近N天"；否则显示"截止到 YYYY-MM-DD"

## 影响

- **受影响的规范**：CLI 接口、Content 接口契约、雪球站点扩展
- **受影响的代码**：
  - `internal/scraper/scraper.go`：Content 接口新增 `ToCSV()`
  - `internal/formatter/formatter.go`：新增 "csv" 分发
  - `internal/sites/generic/content.go`：PageContent 新增 `ToCSV()`
  - `internal/sites/xueqiu/scraper.go`：Name() 改为 `"xueqiu.comment"`，解析 `opts.Extra["last"]` → cutoff
  - `internal/sites/xueqiu/client.go`：FetchByURL/FetchDiscussions 签名 `days int` → `cutoff time.Time`
  - `internal/sites/xueqiu/parser.go`：新增 `ParseLast(s string) (time.Time, error)`
  - `internal/sites/xueqiu/content.go`：XueqiuContent `days int` → `cutoff time.Time`，新增 `ToCSV()`
  - `internal/sites/xueqiu/finreport_client.go`（新建）：FinReportClient，FetchAllReports()
  - `internal/sites/xueqiu/finreport_content.go`（新建）：FinReportContent，实现 Content 接口
  - `internal/sites/xueqiu/finreport_scraper.go`（新建）：XueqiuFinReportScraper，init() 注册
  - `main.go`：`--last-days` → `--last`，新增 csv 格式，Extra 传 "last"
