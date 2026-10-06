package handler

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/service"
)

type LLMProviderHandler struct {
	svc *service.LLMProviderService
}

func NewLLMProviderHandler(svc *service.LLMProviderService) *LLMProviderHandler {
	return &LLMProviderHandler{svc: svc}
}

func (h *LLMProviderHandler) List(c *gin.Context) {
	rows, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

func (h *LLMProviderHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	p, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

func (h *LLMProviderHandler) Create(c *gin.Context) {
	var in service.CreateLLMProviderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	p, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

func (h *LLMProviderHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in service.UpdateLLMProviderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	p, err := h.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

func (h *LLMProviderHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *LLMProviderHandler) TestNew(c *gin.Context) {
	var in service.TestConnectionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	result, err := h.svc.TestConnection(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *LLMProviderHandler) TestExisting(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	result, err := h.svc.TestExistingProvider(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}