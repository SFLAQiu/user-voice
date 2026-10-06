package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apperr "github.com/feedback/internal/pkg/errors"
)

type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type Page struct {
	List     any   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// OK writes a success response.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: apperr.CodeOK, Message: "ok", Data: data})
}

// Fail writes an error response. HTTP status mirrors the app code class.
func Fail(c *gin.Context, err error) {
	var ae *apperr.AppError
	if errors.As(err, &ae) {
		c.JSON(httpStatus(ae.Code), Body{Code: ae.Code, Message: ae.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, Body{
		Code:    apperr.CodeInternal,
		Message: "internal server error",
	})
}

func httpStatus(code int) int {
	switch code {
	case apperr.CodeInvalidParam:
		return http.StatusBadRequest
	case apperr.CodeUnauthorized:
		return http.StatusUnauthorized
	case apperr.CodeForbidden:
		return http.StatusForbidden
	case apperr.CodeNotFound:
		return http.StatusNotFound
	case apperr.CodeRateLimited:
		return http.StatusTooManyRequests
	case apperr.CodeExternalFailure:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}
