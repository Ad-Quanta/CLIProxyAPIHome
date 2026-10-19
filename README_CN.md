# Home

[English](README.md) | 中文

Home 是一个为 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) 设计的全面凭证**管理、调度和分发**服务。

它作为在大规模 CLIProxyAPI 集群部署中管理和调度 API 凭证的中心枢纽。CLIProxyAPI 节点与 Home 通信以获取必要的 API 凭证并上报使用统计数据。

## Home 与 Home Management Center（HMC）的关系

**Home 与 HMC 在两个开源仓库中独立维护，并在 Home 的正式发行版中一起打包交付。** 本仓库提供 Go 后端，负责凭证管理、调度、认证、数据库存储及 API。[Home Management Center（HMC）](https://github.com/router-for-me/Home-Management-Center) 提供 React/TypeScript 前端，包括首页、管理员控制台和用户工作台。

运行时，HMC 在浏览器中执行，通过 `/v8/management` 下的 [Management API](docs/management/api_CN.md) 和 `/user` 下的 [User API](docs/user/api_CN.md) 与 Home 交互；Home 处理请求并将变更写入数据库。构建时，HMC 生成的 HTML、JavaScript、CSS 等静态资源通过 Go 的 `go:embed` 嵌入 Home 可执行文件。

### 部署正式发行版

Home 官方二进制和 Docker 镜像已包含 HMC 前端。部署 Home 及其数据库后，访问 Home 的 HTTP 地址，例如 `http://localhost:8327`：

| 路径 | 页面 |
| --- | --- |
| `/` 或 `/index.html` | 首页 |
| `/management.html` | 管理员控制台 |
| `/user.html` | 用户工作台 |

部署机器无需安装 Node.js、Bun 或单独运行前端服务。使用管理员控制台时，保持 `management.disable-control-panel` 为 `false`，并配置管理密钥；具体见 [Management API 认证说明](docs/management/api_CN.md)。数据库配置与配置导入方式见下文[集群模式](#集群模式)。

前端生产构建使用页面同源的 API。如果将 HMC 静态资源单独托管，需要在页面所在域名下通过反向代理，将 `/v8/management/*` 和 `/user/*` 请求转发给 Home。前端仅在开发模式下支持填写其他后端地址。

### 本地同时开发两个项目

将两个仓库克隆到同一父目录下。Home 需要 [go.mod](go.mod) 中声明的 Go 版本；HMC 需要 Node.js 24+ 和 Bun 1.3+。

```bash
git clone https://github.com/router-for-me/CLIProxyAPIHome.git
git clone https://github.com/router-for-me/Home-Management-Center.git
```

首次在本地搭建且没有 `cluster.yaml` 时，将 Home 的 `config.example.yaml` 复制为 `config.yaml`，填写非空的 `management.secret-key`。先将配置导入本地 SQLite 数据库，再在一个终端中启动 Home：

```bash
cd CLIProxyAPIHome
go run ./cmd/home -config config.yaml -import
go run ./cmd/home -addr 127.0.0.1:8327
```

导入属于初始化步骤；仅修改 `config.yaml` 不会更新数据库中的运行配置。已有配置好的后端时，可以直接连接该后端。

在另一个终端中启动 HMC：

```bash
cd Home-Management-Center
bun install --frozen-lockfile
bun run dev
```

打开 `http://localhost:3000/management.html`，将管理端点填写为 `http://127.0.0.1:8327`，使用管理密钥登录。HMC 开发服务负责前端热更新，Home 负责 API，修改前端无需重新编译 Home。开发用户工作台时，访问 `http://localhost:3000/user.html`；HMC 开发服务默认将 `/user` API 请求代理到 `http://127.0.0.1:8327`，后端地址不同时可在启动 HMC 时设置 `MANAGEMENT_DEV_PROXY_TARGET`。

### 前端内嵌构建与发布流程

Home 源码仓库不包含生成后的 HMC 资源。要在本地构建包含完整前端的可执行文件，需要先构建 HMC，再将整个 `dist/` 目录复制到 Home 的内嵌资源目录。从两个仓库的共同父目录开始执行：

```bash
cd Home-Management-Center
bun install --frozen-lockfile
bun run build:embedded
cp -R dist/. ../CLIProxyAPIHome/internal/managementasset/static/
cd ../CLIProxyAPIHome
go build -o CLIProxyAPIHome ./cmd/home
```

前端产物包括 `index.html`、`management.html`、`user.html`、`assets/` 目录，以及记录前端构建信息的 `panel-manifest.json`。复制时应包含完整产物和 HTML 引用的脚本、样式。`internal/managementasset/static/` 下的生成文件已被 Git 忽略。单独执行 `go build` 或 `docker build` 不会拉取或构建 HMC，执行前应先准备这些资源。编译完成后，Home 可执行文件即可提供内嵌页面，运行时无需保留 HMC 源码目录。

如果只想在本地验证构建好的前端而不重新编译 Home，可在启动 Home 时将 `MANAGEMENT_STATIC_PATH` 设为 HMC 完整 `dist/` 目录的绝对路径，显式使用该目录替代内嵌资源。

[二进制发布工作流](.github/workflows/release.yaml) 在推送 Home 标签时运行，[Docker 工作流](.github/workflows/docker-image.yml) 在推送 `v*` 标签时运行。两者均按以下顺序执行：

1. 克隆 HMC 默认分支，执行 `bun install --frozen-lockfile` 和 `bun run build:embedded`。
2. 将完整前端产物复制到 `internal/managementasset/static/`，传递给各平台构建任务。
3. 编译包含内嵌前端的 Home，再发布二进制压缩包或 Docker 镜像。

当前工作流仍要求配置 `HOME_MANAGEMENT_CENTER_TOKEN` 仓库 Secret 来克隆 HMC，Fork 后复用工作流时也需配置。工作流使用构建时的 HMC 默认分支版本，不会按 Home 发布标签匹配 HMC 标签；自定义构建若需固定前端版本，应先在 HMC 仓库检出指定版本再构建。

Home 运行时不会下载或自动更新前端。`management.panel-github-repository` 配置仅为兼容保留，修改它不会切换已内嵌的前端。常规部署通过更新重新构建的 Home 二进制或 Docker 镜像来更新前端。

### 问题反馈与贡献

提交问题、功能建议和 Pull Request 时，请根据涉及的组件选择对应仓库：

- **前端 / Web UI**：页面布局、交互、主题、翻译或浏览器兼容性相关问题，请提交到 [HMC Issues](https://github.com/router-for-me/Home-Management-Center/issues)，代码修改提交至 HMC 仓库。
- **Home 后端**：Management API 行为、凭证调度、数据库存储、集群或 RESP 通信相关问题，请提交到 [CLIProxyAPIHome Issues](https://github.com/router-for-me/CLIProxyAPIHome/issues)，代码修改提交至本仓库。

同时涉及前后端的改动，应在两个仓库中关联对应的 Issue 或 Pull Request。API 变更还需同步更新 Home 的 [Management API 文档](docs/management/api_CN.md) 或 [User API 文档](docs/user/api_CN.md)，使 HMC 按同一接口约定完成对接。

## 核心特性

- **可扩展的节点管理**：支持连接无限数量的 CLIProxyAPI 节点进行凭证分发，非常适合大规模部署。
- **多提供商 OAuth2 支持**：管理包括 OpenAI、Claude、Codex、Antigravity、Kimi、xAI 等在内的各种 AI 模型提供商的 OAuth2 凭证。
- **高可用性**：支持多账号管理和轮询负载均衡，以确保最佳性能和可靠性。
- **安全访问**：提供精简的 CLI 身份验证流程，确保安全访问 API。
- **插件支持**：支持通过 Home Management 安装并启用兼容 CLIProxyAPI 的插件。
- **灵活的上游集成**：通过配置轻松集成上游提供商，包括：
  - OpenAI 兼容提供商（例如 OpenRouter）
  - OpenAI Responses 协议提供商
  - Anthropic Messages 协议提供商
  - Google Gemini 协议提供商

## 集群模式

Home 现在始终使用数据库驱动的运行时。没有 `cluster.yaml` 时，Home 使用本地 SQLite 数据库运行同一套调度、节点发现、选主、事件监听、Management 和 RESP 逻辑。默认 SQLite 路径是 `home.db`，可以通过 `-sqlite-path` 覆盖。

存在 `cluster.yaml` 时，Home 启用集群模式，并使用该文件中声明的数据库后端。可以复制 `cluster.sqlite.example.yaml` 或 `cluster.pgsql.example.yaml` 为 `cluster.yaml` 后再修改。PostgreSQL 既支持拆分连接字段，也支持在 `pgsql.postgres_uri` 中填写连接字符串。多节点部署推荐使用 PostgreSQL；SQLite 也可以作为集群后端，但必须配置 `node.external-ip`，否则其他 Home 节点和 CPA 客户端无法稳定发现这个节点。

可以在所选的 `pgsql` 或 `sqlite` 后端下设置 `slow-query-threshold`，控制慢 SQL 日志阈值。该字段接受 `"500ms"`、`"1s"` 等 Go duration 字符串，默认为 `"200ms"`。

`config.yaml` 和 `auth-dir` 不再是运行时存储，只作为导入/导出的交换格式。迁移已有本地配置时，先运行：

```bash
./CLIProxyAPIHome -import
```

导入默认读取当前目录的 `config.yaml`，再根据该配置里的 `auth-dir` 查找凭证。可以通过 `-config <path>` 指定配置文件，通过 `-auth-dir <path>` 覆盖凭证目录，通过 `-sqlite-path <path>` 导入到非默认 SQLite 数据库。导入是幂等的，同一批文件重复导入不会产生重复记录。

需要将配置和凭证导出为本地文件时，运行：

```bash
./CLIProxyAPIHome -export
```

导出默认会写出当前目录的 `config.yaml`，并把凭证写到 `~/.cli-proxy-api/`。可以通过 `-export-dir <path>` 写出 `<path>/config.yaml`，并把凭证写到 `<path>/auths/`。如果目标 `config.yaml` 已存在，或者目标凭证目录已存在且非空，导出会拒绝覆盖。

`-export` / `-import` 保留旧版交换格式。API key 只会导出实际 key 值，因此 Home 数据库专属的 API key 展示名称、用户归属、channel/model group 绑定等元数据不构成完整备份。完整数据库备份和恢复应使用 `-db-export <snapshot.zip>` 与 `-db-import <snapshot.zip>`。数据库快照 v4 包含 API key 展示名称，旧版 Home 无法导入；升级前应保留数据库备份或旧版本兼容快照，以便回滚。

集群模式下，Management API 会直接操作数据库中的数据。默认监听端口来自 `cluster.yaml` 的 `node.port`；也可以通过启动参数 `-addr` 覆盖监听地址。如果前置反代改变了 client 实际需要连接的端口，可以设置 `node.external-port`；未设置时，对外发布的集群节点端口仍以最终监听端口为准。

RESP 密码认证已经移除。Home RESP 只接受通过 mTLS 认证的客户端；CPA 应使用 `-home-jwt` 或配置好的 TLS 材料连接。`allow-host` 只是 IP 允许列表，不是密码机制。

Docker Compose 使用 `docker-compose.sqlite.yml` 或 `docker-compose.pgsql.yml`。将对应的 `cluster.sqlite.example.yaml` 或 `cluster.pgsql.example.yaml` 复制为 `cluster.yaml` 后，SQLite 路径使用 `/CLIProxyAPIHome/data/home.db`，PostgreSQL 主机使用 `postgres`。导入已有本地配置时二选一：

```bash
COMPOSE_FILE=docker-compose.sqlite.yml
# PostgreSQL: COMPOSE_FILE=docker-compose.pgsql.yml
docker compose -f "$COMPOSE_FILE" run --rm \
  -v "$PWD/config.yaml:/CLIProxyAPIHome/config.yaml:ro" \
  -v "$PWD/auths:/root/.cli-proxy-api" \
  home ./CLIProxyAPIHome -import
```

## 严格凭证并发限制

启用 credential concurrency policy 前，必须遵循[严格凭证并发限制部署运行手册](docs/CN.md)。手册规定了 SQLite 和 PostgreSQL 所需的拓扑、certificate identity 要求、capability 检查和 fail-closed mixed-version 发布方式。

## 贡献

欢迎贡献！请随时提交 Pull Request。

1. Fork 仓库
2. 创建您的功能分支（`git checkout -b feature/amazing-feature`）
3. 提交您的更改（`git commit -m 'Add some amazing feature'`）
4. 推送到分支（`git push origin feature/amazing-feature`）
5. 打开 Pull Request

## 许可证

此项目根据 MIT 许可证授权 - 有关详细信息，请参阅 [LICENSE](LICENSE) 文件。
