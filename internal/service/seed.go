package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/feedback/internal/config"
	"github.com/feedback/internal/model"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/repository"
)

// EnsureAdminSeed creates the default admin user if none exists.
func EnsureAdminSeed(ctx context.Context, users *repository.UserRepo, cfg config.AdminSeedConfig) error {
	if cfg.Username == "" || cfg.Password == "" {
		return nil
	}
	existing, err := users.FindByUsername(ctx, cfg.Username)
	if err != nil {
		return apperr.Internal("seed lookup", err)
	}
	if existing != nil {
		return nil
	}
	hash, err := HashPassword(cfg.Password)
	if err != nil {
		return apperr.Internal("hash seed pw", err)
	}
	u := &model.User{
		Username:     cfg.Username,
		PasswordHash: hash,
		Role:         model.RoleAdmin,
		Status:       model.UserStatusEnabled,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return users.Create(ctx, u)
}

// 默认权限组定义
func defaultAdminPermissions() model.JSON {
	perms := make(map[string][]string, len(PredefinedPages))
	for page, keys := range PredefinedPages {
		copied := make([]string, len(keys))
		copy(copied, keys)
		perms[page] = copied
	}
	b, _ := json.Marshal(perms)
	return model.JSON(b)
}

func defaultViewerPermissions() model.JSON {
	// 观察者只有 view 权限，且不能看用户管理
	perms := make(map[string][]string)
	for page := range PredefinedPages {
		if page == "users" {
			continue
		}
		perms[page] = []string{"view"}
	}
	b, _ := json.Marshal(perms)
	return model.JSON(b)
}

// EnsureDefaultPermissionGroups 确保默认权限组存在，并迁移现有用户。
func EnsureDefaultPermissionGroups(ctx context.Context,
	groupRepo *repository.PermissionGroupRepo,
	userRepo *repository.UserRepo,
	overrideRepo *repository.UserPermissionOverridesRepo,
) error {
	// 创建默认"管理员"权限组
	adminGroup, err := groupRepo.FindByName(ctx, "管理员")
	if err != nil {
		return err
	}
	if adminGroup == nil {
		adminGroup = &model.PermissionGroup{
			Name:        "管理员",
			Description: "默认管理员权限组——所有页面读写权限",
			Permissions: defaultAdminPermissions(),
		}
		if err := groupRepo.Create(ctx, adminGroup); err != nil {
			return err
		}
	}

	// 创建默认"观察者"权限组
	viewerGroup, err := groupRepo.FindByName(ctx, "观察者")
	if err != nil {
		return err
	}
	if viewerGroup == nil {
		viewerGroup = &model.PermissionGroup{
			Name:        "观察者",
			Description: "默认观察者权限组——只读权限",
			Permissions: defaultViewerPermissions(),
		}
		if err := groupRepo.Create(ctx, viewerGroup); err != nil {
			return err
		}
	}

	// 迁移现有用户到对应权限组（仅尚未分配组的用户）
	allUsers, err := userRepo.List(ctx)
	if err != nil {
		return err
	}
	for _, u := range allUsers {
		groups, err := userRepo.FindGroupsByUserID(ctx, u.ID)
		if err != nil {
			return err
		}
		if len(groups) > 0 {
			continue
		}
		var gid uint64
		if u.Role == model.RoleAdmin {
			gid = adminGroup.ID
		} else {
			gid = viewerGroup.ID
		}
		_ = userRepo.SetGroups(ctx, u.ID, []uint64{gid})
	}

	return nil
}
