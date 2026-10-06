// Package scheduler wires data source sync jobs into a cron runner.
package scheduler

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/feedback/internal/pkg/logger"
	"github.com/feedback/internal/repository"
	"github.com/feedback/internal/service"
)

type Scheduler struct {
	cron      *cron.Cron
	repo      *repository.DataSourceRepo
	ds        *service.DataSourceService
	db        *gorm.DB
	mu        sync.Mutex
	entries   map[uint64]cron.EntryID
	syncLocks map[uint64]*sync.Mutex
}

func New(db *gorm.DB, repo *repository.DataSourceRepo, ds *service.DataSourceService) *Scheduler {
	return &Scheduler{
		cron:      cron.New(cron.WithParser(cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow))),
		repo:      repo,
		ds:        ds,
		db:        db,
		entries:   make(map[uint64]cron.EntryID),
		syncLocks: make(map[uint64]*sync.Mutex),
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	if err := s.Reload(ctx); err != nil {
		return err
	}
	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
}

// Reload re-registers all enabled data sources, removing any whose schedule changed.
func (s *Scheduler) Reload(ctx context.Context) error {
	// Load from DB first — don't destroy existing entries until we know the new set.
	rows, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return err
	}

	// Also query all sources to surface status mismatches in logs.
	allRows, allErr := s.repo.List(ctx)
	if allErr == nil {
		for _, ds := range allRows {
			if ds.Status != 1 {
				logger.L.Warn("data source is disabled, will not be scheduled",
					zap.Uint64("id", ds.ID), zap.String("name", ds.Name), zap.Int("status", ds.Status))
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Now safe to clear old entries.
	for _, id := range s.entryIDs() {
		s.cron.Remove(id)
	}
	s.entries = make(map[uint64]cron.EntryID)

	for _, ds := range rows {
		dsID := ds.ID
		spec := strings.TrimSpace(ds.SyncCron)
		// Normalize: 5-field → prepend "0 " so the parser always sees 6 fields.
		if len(strings.Fields(spec)) == 5 {
			spec = "0 " + spec
		}
		entryID, err := s.cron.AddFunc(spec, func() {
			s.runOne(dsID)
		})
		if err != nil {
			logger.L.Warn("schedule data source failed",
				zap.Uint64("data_source_id", dsID), zap.String("name", ds.Name),
				zap.String("cron", ds.SyncCron), zap.String("normalized", spec), zap.Error(err))
			continue
		}
		s.entries[dsID] = entryID
		logger.L.Info("scheduled data source",
			zap.Uint64("id", dsID), zap.String("name", ds.Name), zap.String("cron", spec))
	}
	logger.L.Info("scheduler reloaded", zap.Int("active_jobs", len(s.entries)))
	return nil
}

func (s *Scheduler) entryIDs() []cron.EntryID {
	out := make([]cron.EntryID, 0, len(s.entries))
	for _, v := range s.entries {
		out = append(out, v)
	}
	return out
}

func (s *Scheduler) runOne(id uint64) {
	// 获取或创建该数据源的内存锁
	s.mu.Lock()
	if _, ok := s.syncLocks[id]; !ok {
		s.syncLocks[id] = &sync.Mutex{}
	}
	dsLock := s.syncLocks[id]
	s.mu.Unlock()

	// 非阻塞尝试获取锁
	if !dsLock.TryLock() {
		logger.L.Info("sync skipped (lock busy)", zap.Uint64("id", id))
		return
	}
	defer dsLock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	n, err := s.ds.Sync(ctx, id)
	if err != nil {
		logger.L.Error("sync failed", zap.Uint64("id", id), zap.Error(err))
		return
	}
	logger.L.Info("sync ok", zap.Uint64("id", id), zap.Int("inserted", n))
}
