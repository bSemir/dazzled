# syntax=docker/dockerfile:1.7
# Build the static frontend and Go server into one minimal runtime image.
FROM node:22-alpine AS frontend
WORKDIR /src
RUN corepack enable
WORKDIR /src/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store pnpm install --frozen-lockfile
COPY frontend/app ./app
COPY frontend/components ./components
COPY frontend/lib ./lib
COPY frontend/public ./public
COPY frontend/next.config.mjs frontend/postcss.config.mjs frontend/tsconfig.json ./
ENV NODE_ENV=production
RUN pnpm build

FROM golang:1.26.10-alpine AS backend
RUN apk add --no-cache build-base
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod go mod download
COPY backend ./
RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -trimpath -ldflags='-s -w' -o /out/dazzled ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates sqlite su-exec \
    && addgroup -S -g 10001 dazzled \
    && adduser -S -D -H -u 10001 -G dazzled dazzled \
    && mkdir -p /data /app/web \
    && chown dazzled:dazzled /data
WORKDIR /app
COPY --from=backend /out/dazzled /app/dazzled
COPY --from=frontend /src/frontend/out /app/web
COPY docker/entrypoint.sh /usr/local/bin/dazzled-entrypoint
RUN chmod 0555 /usr/local/bin/dazzled-entrypoint
USER root
VOLUME ["/data"]
ENV LISTEN_ADDR=:8080 DATABASE_PATH=/data/dazzled.db WEB_DIR=/app/web
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s CMD su-exec dazzled wget -q -O /dev/null http://127.0.0.1:8080/readyz || exit 1
ENTRYPOINT ["/usr/local/bin/dazzled-entrypoint"]
