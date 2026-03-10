# Auto Search

一个用 Go 编写的 AI 新闻素材采集系统。

当前流程已经打通：

```text
Google News RSS
-> 发现新闻条目
-> 解码原站 URL
-> defuddle 提取正文 Markdown
-> NVIDIA/OpenAI 兼容接口做 AI 清洗
-> 打分、打标签
-> 写入 SQLite / MySQL
-> 前端页面只展示 cleaned 数据
```

## 当前能力

- 数据源：Google News RSS
- 链路：
  - `discover run` 抓 RSS 并写入待处理内容
  - `extract run` 解析原站 URL，并通过 `http://defuddle.md/{url}` 提取正文
  - `clean run` 调用 AI 做清洗、分类、打分、标签化
  - `serve` 提供前端页面和 `/api/cleaned` 接口
- 数据库：
  - 本地开发默认用 SQLite
  - 生产可切换到 MySQL
- 前端：
  - 只展示 `status = cleaned` 的内容
  - 支持搜索、按类型过滤、按语言过滤

## 目录说明

- [cmd/auto-search/main.go](/Users/hxzhouh/workspace/github/me/auto-search/cmd/auto-search/main.go)
  CLI 入口
- [internal/discovery/service.go](/Users/hxzhouh/workspace/github/me/auto-search/internal/discovery/service.go)
  RSS 发现流程
- [internal/extraction/service.go](/Users/hxzhouh/workspace/github/me/auto-search/internal/extraction/service.go)
  原文提取流程
- [internal/cleaning/service.go](/Users/hxzhouh/workspace/github/me/auto-search/internal/cleaning/service.go)
  AI 清洗流程
- [internal/webui/server.go](/Users/hxzhouh/workspace/github/me/auto-search/internal/webui/server.go)
  前端页面和 API
- [configs/config.local.json](/Users/hxzhouh/workspace/github/me/auto-search/configs/config.local.json)
  本地配置
- [configs/config.mysql.example.json](/Users/hxzhouh/workspace/github/me/auto-search/configs/config.mysql.example.json)
  MySQL 配置示例

## 配置说明

本地默认配置文件：

- [config.local.json](/Users/hxzhouh/workspace/github/me/auto-search/configs/config.local.json)

关键配置项：

- `database.driver`
  可选 `sqlite` / `mysql`
- `defuddle.base_url`
  当前默认 `http://defuddle.md`
- `ai.base_url`
  当前使用 NVIDIA OpenAI 兼容接口
- `ai.model`
  当前可用模型为 `nvidia/nemotron-3-nano-30b-a3b`
- `ai.rpm_limit`
  当前按 `40 RPM` 控速
- `ai.timeout_seconds`
  AI 请求超时，当前默认 `120`

## 本地运行

### 1. 初始化数据库

```bash
go run ./cmd/auto-search migrate -config configs/config.local.json
```

### 2. 查看启用的 query

```bash
go run ./cmd/auto-search queries list -config configs/config.local.json
```

### 3. 抓取新闻条目

```bash
go run ./cmd/auto-search discover run -config configs/config.local.json
```

### 4. 提取正文

```bash
go run ./cmd/auto-search extract run -config configs/config.local.json -limit 20
```

### 5. AI 清洗

```bash
go run ./cmd/auto-search clean run -config configs/config.local.json -limit 10
```

### 6. 启动前端页面

```bash
go run ./cmd/auto-search serve -config configs/config.local.json -addr 127.0.0.1:8080
```

打开：

```text
http://127.0.0.1:8080
```

也可以直接在配置文件里设置监听地址：

```json
"web": {
  "host": "127.0.0.1",
  "port": 8080
}
```

`serve` 默认读取配置里的 `web.host` 和 `web.port`，如果传了 `-addr`，则以命令行参数为准。

## 前端接口

- `GET /`
  前端页面
- `GET /api/cleaned?limit=200`
  返回 JSON 格式的 cleaned 数据，响应结构为 `{"source":"cleaned","limit":200,"count":3,"items":[...]}`，默认 `limit=200`，最大 `500`
- `GET /healthz`
  健康检查

## 当前状态

当前本地 SQLite 数据状态：

- `cleaned = 3`
- `extracted = 1`
- `pending = 424`

说明：

- 新提取的数据已经会自动解出 Google News 原站 URL
- 历史旧数据里仍可能存在 Google News 中转链接，后续可以做回填修复

## 部署

仓库内提供远端部署脚本：

- [deploy.sh](/Users/hxzhouh/workspace/github/me/auto-search/deploy.sh)

它会执行：

```text
git pull
-> go build
-> 停旧进程
-> 后台启动 serve
```

最小用法：

```bash
bash deploy.sh
```

可覆盖环境变量：

- `APP_DIR`
- `GIT_BRANCH`
- `CONFIG_PATH`
- `LOG_DIR`
- `BIN_DIR`
- `RUN_DIR`

`deploy.sh` 的监听地址只读取配置文件里的 `web.host` 和 `web.port`；如果配置里没有，则默认使用 `0.0.0.0:8080`。

## 验证

项目当前测试命令：

```bash
go test ./...
```
