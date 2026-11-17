---
name: run-local-stack
description: Build the plugin and run it in a local Grafana with a RabbitMQ broker (docker compose), push demo data, rebuild after changes and read the plugin logs. Use to try a change in the real app or to debug streaming behaviour.
---

# Run the local Grafana + RabbitMQ stack

The stack is `docker-compose.yaml`: `grafana` (extends `.config/docker-compose-base.yaml`, mounts `dist/` as the plugin
and `provisioning/`) and `rabbitmq` (management image, stream plugin enabled via `docker/rabbitmq/enabled_plugins`).
Only one copy can run at a time (fixed container names), so stop it in other worktrees first.

## Steps

1. Build both halves into `dist/` (the container runs the **Linux** binary, also on Windows/macOS hosts):
   ```bash
   npm install            # first time only
   npm run build          # or `npm run dev` in another terminal for watch mode
   mage -v build:linux    # dist/gpx_rabbitmq_linux_amd64 (use build:linuxARM64 on ARM hosts)
   ```
2. Start the stack and wait until both containers are healthy:
   ```bash
   docker compose up -d --build --wait
   curl -s http://localhost:3000/api/health
   ```
   - Grafana: http://localhost:3000 (anonymous admin). Override the version with `GRAFANA_VERSION=13.3.1` (and
     `GRAFANA_IMAGE=grafana` for OSS).
   - RabbitMQ management: http://localhost:15672 (guest/guest). Override with `RABBITMQ_VERSION=4.1`.
3. Check the provisioned data source (uid `rabbitmq`) and the stream it declares:
   ```bash
   curl -s http://localhost:3000/api/datasources/uid/rabbitmq/health      # {"message":"RabbitMQ Connected","status":"OK"}
   curl -s -u guest:guest http://localhost:15672/api/queues | grep -o '"name":"[^"]*"'
   ```
4. Push demo data (random `{"name","value"}` JSON every 100 ms into `rabbitmq.stream`) and open the provisioned
   "RabbitMQ Dashboard":
   ```bash
   go run ./demo
   ```
   Or publish a single message through the management API:
   ```bash
   curl -s -u guest:guest -H 'content-type: application/json' \
     -d '{"properties":{},"routing_key":"rabbitmq.stream","payload":"{\"name\":\"manual\",\"value\":1}","payload_encoding":"string"}' \
     http://localhost:15672/api/exchanges/%2F/amq.default/publish
   ```
5. After changing code:
   - frontend: `npm run build` (or let `npm run dev` rebuild) and reload the browser;
   - backend: `mage -v build:linux && docker compose restart grafana`;
   - `src/plugin.json` or provisioning: `docker compose restart grafana`.
6. Read the plugin logs (the base compose file sets `GF_LOG_FILTERS=plugin.maormil-rabbitmq-datasource:debug`):
   ```bash
   docker logs maormil-rabbitmq-datasource 2>&1 | grep 'plugin.maormil-rabbitmq-datasource' | tail -50
   docker logs maormil-rabbitmq-datasource 2>&1 | grep -E 'panic|plugin process exited'   # crashes
   ```
7. Stop with `docker compose down` (add `-v` only if you also want to drop volumes).

## Notes

- Windows Git Bash: quote paths (the repo path has spaces); for `docker run -v` use `MSYS_NO_PATHCONV=1` and `$(pwd -W)`.
- A plugin crash shows as `plugin process exited` followed by `Restarting plugin`; a message that crashes the
  plugin is replayed from the first offset after every restart. Delete the stream in the management UI to stop it.
