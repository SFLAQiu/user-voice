package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/feedback/internal/crypto"
	"github.com/feedback/internal/datasource"
	"github.com/feedback/internal/datasource/httpapi"
	"github.com/feedback/internal/datasource/mysql"
	"github.com/feedback/internal/model"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/logger"
	"github.com/feedback/internal/repository"
)

// DataSourceService manages CRUD plus orchestrates sync runs.
type DataSourceService struct {
	repo          *repository.DataSourceRepo
	cursors       *repository.SyncCursorRepo
	fbRepo        *repository.FeedbackRepo
	syncLogRepo   *repository.SyncLogRepo
	enumConfigSvc *EnumConfigService
	registry      *datasource.Registry
	cipher        *crypto.Cipher
	bucketSvc     *MetricBucketService
}

func NewDataSourceService(repo *repository.DataSourceRepo, cursors *repository.SyncCursorRepo,
	fbRepo *repository.FeedbackRepo, syncLogRepo *repository.SyncLogRepo, enumConfigSvc *EnumConfigService, reg *datasource.Registry, c *crypto.Cipher, bucketSvc *MetricBucketService) *DataSourceService {
	return &DataSourceService{repo: repo, cursors: cursors, fbRepo: fbRepo, syncLogRepo: syncLogRepo, enumConfigSvc: enumConfigSvc, registry: reg, cipher: c, bucketSvc: bucketSvc}
}

type CreateDataSourceInput struct {
	Name     string          `json:"name" binding:"required,max=128"`
	Type     string          `json:"type" binding:"required,max=32"`
	Config   json.RawMessage `json:"config" binding:"required"`
	SyncCron string          `json:"sync_cron"`
	Status   *int            `json:"status"`
}

func (s *DataSourceService) Create(ctx context.Context, in CreateDataSourceInput) (*model.DataSource, error) {
	if err := s.validateAndEncrypt(in.Type, &in.Config); err != nil {
		return nil, err
	}
	ds := &model.DataSource{
		Name:       in.Name,
		Type:       in.Type,
		ConfigJSON: model.JSON(in.Config),
		SyncCron:   defaultStr(in.SyncCron, "0 */5 * * * *"),
		Status:     model.DataSourceStatusEnabled,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if in.Status != nil {
		ds.Status = *in.Status
	}
	if err := s.repo.Create(ctx, ds); err != nil {
		return nil, apperr.Internal("create data source", err)
	}
	return ds, nil
}

type UpdateDataSourceInput struct {
	Name     *string         `json:"name"`
	Config   json.RawMessage `json:"config"`
	SyncCron *string         `json:"sync_cron"`
	Status   *int            `json:"status"`
}

func (s *DataSourceService) Update(ctx context.Context, id uint64, in UpdateDataSourceInput) (*model.DataSource, error) {
	ds, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.Internal("load data source", err)
	}
	if ds == nil {
		return nil, apperr.NotFound("data source not found")
	}
	if in.Name != nil {
		ds.Name = *in.Name
	}
	if len(in.Config) > 0 {
		if err := s.validateAndEncrypt(ds.Type, &in.Config); err != nil {
			return nil, err
		}
		ds.ConfigJSON = model.JSON(in.Config)
	}
	if in.SyncCron != nil {
		ds.SyncCron = *in.SyncCron
	}
	if in.Status != nil {
		ds.Status = *in.Status
	}
	ds.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, ds); err != nil {
		return nil, apperr.Internal("update data source", err)
	}
	return ds, nil
}

func (s *DataSourceService) Delete(ctx context.Context, id uint64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperr.Internal("delete data source", err)
	}
	return nil
}

func (s *DataSourceService) List(ctx context.Context) ([]model.DataSource, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, apperr.Internal("list data sources", err)
	}
	return rows, nil
}

func (s *DataSourceService) Get(ctx context.Context, id uint64) (*model.DataSource, error) {
	ds, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.Internal("load data source", err)
	}
	if ds == nil {
		return nil, apperr.NotFound("data source not found")
	}
	return ds, nil
}

func (s *DataSourceService) ListSyncLogs(ctx context.Context, sourceID uint64, limit int) ([]model.SyncLog, error) {
	logs, err := s.syncLogRepo.ListBySource(ctx, sourceID, limit)
	if err != nil {
		return nil, apperr.Internal("list sync logs", err)
	}
	return logs, nil
}

// AggregatedEnums returns enum mappings. Reads from enum_configs table first,
// then merges data_source config enums as fallback for fields not yet in the table.
func (s *DataSourceService) AggregatedEnums(ctx context.Context) (map[string]map[string]string, error) {
	merged := make(map[string]map[string]string)

	// Primary source: enum_configs table
	if s.enumConfigSvc != nil {
		tableEnums, err := s.enumConfigSvc.AggregatedMap(ctx)
		if err == nil && len(tableEnums) > 0 {
			for field, vals := range tableEnums {
				merged[field] = vals
			}
		}
	}

	// Fallback: data_source config enums (only fill fields not yet in enum_configs)
	rows, err := s.repo.List(ctx)
	if err != nil {
		if len(merged) > 0 {
			return merged, nil
		}
		return nil, apperr.Internal("list data sources for enums", err)
	}
	for _, ds := range rows {
		var cfg map[string]any
		if err := json.Unmarshal(ds.ConfigJSON, &cfg); err != nil {
			continue
		}
		enumsRaw, ok := cfg["enums"]
		if !ok {
			continue
		}
		enumsMap, ok := enumsRaw.(map[string]any)
		if !ok {
			continue
		}
		for field, vals := range enumsMap {
			if merged[field] != nil {
				continue // already have from enum_configs table
			}
			valsMap, ok := vals.(map[string]any)
			if !ok {
				continue
			}
			merged[field] = make(map[string]string)
			for k, v := range valsMap {
				if s, ok := v.(string); ok {
					merged[field][k] = s
				}
			}
		}
	}
	return merged, nil
}

// Sync runs the plugin once for the given data source.
func (s *DataSourceService) Sync(ctx context.Context, id uint64) (int, error) {
	ds, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return 0, apperr.Internal("load data source", err)
	}
	if ds == nil {
		return 0, apperr.NotFound("data source not found")
	}
	plugin, err := s.registry.Get(ds.Type)
	if err != nil {
		return 0, apperr.InvalidParam(err.Error())
	}

	syncLog := &model.SyncLog{
		SourceID:  id,
		Status:    model.SyncStatusRunning,
		StartedAt: time.Now(),
	}
	if err := s.syncLogRepo.Create(ctx, syncLog); err != nil {
		logger.L.Error("sync: create sync_log failed, will attempt again after pull", zap.Uint64("source_id", id), zap.Error(err))
	}

	mc, _ := s.cursors.Get(ctx, ds.ID, "default")
	var cursor *datasource.Cursor
	if mc != nil {
		cursor = &datasource.Cursor{
			Key:            mc.CursorKey,
			LastOriginalID: mc.LastOriginalID,
			LastCreatedAt:  mc.LastCreatedAt,
		}
	}
	exists := func(origID string) (bool, error) {
		return s.fbRepo.ExistsOriginal(ctx, ds.ID, origID)
	}

	// Merge global enums into config so the plugin can resolve labels during pull.
	configWithEnums := ds.ConfigJSON
	if s.enumConfigSvc != nil {
		globalEnums, err := s.enumConfigSvc.AggregatedMap(ctx)
		if err == nil && len(globalEnums) > 0 {
			configWithEnums = mergeEnumsIntoConfig(ds.ConfigJSON, globalEnums)
		}
	}

	now := time.Now()
	res, err := plugin.Pull(ctx, json.RawMessage(configWithEnums), cursor, exists)
	if err != nil {
		s.finishSyncLog(ctx, syncLog, ds.ID, model.SyncStatusFailed, truncate(err.Error(), 1000), now)
		return 0, apperr.Wrap(apperr.CodeExternalFailure, "data source pull failed", err)
	}

	inserted := 0
	if len(res.Items) > 0 {
		toSave, err := itemsToModels(ds.ID, res.Items)
		if err != nil {
			s.finishSyncLog(ctx, syncLog, ds.ID, model.SyncStatusFailed, err.Error(), now)
			return 0, apperr.Internal("convert items", err)
		}
		// 按 original_created_at 升序排序，确保时间越早 ID 越小
		sort.Slice(toSave, func(i, j int) bool {
			return toSave[i].OriginalCreatedAt.Before(toSave[j].OriginalCreatedAt)
		})
		n, err := s.fbRepo.UpsertBatch(ctx, toSave)
		if err != nil {
			s.finishSyncLog(ctx, syncLog, ds.ID, model.SyncStatusFailed, err.Error(), now)
			return 0, apperr.Internal("upsert feedbacks", err)
		}
		inserted = int(n)
		// 同步入库后增量写入时间网格预聚合
		if s.bucketSvc != nil {
			if err := s.bucketSvc.IncrementFromFeedbacks(ctx, toSave); err != nil {
				logger.L.Warn("metric-bucket: increment from sync", zap.Uint64("source_id", ds.ID), zap.Error(err))
			}
		}
	}
	if res.NextCursor != nil {
		_ = s.cursors.Upsert(ctx, &model.SyncCursor{
			SourceID:       ds.ID,
			CursorKey:      defaultStr(res.NextCursor.Key, "default"),
			LastOriginalID: res.NextCursor.LastOriginalID,
			LastCreatedAt:  res.NextCursor.LastCreatedAt,
		})
	}
	syncLog.FetchedCount = len(res.Items)
	syncLog.InsertedCount = inserted
	s.finishSyncLog(ctx, syncLog, ds.ID, model.SyncStatusSuccess, "", now)
	return inserted, nil
}

// finishSyncLog updates the sync_log and data_source status.
// Errors are logged but never block the caller — the sync result
// is already determined, and DB failures here should not prevent
// the next cron tick from running.
func (s *DataSourceService) finishSyncLog(ctx context.Context, syncLog *model.SyncLog, dsID uint64, status, errMsg string, at time.Time) {
	syncLog.Status = status
	if errMsg != "" {
		syncLog.ErrorMessage = truncate(errMsg, 1000)
	}
	syncLog.FinishedAt = &at
	if err := s.syncLogRepo.Update(ctx, syncLog); err != nil {
		logger.L.Error("sync: update sync_log failed", zap.Uint64("source_id", dsID), zap.String("status", status), zap.Error(err))
	}
	if err := s.repo.UpdateSyncStatus(ctx, dsID, status, truncate(errMsg, 1000), at); err != nil {
		logger.L.Error("sync: update data_source status failed", zap.Uint64("source_id", dsID), zap.String("status", status), zap.Error(err))
	}
}

// --- helpers ---

func (s *DataSourceService) validateAndEncrypt(typ string, raw *json.RawMessage) error {
	p, err := s.registry.Get(typ)
	if err != nil {
		return apperr.InvalidParam(err.Error())
	}
	if err := p.Validate(*raw); err != nil {
		return apperr.InvalidParam(fmt.Sprintf("invalid config: %v", err))
	}
	enc, err := s.encryptSensitive(typ, *raw)
	if err != nil {
		return apperr.Internal("encrypt sensitive fields", err)
	}
	*raw = enc
	return nil
}

// encryptSensitive walks known sensitive keys per plugin type.
func (s *DataSourceService) encryptSensitive(typ string, raw json.RawMessage) (json.RawMessage, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	switch typ {
	case httpapi.Type:
		// 原有 httpapi 敏感字段处理
		if v, ok := m["cookie"].(string); ok && v != "" && !crypto.IsEncrypted(v) {
			enc, err := s.cipher.Encrypt(v)
			if err != nil {
				return nil, err
			}
			m["cookie"] = enc
		}
		if h, ok := m["headers"].(map[string]any); ok {
			for k, v := range h {
				if !strings.EqualFold(k, "cookie") && !strings.EqualFold(k, "authorization") {
					continue
				}
				if s2, ok := v.(string); ok && s2 != "" && !crypto.IsEncrypted(s2) {
					enc, err := s.cipher.Encrypt(s2)
					if err != nil {
						return nil, err
					}
					h[k] = enc
				}
			}
			m["headers"] = h
		}
	case mysql.Type:
		if v, ok := m["password"].(string); ok && v != "" && !crypto.IsEncrypted(v) {
			enc, err := s.cipher.Encrypt(v)
			if err != nil {
				return nil, err
			}
			m["password"] = enc
		}
	}
	return json.Marshal(m)
}

func itemsToModels(sourceID uint64, items []datasource.FeedbackItem) ([]*model.Feedback, error) {
	out := make([]*model.Feedback, 0, len(items))
	now := time.Now()
	for _, it := range items {
		raw, err := json.Marshal(it.Raw)
		if err != nil {
			return nil, err
		}
		var imagesJSON model.JSON
		if len(it.Images) > 0 {
			b, _ := json.Marshal(it.Images)
			imagesJSON = b
		}
		var videosJSON model.JSON
		if len(it.Videos) > 0 {
			b, _ := json.Marshal(it.Videos)
			videosJSON = b
		}
		out = append(out, &model.Feedback{
			SourceID:          sourceID,
			OriginalID:        it.OriginalID,
			AppID:             it.AppID,
			AppName:           it.AppName,
			Platform:          it.Platform,
			PlatformID:        it.PlatformID,
			UserID:            it.UserID,
			UserName:          it.UserName,
			UserMode:          it.UserMode,
			Content:           it.Content,
			Images:            imagesJSON,
			Videos:            videosJSON,
			PhoneModel:        it.PhoneModel,
			AppVersion:        it.AppVersion,
			ChannelID:         it.ChannelID,
			QQ:                it.QQ,
			FileURL:           it.FileURL,
			RawJSON:           model.JSON(raw),
			OriginalCreatedAt: it.OriginalCreatedAt,
			SyncedAt:          now,
			CreatedAt:         now,
			UpdatedAt:         now,
		})
	}
	return out, nil
}

func defaultStr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// mergeEnumsIntoConfig merges global enums into the data source config JSON.
// Config-level enums take precedence; global enums fill fields not present in config.
func mergeEnumsIntoConfig(raw model.JSON, globalEnums map[string]map[string]string) model.JSON {
	if len(raw) == 0 {
		return raw
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return raw
	}
	configEnums, _ := cfg["enums"].(map[string]any)
	for field, labels := range globalEnums {
		if configEnums != nil && configEnums[field] != nil {
			continue // config-level override exists
		}
		if cfg["enums"] == nil {
			cfg["enums"] = make(map[string]any)
		}
		enumsMap := cfg["enums"].(map[string]any)
		labelMap := make(map[string]any)
		for k, v := range labels {
			labelMap[k] = v
		}
		enumsMap[field] = labelMap
	}
	out, _ := json.Marshal(cfg)
	return model.JSON(out)
}