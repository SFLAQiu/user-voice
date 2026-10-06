package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

type DataSourceRepo struct{ db *gorm.DB }

func NewDataSourceRepo(db *gorm.DB) *DataSourceRepo { return &DataSourceRepo{db: db} }

func (r *DataSourceRepo) Create(ctx context.Context, ds *model.DataSource) error {
	return r.db.WithContext(ctx).Create(ds).Error
}

func (r *DataSourceRepo) Update(ctx context.Context, ds *model.DataSource) error {
	return r.db.WithContext(ctx).Save(ds).Error
}

func (r *DataSourceRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.DataSource{}, id).Error
}

func (r *DataSourceRepo) FindByID(ctx context.Context, id uint64) (*model.DataSource, error) {
	var ds model.DataSource
	err := r.db.WithContext(ctx).First(&ds, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ds, nil
}

func (r *DataSourceRepo) List(ctx context.Context) ([]model.DataSource, error) {
	var out []model.DataSource
	err := r.db.WithContext(ctx).Order("id desc").Find(&out).Error
	return out, err
}

func (r *DataSourceRepo) ListEnabled(ctx context.Context) ([]model.DataSource, error) {
	var out []model.DataSource
	err := r.db.WithContext(ctx).Where("status = ?", model.DataSourceStatusEnabled).
		Order("id asc").Find(&out).Error
	return out, err
}

func (r *DataSourceRepo) UpdateSyncStatus(ctx context.Context, id uint64, status, errMsg string, at time.Time) error {
	updates := map[string]any{
		"last_sync_status": status,
		"last_sync_at":     at,
		"last_sync_error":  errMsg,
	}
	return r.db.WithContext(ctx).Model(&model.DataSource{}).
		Where("id = ?", id).Updates(updates).Error
}

// --- SyncCursor ---

type SyncCursorRepo struct{ db *gorm.DB }

func NewSyncCursorRepo(db *gorm.DB) *SyncCursorRepo { return &SyncCursorRepo{db: db} }

func (r *SyncCursorRepo) Get(ctx context.Context, sourceID uint64, key string) (*model.SyncCursor, error) {
	var c model.SyncCursor
	err := r.db.WithContext(ctx).
		Where("source_id = ? AND cursor_key = ?", sourceID, key).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *SyncCursorRepo) Upsert(ctx context.Context, c *model.SyncCursor) error {
	return r.db.WithContext(ctx).
		Where("source_id = ? AND cursor_key = ?", c.SourceID, c.CursorKey).
		Assign(map[string]any{
			"last_original_id": c.LastOriginalID,
			"last_created_at":  c.LastCreatedAt,
		}).
		FirstOrCreate(c).Error
}

// --- SyncLog ---

type SyncLogRepo struct{ db *gorm.DB }

func NewSyncLogRepo(db *gorm.DB) *SyncLogRepo { return &SyncLogRepo{db: db} }

func (r *SyncLogRepo) Create(ctx context.Context, log *model.SyncLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *SyncLogRepo) Update(ctx context.Context, log *model.SyncLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *SyncLogRepo) ListBySource(ctx context.Context, sourceID uint64, limit int) ([]model.SyncLog, error) {
	var out []model.SyncLog
	err := r.db.WithContext(ctx).
		Where("source_id = ?", sourceID).
		Order("started_at DESC").
		Limit(limit).
		Find(&out).Error
	return out, err
}
