# ---------- builder ----------
FROM golang:1.24.6-alpine AS build
WORKDIR /app
# 依赖
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download

# 拷贝源码并构建（与你的 mustLoadTemplates 配合：WORKDIR 就是 /app）
COPY . .
# 关闭 CGO，做静态二进制
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/bin/server ./cmd/server

# ---------- runtime ----------
FROM alpine:3.20
WORKDIR /app
# 运行用户 & 目录
RUN adduser -D -u 10001 appuser && \
    mkdir -p /data && chown -R appuser:appuser /data

# 放二进制 + 模板（**关键**：模板路径保持 /app/web/templates，与你的 mustLoadTemplates 对齐）
COPY --from=build /app/bin/server /app/server
COPY --from=build /app/web/templates /app/web/templates

ENV APP_ADDR=:8080 \
    DATA_DIR=/data
EXPOSE 8080
USER appuser
ENTRYPOINT ["/app/server"]
