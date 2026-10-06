package service

import (
	"context"
	"strings"
	"time"

	"github.com/feedback/internal/model"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/repository"
)

type FeedbackService struct {
	repo      *repository.FeedbackRepo
	bucketSvc *MetricBucketService
}

func NewFeedbackService(repo *repository.FeedbackRepo, bucketSvc *MetricBucketService) *FeedbackService {
	return &FeedbackService{repo: repo, bucketSvc: bucketSvc}
}

type ListFeedbackInput struct {
	StartTime      string
	EndTime        string
	AppID          string
	Platform       string
	PlatformID     string
	Category       string
	BusinessModule string
	Sentiment      string
	AppVersion     string
	UserMode       string
	UserID         string
	Keyword        string
	OrderBy        string
	OrderDir       string
	Page           int
	PageSize       int
}

func (s *FeedbackService) List(ctx context.Context, in ListFeedbackInput) ([]model.Feedback, int64, error) {
	q := repository.ListQuery{
		Sentiment:  in.Sentiment,
		AppVersion: in.AppVersion,
		Keyword:    in.Keyword,
		OrderBy:    in.OrderBy,
		OrderDir:   in.OrderDir,
		Page:       in.Page,
		PageSize:   in.PageSize,
	}
	if t, err := parseTime(in.StartTime); err == nil && t != nil {
		q.StartTime = t
	} else if err != nil {
		return nil, 0, apperr.InvalidParam("invalid start_time")
	}
	if t, err := parseTime(in.EndTime); err == nil && t != nil {
		q.EndTime = t
	} else if err != nil {
		return nil, 0, apperr.InvalidParam("invalid end_time")
	}
	if in.AppID != "" {
		n, err := parseIntStrict(in.AppID)
		if err != nil {
			return nil, 0, apperr.InvalidParam("invalid app_id")
		}
		q.AppID = &n
	}
	q.Platforms = splitNonEmpty(in.Platform)
	if in.PlatformID != "" {
		ids, err := parseIntList(in.PlatformID)
		if err != nil {
			return nil, 0, apperr.InvalidParam("invalid platform_id")
		}
		q.PlatformIDs = ids
	}
	q.Categories = splitNonEmpty(in.Category)
	q.BusinessModules = splitNonEmpty(in.BusinessModule)
	if in.UserMode != "" {
		ids, err := parseIntList(in.UserMode)
		if err != nil {
			return nil, 0, apperr.InvalidParam("invalid user_mode")
		}
		q.UserModes = ids
	}

	q.UserID = in.UserID
	rows, total, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, 0, apperr.Internal("list feedbacks", err)
	}
	for i := range rows {
		rows[i].QQ = maskQQ(rows[i].QQ)
	}
	return rows, total, nil
}

func (s *FeedbackService) Detail(ctx context.Context, id uint64) (*model.Feedback, error) {
	f, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.Internal("load feedback", err)
	}
	if f == nil {
		return nil, apperr.NotFound("feedback not found")
	}
	f.QQ = maskQQ(f.QQ)
	return f, nil
}

func (s *FeedbackService) UpdateCategory(ctx context.Context, id uint64, category, businessModule string) error {
	if category == "" {
		return apperr.InvalidParam("category required")
	}
	fb, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperr.Internal("load feedback", err)
	}
	if fb == nil {
		return apperr.NotFound("feedback not found")
	}
	oldDims := repository.BucketDimsFromFeedback(*fb)
	newStatus := model.CategoryStatusManual
	if err := s.repo.UpdateCategory(ctx, id, category, businessModule, model.CategoryStatusManual); err != nil {
		return apperr.Internal("update category", err)
	}
	if s.bucketSvc != nil {
		fb.Category = category
		fb.BusinessModule = businessModule
		fb.CategoryStatus = newStatus
		newDims := repository.BucketDimsFromFeedback(*fb)
		if oldDims != newDims {
			if err := s.bucketSvc.MoveFeedbackBucketWithSource(ctx, *fb, oldDims, "manual", "manual_classify_move", "手动变更分类"); err != nil {
				return apperr.Internal("update metric bucket", err)
			}
		}
	}
	return nil
}

// --- helpers ---

func parseTime(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, apperr.InvalidParam("bad time format")
}

func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func parseIntStrict(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, apperr.InvalidParam("empty int")
	}
	return atoiStrict(s)
}

func parseIntList(s string) ([]int, error) {
	parts := splitNonEmpty(s)
	if len(parts) == 0 {
		return nil, nil
	}
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := atoiStrict(p)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func atoiStrict(s string) (int, error) {
	n := 0
	neg := false
	i := 0
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		if s[0] == '-' {
			neg = true
		}
		i = 1
	}
	if i == len(s) {
		return 0, apperr.InvalidParam("not a number")
	}
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, apperr.InvalidParam("not a number")
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return n, nil
}

func maskQQ(qq string) string {
	if len(qq) <= 4 {
		return qq
	}
	return qq[:3] + "***" + qq[len(qq)-2:]
}
