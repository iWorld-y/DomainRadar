package service

import (
	"context"
	pb "github.com/iWorld-y/domain_radar/app/display/api/display/v1"
	"github.com/iWorld-y/domain_radar/app/display/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
)

type DisplayService struct {
	pb.UnimplementedDisplayServer
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

func (s *DisplayService) Register(ctx context.Context, req *pb.RegisterReq) (*pb.RegisterReply, error) {
	err := s.ucUser.Register(ctx, req.Username, req.Password)
	if err != nil {
		return &pb.RegisterReply{Success: false, Message: err.Error()}, nil
	}
	return &pb.RegisterReply{Success: true, Message: "success"}, nil
}

func (s *DisplayService) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginReply, error) {
	token, err := s.ucUser.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	return &pb.LoginReply{Token: token, Username: req.Username}, nil
}

func (s *DisplayService) ListReports(ctx context.Context, req *pb.ListReportsReq) (*pb.ListReportsReply, error) {
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 10
	}

	reports, total, err := s.ucReport.List(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	list := make([]*pb.ReportSummary, 0, len(reports))
	for _, r := range reports {
		list = append(list, &pb.ReportSummary{
			Id:         int32(r.ID),
			DomainName: r.DomainName,
			Score:      int32(r.Score),
			CreatedAt:  r.CreatedAt,
		})
	}

	return &pb.ListReportsReply{
		Reports: list,
		Total:   int32(total),
	}, nil
}

func (s *DisplayService) GetReport(ctx context.Context, req *pb.GetReportReq) (*pb.GetReportReply, error) {
	r, err := s.ucReport.Get(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}

	articles := make([]*pb.Article, 0, len(r.Articles))
	for _, a := range r.Articles {
		articles = append(articles, &pb.Article{
			Title:   a.Title,
			Link:    a.Link,
			Source:  a.Source,
			PubDate: a.PubDate,
		})
	}

	return &pb.GetReportReply{
		Id:         int32(r.ID),
		DomainName: r.DomainName,
		Overview:   r.Overview,
		Trends:     r.Trends,
		Score:      int32(r.Score),
		KeyEvents:  r.KeyEvents,
		Articles:   articles,
	}, nil
}
