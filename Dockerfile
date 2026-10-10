# syntax=docker/dockerfile:1

# ---- build: static binaries (no cgo; WebP is compiled in as WebAssembly) ----
FROM golang:1.27.1-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/cardmaker ./cmd/cardmaker \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/cardmaker-server ./cmd/cardmaker-server

# ---- runtime: Chromium plus the two binaries and the themes ----
FROM debian:bookworm-slim

# curl is only for the HEALTHCHECK. Themes embed their own fonts, so no system
# fonts are needed beyond what Chromium pulls in.
RUN apt-get update \
 && apt-get install -y --no-install-recommends chromium curl ca-certificates \
 && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/cardmaker /out/cardmaker-server /usr/local/bin/
COPY themes /app/themes

# Runs as an unprivileged user. HOME points to /tmp so Chromium has a writable
# profile directory whatever UID the container is started with (--user works,
# and so does a read-only root filesystem with a tmpfs on /tmp).
RUN useradd --system --uid 10001 --no-create-home cardmaker
USER cardmaker
WORKDIR /app
ENV HOME=/tmp

# Containers usually forbid the user namespaces the Chromium sandbox needs, so
# the sandbox is turned off here and the container is the isolation boundary:
# keep the user unprivileged and drop capabilities (see docker-compose.yml).
ENV CARDMAKER_CHROME=/usr/bin/chromium \
    CARDMAKER_CHROME_NO_SANDBOX=1

# The server listens on every interface, so it refuses to start without a
# token: pass -e CARDMAKER_API_TOKEN=... (see docs/API.md).
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -fsS http://127.0.0.1:8080/healthz || exit 1

# Run with --init: Chromium spawns child processes that need reaping.
CMD ["cardmaker-server", "-themes", "/app/themes", "-addr", "0.0.0.0:8080"]
