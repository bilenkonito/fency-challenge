# Build stage.
FROM golang:1.26.4-alpine AS build
WORKDIR /src

# Dependencies are vendored (see ./vendor) to prevent network related issues in CI/CD (e.g. GitHub Actions runners without internet access).
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -trimpath -ldflags="-s -w" -o /out/auth-service .

# Runtime stage.
FROM gcr.io/distroless/static-debian13:nonroot
WORKDIR /app
COPY --from=build /out/auth-service /app/auth-service

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/auth-service"]
