FROM golang:1.24-alpine

# 设置工作目录
WORKDIR /app

# 设置 GOPROXY 以解决下载超时问题
ENV GOPROXY=https://goproxy.cn,direct

# 允许自动下载所需的Go工具链版本
ENV GOTOOLCHAIN=auto

# 复制 go.mod 和 go.sum 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN go build -o bangumipikpak .

# 设置时区为亚洲/上海，确保定时任务按照正确的时区执行
RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone



# 运行应用
CMD ["./bangumipikpak"]