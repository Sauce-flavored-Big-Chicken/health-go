# health-go

一个基于 Go + Gin 的健康社区后端服务，支持用户登录注册、资讯、活动、课程、评论、文件上传等接口，并内置静态资源与接口文档页面。

## 功能概览

- 用户体系：登录、手机验证码登录、注册、个人信息维护
- 内容能力：新闻、公告、评论、点赞
- 社区能力：友邻帖子与评论、活动报名与签到
- 文件能力：通用上传、答题素材上传
- 文档与静态资源：`/static`、`/storage`、`/docs`

## 项目结构

```text
cmd/server/              服务入口
cmd/sqlite_copy/         sqlite 工具
public/static/           前端静态资源
public/storage/          上传与存储目录
scripts/                 启动/部署脚本（含 Windows）
linux-run.sh             Linux 一键拉取并启动脚本
```

## 环境要求

- Go `1.24+`
- Linux / macOS / Windows
- 默认使用 SQLite（开箱即用）
- 可选 MySQL

## 配置

复制并修改环境变量文件：

```bash
cp .env.example .env
```

主要配置项：

- `DB_DRIVER`：`sqlite`（默认）或 `mysql`
- `SQLITE_PATH`：SQLite 数据库文件路径（默认 `health.db`）
- `MYSQL_DSN`：MySQL 连接串（仅 `DB_DRIVER=mysql` 时生效）
- `APP_ADDR`：服务监听地址（默认 `:8080`）
- `HTTP_URL`：对外访问地址（用于部分返回字段）

## 本地运行

```bash
go mod tidy
go run ./cmd/server
```

启动后访问：

- 首页：`http://127.0.0.1:8080/`
- API 文档页：`http://127.0.0.1:8080/static/docs/index.html`
- Markdown 文档：`http://127.0.0.1:8080/docs/api.md`

## 构建与运行二进制

```bash
go build -o server ./cmd/server
./server
```

## Linux 一键更新并启动

仓库内置脚本 [`linux-run.sh`](./linux-run.sh)，可自动：

1. 克隆或进入项目目录
2. 拉取 `main` 最新代码
3. 编译服务
4. 后台启动并写入日志

直接执行：

```bash
bash linux-run.sh
```

也可在服务器通过脚本 URL 直接执行：

```bash
curl -fsSL https://raw.githubusercontent.com/Sauce-flavored-Big-Chicken/health-go/main/linux-run.sh | bash
```

或下载后执行：

```bash
wget -O linux-run.sh https://raw.githubusercontent.com/Sauce-flavored-Big-Chicken/health-go/main/linux-run.sh
chmod +x linux-run.sh
./linux-run.sh
```

可选环境变量：`REPO_URL`、`BRANCH`、`APP_DIR`

## 常见命令

```bash
# 查看运行日志
tail -f ~/health-go/logs/server.log

# 查看进程 PID
cat ~/health-go/.server.pid
```

## 说明

- 旧版详细接口文档已迁移到 `public/static/docs/api.md` 与页面文档中维护。
- 根目录仍保留部分历史文档文件用于兼容已有路由，不建议随意删除。
