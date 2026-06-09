# The data-service talks to its database over ODBC, which requires cgo and the
# unixODBC libraries plus a driver. The image therefore uses a Debian base
# (not distroless static) and ships the SQLite ODBC driver for the default
# development backend.

# Build stage.
FROM golang:1.26.4-trixie AS build
WORKDIR /src

# unixODBC headers are needed to compile the cgo ODBC driver.
RUN apt-get update \
    && apt-get install -y --no-install-recommends unixodbc-dev \
    && rm -rf /var/lib/apt/lists/*

# Dependencies are vendored, so the build needs no Go module network access.
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -mod=vendor -trimpath -ldflags="-s -w" -o /out/data-service .

# Runtime stage.
FROM debian:trixie-slim
WORKDIR /app

# Runtime ODBC libraries + the SQLite ODBC driver (auto-registered in
# /etc/odbcinst.ini by the package), plus CA certs for any remote DB.
RUN apt update \
    && apt install -y --no-install-recommends \
        unixodbc \
        libsqliteodbc \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/data-service /app/data-service

# Persist the default SQLite database on a volume and run as a non-root user
# that owns it.
RUN useradd --system --uid 10001 --create-home appuser \
    && mkdir -p /data \
    && chown appuser:appuser /data
VOLUME ["/data"]

USER appuser
EXPOSE 9090
ENTRYPOINT ["/app/data-service"]
