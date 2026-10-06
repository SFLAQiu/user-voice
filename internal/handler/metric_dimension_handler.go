package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/repository"
	"github.com/feedback/internal/service"
)

// MetricDimensionHandler 维度配置管理 API。
type MetricDimensionHandler struct {
	svc     *service.DimensionSchemaService
	logRepo *repository.MetricBucketUpdateLogRepo
}

func NewMetricDimensionHandler(svc *service.DimensionSchemaService, logRepo *repository.MetricBucketUpdateLogRepo) *MetricDimensionHandler {
	return &MetricDimensionHandler{svc: svc, logRepo: logRepo}
}

// List 返回所有维度配置（含禁用）。
func (h *MetricDimensionHandler) List(c *gin.Context) {
	rows, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

type AddDimensionInput struct {
	Field        string `json:"field" binding:"required,max=64"`
	Label        string `json:"label" binding:"required,max=128"`
	DataType     string `json:"data_type" binding:"required,oneof=string int"`
	DefaultValue string `json:"default_value" binding:"max=64"`
}

func (h *MetricDimensionHandler) Add(c *gin.Context) {
	var in AddDimensionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	if err := h.svc.AddDimension(c.Request.Context(), in.Field, in.Label, in.DataType, in.DefaultValue); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"message": "维度已添加，重建任务已启动"})
}

func (h *MetricDimensionHandler) Remove(c *gin.Context) {
	field := c.Param("field")
	if field == "" {
		response.Fail(c, apperr.InvalidParam("field 参数缺失"))
		return
	}
	if err := h.svc.RemoveDimension(c.Request.Context(), field); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"message": "维度已删除，重建任务已启动"})
}

func (h *MetricDimensionHandler) RebuildStatus(c *gin.Context) {
	status := h.svc.GetStatus()
	response.OK(c, status)
}

// TriggerBackfill 手动触发全量回填历史数据（首次部署或数据修复时调用）。
func (h *MetricDimensionHandler) TriggerBackfill(c *gin.Context) {
	if err := h.svc.TriggerBackfill(c.Request.Context()); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"message": "全量回填任务已启动"})
}

// ListUpdateLogs 返回最近的桶更新日志。
func (h *MetricDimensionHandler) ListUpdateLogs(c *gin.Context) {
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	logs, err := h.logRepo.ListRecent(c.Request.Context(), limit)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, logs)
}
