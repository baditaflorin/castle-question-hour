# syntax=docker/dockerfile:1.7

# ----- builder -----
FROM golang:1.25-alpine AS build
ARG VERSION=dev
WORKDIR /src

# Cache go modules separately from source for fast incremental builds.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o /out/server ./cmd/server

# ----- runtime -----
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
ARG VERSION=dev

# OCI labels
LABEL org.opencontainers.image.source="https://github.com/baditaflorin/castle-question-hour"
LABEL org.opencontainers.image.revision="${VERSION}"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.title="castle-question-hour"
LABEL org.opencontainers.image.description="Anonymous hourly reflection prompts for castle gatherings."
LABEL org.opencontainers.image.licenses="MIT"

USER nonroot:nonroot
WORKDIR /app

COPY --from=build /out/server /app/server
COPY configs/questions.json /etc/cqh/questions.json

EXPOSE 8080
ENV SERVER_ADDR=":8080" \
    QUESTIONS_FILE=/etc/cqh/questions.json \
    LOG_LEVEL=info

HEALTHCHECK --interval=15s --timeout=3s --retries=3 \
    CMD ["/app/server", "-healthcheck"]

ENTRYPOINT ["/app/server"]
