package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof" // 自動註冊 pprof 處理器
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-identity-cat/cmd"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/middleware"
	"github.com/jvdiamondtech/ms-identity-cat/internal/di"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/database/mysql"
	sLog "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/logger"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	port int
)

// Command 創建並返回web子命令
func Command() *cobra.Command {
	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Start the web server",
		Long:  `Start the web server to handle HTTP API requests`,
		Run:   runWebServer,
	}

	webCmd.Flags().IntVarP(&port, "port", "p", 0, "server port (default is from config)")

	return webCmd
}

func init() {
	cmd.AddCommand(Command())
}

// runWebServer 啟動Web服務
func runWebServer(cobraCmd *cobra.Command, args []string) {
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()
	// 獲取ServiceLogger實例
	sLogger := sLog.NewServiceLogger(cfg)
	// 主程序的Context
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// 初始化追踪器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		sLogger.FatalLog("Failed to initialize web tracer", sLogger.Error("err", err))
	}
	defer tracer.Shutdown(context.Background())
	sLogger.InfoLog("Successfully initialized web tracer!")

	ctx, rootSpan := tracing.StartSpan(rootCtx, "WebService")
	defer rootSpan.End()

	// 初始化DB連線
	db, err := mysql.NewDatabase(cfg)
	if err != nil {
		sLogger.FatalWithContext(rootCtx, "[Fatal][Web][runWebServer] Failed to initialize database", sLogger.Error("err", err))
	}
	defer func() {
		err = db.Close() // 主程序結束後關閉DB連線
		if err != nil {
			sLogger.ErrorLog("Failed to close database connection", sLogger.Error("err", err))
		}
		sLogger.InfoWithContext(rootCtx, "[Info][Web][runWebServer] Database connection closed successfully")
	}()
	sLogger.InfoLog("Successfully initialized DB connection!")

	if err = db.Ping(); err != nil {
		sLogger.FatalLog("Failed to ping database", sLogger.Error("err", err))
	}

	// 初始化Redis連線
	redisManager := redis.NewRedisManager(cfg)
	defer func() {
		err = redisManager.Close() // 主程序結束後關閉Redis連線
		if err != nil {
			sLogger.ErrorLog("Failed to close Redis connection", sLogger.Error("err", err))
		}
		sLogger.InfoWithContext(rootCtx, "[Info][Web][runWebServer] Redis connection closed successfully")
	}()
	if err = redisManager.Connect(rootCtx); err != nil {
		sLogger.FatalLog("Failed to connect to Redis after retry", sLogger.Error("err", err))
	}
	sLogger.InfoLog("Successfully initialized Redis connection!")

	// 使用Wire初始化HTTP處理器
	httpHandler, err := di.InitializeWebServer(cfg, logger, sLogger, redisManager, db.GetDBConnection())
	if err != nil {
		sLogger.FatalLog("Failed to initialize web server",
			sLogger.Error("err", err),
			sLogger.String("DB_HOST", viper.GetString("DB_HOST")),
			sLogger.String("DB_USER", viper.GetString("DB_USER")),
			sLogger.String("DB_PASSWORD", viper.GetString("DB_PASSWORD")))
	}

	if port == 0 {
		port = 8080 // 默認端口
	} else {
		port = cfg.App.Port
	}

	// 設置 Gin 模式
	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// 創建 Gin 路由
	router := gin.Default()

	// 添加追踪中間件
	router.Use(middleware.TracingMiddleware())

	// 註冊路由
	httpHandler.RegisterRoutes(router)

	// 創建HTTP服務器
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	// 啟動調試服務器，用於性能分析和調試
	go func() {
		// 創建調試服務器的路由
		debugMux := http.NewServeMux()

		// 添加一個簡單的健康檢查端點
		debugMux.HandleFunc("/debug/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// 添加一個顯示運行時信息的端點
		debugMux.HandleFunc("/debug/info", func(w http.ResponseWriter, r *http.Request) {
			info := map[string]interface{}{
				"go_version": runtime.Version(),
				"goroutines": runtime.NumGoroutine(),
				"cpus":       runtime.NumCPU(),
				"time":       time.Now().Format(time.RFC3339),
			}
			json.NewEncoder(w).Encode(info)
		})

		serverDebug := &http.Server{
			Addr:    fmt.Sprintf(":%d", 6060),
			Handler: debugMux,
		}

		sLogger.InfoLog("Starting debug server on port 6060")
		if err := serverDebug.ListenAndServe(); err != nil {
			sLogger.ErrorLog("Failed to start debug server", sLogger.Error("err", err))
		}
	}()
	// 在後台運行服務器
	go func() {
		sLogger.InfoLog("Starting web server", sLogger.Int("port", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sLogger.FatalLog("Failed to start server", sLogger.Error("err", err))
		}
	}()
	// 等待中斷信號優雅地關閉服務器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	sLogger.InfoLog("Shutting down web server...")

	// 記錄關閉事件
	tracing.TraceEvent(rootSpan, "Shutting down web server")

	// 創建上下文用於通知服務器關閉
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		sLogger.FatalLog("Server forced to shutdown", sLogger.Error("err", err))
	}

	// 記錄成功關閉
	tracing.TraceEvent(rootSpan, "Web server exited gracefully")

	sLogger.InfoLog("Web server exited")
}
