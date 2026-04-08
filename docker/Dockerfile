FROM golang:1.26-alpine AS go-builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/mememory-server
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /admin ./cmd/mememory-admin

FROM node:22-alpine AS web-builder

WORKDIR /web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/ .
RUN pnpm build

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=go-builder /server /usr/local/bin/server
COPY --from=go-builder /admin /usr/local/bin/admin
COPY --from=web-builder /web/dist /app/web/dist

WORKDIR /app
EXPOSE 4200
ENTRYPOINT ["admin"]
