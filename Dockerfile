# syntax=docker/dockerfile:1

# ---------- 构建阶段 ----------
# go.mod 要求 go 1.27.1；若基础镜像补丁版本较低，GOTOOLCHAIN=auto 会自动拉取对应工具链
FROM golang:1.27 AS build
WORKDIR /src

# 纯 Go 编译，产出静态二进制，运行阶段无需 libc
ENV CGO_ENABLED=0 GOTOOLCHAIN=auto

# 先拷贝依赖清单，利用缓存加速重复构建
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/starlang-bridge .

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
