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
	// Name is the name of the compiled software.
	Name string = "display"
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "app/display/configs/config.yaml", "config path, eg: -conf config.yaml")
}

func main() {
	flag.Parse()
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
	)

	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	// Manual Dependency Injection
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

	httpSrv := server.NewHTTPServer(bc.Server, bc.Auth, displayService, logger)

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
