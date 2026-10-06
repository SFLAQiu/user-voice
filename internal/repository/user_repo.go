package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

type UserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id uint64) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, id uint64, at time.Time) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).Update("last_login_at", at).Error
}

func (r *UserRepo) List(ctx context.Context) ([]model.User, error) {
	var out []model.User
	err := r.db.WithContext(ctx).Order("id desc").Find(&out).Error
	return out, err
}

func (r *UserRepo) SetStatus(ctx context.Context, id uint64, status int) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).Update("status", status).Error
}

// FindGroupsByUserID 查询用户所属的权限组。
func (r *UserRepo) FindGroupsByUserID(ctx context.Context, userID uint64) ([]model.PermissionGroup, error) {
	var groups []model.PermissionGroup
	err := r.db.WithContext(ctx).
		Joins("JOIN user_permission_groups upg ON upg.group_id = permission_groups.id").
		Where("upg.user_id = ?", userID).
		Find(&groups).Error
	return groups, err
}

// SetGroups 覆盖式设置用户的权限组。
func (r *UserRepo) SetGroups(ctx context.Context, userID uint64, groupIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserPermissionGroup{}).Error; err != nil {
			return err
		}
		if len(groupIDs) == 0 {
			return nil
		}
		for _, gid := range groupIDs {
			upg := model.UserPermissionGroup{UserID: userID, GroupID: gid}
			if err := tx.Create(&upg).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// FindOverridesByUserID 查询用户的权限覆盖列表。
func (r *UserRepo) FindOverridesByUserID(ctx context.Context, userID uint64) ([]model.UserPermissionOverride, error) {
	var out []model.UserPermissionOverride
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&out).Error
	return out, err
}

// --- Login attempts ---

type LoginAttemptRepo struct{ db *gorm.DB }

func NewLoginAttemptRepo(db *gorm.DB) *LoginAttemptRepo { return &LoginAttemptRepo{db: db} }

func (r *LoginAttemptRepo) Record(ctx context.Context, identifier string, success bool) error {
	rec := &model.LoginAttempt{
		Identifier:  identifier,
		AttemptedAt: time.Now(),
		Success:     boolToInt(success),
	}
	return r.db.WithContext(ctx).Create(rec).Error
}

func (r *LoginAttemptRepo) CountFailuresSince(ctx context.Context, identifier string, since time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.LoginAttempt{}).
		Where("identifier = ? AND attempted_at >= ? AND success = 0", identifier, since).
		Count(&n).Error
	return n, err
}

func (r *LoginAttemptRepo) Cleanup(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).Where("attempted_at < ?", before).
		Delete(&model.LoginAttempt{}).Error
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
