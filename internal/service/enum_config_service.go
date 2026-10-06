package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/feedback/internal/model"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/repository"
)

type EnumConfigService struct {
	repo *repository.EnumConfigRepo
}

func NewEnumConfigService(repo *repository.EnumConfigRepo) *EnumConfigService {
	return &EnumConfigService{repo: repo}
}

type CreateEnumConfigInput struct {
	Field  string          `json:"field" binding:"required,max=64"`
	Labels json.RawMessage `json:"labels" binding:"required"`
}

type UpdateEnumConfigInput struct {
	Field  *string         `json:"field"`
	Labels json.RawMessage `json:"labels"`
}

func (s *EnumConfigService) List(ctx context.Context) ([]model.EnumConfig, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, apperr.Internal("list enum configs", err)
	}
	return rows, nil
}

func (s *EnumConfigService) Create(ctx context.Context, in CreateEnumConfigInput) (*model.EnumConfig, error) {
	existing, err := s.repo.FindByField(ctx, in.Field)
	if err != nil {
		return nil, apperr.Internal("check enum config field", err)
	}
	if existing != nil {
		return nil, apperr.InvalidParam("field already exists: " + in.Field)
	}
	now := time.Now()
	ec := &model.EnumConfig{
		Field:     in.Field,
		Labels:    model.JSON(in.Labels),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, ec); err != nil {
		return nil, apperr.Internal("create enum config", err)
	}
	return ec, nil
}

func (s *EnumConfigService) Update(ctx context.Context, id uint64, in UpdateEnumConfigInput) (*model.EnumConfig, error) {
	ec, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.Internal("load enum config", err)
	}
	if ec == nil {
		return nil, apperr.NotFound("enum config not found")
	}
	if in.Field != nil {
		existing, err := s.repo.FindByField(ctx, *in.Field)
		if err != nil {
			return nil, apperr.Internal("check enum config field", err)
		}
		if existing != nil && existing.ID != id {
			return nil, apperr.InvalidParam("field already exists: " + *in.Field)
		}
		ec.Field = *in.Field
	}
	if len(in.Labels) > 0 {
		ec.Labels = model.JSON(in.Labels)
	}
	ec.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, ec); err != nil {
		return nil, apperr.Internal("update enum config", err)
	}
	return ec, nil
}

func (s *EnumConfigService) Delete(ctx context.Context, id uint64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperr.Internal("delete enum config", err)
	}
	return nil
}

// AggregatedMap returns all enum configs as a single lookup map[field][value]→label.
func (s *EnumConfigService) AggregatedMap(ctx context.Context) (map[string]map[string]string, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, apperr.Internal("list enum configs for aggregated map", err)
	}
	merged := make(map[string]map[string]string)
	for _, ec := range rows {
		var labels map[string]any
		if err := json.Unmarshal(ec.Labels, &labels); err != nil {
			continue
		}
		if merged[ec.Field] == nil {
			merged[ec.Field] = make(map[string]string)
		}
		for k, v := range labels {
			if s, ok := v.(string); ok {
				merged[ec.Field][k] = s
			}
		}
	}
	return merged, nil
}