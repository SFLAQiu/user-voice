package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/feedback/internal/classifier"
	"github.com/feedback/internal/model"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/repository"
)

// ActiveClassifierConfig holds the resolved configuration used by the classifier.
type ActiveClassifierConfig struct {
	PromptTemplate    string
	Categories        []string
	BusinessModules   []classifier.ModuleItem
	PromptConfigID    uint64
}

type ClassifierConfigService struct {
	repo *repository.ClassifierConfigRepo
}

func NewClassifierConfigService(repo *repository.ClassifierConfigRepo) *ClassifierConfigService {
	return &ClassifierConfigService{repo: repo}
}

type CreateClassifierConfigInput struct {
	Type      string          `json:"type" binding:"required,max=32"`
	Name      string          `json:"name" binding:"required,max=128"`
	Content   string          `json:"content"`
	SortOrder int             `json:"sort_order"`
	Enabled   *int            `json:"enabled"`
	IsDefault *int            `json:"is_default"`
}

type UpdateClassifierConfigInput struct {
	Name      *string         `json:"name"`
	Content   *string         `json:"content"`
	SortOrder *int            `json:"sort_order"`
	Enabled   *int            `json:"enabled"`
	IsDefault *int            `json:"is_default"`
}

func (s *ClassifierConfigService) List(ctx context.Context) ([]model.ClassifierConfig, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, apperr.Internal("list classifier configs", err)
	}
	return rows, nil
}

func (s *ClassifierConfigService) Create(ctx context.Context, in CreateClassifierConfigInput) (*model.ClassifierConfig, error) {
	existing, err := s.repo.FindByTypeAndName(ctx, in.Type, in.Name)
	if err != nil {
		return nil, apperr.Internal("check classifier config", err)
	}
	if existing != nil {
		return nil, apperr.InvalidParam("config already exists: " + in.Type + "/" + in.Name)
	}
	enabled := 1
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	isDefault := 0
	if in.IsDefault != nil {
		isDefault = *in.IsDefault
	}
	c := &model.ClassifierConfig{
		Type:      in.Type,
		Name:      in.Name,
		Content:   in.Content,
		SortOrder: in.SortOrder,
		Enabled:   enabled,
		IsDefault: isDefault,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, apperr.Internal("create classifier config", err)
	}
	return c, nil
}

func (s *ClassifierConfigService) Update(ctx context.Context, id uint64, in UpdateClassifierConfigInput) (*model.ClassifierConfig, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.Internal("load classifier config", err)
	}
	if c == nil {
		return nil, apperr.NotFound("classifier config not found")
	}
	if in.Name != nil {
		existing, err := s.repo.FindByTypeAndName(ctx, c.Type, *in.Name)
		if err != nil {
			return nil, apperr.Internal("check classifier config name", err)
		}
		if existing != nil && existing.ID != id {
			return nil, apperr.InvalidParam("config already exists: " + c.Type + "/" + *in.Name)
		}
		c.Name = *in.Name
	}
	if in.Content != nil {
		c.Content = *in.Content
	}
	if in.SortOrder != nil {
		c.SortOrder = *in.SortOrder
	}
	if in.Enabled != nil {
		c.Enabled = *in.Enabled
	}
	if in.IsDefault != nil {
		c.IsDefault = *in.IsDefault
	}
	c.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, apperr.Internal("update classifier config", err)
	}
	return c, nil
}

func (s *ClassifierConfigService) Delete(ctx context.Context, id uint64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperr.Internal("delete classifier config", err)
	}
	return nil
}

// GetActiveConfig returns the currently effective classification configuration.
func (s *ClassifierConfigService) GetActiveConfig(ctx context.Context) (*ActiveClassifierConfig, error) {
	promptCfg, err := s.repo.FindDefaultPrompt(ctx)
	if err != nil {
		return nil, apperr.Internal("load default prompt", err)
	}
	template := classifier.FallbackPromptTemplate
	promptConfigID := uint64(0)
	if promptCfg != nil {
		template = promptCfg.Content
		promptConfigID = promptCfg.ID
	}

	cats, err := s.repo.ListByType(ctx, model.ClassifierConfigTypeCategory)
	if err != nil {
		return nil, apperr.Internal("load categories", err)
	}
	catNames := make([]string, 0, len(cats))
	for _, c := range cats {
		catNames = append(catNames, c.Name)
	}

	modules, err := s.repo.ListByType(ctx, model.ClassifierConfigTypeBusinessModule)
	if err != nil {
		return nil, apperr.Internal("load business modules", err)
	}
	modItems := make([]classifier.ModuleItem, 0, len(modules))
	for _, m := range modules {
		modItems = append(modItems, classifier.ModuleItem{Name: m.Name, Description: m.Content})
	}

	return &ActiveClassifierConfig{
		PromptTemplate:  template,
		Categories:      catNames,
		BusinessModules: modItems,
		PromptConfigID:  promptConfigID,
	}, nil
}

// AggregatedEnums returns categories and business modules as enum maps for the frontend.
func (s *ClassifierConfigService) AggregatedEnums(ctx context.Context) (map[string][]string, error) {
	cats, err := s.repo.ListByType(ctx, model.ClassifierConfigTypeCategory)
	if err != nil {
		return nil, apperr.Internal("load categories", err)
	}
	modules, err := s.repo.ListByType(ctx, model.ClassifierConfigTypeBusinessModule)
	if err != nil {
		return nil, apperr.Internal("load business modules", err)
	}

	catNames := make([]string, 0, len(cats))
	for _, c := range cats {
		catNames = append(catNames, c.Name)
	}
	modNames := make([]string, 0, len(modules))
	for _, m := range modules {
		modNames = append(modNames, m.Name)
	}

	return map[string][]string{
		"categories":       catNames,
		"business_modules": modNames,
	}, nil
}

// BuildClassificationPrompt builds the full prompt from the active config and feedback content.
func (s *ClassifierConfigService) BuildClassificationPrompt(ctx context.Context, content string) (string, *ActiveClassifierConfig, error) {
	cfg, err := s.GetActiveConfig(ctx)
	if err != nil {
		return "", nil, err
	}
	prompt := classifier.BuildPrompt(
		cfg.PromptTemplate,
		classifier.FormatCategories(cfg.Categories),
		classifier.FormatBusinessModules(cfg.BusinessModules),
		content,
	)
	return prompt, cfg, nil
}

// marshalParsedResult serializes a classifier Result to JSON bytes for storage.
func marshalParsedResult(r classifier.Result) (model.JSON, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return model.JSON(b), nil
}