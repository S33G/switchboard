FROM golang:1.25.4-alpine AS builder

WORKDIR /src/backend

# Copy dependency files first for better caching
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source code only after dependencies are cached
COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/switchboard .


FROM node:22-alpine AS ui-builder

WORKDIR /src/ui

RUN corepack enable

# Copy dependency files first for better caching
COPY ui/package.json ui/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Copy source code only after dependencies are cached
COPY ui/ ./
RUN pnpm build


FROM nginx:1.27-alpine

# OCI labels to link package to repository
LABEL org.opencontainers.image.source=https://github.com/S33G/switchboard
LABEL org.opencontainers.image.description="Central nginx reverse proxy for all your Docker containers across multiple hosts"
LABEL org.opencontainers.image.licenses=MIT

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

RUN rm -f /etc/nginx/conf.d/default.conf
COPY deploy/nginx/00-switchboard.conf /etc/nginx/conf.d/00-switchboard.conf

COPY deploy/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV API_PORT=8069
ENV NGINX_CONF_GEN_ENABLED=1
ENV DOCKER_API_VERSION=1.49

EXPOSE 80 443 8069

ENTRYPOINT ["dumb-init", "--", "/entrypoint.sh"]
