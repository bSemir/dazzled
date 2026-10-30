#!/bin/sh
set -eu
umask 077

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

: "${RCLONE_REMOTE:?Set RCLONE_REMOTE to a configured offsite rclone remote, such as offsite:dazzled-backups}"
backup_dir=${BACKUP_DIR:-"$repo_root/backups"}
retention_days=${BACKUP_RETENTION_DAYS:-7}
case "$retention_days" in
	''|*[!0-9]*) echo "BACKUP_RETENTION_DAYS must be a non-negative integer" >&2; exit 2 ;;
esac

mkdir -p "$backup_dir"
timestamp=$(date -u +%Y%m%dT%H%M%SZ)-$$
filename="dazzled-${timestamp}.db"
container_path="/data/$filename"
container_id=$(docker compose ps -q --status running dazzled)
if [ -z "$container_id" ]; then
	echo "Dazzled must be running before taking a backup" >&2
	exit 1
fi

staging_dir=$(mktemp -d "$backup_dir/.dazzled-backup.XXXXXX")
cleanup() {
	docker compose exec -T dazzled rm -f "$container_path" >/dev/null 2>&1 || true
	rm -rf "$staging_dir"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

docker compose exec -T dazzled sqlite3 /data/dazzled.db ".backup $container_path"
integrity=$(docker compose exec -T dazzled sqlite3 "$container_path" 'PRAGMA integrity_check;')
if [ "$integrity" != "ok" ]; then
	echo "SQLite backup integrity check failed: $integrity" >&2
	exit 1
fi

docker cp "$container_id:$container_path" "$staging_dir/$filename"
chmod 600 "$staging_dir/$filename"
mv "$staging_dir/$filename" "$backup_dir/$filename"
rclone copy "$backup_dir/$filename" "$RCLONE_REMOTE"
find "$backup_dir" -type f -name 'dazzled-*.db' -mtime "+$retention_days" -delete
printf 'Verified and uploaded %s\n' "$filename"
