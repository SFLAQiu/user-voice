package service

import (
	"context"
	"encoding/json"

	"github.com/feedback/internal/model"
	"github.com/feedback/internal/repository"
)

// 预定义页面与权限映射——所有可用 page_key 及其 perm_key 列表。
var PredefinedPages = map[string][]string{
	"feedback":              {"view", "edit"},
	"dashboard":             {"view", "edit"},
	"alert":                 {"view", "edit"},
	"notification_channels": {"view", "edit"},
	"datasources":           {"view", "edit"},
	"enum_configs":          {"view", "edit"},
	"classifier_configs":    {"view", "edit"},
	"metric_dimensions":     {"view", "edit"},
	"llm_providers":         {"view", "edit"},
	"users":                 {"view", "edit"},
	"guide":                 {"view"},
}

type PermissionService struct {
	groupRepo    *repository.PermissionGroupRepo
	userRepo     *repository.UserRepo
	overrideRepo *repository.UserPermissionOverridesRepo
}

func NewPermissionService(
	groupRepo *repository.PermissionGroupRepo,
	userRepo *repository.UserRepo,
	overrideRepo *repository.UserPermissionOverridesRepo,
) *PermissionService {
	return &PermissionService{
		groupRepo:    groupRepo,
		userRepo:     userRepo,
		overrideRepo: overrideRepo,
	}
}

// OverrideRepo 暴露权限覆盖仓储（供 handler 使用）。
func (s *PermissionService) OverrideRepo() *repository.UserPermissionOverridesRepo {
	return s.overrideRepo
}

// GetEffectivePermissions 计算用户的最终权限。
func (s *PermissionService) GetEffectivePermissions(ctx context.Context, userID uint64) (map[string][]string, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	// 超管拥有所有权限
	if user.Role == model.RoleAdmin {
		result := make(map[string][]string, len(PredefinedPages))
		for page, perms := range PredefinedPages {
			copied := make([]string, len(perms))
			copy(copied, perms)
			result[page] = copied
		}
		return result, nil
	}

	effective := make(map[string]map[string]bool)

	// 合并所有分组的权限
	groups, err := s.userRepo.FindGroupsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		addPermissions(effective, g.Permissions)
	}

	// 合并个人权限覆盖
	overrides, err := s.userRepo.FindOverridesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, o := range overrides {
		if effective[o.PageKey] == nil {
			effective[o.PageKey] = make(map[string]bool)
		}
		if o.GrantType == 1 {
			effective[o.PageKey][o.PermKey] = true
		} else {
			effective[o.PageKey][o.PermKey] = false
		}
	}

	// 转为 []string
	result := make(map[string][]string)
	for page, perms := range effective {
		for perm, granted := range perms {
			if granted {
				result[page] = append(result[page], perm)
			}
		}
	}
	return result, nil
}

// HasPermission 检查用户是否有指定页面+操作的权限。
func (s *PermissionService) HasPermission(ctx context.Context, userID uint64, pageKey, permKey string) bool {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return false
	}
	if user.Role == model.RoleAdmin {
		return true
	}
	perms, err := s.GetEffectivePermissions(ctx, userID)
	if err != nil {
		return false
	}
	for _, p := range perms[pageKey] {
		if p == permKey {
			return true
		}
	}
	return false
}

func addPermissions(effective map[string]map[string]bool, raw model.JSON) {
	if len(raw) == 0 {
		return
	}
	var perms map[string][]string
	if err := json.Unmarshal(raw, &perms); err != nil {
		return
	}
	for page, keys := range perms {
		if effective[page] == nil {
			effective[page] = make(map[string]bool)
		}
		for _, k := range keys {
			effective[page][k] = true
		}
	}
}
