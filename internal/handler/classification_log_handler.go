package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/service"
)

type ClassificationLogHandler struct {
	svc *service.ClassificationLogService
}

func NewClassificationLogHandler(svc *service.ClassificationLogService) *ClassificationLogHandler {
	return &ClassificationLogHandler{svc: svc}
}

func (h *ClassificationLogHandler) ListByFeedback(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	rows, err := h.svc.ListByFeedbackID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}