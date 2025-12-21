# 构建阶段
FROM golang:1.24.11-alpine AS builder

LABEL authors="daheige"

# 设置golang环境变量和禁用CGO,开启go mod机制
ENV GO111MODULE=on CGO_ENABLED=0 \
    GOPROXY=https://goproxy.cn,direct

# 安装必要的构建工具
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum文件
COPY go.mod ./
COPY go.sum ./
COPY . ./

# 下载依赖
RUN go mod download && go mod verify

# 构建gRPC服务
RUN cd /app/cmd/rpc && go build -ldflags='-w -s' -o /app/main

# 运行阶段
FROM alpine:latest
#FROM alpine:3.21

#解决docker时区问题和中文乱码问题
ENV TZ=Asia/Shanghai LANG="zh_CN.UTF-8"

# 设置工作目录
WORKDIR /app

EXPOSE 50051 8090

# 解决http x509证书问题，需要安装证书
RUN echo $GOPROXY && echo "export LC_ALL=$LANG"  >>  /etc/profile \
    && echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.23/main/" > /etc/apk/repositories \
    && apk update \
    && apk upgrade \
    && apk --no-cache add tzdata ca-certificates bash vim bash-doc bash-completion curl \
    && ln -snf  /usr/share/zoneinfo/$TZ /etc/localtime \
    && echo $TZ > /etc/timezone \
    && rm -rf /var/cache/apk/* /tmp/* /var/tmp/* $HOME/.cache \
    && mkdir -p /app/bin

# 将构建好的二进制copy到/app目录中
COPY --from=builder /app/main /app/main
COPY bin/entrypoint.sh /app/bin/entrypoin.sh
RUN chmod +x /app/bin/entrypoin.sh

# 启动容器运行服务
ENTRYPOINT ["/app/bin/entrypoin.sh"]
