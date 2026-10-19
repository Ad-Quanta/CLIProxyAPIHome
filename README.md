# Home

English | [中文](README_CN.md)

Home is a comprehensive credential **management, scheduling, and distribution** service designed for [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI).

It serves as a centralized hub for managing and scheduling API credentials in large-scale CLIProxyAPI cluster deployments. CLIProxyAPI nodes communicate with Home to retrieve necessary API credentials and report usage statistics.

## Home and Home Management Center (HMC)

**Home and HMC are maintained in separate open-source repositories and bundled together in official Home releases.** This repository contains the Go backend, which owns credential management, scheduling, authentication, database storage, and APIs. [Home Management Center (HMC)](https://github.com/router-for-me/Home-Management-Center) contains the React/TypeScript frontend, including the landing page, administrator console, and user workspace.

At runtime, HMC runs in the browser and calls Home's [Management API](docs/management/api.md) at `/v8/management` and [User API](docs/user/api.md) at `/user`. Home handles these requests and persists changes in its database. At build time, HMC's HTML, JavaScript, CSS, and other static assets are embedded into the Home executable through Go's `go:embed`.

### Deploying an official release

Official Home binaries and Docker images already include the HMC frontend. Deploy Home and its database, then open the Home HTTP address, such as `http://localhost:8327`:

| Path | Page |
| --- | --- |
| `/` or `/index.html` | Landing page |
| `/management.html` | Administrator console |
| `/user.html` | User workspace |

Node.js, Bun, and a separate frontend server are not required on the deployment host. Keep `management.disable-control-panel` set to `false` and configure a management key to use the administrator console; see [Management API authentication](docs/management/api.md#authentication). Database setup and configuration import are described under [Cluster Mode](#cluster-mode).

The production frontend uses APIs on the same origin as the page. If you host HMC assets separately, configure that origin's reverse proxy to forward `/v8/management/*` and `/user/*` to Home. Selecting a different backend URL in the frontend is available only in development mode.

### Developing both projects locally

Clone the repositories into sibling directories. Home requires the Go version declared in [go.mod](go.mod); HMC requires Node.js 24+ and Bun 1.3+.

```bash
git clone https://github.com/router-for-me/CLIProxyAPIHome.git
git clone https://github.com/router-for-me/Home-Management-Center.git
```

For a new local setup without `cluster.yaml`, copy Home's `config.example.yaml` to `config.yaml` and set a non-empty `management.secret-key`. Import that configuration into the local SQLite database, then start Home in one terminal:

```bash
cd CLIProxyAPIHome
go run ./cmd/home -config config.yaml -import
go run ./cmd/home -addr 127.0.0.1:8327
```

Import is an initialization step; editing `config.yaml` alone does not update the database-backed runtime. An existing configured backend can be used directly.

Start HMC in another terminal:

```bash
cd Home-Management-Center
bun install --frozen-lockfile
bun run dev
```

Open `http://localhost:3000/management.html`, enter `http://127.0.0.1:8327` as the management endpoint, and sign in with the management key. HMC serves frontend changes with hot reload while Home serves the APIs; frontend edits do not require rebuilding Home. For the user workspace at `http://localhost:3000/user.html`, HMC's development server proxies `/user` API requests to `http://127.0.0.1:8327` by default. Set `MANAGEMENT_DEV_PROXY_TARGET` when starting HMC if that backend address differs.

### Building and releasing the bundled frontend

The Home source checkout does not include generated HMC assets. To build a complete local executable, first build HMC and copy its entire `dist/` directory into Home's embedded asset directory. Starting from the parent of the two sibling repositories:

```bash
cd Home-Management-Center
bun install --frozen-lockfile
bun run build:embedded
cp -R dist/. ../CLIProxyAPIHome/internal/managementasset/static/
cd ../CLIProxyAPIHome
go build -o CLIProxyAPIHome ./cmd/home
```

The bundle includes `index.html`, `management.html`, `user.html`, the `assets/` directory, and `panel-manifest.json` with frontend build metadata. Copy the complete bundle, including the scripts and styles referenced by the HTML entries. The generated files under `internal/managementasset/static/` are ignored by Git. A plain `go build` or `docker build` does not fetch or build HMC; prepare these assets before either command. Once built, the executable serves its embedded frontend without needing the HMC source directory.

For local verification of an already-built frontend without recompiling Home, set `MANAGEMENT_STATIC_PATH` to the absolute path of HMC's complete `dist/` directory when starting Home. This explicitly replaces the embedded asset source with that directory.

The [binary release workflow](.github/workflows/release.yaml) runs on Home tag pushes, and the [Docker workflow](.github/workflows/docker-image.yml) runs on `v*` tags. Both follow this sequence:

1. Clone HMC's default branch and run `bun install --frozen-lockfile` and `bun run build:embedded`.
2. Copy the complete frontend bundle into `internal/managementasset/static/` and pass it to the platform build jobs.
3. Compile Home with the embedded frontend, then publish release archives or Docker images.

The workflows currently require the `HOME_MANAGEMENT_CENTER_TOKEN` repository secret to clone HMC, including when reused in a fork. They use HMC's default-branch revision at build time, without matching it to the Home release tag. For a custom build with a specific frontend version, check out the desired HMC revision before building.

Home does not download or automatically update the frontend at runtime. The `management.panel-github-repository` setting is retained for compatibility and does not switch the embedded bundle. Frontend updates reach normal deployments through a newly built Home binary or Docker image.

### Issues and contributions

Submit issues, feature requests, and pull requests to the repository responsible for the affected component:

- **Frontend / Web UI**: For page layout, interactions, themes, translations, or browser compatibility, use [HMC Issues](https://github.com/router-for-me/Home-Management-Center/issues) and submit code changes to HMC.
- **Home backend**: For Management API behavior, credential scheduling, database storage, clustering, or RESP communication, use [CLIProxyAPIHome Issues](https://github.com/router-for-me/CLIProxyAPIHome/issues) and submit code changes here.

Changes spanning the frontend and backend should link the corresponding issues or pull requests in both repositories. API changes must also update Home's [Management API documentation](docs/management/api.md) or [User API documentation](docs/user/api.md), as applicable, so HMC can integrate against the same contract.

## Key Features

- **Scalable Node Management**: Supports connecting an unlimited number of CLIProxyAPI nodes for credential distribution, ideal for large-scale deployments.
- **Multi-Provider OAuth2 Support**: Manages OAuth2 credentials for various AI model providers, including OpenAI, Claude, Codex, Antigravity, Kimi, xAI, and more.
- **High Availability**: Supports multi-account management and round-robin load balancing to ensure optimal performance and reliability.
- **Secure Access**: Provides a streamlined CLI authentication flow to ensure secure access to APIs.
- **Plugin Support**: Supports installing and enabling CLIProxyAPI-compatible plugins through Home Management.
- **Flexible Upstream Integration**: Easily integrate upstream providers via configuration, including:
  - OpenAI-compatible providers (e.g., OpenRouter)
  - OpenAI Responses protocol providers
  - Anthropic Messages protocol providers
  - Google Gemini protocol providers

## Cluster Mode

Home now always runs from a database-backed runtime. When `cluster.yaml` is absent, Home runs the same scheduler, node discovery, leader election, event watcher, management, and RESP logic on a local SQLite database. The default SQLite path is `home.db`; use `-sqlite-path` to override it.

When `cluster.yaml` exists, Home enables cluster mode and uses the database backend declared in that file. Copy `cluster.sqlite.example.yaml` or `cluster.pgsql.example.yaml` to `cluster.yaml` as a starting point. PostgreSQL accepts either individual connection fields or a connection string in `pgsql.postgres_uri`. PostgreSQL is recommended for multi-node deployments. SQLite is also supported, but cluster SQLite requires `node.external-ip` so other nodes and CPA clients can reach this node.

Set `slow-query-threshold` under the selected `pgsql` or `sqlite` backend to control slow SQL logging. It accepts Go duration strings such as `"500ms"` or `"1s"` and defaults to `"200ms"`.

`config.yaml` and `auth-dir` are no longer runtime storage. They are import/export exchange formats only. To migrate an existing local setup into the database, run:

```bash
./CLIProxyAPIHome -import
```

By default, import reads `./config.yaml` and then resolves credentials from the `auth-dir` value inside that config. Use `-config <path>` to select another config file, `-auth-dir <path>` to override credential discovery, and `-sqlite-path <path>` when importing into a non-default SQLite database. Import is idempotent, so rerunning it with the same files does not duplicate records.

To export config and credentials back to local files, run:

```bash
./CLIProxyAPIHome -export
```

Export writes `./config.yaml` and credential files under `~/.cli-proxy-api/` by default. Use `-export-dir <path>` to write `<path>/config.yaml` and credential files under `<path>/auths/`. It refuses to overwrite an existing `config.yaml` or a non-empty credential directory at the selected target.

`-export` / `-import` preserve the legacy exchange format. API keys are exported only as their actual key values, so Home database-only metadata such as API key display names, user ownership, and channel/model-group bindings is not a complete backup. Use `-db-export <snapshot.zip>` and `-db-import <snapshot.zip>` for full database backup and restore. Database snapshot format v4 includes API key display names and cannot be imported by older Home versions. Before upgrading, retain a database backup or an older-version-compatible snapshot for rollback.

In cluster mode, the Management API operates directly on database data. The default listen port comes from `node.port` in `cluster.yaml`; the startup `-addr` flag can override the listen address. Set `node.external-port` when a reverse proxy changes the port that clients must use; when omitted, the advertised cluster node port follows the final listen port.

RESP password authentication has been removed. Home RESP access only accepts mTLS-authenticated clients; CPA should connect with `-home-jwt` or configured TLS material. `allow-host` is only an IP allowlist and is not a password mechanism.

Docker Compose uses `docker-compose.sqlite.yml` or `docker-compose.pgsql.yml`. Copy the matching `cluster.sqlite.example.yaml` or `cluster.pgsql.example.yaml` to `cluster.yaml`, then use `/CLIProxyAPIHome/data/home.db` for SQLite or `postgres` as the PostgreSQL host. Select either database type when importing existing local state:

```bash
COMPOSE_FILE=docker-compose.sqlite.yml
# PostgreSQL: COMPOSE_FILE=docker-compose.pgsql.yml
docker compose -f "$COMPOSE_FILE" run --rm \
  -v "$PWD/config.yaml:/CLIProxyAPIHome/config.yaml:ro" \
  -v "$PWD/auths:/root/.cli-proxy-api" \
  home ./CLIProxyAPIHome -import
```

## Strict Credential Concurrency Limits

Before enabling credential concurrency policies, follow the [strict credential concurrency limits deployment runbook](docs/README.md). It defines the required SQLite and PostgreSQL topologies, certificate identity requirements, capability checks, and fail-closed mixed-version rollout.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository.
2. Create your feature branch (`git checkout -b feature/amazing-feature`).
3. Commit your changes (`git commit -m 'Add some amazing feature'`).
4. Push to the branch (`git push origin feature/amazing-feature`).
5. Open a Pull Request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
