package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/iWorld-y/domain_radar/api/proto/display/v1"
	"github.com/iWorld-y/domain_radar/app/display/internal/biz"
)

type DisplayService struct {
	v1.UnimplementedDisplayServer
	ucUser   *biz.UserUseCase
	ucReport *biz.ReportUseCase
	log      *log.Helper
}

func NewDisplayService(ucUser *biz.UserUseCase, ucReport *biz.ReportUseCase, logger log.Logger) *DisplayService {
	return &DisplayService{
		ucUser:   ucUser,
		ucReport: ucReport,
		log:      log.NewHelper(logger),
	}
}

func (s *DisplayService) Register(ctx context.Context, req *v1.RegisterReq) (*v1.RegisterReply, error) {
	err := s.ucUser.Register(ctx, req.Username, req.Password)
	if err != nil {
		return &v1.RegisterReply{Success: false, Message: err.Error()}, nil
	}
	return &v1.RegisterReply{Success: true, Message: "success"}, nil
}

func (s *DisplayService) Login(ctx context.Context, req *v1.LoginReq) (*v1.LoginReply, error) {
	token, err := s.ucUser.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	return &v1.LoginReply{Token: token, Username: req.Username}, nil
}

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

func (s *DisplayService) GetReport(ctx context.Context, req *v1.GetReportReq) (*v1.GetReportReply, error) {
	r, err := s.ucReport.GetByID(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}

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

	if r.DeepAnalysis != nil {
		reply.MacroTrends = r.DeepAnalysis.MacroTrends
		reply.Opportunities = r.DeepAnalysis.Opportunities
		reply.Risks = r.DeepAnalysis.Risks
		reply.ActionGuides = r.DeepAnalysis.ActionGuides
	}

	return reply, nil
}
