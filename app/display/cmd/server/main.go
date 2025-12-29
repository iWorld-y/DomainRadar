package main

import (
	"flag"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/iWorld-y/domain_radar/app/display/internal/conf"
	"github.com/iWorld-y/domain_radar/app/display/internal/data"
	"github.com/iWorld-y/domain_radar/app/display/internal/server"
	"github.com/iWorld-y/domain_radar/app/display/internal/service"
	"github.com/iWorld-y/domain_radar/app/display/internal/usecase"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name 是服务的名称
	Name string = "display"
	// Version 是服务的版本号
	Version string
	// flagconf 是配置文件的路径命令行参数
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	// 初始化命令行参数，默认指向 display 项目的配置文件
	flag.StringVar(&flagconf, "conf", "app/display/configs/config.yaml", "config path, eg: -conf config.yaml")
}

func main() {
	flag.Parse()
	// 初始化日志记录器，包含时间戳、调用者信息、服务ID等上下文
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
	)

	// 初始化配置加载器
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	// 扫描配置到 Bootstrap 结构体
	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	// 手动进行依赖注入 (未来可考虑使用 google/wire)
	d, cleanup, err := data.NewData(bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	userRepo := data.NewUserRepo(d, logger)
	reportRepo := data.NewReportRepo(d, logger)

	userUseCase := usecase.NewUserUseCase(userRepo, bc.Auth, logger)
	reportUseCase := usecase.NewReportUseCase(reportRepo, logger)

	displayService := service.NewDisplayService(userUseCase, reportUseCase, logger, bc.Data)

	// 初始化 HTTP 服务器
	httpSrv := server.NewHTTPServer(bc.Server, bc.Auth, displayService, logger)

	// 创建并运行 Kratos 应用
	app := kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			httpSrv,
		),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
