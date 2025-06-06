FROM golang:1.23-alpine AS builder

WORKDIR /app

# 複製模組文件
COPY go.mod go.sum ./

# 下載依賴
RUN go mod download

# 複製源代碼
COPY . .

# 構建應用程序
ARG CI_COMMIT_SHA
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.Version=$CI_COMMIT_SHA" -o fat-identity-cat

# 創建最終運行時映像
FROM alpine:latest

# 安裝必要的運行時依賴
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 從構建階段複製編譯後的應用程序
COPY --from=builder /app/fat-identity-cat .

# 複製配置文件
COPY --from=builder /app/.env* ./

# 設置時區
ENV TZ=Asia/Taipei