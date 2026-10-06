package service

import (
	"context"

	"github.com/feedback/internal/model"
	"github.com/feedback/internal/repository"
)

type ClassificationLogService struct {
	repo            *repository.ClassificationLogRepo
	llmProviderRepo *repository.LLMProviderRepo
}

func NewClassificationLogService(repo *repository.ClassificationLogRepo, llmProviderRepo *repository.LLMProviderRepo) *ClassificationLogService {
	return &ClassificationLogService{repo: repo, llmProviderRepo: llmProviderRepo}
}

func (s *ClassificationLogService) ListByFeedbackID(ctx context.Context, feedbackID uint64) ([]model.ClassificationLog, error) {
	logs, err := s.repo.ListByFeedbackID(ctx, feedbackID)
	if err != nil {
		return nil, err
	}

	providerIDs := make(map[uint64]bool)
	for _, l := range logs {
		if l.ProviderID != nil {
			providerIDs[*l.ProviderID] = true
		}
	}

	if len(providerIDs) > 0 {
		providers, _ := s.llmProviderRepo.List(ctx)
		nameMap := make(map[uint64]string, len(providers))
		for _, p := range providers {
			nameMap[p.ID] = p.Name
		}
		result := make([]model.ClassificationLog, len(logs))
		for i, l := range logs {
			result[i] = l
			if l.ProviderID != nil {
				if name, ok := nameMap[*l.ProviderID]; ok {
					result[i].ProviderName = name
				}
			}
		}
		return result, nil
	}

	return logs, nil
}

func (s *ClassificationLogService) Create(ctx context.Context, log *model.ClassificationLog) error {
	return s.repo.Create(ctx, log)
}

func (s *ClassificationLogService) NextAttempt(ctx context.Context, feedbackID uint64) (int, error) {
	count, err := s.repo.CountByFeedbackID(ctx, feedbackID)
	if err != nil {
		return 1, err
	}
	return count + 1, nil
}