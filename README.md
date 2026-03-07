# health-go

这是从 `health`(ThinkPHP) 迁移出的 Go 后端版本，保持原接口路径与返回结构。  
现在默认使用独立 SQLite（不影响原 PHP/MySQL）。

## 运行

1. 先生成 SQLite 副本（从 `../health/health.sql` 导入）：

```bash
cd health-go
go run ./cmd/sqlite_copy
```

2. 配置环境变量（已内置 `.env`，可直接用）：

```bash
cat .env
```

3. 启动 Go 服务（自动读取 `.env`）：

```bash
go run ./cmd/server
```

## Windows 部署

1. 将整个 `health-go` 目录复制到 Windows（包含 `health.db`、`.env`、`public`）。
2. 安装 Go 1.22+，确认 `go version` 可用。
3. 在 Windows 终端启动：

```bat
cd /d D:\your-path\health-go
go run .\cmd\server
```

4. 访问地址：
- 本机：`http://127.0.0.1:8080`
- 局域网：`http://<Windows_IPv4>:8080`

5. 如需可执行文件：

```bat
cd /d D:\your-path\health-go
go build -o health-go.exe .\cmd\server
.\health-go.exe
```

6. 如局域网访问失败，请在 Windows 防火墙放行 `8080` 入站。

## 智能启动脚本（含国内源判断）

已提供脚本：
- `scripts/smart-start.ps1`
- `scripts/smart-start.bat`

功能：
- 自动读取 `.env`
- 自动探测网络环境并设置 `GOPROXY`
  - 可访问国际网络：`https://proxy.golang.org,direct`
  - 受限/国内网络：`https://goproxy.cn,direct`
- 若 `DB_DRIVER=sqlite` 且缺少 `health.db`，自动执行 `go run ./cmd/sqlite_copy` 生成
- 自动执行 `go mod tidy` 后启动服务

Windows 使用：

```bat
cd /d D:\your-path\health-go
scripts\smart-start.bat
```

可选参数：

```bat
scripts\smart-start.bat -Build      :: 先编译再启动 exe
scripts\smart-start.bat -NoMirror   :: 不自动切换 GOPROXY
scripts\smart-start.bat -Addr :9090 :: 指定监听端口
```

## Markdown 文档网页预览

已提供一个静态页面，可直接读取并渲染 Markdown API 文档：

- 页面地址：`/static/docs/index.html`
- Markdown 文件：`/static/docs/api.md`

启动服务后可直接访问：

```text
http://127.0.0.1:8080/static/docs/index.html
```

## 已迁移内容

- 登录、短信登录、注册
- 用户信息/密码/昵称/头像/联系人
- 轮播图、公告、社区
- 新闻分类/列表/详情/点赞
- 评论发布/列表/点赞
- 活动推荐/列表/分类/详情/搜索
- 课程列表/详情/章节
- 通用上传
- `material/answer` 路由（按现有 SQL 结构做兼容实现）

## 说明

- 数据库表名按 ThinkPHP 前缀映射：`tp_*`
- `health-go/health.db` 为 Go 专用副本，不会改动 PHP 的 MySQL
- JWT 规则与 PHP 一致：`HS256`，密钥为 `md5("tp6.1.4")`，默认 2 小时过期
- 静态资源目录已复制到 `health-go/public`
