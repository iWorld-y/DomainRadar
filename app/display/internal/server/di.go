package server

import (
	"github.com/google/wire"
	"github.com/iWorld-y/domain_radar/app/display/internal/data"
	"github.com/iWorld-y/domain_radar/app/display/internal/service"
	"github.com/iWorld-y/domain_radar/app/display/internal/usecase"
)

// ProviderSet 是展示服务的依赖注入 Provider 集合
var ProviderSet = wire.NewSet(
	// Server providers
	NewHTTPServer,

	// Data providers
	data.NewData,
	data.NewUserRepo,
	data.NewReportRepo,

	// UseCase providers
	usecase.NewUserUseCase,
	usecase.NewReportUseCase,

	// Service providers
	service.NewDisplayService,
)
