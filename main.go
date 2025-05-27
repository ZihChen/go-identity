package main

import (
	"github.com/jvdiamondtech/ms-identity-cat/cmd"
	_ "github.com/jvdiamondtech/ms-identity-cat/cmd/consumer"
	_ "github.com/jvdiamondtech/ms-identity-cat/cmd/web"
	_ "github.com/jvdiamondtech/ms-identity-cat/cmd/worker"
	_ "github.com/jvdiamondtech/ms-identity-cat/docs" // 導入 swagger 文檔
)

// @title Identity Service API
// @version 1.0
// @description 用於管理商戶、玩家和管理員的身份服務
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.jvdiamondtech.com/support
// @contact.email support@jvdiamondtech.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @BasePath
func main() {
	cmd.Execute()
}
