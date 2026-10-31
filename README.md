# Dazzled

A satirical public record for blinding headlights. The app includes an incident report form, aggregated incident leaderboards, a Peak Blindness chart (shown after 30 timed reports), a browser-local Retinal Damage Meter, a citation generator, and the passenger-mode Blind Drive game with a public score board.

Visit the live app at [dazzled-4g7p.onrender.com](https://dazzled-4g7p.onrender.com/).

## Stack and layout

- `frontend/app/`, `frontend/components/`, `frontend/lib/`: Next.js static React application with TypeScript.
- `backend/cmd/server`: composition root, process lifecycle, and graceful shutdown.
- `backend/api/rest/v1/core`: HTTP router, middleware, health checks, and static export serving.
- `backend/api/rest/v1/handlers`: REST controllers and request/response handling, organized by feature.
- `backend/api/rest/v1/ports/inbound`: service contracts used by REST handlers.
- `backend/internal/domain`: incident, score, and statistics types.
- `backend/internal/pkg/services`: business validation and feature use cases.
- `backend/internal/pkg/ports/outbound`: persistence contracts implemented by adapters.
- `backend/internal/pkg/database/sqlite`: SQLite connection and repository adapter.
- `backend/internal/pkg/database/migrations`: embedded, versioned migrations and transaction runner.
- `docs/architecture.md`: architecture, security boundaries, and SQLite trade-offs.

SQLite is the source of truth for incidents and game scores. Citations remain locally generated satire and are not legal documents. Blind Drive scores are anonymous and time-checked, though the browser still controls the game simulation.

## Data model

Migration `001_initial.sql` creates `incidents` and `leaderboard_scores`; `002_vehicle_choices_and_location.sql` expands vehicle and brand options and adds an optional user-entered location; `003_police_car_vehicle_type.sql` adds police cars as a reportable vehicle; `004_incident_local_hour.sql` adds the reporter-selected local hour for the Peak Blindness chart; `005_leaderboard_game_sessions.sql` adds one-use server-tracked game sessions. Existing reports are preserved by the migrations. Older reports have no local-hour value and are excluded from that chart. `schema_migrations` records applied versions. All writes use bound SQL parameters; migrations are applied in a transaction. No demo rows are inserted: a new database starts with zero reports and an empty scoreboard.

## Run locally with Docker

Docker Desktop must be running. No Node.js, pnpm, or Go installation is needed on the host for this path.

```sh
docker compose up --build
```

Open http://localhost:8080. Compose stores the SQLite database in the `dazzled-data` named volume. Stop the app with Ctrl+C, or run `docker compose down` in another terminal. `docker compose down -v` also deletes the database volume.

## Local development

For running the frontend and backend directly on your Mac, install Go 1.26.10, Node.js 22+, pnpm 12.3.4, and a C toolchain for the `mattn/go-sqlite3` driver. The Go toolchain is pinned to a patched release in `backend/go.mod`.

Copy `.env.example` to `.env`. In one terminal, run `make backend-dev`; in another, run `make frontend-dev`. Open http://localhost:3000. Next.js proxies `/api/*` to `http://localhost:8080`.

`DATABASE_PATH` selects the writable SQLite file (default `../data/dazzled.db` when started from the repository Makefile). The backend creates the parent directory and applies pending migrations when it starts. `WEB_DIR` points to the exported static site and defaults to `../frontend/out` when the Go server runs from `backend`. `LISTEN_ADDR` defaults to `:8080`. `API_ORIGIN` changes the development proxy target. `TRUST_PROXY_HEADERS` defaults to `false`; enable it only when the trusted reverse proxy overwrites `X-Forwarded-For`. `GAME_TOKEN_SECRET` is optional; provide a stable value of at least 32 bytes to preserve active runs across restarts, otherwise an ephemeral key is generated.

Useful commands:

```sh
make setup
make backend-dev
make frontend-dev
make migrate
make test
make lint
make build
```

The production image combines the frontend export and backend in one service. `make setup` installs the frontend and Go dependencies.

After `make build`, you can run the same production binary locally with:

```sh
DATABASE_PATH=./data/dazzled.db WEB_DIR=./frontend/out ./dist/dazzled
```

It listens on `:8080` by default and serves both the exported site and `/api/*` from that origin.

## Production with Compose

```sh
make docker-up
```

Run `make docker-down` to stop and remove the container. For foreground logs during local evaluation, use `docker compose up --build` as shown above.

The app listens on port 8080 and persists the database in the `dazzled-data` named volume. Anonymous write requests are limited to 60 per minute per client address, separately for incident reports and the combined leaderboard start/score endpoints. The limiter has a 4,096-entry LRU-bounded in-memory table and resets on restart. IPv6 addresses are grouped by `/64`; IPv4 addresses are keyed individually. This limits bursts but can still affect users sharing an address, and it cannot stop distributed traffic. Put a TLS reverse proxy in front of the app before public exposure and configure its own request-rate limits too. Back up before upgrades. The app applies migrations on startup; deploy a single instance for each SQLite file and coordinate upgrades so only one instance starts against that file at a time.

Each JSON write accepts one object with a 16 KiB maximum body; unknown fields and extra JSON values are rejected. An incident POST creates exactly one incident—there is no bulk-insert endpoint.

By default, the limiter keys requests by the TCP peer address, and a client-supplied `X-Forwarded-For` value is ignored. The automated check confirms this behavior. If the app is behind a trusted proxy that **overwrites** `X-Forwarded-For` with the original client address, set `TRUST_PROXY_HEADERS=true`; do not enable this when clients can reach the Go server directly or when the proxy merely appends to a client-supplied header. This setting trusts the first address in that header, so verify the deployed proxy path before enabling it. The frontend also keeps a count of successful reports in that browser's local storage; it is a novelty counter and is never sent to the server or leaderboard.

The Blind Drive server issues a signed token when a run starts. The start time is recorded in SQLite; posting a score consumes that session once, and the server rejects scores above `0.35 km/s × server elapsed time + 0.25 km`. A random signing key is generated at startup unless `GAME_TOKEN_SECRET` is set. Set a stable secret of at least 32 bytes (for example, `openssl rand -hex 32`) if unfinished runs should remain valid across restarts; keep it private and out of the repository. This makes score replay and absurd instant scores harder, but it cannot prove that the browser game was played honestly within the allowed speed.

### Incident location and privacy

The optional location is free text entered by the reporter, limited to 100 characters. The prompt asks for a city, road, or nearby landmark and says not to enter a home address. “Approximate” is guidance, not a precision guarantee: the app does not collect GPS, geocode the text, or validate how specific it is. The value is stored with the incident in SQLite indefinitely; there is no automatic incident-retention schedule. It is not returned by the public API or displayed in the site. The public incident endpoints expose a count and aggregate statistics only. The operator, or anyone with access to the database file or its backups, can read the stored location text. Delete the database/records and relevant backups to remove it.

The entrypoint starts as root only long enough to recursively set `/data` ownership to UID/GID `10001:10001`, then drops privileges with `su-exec`. The Go server runs as `dazzled`; the binary and static files stay root-owned. This also handles an existing volume or root-owned PaaS disk. If a platform forbids root entrypoints, provision `/data` with owner `10001:10001` or use Kubernetes `fsGroup: 10001`.

Before deploying a new image over an existing volume, take a backup. You can migrate its ownership once while the service is stopped; the entrypoint also corrects it during startup:

```sh
mkdir -p ./data
sudo chown 10001:10001 ./data
```

For Kubernetes, set pod `fsGroup: 10001` or provision the volume with matching ownership. If upgrading an existing named Docker volume created by an older image, stop the app and update its ownership once before starting the new image:

```sh
docker compose stop dazzled
docker run --rm -v dazzled_dazzled-data:/data alpine:3.22 chown -R 10001:10001 /data
docker compose start dazzled
```

Replace `dazzled_dazzled-data` with the actual volume name shown by `docker volume ls` if your Compose project uses a different name. Back up the database before changing an existing production volume.

Migrations are forward-only. To roll back an application release, stop it, restore the pre-upgrade database backup, then deploy the previous image.

Back up the live database through SQLite's online backup command, then copy the snapshot out of the container:

```sh
docker compose exec dazzled sqlite3 /data/dazzled.db ".backup /data/dazzled-backup.db"
docker cp "$(docker compose ps -q dazzled):/data/dazzled-backup.db" ./dazzled-backup.db
docker compose exec dazzled rm -f /data/dazzled-backup.db
```

For scheduled off-host backups on a Linux Docker host, install `rclone` and configure a private remote (for example, an S3-compatible bucket). The script creates a SQLite online backup, checks it with `PRAGMA integrity_check`, uploads it, and keeps seven days of local snapshots. The remote's credentials stay in the host's rclone config, never in this repository.

Test one backup manually:

```sh
RCLONE_REMOTE=offsite:dazzled-backups ./scripts/backup.sh
```

Then schedule it daily with the host's crontab (replace `/opt/dazzled` and the remote name):

```cron
15 2 * * * cd /opt/dazzled && RCLONE_REMOTE=offsite:dazzled-backups ./scripts/backup.sh >> ./backups/backup.log 2>&1
```

For restore, stop the service and copy the backup into the Compose volume (the volume is discovered from its Compose label):

```sh
docker compose stop dazzled
VOLUME_NAME="$(docker volume ls --filter label=com.docker.compose.volume=dazzled-data -q)"
docker run --rm -v "$VOLUME_NAME:/data" -v "$PWD:/backup:ro" alpine:3.22 sh -c 'cp /backup/dazzled-backup.db /data/dazzled.db && rm -f /data/dazzled.db-wal /data/dazzled.db-shm && chown 10001:10001 /data/dazzled.db && chmod 640 /data/dazzled.db'
docker compose start dazzled
```

## Tests and checks

`make test` runs Go race-enabled repository/API tests, React workflow tests, and the TypeScript compiler. `make lint` checks Go formatting, `go vet`, ESLint, and TypeScript types. `make build` creates the static frontend and optimized Go binary. Dependency scans can be run with `govulncheck ./...` from `backend` and `pnpm audit` from `frontend`.

## Troubleshooting

- Frontend reports cannot reach the server: confirm the Go server is listening on port 8080 and that `API_ORIGIN` matches it.
- Statistics or scoreboards are empty: a fresh database starts empty; submit a report or a game score to create real records.
- The server cannot open SQLite: check that `DATABASE_PATH`'s parent exists and is writable by the process. In Compose, keep `/data` mounted to `dazzled-data`.
- The static app returns 404 in production: confirm `WEB_DIR` points to the Next.js `out` directory (the image sets `/app/web`).
- SQLite reports lock contention: run only one app process per database file, avoid other writers on its volume, and take a backup before recovery work.

## Limitations

This app has no user accounts because the source experience had none. Incident reports remain anonymous and self-reported. The write limiter is a burst control, not a storage quota or protection from distributed bots. Blind Drive scores now require a signed, single-use session and are checked against server-measured elapsed time. The browser still controls the game, so a score within the generous speed ceiling can be manipulated. Add edge or reverse-proxy rate limiting before internet exposure. Responses include CSP with `frame-ancestors 'none'`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, and a referrer policy. SQLite is a single-host database and should be replaced when the workload requires concurrent multi-host writes or high availability.
