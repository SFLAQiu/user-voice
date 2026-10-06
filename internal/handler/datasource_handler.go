package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/service"
)

type DataSourceHandler struct {
	svc      *service.DataSourceService
	onChange func()
}

func NewDataSourceHandler(svc *service.DataSourceService, onChange func()) *DataSourceHandler {
	return &DataSourceHandler{svc: svc, onChange: onChange}
}

func (h *DataSourceHandler) List(c *gin.Context) {
	rows, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

func (h *DataSourceHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	ds, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ds)
}

func (h *DataSourceHandler) Create(c *gin.Context) {
	var in service.CreateDataSourceInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	ds, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if h.onChange != nil {
		h.onChange()
	}
	response.OK(c, ds)
}

func (h *DataSourceHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in service.UpdateDataSourceInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	ds, err := h.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if h.onChange != nil {
		h.onChange()
	}
	response.OK(c, ds)
}

func (h *DataSourceHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	if h.onChange != nil {
		h.onChange()
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *DataSourceHandler) Sync(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	n, err := h.svc.Sync(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"inserted": n})
}

func (h *DataSourceHandler) ListSyncLogs(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	limit := parseIntQuery(c, "limit", 20)
	logs, err := h.svc.ListSyncLogs(c.Request.Context(), id, limit)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, logs)
}

func (h *DataSourceHandler) AggregatedEnums(c *gin.Context) {
	enums, err := h.svc.AggregatedEnums(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, enums)
}

func parseIntQuery(c *gin.Context, key string, defaultVal int) int {
	s := c.Query(key)
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return n
}

func parseUintParam(c *gin.Context, name string) (uint64, error) {
	s := c.Param(name)
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, apperr.InvalidParam("invalid " + name)
	}
	return n, nil
}
