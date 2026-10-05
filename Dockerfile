# syntax=docker/dockerfile:1

# ---------- 构建阶段 ----------
FROM golang:1.27.1 AS build
WORKDIR /src

# 纯 Go 编译，产出静态二进制；GOTOOLCHAIN=local 禁止容器内再联网下载工具链
ENV CGO_ENABLED=0 GOTOOLCHAIN=local

# 使用 vendor 目录离线构建，构建期无需访问任何模块代理
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -mod=vendor -trimpath -ldflags="-s -w" -o /out/starlang-bridge .

# ---------- 运行阶段 ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget

ENV TZ=Asia/Shanghai
WORKDIR /app

COPY --from=build /out/starlang-bridge /app/starlang-bridge
COPY config/scenarios.json /app/config/scenarios.json

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/starlang-bridge"]
