package service

import (
	"context"
	"sync"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	v1 "github.com/iWorld-y/domain_radar/api/proto/display/v1"
	"github.com/iWorld-y/domain_radar/app/display/internal/usecase"
	"github.com/iWorld-y/domain_radar/app/domain_radar/pkg/engine"
)

// TaskStatus 表示后台任务的状态
type TaskStatus struct {
	Status   string // "pending", "running", "completed", "failed"
	Progress int    // 进度 (0-100)
	Message  string // 状态信息或错误详情
}

// DisplayService 实现了展示服务的 API 接口
type DisplayService struct {
	v1.UnimplementedDisplayServer
	ucUser   *usecase.UserUseCase
	ucReport *usecase.ReportUseCase
	log      *log.Helper

	// 任务管理相关
	tasks  sync.Map // map[string]*TaskStatus
	engine *engine.Engine
}

// NewDisplayService 创建展示服务实例
func NewDisplayService(
	ucUser *usecase.UserUseCase,
	ucReport *usecase.ReportUseCase,
	logger log.Logger,
	eng *engine.Engine,
) *DisplayService {
	return &DisplayService{
		ucUser:   ucUser,
		ucReport: ucReport,
		log:      log.NewHelper(logger),
		engine:   eng,
	}
}

// Register 用户注册
func (s *DisplayService) Register(ctx context.Context, req *v1.RegisterReq) (*v1.RegisterReply, error) {
	err := s.ucUser.Register(ctx, req.Username, req.Password)
	if err != nil {
		return &v1.RegisterReply{Success: false, Message: err.Error()}, nil
	}
	return &v1.RegisterReply{Success: true, Message: "success"}, nil
}

// Login 用户登录并返回 JWT 令牌
func (s *DisplayService) Login(ctx context.Context, req *v1.LoginReq) (*v1.LoginReply, error) {
	token, err := s.ucUser.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	return &v1.LoginReply{Token: token, Username: req.Username}, nil
}

// ListReports 分页获取报告列表摘要
func (s *DisplayService) ListReports(ctx context.Context, req *v1.ListReportsReq) (*v1.ListReportsReply, error) {
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 10
	}

	summaries, total, err := s.ucReport.List(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	list := make([]*v1.ReportSummary, 0, len(summaries))
	for _, s := range summaries {
		list = append(list, &v1.ReportSummary{
			Id:           int32(s.ID),
			Title:        s.Title,
			Date:         s.Date,
			DomainCount:  int32(s.DomainCount),
			AverageScore: int32(s.AverageScore),
		})
	}

	return &v1.ListReportsReply{
		Reports: list,
		Total:   int32(total),
	}, nil
}

// GetReport 获取指定 ID 的报告详情
func (s *DisplayService) GetReport(ctx context.Context, req *v1.GetReportReq) (*v1.GetReportReply, error) {
	// 从 JWT 上下文中提取用户信息
	claims, ok := jwt.FromContext(ctx)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "missing jwt token")
	}
	mapClaims, ok := claims.(jwtv5.MapClaims)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "invalid jwt token")
	}
	username, ok := mapClaims["username"].(string)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "invalid username in token")
	}

	u, err := s.ucUser.GetProfile(ctx, username)
	if err != nil {
		return nil, err
	}

	r, err := s.ucReport.GetByID(ctx, int(req.Id), u.ID)
	if err != nil {
		return nil, err
	}

	// 转换领域报告详情
	domains := make([]*v1.DomainReport, 0, len(r.Domains))
	for _, d := range r.Domains {
		articles := make([]*v1.Article, 0, len(d.Articles))
		for _, a := range d.Articles {
			articles = append(articles, &v1.Article{
				Title:   a.Title,
				Link:    a.Link,
				Source:  a.Source,
				PubDate: a.PubDate,
			})
		}
		domains = append(domains, &v1.DomainReport{
			Id:         int32(d.ID),
			DomainName: d.DomainName,
			Overview:   d.Overview,
			Trends:     d.Trends,
			Score:      int32(d.Score),
			KeyEvents:  d.KeyEvents,
			Articles:   articles,
		})
	}

	reply := &v1.GetReportReply{
		Id:      int32(r.ID),
		Date:    r.Date,
		Domains: domains,
	}

	// 如果存在深度分析，则填充深度分析数据
	if r.DeepAnalysis != nil {
		reply.DeepAnalysis = &v1.DeepAnalysis{
			MacroTrends:   r.DeepAnalysis.MacroTrends,
			Opportunities: r.DeepAnalysis.Opportunities,
			Risks:         r.DeepAnalysis.Risks,
			ActionGuides:  r.DeepAnalysis.ActionGuides,
		}
	}

	return reply, nil
}

// GetProfile 获取当前登录用户的个人资料
func (s *DisplayService) GetProfile(ctx context.Context, req *v1.GetProfileReq) (*v1.GetProfileReply, error) {
	claims, ok := jwt.FromContext(ctx)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "missing jwt token")
	}
	mapClaims, ok := claims.(jwtv5.MapClaims)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "invalid jwt token")
	}
	username, ok := mapClaims["username"].(string)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "invalid username in token")
	}

	u, err := s.ucUser.GetProfile(ctx, username)
	if err != nil {
		return nil, err
	}
	return &v1.GetProfileReply{Username: u.Username, Persona: u.Persona, Domains: u.Domains}, nil
}

// UpdateProfile 更新用户个人资料（如关注领域、用户画像）
func (s *DisplayService) UpdateProfile(ctx context.Context, req *v1.UpdateProfileReq) (*v1.UpdateProfileReply, error) {
	claims, ok := jwt.FromContext(ctx)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "missing jwt token")
	}
	mapClaims, ok := claims.(jwtv5.MapClaims)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "invalid jwt token")
	}
	username, ok := mapClaims["username"].(string)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "invalid username in token")
	}

	s.log.Infof("UpdateProfile: username=%s, domains=%v", username, req.Domains)

	err := s.ucUser.UpdateProfile(ctx, username, req.Persona, req.Domains)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateProfileReply{Success: true}, nil
}

// TriggerReport 异步触发一次领域雷达报告生成任务
func (s *DisplayService) TriggerReport(ctx context.Context, req *v1.TriggerReportReq) (*v1.TriggerReportReply, error) {
	if s.engine == nil {
		return nil, errors.InternalServer("ENGINE_NOT_INIT", "domain radar engine not initialized")
	}

	claims, ok := jwt.FromContext(ctx)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "missing jwt token")
	}
	mapClaims, ok := claims.(jwtv5.MapClaims)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "invalid jwt token")
	}
	username, ok := mapClaims["username"].(string)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "invalid username in token")
	}

	u, err := s.ucUser.GetProfile(ctx, username)
	if err != nil {
		return nil, err
	}

	s.log.Infof("TriggerReport: username=%s, domains=%v, len=%d", username, u.Domains, len(u.Domains))

	if len(u.Domains) == 0 {
		return nil, errors.BadRequest("NO_DOMAINS", "please configure interested domains in profile first")
	}

	taskID := uuid.New().String()
	s.tasks.Store(taskID, &TaskStatus{Status: "pending", Progress: 0, Message: "Initializing..."})

	// 在后台协程中执行耗时的分析任务
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Errorf("Recovered from panic: %v", r)
				s.tasks.Store(taskID, &TaskStatus{Status: "failed", Progress: 100, Message: "Internal Panic"})
			}
		}()

		s.tasks.Store(taskID, &TaskStatus{Status: "running", Progress: 5, Message: "Starting..."})

		// 调用领域雷达引擎开始执行
		err := s.engine.Run(context.Background(), engine.RunOptions{
			UserID:  u.ID,
			Domains: u.Domains,
			Persona: u.Persona,
			ProgressCallback: func(status string, progress int) {
				s.tasks.Store(taskID, &TaskStatus{Status: "running", Progress: progress, Message: status})
			},
		})

		if err != nil {
			s.tasks.Store(taskID, &TaskStatus{Status: "failed", Progress: 100, Message: err.Error()})
		} else {
			s.tasks.Store(taskID, &TaskStatus{Status: "completed", Progress: 100, Message: "Completed"})
		}
	}()

	return &v1.TriggerReportReply{TaskId: taskID, Message: "Task started"}, nil
}

// GetTaskStatus 查询后台任务的执行进度和状态
func (s *DisplayService) GetTaskStatus(ctx context.Context, req *v1.GetTaskStatusReq) (*v1.GetTaskStatusReply, error) {
	val, ok := s.tasks.Load(req.TaskId)
	if !ok {
		return nil, errors.NotFound("TASK_NOT_FOUND", "task not found")
	}
	status := val.(*TaskStatus)
	return &v1.GetTaskStatusReply{
		Status:   status.Status,
		Progress: int32(status.Progress),
		Message:  status.Message,
	}, nil
}
