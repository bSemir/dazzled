#!/bin/sh
set -eu

chown -R dazzled:dazzled /data
exec su-exec dazzled /app/dazzled "$@"
