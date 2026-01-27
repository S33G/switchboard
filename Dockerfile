FROM golang:1.25.4-alpine AS builder

WORKDIR /src/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/switchboard .

FROM node:22-alpine AS ui-builder

WORKDIR /src/ui

RUN corepack enable

COPY ui/package.json ui/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY ui/ ./
RUN pnpm build

FROM alpine:3.21

RUN apk add --no-cache \
      bash \
      ca-certificates \
      dumb-init \
      docker-cli \
      openssh-client \
      tzdata

WORKDIR /app

COPY --from=builder /out/switchboard /usr/local/bin/switchboard
COPY --from=ui-builder /src/ui/out /app/ui

COPY deploy/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV API_PORT=80
ENV NGINX_CONF_GEN_ENABLED=1
ENV DOCKER_API_VERSION=1.49

EXPOSE 80 6060

ENTRYPOINT ["dumb-init", "--", "/entrypoint.sh"]
