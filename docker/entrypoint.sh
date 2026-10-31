#!/bin/sh
# Fix persistent-data ownership before dropping privileges to the app user.
set -eu

chown -R dazzled:dazzled /data
exec su-exec dazzled /app/dazzled "$@"
