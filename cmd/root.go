package cmd

import (
	"fmt"
	"os"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/logger"
	"github.com/spf13/cobra"
)

var (
	cfgFile       string
	cfg           *config.Config
	serviceLogger infraport.Logger
)

// rootCmd 表示基礎命令，沒有調用其他命令時運行
var rootCmd = &cobra.Command{
	Use:   "ms-identity-cat",
	Short: "Identity Service for Merchant, Player, and Manager management",
	Long: `Identity Service is a microservice for managing Merchant, Player, 
and Manager identities with synchronization capabilities through KDS.`,
}

// Execute 添加所有子命令到根命令並設置標誌。
// 這會被 main.main() 調用。只需調用一次。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .env)")
}

// initConfig 讀取配置文件和環境變量
func initConfig() {
	var err error

	// 初始化配置
	cfg, err = config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日誌
	serviceLogger = initLogger(cfg)
}

// initLogger 初始化日誌
func initLogger(cfg *config.Config) infraport.Logger {
	return logger.NewServiceLogger(cfg)
}

// GetConfig 獲取配置
func GetConfig() *config.Config {
	return cfg
}

// GetLogger 獲取日誌
func GetLogger() infraport.Logger {
	return serviceLogger
}

// AddCommand 添加命令到根命令
func AddCommand(cmds ...*cobra.Command) {
	rootCmd.AddCommand(cmds...)
}
