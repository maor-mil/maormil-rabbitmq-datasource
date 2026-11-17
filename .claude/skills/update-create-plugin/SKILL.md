---
name: update-create-plugin
description: Update the plugin scaffolding to the latest @grafana/create-plugin (configs in .config/, dependencies, migrations) and verify nothing broke, including on Windows where the CLI refuses to run. Use when asked to update create-plugin, the Grafana packages or the plugin tooling.
---

# Update @grafana/create-plugin

The current scaffolding version is in `.config/.cprc.json`. Docs:
https://grafana.com/developers/plugin-tools/how-to-guides/updating-a-plugin.md and the create-plugin CHANGELOG
(https://github.com/grafana/plugin-tools/blob/main/packages/create-plugin/CHANGELOG.md).

## Steps

1. Start from a clean working tree on a new branch (the CLI refuses to run with uncommitted changes unless `--force`).
2. Run the update:
   - Linux/macOS/WSL: `npx @grafana/create-plugin@latest update --commit` (one commit per migration).
   - **Windows**: the CLI exits with "Create plugin does not support Windows". Run it in a container instead. Inside
     the container the worktree's `.git` file points to a Windows path, so git does not work there: pass `--force`
     (skips the git checks) and commit from Windows afterwards:
     ```bash
     MSYS_NO_PATHCONV=1 docker run --rm -v "$(pwd -W):/app" -v cp-npm-cache:/root/.npm -w /app -e CI=true node:24 \
       bash -c 'npx --yes @grafana/create-plugin@latest update --force < /dev/null'
     rm -rf node_modules && npm install   # replace the Linux node_modules created in the container
     ```
3. Review the diff. Migrations only touch what they know about; check in particular:
   - `package.json`: the migrations may leave `@grafana/*`, React and TypeScript older than the current template
     (`packages/create-plugin/templates/common/_package.json` in grafana/plugin-tools). Align them, keeping
     `grafanaDependency` in `src/plugin.json` consistent with the `@grafana/*` major.
   - `docker-compose.yaml`: must keep the `rabbitmq` service, `depends_on` and the `grafana_version` override.
   - `.github/workflows/`: `update` does **not** rewrite `ci.yml`/`release.yml`; compare with
     `templates/github/workflows/` and port changes while keeping this repo's differences (PR trigger on any base
     branch, RabbitMQ in E2E, report upload, `go-version`/`node-version` in `release.yml`).
   - New agent files (`.config/AGENTS/`) are not added by `update`; copy them from `templates/common` if they change.
4. Go side: `go get github.com/grafana/grafana-plugin-sdk-go@latest && go mod tidy` (the `go` directive follows the
   SDK's requirement; the Go toolchain downloads it automatically).
5. React major bumps: `npx -y @grafana/react-detect@latest`.
6. Verify everything: `npm run typecheck && npm run lint && npm run test:ci && npm run build`,
   `mage -v buildAll && mage coverage`, golangci-lint (see CLAUDE.md), then the E2E suite against the new default
   Grafana version (`run-local-stack` + `e2e-test` skills). Push a draft PR so the CI matrix covers the minimum
   and latest Grafana versions.
7. Add a `## Unreleased` CHANGELOG entry (call out a raised minimum Grafana version as a breaking change).
