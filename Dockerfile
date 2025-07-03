FROM --platform=${BUILDPLATFORM} golang:1.23-alpine AS builder

# 添加構建參數
ARG TARGETOS
ARG TARGETARCH


WORKDIR /app

# 複製模組文件
COPY go.mod go.sum ./

# 下載依賴
RUN go mod download

# 複製源代碼
COPY . .

# 構建應用程序
ARG CI_COMMIT_SHA
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -gcflags="all=-N -l"  \
    -ldflags "-X main.Version=$CI_COMMIT_SHA"  \
    -o fat-identity-cat

# Install Delve debugger 並指定安裝位置
RUN GOBIN=/usr/local/bin go install github.com/go-delve/delve/cmd/dlv@latest

# 創建最終運行時映像
FROM alpine:latest

# 安裝必要的運行時依賴
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    busybox-extras \
    libc6-compat

WORKDIR /app

# 從構建階段複製編譯後的應用程序
COPY --from=builder /app/fat-identity-cat .
# 添加這行來複製 dlv
COPY --from=builder /usr/local/bin/dlv /usr/local/bin/dlv

# 複製配置文件
COPY --from=builder /app/.env* ./

# 設置時區
ENV TZ=Asia/Taipei