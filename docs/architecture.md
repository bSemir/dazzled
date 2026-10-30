# Architecture

Dazzled is a same-origin web application. The statically exported React/TypeScript interface is served by the Go process. The browser calls `/api/*`; local Next.js development proxies those requests to the Go server.

## Request flow

The Go backend separates HTTP handlers, application services, repository interfaces, and SQLite persistence. The server entry point wires these parts together. The frontend submits incident reports and game scores as JSON. Incident reports can include an optional free-text place (up to 100 characters); the app does not request GPS coordinates. The text is stored indefinitely with the report, can be read by the operator from SQLite, and is not returned by public APIs. Public incident endpoints return counts and aggregates only. HTTP handlers strictly decode bounded request bodies and map service errors to stable response codes. SQLite is authoritative for records and aggregates.

The SQLite connection configures foreign-key enforcement, WAL journaling, a 5-second busy timeout, and one open connection. Schema changes are versioned SQL files under `backend/internal/pkg/database/migrations` and applied transactionally at process startup by the migration runner. Run one application instance per database file; take a backup before deploying schema changes.

## Security and operational boundaries

The public incident and score endpoints are intentionally anonymous because the existing UI has no account or identity workflow. POST requests are rate-limited to 60 per minute by address; IPv6 addresses are grouped by `/64`, the in-memory table is capped at 4,096 entries with LRU eviction, and proxy headers are ignored unless explicitly trusted. Server-side validation, strict JSON decoding, a 16 KiB body cap, parameterized SQL, generic client errors, response security headers, and non-root container execution limit common risks. A reverse proxy should add its own rate limits. Blind Drive scores require a signed one-use session and are checked against server elapsed time, but the browser game itself is not server-simulated and can still be manipulated within the permitted ceiling.

SQLite is intended for a single host and modest write traffic. Use persistent local/block storage and scheduled consistent backups (`sqlite3 /data/dazzled.db '.backup /backup/dazzled.db'`). Restore by stopping the service, replacing the database file, and restarting. Move to a client-server database if write lock contention, multi-host deployment, or storage-level availability requirements exceed SQLite's single-host model.
