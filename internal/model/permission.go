package model

import "time"

// PermissionGroup 权限组——定义一组页面+操作的权限集合。
type PermissionGroup struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;uniqueIndex" json:"name"`
	Description string    `gorm:"size:255" json:"description,omitempty"`
	Permissions JSON      `gorm:"column:permissions;not null" json:"permissions" comment:"权限定义：{page_key: [perm_key, ...]}"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (PermissionGroup) TableName() string { return "permission_groups" }

// UserPermissionGroup 用户与权限组的多对多关联。
type UserPermissionGroup struct {
	UserID  uint64 `gorm:"primaryKey;column:user_id" json:"user_id"`
	GroupID uint64 `gorm:"primaryKey;column:group_id" json:"group_id"`
}

func (UserPermissionGroup) TableName() string { return "user_permission_groups" }

// UserPermissionOverride 单用户权限覆盖（显式授权或拒绝某个页面+操作）。
type UserPermissionOverride struct {
	UserID    uint64 `gorm:"primaryKey;column:user_id" json:"user_id"`
	PageKey   string `gorm:"primaryKey;size:64;column:page_key" json:"page_key"`
	PermKey   string `gorm:"primaryKey;size:64;column:perm_key" json:"perm_key"`
	GrantType int    `gorm:"column:grant_type;default:1" json:"grant_type" comment:"1=授权, 0=拒绝"`
}

func (UserPermissionOverride) TableName() string { return "user_permission_overrides" }
