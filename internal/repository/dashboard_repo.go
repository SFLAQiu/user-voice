package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

type DashboardRepo struct{ db *gorm.DB }

func NewDashboardRepo(db *gorm.DB) *DashboardRepo { return &DashboardRepo{db: db} }

func (r *DashboardRepo) List(ctx context.Context) ([]model.Dashboard, error) {
	var rows []model.Dashboard
	err := r.db.WithContext(ctx).Order("id DESC").Find(&rows).Error
	return rows, err
}

func (r *DashboardRepo) Get(ctx context.Context, id uint64) (*model.Dashboard, error) {
	var d model.Dashboard
	if err := r.db.WithContext(ctx).First(&d, id).Error; err != nil {
		return nil, err
	}
	var panels []model.Panel
	if err := r.db.WithContext(ctx).Where("dashboard_id = ?", id).Order("id ASC").Find(&panels).Error; err != nil {
		return nil, err
	}
	d.Panels = panels
	return &d, nil
}

func (r *DashboardRepo) Create(ctx context.Context, d *model.Dashboard) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *DashboardRepo) Update(ctx context.Context, d *model.Dashboard) error {
	return r.db.WithContext(ctx).Save(d).Error
}

func (r *DashboardRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("dashboard_id = ?", id).Delete(&model.Panel{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Dashboard{}, id).Error
	})
}

// ListByIDs 批量查询仪表盘，用于告警管理页面填充仪表盘名称。
func (r *DashboardRepo) ListByIDs(ctx context.Context, ids []uint64) ([]model.Dashboard, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []model.Dashboard
	return rows, r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error
}

type PanelRepo struct{ db *gorm.DB }

func NewPanelRepo(db *gorm.DB) *PanelRepo { return &PanelRepo{db: db} }

func (r *PanelRepo) ListByDashboard(ctx context.Context, dashboardID uint64) ([]model.Panel, error) {
	var rows []model.Panel
	err := r.db.WithContext(ctx).Where("dashboard_id = ?", dashboardID).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *PanelRepo) Get(ctx context.Context, id uint64) (*model.Panel, error) {
	var p model.Panel
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PanelRepo) Create(ctx context.Context, p *model.Panel) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *PanelRepo) Update(ctx context.Context, p *model.Panel) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *PanelRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.Panel{}, id).Error
}

// ListByIDs 批量查询面板，用于告警管理页面填充面板名称。
func (r *PanelRepo) ListByIDs(ctx context.Context, ids []uint64) ([]model.Panel, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []model.Panel
	return rows, r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error
}
