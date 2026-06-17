FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

EXPOSE 8080

COPY . .
ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target="/root/.cache/go-build" go build -o srv cmd/api/main.go

FROM alpine:3.24.1 AS runner

WORKDIR /bin

RUN apk add --no-cache curl

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD [ "curl -f localhost:8080/api/health || exit 1" ]

COPY ship.yml /etc/ship.yml
COPY --from=builder /app/srv /bin/srv

CMD ["/bin/srv"]
