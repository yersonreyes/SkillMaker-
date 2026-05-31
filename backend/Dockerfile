# Stage 1: Build static binary
FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Then copy source
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Stage 2: Runtime (distroless)
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=build /out/api /app/api

EXPOSE 3000
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]
