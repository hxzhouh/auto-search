---

# Golang 内容抓取、AI 清洗、打标签、入库系统

## 需求文档 v1

## 1. 项目目标

开发一个使用 **Golang** 编写的内容采集程序，每天定时从 **Google News RSS** 抓取 AI 相关新闻，进一步抓取原文正文，使用 AI 对内容进行清洗和结构化分类，最后写入 **MySQL**，形成一个**已经分好类的素材库**，供后续选题、写作和分析使用。

---

## 2. 第一版范围

第一版只做一个信息源：

* **Google News RSS**

第一版目标流程：

```text
Google News RSS
→ 解析条目
→ 抓原文页面
→ 提取正文
→ AI 清洗 / 分类 / 摘要 / 打标签
→ 去重
→ 写入 MySQL
```

输出结果不是日报，不是综述，而是：

> 一套已经清洗好、分好类、可检索、可用于后续写作的素材库。

---

## 3. 运行方式

第一版程序形态：

* **CLI 程序**
* 后台长时间运行，每小时执行一次，抓取任务。


内部完成：

```text
fetch -> parse -> extract -> ai_clean -> tag -> save
```

---

## 4. 数据源设计

## 4.1 固定信息源

只抓：

* Google News RSS

格式类似：

```text
https://news.google.com/rss/search?q=...+when:1h

# 例如：
https://news.google.com/rss/search?q=AI+developer+when:1h
```

---

## 4.2 Query 列表设计

你说后期会从数据库读取，所以第一版建议把 query 也做成数据配置，而不是写死在代码里。

建议建一个 `feed_queries` 表，字段类似：

* `id`
* `name`
* `query_text`
* `lang`
* `region`
* `enabled`
* `priority`
* `created_at`
* `updated_at`

例如：

| name         | query_text               | enabled | priority |
| ------------ | ------------------------ | ------: | -------: |
| ai_developer | AI developer when:1d     |       1 |      100 |
| ai_coding    | AI coding when:1d        |       1 |       95 |
| openai       | OpenAI when:1d           |       1 |      100 |
| claude_code  | "Claude Code" when:1d    |       1 |       95 |
| cursor       | Cursor AI coding when:1d |       1 |       90 |

---

## 4.3 推荐的初始 Query 列表

第一版不要太多，建议先 8 到 12 条，够用了。

### 核心产品 / 公司

* `OpenAI when:1d`
* `Anthropic when:1d`
* `Google AI when:1d`
* `"Claude Code" when:1d`
* `Cursor AI coding when:1d`
* `GitHub Copilot when:1d`

### 开发者主题

* `AI developer when:1d`
* `AI coding when:1d`
* `AI coding agent when:1d`
* `developer tools AI when:1d`

### 行业趋势

* `AI startup when:1d`
* `AI agent enterprise when:1d`

### 后续可扩展

* `Gemini API when:1d`
* `DeepSeek when:1d`
* `AI benchmark when:1d`
* `AI security when:1d`
* `AI browser when:1d`
* `AI office tools when:1d`

---

## 5. 抓取频率

第一版：

* **每小时执行一次**

建议执行时间：

* 早上 7:00~9:00 之间

这样更适合做“今日可写素材库”。

---

## 6. 内容处理流程

## 6.1 第一步：抓取 RSS 条目

对每个 query：

1. 请求 Google News RSS
2. 解析 XML
3. 提取每条新闻的基础元数据

基础字段至少包括：

* 标题
* Google News 跳转链接
* 发布时间
* 来源媒体
* query 来源
* RSS 摘要
* 原始 XML 片段

---

## 6.2 第二步：解析原文链接

Google News 给出的链接往往不是最终原文链接，所以需要：

1. 跟随跳转
2. 获取最终 article URL
3. 保存 canonical URL

这一层很重要，因为后续去重和正文提取都依赖最终链接。

---

## 6.3 第三步：抓取原文页面

对最终 article URL：

1. 请求 HTML
2. 保存原始 HTML
3. 做正文提取

---

## 6.4 第四步：正文提取

目标是从原文页面中提取：

* article title
* author
* publish time（如果能提取）
* main content text
* optional: lead / summary

第一版建议策略：

* 优先提取**纯文本正文**
* 不强求保留复杂排版
* HTML 原文仍然入库，便于后续补救

---

## 7. AI 处理模块

---

## 7.1 AI 模块负责什么

第一版 AI 模块建议负责：

### 1）正文清洗

把原文正文处理成更干净的内容，例如：

* 去掉导航、版权、推荐阅读、广告尾巴
* 去掉明显无关段落
* 去掉重复段落
* 标准化空白字符
* 提取真正正文

### 2）语言识别

识别内容语言，例如：

* zh
* en
* mixed

### 3）内容分类

给内容打主分类，例如：

* news
* release
* tutorial
* opinion
* discussion
* funding
* security
* benchmark

### 4）主题标签生成

输出多个主题标签，例如：

* openai
* claude_code
* cursor
* ai_coding
* developer_tools
* agent
* startup
* enterprise
* security

---

## 7.2 AI 模块不建议第一版做的事

先别做太重：

* 不做长篇重写
* 不做自动成文
* 不做复杂事实核查
* 不做 embedding 检索
* 不做自动配图

第一版 AI 只做：

> 清洗、分类、标签化、结构化

---

## 7.3 AI 输出结构建议

AI 处理后应返回一个结构化结果：

* `cleaned_title`
* `cleaned_summary`
* `cleaned_content`
* `language`
* `content_type`
* `topics[]`
* `tags[]`
* `quality_score`
* `importance_score`
* `is_relevant`
* `reason`

这里最有价值的是：

* `content_type`
* `tags`
* `importance_score`
* `is_relevant`

因为你最终要的是素材库，不是简单存档。

---

## 8. 去重规则

必须做，不然同一新闻会重复很多次。

建议至少三层去重：

### 第一层：最终原文 URL 去重

* `canonical_url` 唯一

### 第二层：标题哈希去重

* 规范化标题后做 hash

### 第三层：内容哈希去重

* `cleaned_content` 做 hash

第一版主去重规则建议：

> `canonical_url` 为主，标题 hash 为辅

---

## 9. MySQL 表设计

你已经确定要：

* 主表
* tag 关系表

这个方向对。

---

## 9.1 主表 `contents`

建议字段：

* `id`

* `source`
  值固定：`google_news`

* `query_id`

* `query_text`

* `rss_title`

* `rss_link`

* `rss_source_site`

* `rss_summary`

* `rss_published_at`

* `final_url`

* `canonical_url`

* `url_hash`

* `title_hash`

* `content_hash`

* `article_title`

* `article_author`

* `article_published_at`

* `raw_html`

* `raw_content_text`

* `cleaned_title`

* `cleaned_summary`

* `cleaned_content`

* `language`

* `content_type`

* `quality_score`

* `importance_score`

* `is_relevant`

* `ai_reason`

* `status`
  例如：

  * pending
  * fetched
  * extracted
  * cleaned
  * saved
  * failed
  * duplicate

* `raw_payload`

* `created_at`

* `updated_at`

---

## 9.2 标签表 `tags`

建议字段：

* `id`
* `name`
* `category`
* `created_at`

其中 `category` 可取：

* company
* product
* topic
* type
* source
* region
* language

例如：

| name            | category |
| --------------- | -------- |
| openai          | company  |
| claude_code     | product  |
| ai_coding       | topic    |
| developer_tools | topic    |
| news            | type     |

---

## 9.3 关系表 `content_tags`

建议字段：

* `content_id`
* `tag_id`
* `created_at`

联合唯一键：

* `(content_id, tag_id)`

---

## 10. 素材库的最终效果

你最后要拿到的不是“所有新闻”，而是一个可筛选、可搜索的素材库。

理想状态下，你可以查：

### 今日高价值素材

* 最近一天
* `importance_score` 高
* `is_relevant = true`

### 某主题素材

* `ai_coding`
* `agent`
* `developer_tools`

### 某公司素材

* `openai`
* `anthropic`
* `google`

### 某内容类型

* `release`
* `opinion`
* `tutorial`

也就是说，系统产出的是：

> 已经分好类、去好重、提炼过的内容原料。

---

## 11. 程序模块划分

建议 Codex 按模块生成，不要一锅炖。

### 1）配置模块

* MySQL 配置
* AI 配置
* 抓取超时配置

### 2）Query 读取模块

* 从数据库读取启用的 query

### 3）RSS 抓取模块

* 请求 RSS
* 解析 XML
* 返回统一结构

### 4）链接解析模块

* 获取最终 article URL

### 5）网页抓取模块

* 拉取原文 HTML

### 6）正文提取模块

* 从 HTML 提取正文文本

### 7）AI 清洗与标签模块

* 调用模型
* 返回结构化结果

### 8）去重模块

* URL / title / content hash 检查

### 9）入库模块

* 保存主表
* 保存标签
* 保存关系表

### 10）任务编排模块

* 串联整条 pipeline
* 错误重试
* 日志记录

---

## 12. 第一版非目标

这能帮助 Codex 别乱发挥。

第一版不做：

* 多数据源
* 前端后台页面
* 自动发公众号
* 自动写文章
* 向量检索
* 推荐系统
* 实时抓取
* 分布式 worker
* 复杂反爬

---

## 13. 第一版推荐补充的字段

我建议再补两个非常有用的字段：

### `writeworthy_score`

表示“值不值得写”

这个分数可以由 AI 给出，范围例如 1~10。
判断依据：

* 信息量
* 时效性
* 影响范围
* 可展开性

### `angle_hint`

表示“可写角度”

例如：

* OpenAI 正在抢开发环境入口
* AI coding 竞争从模型转向工作流
* AI agent 安全开始成为企业刚需

这两个字段对你后续挑选“一天只写一篇”非常有用。

---

## 14. 我帮你收敛后的最终需求一句话

> 用 Golang 开发一个每天执行一次的 Google News RSS 内容采集程序，从数据库读取 query 列表，抓取新闻条目并进一步抓取原文，使用 AI 对正文做清洗、分类和标签化，去重后写入 MySQL 的主表与 tag 关系表，最终形成一个已经分好类、适合后续选题与写作的素材库。

---

## 15. 接下来最适合交给 Codex 的任务顺序

不要一口气让它全做，建议拆成 4 步。

### 第一步

先生成：

* MySQL 表结构
* Go 项目骨架
* 配置加载
* query 读取

### 第二步

再生成：

* Google News RSS 抓取
* XML 解析
* 链接解析
* 去重逻辑

### 第三步

再生成：

* 原文抓取
* 正文提取
* HTML 保存

### 第四步

最后生成：

* AI 清洗接口
* tag 入库
* pipeline 编排

这样成功率更高。

---

我建议下一步，我直接帮你写一份 **Codex 可直接执行的开发任务说明书**，包括：

* 项目目录结构
* MySQL DDL
* Go 包划分
* CLI 命令设计
* 每一步的验收标准
