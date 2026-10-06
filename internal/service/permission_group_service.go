package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/feedback/internal/model"
	"github.com/feedback/internal/repository"
)

type PermissionGroupService struct {
	repo *repository.PermissionGroupRepo
}

func NewPermissionGroupService(repo *repository.PermissionGroupRepo) *PermissionGroupService {
	return &PermissionGroupService{repo: repo}
}

// ListAll 返回所有权限组。
func (s *PermissionGroupService) ListAll(ctx context.Context) ([]model.PermissionGroup, error) {
	return s.repo.List(ctx)
}

// Get 返回单个权限组。
func (s *PermissionGroupService) Get(ctx context.Context, id uint64) (*model.PermissionGroup, error) {
	return s.repo.FindByID(ctx, id)
}

// Create 创建权限组，校验唯一性及权限格式。
func (s *PermissionGroupService) Create(ctx context.Context, name string, description string, permissions model.JSON) (*model.PermissionGroup, error) {
	if name == "" {
		return nil, fmt.Errorf("权限组名称不能为空")
	}
	existing, err := s.repo.FindByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("权限组名称 %q 已存在", name)
	}
	if err := validatePermissions(permissions); err != nil {
		return nil, err
	}
	g := &model.PermissionGroup{
		Name:        name,
		Description: description,
		Permissions: permissions,
	}
	if err := s.repo.Create(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

// Update 更新权限组。
func (s *PermissionGroupService) Update(ctx context.Context, id uint64, name string, description string, permissions model.JSON) (*model.PermissionGroup, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("权限组不存在")
	}
	if name == "" {
		return nil, fmt.Errorf("权限组名称不能为空")
	}
	// 检查名称唯一性（排除自身）
	dup, err := s.repo.FindByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if dup != nil && dup.ID != id {
		return nil, fmt.Errorf("权限组名称 %q 已存在", name)
	}
	if err := validatePermissions(permissions); err != nil {
		return nil, err
	}
	existing.Name = name
	existing.Description = description
	existing.Permissions = permissions
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// Delete 删除权限组。
func (s *PermissionGroupService) Delete(ctx context.Context, id uint64) error {
	return s.repo.Delete(ctx, id)
}

// validatePermissions 校验 permission JSON 的 page_key 和 perm_key 是否合法。
func validatePermissions(raw model.JSON) error {
	if len(raw) == 0 {
		return fmt.Errorf("权限内容不能为空")
	}
	var perms map[string][]string
	if err := json.Unmarshal(raw, &perms); err != nil {
		return fmt.Errorf("权限格式错误")
	}
	for pageKey, permKeys := range perms {
		available, ok := PredefinedPages[pageKey]
		if !ok {
			return fmt.Errorf("未知的页面标识: %q", pageKey)
		}
		for _, pk := range permKeys {
			found := false
			for _, a := range available {
				if a == pk {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("页面 %q 不支持的权限操作: %q", pageKey, pk)
			}
		}
	}
	return nil
}
