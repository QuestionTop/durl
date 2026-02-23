编写一个类似curl的工具，但是Curl只能加载静态页面，对于需要js请求和渲染的，他没有办法对动态渲染的网页做抓取，请给予playwright来实现一个类似curl的命令行工具durl，用go语言实现。


还加一个命令，保留内容级别，比如保留full，就是包含js,css都在，只保留html，body，或者content（这个是保留正文内容，需要智能识别一下），也可以输入xpath或css选择器，只保留指定内容。默认是保留content


#########

我要在当前项目中，增加一个参数-site xueqiu 表示专门对雪球上的内容做搜索，搜索雪球内容其实是搜索股票名称和代码，找到具体的股票地址，比如 搜索 燕京啤酒 需要映射到 https://xueqiu.com/S/SZ000729 再获取 .stock-timeline 区域的数据； 比如 搜索 09988 其实相当于搜索阿里巴巴，需要映射到 https://xueqiu.com/S/09988 页面再获取用户数据，请深入分析雪球网站的内容布局，输入任意公司名称或者股票代码，找到具体的讨论页面，抓取最近1年的讨论内容，输出


可能是直接打开页面，然后访问类似URL获得内容 https://xueqiu.com/query/v1/symbol/search/status.json?count=10&comment=0&symbol=SZ000729&hl=0&source=all&sort=time&page=1&q=&type=11&md5__1038=222029ad07-cFGItKAcIW_b2tFtiQ_IgFg%3DPTMYgGWTXgrgFVssAkygl_eq56GoxyTkZuQqS7xLABjrgtug0PGtgkPrZgbdgTGgSPGGiPrFP%3DlPABg%3DgGhIgsdgTiGPT2_xSvtlogp%2FfHgGPrlgpPrBUo%2Fgzog_JgrXLggWgYqKOwPA0lr_BjGbklASW2JgquTx%2FWTWATgvuPqg



$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
$session.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"
$session.Cookies.Add((New-Object System.Net.Cookie("acw_tc", "2760774017718243957424833e32675014bbd6feb4fdef36436d51fc14b9f1", "/", "xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("xq_a_token", "d51dff9ce4c54877fd40470706d55c2fe08b4640", "/", ".xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("xqat", "d51dff9ce4c54877fd40470706d55c2fe08b4640", "/", ".xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("xq_r_token", "631cac5ea028b4b3ac3a2dcb79aa072b1dcc81cf", "/", ".xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("xq_id_token", "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.eyJ1aWQiOi0xLCJpc3MiOiJ1YyIsImV4cCI6MTc3MzI3ODM5MywiY3RtIjoxNzcxODI0MzQ5Njc2LCJjaWQiOiJkOWQwbjRBWnVwIn0.TWrY6mqa7dpMHH0DWGS-29AzDAI_BM2Z5YnU6N-uze6XZKXLYz1PZUMNsaB4F1YAjqE6Q7wx8s1CF8KiRQ3f17ZRPTmqaf4ce2nsrS_Sus0ttzxLHkLVkAKifyv-3fdEbfGVWPe1i0ktXgAsliHkQSYlvIupPOz3nCAq94dHMsPH7YDyL26qB1uYn7Bdbjv-SuJLN17g6hR3Fs6al6fC5GQHsJ-Epav0LCavOIJdvztvig28NjJzXG6rAFuvl9C_zT6DY3SKrmAVbrRzaZDuI_Q7A-Wp2QeU2q4pnM6rKwmLfVtg8BgrbcGZEXTdkUpqFdP8mscZwE5e7NaK8IKwJQ", "/", ".xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("cookiesu", "311771824396540", "/", ".xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("u", "311771824396540", "/", ".xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("is_overseas", "0", "/", ".xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("device_id", "5d6392876a6d5a8b02f20c5c448657df", "/", ".xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("smidV2", "2026022313263837f3ff023d8c9b1faa95bc8f4374eb4800c8a55f86aea1ff0", "/", "xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("Hm_lvt_1db88642e346389874251b5a1eded6e3", "1771824398", "/", ".xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("Hm_lpvt_1db88642e346389874251b5a1eded6e3", "1771824398", "/", ".xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("HMACCOUNT", "4A542470842A3054", "/", ".xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie(".thumbcache_f24b8bbe5a5934237bbc0eda20c1b6e7", "zY3EajkSIw74LUSeL/H3EMXljCMOHol83rC2rHdtdDFB6WIXpIvcQ2eJu6w+nlmOd04cp+Fefg9Rs4P+FH13NQ%3D%3D", "/", "xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("ssxmod_itna", "1-Gu0QY5AKGKBILx4qeuxQKx7K0KG7vqDRDl4BtGRSDIde7=GFDmx0p_gE=REIIEhBPd_xDIPqvQ8R0mDBkb_rDnqD8XDQeDvKZ82GpB8QQ2uotwZDgu4k7z7uivoFujihxQhqHZ1y2_NVD_0qXr_OeDU4GnD064QCGrDYYjDBYD74G_DDeDixGmFeDStxD9DGP=x1WbgeDEDYpWxiUea2c7xDLTnmbOwDDBI3rdWEuhDDXhWzAq_Chq_PD_bW9TjYn4_z_oneDMWxGXbDlhnly05sYWwtM6spo3xB6BxBQbyPWdIETadZcrCe1Qnb_xmbGT3GNtGrFu4Khiz9Kt0ejqbbDrbG7zTqbRY/_55xD84mdOodD5wizcMzE8rSkeiSPpoNq8oPAx2B=o7qqBx1C5ZB5xoKMADioxG7qtYKHQoelD4D", "/", ".xueqiu.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("ssxmod_itna2", "1-Gu0QY5AKGKBILx4qeuxQKx7K0KG7vqDRDl4BtGRSDIde7=GFDmx0p_gE=REIIEhBPd_xDIPqvQ8R0rD88nRvwmUm8RmllxWBU4o_FpfD", "/", ".xueqiu.com")))
Invoke-WebRequest -UseBasicParsing -Uri "https://xueqiu.com/query/v1/symbol/search/status.json?count=10&comment=0&symbol=SZ000729&hl=0&source=all&sort=time&page=1&q=&type=11&md5__1038=222029ad07-cFGItKAcIW_b2tFtiQ_IgFg%3DPTMYgGWTXgrgFVssAkygl_eq56GoxyTkZuQqS7xLABjrgtug0PGtgkPrZgbdgTGgSPGGiPrFP%3DlPABg%3DgGhIgsdgTiGPT2_xSvtlogp%2FfHgGPrlgpPrBUo%2Fgzog_JgrXLggWgYqKOwPA0lr_BjGbklASW2JgquTx%2FWTWATgvuPqg" `
-WebSession $session `
-Headers @{
"Accept"="*/*"
  "Accept-Encoding"="gzip, deflate, br, zstd"
  "Accept-Language"="zh-CN,zh;q=0.9"
  "Cache-Control"="no-cache"
  "Pragma"="no-cache"
  "Referer"="https://xueqiu.com/S/SZ000729"
  "Sec-Fetch-Dest"="empty"
  "Sec-Fetch-Mode"="cors"
  "Sec-Fetch-Site"="same-origin"
  "X-Requested-With"="XMLHttpRequest"
  "elastic-apm-traceparent"="00-23535e98bb2567a681094dbfc4cfa2b4-f9a19650dd15ddf9-00"
  "sec-ch-ua"="`"Not:A-Brand`";v=`"99`", `"Google Chrome`";v=`"145`", `"Chromium`";v=`"145`""
  "sec-ch-ua-mobile"="?0"
  "sec-ch-ua-platform"="`"Windows`""
}


根本问题：雪球使用阿里云安全 SDK 生成 xq_a_token，在 headless 浏览器中无法生成，导致所有 API 调用失败（404/400）。
解决方案：完全放弃 API 调用，改用 DOM 抓取 + 分页翻页：
1. browser.go：新增 NewBrowserHeaded() 方法，以有界面模式启动浏览器（绕过部分反爬检测）
2. xueqiu.go 重写：
   - Init()：访问主页 + 注入 navigator.webdriver=undefined stealth
   - ResolveStockCode()：直接访问 xueqiu.com/k?q=... 搜索页，从 DOM 提取股票代码
   - FetchDiscussions()：访问 /S/{code} 页面，通过点击 .pagination__next 翻页，每页提取 10 条讨论
   - extractVisibleItems()：从 .timeline__item DOM 提取作者、内容、时间、统计数据
   - parseRelativeTime()：解析"5分钟前"、"2小时前"、"3天前"等相对时间
3. main.go：runXueqiu() 改为使用 NewBrowserHeaded()



项目进行重构，我的目的是做一个替代curl的使用playwright加载动态网页获取网页内容的工具，但是主要的目的是加载动态网页，并获取网页内容，有通用的功能类似curl的内容，但是也有针对特定网站做的爬取，比如我已经实现的针对雪球的爬取，后面会针对其他例如新浪进行爬取等，现在我要做重构，要求做一个非常良好的设计（请好好思考良好设计的要素），一定是良好的设计，我对代码有洁癖；小提示 把 @cmd\durl\main.go 移动到 项目根目录下
