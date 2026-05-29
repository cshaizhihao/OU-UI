FROM node:22-alpine AS web-builder
WORKDIR /src
RUN corepack enable && corepack prepare pnpm@10.25.0 --activate
COPY package.json pnpm-workspace.yaml tsconfig.base.json ./
COPY apps/web/package.json apps/web/package.json
RUN pnpm install --no-frozen-lockfile
COPY apps/web apps/web
RUN pnpm --filter @ou-ui/web build

FROM golang:1.24-alpine AS server-builder
WORKDIR /src
RUN apk add --no-cache gcc musl-dev
COPY go.mod ./
COPY apps/server apps/server
COPY apps/agent apps/agent
COPY internal internal
RUN go mod tidy
RUN CGO_ENABLED=1 GOOS=linux go build -o /out/ou-ui-server ./apps/server
RUN CGO_ENABLED=1 GOOS=linux go build -o /out/ou-ui-agent ./apps/agent

FROM nginx:1.27-alpine AS web
COPY --from=web-builder /src/apps/web/dist /usr/share/nginx/html
COPY deploy/nginx/entrypoint.sh /docker-entrypoint.d/90-ou-ui-config.sh
RUN chmod +x /docker-entrypoint.d/90-ou-ui-config.sh
EXPOSE 3000

FROM alpine:3.20 AS server
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=server-builder /out/ou-ui-server /app/ou-ui-server
COPY --from=server-builder /out/ou-ui-agent /app/ou-ui-agent
EXPOSE 8080
CMD ["/app/ou-ui-server"]
